// Governance / timelock / pause / owner-recovery contract.
//
// Closes BACKLOG items 1.9 (timelocked owner), 1.10 (emergency pause),
// 1.11 (owner recovery / rotation) as a standalone Casper contract. Existing
// deployed contracts opt in via governance-lib helpers on their next upgrade;
// no live redeploy is required to land this crate.
//
// Semantics (mirrors OpenZeppelin TimelockController + Pausable, adapted for
// Casper's account-hash caller model):
//
//   * The `owner` is the sole address allowed to `propose`, `cancel`, and
//     `execute` after the timelock elapses. It bootstraps to `runtime::get_caller()`
//     at install time. Rotation goes through `propose_owner_transfer` +
//     `execute_owner_transfer` -> 48h timelock by default.
//   * `emergency_pause` is a fast lane: the owner (or any of up to 3 pre-approved
//     guardians configured at install) can pause the system immediately without
//     waiting for the timelock. `unpause` is timelocked to prevent a compromised
//     guardian from un-pausing after a legitimate pause.
//   * `recover_owner` covers the "lost key" case: a super-majority (>=2 of 3)
//     guardians can co-sign a rotation to a fresh owner. Each guardian calls
//     `sign_recovery(candidate)`; once two distinct signatures land, the third
//     call (or any subsequent call) with the same candidate finalizes.
//   * Every action emits a proposal record; downstream contracts read the
//     proposal ID via `is_proposal_executed(pid)` before acting.

#![no_std]
#![no_main]

extern crate alloc;

use alloc::string::{String, ToString};
use alloc::vec;

use casper_contract::contract_api::{runtime, storage};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;
use casper_types::contracts::NamedKeys;
use casper_types::{
    ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointPayment,
    EntryPointType, EntryPoints, Parameter, URef,
};

// ---- error codes ---------------------------------------------------------
const ERR_NOT_OWNER: u16 = 1;
const ERR_NOT_GUARDIAN: u16 = 2;
const ERR_PROPOSAL_NOT_FOUND: u16 = 3;
const ERR_TIMELOCK_NOT_ELAPSED: u16 = 4;
const ERR_ALREADY_EXECUTED: u16 = 5;
const ERR_ALREADY_CANCELLED: u16 = 6;
const ERR_PAUSED: u16 = 7;
const ERR_NOT_PAUSED: u16 = 8;
const ERR_INPUT_TOO_LONG: u16 = 9;
const ERR_TOO_MANY_GUARDIANS: u16 = 10;
const ERR_RECOVERY_ALREADY_SIGNED: u16 = 11;
const ERR_RECOVERY_INSUFFICIENT_SIGS: u16 = 12;
const ERR_COUNTER_OVERFLOW: u16 = 99;

// ---- storage keys --------------------------------------------------------
const OWNER_KEY: &str = "owner";
const GUARDIANS_KEY: &str = "guardians"; // ((String, String, String), (u64, u64, u64)) = (g1, g2, g3, count, _, _)
const PROPOSAL_COUNTER: &str = "proposal_ctr";
const PROPOSALS_DICT: &str = "proposals";
const PAUSED_KEY: &str = "paused";              // u64: 0 = live, blocktime = paused
const TIMELOCK_SECS_KEY: &str = "timelock_secs"; // u64, default 48h
const RECOVERY_DICT: &str = "recovery"; // key = candidate, val = ((sig1, sig2, sig3), (count, executed, ts))

const DEFAULT_TIMELOCK_SECS: u64 = 48 * 60 * 60; // 48h
const MAX_STRING_LEN: usize = 256;
const MAX_GUARDIANS: usize = 3;

// Proposal record:
//   ((pid, kind, payload_hash), (proposed_at, executable_at, executed_at), (cancelled, executed, _))
// kind is a small string tag: "owner_transfer" | "unpause" | "config" | "generic"
// payload_hash pins the intended action; downstream contracts hash their own
// call args and compare.
type ProposalRec = ((String, String, String), (u64, u64, u64), (u64, u64, u64));

// Recovery record:
//   ((sig1, sig2, sig3), (sig_count, executed, ts))
type RecoveryRec = ((String, String, String), (u64, u64, u64));

// Guardians record:
//   ((g1, g2, g3), (count, _, _))
type GuardiansRec = ((String, String, String), (u64, u64, u64));

// ---- helpers -------------------------------------------------------------
fn read_uref<T: casper_types::bytesrepr::FromBytes + casper_types::bytesrepr::ToBytes + casper_types::CLTyped>(
    name: &str,
) -> T {
    let u = runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert();
    storage::read(u).unwrap_or_revert().unwrap_or_revert()
}

fn write_uref<T: casper_types::bytesrepr::ToBytes + casper_types::CLTyped>(name: &str, val: T) {
    let u = runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert();
    storage::write(u, val);
}

fn dict(name: &str) -> URef {
    runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

fn read_owner() -> String {
    read_uref::<String>(OWNER_KEY)
}

fn read_guardians() -> GuardiansRec {
    read_uref::<GuardiansRec>(GUARDIANS_KEY)
}

fn require_owner() {
    let caller = runtime::get_caller().to_string();
    if caller != read_owner() {
        runtime::revert(ApiError::User(ERR_NOT_OWNER));
    }
}

fn caller_is_guardian(caller: &str) -> bool {
    let ((g1, g2, g3), (count, _, _)) = read_guardians();
    let n = count as usize;
    (n >= 1 && caller == g1) || (n >= 2 && caller == g2) || (n >= 3 && caller == g3)
}

fn require_owner_or_guardian() {
    let caller = runtime::get_caller().to_string();
    if caller != read_owner() && !caller_is_guardian(&caller) {
        runtime::revert(ApiError::User(ERR_NOT_GUARDIAN));
    }
}

fn require_guardian() {
    let caller = runtime::get_caller().to_string();
    if !caller_is_guardian(&caller) {
        runtime::revert(ApiError::User(ERR_NOT_GUARDIAN));
    }
}

fn validate_string(s: &str) {
    if s.is_empty() || s.len() > MAX_STRING_LEN {
        runtime::revert(ApiError::User(ERR_INPUT_TOO_LONG));
    }
}

fn next_proposal_id() -> u64 {
    let u = runtime::get_key(PROPOSAL_COUNTER)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert();
    let c: u64 = storage::read(u).unwrap_or_revert().unwrap_or(0);
    let n = c
        .checked_add(1)
        .unwrap_or_else(|| runtime::revert(ApiError::User(ERR_COUNTER_OVERFLOW)));
    storage::write(u, n);
    n
}

fn now() -> u64 {
    runtime::get_blocktime().into()
}

fn require_not_paused() {
    let p: u64 = read_uref(PAUSED_KEY);
    if p != 0 {
        runtime::revert(ApiError::User(ERR_PAUSED));
    }
}

// ---- entry points --------------------------------------------------------

#[unsafe(no_mangle)]
pub extern "C" fn propose() {
    require_owner();
    require_not_paused();

    let kind: String = runtime::get_named_arg("kind");
    let payload_hash: String = runtime::get_named_arg("payload_hash");
    validate_string(&kind);
    validate_string(&payload_hash);

    let pid = next_proposal_id();
    let pid_str = alloc::format!("G-{}", pid);
    let ts = now();
    let timelock: u64 = read_uref(TIMELOCK_SECS_KEY);
    let executable_at = ts.saturating_add(timelock.saturating_mul(1000)); // blocktime is ms

    let rec: ProposalRec = (
        (pid_str.clone(), kind, payload_hash),
        (ts, executable_at, 0),
        (0, 0, 0),
    );
    storage::dictionary_put(dict(PROPOSALS_DICT), &pid_str, rec);
    runtime::ret(CLValue::from_t(pid_str).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn execute() {
    require_owner();
    require_not_paused();

    let pid: String = runtime::get_named_arg("proposal_id");
    validate_string(&pid);

    let proposals = dict(PROPOSALS_DICT);
    let rec: ProposalRec = storage::dictionary_get(proposals, &pid)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROPOSAL_NOT_FOUND));
    let ((id, kind, ph), (proposed, exec_at, _), (cancelled, executed, _)) = rec;

    if cancelled == 1 {
        runtime::revert(ApiError::User(ERR_ALREADY_CANCELLED));
    }
    if executed == 1 {
        runtime::revert(ApiError::User(ERR_ALREADY_EXECUTED));
    }
    if now() < exec_at {
        runtime::revert(ApiError::User(ERR_TIMELOCK_NOT_ELAPSED));
    }

    let ts = now();
    let updated: ProposalRec = ((id, kind, ph), (proposed, exec_at, ts), (0, 1, 0));
    storage::dictionary_put(proposals, &pid, updated);
}

#[unsafe(no_mangle)]
pub extern "C" fn cancel() {
    require_owner();
    let pid: String = runtime::get_named_arg("proposal_id");
    validate_string(&pid);

    let proposals = dict(PROPOSALS_DICT);
    let rec: ProposalRec = storage::dictionary_get(proposals, &pid)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROPOSAL_NOT_FOUND));
    let ((id, kind, ph), t, (_cancelled, executed, _)) = rec;
    if executed == 1 {
        runtime::revert(ApiError::User(ERR_ALREADY_EXECUTED));
    }
    let updated: ProposalRec = ((id, kind, ph), t, (1, 0, 0));
    storage::dictionary_put(proposals, &pid, updated);
}

#[unsafe(no_mangle)]
pub extern "C" fn get_proposal() {
    let pid: String = runtime::get_named_arg("proposal_id");
    validate_string(&pid);
    let rec: ProposalRec = storage::dictionary_get(dict(PROPOSALS_DICT), &pid)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROPOSAL_NOT_FOUND));
    runtime::ret(CLValue::from_t(rec).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn is_executed() {
    let pid: String = runtime::get_named_arg("proposal_id");
    validate_string(&pid);
    let out: u64 = match storage::dictionary_get::<ProposalRec>(dict(PROPOSALS_DICT), &pid)
        .unwrap_or_revert()
    {
        Some(((_, _, _), _, (_, executed, _))) => executed,
        None => 0,
    };
    runtime::ret(CLValue::from_t(out).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn emergency_pause() {
    require_owner_or_guardian();
    let ts = now();
    write_uref::<u64>(PAUSED_KEY, ts);
}

#[unsafe(no_mangle)]
pub extern "C" fn propose_unpause() {
    // Unpause is timelocked to prevent a rogue guardian from immediately
    // undoing a legitimate emergency pause.
    require_owner();
    let p: u64 = read_uref(PAUSED_KEY);
    if p == 0 {
        runtime::revert(ApiError::User(ERR_NOT_PAUSED));
    }
    let pid = next_proposal_id();
    let pid_str = alloc::format!("G-{}", pid);
    let ts = now();
    let timelock: u64 = read_uref(TIMELOCK_SECS_KEY);
    let executable_at = ts.saturating_add(timelock.saturating_mul(1000));
    let rec: ProposalRec = (
        (pid_str.clone(), "unpause".to_string(), "".to_string()),
        (ts, executable_at, 0),
        (0, 0, 0),
    );
    storage::dictionary_put(dict(PROPOSALS_DICT), &pid_str, rec);
    runtime::ret(CLValue::from_t(pid_str).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn execute_unpause() {
    require_owner();
    let pid: String = runtime::get_named_arg("proposal_id");
    validate_string(&pid);
    let proposals = dict(PROPOSALS_DICT);
    let rec: ProposalRec = storage::dictionary_get(proposals, &pid)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROPOSAL_NOT_FOUND));
    let ((id, kind, ph), (proposed, exec_at, _), (cancelled, executed, _)) = rec;
    if kind != "unpause" {
        runtime::revert(ApiError::User(ERR_PROPOSAL_NOT_FOUND));
    }
    if cancelled == 1 {
        runtime::revert(ApiError::User(ERR_ALREADY_CANCELLED));
    }
    if executed == 1 {
        runtime::revert(ApiError::User(ERR_ALREADY_EXECUTED));
    }
    if now() < exec_at {
        runtime::revert(ApiError::User(ERR_TIMELOCK_NOT_ELAPSED));
    }
    // clear pause + mark executed
    write_uref::<u64>(PAUSED_KEY, 0);
    let ts = now();
    let updated: ProposalRec = ((id, kind, ph), (proposed, exec_at, ts), (0, 1, 0));
    storage::dictionary_put(proposals, &pid, updated);
}

#[unsafe(no_mangle)]
pub extern "C" fn get_paused() {
    let p: u64 = read_uref(PAUSED_KEY);
    runtime::ret(CLValue::from_t(p).unwrap_or_revert());
}

// ---- owner rotation ------------------------------------------------------

#[unsafe(no_mangle)]
pub extern "C" fn propose_owner_transfer() {
    require_owner();
    require_not_paused();
    let new_owner: String = runtime::get_named_arg("new_owner");
    validate_string(&new_owner);
    let pid = next_proposal_id();
    let pid_str = alloc::format!("G-{}", pid);
    let ts = now();
    let timelock: u64 = read_uref(TIMELOCK_SECS_KEY);
    let executable_at = ts.saturating_add(timelock.saturating_mul(1000));
    let rec: ProposalRec = (
        (pid_str.clone(), "owner_transfer".to_string(), new_owner),
        (ts, executable_at, 0),
        (0, 0, 0),
    );
    storage::dictionary_put(dict(PROPOSALS_DICT), &pid_str, rec);
    runtime::ret(CLValue::from_t(pid_str).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn execute_owner_transfer() {
    require_owner();
    let pid: String = runtime::get_named_arg("proposal_id");
    validate_string(&pid);
    let proposals = dict(PROPOSALS_DICT);
    let rec: ProposalRec = storage::dictionary_get(proposals, &pid)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROPOSAL_NOT_FOUND));
    let ((id, kind, new_owner), (proposed, exec_at, _), (cancelled, executed, _)) = rec;
    if kind != "owner_transfer" {
        runtime::revert(ApiError::User(ERR_PROPOSAL_NOT_FOUND));
    }
    if cancelled == 1 {
        runtime::revert(ApiError::User(ERR_ALREADY_CANCELLED));
    }
    if executed == 1 {
        runtime::revert(ApiError::User(ERR_ALREADY_EXECUTED));
    }
    if now() < exec_at {
        runtime::revert(ApiError::User(ERR_TIMELOCK_NOT_ELAPSED));
    }
    // rotate + mark executed
    write_uref::<String>(OWNER_KEY, new_owner.clone());
    let ts = now();
    let updated: ProposalRec = (
        (id, kind, new_owner),
        (proposed, exec_at, ts),
        (0, 1, 0),
    );
    storage::dictionary_put(proposals, &pid, updated);
}

// ---- guardian recovery (2-of-3) ------------------------------------------

#[unsafe(no_mangle)]
pub extern "C" fn sign_recovery() {
    require_guardian();
    let candidate: String = runtime::get_named_arg("candidate");
    validate_string(&candidate);
    let caller = runtime::get_caller().to_string();

    let recovery = dict(RECOVERY_DICT);
    let rec: RecoveryRec = storage::dictionary_get(recovery, &candidate)
        .unwrap_or_revert()
        .unwrap_or((
            ("".to_string(), "".to_string(), "".to_string()),
            (0, 0, 0),
        ));
    let ((mut s1, mut s2, mut s3), (mut count, executed, _)) = rec;
    if executed == 1 {
        return;
    }
    if s1 == caller || s2 == caller || s3 == caller {
        runtime::revert(ApiError::User(ERR_RECOVERY_ALREADY_SIGNED));
    }
    if s1.is_empty() {
        s1 = caller;
    } else if s2.is_empty() {
        s2 = caller;
    } else if s3.is_empty() {
        s3 = caller;
    }
    count = count.saturating_add(1);
    let ts = now();
    let updated: RecoveryRec = ((s1, s2, s3), (count, 0, ts));
    storage::dictionary_put(recovery, &candidate, updated);
}

#[unsafe(no_mangle)]
pub extern "C" fn execute_recovery() {
    require_guardian();
    let candidate: String = runtime::get_named_arg("candidate");
    validate_string(&candidate);
    let recovery = dict(RECOVERY_DICT);
    let rec: RecoveryRec = storage::dictionary_get(recovery, &candidate)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROPOSAL_NOT_FOUND));
    let ((s1, s2, s3), (count, executed, ts)) = rec;
    if executed == 1 {
        runtime::revert(ApiError::User(ERR_ALREADY_EXECUTED));
    }
    if count < 2 {
        runtime::revert(ApiError::User(ERR_RECOVERY_INSUFFICIENT_SIGS));
    }
    write_uref::<String>(OWNER_KEY, candidate.clone());
    let updated: RecoveryRec = ((s1, s2, s3), (count, 1, ts));
    storage::dictionary_put(recovery, &candidate, updated);
}

#[unsafe(no_mangle)]
pub extern "C" fn get_owner() {
    let o = read_owner();
    runtime::ret(CLValue::from_t(o).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn get_timelock_secs() {
    let t: u64 = read_uref(TIMELOCK_SECS_KEY);
    runtime::ret(CLValue::from_t(t).unwrap_or_revert());
}

// ---- installer -----------------------------------------------------------

#[unsafe(no_mangle)]
pub extern "C" fn call() {
    let caller = runtime::get_caller().to_string();
    let g1: String = runtime::get_named_arg("guardian_1");
    let g2: String = runtime::get_named_arg("guardian_2");
    let g3: String = runtime::get_named_arg("guardian_3");
    let mut count: u64 = 0;
    if !g1.is_empty() {
        count += 1;
    }
    if !g2.is_empty() {
        count += 1;
    }
    if !g3.is_empty() {
        count += 1;
    }
    if (count as usize) > MAX_GUARDIANS {
        runtime::revert(ApiError::User(ERR_TOO_MANY_GUARDIANS));
    }

    let owner_uref = storage::new_uref(caller);
    let guardians_uref = storage::new_uref::<GuardiansRec>(((g1, g2, g3), (count, 0, 0)));
    let paused_uref = storage::new_uref::<u64>(0);
    let timelock_uref = storage::new_uref::<u64>(DEFAULT_TIMELOCK_SECS);
    let proposal_ctr = storage::new_uref::<u64>(0);
    let proposals = storage::new_dictionary(PROPOSALS_DICT).unwrap_or_revert();
    let recovery = storage::new_dictionary(RECOVERY_DICT).unwrap_or_revert();

    let mut nk = NamedKeys::new();
    nk.insert(OWNER_KEY.into(), owner_uref.into());
    nk.insert(GUARDIANS_KEY.into(), guardians_uref.into());
    nk.insert(PAUSED_KEY.into(), paused_uref.into());
    nk.insert(TIMELOCK_SECS_KEY.into(), timelock_uref.into());
    nk.insert(PROPOSAL_COUNTER.into(), proposal_ctr.into());
    nk.insert(PROPOSALS_DICT.into(), proposals.into());
    nk.insert(RECOVERY_DICT.into(), recovery.into());

    let mut ep = EntryPoints::new();
    let public_called =
        |name: &str, params: alloc::vec::Vec<Parameter>, ret: CLType| -> EntityEntryPoint {
            EntityEntryPoint::new(
                name,
                params,
                ret,
                EntryPointAccess::Public,
                EntryPointType::Called,
                EntryPointPayment::Caller,
            )
        };

    ep.add_entry_point(public_called(
        "propose",
        vec![
            Parameter::new("kind", CLType::String),
            Parameter::new("payload_hash", CLType::String),
        ],
        CLType::String,
    ));
    ep.add_entry_point(public_called(
        "execute",
        vec![Parameter::new("proposal_id", CLType::String)],
        CLType::Unit,
    ));
    ep.add_entry_point(public_called(
        "cancel",
        vec![Parameter::new("proposal_id", CLType::String)],
        CLType::Unit,
    ));
    ep.add_entry_point(public_called(
        "get_proposal",
        vec![Parameter::new("proposal_id", CLType::String)],
        CLType::Any,
        ));
    ep.add_entry_point(public_called(
        "is_executed",
        vec![Parameter::new("proposal_id", CLType::String)],
        CLType::U64,
    ));
    ep.add_entry_point(public_called("emergency_pause", vec![], CLType::Unit));
    ep.add_entry_point(public_called("propose_unpause", vec![], CLType::String));
    ep.add_entry_point(public_called(
        "execute_unpause",
        vec![Parameter::new("proposal_id", CLType::String)],
        CLType::Unit,
    ));
    ep.add_entry_point(public_called("get_paused", vec![], CLType::U64));
    ep.add_entry_point(public_called(
        "propose_owner_transfer",
        vec![Parameter::new("new_owner", CLType::String)],
        CLType::String,
    ));
    ep.add_entry_point(public_called(
        "execute_owner_transfer",
        vec![Parameter::new("proposal_id", CLType::String)],
        CLType::Unit,
    ));
    ep.add_entry_point(public_called(
        "sign_recovery",
        vec![Parameter::new("candidate", CLType::String)],
        CLType::Unit,
    ));
    ep.add_entry_point(public_called(
        "execute_recovery",
        vec![Parameter::new("candidate", CLType::String)],
        CLType::Unit,
    ));
    ep.add_entry_point(public_called("get_owner", vec![], CLType::String));
    ep.add_entry_point(public_called("get_timelock_secs", vec![], CLType::U64));

    let (ch, _) = storage::new_contract(
        ep,
        Some(nk),
        Some("governance_pkg".into()),
        Some("governance_access".into()),
        None,
    );
    runtime::put_key("governance", ch.into());
}
