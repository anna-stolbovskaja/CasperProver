package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/store"
)

// -------- apikey.go: generateAPIKey / hashAPIKey --------

func TestGenerateAPIKey_FormatAndEntropy(t *testing.T) {
	k1, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey: %v", err)
	}
	if !strings.HasPrefix(k1, apiKeyPrefix) {
		t.Fatalf("expected key to start with %q, got %q", apiKeyPrefix, k1)
	}
	// prefix + 64 hex chars (32 bytes)
	wantLen := len(apiKeyPrefix) + apiKeyRawBytes*2
	if got := len(k1); got != wantLen {
		t.Fatalf("expected len %d, got %d (key=%q)", wantLen, got, k1)
	}
	// Two consecutive calls must never collide.
	k2, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey 2: %v", err)
	}
	if k1 == k2 {
		t.Fatalf("two random keys must differ, both were %q", k1)
	}
}

func TestHashAPIKey_DeterministicAndDifferent(t *testing.T) {
	h1 := hashAPIKey("sk_live_deadbeef")
	h2 := hashAPIKey("sk_live_deadbeef")
	if h1 != h2 {
		t.Fatalf("hashAPIKey must be deterministic (%q vs %q)", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("sha256 hex must be 64 chars, got %d", len(h1))
	}
	h3 := hashAPIKey("sk_live_deadbeeg")
	if h1 == h3 {
		t.Fatalf("hashAPIKey must differ across inputs")
	}
	// Plaintext must NOT appear anywhere in the hash.
	if strings.Contains(h1, "sk_live") || strings.Contains(h1, "deadbeef") {
		t.Fatalf("hash leaks plaintext: %q", h1)
	}
}

// -------- fake apiKeyStore --------

type fakeKeyStore struct {
	mu      sync.Mutex
	byID    map[string]*store.APIKeyRecord
	byHash  map[string]*store.APIKeyRecord
	insertErr error
}

func newFakeKeyStore() *fakeKeyStore {
	return &fakeKeyStore{
		byID:   make(map[string]*store.APIKeyRecord),
		byHash: make(map[string]*store.APIKeyRecord),
	}
}

func (f *fakeKeyStore) InsertAPIKey(ctx context.Context, r *store.APIKeyRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.insertErr != nil {
		return f.insertErr
	}
	if _, ok := f.byHash[r.KeyHash]; ok {
		return errors.New("collision")
	}
	cp := *r
	f.byID[r.ID] = &cp
	f.byHash[r.KeyHash] = &cp
	return nil
}

func (f *fakeKeyStore) LookupAPIKeyByHash(ctx context.Context, h string) (*store.APIKeyRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.byHash[h]
	if !ok {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (f *fakeKeyStore) RevokeAPIKey(ctx context.Context, id string, at int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.byID[id]
	if !ok || r.Revoked {
		return errors.New("no unrevoked row")
	}
	r.Revoked = true
	r.RevokedAt = at
	return nil
}

// -------- POST /admin/keys/issue --------

func TestAdminIssueKey_ReturnsPlaintextOnceAndStoresOnlyHash(t *testing.T) {
	s := newTestServer("")
	fake := newFakeKeyStore()
	s.keys = fake

	body := `{"wallet":"account-hash-abc","scope":"user"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()

	s.adminIssueKey(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp issueKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON: %v (raw=%s)", err, rec.Body.String())
	}
	if !strings.HasPrefix(resp.APIKey, apiKeyPrefix) {
		t.Fatalf("expected plaintext api_key in response, got %q", resp.APIKey)
	}
	if resp.Wallet != "account-hash-abc" || resp.Scope != "user" || resp.ID == "" {
		t.Fatalf("unexpected response fields: %+v", resp)
	}

	// Verify only the hash landed in the store, never plaintext.
	stored, err := fake.LookupAPIKeyByHash(context.Background(), hashAPIKey(resp.APIKey))
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if stored == nil {
		t.Fatalf("hash of issued key not found in store")
	}
	if stored.KeyHash == resp.APIKey {
		t.Fatalf("store MUST NOT hold plaintext, but KeyHash==APIKey")
	}
	if strings.Contains(stored.KeyHash, "sk_live") {
		t.Fatalf("stored hash appears to contain plaintext: %q", stored.KeyHash)
	}
}

func TestAdminIssueKey_RejectsMissingWallet(t *testing.T) {
	s := newTestServer("")
	s.keys = newFakeKeyStore()

	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	s.adminIssueKey(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without wallet, got %d", rec.Code)
	}
}

func TestAdminIssueKey_UnavailableWithoutStore(t *testing.T) {
	s := newTestServer("")
	// keys and db both nil \u2014 no key store available
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", bytes.NewReader([]byte(`{"wallet":"w"}`)))
	rec := httptest.NewRecorder()
	s.adminIssueKey(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without store, got %d", rec.Code)
	}
}

// -------- Hash-verification of issued key --------

func TestIssuedKey_VerifiesViaHashOnRequest(t *testing.T) {
	// Simulate the auth flow: issue a key, then the middleware
	// checks an incoming request by hashing X-API-Key and looking
	// it up in the store. The stored hash must match.
	s := newTestServer("")
	fake := newFakeKeyStore()
	s.keys = fake

	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue",
		bytes.NewReader([]byte(`{"wallet":"w1"}`)))
	rec := httptest.NewRecorder()
	s.adminIssueKey(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("issue failed: %d", rec.Code)
	}
	var resp issueKeyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	// Verify: hash the presented plaintext, look it up.
	found, err := s.lookupAPIKey(context.Background(), hashAPIKey(resp.APIKey))
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if found == nil {
		t.Fatalf("issued key not found on hash lookup")
	}
	if found.Wallet != "w1" || found.Revoked {
		t.Fatalf("unexpected stored record: %+v", found)
	}

	// A different plaintext must NOT match.
	other, err := s.lookupAPIKey(context.Background(), hashAPIKey("sk_live_wrong"))
	if err != nil {
		t.Fatalf("lookup2: %v", err)
	}
	if other != nil {
		t.Fatalf("wrong plaintext must not resolve, got %+v", other)
	}
}

// -------- POST /admin/keys/revoke --------

func TestAdminRevokeKey_HappyPath(t *testing.T) {
	s := newTestServer("")
	fake := newFakeKeyStore()
	s.keys = fake

	// Issue first.
	req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue",
		bytes.NewReader([]byte(`{"wallet":"w2"}`)))
	rec := httptest.NewRecorder()
	s.adminIssueKey(rec, req)
	var issued issueKeyResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &issued)

	// Revoke.
	body := `{"id":"` + issued.ID + `"}`
	req = httptest.NewRequest(http.MethodPost, "/admin/keys/revoke",
		bytes.NewReader([]byte(body)))
	rec = httptest.NewRecorder()
	s.adminRevokeKey(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// After revoke: lookup must still find the row but with Revoked=true.
	found, _ := s.lookupAPIKey(context.Background(), hashAPIKey(issued.APIKey))
	if found == nil || !found.Revoked {
		t.Fatalf("revoke did not flip Revoked flag: %+v", found)
	}

	// Revoke again \u2192 404.
	req = httptest.NewRequest(http.MethodPost, "/admin/keys/revoke",
		bytes.NewReader([]byte(body)))
	rec = httptest.NewRecorder()
	s.adminRevokeKey(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("double-revoke expected 404, got %d", rec.Code)
	}
}

// -------- Admin gate: /admin/keys/issue is ADMIN-gated --------

func TestAdminKeysIssue_IsAdminGated(t *testing.T) {
	s := newTestServer("secret123")
	s.keys = newFakeKeyStore()
	handler := s.authMiddleware(http.HandlerFunc(s.adminIssueKey))

	body := []byte(`{"wallet":"w3"}`)

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"missing header \u2192 401", "", http.StatusUnauthorized},
		{"wrong header \u2192 403", "wrong", http.StatusForbidden},
		{"correct header \u2192 201", "secret123", http.StatusCreated},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/admin/keys/issue", bytes.NewReader(body))
			if c.header != "" {
				req.Header.Set("X-API-Key", c.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("%s: expected %d, got %d body=%s", c.name, c.want, rec.Code, rec.Body.String())
			}
		})
	}
}
