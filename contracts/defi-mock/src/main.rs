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
    ApiError, CLType, CLValue, EntryPoint, EntryPointAccess, EntryPointType, EntryPoints,
    Key, Parameter, URef,
};

const ERR_KYC_FAILED: u16 = 10;
const ERR_ALREADY_WHITELISTED: u16 = 11;

const VERIFIER_HASH: &str = "verifier_hash";
const WHITELIST_DICT: &str = "whitelist";

fn dict(name: &str) -> URef {
    runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

#[no_mangle]
pub extern "C" fn check_kyc() {
    let pid: String = runtime::get_named_arg("proof_id");

    let vg = runtime::get_key(VERIFIER_HASH).unwrap_or_revert();
    let mut args = runtime::RuntimeArgs::new();
    args.insert("proof_id", pid).unwrap_or_revert();

    let valid: bool = runtime::call_contract(
        vg.into_hash().unwrap_or_revert().into(),
        "is_valid",
        args,
    );

    runtime::ret(CLValue::from_t(valid).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn grant_access() {
    let user: AccountHash = runtime::get_named_arg("user");
    let pid: String = runtime::get_named_arg("proof_id");

    let vg = runtime::get_key(VERIFIER_HASH).unwrap_or_revert();
    let mut args = runtime::RuntimeArgs::new();
    args.insert("proof_id", pid.clone()).unwrap_or_revert();

    let valid: bool = runtime::call_contract(
        vg.into_hash().unwrap_or_revert().into(),
        "is_valid",
        args,
    );
    if !valid {
        runtime::revert(ApiError::User(ERR_KYC_FAILED));
    }

    let wl = dict(WHITELIST_DICT);
    let ts: u64 = runtime::get_blocktime().into();
    let entry = (user.to_string(), pid, ts);
    storage::dictionary_put(wl, &user.to_string(), CLValue::from_t(entry).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn is_whitelisted() {
    let user: String = runtime::get_named_arg("user");
    let wl = dict(WHITELIST_DICT);
    let entry: Option<(String, String, u64)> =
        storage::dictionary_get(wl, &user).unwrap_or_revert();
    runtime::ret(CLValue::from_t(entry.is_some()).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn call() {
    let verifier: Key = runtime::get_named_arg("verifier_contract");

    let wl = storage::new_dictionary(WHITELIST_DICT).unwrap_or_revert();

    let mut nk = NamedKeys::new();
    nk.insert(VERIFIER_HASH.into(), verifier);
    nk.insert(WHITELIST_DICT.into(), wl.into());

    let mut ep = EntryPoints::new();
    ep.add_entry_point(EntryPoint::new(
        "check_kyc",
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Bool,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));
    ep.add_entry_point(EntryPoint::new(
        "grant_access",
        vec![
            Parameter::new("user", CLType::ByteArray(32)),
            Parameter::new("proof_id", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));
    ep.add_entry_point(EntryPoint::new(
        "is_whitelisted",
        vec![Parameter::new("user", CLType::String)],
        CLType::Bool,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));

    let (ch, _) = storage::new_contract(
        ep,
        Some(nk),
        Some("defi_mock_pkg".into()),
        Some("defi_mock_access".into()),
    );
    runtime::put_key("defi_mock", ch.into());
}
