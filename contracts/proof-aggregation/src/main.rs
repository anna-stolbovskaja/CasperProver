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

const CONTRACT_NAME: &str = "ProofAggregationRegistry";
const BATCHES_DICT: &str = "batches";
const PROOFS_DICT: &str = "proofs";
const BATCH_COUNT_KEY: &str = "batch_count";
const TOTAL_PROOFS_KEY: &str = "total_proofs";
const FINALIZED_BATCHES_KEY: &str = "finalized_batches";

const ERR_BATCH_EXISTS: u16 = 1;
const ERR_BATCH_NOT_FOUND: u16 = 2;
const ERR_BATCH_FINALIZED: u16 = 3;
const ERR_BATCH_NOT_FINALIZED: u16 = 4;
const ERR_NOT_CREATOR: u16 = 5;
const ERR_BATCH_FULL: u16 = 6;
const ERR_PROOF_NOT_FOUND: u16 = 7;
const ERR_INVALID_PROOF: u16 = 8;

type BatchRecord = (
    (String, String, String),
    (u64, u64, u64),
    (u64, u64, u64),
);

fn get_batches_uref() -> URef {
    let key: Key = runtime::get_key(BATCHES_DICT).unwrap_or_revert();
    key.into_uref().unwrap_or_revert()
}

fn get_proofs_uref() -> URef {
    let key: Key = runtime::get_key(PROOFS_DICT).unwrap_or_revert();
    key.into_uref().unwrap_or_revert()
}

fn account_hash_to_string(account: AccountHash) -> String {
    let bytes = account.as_bytes();
    let mut hex_str = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        hex_str.push_str(&format!("{:02x}", byte));
    }
    hex_str
}

fn string_to_account_hash(s: &str) -> AccountHash {
    let mut bytes = [0u8; 32];
    let hex_bytes = s.as_bytes();
    for i in 0..32 {
        let high = hex_bytes[i * 2];
        let low = hex_bytes[i * 2 + 1];
        let high_val = if high >= b'a' { high - b'a' + 10 } else { high - b'0' };
        let low_val = if low >= b'a' { low - b'a' + 10 } else { low - b'0' };
        bytes[i] = (high_val << 4) | low_val;
    }
    AccountHash::new(bytes)
}

fn get_current_timestamp() -> u64 {
    runtime::get_blocktime().into()
}

#[no_mangle]
pub extern "C" fn create_batch() {
    let batch_id: String = runtime::get_named_arg("batch_id");
    let merkle_root: String = runtime::get_named_arg("merkle_root");
    let proof_count: u64 = runtime::get_named_arg("proof_count");

    let batches_uref = get_batches_uref();
    let existing: Option<BatchRecord> = storage::dictionary_get(batches_uref, &batch_id).unwrap_or_revert();
    if existing.is_some() {
        runtime::revert(ApiError::User(ERR_BATCH_EXISTS));
    }

    let caller = runtime::get_caller();
    let creator = account_hash_to_string(caller);
    let created_at = get_current_timestamp();

    let batch: BatchRecord = (
        (batch_id.clone(), creator, merkle_root),
        (proof_count, 0, created_at),
        (0, 0, 0),
    );

    storage::dictionary_put(batches_uref, &batch_id, batch);

    let batch_count_uref: URef = runtime::get_key(BATCH_COUNT_KEY).unwrap_or_revert().into_uref().unwrap_or_revert();
    let current_count: u64 = storage::read(batch_count_uref).unwrap_or_revert().unwrap_or(0);
    storage::write(batch_count_uref, current_count + 1);
}

#[no_mangle]
pub extern "C" fn add_proof() {
    let batch_id: String = runtime::get_named_arg("batch_id");
    let proof_hash: String = runtime::get_named_arg("proof_hash");
    let leaf_index: u64 = runtime::get_named_arg("leaf_index");

    let batches_uref = get_batches_uref();
    let batch_opt: Option<BatchRecord> = storage::dictionary_get(batches_uref, &batch_id).unwrap_or_revert();
    let batch = batch_opt.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_BATCH_NOT_FOUND)));

    let ((batch_id_val, creator, merkle_root), (proof_count, finalized, created_at), (verified_count, added_count, status)) = batch;

    if finalized != 0 {
        runtime::revert(ApiError::User(ERR_BATCH_FINALIZED));
    }

    if added_count >= 256 {
        runtime::revert(ApiError::User(ERR_BATCH_FULL));
    }

    let caller = runtime::get_caller();
    let caller_str = account_hash_to_string(caller);
    if creator != caller_str {
        runtime::revert(ApiError::User(ERR_NOT_CREATOR));
    }

    let new_added = added_count + 1;
    let proof_key = format!("{}:{}", batch_id, proof_hash);

    let proofs_uref = get_proofs_uref();
    let existing_proof: Option<String> = storage::dictionary_get(proofs_uref, &proof_key).unwrap_or_revert();
    if existing_proof.is_some() {
        runtime::revert(ApiError::User(ERR_BATCH_EXISTS));
    }

    storage::dictionary_put(proofs_uref, &proof_key, leaf_index.to_string());

    let updated_batch: BatchRecord = (
        (batch_id_val.clone(), creator, merkle_root),
        (proof_count, finalized, created_at),
        (verified_count, new_added, status),
    );

    storage::dictionary_put(batches_uref, &batch_id, updated_batch);

    let total_proofs_uref: URef = runtime::get_key(TOTAL_PROOFS_KEY).unwrap_or_revert().into_uref().unwrap_or_revert();
    let current_total: u64 = storage::read(total_proofs_uref).unwrap_or_revert().unwrap_or(0);
    storage::write(total_proofs_uref, current_total + 1);
}

#[no_mangle]
pub extern "C" fn verify_inclusion() {
    let batch_id: String = runtime::get_named_arg("batch_id");
    let proof_hash: String = runtime::get_named_arg("proof_hash");
    let _path: String = runtime::get_named_arg("path");

    let batches_uref = get_batches_uref();
    let batch_opt: Option<BatchRecord> = storage::dictionary_get(batches_uref, &batch_id).unwrap_or_revert();
    let batch = batch_opt.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_BATCH_NOT_FOUND)));

    let ((_, _, merkle_root), (_, finalized, _), (verified_count, added_count, status)) = batch;

    if finalized == 0 {
        runtime::revert(ApiError::User(ERR_BATCH_NOT_FINALIZED));
    }

    let proof_key = format!("{}:{}", batch_id, proof_hash);
    let proofs_uref = get_proofs_uref();
    let proof_data: Option<String> = storage::dictionary_get(proofs_uref, &proof_key).unwrap_or_revert();

    if proof_data.is_none() {
        runtime::revert(ApiError::User(ERR_PROOF_NOT_FOUND));
    }

    let new_verified = verified_count + 1;

    let updated_batch: BatchRecord = (
        (batch.0.0, batch.0.1, merkle_root),
        (batch.1.0, finalized, batch.1.2),
        (new_verified, added_count, status),
    );

    storage::dictionary_put(batches_uref, &batch_id, updated_batch);
}

#[no_mangle]
pub extern "C" fn finalize_batch() {
    let batch_id: String = runtime::get_named_arg("batch_id");

    let batches_uref = get_batches_uref();
    let batch_opt: Option<BatchRecord> = storage::dictionary_get(batches_uref, &batch_id).unwrap_or_revert();
    let batch = batch_opt.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_BATCH_NOT_FOUND)));

    let ((batch_id_val, creator, merkle_root), (proof_count, finalized, created_at), (verified_count, added_count, status)) = batch;

    if finalized != 0 {
        runtime::revert(ApiError::User(ERR_BATCH_FINALIZED));
    }

    let caller = runtime::get_caller();
    let caller_str = account_hash_to_string(caller);
    if creator != caller_str {
        runtime::revert(ApiError::User(ERR_NOT_CREATOR));
    }

    let updated_batch: BatchRecord = (
        (batch_id_val, creator, merkle_root),
        (proof_count, 1, created_at),
        (verified_count, added_count, status),
    );

    storage::dictionary_put(batches_uref, &batch_id, updated_batch);

    let finalized_batches_uref: URef = runtime::get_key(FINALIZED_BATCHES_KEY).unwrap_or_revert().into_uref().unwrap_or_revert();
    let current_finalized: u64 = storage::read(finalized_batches_uref).unwrap_or_revert().unwrap_or(0);
    storage::write(finalized_batches_uref, current_finalized + 1);
}

#[no_mangle]
pub extern "C" fn get_batch() {
    let batch_id: String = runtime::get_named_arg("batch_id");

    let batches_uref = get_batches_uref();
    let batch_opt: Option<BatchRecord> = storage::dictionary_get(batches_uref, &batch_id).unwrap_or_revert();
    let batch = batch_opt.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_BATCH_NOT_FOUND)));

    let cl_value = CLValue::from_t(batch).unwrap_or_revert();
    runtime::ret(cl_value);
}

#[no_mangle]
pub extern "C" fn get_stats() {
    let batch_count_uref: URef = runtime::get_key(BATCH_COUNT_KEY).unwrap_or_revert().into_uref().unwrap_or_revert();
    let batch_count: u64 = storage::read(batch_count_uref).unwrap_or_revert().unwrap_or(0);

    let total_proofs_uref: URef = runtime::get_key(TOTAL_PROOFS_KEY).unwrap_or_revert().into_uref().unwrap_or_revert();
    let total_proofs: u64 = storage::read(total_proofs_uref).unwrap_or_revert().unwrap_or(0);

    let finalized_batches_uref: URef = runtime::get_key(FINALIZED_BATCHES_KEY).unwrap_or_revert().into_uref().unwrap_or_revert();
    let finalized_batches: u64 = storage::read(finalized_batches_uref).unwrap_or_revert().unwrap_or(0);

    let stats = (batch_count, total_proofs, finalized_batches);
    let cl_value = CLValue::from_t(stats).unwrap_or_revert();
    runtime::ret(cl_value);
}

#[no_mangle]
pub extern "C" fn call() {
    let mut entry_points = EntryPoints::new();

    entry_points.add_entry_point(EntityEntryPoint::new(
        "create_batch".to_string(),
        vec![
            Parameter::new("batch_id", CLType::String),
            Parameter::new("merkle_root", CLType::String),
            Parameter::new("proof_count", CLType::U64),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "add_proof".to_string(),
        vec![
            Parameter::new("batch_id", CLType::String),
            Parameter::new("proof_hash", CLType::String),
            Parameter::new("leaf_index", CLType::U64),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "verify_inclusion".to_string(),
        vec![
            Parameter::new("batch_id", CLType::String),
            Parameter::new("proof_hash", CLType::String),
            Parameter::new("path", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "finalize_batch".to_string(),
        vec![
            Parameter::new("batch_id", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "get_batch".to_string(),
        vec![
            Parameter::new("batch_id", CLType::String),
        ],
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

    let mut named_keys = NamedKeys::new();

    let batches_uref = storage::new_dictionary(BATCHES_DICT).unwrap_or_revert();
    named_keys.insert(BATCHES_DICT.to_string(), batches_uref.into());

    let proofs_uref = storage::new_dictionary(PROOFS_DICT).unwrap_or_revert();
    named_keys.insert(PROOFS_DICT.to_string(), proofs_uref.into());

    let batch_count_uref = storage::new_uref(0u64);
    named_keys.insert(BATCH_COUNT_KEY.to_string(), batch_count_uref.into());

    let total_proofs_uref = storage::new_uref(0u64);
    named_keys.insert(TOTAL_PROOFS_KEY.to_string(), total_proofs_uref.into());

    let finalized_batches_uref = storage::new_uref(0u64);
    named_keys.insert(FINALIZED_BATCHES_KEY.to_string(), finalized_batches_uref.into());

    let (contract_hash, _version) = storage::new_contract(entry_points, Some(named_keys), None, None);
    runtime::put_key(CONTRACT_NAME, contract_hash.into());
}