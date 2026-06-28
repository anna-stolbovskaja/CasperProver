#![no_std]
#![no_main]

extern crate alloc;

use alloc::collections::BTreeMap;
use alloc::string::{String, ToString};
use alloc::vec;
use alloc::vec::Vec;

use casper_contract::contract_api::{runtime, storage};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;
use casper_types::account::AccountHash;
use casper_types::contracts::NamedKeys;
use casper_types::{
    ApiError, CLType, CLValue, EntryPoint, EntryPointAccess, EntryPointType, EntryPoints,
    Key, Parameter, URef,
};

const ERR_NOT_FOUND: u16 = 1;
const ERR_RATE_LIMIT: u16 = 4;
const ERR_INVALID_PROOF: u16 = 5;

const REGISTRY_HASH: &str = "registry_hash";
const VERIFY_COUNTS: &str = "verify_counts";
const MAX_VERIFY_PER_BLOCK: u64 = 100;

fn dict(name: &str) -> URef {
    runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

fn check_rate_limit(caller: &AccountHash) {
    let d = dict(VERIFY_COUNTS);
    let ts: u64 = runtime::get_blocktime().into();
    let key = alloc::format!("{}_{}", caller, ts);
    let count: u64 = storage::dictionary_get(d, &key)
        .unwrap_or_revert()
        .unwrap_or(0);
    if count >= MAX_VERIFY_PER_BLOCK {
        runtime::revert(ApiError::User(ERR_RATE_LIMIT));
    }
    storage::dictionary_put(d, &key, CLValue::from_t(count + 1).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn verify() {
    let pid: String = runtime::get_named_arg("proof_id");
    let caller = runtime::get_caller();
    check_rate_limit(&caller);

    let reg = runtime::get_key(REGISTRY_HASH).unwrap_or_revert();
    let args = runtime::RuntimeArgs::new();
    let mut a = args;
    a.insert("proof_id", pid.clone()).unwrap_or_revert();
    let rec: (String, String, String, String, String, String, u64, bool, bool, String, String) =
        runtime::call_contract(reg.into_hash().unwrap_or_revert().into(), "get_proof", a);

    let (_id, _agent, _ph, _ih, _oh, _mh, _ts, valid, revoked, _reason, _uc) = rec;
    let result = valid && !revoked;

    let mut ev = BTreeMap::new();
    let t = if result { "VerificationPassed" } else { "VerificationFailed" };
    ev.insert("t".into(), t.to_string());
    ev.insert("id".into(), pid);
    let cl = CLValue::from_t(ev).unwrap_or_revert();
    runtime::put_key(
        &alloc::format!("e_{}", runtime::get_blocktime().into()),
        Key::from(storage::new_uref(cl)),
    );

    runtime::ret(CLValue::from_t(result).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn is_valid() {
    let pid: String = runtime::get_named_arg("proof_id");

    let reg = runtime::get_key(REGISTRY_HASH).unwrap_or_revert();
    let mut args = runtime::RuntimeArgs::new();
    args.insert("proof_id", pid).unwrap_or_revert();
    let rec: (String, String, String, String, String, String, u64, bool, bool, String, String) =
        runtime::call_contract(reg.into_hash().unwrap_or_revert().into(), "get_proof", args);

    let (_, _, _, _, _, _, _, valid, revoked, _, _) = rec;
    runtime::ret(CLValue::from_t(valid && !revoked).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn batch_check() {
    let pids: Vec<String> = runtime::get_named_arg("proof_ids");
    let caller = runtime::get_caller();
    check_rate_limit(&caller);

    let reg = runtime::get_key(REGISTRY_HASH).unwrap_or_revert();
    let mut results: Vec<(String, bool)> = Vec::with_capacity(pids.len());

    for pid in pids {
        let mut args = runtime::RuntimeArgs::new();
        args.insert("proof_id", pid.clone()).unwrap_or_revert();
        let rec: (String, String, String, String, String, String, u64, bool, bool, String, String) =
            runtime::call_contract(
                reg.into_hash().unwrap_or_revert().into(),
                "get_proof",
                args,
            );
        let (_, _, _, _, _, _, _, valid, revoked, _, _) = rec;
        results.push((pid, valid && !revoked));
    }

    runtime::ret(CLValue::from_t(results).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn call() {
    let registry: Key = runtime::get_named_arg("registry_contract");

    let vc = storage::new_dictionary(VERIFY_COUNTS).unwrap_or_revert();

    let mut nk = NamedKeys::new();
    nk.insert(REGISTRY_HASH.into(), registry);
    nk.insert(VERIFY_COUNTS.into(), vc.into());

    let mut ep = EntryPoints::new();
    ep.add_entry_point(EntryPoint::new(
        "verify",
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Bool,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));
    ep.add_entry_point(EntryPoint::new(
        "is_valid",
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Bool,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));
    ep.add_entry_point(EntryPoint::new(
        "batch_check",
        vec![Parameter::new("proof_ids", CLType::List(Box::new(CLType::String)))],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));

    let (ch, _) = storage::new_contract(
        ep,
        Some(nk),
        Some("verifier_gate_pkg".into()),
        Some("verifier_gate_access".into()),
    );
    runtime::put_key("verifier_gate", ch.into());
}
