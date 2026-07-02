#![no_std]
#![no_main]

extern crate alloc;

use alloc::format;
use alloc::string::{String, ToString};
use alloc::vec;
use alloc::vec::Vec;

use casper_contract::contract_api::{runtime, storage, system};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;
use casper_types::account::AccountHash;
use casper_types::contracts::NamedKeys;
use casper_types::{
    ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointType, EntryPoints, Key,
    Parameter, URef, U512, EntryPointPayment,
};

const CONTRACT_NAME: &str = "ProofAggregationRegistry";
const BATCHES_DICT: &str = "batches";
const PROOFS_DICT: &str = "proofs";
const BATCH_COUNT_KEY: &str = "batch_count";
const TOTAL_PROOFS_KEY: &str = "total_proofs";
const FINALIZED_BATCHES_KEY: &str = "finalized_batches";
const INSTALLER_KEY: &str = "installer";
const PRICE_BPS_KEY: &str = "price_bps";
const DEFAULT_PRICE_BPS: u64 = 0; // Default price in basis points (0 = free)

const ERR_BATCH_EXISTS: u16 = 1;
const ERR_BATCH_NOT_FOUND: u16 = 2;
const ERR_BATCH_FINALIZED: u16 = 3;
const ERR_BATCH_NOT_FINALIZED: u16 = 4;
const ERR_NOT_CREATOR: u16 = 5;
const ERR_BATCH_FULL: u16 = 6;
const ERR_PROOF_NOT_FOUND: u16 = 7;
const ERR_INVALID_PROOF: u16 = 8; // Reused for general invalid data
const ERR_INVALID_ACCOUNT_HASH: u16 = 9;

/// Store batch fields in the dictionary as a nested triple of triples.
/// Layout: ((batch_id, creator, merkle_root), (max_proofs, finalized_status, created_at), (verified_count, added_count, price_bps))
/// `finalized_status`: 0 for not finalized, 1 for finalized.
/// `price_bps`: captured at creation time to prevent recalculation drift.
type BatchRecord = ((String, String, String), (u64, u64, u64), (u64, u64, u64));

fn get_dict_uref(name: &str) -> URef {
    runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

fn read_price_bps() -> u64 {
    let uref = runtime::get_key(PRICE_BPS_KEY)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert();
    storage::read::<u64>(uref)
        .unwrap_or_revert()
        .unwrap_or(DEFAULT_PRICE_BPS)
}

fn account_hash_to_string(account: AccountHash) -> String {
    account.to_string()
}

fn string_to_account_hash(s: &str) -> AccountHash {
    AccountHash::from_formatted_str(s)
        .unwrap_or_else(|_| runtime::revert(ApiError::User(ERR_INVALID_ACCOUNT_HASH)))
}

fn get_current_timestamp() -> u64 {
    runtime::get_blocktime().into()
}

#[no_mangle]
pub extern "C" fn create_batch() {
    let batch_id: String = runtime::get_named_arg("batch_id");
    let merkle_root: String = runtime::get_named_arg("merkle_root");
    let max_proofs: u64 = runtime::get_named_arg("max_proofs");

    let batches_uref = get_dict_uref(BATCHES_DICT);
    let existing: Option<BatchRecord> = storage::dictionary_get(batches_uref, &batch_id).unwrap_or_revert();
    if existing.is_some() {
        runtime::revert(ApiError::User(ERR_BATCH_EXISTS));
    }

    let caller = runtime::get_caller();
    let creator = account_hash_to_string(caller);
    let created_at = get_current_timestamp();
    let price_bps = read_price_bps();

    let batch: BatchRecord = (
        (batch_id.clone(), creator, merkle_root),
        (max_proofs, 0, created_at), // 0 for not finalized
        (0, 0, price_bps), // verified_count, added_count, price_bps
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

    let batches_uref = get_dict_uref(BATCHES_DICT);
    let batch_opt: Option<BatchRecord> = storage::dictionary_get(batches_uref, &batch_id).unwrap_or_revert();
    let mut batch = batch_opt.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_BATCH_NOT_FOUND)));

    let ((batch_id_val, creator, merkle_root), (max_proofs, finalized, created_at), (verified_count, mut added_count, price_bps)) = batch;

    if finalized != 0 { // 0 for not finalized, 1 for finalized
        runtime::revert(ApiError::User(ERR_BATCH_FINALIZED));
    }

    if added_count >= max_proofs {
        runtime::revert(ApiError::User(ERR_BATCH_FULL));
    }

    let caller = runtime::get_caller();
    let caller_str = account_hash_to_string(caller);
    if creator != caller_str {
        runtime::revert(ApiError::User(ERR_NOT_CREATOR));
    }

    let proof_key = format!("{}:{}", batch_id, proof_hash);

    let proofs_uref = get_dict_uref(PROOFS_DICT);
    let existing_proof: Option<String> = storage::dictionary_get(proofs_uref, &proof_key).unwrap_or_revert();
    if existing_proof.is_some() {
        runtime::revert(ApiError::User(ERR_BATCH_EXISTS)); // Proof already added to this batch
    }

    storage::dictionary_put(proofs_uref, &proof_key, leaf_index.to_string());

    added_count += 1;

    let updated_batch: BatchRecord = (
        (batch_id_val, creator, merkle_root),
        (max_proofs, finalized, created_at),
        (verified_count, added_count, price_bps),
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
    let _path: String = runtime::get_named_arg("path"); // Path is not used for on-chain verification in this contract

    let batches_uref = get_dict_uref(BATCHES_DICT);
    let batch_opt: Option<BatchRecord> = storage::dictionary_get(batches_uref, &batch_id).unwrap_or_revert();
    let mut batch = batch_opt.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_BATCH_NOT_FOUND)));

    let ((batch_id_val, creator, merkle_root), (max_proofs, finalized, created_at), (mut verified_count, added_count, price_bps)) = batch;

    if finalized == 0 { // 0 for not finalized
        runtime::revert(ApiError::User(ERR_BATCH_NOT_FINALIZED));
    }

    let proof_key = format!("{}:{}", batch_id, proof_hash);
    let proofs_uref = get_dict_uref(PROOFS_DICT);
    let proof_data: Option<String> = storage::dictionary_get(proofs_uref, &proof_key).unwrap_or_revert();

    if proof_data.is_none() {
        runtime::revert(ApiError::User(ERR_PROOF_NOT_FOUND));
    }

    verified_count += 1;

    let updated_batch: BatchRecord = (
        (batch_id_val, creator, merkle_root),
        (max_proofs, finalized, created_at),
        (verified_count, added_count, price_bps),
    );

    storage::dictionary_put(batches_uref, &batch_id, updated_batch);
}

#[no_mangle]
pub extern "C" fn finalize_batch() {
    let batch_id: String = runtime::get_named_arg("batch_id");

    let batches_uref = get_dict_uref(BATCHES_DICT);
    let batch_opt: Option<BatchRecord> = storage::dictionary_get(batches_uref, &batch_id).unwrap_or_revert();
    let mut batch = batch_opt.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_BATCH_NOT_FOUND)));

    let ((batch_id_val, creator, merkle_root), (max_proofs, mut finalized, created_at), (verified_count, added_count, price_bps)) = batch;

    if finalized != 0 {
        runtime::revert(ApiError::User(ERR_BATCH_FINALIZED));
    }

    let caller = runtime::get_caller();
    let caller_str = account_hash_to_string(caller);
    if creator != caller_str {
        runtime::revert(ApiError::User(ERR_NOT_CREATOR));
    }

    finalized = 1; // Set finalized status to 1

    let updated_batch: BatchRecord = (
        (batch_id_val, creator, merkle_root),
        (max_proofs, finalized, created_at),
        (verified_count, added_count, price_bps),
    );

    storage::dictionary_put(batches_uref, &batch_id, updated_batch);

    let finalized_batches_uref: URef = runtime::get_key(FINALIZED_BATCHES_KEY).unwrap_or_revert().into_uref().unwrap_or_revert();
    let current_finalized: u64 = storage::read(finalized_batches_uref).unwrap_or_revert().unwrap_or(0);
    storage::write(finalized_batches_uref, current_finalized + 1);
}

#[no_mangle]
pub extern "C" fn get_batch() {
    let batch_id: String = runtime::get_named_arg("batch_id");

    let batches_uref = get_dict_uref(BATCHES_DICT);
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
pub extern "C" fn configure_price() {
    let new_price_bps: u64 = runtime::get_named_arg("new_price_bps");

    let installer_key = runtime::get_key(INSTALLER_KEY)
        .unwrap_or_revert();
    let installer_account_hash = installer_key.into_account().unwrap_or_revert();

    let caller = runtime::get_caller();

    if caller != installer_account_hash {
        runtime::revert(ApiError::PermissionDenied);
    }

    let price_bps_uref = runtime::get_key(PRICE_BPS_KEY)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert();

    storage::write(price_bps_uref, new_price_bps);
}

#[no_mangle]
pub extern "C" fn call() {
    let installer = runtime::get_caller();

    let mut entry_points = EntryPoints::new();

    entry_points.add_entry_point(EntityEntryPoint::new(
        "create_batch",
        vec![
            Parameter::new("batch_id", CLType::String),
            Parameter::new("merkle_root", CLType::String),
            Parameter::new("max_proofs", CLType::U64),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "add_proof",
        vec![
            Parameter::new("batch_id", CLType::String),
            Parameter::new("proof_hash", CLType::String),
            Parameter::new("leaf_index", CLType::U64),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "verify_inclusion",
        vec![
            Parameter::new("batch_id", CLType::String),
            Parameter::new("proof_hash", CLType::String),
            Parameter::new("path", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "finalize_batch",
        vec![
            Parameter::new("batch_id", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "get_batch",
        vec![
            Parameter::new("batch_id", CLType::String),
        ],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "get_stats",
        vec![],
        CLType::Any,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "configure_price",
        vec![Parameter::new("new_price_bps", CLType::U64)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    let mut named_keys = NamedKeys::new();

    let batches_uref = storage::new_dictionary(BATCHES_DICT).unwrap_or_revert();
    named_keys.insert(BATCHES_DICT.into(), batches_uref.into());

    let proofs_uref = storage::new_dictionary(PROOFS_DICT).unwrap_or_revert();
    named_keys.insert(PROOFS_DICT.into(), proofs_uref.into());

    let batch_count_uref = storage::new_uref(0u64);
    named_keys.insert(BATCH_COUNT_KEY.into(), batch_count_uref.into());

    let total_proofs_uref = storage::new_uref(0u64);
    named_keys.insert(TOTAL_PROOFS_KEY.into(), total_proofs_uref.into());

    let finalized_batches_uref = storage::new_uref(0u64);
    named_keys.insert(FINALIZED_BATCHES_KEY.into(), finalized_batches_uref.into());

    let price_bps_uref = storage::new_uref(DEFAULT_PRICE_BPS);
    named_keys.insert(PRICE_BPS_KEY.into(), price_bps_uref.into());

    named_keys.insert(INSTALLER_KEY.into(), Key::Account(installer));

    let (contract_hash, _package_hash) = storage::new_contract(
        entry_points,
        Some(named_keys),
        Some(format!("{}_package_hash", CONTRACT_NAME).into()),
        Some(format!("{}_access_uref", CONTRACT_NAME).into()),
        None,
    );
    runtime::put_key(CONTRACT_NAME, contract_hash.into());
}