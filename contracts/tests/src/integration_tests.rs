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
