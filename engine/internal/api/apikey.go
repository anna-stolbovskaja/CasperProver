package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// apiKeyPrefix is the human-readable prefix of every issued key so
// operators can spot one at a glance in logs / config. sk_live_ is the
// widely-known Stripe-style convention.
const apiKeyPrefix = "sk_live_"

// apiKeyRawBytes is the number of random bytes packed into a fresh key.
// 32 bytes = 256 bits of entropy = 64 hex characters after the prefix.
const apiKeyRawBytes = 32

// Scopes exposed on issued keys.
//
// A closed enum, deliberately small. Any downstream route that gates on
// a scope must pull the constant from here — no free-form strings, no
// wildcards, no "admin" (that's what X-Admin-API-Key is for; see PR-3).
//
//   - ScopeSubmit:        can POST /proofs, /proofs/batch, /proofs/{id}/revoke,
//                         /inference/{prove,verify}, /verify, /zk/*, /pq/*,
//                         /proof-chain/validate. The full tenant write surface.
//   - ScopeVerifyOnly:    can POST /verify, /zk/verify-groth16,
//                         /zk/batch-verify, /zk/groth16-real/verify,
//                         /pq/verify-sphincs, /pq/hybrid-verify.
//                         Read-only from the chain / registry point of view.
//                         Cannot submit or mutate anything.
//   - ScopeAdminReadonly: can hit the diagnostic surface an operator
//                         needs (currently: /kyc/check) without being
//                         able to grant KYC or finalize batches. Not
//                         a superset of Submit — a distinct axis.
//
// Grow this only after a review; every new scope changes the tenant
// trust boundary.
const (
	ScopeSubmit        = "submit"
	ScopeVerifyOnly    = "verify_only"
	ScopeAdminReadonly = "admin_readonly"
)

// ValidScopes lists the enum values accepted at issuance time. Kept as
// a package-level slice so tests / docs can enumerate without touching
// the constants list.
var ValidScopes = []string{ScopeSubmit, ScopeVerifyOnly, ScopeAdminReadonly}

// isValidScope reports whether s is one of ValidScopes.
func isValidScope(s string) bool {
	for _, v := range ValidScopes {
		if v == s {
			return true
		}
	}
	return false
}

// generateAPIKey returns a fresh plaintext key in the format
// `sk_live_<64 hex chars>`. It's the ONLY place plaintext ever exists —
// callers must return it once to the user and never persist it.
func generateAPIKey() (string, error) {
	buf := make([]byte, apiKeyRawBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	return apiKeyPrefix + hex.EncodeToString(buf), nil
}

// hashAPIKey deterministically maps a plaintext key to its sha256 hex
// digest. This is what gets stored in the DB; the plaintext never
// leaves the caller-visible response body of POST /admin/keys/issue.
func hashAPIKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
