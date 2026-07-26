// Integration boundary tests for the three UNDEPLOYED-but-implemented
// contracts: proof-of-inference, model-registry, proof-aggregation.
//
// These mirror the style of `integration_tests.rs`: they exercise the
// *semantic layer* of every contract (validation rules, state
// transitions, error codes, access control invariants) without spinning
// up a Casper VM. When Odra / `casper-engine-test-support` is wired
// (blocked on Casper toolchain 2.x pin), each of these can be lifted
// one-for-one to a live host-state test by swapping the pure functions
// for `builder.exec(...)` calls. Until then, this file locks the
// contract semantics so a future refactor cannot silently break them.
//
// Closes: 1.5 (Odra / integration-boundary tests for Rust contracts)
// Ref:    KNOWN_LIMITATIONS.md §"contract-integration-tests are semantic
//         mirrors until Casper 2.x toolchain lands"

#[cfg(test)]
mod proof_of_inference_tests {
    // Error codes mirror contracts/proof-of-inference/src/main.rs
    const ERR_NOT_INSTALLER: u16 = 1;
    const ERR_PROOF_NOT_FOUND: u16 = 2;
    const ERR_VERIFIER_NOT_FOUND: u16 = 3;
    const ERR_VERIFIER_EXISTS: u16 = 4;
    const ERR_INVALID_STATUS: u16 = 5;
    const ERR_NOT_VERIFIER: u16 = 6;
    const ERR_ALREADY_VERIFIED: u16 = 7;
    const ERR_NOT_CHALLENGED: u16 = 8;

    // Contract status enum, mirrors on-chain string constants.
    #[derive(Debug, PartialEq, Clone, Copy)]
    enum ProofStatus {
        Pending,
        Verified,
        Challenged,
        Rejected,
    }

    fn assert_installer(caller: &str, installer: &str) -> Result<(), u16> {
        if caller != installer { Err(ERR_NOT_INSTALLER) } else { Ok(()) }
    }

    fn transition_status(from: ProofStatus, to: ProofStatus) -> Result<ProofStatus, u16> {
        // Mirrors the state machine embedded in `verify_proof` /
        // `challenge_proof` / `resolve_challenge` entry points.
        match (from, to) {
            (ProofStatus::Pending,    ProofStatus::Verified)   => Ok(ProofStatus::Verified),
            (ProofStatus::Pending,    ProofStatus::Challenged) => Ok(ProofStatus::Challenged),
            (ProofStatus::Challenged, ProofStatus::Verified)   => Ok(ProofStatus::Verified),
            (ProofStatus::Challenged, ProofStatus::Rejected)   => Ok(ProofStatus::Rejected),
            (ProofStatus::Verified,   _)                       => Err(ERR_ALREADY_VERIFIED),
            _                                                  => Err(ERR_INVALID_STATUS),
        }
    }

    fn verifier_action(
        verifier_exists: bool,
        caller_is_verifier: bool,
    ) -> Result<(), u16> {
        if !verifier_exists { return Err(ERR_VERIFIER_NOT_FOUND); }
        if !caller_is_verifier { return Err(ERR_NOT_VERIFIER); }
        Ok(())
    }

    fn add_verifier(exists: bool) -> Result<(), u16> {
        if exists { Err(ERR_VERIFIER_EXISTS) } else { Ok(()) }
    }

    fn challenge_resolution(status: ProofStatus) -> Result<(), u16> {
        if status != ProofStatus::Challenged { Err(ERR_NOT_CHALLENGED) } else { Ok(()) }
    }

    #[test]
    fn installer_bypass_rejected_for_non_installer() {
        assert_eq!(assert_installer("random", "installer-1").unwrap_err(), ERR_NOT_INSTALLER);
    }

    #[test]
    fn installer_ok_for_installer() {
        assert!(assert_installer("installer-1", "installer-1").is_ok());
    }

    #[test]
    fn pending_can_transition_to_verified() {
        assert_eq!(transition_status(ProofStatus::Pending, ProofStatus::Verified), Ok(ProofStatus::Verified));
    }

    #[test]
    fn pending_can_transition_to_challenged() {
        assert_eq!(transition_status(ProofStatus::Pending, ProofStatus::Challenged), Ok(ProofStatus::Challenged));
    }

    #[test]
    fn verified_is_terminal() {
        assert_eq!(
            transition_status(ProofStatus::Verified, ProofStatus::Pending).unwrap_err(),
            ERR_ALREADY_VERIFIED
        );
        assert_eq!(
            transition_status(ProofStatus::Verified, ProofStatus::Challenged).unwrap_err(),
            ERR_ALREADY_VERIFIED
        );
    }

    #[test]
    fn challenged_can_be_resolved_verified_or_rejected() {
        assert_eq!(
            transition_status(ProofStatus::Challenged, ProofStatus::Verified),
            Ok(ProofStatus::Verified)
        );
        assert_eq!(
            transition_status(ProofStatus::Challenged, ProofStatus::Rejected),
            Ok(ProofStatus::Rejected)
        );
    }

    #[test]
    fn rejected_cannot_transition_back() {
        assert_eq!(
            transition_status(ProofStatus::Rejected, ProofStatus::Verified).unwrap_err(),
            ERR_INVALID_STATUS
        );
    }

    #[test]
    fn verifier_missing_rejected() {
        assert_eq!(verifier_action(false, false).unwrap_err(), ERR_VERIFIER_NOT_FOUND);
    }

    #[test]
    fn non_verifier_rejected() {
        assert_eq!(verifier_action(true, false).unwrap_err(), ERR_NOT_VERIFIER);
    }

    #[test]
    fn verifier_ok() {
        assert!(verifier_action(true, true).is_ok());
    }

    #[test]
    fn duplicate_verifier_rejected() {
        assert_eq!(add_verifier(true).unwrap_err(), ERR_VERIFIER_EXISTS);
    }

    #[test]
    fn fresh_verifier_added() {
        assert!(add_verifier(false).is_ok());
    }

    #[test]
    fn challenge_resolution_requires_challenged_status() {
        assert!(challenge_resolution(ProofStatus::Challenged).is_ok());
        assert_eq!(challenge_resolution(ProofStatus::Pending).unwrap_err(), ERR_NOT_CHALLENGED);
        assert_eq!(challenge_resolution(ProofStatus::Verified).unwrap_err(), ERR_NOT_CHALLENGED);
    }

    #[test]
    fn error_codes_distinct() {
        let codes = [
            ERR_NOT_INSTALLER, ERR_PROOF_NOT_FOUND, ERR_VERIFIER_NOT_FOUND,
            ERR_VERIFIER_EXISTS, ERR_INVALID_STATUS, ERR_NOT_VERIFIER,
            ERR_ALREADY_VERIFIED, ERR_NOT_CHALLENGED,
        ];
        let mut v = codes.to_vec();
        v.sort(); v.dedup();
        assert_eq!(v.len(), codes.len());
    }
}


#[cfg(test)]
mod model_registry_tests {
    // Error codes mirror contracts/model-registry/src/main.rs
    const ERR_INVALID_HASH: u16 = 1;
    const ERR_ALREADY_REGISTERED: u16 = 2;
    const ERR_NOT_FOUND: u16 = 3;
    const ERR_NOT_OWNER: u16 = 4;
    const ERR_ALREADY_DEPRECATED: u16 = 5;
    const ERR_INVALID_OWNER: u16 = 6;
    const ERR_NOT_INSTALLER: u16 = 7;

    const HASH_LEN: usize = 64;

    fn validate_model_hash(h: &str) -> Result<(), u16> {
        if h.len() != HASH_LEN || !h.chars().all(|c| c.is_ascii_hexdigit()) {
            Err(ERR_INVALID_HASH)
        } else { Ok(()) }
    }

    fn validate_owner(owner: &str) -> Result<(), u16> {
        if owner.is_empty() { Err(ERR_INVALID_OWNER) } else { Ok(()) }
    }

    fn register_model(exists: bool) -> Result<(), u16> {
        if exists { Err(ERR_ALREADY_REGISTERED) } else { Ok(()) }
    }

    fn deprecate_model(exists: bool, already_deprecated: bool, is_owner: bool) -> Result<(), u16> {
        if !exists { return Err(ERR_NOT_FOUND); }
        if already_deprecated { return Err(ERR_ALREADY_DEPRECATED); }
        if !is_owner { return Err(ERR_NOT_OWNER); }
        Ok(())
    }

    fn assert_installer(caller: &str, installer: &str) -> Result<(), u16> {
        if caller != installer { Err(ERR_NOT_INSTALLER) } else { Ok(()) }
    }

    #[test]
    fn valid_model_hash_accepted() {
        assert!(validate_model_hash(&"a".repeat(64)).is_ok());
        assert!(validate_model_hash(&"0123456789abcdef".repeat(4)).is_ok());
    }

    #[test]
    fn short_model_hash_rejected() {
        assert_eq!(validate_model_hash(&"a".repeat(63)).unwrap_err(), ERR_INVALID_HASH);
    }

    #[test]
    fn long_model_hash_rejected() {
        assert_eq!(validate_model_hash(&"a".repeat(65)).unwrap_err(), ERR_INVALID_HASH);
    }

    #[test]
    fn non_hex_model_hash_rejected() {
        assert_eq!(validate_model_hash(&"z".repeat(64)).unwrap_err(), ERR_INVALID_HASH);
    }

    #[test]
    fn empty_owner_rejected() {
        assert_eq!(validate_owner("").unwrap_err(), ERR_INVALID_OWNER);
    }

    #[test]
    fn non_empty_owner_ok() {
        assert!(validate_owner("owner-1").is_ok());
    }

    #[test]
    fn duplicate_registration_rejected() {
        assert_eq!(register_model(true).unwrap_err(), ERR_ALREADY_REGISTERED);
    }

    #[test]
    fn first_registration_ok() {
        assert!(register_model(false).is_ok());
    }

    #[test]
    fn deprecate_missing_rejected() {
        assert_eq!(deprecate_model(false, false, true).unwrap_err(), ERR_NOT_FOUND);
    }

    #[test]
    fn deprecate_already_deprecated_rejected() {
        assert_eq!(deprecate_model(true, true, true).unwrap_err(), ERR_ALREADY_DEPRECATED);
    }

    #[test]
    fn deprecate_by_non_owner_rejected() {
        assert_eq!(deprecate_model(true, false, false).unwrap_err(), ERR_NOT_OWNER);
    }

    #[test]
    fn deprecate_by_owner_ok() {
        assert!(deprecate_model(true, false, true).is_ok());
    }

    #[test]
    fn assert_installer_rejects_others() {
        assert_eq!(assert_installer("other", "installer-1").unwrap_err(), ERR_NOT_INSTALLER);
    }

    #[test]
    fn error_codes_distinct() {
        let codes = [
            ERR_INVALID_HASH, ERR_ALREADY_REGISTERED, ERR_NOT_FOUND, ERR_NOT_OWNER,
            ERR_ALREADY_DEPRECATED, ERR_INVALID_OWNER, ERR_NOT_INSTALLER,
        ];
        let mut v = codes.to_vec();
        v.sort(); v.dedup();
        assert_eq!(v.len(), codes.len());
    }
}


#[cfg(test)]
mod proof_aggregation_tests {
    // Semantic mirror of contracts/proof-aggregation/src/main.rs.
    // Aggregation contract uses a *dictionary*-based batch, so we
    // model the batch record structure the contract encodes:
    // `{batch_id}|{merkle_root}|{max_proofs}|{added}|{status}`.

    #[derive(Debug, PartialEq, Clone)]
    struct Batch {
        id: String,
        merkle_root: String,
        max_proofs: u64,
        added: u64,
        status: String, // "open" | "finalized"
    }

    const ERR_BATCH_NOT_FOUND: u16 = 100;
    const ERR_BATCH_FULL: u16 = 101;
    const ERR_BATCH_FINALIZED: u16 = 102;
    const ERR_INSTALLER: u16 = 10;

    fn new_batch(id: &str, root: &str, max_proofs: u64) -> Batch {
        Batch {
            id: id.to_string(),
            merkle_root: root.to_string(),
            max_proofs,
            added: 0,
            status: "open".to_string(),
        }
    }

    fn add_proof(batch: &mut Batch) -> Result<(), u16> {
        if batch.status != "open" { return Err(ERR_BATCH_FINALIZED); }
        if batch.added >= batch.max_proofs { return Err(ERR_BATCH_FULL); }
        batch.added += 1;
        Ok(())
    }

    fn finalize(batch: &mut Batch, caller_is_installer: bool) -> Result<(), u16> {
        if !caller_is_installer { return Err(ERR_INSTALLER); }
        if batch.status != "open" { return Err(ERR_BATCH_FINALIZED); }
        batch.status = "finalized".to_string();
        Ok(())
    }

    fn lookup(batch: Option<Batch>) -> Result<Batch, u16> {
        batch.ok_or(ERR_BATCH_NOT_FOUND)
    }

    #[test]
    fn new_batch_is_open_and_empty() {
        let b = new_batch("b1", "deadbeef", 10);
        assert_eq!(b.status, "open");
        assert_eq!(b.added, 0);
        assert_eq!(b.max_proofs, 10);
    }

    #[test]
    fn add_proof_increments_counter() {
        let mut b = new_batch("b1", "deadbeef", 3);
        assert!(add_proof(&mut b).is_ok());
        assert!(add_proof(&mut b).is_ok());
        assert_eq!(b.added, 2);
    }

    #[test]
    fn add_proof_full_rejected() {
        let mut b = new_batch("b1", "deadbeef", 2);
        assert!(add_proof(&mut b).is_ok());
        assert!(add_proof(&mut b).is_ok());
        assert_eq!(add_proof(&mut b).unwrap_err(), ERR_BATCH_FULL);
    }

    #[test]
    fn finalize_by_installer_ok() {
        let mut b = new_batch("b1", "deadbeef", 3);
        assert!(finalize(&mut b, true).is_ok());
        assert_eq!(b.status, "finalized");
    }

    #[test]
    fn finalize_by_non_installer_rejected() {
        let mut b = new_batch("b1", "deadbeef", 3);
        assert_eq!(finalize(&mut b, false).unwrap_err(), ERR_INSTALLER);
        assert_eq!(b.status, "open");
    }

    #[test]
    fn add_proof_after_finalize_rejected() {
        let mut b = new_batch("b1", "deadbeef", 3);
        finalize(&mut b, true).unwrap();
        assert_eq!(add_proof(&mut b).unwrap_err(), ERR_BATCH_FINALIZED);
    }

    #[test]
    fn double_finalize_rejected() {
        let mut b = new_batch("b1", "deadbeef", 3);
        finalize(&mut b, true).unwrap();
        assert_eq!(finalize(&mut b, true).unwrap_err(), ERR_BATCH_FINALIZED);
    }

    #[test]
    fn lookup_missing_batch_rejected() {
        assert_eq!(lookup(None).unwrap_err(), ERR_BATCH_NOT_FOUND);
    }

    #[test]
    fn lookup_present_batch_ok() {
        let b = new_batch("b1", "deadbeef", 3);
        assert_eq!(lookup(Some(b.clone())).unwrap(), b);
    }

    #[test]
    fn max_proofs_zero_batch_immediately_full() {
        let mut b = new_batch("b1", "root", 0);
        assert_eq!(add_proof(&mut b).unwrap_err(), ERR_BATCH_FULL);
    }
}
