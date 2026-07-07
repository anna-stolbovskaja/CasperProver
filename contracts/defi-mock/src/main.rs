#![no_std]
#![no_main]

extern crate alloc;

use alloc::string::{String, ToString};
use alloc::vec;

use casper_contract::contract_api::{runtime, storage};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;
use casper_types::account::AccountHash;
use casper_types::contracts::NamedKeys;
use casper_types::{EntryPointPayment, 
    ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointType, EntryPoints, Key,
    Parameter, URef,
};

const ERR_KYC_FAILED: u16 = 10;
const ERR_UNAUTHORIZED: u16 = 11;
const ERR_INPUT_TOO_LONG: u16 = 12;
const ERR_ALREADY_WHITELISTED: u16 = 13;

const VERIFIER_HASH: &str = "verifier_hash";
const WHITELIST_DICT: &str = "whitelist";
const ADMIN_KEY: &str = "admin";
const MAX_PROOF_ID_LEN: usize = 128;

fn dict(name: &str) -> URef {
    runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

fn get_admin() -> AccountHash {
    let key = runtime::get_key(ADMIN_KEY).unwrap_or_revert();
    match key {
        Key::Account(h) => h,
        _ => runtime::revert(ApiError::User(ERR_UNAUTHORIZED)),
    }
}

fn validate_proof_id(pid: &str) {
    if pid.is_empty() || pid.len() > MAX_PROOF_ID_LEN {
        runtime::revert(ApiError::User(ERR_INPUT_TOO_LONG));
    }
}

fn call_is_valid(pid: &str) -> bool {
    let vg = runtime::get_key(VERIFIER_HASH).unwrap_or_revert();
    let mut args = casper_types::RuntimeArgs::new();
    args.insert("proof_id", pid.to_string()).unwrap_or_revert();
    runtime::call_contract(vg.into_hash_addr().unwrap_or_revert().into(), "is_valid", args)
}

/// Verify KYC status without modifying state. Public read-only.
#[unsafe(no_mangle)]
pub extern "C" fn check_kyc() {
    let pid: String = runtime::get_named_arg("proof_id");
    validate_proof_id(&pid);
    let valid = call_is_valid(&pid);
    runtime::ret(CLValue::from_t(valid).unwrap_or_revert());
}

/// Grant DeFi access to a user after KYC proof verification.
/// Only the contract admin (deployer) can invoke this entry point.
#[unsafe(no_mangle)]
pub extern "C" fn grant_access() {
    let caller = runtime::get_caller();
    let admin = get_admin();
    if caller != admin {
        runtime::revert(ApiError::User(ERR_UNAUTHORIZED));
    }

    let user: AccountHash = runtime::get_named_arg("user");
    let pid: String = runtime::get_named_arg("proof_id");
    validate_proof_id(&pid);

    let wl = dict(WHITELIST_DICT);
    let key = user.to_string();

    // Refuse to silently overwrite an existing, still-active entry. A caller
    // who wants to re-whitelist after a revocation must go through
    // revoke_access first, so the on-chain history stays explicit instead of
    // a grant_access call quietly clobbering a prior grant/proof_id/timestamp.
    let existing: Option<(String, String, u64)> =
        storage::dictionary_get(wl, &key).unwrap_or_revert();
    if let Some((_, _, ts)) = existing {
        if ts > 0 {
            runtime::revert(ApiError::User(ERR_ALREADY_WHITELISTED));
        }
    }

    let valid = call_is_valid(&pid);
    if !valid {
        runtime::revert(ApiError::User(ERR_KYC_FAILED));
    }

    let ts: u64 = runtime::get_blocktime().into();
    let entry = (key.clone(), pid, ts);
    storage::dictionary_put(wl, &key, entry);
}

/// Revoke DeFi access for a user. Admin only.
#[unsafe(no_mangle)]
pub extern "C" fn revoke_access() {
    let caller = runtime::get_caller();
    let admin = get_admin();
    if caller != admin {
        runtime::revert(ApiError::User(ERR_UNAUTHORIZED));
    }

    let user: AccountHash = runtime::get_named_arg("user");
    let wl = dict(WHITELIST_DICT);
    // Overwrite with a tombstone entry (empty proof, timestamp 0)
    let revoked: (String, String, u64) = (user.to_string(), String::new(), 0);
    storage::dictionary_put(wl, &user.to_string(), revoked);
}

/// Check if a user is whitelisted. Public read-only.
///
/// Takes a typed `AccountHash` (fixed 32-byte value, enforced by the CLType
/// at the entry-point boundary) rather than a free-form `String`. The
/// previous version accepted an arbitrary caller-supplied string and looked
/// it up directly against the dictionary key, which only worked if the
/// caller happened to format it identically to `AccountHash::to_string()`
/// (lowercase hex, no prefix) - any other casing/prefix/encoding silently
/// returned "not whitelisted" instead of failing loudly. Deriving the lookup
/// key from the same typed value used by grant_access/revoke_access removes
/// that whole class of client-side formatting bugs.
#[unsafe(no_mangle)]
pub extern "C" fn is_whitelisted() {
    let user: AccountHash = runtime::get_named_arg("user");
    let wl = dict(WHITELIST_DICT);
    let entry: Option<(String, String, u64)> =
        storage::dictionary_get(wl, &user.to_string()).unwrap_or_revert();
    // Check entry exists AND is not a revoked tombstone (timestamp > 0)
    let whitelisted = match entry {
        Some((_, _, ts)) => ts > 0,
        None => false,
    };
    runtime::ret(CLValue::from_t(whitelisted).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn call() {
    let verifier: Key = runtime::get_named_arg("verifier_contract");
    let admin = runtime::get_caller();

    let wl = storage::new_dictionary(WHITELIST_DICT).unwrap_or_revert();

    let mut nk = NamedKeys::new();
    nk.insert(VERIFIER_HASH.into(), verifier);
    nk.insert(WHITELIST_DICT.into(), wl.into());
    nk.insert(ADMIN_KEY.into(), Key::Account(admin));

    let mut ep = EntryPoints::new();
    ep.add_entry_point(EntityEntryPoint::new(
        "check_kyc",
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Bool,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "grant_access",
        vec![
            Parameter::new("user", CLType::ByteArray(32)),
            Parameter::new("proof_id", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "revoke_access",
        vec![Parameter::new("user", CLType::ByteArray(32))],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "is_whitelisted",
        vec![Parameter::new("user", CLType::ByteArray(32))],
        CLType::Bool,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    let (ch, _) = storage::new_contract(
        ep,
        Some(nk),
        Some("defi_mock_pkg".into()),
        Some("defi_mock_access".into()),
        None,
    );
    runtime::put_key("defi_mock", ch.into());
}
