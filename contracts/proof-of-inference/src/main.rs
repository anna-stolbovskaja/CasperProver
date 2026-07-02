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

// Constants for named keys and dictionary names
const PROOF_DICTIONARY: &str = "proof_dict";
const VERIFIER_DICTIONARY: &str = "verifier_dict";
const PROOF_COUNTER_KEY: &str = "proof_counter";
const INSTALLER_KEY: &str = "installer";
const PRICE_BPS_KEY: &str = "price_bps"; // New key for revenue model

// Default values
const DEFAULT_PRICE_BPS: u64 = 0; // Default to 0, can be configured

// Error codes
const ERR_NOT_INSTALLER: u16 = 1;
const ERR_PROOF_NOT_FOUND: u16 = 2;
const ERR_VERIFIER_NOT_FOUND: u16 = 3;
const ERR_VERIFIER_EXISTS: u16 = 4;
const ERR_INVALID_STATUS: u16 = 5;
const ERR_NOT_VERIFIER: u16 = 6;
const ERR_ALREADY_VERIFIED: u16 = 7;
const ERR_NOT_CHALLENGED: u16 = 8;

// Proof status:
// 0 = Registered (initial state)
// 1 = Verified
// 2 = Challenged
// 3 = Rejected (challenge resolved, proof invalid)
// 4 = Resolved (challenge resolved, proof valid)

/// Store proof fields in the dictionary as a nested triple of triples.
/// Layout: ((proof_id, agent_id, model_hash), (input_hash, output_hash, proof_hash), ((timestamp, status, evidence), price_bps))
/// price_bps is captured at creation time for the revenue model.
type ProofRecord = (
    (String, String, String),      // (proof_id, agent_id, model_hash)
    (String, String, String),      // (input_hash, output_hash, proof_hash)
    ((u64, u64, String), u64),     // ((timestamp, status, evidence), price_bps)
);

/// Store verifier fields in the dictionary as a nested triple and a triple.
/// Layout: ((verifier_id, pub_key, registered_at), (verified_count, is_active, revoked_at))
type VerifierRecord = (
    (String, String, u64), // (verifier_id, pub_key, registered_at)
    (u64, u64, u64),       // (verified_count, is_active, revoked_at)
);

// Helper functions for storage access and assertions

/// Retrieves the AccountHash of the contract installer.
fn get_installer() -> AccountHash {
    let key: Key = runtime::get_key(INSTALLER_KEY).unwrap_or_revert();
    key.into_account().unwrap_or_revert()
}

/// Asserts that the caller is the contract installer, reverting if not.
fn assert_installer() {
    if runtime::get_caller() != get_installer() {
        runtime::revert(ApiError::User(ERR_NOT_INSTALLER));
    }
}

/// Retrieves the URef for a named dictionary.
fn get_dict_uref(name: &str) -> URef {
    runtime::get_key(name)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert()
}

/// Constructs a dictionary key for a proof.
fn proof_key(proof_id: &str) -> String {
    format!("proof:{}", proof_id)
}

/// Constructs a dictionary key for a verifier.
fn verifier_key(verifier_id: &str) -> String {
    format!("verifier:{}", verifier_id)
}

/// Reads the current price basis points from storage.
fn read_price_bps() -> u64 {
    let uref = runtime::get_key(PRICE_BPS_KEY)
        .unwrap_or_revert()
        .into_uref()
        .unwrap_or_revert();
    storage::read::<u64>(uref)
        .unwrap_or_revert()
        .unwrap_or(DEFAULT_PRICE_BPS)
}

// Entry points

/// Registers a new proof with its associated metadata.
///
/// Parameters:
/// - `model_hash`: Hash of the AI model used.
/// - `input_hash`: Hash of the input data.
/// - `output_hash`: Hash of the output data.
/// - `proof_hash`: Hash of the inference proof.
/// - `agent_id`: Identifier of the agent submitting the proof.
/// - `price_bps`: Basis points for the price associated with this proof.
#[no_mangle]
pub extern "C" fn register_proof() {
    let model_hash: String = runtime::get_named_arg("model_hash");
    let input_hash: String = runtime::get_named_arg("input_hash");
    let output_hash: String = runtime::get_named_arg("output_hash");
    let proof_hash: String = runtime::get_named_arg("proof_hash");
    let agent_id: String = runtime::get_named_arg("agent_id");
    let price_bps: u64 = runtime::get_named_arg("price_bps");

    let counter_key = runtime::get_key(PROOF_COUNTER_KEY).unwrap_or_revert();
    let counter_uref = counter_key.into_uref().unwrap_or_revert();

    let proof_id_num: u64 = storage::read(counter_uref)
        .unwrap_or_revert()
        .unwrap_or(0u64);
    let proof_id = proof_id_num.to_string();

    storage::write(counter_uref, proof_id_num + 1);

    let timestamp: u64 = runtime::get_blocktime().into();

    let proof_record: ProofRecord = (
        (proof_id.clone(), agent_id, model_hash),
        (input_hash, output_hash, proof_hash),
        ((timestamp, 0u64, String::new()), price_bps),
    );

    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    storage::dictionary_put(dict_uref, &proof_key(&proof_id), proof_record);
}

/// Verifies a registered proof. Only active verifiers can call this.
///
/// Parameters:
/// - `proof_id`: The ID of the proof to verify.
#[no_mangle]
pub extern "C" fn verify_proof() {
    let proof_id: String = runtime::get_named_arg("proof_id");
    let caller = runtime::get_caller();

    // Update proof status
    let proof_dict_uref = get_dict_uref(PROOF_DICTIONARY);
    let proof_key_str = proof_key(&proof_id);

    let mut proof_record: ProofRecord = storage::dictionary_get(proof_dict_uref, &proof_key_str)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROOF_NOT_FOUND));

    if proof_record.2.0.1 != 0 { // Check status (0 = Registered)
        runtime::revert(ApiError::User(ERR_ALREADY_VERIFIED));
    }

    proof_record.2.0.1 = 1; // Set status to Verified

    storage::dictionary_put(proof_dict_uref, &proof_key_str, proof_record);

    // Update verifier stats
    let verifier_dict_uref = get_dict_uref(VERIFIER_DICTIONARY);
    let verifier_id = caller.to_string();
    let verifier_key_str = verifier_key(&verifier_id);

    let mut verifier_record: VerifierRecord = storage::dictionary_get(verifier_dict_uref, &verifier_key_str)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_NOT_VERIFIER));

    if verifier_record.1.1 == 0 { // Check if verifier is active
        runtime::revert(ApiError::User(ERR_NOT_VERIFIER));
    }

    verifier_record.1.0 += 1; // Increment verified_count

    storage::dictionary_put(verifier_dict_uref, &verifier_key_str, verifier_record);
}

/// Challenges a registered or verified proof, providing evidence.
///
/// Parameters:
/// - `proof_id`: The ID of the proof to challenge.
/// - `evidence`: String containing evidence for the challenge.
#[no_mangle]
pub extern "C" fn challenge_proof() {
    let proof_id: String = runtime::get_named_arg("proof_id");
    let evidence: String = runtime::get_named_arg("evidence");

    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    let proof_key_str = proof_key(&proof_id);

    let mut record: ProofRecord = storage::dictionary_get(dict_uref, &proof_key_str)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROOF_NOT_FOUND));

    // Proof must be in Registered (0) or Verified (1) state to be challenged
    if record.2.0.1 != 0 && record.2.0.1 != 1 {
        runtime::revert(ApiError::User(ERR_INVALID_STATUS));
    }

    record.2.0.1 = 2; // Set status to Challenged
    record.2.0.2 = evidence; // Store evidence

    storage::dictionary_put(dict_uref, &proof_key_str, record);
}

/// Resolves a challenged proof. Only the installer can call this.
///
/// Parameters:
/// - `proof_id`: The ID of the proof to resolve.
/// - `valid`: 0 if the proof is invalid (rejected), 1 if valid (resolved).
#[no_mangle]
pub extern "C" fn resolve_challenge() {
    assert_installer();

    let proof_id: String = runtime::get_named_arg("proof_id");
    let valid: u64 = runtime::get_named_arg("valid");

    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    let proof_key_str = proof_key(&proof_id);

    let mut record: ProofRecord = storage::dictionary_get(dict_uref, &proof_key_str)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROOF_NOT_FOUND));

    if record.2.0.1 != 2 { // Proof must be in Challenged (2) state
        runtime::revert(ApiError::User(ERR_NOT_CHALLENGED));
    }

    if valid == 0 {
        record.2.0.1 = 3; // Set status to Rejected
    } else {
        record.2.0.1 = 4; // Set status to Resolved (valid)
    }

    storage::dictionary_put(dict_uref, &proof_key_str, record);
}

/// Registers a new verifier. Only the installer can call this.
///
/// Parameters:
/// - `verifier_id`: The AccountHash of the verifier as a String.
/// - `pub_key`: Public key associated with the verifier.
#[no_mangle]
pub extern "C" fn register_verifier() {
    assert_installer();

    let verifier_id: String = runtime::get_named_arg("verifier_id");
    let pub_key: String = runtime::get_named_arg("pub_key");

    let dict_uref = get_dict_uref(VERIFIER_DICTIONARY);
    let v_key = verifier_key(&verifier_id);

    let existing: Option<VerifierRecord> = storage::dictionary_get(dict_uref, &v_key)
        .unwrap_or_revert();

    if existing.is_some() {
        runtime::revert(ApiError::User(ERR_VERIFIER_EXISTS));
    }

    let timestamp: u64 = runtime::get_blocktime().into();

    let record: VerifierRecord = (
        (verifier_id, pub_key, timestamp),
        (0u64, 1u64, 0u64), // verified_count, is_active (1=active), revoked_at
    );

    storage::dictionary_put(dict_uref, &v_key, record);
}

/// Revokes a verifier's active status. Only the installer can call this.
///
/// Parameters:
/// - `verifier_id`: The AccountHash of the verifier as a String.
#[no_mangle]
pub extern "C" fn revoke_verifier() {
    assert_installer();

    let verifier_id: String = runtime::get_named_arg("verifier_id");

    let dict_uref = get_dict_uref(VERIFIER_DICTIONARY);
    let v_key = verifier_key(&verifier_id);

    let mut record: VerifierRecord = storage::dictionary_get(dict_uref, &v_key)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_VERIFIER_NOT_FOUND));

    record.1.1 = 0; // Set is_active to 0 (inactive)
    record.1.2 = runtime::get_blocktime().into(); // Set revoked_at timestamp

    storage::dictionary_put(dict_uref, &v_key, record);
}

/// Retrieves a proof record by its ID.
///
/// Parameters:
/// - `proof_id`: The ID of the proof to retrieve.
/// Returns: The ProofRecord as CLValue::Any.
#[no_mangle]
pub extern "C" fn get_proof() {
    let proof_id: String = runtime::get_named_arg("proof_id");

    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    let proof_key_str = proof_key(&proof_id);

    let record: ProofRecord = storage::dictionary_get(dict_uref, &proof_key_str)
        .unwrap_or_revert()
        .unwrap_or_revert_with(ApiError::User(ERR_PROOF_NOT_FOUND));

    runtime::ret(CLValue::from_t(record).unwrap_or_revert());
}

/// Retrieves statistics about all proofs.
/// Returns: A tuple (total_proofs, registered, verified, challenged, rejected, resolved) as CLValue::Any.
#[no_mangle]
pub extern "C" fn get_stats() {
    let dict_uref = get_dict_uref(PROOF_DICTIONARY);
    let counter_key = runtime::get_key(PROOF_COUNTER_KEY).unwrap_or_revert();
    let counter_uref = counter_key.into_uref().unwrap_or_revert();
    let total_proofs: u64 = storage::read(counter_uref)
        .unwrap_or_revert()
        .unwrap_or(0);

    let mut registered: u64 = 0;
    let mut verified: u64 = 0;
    let mut challenged: u64 = 0;
    let mut rejected: u64 = 0;
    let mut resolved: u64 = 0;

    for i in 0..total_proofs {
        let proof_id = i.to_string();
        let key_str = proof_key(&proof_id);

        let record: Option<ProofRecord> = storage::dictionary_get(dict_uref, &key_str)
            .unwrap_or_revert();

        if let Some(r) = record {
            match r.2.0.1 { // Accessing the status field
                0 => registered += 1,
                1 => verified += 1,
                2 => challenged += 1,
                3 => rejected += 1,
                4 => resolved += 1,
                _ => {}
            }
        }
    }

    let stats = (
        (total_proofs, registered, verified),
        (challenged, rejected, resolved),
    );

    runtime::ret(CLValue::from_t(stats).unwrap_or_revert());
}

/// Creates and returns the EntryPoints for the contract.
fn create_entry_points() -> EntryPoints {
    let mut entry_points = EntryPoints::new();

    entry_points.add_entry_point(EntityEntryPoint::new(
        "register_proof",
        vec![
            Parameter::new("model_hash", CLType::String),
            Parameter::new("input_hash", CLType::String),
            Parameter::new("output_hash", CLType::String),
            Parameter::new("proof_hash", CLType::String),
            Parameter::new("agent_id", CLType::String),
            Parameter::new("price_bps", CLType::U64), // New parameter
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "verify_proof",
        vec![Parameter::new("proof_id", CLType::String)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "challenge_proof",
        vec![
            Parameter::new("proof_id", CLType::String),
            Parameter::new("evidence", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "resolve_challenge",
        vec![
            Parameter::new("proof_id", CLType::String),
            Parameter::new("valid", CLType::U64),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "register_verifier",
        vec![
            Parameter::new("verifier_id", CLType::String),
            Parameter::new("pub_key", CLType::String),
        ],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "revoke_verifier",
        vec![Parameter::new("verifier_id", CLType::String)],
        CLType::Unit,
        EntryPointAccess::Public,
        EntryPointType::Called,
        EntryPointPayment::Caller,
    ));

    entry_points.add_entry_point(EntityEntryPoint::new(
        "get_proof",
        vec![Parameter::new("proof_id", CLType::String)],
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

    entry_points
}

/// Contract installation entry point.
/// Initializes dictionaries, counter, installer key, and deploys the contract.
#[no_mangle]
pub extern "C" fn call() {
    let installer = runtime::get_caller();

    // Create dictionaries and URefs
    let proof_dict = storage::new_dictionary(PROOF_DICTIONARY).unwrap_or_revert();
    let verifier_dict = storage::new_dictionary(VERIFIER_DICTIONARY).unwrap_or_revert();
    let counter_uref = storage::new_uref(0u64);
    let price_bps_uref = storage::new_uref(DEFAULT_PRICE_BPS); // Initialize price_bps

    // Prepare named keys for the contract
    let mut named_keys = NamedKeys::new();
    named_keys.insert(PROOF_DICTIONARY.into(), proof_dict.into());
    named_keys.insert(VERIFIER_DICTIONARY.into(), verifier_dict.into());
    named_keys.insert(PROOF_COUNTER_KEY.into(), counter_uref.into());
    named_keys.insert(INSTALLER_KEY.into(), Key::Account(installer));
    named_keys.insert(PRICE_BPS_KEY.into(), price_bps_uref.into());

    // Create entry points
    let entry_points = create_entry_points();

    // Store contract
    let (contract_hash, _) = storage::new_contract(
        entry_points,
        Some(named_keys),
        Some("proof_of_inference_package_hash".into()),
        Some("proof_of_inference_access_uref".into()),
        None, // No contract_hash_name, will put_key manually
    );

    // Put the contract hash in the caller's account
    runtime::put_key("proof_of_inference_contract", contract_hash.into());
}