#![no_std]
#![no_main]

extern crate alloc;

use alloc::string::{String, ToString};
use alloc::vec;

use casper_contract::contract_api::{runtime, storage};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;
use casper_types::contracts::NamedKeys;
use casper_types::{EntryPointPayment, 
    ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointType, EntryPoints,
    Parameter, URef,
};

const ERR_NOT_FOUND: u16 = 1;
const ERR_UNAUTHORIZED: u16 = 2;
const ERR_ALREADY_REVOKED: u16 = 3;
const ERR_AGENT_EXISTS: u16 = 6;
const ERR_AGENT_NOT_FOUND: u16 = 7;
const ERR_INPUT_TOO_LONG: u16 = 8;

const PROOFS_DICT: &str = "proofs";
const MAX_HASH_LEN: usize = 128;
const MAX_AGENT_ID_LEN: usize = 64;
const AGENTS_DICT: &str = "agents";
const PROOF_COUNTER: &str = "pctr";

// Proof: ((proof_id, agent, proof_hash), (input_hash, output_hash, model_hash), (timestamp, valid, revoked))
// valid/revoked: 1=true 0=false
type ProofRec = ((String, String, String), (String, String, String), (u64, u64, u64));

// Agent: ((agent_id, owner, model_hash), (total, verified, failed), (score, registered_at))
type AgentRec = ((String, String, String), (u64, u64, u64), (u64, u64));

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

fn read_agent(agents: URef, key: &str) -> AgentRec {
    storage::dictionary_get::<AgentRec>(agents, key)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_AGENT_NOT_FOUND))
}

fn read_proof(proofs: URef, key: &str) -> ProofRec {
    storage::dictionary_get::<ProofRec>(proofs, key)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_NOT_FOUND))
}

fn validate_hash(h: &str) {
    if h.is_empty() || h.len() > MAX_HASH_LEN {
        runtime::revert(ApiError::User(ERR_INPUT_TOO_LONG));
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn submit_proof() {
    let caller = runtime::get_caller();
    let ph: String = runtime::get_named_arg("proof_hash");
    let ih: String = runtime::get_named_arg("input_hash");
    let oh: String = runtime::get_named_arg("output_hash");
    let mh: String = runtime::get_named_arg("model_hash");

    validate_hash(&ph);
    validate_hash(&ih);
    validate_hash(&oh);
    validate_hash(&mh);

    let agents = dict(AGENTS_DICT);
    let ((aid, owner, amh), (total, verified, failed), (score, reg)) =
        read_agent(agents, &caller.to_string());

    let pid = alloc::format!("P-{}", next_id());
    let ts: u64 = runtime::get_blocktime().into();

    let rec: ProofRec = (
        (pid.clone(), caller.to_string(), ph),
        (ih, oh, mh),
        (ts, 1, 0),
    );
    let proofs = dict(PROOFS_DICT);
    storage::dictionary_put(proofs, &pid, rec);

    let updated: AgentRec = ((aid, owner, amh), (total + 1, verified, failed), (score, reg));
    storage::dictionary_put(agents, &caller.to_string(), updated);

    runtime::ret(CLValue::from_t(pid).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn get_proof() {
    let pid: String = runtime::get_named_arg("proof_id");
    validate_hash(&pid);
    let proofs = dict(PROOFS_DICT);
    let rec = read_proof(proofs, &pid);
    runtime::ret(CLValue::from_t(rec).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn revoke_proof() {
    let pid: String = runtime::get_named_arg("proof_id");
    validate_hash(&pid);
    let caller = runtime::get_caller();

    let proofs = dict(PROOFS_DICT);
    let ((id, agent, ph), (ih, oh, mh), (ts, _valid, revoked)) = read_proof(proofs, &pid);

    if caller.to_string() != agent {
        runtime::revert(ApiError::User(ERR_UNAUTHORIZED));
    }
    if revoked == 1 {
        runtime::revert(ApiError::User(ERR_ALREADY_REVOKED));
    }

    let updated: ProofRec = ((id, agent.clone(), ph), (ih, oh, mh), (ts, 0, 1));
    storage::dictionary_put(proofs, &pid, updated);

    // Update agent stats: increment failed count
    let agents = dict(AGENTS_DICT);
    let agent_rec: Option<AgentRec> =
        storage::dictionary_get(agents, &agent).unwrap_or_revert();
    if let Some(((aid, owner, amh), (total, verified, failed), (score, reg))) = agent_rec {
        let new_score = if score > 5 { score - 5 } else { 0 };
        let updated_agent: AgentRec =
            ((aid, owner, amh), (total, verified, failed + 1), (new_score, reg));
        storage::dictionary_put(agents, &agent, updated_agent);
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn register_agent() {
    let caller = runtime::get_caller();
    let agent_id: String = runtime::get_named_arg("agent_id");
    let model_hash: String = runtime::get_named_arg("model_hash");

    if agent_id.is_empty() || agent_id.len() > MAX_AGENT_ID_LEN {
        runtime::revert(ApiError::User(ERR_INPUT_TOO_LONG));
    }
    validate_hash(&model_hash);

    let agents = dict(AGENTS_DICT);
    let existing: Option<AgentRec> =
        storage::dictionary_get(agents, &caller.to_string()).unwrap_or_revert();
    if existing.is_some() {
        runtime::revert(ApiError::User(ERR_AGENT_EXISTS));
    }

    let ts: u64 = runtime::get_blocktime().into();
    let rec: AgentRec = (
        (agent_id, caller.to_string(), model_hash),
        (0, 0, 0),
        (50, ts),
    );
    storage::dictionary_put(agents, &caller.to_string(), rec);
}

#[unsafe(no_mangle)]
pub extern "C" fn get_reputation() {
    let agent: String = runtime::get_named_arg("agent");
    let agents = dict(AGENTS_DICT);
    let rec = read_agent(agents, &agent);
    runtime::ret(CLValue::from_t(rec).unwrap_or_revert());
}

#[unsafe(no_mangle)]
pub extern "C" fn call() {
    let proofs = storage::new_dictionary(PROOFS_DICT).unwrap_or_revert();
    let agents = storage::new_dictionary(AGENTS_DICT).unwrap_or_revert();
    let ctr = storage::new_uref(0u64);

    let mut nk = NamedKeys::new();
    nk.insert(PROOFS_DICT.into(), proofs.into());
    nk.insert(AGENTS_DICT.into(), agents.into());
    nk.insert(PROOF_COUNTER.into(), ctr.into());

    let mut ep = EntryPoints::new();
    ep.add_entry_point(EntityEntryPoint::new(
        "submit_proof",
        vec![
            Parameter::new("proof_hash", CLType::String),
            Parameter::new("input_hash", CLType::String),
            Parameter::new("output_hash", CLType::String),
            Parameter::new("model_hash", CLType::String),
        ],
        CLType::String,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "get_proof",
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "revoke_proof",
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "register_agent",
        vec![
            Parameter::new("agent_id", CLType::String),
            Parameter::new("model_hash", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));
    ep.add_entry_point(EntityEntryPoint::new(
        "get_reputation",
        vec![Parameter::new("agent", CLType::String)],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    let (ch, _) = storage::new_contract(
        ep,
        Some(nk),
        Some("proof_registry_pkg".into()),
        Some("proof_registry_access".into()),
        None,
    );
    runtime::put_key("proof_registry", ch.into());
}
