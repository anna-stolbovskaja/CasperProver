package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// apiKeyPrefix is the human-readable prefix of every issued key so
// operators can spot one at a glance in logs / config. sk_live_ is the
// widely-known Stripe-style convention; keeps parity with the rest of
// the ecosystem without inventing a new scheme.
const apiKeyPrefix = "sk_live_"

// apiKeyRawBytes is the number of random bytes packed into a fresh key.
// 32 bytes = 256 bits of entropy = 64 hex characters after the prefix.
const apiKeyRawBytes = 32

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
