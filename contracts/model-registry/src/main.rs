#![no_std]
#![no_main]
extern crate alloc;

use alloc::string::String;
use alloc::vec::Vec;
use alloc::vec;
use casper_contract::contract_api::{runtime, storage, system};
use casper_types::{ApiError, CLType, CLValue, EntityEntryPoint, EntryPointAccess, EntryPointType, EntryPoints, Parameter, URef, U512, Key};
use casper_types::account::AccountHash;
use casper_types::contracts::NamedKeys;
use casper_types::EntryPointPayment;
use casper_contract::unwrap_or_revert::UnwrapOrRevert;

const MODELS_DICT: &str = "models_dict";
const MODELS_REF: &str = "models_ref";

const ERR_INVALID_HASH: u16 = 1;
const ERR_ALREADY_REGISTERED: u16 = 2;
const ERR_NOT_FOUND: u16 = 3;
const ERR_NOT_OWNER: u16 = 4;
const ERR_ALREADY_DEPRECATED: u16 = 5;
const ERR_INVALID_OWNER: u16 = 6;

fn get_models_uref() -> URef {
    let key: Key = runtime::get_key(MODELS_REF).unwrap_or_revert().into_uref().unwrap_or_revert().into();
    key.into_uref().unwrap_or_revert()
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

#[no_mangle]
pub extern "C" fn register_model() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    let name: String = runtime::get_named_arg("name");
    let version: String = runtime::get_named_arg("version");
    let ipfs_cid: String = runtime::get_named_arg("ipfs_cid");
    
    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }
    
    let uref = get_models_uref();
    let existing: Option<String> = storage::dictionary_get(uref, &model_hash).unwrap_or_revert();
    if existing.is_some() {
        runtime::revert(ApiError::User(ERR_ALREADY_REGISTERED));
    }
    
    let caller = runtime::get_caller();
    let timestamp = get_timestamp();
    let owner = format!("{:?}", caller);
    
    let record = (
        (model_hash.clone(), name, owner),
        (version, ipfs_cid, timestamp),
        (0u64, 0u64, timestamp),
    );
    
    storage::dictionary_put(uref, &model_hash, record);
}

#[no_mangle]
pub extern "C" fn update_model() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    let version: String = runtime::get_named_arg("version");
    let ipfs_cid: String = runtime::get_named_arg("ipfs_cid");
    
    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }
    
    let uref = get_models_uref();
    let record: Option<((String, String, String), (String, String, u64), (u64, u64, u64))> = 
        storage::dictionary_get(uref, &model_hash).unwrap_or_revert();
    
    let mut record = record.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_NOT_FOUND)));
    
    let caller = runtime::get_caller();
    let owner_str = format!("{:?}", caller);
    if record.0.2 != owner_str {
        runtime::revert(ApiError::User(ERR_NOT_OWNER));
    }
    
    if record.2.1 != 0 {
        runtime::revert(ApiError::User(ERR_ALREADY_DEPRECATED));
    }
    
    let timestamp = get_timestamp();
    record.1.0 = version;
    record.1.1 = ipfs_cid;
    record.2.2 = timestamp;
    
    storage::dictionary_put(uref, &model_hash, record);
}

#[no_mangle]
pub extern "C" fn deprecate_model() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    
    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }
    
    let uref = get_models_uref();
    let record: Option<((String, String, String), (String, String, u64), (u64, u64, u64))> = 
        storage::dictionary_get(uref, &model_hash).unwrap_or_revert();
    
    let mut record = record.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_NOT_FOUND)));
    
    let caller = runtime::get_caller();
    let owner_str = format!("{:?}", caller);
    if record.0.2 != owner_str {
        runtime::revert(ApiError::User(ERR_NOT_OWNER));
    }
    
    if record.2.1 != 0 {
        runtime::revert(ApiError::User(ERR_ALREADY_DEPRECATED));
    }
    
    record.2.1 = get_timestamp();
    
    storage::dictionary_put(uref, &model_hash, record);
}

#[no_mangle]
pub extern "C" fn verify_model() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    
    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }
    
    let uref = get_models_uref();
    let record: Option<((String, String, String), (String, String, u64), (u64, u64, u64))> = 
        storage::dictionary_get(uref, &model_hash).unwrap_or_revert();
    
    let mut record = record.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_NOT_FOUND)));
    
    record.2.0 = get_timestamp();
    
    storage::dictionary_put(uref, &model_hash, record);
}

#[no_mangle]
pub extern "C" fn get_model() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    
    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }
    
    let uref = get_models_uref();
    let record: Option<((String, String, String), (String, String, u64), (u64, u64, u64))> = 
        storage::dictionary_get(uref, &model_hash).unwrap_or_revert();
    
    let record = record.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_NOT_FOUND)));
    
    let cl_value = CLValue::from_t(record).unwrap_or_revert();
    runtime::ret(cl_value);
}

#[no_mangle]
pub extern "C" fn transfer_ownership() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    let new_owner: String = runtime::get_named_arg("new_owner");
    
    if !validate_model_hash(&model_hash) {
        runtime::revert(ApiError::User(ERR_INVALID_HASH));
    }
    
    if new_owner.len() != 64 || !new_owner.starts_with("account-hash-") {
        if new_owner.len() != 66 {
            let _ = new_owner.parse::<AccountHash>().unwrap_or_else(|_| runtime::revert(ApiError::User(ERR_INVALID_OWNER)));
        }
    }
    
    let uref = get_models_uref();
    let record: Option<((String, String, String), (String, String, u64), (u64, u64, u64))> = 
        storage::dictionary_get(uref, &model_hash).unwrap_or_revert();
    
    let mut record = record.unwrap_or_else(|| runtime::revert(ApiError::User(ERR_NOT_FOUND)));
    
    let caller = runtime::get_caller();
    let owner_str = format!("{:?}", caller);
    if record.0.2 != owner_str {
        runtime::revert(ApiError::User(ERR_NOT_OWNER));
    }
    
    record.0.2 = new_owner;
    record.2.2 = get_timestamp();
    
    storage::dictionary_put(uref, &model_hash, record);
}

#[no_mangle]
pub extern "C" fn call() {
    let mut entry_points = EntryPoints::new();
    
    entry_points.add_entrypoint(EntityEntryPoint::new(
        "register_model",
        vec![
            Parameter::new("model_hash", String::cl_type()),
            Parameter::new("name", String::cl_type()),
            Parameter::new("version", String::cl_type()),
            Parameter::new("ipfs_cid", String::cl_type()),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));
    
    entry_points.add_entrypoint(EntityEntryPoint::new(
        "update_model",
        vec![
            Parameter::new("model_hash", String::cl_type()),
            Parameter::new("version", String::cl_type()),
            Parameter::new("ipfs_cid", String::cl_type()),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));
    
    entry_points.add_entrypoint(EntityEntryPoint::new(
        "deprecate_model",
        vec![Parameter::new("model_hash", String::cl_type())],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));
    
    entry_points.add_entrypoint(EntityEntryPoint::new(
        "verify_model",
        vec![Parameter::new("model_hash", String::cl_type())],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));
    
    entry_points.add_entrypoint(EntityEntryPoint::new(
        "get_model",
        vec![Parameter::new("model_hash", String::cl_type())],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));
    
    entry_points.add_entrypoint(EntityEntryPoint::new(
        "transfer_ownership",
        vec![
            Parameter::new("model_hash", String::cl_type()),
            Parameter::new("new_owner", String::cl_type()),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Contract,
        EntryPointPayment::Caller,
    ));
    
    let dict_uref = storage::new_dictionary(MODELS_DICT).unwrap_or_revert();
    let mut named_keys = NamedKeys::new();
    named_keys.insert(MODELS_REF.to_string(), dict_uref.into());
    
    storage::new_contract(entry_points, Some(named_keys), None, None);
}