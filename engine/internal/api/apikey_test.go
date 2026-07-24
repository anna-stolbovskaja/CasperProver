package api

// apikey_test.go exercises the per-wallet API-key layer added in PR-4
// of CP batch-3 revised:
//
//   - generateAPIKey format + entropy
//   - hashAPIKey determinism + no plaintext leak
//   - POST /admin/keys/challenge: happy path, missing wallet, no store
//   - POST /admin/keys/issue signed flow: happy path, replay reject,
//     expired nonce, wrong wallet, tampered signature, invalid scope,
//     mismatched pubkey/wallet, no store
//   - POST /admin/keys/revoke: happy path + double-revoke -> 404
//   - writeAuth accepts a valid sk_live_ key, rejects a revoked one,
//     rejects an unknown one
//   - requireScope: submit-scope hits /proofs, verify_only cannot;
//     shared API_KEY still hits everything
//
// All tests use fakeAPIKeyStore so no Postgres is required.

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/store"
)

// fakeAPIKeyStore is an in-memory apiKeyStore for tests.
type fakeAPIKeyStore struct {
	mu         sync.Mutex
	keys       map[string]*store.APIKeyRecord      // keyed by KeyHash
	byID       map[string]*store.APIKeyRecord      // keyed by ID
	challenges map[string]*store.WalletChallengeRecord
}

func newFakeAPIKeyStore() *fakeAPIKeyStore {
	return &fakeAPIKeyStore{
		keys:       map[string]*store.APIKeyRecord{},
		byID:       map[string]*store.APIKeyRecord{},
		challenges: map[string]*store.WalletChallengeRecord{},
	}
}

func (f *fakeAPIKeyStore) InsertAPIKey(_ context.Context, rec *store.APIKeyRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.keys[rec.KeyHash]; ok {
		return sql.ErrNoRows // stand-in for uniqueness violation
	}
	cp := *rec
	f.keys[rec.KeyHash] = &cp
	f.byID[rec.ID] = &cp
	return nil
}

func (f *fakeAPIKeyStore) LookupAPIKeyByHash(_ context.Context, h string) (*store.APIKeyRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.keys[h]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *r
	return &cp, nil
}

func (f *fakeAPIKeyStore) RevokeAPIKey(_ context.Context, id string, revokedAt int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.byID[id]
	if !ok || r.Revoked {
		return sql.ErrNoRows
	}
	r.Revoked = true
	r.RevokedAt = revokedAt
	return nil
}

func (f *fakeAPIKeyStore) InsertWalletChallenge(_ context.Context, rec *store.WalletChallengeRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.challenges[rec.Nonce]; ok {
		return sql.ErrNoRows
	}
	cp := *rec
	f.challenges[rec.Nonce] = &cp
	return nil
}

func (f *fakeAPIKeyStore) LookupWalletChallenge(_ context.Context, nonce string) (*store.WalletChallengeRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.challenges[nonce]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *r
	return &cp, nil
}

func (f *fakeAPIKeyStore) MarkWalletChallengeConsumed(_ context.Context, nonce string, at int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.challenges[nonce]
	if !ok || r.ConsumedAt != 0 {
		return sql.ErrNoRows
	}
	r.ConsumedAt = at
	return nil
}

// newKeyTestServer wires a Server with an in-memory apiKeyStore.
// Admin key is set to make sure the admin gate is enforced by buildMux
// but each test presents X-Admin-API-Key when hitting admin routes.
func newKeyTestServer(t *testing.T) (*Server, *fakeAPIKeyStore) {
	t.Helper()
	s := newTestServer("")
	s.adminKey = "admin-secret"
	fs := newFakeAPIKeyStore()
	s.keys = fs
	return s, fs
}

func testMux(s *Server) http.Handler { return s.buildMux() }

// signedIssueBody builds a valid /admin/keys/issue request body from
// a signing keypair and a challenge nonce.
func signedIssueBody(t *testing.T, pub ed25519.PublicKey, priv ed25519.PrivateKey, nonce, scope string) (string, string) {
	t.Helper()
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)
	msg := challengeMessage(nonce, wallet)
	sig := ed25519.Sign(priv, []byte(msg))
	body := map[string]string{
		"wallet":        wallet,
		"scope":         scope,
		"nonce":         nonce,
		"pubkey_hex":    hex.EncodeToString(pub),
		"signature_hex": hex.EncodeToString(sig),
	}
	b, _ := json.Marshal(body)
	return wallet, string(b)
}

func obtainChallenge(t *testing.T, s *Server, wallet string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/challenge",
		strings.NewReader(`{"wallet":"`+wallet+`"}`))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("challenge: got %d, want 200. body=%s", rec.Code, rec.Body.String())
	}
	var resp challengeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode challenge resp: %v (body=%s)", err, rec.Body.String())
	}
	if resp.Nonce == "" || len(resp.Nonce) != 64 {
		t.Fatalf("bad nonce %q", resp.Nonce)
	}
	if !strings.EqualFold(resp.Wallet, wallet) {
		t.Fatalf("bad wallet echo %q vs %q", resp.Wallet, wallet)
	}
	return resp.Nonce
}

func TestGenerateAPIKey_FormatAndEntropy(t *testing.T) {
	k, err := generateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(k, apiKeyPrefix) {
		t.Fatalf("prefix missing: %q", k)
	}
	if len(k) != len(apiKeyPrefix)+2*apiKeyRawBytes {
		t.Fatalf("bad length: %d", len(k))
	}

	// A quick birthday check: 100 keys should never collide.
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		k, err := generateAPIKey()
		if err != nil {
			t.Fatal(err)
		}
		if seen[k] {
			t.Fatalf("duplicate key generated on iter %d", i)
		}
		seen[k] = true
	}
}

func TestHashAPIKey_DeterministicNoPlaintext(t *testing.T) {
	k := "sk_live_" + strings.Repeat("a", 64)
	h1 := hashAPIKey(k)
	h2 := hashAPIKey(k)
	if h1 != h2 {
		t.Fatalf("hash not deterministic: %q vs %q", h1, h2)
	}
	if strings.Contains(h1, "sk_live") || strings.Contains(h1, "aaaaaaaa") {
		t.Fatalf("hash appears to leak plaintext: %q", h1)
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(h1))
	}
}

func TestChallenge_HappyPath(t *testing.T) {
	s, _ := newKeyTestServer(t)
	pub, _, _ := ed25519.GenerateKey(nil)
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)
	nonce := obtainChallenge(t, s, wallet)
	if nonce == "" {
		t.Fatal("empty nonce")
	}
}

func TestChallenge_MissingWallet(t *testing.T) {
	s, _ := newKeyTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/challenge", strings.NewReader(`{}`))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestChallenge_NoStore(t *testing.T) {
	s := newTestServer("")
	s.adminKey = "admin-secret"
	// s.keys stays nil; s.db is nil in tests -> keyStore() returns nil.
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/challenge",
		strings.NewReader(`{"wallet":"01aabb"}`))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueKey_HappyPath(t *testing.T) {
	s, _ := newKeyTestServer(t)
	pub, priv, _ := ed25519.GenerateKey(nil)
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)
	nonce := obtainChallenge(t, s, wallet)

	_, body := signedIssueBody(t, pub, priv, nonce, ScopeSubmit)
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", strings.NewReader(body))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp issueKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resp.APIKey, apiKeyPrefix) {
		t.Fatalf("bad key format: %q", resp.APIKey)
	}
	if resp.Scope != ScopeSubmit {
		t.Fatalf("bad scope: %q", resp.Scope)
	}
	if !strings.EqualFold(resp.Wallet, wallet) {
		t.Fatalf("bad wallet echo: %q vs %q", resp.Wallet, wallet)
	}
}

func TestIssueKey_ReplayNonceRejected(t *testing.T) {
	s, _ := newKeyTestServer(t)
	pub, priv, _ := ed25519.GenerateKey(nil)
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)
	nonce := obtainChallenge(t, s, wallet)

	_, body := signedIssueBody(t, pub, priv, nonce, ScopeSubmit)

	// First call: 201.
	first := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", strings.NewReader(body))
	first.Header.Set("X-Admin-API-Key", "admin-secret")
	first.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec1, first)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first issue: got %d: %s", rec1.Code, rec1.Body.String())
	}

	// Same body again: 401 (nonce consumed).
	second := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", strings.NewReader(body))
	second.Header.Set("X-Admin-API-Key", "admin-secret")
	second.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec2, second)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("replay should be 401, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestIssueKey_ExpiredNonce(t *testing.T) {
	s, fs := newKeyTestServer(t)
	pub, priv, _ := ed25519.GenerateKey(nil)
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)
	nonce := obtainChallenge(t, s, wallet)

	// Manually rewind expiry into the past.
	fs.mu.Lock()
	fs.challenges[nonce].ExpiresAt = time.Now().Add(-time.Minute).Unix()
	fs.mu.Unlock()

	_, body := signedIssueBody(t, pub, priv, nonce, ScopeSubmit)
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", strings.NewReader(body))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired nonce should be 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueKey_WrongWallet(t *testing.T) {
	s, _ := newKeyTestServer(t)
	pub, priv, _ := ed25519.GenerateKey(nil)
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)
	nonce := obtainChallenge(t, s, wallet)

	// Sign with the right key, but claim a different wallet in the body.
	otherPub, _, _ := ed25519.GenerateKey(nil)
	otherWallet := casperEd25519WalletPrefix + hex.EncodeToString(otherPub)
	msg := challengeMessage(nonce, otherWallet)
	sig := ed25519.Sign(priv, []byte(msg))
	body := map[string]string{
		"wallet":        otherWallet,
		"scope":         ScopeSubmit,
		"nonce":         nonce,
		"pubkey_hex":    hex.EncodeToString(pub),
		"signature_hex": hex.EncodeToString(sig),
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", strings.NewReader(string(b)))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wallet mismatch should be 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueKey_TamperedSignature(t *testing.T) {
	s, _ := newKeyTestServer(t)
	pub, priv, _ := ed25519.GenerateKey(nil)
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)
	nonce := obtainChallenge(t, s, wallet)

	_, body := signedIssueBody(t, pub, priv, nonce, ScopeSubmit)

	// Flip the last hex nibble of signature_hex.
	var parsed map[string]string
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatal(err)
	}
	sig := parsed["signature_hex"]
	if sig[len(sig)-1] == '0' {
		parsed["signature_hex"] = sig[:len(sig)-1] + "1"
	} else {
		parsed["signature_hex"] = sig[:len(sig)-1] + "0"
	}
	tampered, _ := json.Marshal(parsed)

	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", strings.NewReader(string(tampered)))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("tampered signature should be 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueKey_InvalidScope(t *testing.T) {
	s, _ := newKeyTestServer(t)
	pub, priv, _ := ed25519.GenerateKey(nil)
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)
	nonce := obtainChallenge(t, s, wallet)

	_, body := signedIssueBody(t, pub, priv, nonce, "root") // not in ValidScopes
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", strings.NewReader(body))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid scope should be 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueKey_MismatchedPubkey(t *testing.T) {
	s, _ := newKeyTestServer(t)
	pub, priv, _ := ed25519.GenerateKey(nil)
	// Advertise wallet for pub, but send *someone else's* pubkey_hex.
	otherPub, _, _ := ed25519.GenerateKey(nil)
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)
	nonce := obtainChallenge(t, s, wallet)

	msg := challengeMessage(nonce, wallet)
	sig := ed25519.Sign(priv, []byte(msg))
	body := map[string]string{
		"wallet":        wallet,
		"scope":         ScopeSubmit,
		"nonce":         nonce,
		"pubkey_hex":    hex.EncodeToString(otherPub),
		"signature_hex": hex.EncodeToString(sig),
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", strings.NewReader(string(b)))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("mismatched pubkey should be 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRevokeKey_Lifecycle(t *testing.T) {
	s, _ := newKeyTestServer(t)
	pub, priv, _ := ed25519.GenerateKey(nil)
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)

	// Issue.
	nonce := obtainChallenge(t, s, wallet)
	_, body := signedIssueBody(t, pub, priv, nonce, ScopeSubmit)
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", strings.NewReader(body))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("issue: got %d: %s", rec.Code, rec.Body.String())
	}
	var issued issueKeyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &issued)

	// Revoke.
	revBody, _ := json.Marshal(map[string]string{"id": issued.ID})
	req2 := httptest.NewRequest(http.MethodPost, "/admin/keys/revoke", strings.NewReader(string(revBody)))
	req2.Header.Set("X-Admin-API-Key", "admin-secret")
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("revoke: got %d: %s", rec2.Code, rec2.Body.String())
	}

	// Second revoke: 404.
	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/admin/keys/revoke", strings.NewReader(string(revBody)))
	req3.Header.Set("X-Admin-API-Key", "admin-secret")
	req3.Header.Set("Content-Type", "application/json")
	testMux(s).ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusNotFound {
		t.Fatalf("double revoke: expected 404, got %d: %s", rec3.Code, rec3.Body.String())
	}
}

// --- writeAuth + requireScope acceptance path ---

// issueFreshKey obtains a fresh sk_live_ key via the full challenge/
// issue flow so tests exercise the same path production uses.
func issueFreshKey(t *testing.T, s *Server, scope string) string {
	t.Helper()
	pub, priv, _ := ed25519.GenerateKey(nil)
	wallet := casperEd25519WalletPrefix + hex.EncodeToString(pub)
	nonce := obtainChallenge(t, s, wallet)
	_, body := signedIssueBody(t, pub, priv, nonce, scope)
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", strings.NewReader(body))
	req.Header.Set("X-Admin-API-Key", "admin-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("issue for scope %s: %d %s", scope, rec.Code, rec.Body.String())
	}
	var resp issueKeyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.APIKey
}

func TestWriteAuth_AcceptsPerWalletKey(t *testing.T) {
	s, _ := newKeyTestServer(t)
	key := issueFreshKey(t, s, ScopeSubmit)

	// Hit a submit-scope route. It should pass auth+scope; the handler
	// may 400 on the body but we only care that auth doesn't reject.
	req := httptest.NewRequest(http.MethodPost, "/proofs",
		strings.NewReader(`{"agent":"a","input":"i","output":"o","model":"m"}`))
	req.Header.Set("X-API-Key", key)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
		t.Fatalf("valid sk_live_ key rejected on /proofs: %d %s", rec.Code, rec.Body.String())
	}
}

func TestWriteAuth_RejectsRevokedKey(t *testing.T) {
	s, _ := newKeyTestServer(t)
	key := issueFreshKey(t, s, ScopeSubmit)

	// Look up the row via the store to grab the ID for revoke.
	rec, err := s.keyStore().LookupAPIKeyByHash(context.Background(), hashAPIKey(key))
	if err != nil {
		t.Fatal(err)
	}
	// Revoke via admin route.
	body, _ := json.Marshal(map[string]string{"id": rec.ID})
	rq := httptest.NewRequest(http.MethodPost, "/admin/keys/revoke", strings.NewReader(string(body)))
	rq.Header.Set("X-Admin-API-Key", "admin-secret")
	rq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	testMux(s).ServeHTTP(rr, rq)
	if rr.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", rr.Code, rr.Body.String())
	}

	// Now try to use it: should 403 (present but rejected).
	req := httptest.NewRequest(http.MethodPost, "/proofs",
		strings.NewReader(`{"agent":"a","input":"i","output":"o","model":"m"}`))
	req.Header.Set("X-API-Key", key)
	req.Header.Set("Content-Type", "application/json")
	rrr := httptest.NewRecorder()
	testMux(s).ServeHTTP(rrr, req)
	if rrr.Code != http.StatusForbidden {
		t.Fatalf("revoked key should be 403, got %d: %s", rrr.Code, rrr.Body.String())
	}
}

func TestWriteAuth_UnknownKey(t *testing.T) {
	s, _ := newKeyTestServer(t)
	fake := apiKeyPrefix + strings.Repeat("f", 64)
	req := httptest.NewRequest(http.MethodPost, "/proofs",
		strings.NewReader(`{"agent":"a","input":"i","output":"o","model":"m"}`))
	req.Header.Set("X-API-Key", fake)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("unknown sk_live_ key should be 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- Scope enforcement ---

func TestRequireScope_VerifyOnlyCannotSubmit(t *testing.T) {
	s, _ := newKeyTestServer(t)
	verifyKey := issueFreshKey(t, s, ScopeVerifyOnly)

	// /proofs requires submit scope.
	req := httptest.NewRequest(http.MethodPost, "/proofs",
		strings.NewReader(`{"agent":"a","input":"i","output":"o","model":"m"}`))
	req.Header.Set("X-API-Key", verifyKey)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("verify_only on /proofs should be 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequireScope_VerifyOnlyCanVerify(t *testing.T) {
	s, _ := newKeyTestServer(t)
	verifyKey := issueFreshKey(t, s, ScopeVerifyOnly)

	// /verify permits both submit and verify_only.
	req := httptest.NewRequest(http.MethodPost, "/verify",
		strings.NewReader(`{"agent":"a","proof_id":"x"}`))
	req.Header.Set("X-API-Key", verifyKey)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code == http.StatusForbidden || rec.Code == http.StatusUnauthorized {
		t.Fatalf("verify_only on /verify should pass auth, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequireScope_AdminReadonlyCannotSubmit(t *testing.T) {
	s, _ := newKeyTestServer(t)
	roKey := issueFreshKey(t, s, ScopeAdminReadonly)

	req := httptest.NewRequest(http.MethodPost, "/proofs",
		strings.NewReader(`{"agent":"a","input":"i","output":"o","model":"m"}`))
	req.Header.Set("X-API-Key", roKey)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin_readonly on /proofs should be 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequireScope_SharedApiKeyIsSuperScope(t *testing.T) {
	// Shared key configured; /proofs should still pass with it even
	// though per-wallet keys would need submit scope.
	s := newTestServer("shared-secret")
	s.adminKey = "admin-secret"
	s.keys = newFakeAPIKeyStore()

	req := httptest.NewRequest(http.MethodPost, "/proofs",
		strings.NewReader(`{"agent":"a","input":"i","output":"o","model":"m"}`))
	req.Header.Set("X-API-Key", "shared-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(s).ServeHTTP(rec, req)
	if rec.Code == http.StatusForbidden || rec.Code == http.StatusUnauthorized {
		t.Fatalf("shared API_KEY should be a super-scope, got %d: %s", rec.Code, rec.Body.String())
	}
}
