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
    ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointPayment, EntryPointType,
    EntryPoints, Key, Parameter, URef, U512,
};

// Constants
const MODELS_DICT: &str = "models_dict";
const PRICE_BPS_KEY: &str = "price_bps";
const INSTALLER_KEY: &str = "installer";

const DEFAULT_PRICE_BPS: u64 = 0; // Default to 0 for no revenue

// Error codes
const ERR_INVALID_HASH: u16 = 1;
const ERR_ALREADY_REGISTERED: u16 = 2;
const ERR_NOT_FOUND: u16 = 3;
const ERR_NOT_OWNER: u16 = 4;
const ERR_ALREADY_DEPRECATED: u16 = 5;
const ERR_INVALID_OWNER: u16 = 6;
const ERR_NOT_INSTALLER: u16 = 7;

// HELPERS (storage access)
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

fn validate_model_hash(model_hash: &str) -> bool {
    if model_hash.len() != 64 {
        return false;
    }
    model_hash.chars().all(|c| c.is_ascii_hexdigit())
}

fn get_timestamp() -> u64 {
    runtime::get_blocktime().into()
}

fn get_installer_account_hash() -> AccountHash {
    runtime::get_key(INSTALLER_KEY)
        .unwrap_or_revert()
        .into_account()
        .unwrap_or_revert()
}

// RECORD TYPE (max 3 elements per tuple, nest triples)
/// Layout: ((name, owner, version), (ipfs_cid, registered_at, verified_at), (deprecated_at, updated_at, price_bps))
type ModelRecord = ((String, String, String), (String, u64, u64), (u64, u64, u64));

#[no_mangle]
pub extern "C" fn register_model() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    let name: String = runtime::get_named_arg("name");
    let version: String = runtime::get_named_arg("version");
    let ipfs_cid: String = runtime::get_named_arg("ipfs_cid");

    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }

    let models_dict_uref = get_dict_uref(MODELS_DICT);
    let existing: Option<ModelRecord> = storage::dictionary_get(models_dict_uref, &model_hash)
        .unwrap_or_revert();
    if existing.is_some() {
        runtime::revert(ApiError::User(ERR_ALREADY_REGISTERED));
    }

    let caller = runtime::get_caller();
    let timestamp = get_timestamp();
    let owner = format!("{:?}", caller); // AccountHash to String
    let price_bps = read_price_bps(); // Capture current price_bps

    let record: ModelRecord = (
        (name, owner, version),
        (ipfs_cid, timestamp, 0u64), // ipfs_cid, registered_at, verified_at (initially 0)
        (0u64, timestamp, price_bps), // deprecated_at (initially 0), updated_at, price_bps
    );

    storage::dictionary_put(models_dict_uref, &model_hash, record);
}

#[no_mangle]
pub extern "C" fn update_model() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    let version: String = runtime::get_named_arg("version");
    let ipfs_cid: String = runtime::get_named_arg("ipfs_cid");

    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }

    let models_dict_uref = get_dict_uref(MODELS_DICT);
    let record: Option<ModelRecord> = storage::dictionary_get(models_dict_uref, &model_hash)
        .unwrap_or_revert();

    let mut record = record.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_NOT_FOUND)));

    let caller = runtime::get_caller();
    let owner_str = format!("{:?}", caller);
    if record.0.1 != owner_str {
        // record.0.1 is owner
        runtime::revert(ApiError::User(ERR_NOT_OWNER));
    }

    if record.2.0 != 0 {
        // record.2.0 is deprecated_at
        runtime::revert(ApiError::User(ERR_ALREADY_DEPRECATED));
    }

    let timestamp = get_timestamp();
    record.0.2 = version; // record.0.2 is version
    record.1.0 = ipfs_cid; // record.1.0 is ipfs_cid
    record.2.1 = timestamp; // record.2.1 is updated_at

    storage::dictionary_put(models_dict_uref, &model_hash, record);
}

#[no_mangle]
pub extern "C" fn deprecate_model() {
    let model_hash: String = runtime::get_named_arg("model_hash");

    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }

    let models_dict_uref = get_dict_uref(MODELS_DICT);
    let record: Option<ModelRecord> = storage::dictionary_get(models_dict_uref, &model_hash)
        .unwrap_or_revert();

    let mut record = record.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_NOT_FOUND)));

    let caller = runtime::get_caller();
    let owner_str = format!("{:?}", caller);
    if record.0.1 != owner_str {
        // record.0.1 is owner
        runtime::revert(ApiError::User(ERR_NOT_OWNER));
    }

    if record.2.0 != 0 {
        // record.2.0 is deprecated_at
        runtime::revert(ApiError::User(ERR_ALREADY_DEPRECATED));
    }

    record.2.0 = get_timestamp(); // Set deprecated_at
    record.2.1 = get_timestamp(); // Also update updated_at

    storage::dictionary_put(models_dict_uref, &model_hash, record);
}

#[no_mangle]
pub extern "C" fn verify_model() {
    let model_hash: String = runtime::get_named_arg("model_hash");

    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }

    let models_dict_uref = get_dict_uref(MODELS_DICT);
    let record: Option<ModelRecord> = storage::dictionary_get(models_dict_uref, &model_hash)
        .unwrap_or_revert();

    let mut record = record.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_NOT_FOUND)));

    // No ownership check needed for verification
    record.1.2 = get_timestamp(); // Set verified_at
    record.2.1 = get_timestamp(); // Also update updated_at

    storage::dictionary_put(models_dict_uref, &model_hash, record);
}

#[no_mangle]
pub extern "C" fn get_model() {
    let model_hash: String = runtime::get_named_arg("model_hash");

    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }

    let models_dict_uref = get_dict_uref(MODELS_DICT);
    let record: Option<ModelRecord> = storage::dictionary_get(models_dict_uref, &model_hash)
        .unwrap_or_revert();

    let record = record.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_NOT_FOUND)));

    let cl_value = CLValue::from_t(record).unwrap_or_revert();
    runtime::ret(cl_value);
}

#[no_mangle]
pub extern "C" fn transfer_ownership() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    let new_owner_account_hash: AccountHash = runtime::get_named_arg("new_owner");

    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }

    let models_dict_uref = get_dict_uref(MODELS_DICT);
    let record: Option<ModelRecord> = storage::dictionary_get(models_dict_uref, &model_hash)
        .unwrap_or_revert();

    let mut record = record.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_NOT_FOUND)));

    let caller = runtime::get_caller();
    let owner_str = format!("{:?}", caller);
    if record.0.1 != owner_str {
        // record.0.1 is owner
        runtime::revert(ApiError::User(ERR_NOT_OWNER));
    }

    record.0.1 = format!("{:?}", new_owner_account_hash); // Update owner, store as String
    record.2.1 = get_timestamp(); // Update updated_at

    storage::dictionary_put(models_dict_uref, &model_hash, record);
}

#[no_mangle]
pub extern "C" fn configure_price_bps() {
    let new_price_bps: u64 = runtime::get_named_arg("new_price_bps");

    let caller = runtime::get_caller();
    let installer = get_installer_account_hash();

    if caller != installer {
        runtime::revert(ApiError::User(ERR_NOT_INSTALLER));
    }

    let price_bps_uref = runtime::get_key(PRICE_BPS_KEY)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert();
    storage::write(price_bps_uref, new_price_bps);
}

#[no_mangle]
pub extern "C" fn get_price_bps() {
    let price_bps = read_price_bps();
    runtime::ret(CLValue::from_t(price_bps).unwrap_or_revert());
}

#[no_mangle]
pub extern "C" fn call() {
    let installer = runtime::get_caller();

    let models_dict = storage::new_dictionary(MODELS_DICT).unwrap_or_revert();
    let price_bps_uref = storage::new_uref(DEFAULT_PRICE_BPS);

    let mut named_keys = NamedKeys::new();
    named_keys.insert(MODELS_DICT.into(), models_dict.into());
    named_keys.insert(PRICE_BPS_KEY.into(), price_bps_uref.into());
    named_keys.insert(INSTALLER_KEY.into(), Key::Account(installer));

    let mut entry_points = EntryPoints::new();

    entry_points.add_entry_point(EntityEntryPoint::new(
        "register_model",
        vec![
            Parameter::new("model_hash", CLType::String),
            Parameter::new("name", CLType::String),
            Parameter::new("version", CLType::String),
            Parameter::new("ipfs_cid", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "update_model",
        vec![
            Parameter::new("model_hash", CLType::String),
            Parameter::new("version", CLType::String),
            Parameter::new("ipfs_cid", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "deprecate_model",
        vec![Parameter::new("model_hash", CLType::String)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "verify_model",
        vec![Parameter::new("model_hash", CLType::String)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "get_model",
        vec![Parameter::new("model_hash", CLType::String)],
        CLType::Any, // Return type is ModelRecord, which is a complex tuple
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "transfer_ownership",
        vec![
            Parameter::new("model_hash", CLType::String),
            Parameter::new("new_owner", CLType::ByteArray(32)), // AccountHash as ByteArray(32)
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "configure_price_bps",
        vec![Parameter::new("new_price_bps", CLType::U64)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "get_price_bps",
        vec![],
        CLType::U64,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    let (contract_hash, _) = storage::new_contract(
        entry_points,
        Some(named_keys),
        Some("model_registry_package_hash".into()),
        Some("model_registry_access_uref".into()),
        None,
    );
    runtime::put_key("model_registry_contract", contract_hash.into());
}