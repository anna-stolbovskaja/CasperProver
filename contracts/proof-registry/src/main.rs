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
    ApiError, CLType, CLValue, EntryPoint, EntryPointAccess, EntryPointType, EntryPoints, Key,
    Parameter, URef,
};

const ERR_NOT_FOUND: u16 = 1;
const ERR_UNAUTHORIZED: u16 = 2;
const ERR_ALREADY_REVOKED: u16 = 3;
const ERR_RATE_LIMIT: u16 = 4;
const ERR_INVALID_PROOF: u16 = 5;
const ERR_AGENT_EXISTS: u16 = 6;
const ERR_AGENT_NOT_FOUND: u16 = 7;

const PROOFS_DICT: &str = "proofs";
const AGENTS_DICT: &str = "agents";
const PROOF_COUNTER: &str = "pctr";

fn emit(typ: &str, id: &str) {
    let mut ev = BTreeMap::new();
    ev.insert("t".into(), typ.to_string());
    ev.insert("id".into(), id.to_string());
    let ts: u64 = runtime::get_blocktime().into();
    ev.insert("ts".into(), ts.to_string());
    let cl = CLValue::from_t(ev).unwrap_or_revert();
    runtime::put_key(
        &alloc::format!("e_{}_{}", typ, ts),
        Key::from(storage::new_uref(cl)),
    );
}

fn dict(name: &str) -> URef {
    runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

fn next_id() -> u64 {
    let u = runtime::get_key(PROOF_COUNTER)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert();
    let c: u64 = storage::read(u).unwrap_or_revert().unwrap_or(0);
    let n = c + 1;
    storage::write(u, n);
    n
}

#[no_mangle]
pub extern "C" fn submit_proof() {
    let caller = runtime::get_caller();
    let ph: String = runtime::get_named_arg("proof_hash");
    let ih: String = runtime::get_named_arg("input_hash");
    let oh: String = runtime::get_named_arg("output_hash");
    let mh: String = runtime::get_named_arg("model_hash");
    let uc: String = runtime::get_named_arg("use_case");

    let agents = dict(AGENTS_DICT);
    let _agent: (String, String, String, u64, u64, u64, u8, u64) =
        storage::dictionary_get(agents, &caller.to_string())
            .unwrap_or_revert()
            .unwrap_or_revert_with(ApiError::User(ERR_AGENT_NOT_FOUND));

    let pid = alloc::format!("P-{}", next_id());
    let ts: u64 = runtime::get_blocktime().into();

    let rec = (
        pid.clone(),
        caller.to_string(),
        ph,
        ih,
        oh,
        mh,
        ts,
        true,
        false,
        String::new(),
        uc,
    );

    let proofs = dict(PROOFS_DICT);
    storage::dictionary_put(proofs, &pid, CLValue::from_t(rec).unwrap_or_revert());

    let (aid, owner, amh, total, verified, failed, score, reg) = _agent;
    let updated = (aid, owner, amh, total + 1, verified, failed, score, reg);
    storage::dictionary_put(
        agents,
        &caller.to_string(),
        CLValue::from_t(updated).unwrap_or_revert(),
    );

    emit("ProofSubmitted", &pid);
    runtime::ret(CLValue::from_t(pid).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn get_proof() {
    let pid: String = runtime::get_named_arg("proof_id");
    let proofs = dict(PROOFS_DICT);
    let rec: (String, String, String, String, String, String, u64, bool, bool, String, String) =
        storage::dictionary_get(proofs, &pid)
            .unwrap_or_revert()
            .unwrap_or_revert_with(ApiError::User(ERR_NOT_FOUND));
    runtime::ret(CLValue::from_t(rec).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn revoke_proof() {
    let pid: String = runtime::get_named_arg("proof_id");
    let reason: String = runtime::get_named_arg("reason");
    let caller = runtime::get_caller();

    let proofs = dict(PROOFS_DICT);
    let rec: (String, String, String, String, String, String, u64, bool, bool, String, String) =
        storage::dictionary_get(proofs, &pid)
            .unwrap_or_revert()
            .unwrap_or_revert_with(ApiError::User(ERR_NOT_FOUND));

    let (id, agent, ph, ih, oh, mh, ts, _valid, revoked, _reason, uc) = rec;

    if caller.to_string() != agent {
        runtime::revert(ApiError::User(ERR_UNAUTHORIZED));
    }
    if revoked {
        runtime::revert(ApiError::User(ERR_ALREADY_REVOKED));
    }

    let updated = (id, agent, ph, ih, oh, mh, ts, false, true, reason, uc);
    storage::dictionary_put(proofs, &pid, CLValue::from_t(updated).unwrap_or_revert());

    emit("ProofRevoked", &pid);
}

#[no_mangle]
pub extern "C" fn register_agent() {
    let caller = runtime::get_caller();
    let agent_id: String = runtime::get_named_arg("agent_id");
    let model_hash: String = runtime::get_named_arg("model_hash");
    let uc: String = runtime::get_named_arg("use_case");

    let agents = dict(AGENTS_DICT);
    let existing: Option<(String, String, String, u64, u64, u64, u8, u64)> =
        storage::dictionary_get(agents, &caller.to_string()).unwrap_or_revert();
    if existing.is_some() {
        runtime::revert(ApiError::User(ERR_AGENT_EXISTS));
    }

    let ts: u64 = runtime::get_blocktime().into();
    let rec = (agent_id, caller.to_string(), model_hash, 0u64, 0u64, 0u64, 50u8, ts);
    storage::dictionary_put(
        agents,
        &caller.to_string(),
        CLValue::from_t(rec).unwrap_or_revert(),
    );

    emit("AgentRegistered", &caller.to_string());
}

#[no_mangle]
pub extern "C" fn get_reputation() {
    let agent: String = runtime::get_named_arg("agent");
    let agents = dict(AGENTS_DICT);
    let rec: (String, String, String, u64, u64, u64, u8, u64) =
        storage::dictionary_get(agents, &agent)
            .unwrap_or_revert()
            .unwrap_or_revert_with(ApiError::User(ERR_AGENT_NOT_FOUND));
    runtime::ret(CLValue::from_t(rec).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn call() {
    let proofs = storage::new_dictionary(PROOFS_DICT).unwrap_or_revert();
    let agents = storage::new_dictionary(AGENTS_DICT).unwrap_or_revert();
    let ctr = storage::new_uref(0u64);

    let mut nk = NamedKeys::new();
    nk.insert(PROOFS_DICT.into(), proofs.into());
    nk.insert(AGENTS_DICT.into(), agents.into());
    nk.insert(PROOF_COUNTER.into(), ctr.into());

    let mut ep = EntryPoints::new();
    ep.add_entry_point(EntryPoint::new(
        "submit_proof",
        vec![
            Parameter::new("proof_hash", CLType::String),
            Parameter::new("input_hash", CLType::String),
            Parameter::new("output_hash", CLType::String),
            Parameter::new("model_hash", CLType::String),
            Parameter::new("use_case", CLType::String),
        ],
        CLType::String,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));
    ep.add_entry_point(EntryPoint::new(
        "get_proof",
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));
    ep.add_entry_point(EntryPoint::new(
        "revoke_proof",
        vec![
            Parameter::new("proof_id", CLType::String),
            Parameter::new("reason", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));
    ep.add_entry_point(EntryPoint::new(
        "register_agent",
        vec![
            Parameter::new("agent_id", CLType::String),
            Parameter::new("model_hash", CLType::String),
            Parameter::new("use_case", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));
    ep.add_entry_point(EntryPoint::new(
        "get_reputation",
        vec![Parameter::new("agent", CLType::String)],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Contract,
    ));

    let (ch, _) = storage::new_contract(
        ep,
        Some(nk),
        Some("proof_registry_pkg".into()),
        Some("proof_registry_access".into()),
    );
    runtime::put_key("proof_registry", ch.into());
}
