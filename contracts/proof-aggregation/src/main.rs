#![no_std]
#![no_main]

extern crate alloc;

use alloc::format;
use alloc::string::{String, ToString};
use alloc::vec;
use alloc::vec::Vec;

use casper_contract::contract_api::{runtime, storage};
use casper_contract::unwrap_or_revert::UnwrapOrRevert;
use casper_types::account::AccountHash;
use casper_types::contracts::NamedKeys;
use casper_types::{
    ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointType, EntryPoints, Key,
    Parameter, URef, U512, EntryPointPayment,
};

const CONTRACT_NAME: &str = "proof_aggregation";
const BATCHES_DICT: &str = "agg_batches";
const BATCH_COUNT_KEY: &str = "agg_batch_count";
const INSTALLER_KEY: &str = "agg_installer";
const PRICE_BPS_KEY: &str = "agg_price_bps";

#[no_mangle]
pub extern "C" fn create_batch() {
    let batch_id: String = runtime::get_named_arg("batch_id");
    let merkle_root: String = runtime::get_named_arg("merkle_root");
    let max_proofs: u64 = runtime::get_named_arg("max_proofs");

    let batches_uref: URef = runtime::get_key(BATCHES_DICT)
        .unwrap_or_revert_with(ApiError::MissingKey)
        .into_uref()
        .unwrap_or_revert_with(ApiError::UnexpectedKeyVariant);

    let value = format!("{}|{}|{}|0|open", batch_id, merkle_root, max_proofs);
    storage::dictionary_put(batches_uref, &batch_id, value);

    let count_uref: URef = runtime::get_key(BATCH_COUNT_KEY)
        .unwrap_or_revert_with(ApiError::MissingKey)
        .into_uref()
        .unwrap_or_revert_with(ApiError::UnexpectedKeyVariant);
    let count: u64 = storage::read(count_uref)
        .unwrap_or_revert()
        .unwrap_or(0u64);
    storage::write(count_uref, count + 1);
}

#[no_mangle]
pub extern "C" fn add_proof() {
    let batch_id: String = runtime::get_named_arg("batch_id");
    let proof_hash: String = runtime::get_named_arg("proof_hash");
    let leaf_index: u64 = runtime::get_named_arg("leaf_index");

    let batches_uref: URef = runtime::get_key(BATCHES_DICT)
        .unwrap_or_revert_with(ApiError::MissingKey)
        .into_uref()
        .unwrap_or_revert_with(ApiError::UnexpectedKeyVariant);

    let key = format!("{}:{}", batch_id, proof_hash);
    let value = format!("{}|{}|{}", proof_hash, leaf_index, "added");
    storage::dictionary_put(batches_uref, &key, value);
}

#[no_mangle]
pub extern "C" fn finalize_batch() {
    let batch_id: String = runtime::get_named_arg("batch_id");
    
    let installer_key = runtime::get_key(INSTALLER_KEY)
        .unwrap_or_revert_with(ApiError::MissingKey);
    let installer = match installer_key {
        Key::Account(hash) => hash,
        _ => runtime::revert(ApiError::User(10)),
    };
    let caller = runtime::get_caller();
    if caller != installer {
        runtime::revert(ApiError::PermissionDenied);
    }

    let batches_uref: URef = runtime::get_key(BATCHES_DICT)
        .unwrap_or_revert_with(ApiError::MissingKey)
        .into_uref()
        .unwrap_or_revert_with(ApiError::UnexpectedKeyVariant);

    let value = format!("{}|finalized", batch_id);
    storage::dictionary_put(batches_uref, &batch_id, value);
}

#[no_mangle]
pub extern "C" fn get_stats() {
    let count_uref: URef = runtime::get_key(BATCH_COUNT_KEY)
        .unwrap_or_revert_with(ApiError::MissingKey)
        .into_uref()
        .unwrap_or_revert_with(ApiError::UnexpectedKeyVariant);
    let count: u64 = storage::read(count_uref)
        .unwrap_or_revert()
        .unwrap_or(0u64);
    
    let result = CLValue::from_t(count).unwrap_or_revert();
    runtime::ret(result);
}

fn create_entry_points() -> EntryPoints {
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
        "finalize_batch",
        vec![Parameter::new("batch_id", CLType::String)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "get_stats",
        vec![],
        CLType::U64,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points
}

#[no_mangle]
pub extern "C" fn call() {
    let installer = runtime::get_caller();

    let batches_uref = storage::new_dictionary(BATCHES_DICT).unwrap_or_revert();
    let count_uref = storage::new_uref(0u64);
    let price_uref = storage::new_uref(0u64);

    let mut named_keys = NamedKeys::new();
    named_keys.insert(BATCHES_DICT.into(), batches_uref.into());
    named_keys.insert(BATCH_COUNT_KEY.into(), count_uref.into());
    named_keys.insert(INSTALLER_KEY.into(), Key::Account(installer));
    named_keys.insert(PRICE_BPS_KEY.into(), price_uref.into());

    let entry_points = create_entry_points();

    let (contract_hash, _) = storage::new_contract(
        entry_points,
        Some(named_keys),
        Some("proof_aggregation_package_hash".into()),
        Some("proof_aggregation_access_uref".into()),
        None,
    );

    runtime::put_key(CONTRACT_NAME, contract_hash.into());
}
