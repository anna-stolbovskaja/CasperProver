#![no_std]
#![no_main]

extern crate alloc;

use alloc::string::{String, ToString};
use alloc::vec;
use alloc::vec::Vec;

use casper_contract::contract_api::{runtime, storage};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;
use casper_types::account::AccountHash;
use casper_types::contracts::NamedKeys;
use casper_types::{EntryPointPayment, 
    ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointType, EntryPoints, Key,
    Parameter, URef,
};

const ERR_RATE_LIMIT: u16 = 4;
const ERR_BATCH_TOO_LARGE: u16 = 5;

const REGISTRY_HASH: &str = "registry_hash";
const VERIFY_COUNTS: &str = "verify_counts";
const MAX_VERIFY_PER_BLOCK: u64 = 100;
const MAX_BATCH_SIZE: usize = 50;

// Same type as proof-registry ProofRec
type ProofRec = ((String, String, String), (String, String, String), (u64, u64, u64));

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
    storage::dictionary_put(d, &key, count + 1);
}

fn fetch_proof(pid: &str) -> ProofRec {
    let reg = runtime::get_key(REGISTRY_HASH).unwrap_or_revert();
    let mut args = casper_types::RuntimeArgs::new();
    args.insert("proof_id", pid.to_string()).unwrap_or_revert();
    runtime::call_contract(reg.into_entity_hash_addr().unwrap_or_revert().into(), "get_proof", args)
}

#[unsafe(no_mangle)]
pub extern "C" fn verify() {
    let pid: String = runtime::get_named_arg("proof_id");
    let caller = runtime::get_caller();
    check_rate_limit(&caller);

    let (_, _, (_, valid, revoked)) = fetch_proof(&pid);
    let result = valid == 1 && revoked == 0;

    runtime::ret(CLValue::from_t(result).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn is_valid() {
    let pid: String = runtime::get_named_arg("proof_id");
    let caller = runtime::get_caller();
    check_rate_limit(&caller);
    let (_, _, (_, valid, revoked)) = fetch_proof(&pid);
    runtime::ret(CLValue::from_t(valid == 1 && revoked == 0).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn batch_check() {
    let pids: Vec<String> = runtime::get_named_arg("proof_ids");
    if pids.len() > MAX_BATCH_SIZE {
        runtime::revert(ApiError::User(ERR_BATCH_TOO_LARGE));
    }
    let caller = runtime::get_caller();
    check_rate_limit(&caller);

    let mut results: Vec<(String, u64)> = Vec::with_capacity(pids.len());
    for pid in pids {
        let (_, _, (_, valid, revoked)) = fetch_proof(&pid);
        let ok = if valid == 1 && revoked == 0 { 1u64 } else { 0u64 };
        results.push((pid, ok));
    }
    runtime::ret(CLValue::from_t(results).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn call() {
    let registry: Key = runtime::get_named_arg("registry_contract");

    let vc = storage::new_dictionary(VERIFY_COUNTS).unwrap_or_revert();

    let mut nk = NamedKeys::new();
    nk.insert(REGISTRY_HASH.into(), registry);
    nk.insert(VERIFY_COUNTS.into(), vc.into());

    let mut ep = EntryPoints::new();
    ep.add_entry_point(EntityEntryPoint::new(
        "verify",
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Bool,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "is_valid",
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Bool,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "batch_check",
        vec![Parameter::new(
            "proof_ids",
            CLType::List(alloc::boxed::Box::new(CLType::String)),
        )],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    let (ch, _) = storage::new_contract(
        ep,
        Some(nk),
        Some("verifier_gate_pkg".into()),
        Some("verifier_gate_access".into()),
        None,
    );
    runtime::put_key("verifier_gate", ch.into());
}
