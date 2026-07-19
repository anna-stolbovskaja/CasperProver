// Per-wallet API key issuance / revocation.
//
// The plaintext key exists only inside the JSON body returned by
// POST /admin/keys/issue \u2014 it's never logged, never persisted, and
// never retrievable afterwards. Only sha256(key) is stored.
//
// Every endpoint here is admin-gated (see authMiddleware) so an
// attacker who steals a per-wallet user key still can't mint new ones
// or revoke arbitrary keys.

package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type issueKeyRequest struct {
	Wallet string `json:"wallet"`
	Scope  string `json:"scope,omitempty"` // optional; defaults to "user"
}

type issueKeyResponse struct {
	ID        string `json:"id"`
	APIKey    string `json:"api_key"` // plaintext \u2014 returned ONCE, never again
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

// adminIssueKey handles POST /admin/keys/issue.
//
// Request:   {"wallet": "<casper address>", "scope": "user"}
// Response:  {"id":..., "api_key":"sk_live_...", "wallet":..., "scope":...,
//              "created_at":..., "notice":"..."}
//
// The plaintext api_key is returned exactly once here. If the caller
// loses it they must issue a new one; there is no recovery path.
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
	if req.Wallet == "" {
		s.jsonError(w, "wallet is required", http.StatusBadRequest)
		return
	}
	scope := req.Scope
	if scope == "" {
		scope = "user"
	}

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
	now := time.Now().Unix()

	rec := storeAPIKeyRecord{
		ID:        id,
		KeyHash:   hashAPIKey(plaintext),
		Wallet:    req.Wallet,
		Scope:     scope,
		CreatedAt: now,
	}
	if err := s.insertAPIKey(r.Context(), &rec); err != nil {
		s.log.Error("api key insert failed", "err", err, "wallet", req.Wallet)
		s.jsonError(w, "failed to persist key", http.StatusInternalServerError)
		return
	}
	// Redact plaintext everywhere except the response body itself.
	s.log.Info("api key issued",
		"id", id,
		"wallet", req.Wallet,
		"scope", scope)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(issueKeyResponse{
		ID:        id,
		APIKey:    plaintext,
		Wallet:    req.Wallet,
		Scope:     scope,
		CreatedAt: now,
		Notice:    "This api_key value is returned exactly once. Store it now \u2014 it is unrecoverable.",
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
		// Distinguish \"not found / already revoked\" (404) from real DB errors.
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
