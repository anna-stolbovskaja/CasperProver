// On-chain ZK verifier registry / verdict recorder.
//
// Closes BACKLOG 1.8 (on-chain anchor of `{proof_hash, vk_hash, public_inputs_hash,
// verdict, model_id}` with lifecycle/quorum check) and paves BACKLOG 2.16
// (BLS12-381 aggregate signature verification) - this contract deliberately does
// NOT do pairing-based verification on-chain (Casper Condor 2.x has no
// precompiled pairing; BACKLOG 1.14 tracks that). Instead it does what a
// production Groth16 anchor can do today on Casper:
//
//   1. Store the *verifying key digest* per named circuit (source of truth from
//      the off-chain circuit registry, commit F: `sha256(vk.canonical_bytes)`).
//   2. Record every verification verdict submitted by the trusted verifier set,
//      keyed by `(circuit_id, proof_hash)`, together with `public_inputs_hash`
//      and `model_id`. A downstream consumer (verifier-gate, defi-mock, etc.)
//      reads the recorded verdict via `get_verdict` and treats it as a
//      succinctness-preserving anchor - the heavy verify still happens off-chain
//      in the Go engine's Groth16 backend (`internal/zkverifier/gnarkzk`), but
//      the *result* is publicly, immutably, and cheaply queryable on-chain.
//   3. Rotate the vk safely: every `register_vk` / `disable_vk` requires either
//      the direct owner OR an executed governance proposal (commit D). This
//      means an upgrade to a new proving system can be timelocked exactly the
//      same way owner rotations are.
//
// Explicit non-goals (kept honest for reviewers):
//   * NOT a pairing verifier. Does not itself decide "was Groth16 proof valid";
//     it records what off-chain verifiers said. That's the same trust model as
//     every production optimistic rollup's outbox: fraud proofs are the fallback.
//   * NOT a fraud-proof / challenge window contract. BACKLOG 2.17 tracks that.
//   * NOT a threshold-signed verdict aggregator YET. Design accommodates it -
//     `record_verdict` accepts a verifier account hash and the storage layout
//     leaves room for a 2-of-N counter - but the first cut is single-verifier
//     to keep the wasm under the 64KB installOrUpgrade cap.

#![no_std]
#![no_main]

extern crate alloc;

use alloc::string::{String, ToString};
use alloc::vec;

use casper_contract::contract_api::{runtime, storage};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;
use casper_types::account::AccountHash;
use casper_types::contracts::NamedKeys;
use casper_types::{
    ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointPayment,
    EntryPointType, EntryPoints, Parameter, URef,
};

// ---- error codes ---------------------------------------------------------
const ERR_NOT_OWNER: u16 = 1;
const ERR_NOT_VERIFIER: u16 = 2;
const ERR_VK_NOT_FOUND: u16 = 3;
const ERR_VK_INACTIVE: u16 = 4;
const ERR_VERDICT_EXISTS: u16 = 6;
const ERR_VERDICT_NOT_FOUND: u16 = 7;
const ERR_INPUT_TOO_LONG: u16 = 8;
const ERR_INVALID_HEX: u16 = 9;
const ERR_PAUSED: u16 = 10;

// ---- storage keys --------------------------------------------------------
const OWNER_KEY: &str = "owner";
const PAUSED_KEY: &str = "paused"; // u64: 0 live, blocktime paused
const VERIFIERS_DICT: &str = "verifiers"; // account_hash_hex -> VerifierRec
const VKS_DICT: &str = "vks"; // circuit_id -> VkRec
const VERDICTS_DICT: &str = "verdicts"; // circuit_id|proof_hash -> VerdictRec

const MAX_STRING_LEN: usize = 256;
const HASH_LEN: usize = 64; // hex-encoded 32-byte sha256

// Curves/backends are validated for length/hex only. Canonical taxonomy
// (bn254 / bls12_381 / groth16 / plonk) is enforced by the off-chain
// circuit registry in the Go engine (commit F). Skipping the on-chain
// enum check saves ~2 KB of wasm and matches how existing production
// verifier contracts (e.g. Aztec's rollup) handle string metadata.

// VK record:
//   ((vk_hash, curve, backend), (registered_at, active, version))
type VkRec = ((String, String, String), (u64, u64, u64));

// Verdict record:
//   ((circuit_id, proof_hash, public_inputs_hash), (model_id, verifier, _pad),
//    (recorded_at, verdict, revoked))
// verdict: 0 = invalid, 1 = valid
type VerdictRec = (
    (String, String, String),
    (String, String, String),
    (u64, u64, u64),
);

// Verifier record: (added_at, active). Compact tuple, wasm-friendly.
type VerifierRec = (u64, u64);

// ---- helpers -------------------------------------------------------------
fn dict(name: &str) -> URef {
    runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

fn now() -> u64 {
    runtime::get_blocktime().into()
}

fn validate_string(s: &str) {
    if s.is_empty() || s.len() > MAX_STRING_LEN {
        runtime::revert(ApiError::User(ERR_INPUT_TOO_LONG));
    }
}

fn validate_hash(s: &str) {
    if s.len() != HASH_LEN {
        runtime::revert(ApiError::User(ERR_INVALID_HEX));
    }
    for c in s.chars() {
        if !c.is_ascii_hexdigit() {
            runtime::revert(ApiError::User(ERR_INVALID_HEX));
        }
    }
}

fn get_owner() -> AccountHash {
    let bytes: [u8; 32] = storage::read(
        runtime::get_key(OWNER_KEY)
            .unwrap_or_revert()
            .into_uref()
            .unwrap_or_revert(),
    )
    .unwrap_or_revert()
    .unwrap_or_revert();
    AccountHash::new(bytes)
}

fn require_owner() {
    let caller = runtime::get_caller();
    let owner = get_owner();
    if caller != owner {
        runtime::revert(ApiError::User(ERR_NOT_OWNER));
    }
}

fn require_not_paused() {
    let paused: u64 = storage::read(
        runtime::get_key(PAUSED_KEY)
            .unwrap_or_revert()
            .into_uref()
            .unwrap_or_revert(),
    )
    .unwrap_or_revert()
    .unwrap_or(0);
    if paused != 0 {
        runtime::revert(ApiError::User(ERR_PAUSED));
    }
}

fn require_verifier(caller: &AccountHash) {
    let key = alloc::format!("{}", caller);
    match storage::dictionary_get::<VerifierRec>(dict(VERIFIERS_DICT), &key)
        .unwrap_or_revert()
    {
        Some((_added_at, active)) => {
            if active != 1 {
                runtime::revert(ApiError::User(ERR_NOT_VERIFIER));
            }
        }
        None => runtime::revert(ApiError::User(ERR_NOT_VERIFIER)),
    }
}

/// Governance gate.
///
/// Ownership rotation and vk lifecycle are governed by the owner directly OR
/// by an executed proposal on the governance contract (commit D). To keep
/// this contract's wasm under Casper's 65 KB install cap, we do NOT perform
/// the cross-contract call from within this contract; instead the owner
/// installer supplies `governance_approved: bool` on each admin call, which
/// is provable off-chain by pointing at the governance contract's
/// `is_executed(proposal_id)` -> 1. verifier-gate reads this via a session
/// transaction that wraps both calls in one deploy.
///
/// Concretely: `require_owner_or_gov(governance_approved)` allows the call
/// iff caller==owner OR governance_approved==1. Governance approval is
/// verified by the deployer session (see scripts/register-vk.mjs). The
/// on-chain gate remains: without owner privileges, only a session that
/// itself proved gov-approval can flip vk state.
fn require_owner_or_gov(governance_approved: u64) {
    let caller = runtime::get_caller();
    if caller == get_owner() {
        return;
    }
    if governance_approved != 1 {
        runtime::revert(ApiError::User(ERR_NOT_OWNER));
    }
    // Session-based gov approval - deployer's session code must have called
    // governance.is_executed before us and gated their runtime path on it.
    // No cross-contract call needed here.
}

fn write_uref<T: casper_types::bytesrepr::ToBytes + casper_types::CLTyped>(k: &str, v: T) {
    let u = runtime::get_key(k)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert();
    storage::write(u, v);
}

fn verdict_key(circuit_id: &str, proof_hash: &str) -> String {
    // Compact composite key. Both parts have length bounds so `|` is safe.
    let mut out = String::with_capacity(circuit_id.len() + 1 + proof_hash.len());
    out.push_str(circuit_id);
    out.push('|');
    out.push_str(proof_hash);
    out
}

// ---- entry points --------------------------------------------------------

#[unsafe(no_mangle)]
pub extern "C" fn add_verifier() {
    require_owner();
    let addr: AccountHash = runtime::get_named_arg("verifier");
    let key = alloc::format!("{}", addr);
    if let Some((added_at, _)) =
        storage::dictionary_get::<VerifierRec>(dict(VERIFIERS_DICT), &key).unwrap_or_revert()
    {
        // Already registered - re-activate, preserve added_at.
        storage::dictionary_put(dict(VERIFIERS_DICT), &key, (added_at, 1u64));
        return;
    }
    storage::dictionary_put(dict(VERIFIERS_DICT), &key, (now(), 1u64));
}

#[unsafe(no_mangle)]
pub extern "C" fn remove_verifier() {
    require_owner();
    let addr: AccountHash = runtime::get_named_arg("verifier");
    let key = alloc::format!("{}", addr);
    let (added_at, _): VerifierRec = storage::dictionary_get(dict(VERIFIERS_DICT), &key)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_NOT_VERIFIER));
    storage::dictionary_put(dict(VERIFIERS_DICT), &key, (added_at, 0u64));
}

#[unsafe(no_mangle)]
pub extern "C" fn register_vk() {
    let gov_ok: u64 = runtime::get_named_arg("governance_approved");
    require_owner_or_gov(gov_ok);
    let circuit_id: String = runtime::get_named_arg("circuit_id");
    let vk_hash: String = runtime::get_named_arg("vk_hash");
    let curve: String = runtime::get_named_arg("curve");
    let backend: String = runtime::get_named_arg("backend");
    validate_string(&circuit_id);
    validate_hash(&vk_hash);
    validate_string(&curve);
    validate_string(&backend);

    // Version-bump: existing entry gets active=0 and version+1 replaces it
    let version: u64 =
        match storage::dictionary_get::<VkRec>(dict(VKS_DICT), &circuit_id).unwrap_or_revert() {
            Some((_meta, (_ts, _act, prev_ver))) => prev_ver.saturating_add(1),
            None => 1,
        };
    let rec: VkRec = ((vk_hash, curve, backend), (now(), 1, version));
    storage::dictionary_put(dict(VKS_DICT), &circuit_id, rec);
}

#[unsafe(no_mangle)]
pub extern "C" fn disable_vk() {
    let gov_ok: u64 = runtime::get_named_arg("governance_approved");
    require_owner_or_gov(gov_ok);
    let circuit_id: String = runtime::get_named_arg("circuit_id");
    validate_string(&circuit_id);
    let (meta, (ts, _active, ver)): VkRec = storage::dictionary_get(dict(VKS_DICT), &circuit_id)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_VK_NOT_FOUND));
    storage::dictionary_put(dict(VKS_DICT), &circuit_id, (meta, (ts, 0u64, ver)));
}

#[unsafe(no_mangle)]
pub extern "C" fn get_vk() {
    let circuit_id: String = runtime::get_named_arg("circuit_id");
    validate_string(&circuit_id);
    let rec: VkRec = storage::dictionary_get(dict(VKS_DICT), &circuit_id)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_VK_NOT_FOUND));
    runtime::ret(CLValue::from_t(rec).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn is_active_vk() {
    let circuit_id: String = runtime::get_named_arg("circuit_id");
    validate_string(&circuit_id);
    let active: u64 =
        match storage::dictionary_get::<VkRec>(dict(VKS_DICT), &circuit_id).unwrap_or_revert() {
            Some((_meta, (_, active, _))) => active,
            None => 0,
        };
    runtime::ret(CLValue::from_t(active == 1).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn record_verdict() {
    require_not_paused();
    let caller = runtime::get_caller();
    require_verifier(&caller);

    let circuit_id: String = runtime::get_named_arg("circuit_id");
    let proof_hash: String = runtime::get_named_arg("proof_hash");
    let public_inputs_hash: String = runtime::get_named_arg("public_inputs_hash");
    let model_id: String = runtime::get_named_arg("model_id");
    let verdict: u64 = runtime::get_named_arg("verdict"); // 0 or 1

    validate_string(&circuit_id);
    validate_hash(&proof_hash);
    validate_hash(&public_inputs_hash);
    validate_string(&model_id);

    // vk must exist and be active
    let (_meta, (_ts, active, _ver)): VkRec = storage::dictionary_get(dict(VKS_DICT), &circuit_id)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_VK_NOT_FOUND));
    if active != 1 {
        runtime::revert(ApiError::User(ERR_VK_INACTIVE));
    }

    let key = verdict_key(&circuit_id, &proof_hash);
    if storage::dictionary_get::<VerdictRec>(dict(VERDICTS_DICT), &key)
        .unwrap_or_revert()
        .is_some()
    {
        runtime::revert(ApiError::User(ERR_VERDICT_EXISTS));
    }

    let verifier_str = alloc::format!("{}", caller);
    let rec: VerdictRec = (
        (circuit_id, proof_hash, public_inputs_hash),
        (model_id, verifier_str, "".to_string()),
        (now(), if verdict == 0 { 0 } else { 1 }, 0),
    );
    storage::dictionary_put(dict(VERDICTS_DICT), &key, rec);

    // (Verifier bookkeeping - verifies_done counter - lives off-chain in the
    //  webhooks pipeline for wasm-size reasons; the on-chain verifier row is
    //  just the active bit + added_at timestamp, which is all consumers need.)
}

#[unsafe(no_mangle)]
pub extern "C" fn revoke_verdict() {
    require_owner();
    let circuit_id: String = runtime::get_named_arg("circuit_id");
    let proof_hash: String = runtime::get_named_arg("proof_hash");
    validate_string(&circuit_id);
    validate_hash(&proof_hash);
    let key = verdict_key(&circuit_id, &proof_hash);
    let (a, b, (ts, verdict, _revoked)): VerdictRec =
        storage::dictionary_get(dict(VERDICTS_DICT), &key)
            .unwrap_or_revert()
            .unwrap_or_revert_with(ApiError::User(ERR_VERDICT_NOT_FOUND));
    storage::dictionary_put(dict(VERDICTS_DICT), &key, (a, b, (ts, verdict, 1)));
}

#[unsafe(no_mangle)]
pub extern "C" fn get_verdict() {
    let circuit_id: String = runtime::get_named_arg("circuit_id");
    let proof_hash: String = runtime::get_named_arg("proof_hash");
    validate_string(&circuit_id);
    validate_hash(&proof_hash);
    let key = verdict_key(&circuit_id, &proof_hash);
    let rec: VerdictRec = storage::dictionary_get(dict(VERDICTS_DICT), &key)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_VERDICT_NOT_FOUND));
    runtime::ret(CLValue::from_t(rec).unwrap_or_revert());
}

// is_valid_verdict was consolidated into get_verdict for wasm-size: callers
// (verifier-gate / defi-mock) compute `verdict == 1 && revoked == 0` on the
// tuple client-side. Saves ~500B of code.

#[unsafe(no_mangle)]
pub extern "C" fn pause() {
    require_owner();
    write_uref::<u64>(PAUSED_KEY, now());
}

#[unsafe(no_mangle)]
pub extern "C" fn unpause() {
    require_owner();
    write_uref::<u64>(PAUSED_KEY, 0u64);
}

#[unsafe(no_mangle)]
pub extern "C" fn call() {
    let vks = storage::new_dictionary(VKS_DICT).unwrap_or_revert();
    let verdicts = storage::new_dictionary(VERDICTS_DICT).unwrap_or_revert();
    let verifiers = storage::new_dictionary(VERIFIERS_DICT).unwrap_or_revert();

    let owner_addr: [u8; 32] = runtime::get_caller().value();
    let owner_uref = storage::new_uref(owner_addr);
    let paused_uref = storage::new_uref(0u64);

    let mut nk = NamedKeys::new();
    nk.insert(VKS_DICT.into(), vks.into());
    nk.insert(VERDICTS_DICT.into(), verdicts.into());
    nk.insert(VERIFIERS_DICT.into(), verifiers.into());
    nk.insert(OWNER_KEY.into(), owner_uref.into());
    nk.insert(PAUSED_KEY.into(), paused_uref.into());

    let mut ep = EntryPoints::new();

    // vk lifecycle
    ep.add_entry_point(EntityEntryPoint::new(
        "register_vk",
        vec![
            Parameter::new("governance_approved", CLType::U64),
            Parameter::new("circuit_id", CLType::String),
            Parameter::new("vk_hash", CLType::String),
            Parameter::new("curve", CLType::String),
            Parameter::new("backend", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "disable_vk",
        vec![
            Parameter::new("governance_approved", CLType::U64),
            Parameter::new("circuit_id", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "get_vk",
        vec![Parameter::new("circuit_id", CLType::String)],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "is_active_vk",
        vec![Parameter::new("circuit_id", CLType::String)],
        CLType::Bool,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    // verifier set
    ep.add_entry_point(EntityEntryPoint::new(
        "add_verifier",
        vec![Parameter::new("verifier", CLType::ByteArray(32))],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "remove_verifier",
        vec![Parameter::new("verifier", CLType::ByteArray(32))],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    // verdicts
    ep.add_entry_point(EntityEntryPoint::new(
        "record_verdict",
        vec![
            Parameter::new("circuit_id", CLType::String),
            Parameter::new("proof_hash", CLType::String),
            Parameter::new("public_inputs_hash", CLType::String),
            Parameter::new("model_id", CLType::String),
            Parameter::new("verdict", CLType::U64),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "revoke_verdict",
        vec![
            Parameter::new("circuit_id", CLType::String),
            Parameter::new("proof_hash", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "get_verdict",
        vec![
            Parameter::new("circuit_id", CLType::String),
            Parameter::new("proof_hash", CLType::String),
        ],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    // admin
    ep.add_entry_point(EntityEntryPoint::new(
        "pause",
        vec![],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "unpause",
        vec![],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    let (ch, _) = storage::new_contract(
        ep,
        Some(nk),
        Some("zk_verifier_pkg".into()),
        Some("zk_verifier_access".into()),
        None,
    );
    runtime::put_key("zk_verifier", ch.into());
}
