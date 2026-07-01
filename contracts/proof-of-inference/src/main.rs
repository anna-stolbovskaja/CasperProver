#![no_std]
#![no_main]

extern crate alloc;

use alloc::string::String;
use alloc::vec::Vec;
use alloc::vec;

use casper_contract::contract_api::{runtime, storage, system};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;

use casper_types::{
    ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointType, EntryPoints,
    Parameter, URef, U512, Key,
};
use casper_types::account::AccountHash;
use casper_types::contracts::NamedKeys;
use casper_types::EntryPointPayment;

const PROOF_DICTIONARY: &str = "proof_dict";
const VERIFIER_DICTIONARY: &str = "verifier_dict";
const PROOF_COUNTER_KEY: &str = "proof_counter";
const INSTALLER_KEY: &str = "installer";

const ERR_NOT_INSTALLER: u16 = 1;
const ERR_PROOF_NOT_FOUND: u16 = 2;
const ERR_VERIFIER_NOT_FOUND: u16 = 3;
const ERR_VERIFIER_EXISTS: u16 = 4;
const ERR_INVALID_STATUS: u16 = 5;
const ERR_NOT_VERIFIER: u16 = 6;
const ERR_ALREADY_VERIFIED: u16 = 7;
const ERR_NOT_CHALLENGED: u16 = 8;

fn get_installer() -> AccountHash {
    let key: Key = runtime::get_key(INSTALLER_KEY).unwrap_or_revert();
    key.into_account().unwrap_or_revert()
}

fn assert_installer() {
    if runtime::get_caller() != get_installer() {
        runtime::revert(ApiError::User(ERR_NOT_INSTALLER));
    }
}

fn get_dict_uref(name: &str) -> URef {
    runtime::get_key(name).unwrap_or_revert().into_uref().unwrap_or_revert()
}

fn proof_key(proof_id: &str) -> String {
    alloc::format!("proof:{}", proof_id)
}

fn verifier_key(verifier_id: &str) -> String {
    alloc::format!("verifier:{}", verifier_id)
}

#[no_mangle]
pub extern "C" fn register_proof() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    let input_hash: String = runtime::get_named_arg("input_hash");
    let output_hash: String = runtime::get_named_arg("output_hash");
    let proof_hash: String = runtime::get_named_arg("proof_hash");
    let agent_id: String = runtime::get_named_arg("agent_id");

    let counter_key = runtime::get_key(PROOF_COUNTER_KEY).unwrap_or_revert();
    let counter_uref = counter_key.into_uref().unwrap_or_revert();

    let proof_id_num: u64 = storage::read(counter_uref).unwrap_or(None).unwrap_or(0u64);
    let proof_id = proof_id_num.to_string();

    storage::write(counter_uref, proof_id_num + 1);

    let timestamp: u64 = runtime::get_blocktime().into();

    let proof_record = (
        (proof_id.clone(), agent_id, model_hash),
        (input_hash, output_hash, proof_hash),
        (timestamp, 0u64, String::new()),
    );

    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    storage::dictionary_put(dict_uref, &proof_key(&proof_id), proof_record);
}

#[no_mangle]
pub extern "C" fn verify_proof() {
    let proof_id: String = runtime::get_named_arg("proof_id");

    let caller = runtime::get_caller();
    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    let proof_key_str = proof_key(&proof_id);

    let mut record: (
        (String, String, String),
        (String, String, String),
        (u64, u64, String),
    ) = storage::dictionary_get(dict_uref, &proof_key_str)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROOF_NOT_FOUND));

    if record.2.1 != 0 {
        runtime::revert(ApiError::User(ERR_ALREADY_VERIFIED));
    }

    record.2.1 = 1;

    storage::dictionary_put(dict_uref, &proof_key_str, record);

    let verifier_dict = get_dict_uref(VERIFIER_DICTIONARY);
    let verifier_id = caller.to_string();
    let v_key = verifier_key(&verifier_id);

    let mut v_record: (
        (String, String, u64),
        (u64, u64, u64),
    ) = storage::dictionary_get(verifier_dict, &v_key)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_NOT_VERIFIER));

    v_record.1.0 += 1;

    storage::dictionary_put(verifier_dict, &v_key, v_record);
}

#[no_mangle]
pub extern "C" fn challenge_proof() {
    let proof_id: String = runtime::get_named_arg("proof_id");
    let evidence: String = runtime::get_named_arg("evidence");

    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    let proof_key_str = proof_key(&proof_id);

    let mut record: (
        (String, String, String),
        (String, String, String),
        (u64, u64, String),
    ) = storage::dictionary_get(dict_uref, &proof_key_str)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROOF_NOT_FOUND));

    if record.2.1 != 0 && record.2.1 != 1 {
        runtime::revert(ApiError::User(ERR_INVALID_STATUS));
    }

    record.2.1 = 2;
    record.2.2 = evidence;

    storage::dictionary_put(dict_uref, &proof_key_str, record);
}

#[no_mangle]
pub extern "C" fn resolve_challenge() {
    assert_installer();

    let proof_id: String = runtime::get_named_arg("proof_id");
    let valid: u64 = runtime::get_named_arg("valid");

    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    let proof_key_str = proof_key(&proof_id);

    let mut record: (
        (String, String, String),
        (String, String, String),
        (u64, u64, String),
    ) = storage::dictionary_get(dict_uref, &proof_key_str)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROOF_NOT_FOUND));

    if record.2.1 != 2 {
        runtime::revert(ApiError::User(ERR_NOT_CHALLENGED));
    }

    if valid == 0 {
        record.2.1 = 3;
    } else {
        record.2.1 = 4;
    }

    storage::dictionary_put(dict_uref, &proof_key_str, record);
}

#[no_mangle]
pub extern "C" fn register_verifier() {
    assert_installer();

    let verifier_id: String = runtime::get_named_arg("verifier_id");
    let pub_key: String = runtime::get_named_arg("pub_key");

    let dict_uref = get_dict_uref(VERIFIER_DICTIONARY);
    let v_key = verifier_key(&verifier_id);

    let existing: Option<(
        (String, String, u64),
        (u64, u64, u64),
    )> = storage::dictionary_get(dict_uref, &v_key).unwrap_or_revert();

    if existing.is_some() {
        runtime::revert(ApiError::User(ERR_VERIFIER_EXISTS));
    }

    let timestamp: u64 = runtime::get_blocktime().into();

    let record = (
        (verifier_id, pub_key, timestamp),
        (0u64, 1u64, 0u64),
    );

    storage::dictionary_put(dict_uref, &v_key, record);
}

#[no_mangle]
pub extern "C" fn revoke_verifier() {
    assert_installer();

    let verifier_id: String = runtime::get_named_arg("verifier_id");

    let dict_uref = get_dict_uref(VERIFIER_DICTIONARY);
    let v_key = verifier_key(&verifier_id);

    let mut record: (
        (String, String, u64),
        (u64, u64, u64),
    ) = storage::dictionary_get(dict_uref, &v_key)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_VERIFIER_NOT_FOUND));

    record.1.1 = 0;
    record.1.2 = runtime::get_blocktime().into();

    storage::dictionary_put(dict_uref, &v_key, record);
}

#[no_mangle]
pub extern "C" fn get_proof() {
    let proof_id: String = runtime::get_named_arg("proof_id");

    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    let proof_key_str = proof_key(&proof_id);

    let record: (
        (String, String, String),
        (String, String, String),
        (u64, u64, String),
    ) = storage::dictionary_get(dict_uref, &proof_key_str)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROOF_NOT_FOUND));

    runtime::ret(CLValue::from_t(record).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn get_stats() {
    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    let counter_key = runtime::get_key(PROOF_COUNTER_KEY).unwrap_or_revert();
    let counter_uref = counter_key.into_uref().unwrap_or_revert();
    let total_proofs: u64 = storage::read(counter_uref).unwrap_or(None).unwrap_or(0);

    let mut verified: u64 = 0;
    let mut challenged: u64 = 0;
    let mut rejected: u64 = 0;
    let mut resolved: u64 = 0;

    for i in 0..total_proofs {
        let proof_id = i.to_string();
        let key_str = proof_key(&proof_id);

        let record: Option<(
            (String, String, String),
            (String, String, String),
            (u64, u64, String),
        )> = storage::dictionary_get(dict_uref, &key_str).unwrap_or(None);

        if let Some(r) = record {
            match r.2.1 {
                1 => verified += 1,
                2 => challenged += 1,
                3 => rejected += 1,
                4 => resolved += 1,
                _ => {}
            }
        }
    }

    let stats = (
        total_proofs,
        verified,
        challenged,
        rejected,
        resolved,
    );

    runtime::ret(CLValue::from_t(stats).unwrap_or_revert());
}

fn create_entry_points() -> EntryPoints {
    let mut entry_points = EntryPoints::new();

    entry_points.add_entry_point(EntityEntryPoint::new(
        "register_proof".to_string(),
        vec![
            Parameter::new("model_hash", CLType::String),
            Parameter::new("input_hash", CLType::String),
            Parameter::new("output_hash", CLType::String),
            Parameter::new("proof_hash", CLType::String),
            Parameter::new("agent_id", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "verify_proof".to_string(),
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "challenge_proof".to_string(),
        vec![
            Parameter::new("proof_id", CLType::String),
            Parameter::new("evidence", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "resolve_challenge".to_string(),
        vec![
            Parameter::new("proof_id", CLType::String),
            Parameter::new("valid", CLType::U64),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "register_verifier".to_string(),
        vec![
            Parameter::new("verifier_id", CLType::String),
            Parameter::new("pub_key", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "revoke_verifier".to_string(),
        vec![Parameter::new("verifier_id", CLType::String)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "get_proof".to_string(),
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "get_stats".to_string(),
        vec![],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points
}

#[no_mangle]
pub extern "C" fn call() {
    let mut named_keys = NamedKeys::new();

    let proof_dict = storage::new_dictionary(PROOF_DICTIONARY).unwrap_or_revert();
    let verifier_dict = storage::new_dictionary(VERIFIER_DICTIONARY).unwrap_or_revert();

    named_keys.insert(PROOF_DICTIONARY.to_string(), proof_dict.into());
    named_keys.insert(VERIFIER_DICTIONARY.to_string(), verifier_dict.into());

    let counter_uref = storage::new_uref(0u64);
    named_keys.insert(PROOF_COUNTER_KEY.to_string(), counter_uref.into());

    let installer = runtime::get_caller();
    named_keys.insert(INSTALLER_KEY.to_string(), installer.into());

    let entry_points = create_entry_points();

    let (contract_hash, _version) = storage::new_contract(entry_points, Some(named_keys), None, None);

    runtime::put_key("ProofOfInference", contract_hash.into());
}