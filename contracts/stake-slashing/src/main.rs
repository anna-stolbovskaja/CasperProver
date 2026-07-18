#![no_std]
#![no_main]

extern crate alloc;

use alloc::string::{String, ToString};
use alloc::vec;

use casper_contract::contract_api::{runtime, storage, system};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;
use casper_types::account::AccountHash;
use casper_types::contracts::NamedKeys;
use casper_types::{
    ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointPayment,
    EntryPointType, EntryPoints, Key, Parameter, RuntimeArgs, URef, U512,
};

// Real, on-chain-custody "skin in the game" for CasperProver agents: an
// agent stakes CSPR before it can be trusted, and anyone can permissionlessly
// claim a real slash reward once a proof they submitted is on-chain-revoked
// (proof-registry's existing `revoke_proof`, self-triggered today; a future
// oracle/dispute-resolution flow could call it too - this contract doesn't
// care who revoked it, only that proof-registry's own state says revoked).
//
// This deliberately does NOT reinvent revocation/dispute logic inside this
// contract - it reads proof-registry's existing, already-deployed, already-
// tested `get_proof` via a cross-contract call and reacts to its `revoked`
// flag. That keeps this contract's own attack surface small and auditable.
//
// SECURITY NOTES (read before redeploying with a different slash %/model):
// - report_and_slash is intentionally permissionless (anyone can call it) -
//   the only thing that can be "claimed" is a revocation that ALREADY
//   happened on proof-registry; there's no way to force a revocation from
//   here, so this can't be used to slash an honest agent.
// - Each proof_id can only trigger a slash once (SLASHED_DICT tombstone) -
//   prevents draining an agent's stake by replaying the same revoked proof.
// - Slash amount is capped at the agent's CURRENT recorded stake (U512
//   subtraction only after a checked comparison) - can't underflow/over-slash.
// - stake() itself is NOT a contract entry point that can pull funds from an
//   arbitrary caller purse (that's not something Casper permits safely from
//   inside a called contract) - deposits go through record_stake(), which is
//   only meant to be invoked from the companion `stake-slashing-session`
//   session code that performs the actual purse-to-purse transfer and this
//   call in the SAME deploy, so they can't be split/front-run. record_stake
//   trusts its caller's own AccountHash as the staker, not an arg, so nobody
//   can call it to credit stake to someone else's name without actually
//   being that account.

const ERR_ALREADY_SLASHED: u16 = 2;
const ERR_NOT_REVOKED: u16 = 3;
const ERR_AGENT_MISMATCH: u16 = 4;
const ERR_INSUFFICIENT_STAKE: u16 = 5;
const ERR_INPUT_TOO_LONG: u16 = 6;
const ERR_NO_MATCHING_TRANSFER: u16 = 7;

const PROOF_REGISTRY_HASH: &str = "proof_registry_hash";
const STAKES_DICT: &str = "stakes";
const SLASHED_DICT: &str = "slashed_proofs";
const CONTRACT_PURSE: &str = "contract_purse";
const KEY_TOTAL_RECORDED: &str = "total_recorded";
const MAX_PROOF_ID_LEN: usize = 128;

// Slash 20% of the agent's current stake per confirmed revocation, paid to
// whoever calls report_and_slash (a permissionless bounty for monitoring).
const SLASH_BPS: u64 = 2000; // basis points, 2000 = 20%

// Mirror of proof-registry's ProofRec layout - this contract only reads it.
type ProofRec = (
    (String, String, String),
    (String, String, String),
    (u64, u64, u64),
);

fn dict(name: &str) -> URef {
    runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

fn contract_purse() -> URef {
    runtime::get_key(CONTRACT_PURSE)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

fn get_uref(name: &str) -> URef {
    runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

// Decrease the bookkeeping total when funds leave the contract purse
// (unstake, slash payout) so `available` in record_stake reflects the
// purse's real remaining balance instead of drifting to zero forever.
fn decrease_total_recorded(amount: U512) {
    let total_uref = get_uref(KEY_TOTAL_RECORDED);
    let total_recorded: U512 = storage::read(total_uref).unwrap_or_revert().unwrap_or(U512::zero());
    storage::write(total_uref, total_recorded.saturating_sub(amount));
}

fn validate_proof_id(pid: &str) {
    if pid.is_empty() || pid.len() > MAX_PROOF_ID_LEN {
        runtime::revert(ApiError::User(ERR_INPUT_TOO_LONG));
    }
}

fn get_proof_from_registry(pid: &str) -> ProofRec {
    let registry = runtime::get_key(PROOF_REGISTRY_HASH).unwrap_or_revert();
    let mut args = RuntimeArgs::new();
    args.insert("proof_id", pid.to_string()).unwrap_or_revert();
    runtime::call_contract(
        registry.into_hash_addr().unwrap_or_revert().into(),
        "get_proof",
        args,
    )
}

fn read_stake(stakes: URef, agent: &str) -> U512 {
    storage::dictionary_get::<U512>(stakes, agent)
        .unwrap_or_revert()
        .unwrap_or(U512::zero())
}

/// Returns this contract's purse URef, so the companion session code can
/// transfer CSPR into it as part of the same deploy that then calls
/// record_stake.
#[unsafe(no_mangle)]
pub extern "C" fn get_purse() {
    runtime::ret(CLValue::from_t(contract_purse()).unwrap_or_revert());
}

/// Records that `amount` of CSPR was just deposited into this contract's
/// purse on behalf of the caller. Intended to be invoked as the second half
/// of the atomic stake-slashing-session deploy (transfer, then this call).
/// See the hardening note below: an out-of-band call with no real transfer
/// behind it can no longer inflate the caller's recorded stake.
#[unsafe(no_mangle)]
pub extern "C" fn record_stake() {
    // 2026-07-18 hardening: previously trusted the caller's claimed `amount`
    // with no check that a matching purse transfer ever happened — calling
    // this directly (skipping stake-slashing-session) let anyone inflate
    // their own recorded stake with unbacked amounts. A contract can't pull
    // funds from the caller's purse itself (get_main_purse() only works in
    // session context, which is why the session exists), so instead we make
    // record_stake self-verifying: it can never credit more, in total across
    // all callers, than has actually landed in the contract purse. Any
    // out-of-band call with no real transfer behind it is capped at 0.
    let caller = runtime::get_caller();
    let claimed: U512 = runtime::get_named_arg("amount");

    let total_uref = get_uref(KEY_TOTAL_RECORDED);
    let total_recorded: U512 = storage::read(total_uref).unwrap_or_revert().unwrap_or(U512::zero());
    let actual_balance = system::get_purse_balance(contract_purse()).unwrap_or_revert();
    let available = actual_balance.saturating_sub(total_recorded);
    let credit = if claimed > available { available } else { claimed };
    if credit.is_zero() {
        runtime::revert(ApiError::User(ERR_NO_MATCHING_TRANSFER));
    }

    let stakes = dict(STAKES_DICT);
    let current = read_stake(stakes, &caller.to_string());
    let updated = current.checked_add(credit).unwrap_or_else(|| runtime::revert(ApiError::User(99)));
    storage::dictionary_put(stakes, &caller.to_string(), updated);
    storage::write(total_uref, total_recorded.saturating_add(credit));
}

/// Withdraw up to your current recorded stake back to your own account.
#[unsafe(no_mangle)]
pub extern "C" fn unstake() {
    let caller = runtime::get_caller();
    let amount: U512 = runtime::get_named_arg("amount");

    let stakes = dict(STAKES_DICT);
    let current = read_stake(stakes, &caller.to_string());
    let remaining = current.checked_sub(amount)
        .unwrap_or_else(|| runtime::revert(ApiError::User(ERR_INSUFFICIENT_STAKE)));
    storage::dictionary_put(stakes, &caller.to_string(), remaining);

    system::transfer_from_purse_to_account(contract_purse(), caller, amount, None)
        .unwrap_or_revert();
    decrease_total_recorded(amount);
}

/// Permissionlessly claim a slash reward for an agent whose proof has
/// already been revoked on proof-registry. Reads proof-registry's own
/// on-chain state - does not (and cannot) force a revocation itself.
#[unsafe(no_mangle)]
pub extern "C" fn report_and_slash() {
    let agent: AccountHash = runtime::get_named_arg("agent");
    let pid: String = runtime::get_named_arg("proof_id");
    validate_proof_id(&pid);

    let slashed = dict(SLASHED_DICT);
    let already: Option<u64> = storage::dictionary_get(slashed, &pid).unwrap_or_revert();
    if already.is_some() {
        runtime::revert(ApiError::User(ERR_ALREADY_SLASHED));
    }

    let ((_, proof_agent, _), _, (_, _, revoked)) = get_proof_from_registry(&pid);
    if proof_agent != agent.to_string() {
        runtime::revert(ApiError::User(ERR_AGENT_MISMATCH));
    }
    if revoked != 1 {
        runtime::revert(ApiError::User(ERR_NOT_REVOKED));
    }

    let stakes = dict(STAKES_DICT);
    let current = read_stake(stakes, &agent.to_string());
    let slash_amount = current * U512::from(SLASH_BPS) / U512::from(10_000u64);
    let remaining = if slash_amount > current {
        U512::zero()
    } else {
        current - slash_amount
    };

    storage::dictionary_put(stakes, &agent.to_string(), remaining);
    storage::dictionary_put(slashed, &pid, 1u64);

    if slash_amount > U512::zero() {
        let caller = runtime::get_caller();
        system::transfer_from_purse_to_account(contract_purse(), caller, slash_amount, None)
            .unwrap_or_revert();
        decrease_total_recorded(slash_amount);
    }
}

/// Read an agent's current recorded stake.
#[unsafe(no_mangle)]
pub extern "C" fn get_stake() {
    let agent: AccountHash = runtime::get_named_arg("agent");
    let stakes = dict(STAKES_DICT);
    let amount = read_stake(stakes, &agent.to_string());
    runtime::ret(CLValue::from_t(amount).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn call() {
    let proof_registry: Key = runtime::get_named_arg("proof_registry");

    let stakes = storage::new_dictionary(STAKES_DICT).unwrap_or_revert();
    let slashed = storage::new_dictionary(SLASHED_DICT).unwrap_or_revert();
    let purse = system::create_purse();
    let total_recorded = storage::new_uref(U512::zero());

    let mut nk = NamedKeys::new();
    nk.insert(PROOF_REGISTRY_HASH.into(), proof_registry);
    nk.insert(STAKES_DICT.into(), stakes.into());
    nk.insert(SLASHED_DICT.into(), slashed.into());
    nk.insert(CONTRACT_PURSE.into(), purse.into());
    nk.insert(KEY_TOTAL_RECORDED.into(), total_recorded.into());

    let mut ep = EntryPoints::new();
    ep.add_entry_point(EntityEntryPoint::new(
        "get_purse",
        vec![],
        CLType::URef,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "record_stake",
        vec![Parameter::new("amount", CLType::U512)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "unstake",
        vec![Parameter::new("amount", CLType::U512)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "report_and_slash",
        vec![
            Parameter::new("agent", CLType::ByteArray(32)),
            Parameter::new("proof_id", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "get_stake",
        vec![Parameter::new("agent", CLType::ByteArray(32))],
        CLType::U512,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    let (contract_hash, contract_version) =
        storage::new_contract(ep, Some(nk), Some("stake_slashing_package".to_string()), None, None);

    runtime::put_key("stake_slashing_hash", contract_hash.into());
    runtime::put_key(
        "stake_slashing_version",
        storage::new_uref(contract_version).into(),
    );
}
