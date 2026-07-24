// Per-wallet API key issuance / revocation with signed wallet challenge.
//
// Flow:
//
//  1. Client calls POST /admin/keys/challenge {wallet}.
//     Server generates a 32-byte random nonce, stores {nonce, wallet,
//     expires_at = now+5min}, returns them. This binds the eventual
//     signature to *this* wallet at *this* time.
//
//  2. Client signs message = "cp-issue-key:" || nonce_hex || ":" || wallet
//     with the wallet's ed25519 private key (Casper wallets are ed25519,
//     or a compressed secp256k1 pubkey — we support ed25519 in this
//     hackathon window; secp256k1 is a documented follow-up).
//
//  3. Client calls POST /admin/keys/issue with {wallet, scope, nonce,
//     pubkey_hex, signature_hex}. Server:
//       - looks up the challenge, checks it belongs to this wallet, is
//         unexpired, unconsumed;
//       - checks pubkey_hex hashes/prefixes to the same account_hash as
//         wallet (Casper convention: 01<pubkey> for ed25519);
//       - verifies ed25519(pubkey, message, signature);
//       - checks scope ∈ {submit, verify_only, admin_readonly};
//       - marks the challenge consumed;
//       - only then generates and stores the sk_live_ key.
//
// The plaintext key is returned exactly once in the response body and
// never logged, never persisted.
//
// Both endpoints are admin-gated (adminAuth from PR-3) so an attacker
// who steals a per-wallet user key cannot mint new ones. The signed
// challenge is what upgrades this from "trust whoever asks for a
// wallet=X key" to "trust only the holder of that wallet's private
// key".

package api

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// walletChallengeTTL is how long a challenge remains valid after
// issuance. Short window keeps the replay surface tight; 5 minutes is
// enough for a human to sign in a wallet UI. Tune only after threat-
// modelling the trade-off; don't bump for "convenience".
const walletChallengeTTL = 5 * time.Minute

// walletChallengeMessagePrefix is domain-separation for what the
// wallet signs. Prevents a signature crafted for a different CP
// protocol message (e.g. a proof submission) from being replayed as
// a key-issuance signature.
const walletChallengeMessagePrefix = "cp-issue-key:"

// casperEd25519WalletPrefix is the byte Casper prepends to a raw
// ed25519 pubkey to form its 33-byte account key: 0x01<pubkey_32>.
// A wallet address is then the hex encoding of that 33-byte sequence.
// We compare wallet against "01" + hex(pubkey) case-insensitively.
const casperEd25519WalletPrefix = "01"

type challengeRequest struct {
	Wallet string `json:"wallet"`
}

type challengeResponse struct {
	Nonce     string `json:"nonce"`      // hex, 64 chars (32 random bytes)
	Wallet    string `json:"wallet"`
	Message   string `json:"message"`    // exact bytes the client should sign
	ExpiresAt int64  `json:"expires_at"` // unix seconds
	TTLSecs   int64  `json:"ttl_secs"`
}

type issueKeyRequest struct {
	Wallet       string `json:"wallet"`
	Scope        string `json:"scope"`
	Nonce        string `json:"nonce"`         // from /admin/keys/challenge
	PubkeyHex    string `json:"pubkey_hex"`    // 32-byte ed25519 pubkey, hex
	SignatureHex string `json:"signature_hex"` // 64-byte ed25519 signature, hex
}

type issueKeyResponse struct {
	ID        string `json:"id"`
	APIKey    string `json:"api_key"` // plaintext — returned ONCE, never again
	Wallet    string `json:"wallet"`
	Scope     string `json:"scope"`
	CreatedAt int64  `json:"created_at"`
	Notice    string `json:"notice"`
}

type revokeKeyRequest struct {
	ID string `json:"id"`
}

// newRandomID returns a 128-bit hex id for a key row.
func newRandomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("new id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// newRandomNonce returns a 256-bit hex nonce for the wallet challenge.
func newRandomNonce() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("new nonce: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// challengeMessage returns the exact bytes the wallet is expected to
// sign for issuance. Kept as a helper so the same construction is used
// on issue verification and can be quoted verbatim in the challenge
// response for the client.
func challengeMessage(nonceHex, wallet string) string {
	return walletChallengeMessagePrefix + nonceHex + ":" + strings.ToLower(wallet)
}

// adminIssueChallenge handles POST /admin/keys/challenge.
//
// Request:   {"wallet": "<casper address>"}
// Response:  {"nonce":"<hex>", "wallet":..., "message":"...", "expires_at":..., "ttl_secs":300}
//
// The client signs `message` with the wallet's ed25519 key and returns
// the signature to POST /admin/keys/issue along with the nonce.
func (s *Server) adminIssueChallenge(w http.ResponseWriter, r *http.Request) {
	if s.keyStore() == nil {
		s.jsonError(w, "api key store unavailable: DATABASE_URL not configured", http.StatusServiceUnavailable)
		return
	}
	var req challengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Wallet == "" {
		s.jsonError(w, "wallet is required", http.StatusBadRequest)
		return
	}

	nonce, err := newRandomNonce()
	if err != nil {
		s.log.Error("nonce generation failed", "err", err)
		s.jsonError(w, "internal error generating nonce", http.StatusInternalServerError)
		return
	}
	now := time.Now()
	rec := storeWalletChallengeRecord{
		Nonce:     nonce,
		Wallet:    strings.ToLower(req.Wallet),
		CreatedAt: now.Unix(),
		ExpiresAt: now.Add(walletChallengeTTL).Unix(),
	}
	if err := s.insertWalletChallenge(r.Context(), &rec); err != nil {
		s.log.Error("challenge insert failed", "err", err, "wallet", req.Wallet)
		s.jsonError(w, "failed to persist challenge", http.StatusInternalServerError)
		return
	}
	msg := challengeMessage(nonce, req.Wallet)
	s.log.Info("wallet challenge issued", "wallet", req.Wallet, "expires_at", rec.ExpiresAt)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(challengeResponse{
		Nonce:     nonce,
		Wallet:    strings.ToLower(req.Wallet),
		Message:   msg,
		ExpiresAt: rec.ExpiresAt,
		TTLSecs:   int64(walletChallengeTTL / time.Second),
	})
}

// adminIssueKey handles POST /admin/keys/issue.
//
// See file-level doc comment for the full challenge/signature flow.
// This handler returns the plaintext api_key exactly once; if the
// caller loses it they must issue a new one, there is no recovery.
func (s *Server) adminIssueKey(w http.ResponseWriter, r *http.Request) {
	if s.keyStore() == nil {
		s.jsonError(w, "api key store unavailable: DATABASE_URL not configured", http.StatusServiceUnavailable)
		return
	}

	var req issueKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// --- Field presence ---
	if req.Wallet == "" {
		s.jsonError(w, "wallet is required", http.StatusBadRequest)
		return
	}
	if req.Scope == "" {
		s.jsonError(w, "scope is required (one of: "+strings.Join(ValidScopes, ", ")+")", http.StatusBadRequest)
		return
	}
	if !isValidScope(req.Scope) {
		s.jsonError(w, "invalid scope "+req.Scope+"; must be one of: "+strings.Join(ValidScopes, ", "), http.StatusBadRequest)
		return
	}
	if req.Nonce == "" || req.PubkeyHex == "" || req.SignatureHex == "" {
		s.jsonError(w, "nonce, pubkey_hex, and signature_hex are all required", http.StatusBadRequest)
		return
	}

	// --- Challenge lookup ---
	chal, err := s.lookupWalletChallenge(r.Context(), req.Nonce)
	if err != nil {
		s.log.Warn("challenge lookup failed", "nonce", req.Nonce, "err", err)
		s.jsonError(w, "unknown nonce", http.StatusUnauthorized)
		return
	}
	now := time.Now().Unix()
	if chal.ConsumedAt != 0 {
		s.jsonError(w, "nonce already consumed", http.StatusUnauthorized)
		return
	}
	if chal.ExpiresAt < now {
		s.jsonError(w, "nonce expired", http.StatusUnauthorized)
		return
	}
	if !strings.EqualFold(chal.Wallet, req.Wallet) {
		s.jsonError(w, "nonce was issued for a different wallet", http.StatusUnauthorized)
		return
	}

	// --- Pubkey binds to wallet ---
	// Casper ed25519 wallet = "01" || hex(pubkey_32). Compare
	// case-insensitively; the wallet the client sent must equal the
	// prefix-tagged hex of the pubkey they claim owns it.
	pubkey, err := hex.DecodeString(req.PubkeyHex)
	if err != nil || len(pubkey) != ed25519.PublicKeySize {
		s.jsonError(w, "pubkey_hex must be 32-byte hex", http.StatusBadRequest)
		return
	}
	expectedWallet := casperEd25519WalletPrefix + strings.ToLower(req.PubkeyHex)
	if !strings.EqualFold(expectedWallet, req.Wallet) {
		s.jsonError(w, "pubkey does not match wallet (expected 01<pubkey_hex>)", http.StatusUnauthorized)
		return
	}

	// --- Signature verify ---
	sig, err := hex.DecodeString(req.SignatureHex)
	if err != nil || len(sig) != ed25519.SignatureSize {
		s.jsonError(w, "signature_hex must be 64-byte hex", http.StatusBadRequest)
		return
	}
	msg := challengeMessage(req.Nonce, req.Wallet)
	if !ed25519.Verify(ed25519.PublicKey(pubkey), []byte(msg), sig) {
		s.log.Warn("wallet signature verification failed", "wallet", req.Wallet, "nonce", req.Nonce)
		s.jsonError(w, "signature does not verify against pubkey", http.StatusUnauthorized)
		return
	}

	// --- Consume nonce BEFORE minting the key ---
	// If we minted first and consumption failed, an attacker who
	// races two requests with the same nonce could get two keys.
	if err := s.markWalletChallengeConsumed(r.Context(), req.Nonce, now); err != nil {
		s.log.Error("challenge consume failed", "err", err, "nonce", req.Nonce)
		s.jsonError(w, "failed to consume challenge", http.StatusInternalServerError)
		return
	}

	// --- Mint the key ---
	plaintext, err := generateAPIKey()
	if err != nil {
		s.log.Error("api key generation failed", "err", err)
		s.jsonError(w, "internal error generating key", http.StatusInternalServerError)
		return
	}
	id, err := newRandomID()
	if err != nil {
		s.log.Error("api key id generation failed", "err", err)
		s.jsonError(w, "internal error generating key id", http.StatusInternalServerError)
		return
	}
	rec := storeAPIKeyRecord{
		ID:        id,
		KeyHash:   hashAPIKey(plaintext),
		Wallet:    strings.ToLower(req.Wallet),
		Scope:     req.Scope,
		CreatedAt: now,
	}
	if err := s.insertAPIKey(r.Context(), &rec); err != nil {
		s.log.Error("api key insert failed", "err", err, "wallet", req.Wallet)
		s.jsonError(w, "failed to persist key", http.StatusInternalServerError)
		return
	}
	// Redact plaintext everywhere except the response body.
	s.log.Info("api key issued",
		"id", id,
		"wallet", req.Wallet,
		"scope", req.Scope)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(issueKeyResponse{
		ID:        id,
		APIKey:    plaintext,
		Wallet:    strings.ToLower(req.Wallet),
		Scope:     req.Scope,
		CreatedAt: now,
		Notice:    "This api_key value is returned exactly once. Store it now — it is unrecoverable.",
	})
}

// adminRevokeKey handles POST /admin/keys/revoke.
func (s *Server) adminRevokeKey(w http.ResponseWriter, r *http.Request) {
	if s.keyStore() == nil {
		s.jsonError(w, "api key store unavailable: DATABASE_URL not configured", http.StatusServiceUnavailable)
		return
	}
	var req revokeKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		s.jsonError(w, "id is required", http.StatusBadRequest)
		return
	}
	if err := s.revokeAPIKey(r.Context(), req.ID, time.Now().Unix()); err != nil {
		// Distinguish "not found / already revoked" (404) from real DB errors.
		s.log.Warn("api key revoke failed", "id", req.ID, "err", err)
		s.jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":      req.ID,
		"revoked": true,
	})
}
