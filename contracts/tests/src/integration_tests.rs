// Tests added in commit 4.

#[cfg(test)]
mod proof_registry_tests {
    const ERR_AGENT_EXISTS: u16 = 300;
    const ERR_AGENT_NOT_FOUND: u16 = 301;
    const ERR_PROOF_NOT_FOUND: u16 = 302;
    const ERR_NOT_INSTALLER: u16 = 303;
    const ERR_EMPTY_AGENT_ID: u16 = 304;
    const ERR_INVALID_HASH: u16 = 305;

    const HASH_LEN: usize = 64;
    const INITIAL_SCORE: u64 = 50;
    const REVOKE_PENALTY: u64 = 5;

    fn validate_hash(h: &str) -> Result<(), u16> {
        if h.len() != HASH_LEN {
            return Err(ERR_INVALID_HASH);
        }
        if !h.chars().all(|c| c.is_ascii_hexdigit()) {
            return Err(ERR_INVALID_HASH);
        }
        Ok(())
    }

    fn validate_agent_id(id: &str) -> Result<(), u16> {
        if id.is_empty() {
            return Err(ERR_EMPTY_AGENT_ID);
        }
        Ok(())
    }

    fn compute_score(initial: u64, failed: u64) -> u64 {
        initial.saturating_sub(failed.saturating_mul(REVOKE_PENALTY))
    }

    #[test]
    fn valid_hash_accepted() {
        let h = "a".repeat(64);
        assert!(validate_hash(&h).is_ok());
    }

    #[test]
    fn short_hash_rejected() {
        assert_eq!(validate_hash("abc").unwrap_err(), ERR_INVALID_HASH);
    }

    #[test]
    fn long_hash_rejected() {
        let h = "a".repeat(65);
        assert_eq!(validate_hash(&h).unwrap_err(), ERR_INVALID_HASH);
    }

    #[test]
    fn non_hex_hash_rejected() {
        let h = "g".repeat(64);
        assert_eq!(validate_hash(&h).unwrap_err(), ERR_INVALID_HASH);
    }

    #[test]
    fn empty_agent_id_rejected() {
        assert_eq!(validate_agent_id("").unwrap_err(), ERR_EMPTY_AGENT_ID);
    }

    #[test]
    fn valid_agent_id() {
        assert!(validate_agent_id("agent-007").is_ok());
    }

    #[test]
    fn score_starts_at_50() {
        assert_eq!(compute_score(INITIAL_SCORE, 0), 50);
    }

    #[test]
    fn score_decreases_on_revoke() {
        assert_eq!(compute_score(INITIAL_SCORE, 1), 45);
        assert_eq!(compute_score(INITIAL_SCORE, 3), 35);
    }

    #[test]
    fn score_floors_at_zero() {
        assert_eq!(compute_score(INITIAL_SCORE, 100), 0);
    }

    #[test]
    fn proof_id_sequential() {
        let mut counter = 0u64;
        let ids: Vec<String> = (0..4)
            .map(|_| {
                counter = counter.saturating_add(1);
                format!("proof-{}", counter)
            })
            .collect();
        assert_eq!(ids.len(), 4);
        assert_eq!(ids[0], "proof-1");
        assert_eq!(ids[3], "proof-4");
    }

    #[test]
    fn error_codes_distinct() {
        let codes = [
            ERR_AGENT_EXISTS, ERR_AGENT_NOT_FOUND, ERR_PROOF_NOT_FOUND,
            ERR_NOT_INSTALLER, ERR_EMPTY_AGENT_ID, ERR_INVALID_HASH,
        ];
        let mut sorted = codes.to_vec();
        sorted.sort();
        sorted.dedup();
        assert_eq!(sorted.len(), codes.len());
    }
}


#[cfg(test)]
mod verifier_gate_tests {
    const MAX_BATCH_SIZE: usize = 50;
    const ERR_BATCH_TOO_LARGE: u16 = 400;
    const ERR_RATE_LIMITED: u16 = 401;
    const ERR_INVALID_PROOF_ID: u16 = 402;

    fn validate_batch_size(n: usize) -> Result<(), u16> {
        if n > MAX_BATCH_SIZE {
            Err(ERR_BATCH_TOO_LARGE)
        } else {
            Ok(())
        }
    }

    fn check_rate_limit(count: u64, max: u64) -> Result<(), u16> {
        if count >= max {
            Err(ERR_RATE_LIMITED)
        } else {
            Ok(())
        }
    }

    #[test]
    fn batch_within_limit() {
        assert!(validate_batch_size(1).is_ok());
        assert!(validate_batch_size(50).is_ok());
    }

    #[test]
    fn batch_over_limit() {
        assert_eq!(validate_batch_size(51).unwrap_err(), ERR_BATCH_TOO_LARGE);
    }

    #[test]
    fn batch_zero_accepted() {
        assert!(validate_batch_size(0).is_ok());
    }

    #[test]
    fn rate_limit_ok() {
        assert!(check_rate_limit(0, 10).is_ok());
        assert!(check_rate_limit(9, 10).is_ok());
    }

    #[test]
    fn rate_limit_exceeded() {
        assert_eq!(check_rate_limit(10, 10).unwrap_err(), ERR_RATE_LIMITED);
        assert_eq!(check_rate_limit(100, 10).unwrap_err(), ERR_RATE_LIMITED);
    }
}


#[cfg(test)]
mod defi_mock_tests {
    const ERR_NOT_ADMIN: u16 = 500;
    const ERR_ALREADY_WHITELISTED: u16 = 501;
    const ERR_NOT_WHITELISTED: u16 = 502;
    const ERR_PROOF_INVALID: u16 = 503;
    const ERR_REVOKED: u16 = 504;

    fn check_whitelist_status(whitelisted: bool, revoked: bool) -> Result<(), u16> {
        if revoked {
            return Err(ERR_REVOKED);
        }
        if !whitelisted {
            return Err(ERR_NOT_WHITELISTED);
        }
        Ok(())
    }

    fn check_admin(caller: &str, admin: &str) -> Result<(), u16> {
        if caller != admin {
            Err(ERR_NOT_ADMIN)
        } else {
            Ok(())
        }
    }

    #[test]
    fn whitelisted_user_ok() {
        assert!(check_whitelist_status(true, false).is_ok());
    }

    #[test]
    fn non_whitelisted_rejected() {
        assert_eq!(
            check_whitelist_status(false, false).unwrap_err(),
            ERR_NOT_WHITELISTED
        );
    }

    #[test]
    fn revoked_user_rejected() {
        assert_eq!(
            check_whitelist_status(true, true).unwrap_err(),
            ERR_REVOKED
        );
    }

    #[test]
    fn revoked_takes_priority() {
        assert_eq!(
            check_whitelist_status(false, true).unwrap_err(),
            ERR_REVOKED
        );
    }

    #[test]
    fn admin_authorized() {
        assert!(check_admin("admin-1", "admin-1").is_ok());
    }

    #[test]
    fn non_admin_rejected() {
        assert_eq!(
            check_admin("random", "admin-1").unwrap_err(),
            ERR_NOT_ADMIN
        );
    }
}

// ---------------------------------------------------------------------------
// Governance / timelock / pause tests (BACKLOG 1.9-1.11).
//
// The Casper engine-test-support harness is disabled in this workspace (see
// Cargo.toml), so we mirror the contract semantics as pure-Rust reference
// functions and prove the invariants hold. Every branch in the WASM contract
// has a matching branch here; new branches must add a matching test.
// ---------------------------------------------------------------------------

#[cfg(test)]
mod governance_tests {
    const ERR_NOT_OWNER: u16 = 1;
    const ERR_NOT_GUARDIAN: u16 = 2;
    const ERR_PROPOSAL_NOT_FOUND: u16 = 3;
    const ERR_TIMELOCK_NOT_ELAPSED: u16 = 4;
    const ERR_ALREADY_EXECUTED: u16 = 5;
    const ERR_ALREADY_CANCELLED: u16 = 6;
    const ERR_PAUSED: u16 = 7;
    const ERR_NOT_PAUSED: u16 = 8;
    const ERR_RECOVERY_ALREADY_SIGNED: u16 = 11;
    const ERR_RECOVERY_INSUFFICIENT_SIGS: u16 = 12;

    const DEFAULT_TIMELOCK_SECS: u64 = 48 * 60 * 60;

    #[derive(Clone, Debug)]
    struct Proposal {
        pid: String,
        kind: String,
        payload: String,
        proposed_at_ms: u64,
        executable_at_ms: u64,
        executed: bool,
        cancelled: bool,
    }

    fn make_proposal(pid: &str, kind: &str, payload: &str, now_ms: u64, timelock_secs: u64) -> Proposal {
        Proposal {
            pid: pid.into(),
            kind: kind.into(),
            payload: payload.into(),
            proposed_at_ms: now_ms,
            executable_at_ms: now_ms.saturating_add(timelock_secs.saturating_mul(1000)),
            executed: false,
            cancelled: false,
        }
    }

    fn try_execute(p: &mut Proposal, now_ms: u64, caller_is_owner: bool, paused: bool) -> Result<(), u16> {
        if !caller_is_owner {
            return Err(ERR_NOT_OWNER);
        }
        if paused {
            return Err(ERR_PAUSED);
        }
        if p.cancelled {
            return Err(ERR_ALREADY_CANCELLED);
        }
        if p.executed {
            return Err(ERR_ALREADY_EXECUTED);
        }
        if now_ms < p.executable_at_ms {
            return Err(ERR_TIMELOCK_NOT_ELAPSED);
        }
        p.executed = true;
        Ok(())
    }

    fn try_cancel(p: &mut Proposal, caller_is_owner: bool) -> Result<(), u16> {
        if !caller_is_owner {
            return Err(ERR_NOT_OWNER);
        }
        if p.executed {
            return Err(ERR_ALREADY_EXECUTED);
        }
        p.cancelled = true;
        Ok(())
    }

    fn try_pause(caller_is_owner: bool, caller_is_guardian: bool, paused_ms: &mut u64, now_ms: u64) -> Result<(), u16> {
        if !caller_is_owner && !caller_is_guardian {
            return Err(ERR_NOT_GUARDIAN);
        }
        *paused_ms = now_ms;
        Ok(())
    }

    fn try_execute_unpause(p: &mut Proposal, paused_ms: &mut u64, now_ms: u64, caller_is_owner: bool) -> Result<(), u16> {
        if !caller_is_owner {
            return Err(ERR_NOT_OWNER);
        }
        if p.kind != "unpause" {
            return Err(ERR_PROPOSAL_NOT_FOUND);
        }
        if p.executed {
            return Err(ERR_ALREADY_EXECUTED);
        }
        if p.cancelled {
            return Err(ERR_ALREADY_CANCELLED);
        }
        if now_ms < p.executable_at_ms {
            return Err(ERR_TIMELOCK_NOT_ELAPSED);
        }
        if *paused_ms == 0 {
            return Err(ERR_NOT_PAUSED);
        }
        *paused_ms = 0;
        p.executed = true;
        Ok(())
    }

    // Recovery state (2-of-3 guardians co-sign a candidate).
    #[derive(Default, Clone, Debug)]
    struct Recovery {
        sigs: Vec<String>,
        executed: bool,
    }

    fn try_sign_recovery(rec: &mut Recovery, caller: &str, caller_is_guardian: bool) -> Result<(), u16> {
        if !caller_is_guardian {
            return Err(ERR_NOT_GUARDIAN);
        }
        if rec.executed {
            return Ok(());
        }
        if rec.sigs.iter().any(|s| s == caller) {
            return Err(ERR_RECOVERY_ALREADY_SIGNED);
        }
        rec.sigs.push(caller.to_string());
        Ok(())
    }

    fn try_execute_recovery(rec: &mut Recovery, owner: &mut String, candidate: &str, caller_is_guardian: bool) -> Result<(), u16> {
        if !caller_is_guardian {
            return Err(ERR_NOT_GUARDIAN);
        }
        if rec.executed {
            return Err(ERR_ALREADY_EXECUTED);
        }
        if rec.sigs.len() < 2 {
            return Err(ERR_RECOVERY_INSUFFICIENT_SIGS);
        }
        *owner = candidate.to_string();
        rec.executed = true;
        Ok(())
    }

    // ---- propose / execute / cancel ----

    #[test]
    fn execute_before_timelock_rejected() {
        let mut p = make_proposal("G-1", "generic", "0xabc", 0, DEFAULT_TIMELOCK_SECS);
        assert_eq!(try_execute(&mut p, 100, true, false).unwrap_err(), ERR_TIMELOCK_NOT_ELAPSED);
    }

    #[test]
    fn execute_after_timelock_succeeds() {
        let mut p = make_proposal("G-1", "generic", "0xabc", 0, DEFAULT_TIMELOCK_SECS);
        let after = DEFAULT_TIMELOCK_SECS.saturating_mul(1000) + 1;
        assert!(try_execute(&mut p, after, true, false).is_ok());
        assert!(p.executed);
    }

    #[test]
    fn execute_twice_rejected() {
        let mut p = make_proposal("G-1", "generic", "0xabc", 0, DEFAULT_TIMELOCK_SECS);
        let after = DEFAULT_TIMELOCK_SECS.saturating_mul(1000) + 1;
        assert!(try_execute(&mut p, after, true, false).is_ok());
        assert_eq!(try_execute(&mut p, after, true, false).unwrap_err(), ERR_ALREADY_EXECUTED);
    }

    #[test]
    fn execute_after_cancel_rejected() {
        let mut p = make_proposal("G-1", "generic", "0xabc", 0, DEFAULT_TIMELOCK_SECS);
        try_cancel(&mut p, true).unwrap();
        let after = DEFAULT_TIMELOCK_SECS.saturating_mul(1000) + 1;
        assert_eq!(try_execute(&mut p, after, true, false).unwrap_err(), ERR_ALREADY_CANCELLED);
    }

    #[test]
    fn non_owner_cannot_execute() {
        let mut p = make_proposal("G-1", "generic", "0xabc", 0, DEFAULT_TIMELOCK_SECS);
        let after = DEFAULT_TIMELOCK_SECS.saturating_mul(1000) + 1;
        assert_eq!(try_execute(&mut p, after, false, false).unwrap_err(), ERR_NOT_OWNER);
    }

    #[test]
    fn non_owner_cannot_cancel() {
        let mut p = make_proposal("G-1", "generic", "0xabc", 0, DEFAULT_TIMELOCK_SECS);
        assert_eq!(try_cancel(&mut p, false).unwrap_err(), ERR_NOT_OWNER);
    }

    #[test]
    fn cancel_after_execute_rejected() {
        let mut p = make_proposal("G-1", "generic", "0xabc", 0, DEFAULT_TIMELOCK_SECS);
        let after = DEFAULT_TIMELOCK_SECS.saturating_mul(1000) + 1;
        try_execute(&mut p, after, true, false).unwrap();
        assert_eq!(try_cancel(&mut p, true).unwrap_err(), ERR_ALREADY_EXECUTED);
    }

    // ---- pause ----

    #[test]
    fn owner_can_pause() {
        let mut paused = 0u64;
        assert!(try_pause(true, false, &mut paused, 500).is_ok());
        assert_eq!(paused, 500);
    }

    #[test]
    fn guardian_can_pause() {
        let mut paused = 0u64;
        assert!(try_pause(false, true, &mut paused, 700).is_ok());
        assert_eq!(paused, 700);
    }

    #[test]
    fn random_cannot_pause() {
        let mut paused = 0u64;
        assert_eq!(try_pause(false, false, &mut paused, 700).unwrap_err(), ERR_NOT_GUARDIAN);
    }

    #[test]
    fn execute_blocked_while_paused() {
        let mut p = make_proposal("G-1", "generic", "0xabc", 0, DEFAULT_TIMELOCK_SECS);
        let after = DEFAULT_TIMELOCK_SECS.saturating_mul(1000) + 1;
        assert_eq!(try_execute(&mut p, after, true, true).unwrap_err(), ERR_PAUSED);
    }

    #[test]
    fn unpause_before_timelock_rejected() {
        let mut p = make_proposal("G-1", "unpause", "", 0, DEFAULT_TIMELOCK_SECS);
        let mut paused = 500u64;
        assert_eq!(try_execute_unpause(&mut p, &mut paused, 100, true).unwrap_err(), ERR_TIMELOCK_NOT_ELAPSED);
        assert_eq!(paused, 500);
    }

    #[test]
    fn unpause_after_timelock_clears_flag() {
        let mut p = make_proposal("G-1", "unpause", "", 0, DEFAULT_TIMELOCK_SECS);
        let mut paused = 500u64;
        let after = DEFAULT_TIMELOCK_SECS.saturating_mul(1000) + 1;
        assert!(try_execute_unpause(&mut p, &mut paused, after, true).is_ok());
        assert_eq!(paused, 0);
        assert!(p.executed);
    }

    #[test]
    fn unpause_when_not_paused_rejected() {
        let mut p = make_proposal("G-1", "unpause", "", 0, DEFAULT_TIMELOCK_SECS);
        let mut paused = 0u64;
        let after = DEFAULT_TIMELOCK_SECS.saturating_mul(1000) + 1;
        assert_eq!(try_execute_unpause(&mut p, &mut paused, after, true).unwrap_err(), ERR_NOT_PAUSED);
    }

    #[test]
    fn unpause_wrong_kind_rejected() {
        let mut p = make_proposal("G-1", "generic", "", 0, DEFAULT_TIMELOCK_SECS);
        let mut paused = 500u64;
        let after = DEFAULT_TIMELOCK_SECS.saturating_mul(1000) + 1;
        assert_eq!(try_execute_unpause(&mut p, &mut paused, after, true).unwrap_err(), ERR_PROPOSAL_NOT_FOUND);
    }

    // ---- recovery ----

    #[test]
    fn recovery_needs_two_signatures() {
        let mut rec = Recovery::default();
        try_sign_recovery(&mut rec, "guardian1", true).unwrap();
        let mut owner = "old".to_string();
        assert_eq!(try_execute_recovery(&mut rec, &mut owner, "new-owner", true).unwrap_err(), ERR_RECOVERY_INSUFFICIENT_SIGS);
    }

    #[test]
    fn recovery_succeeds_with_two_distinct_guardians() {
        let mut rec = Recovery::default();
        try_sign_recovery(&mut rec, "g1", true).unwrap();
        try_sign_recovery(&mut rec, "g2", true).unwrap();
        let mut owner = "old".to_string();
        assert!(try_execute_recovery(&mut rec, &mut owner, "new-owner", true).is_ok());
        assert_eq!(owner, "new-owner");
        assert!(rec.executed);
    }

    #[test]
    fn recovery_same_guardian_twice_rejected() {
        let mut rec = Recovery::default();
        try_sign_recovery(&mut rec, "g1", true).unwrap();
        assert_eq!(try_sign_recovery(&mut rec, "g1", true).unwrap_err(), ERR_RECOVERY_ALREADY_SIGNED);
    }

    #[test]
    fn recovery_non_guardian_rejected() {
        let mut rec = Recovery::default();
        assert_eq!(try_sign_recovery(&mut rec, "attacker", false).unwrap_err(), ERR_NOT_GUARDIAN);
    }

    #[test]
    fn recovery_replay_rejected() {
        let mut rec = Recovery::default();
        try_sign_recovery(&mut rec, "g1", true).unwrap();
        try_sign_recovery(&mut rec, "g2", true).unwrap();
        let mut owner = "old".to_string();
        try_execute_recovery(&mut rec, &mut owner, "new-owner", true).unwrap();
        assert_eq!(try_execute_recovery(&mut rec, &mut owner, "new-owner", true).unwrap_err(), ERR_ALREADY_EXECUTED);
    }
}
