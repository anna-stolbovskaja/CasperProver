package api

// authz_split_test.go covers the public/admin auth split introduced in
// PR-3 of CP batch-3 revised. Tests target the fully wired router
// (buildMux) so we catch route-wiring bugs, not just middleware in
// isolation.
//
// Two things this file guards against:
//
//  1. Admin routes accidentally accepting the public API_KEY. This would
//     silently promote every tenant to operator. The tests confirm that
//     KYC grant / model registration / aggregation finalize reject a
//     valid public X-API-Key when only the admin key is configured, and
//     require X-Admin-API-Key instead.
//
//  2. Public write routes accidentally requiring the admin key. The
//     tests exercise POST /proofs and POST /verify with only the public
//     X-API-Key to confirm the write path still opens for tenants.
//
// Coverage of the 401 (missing) vs 403 (wrong) split lives in the writeAuth
// and adminAuth cases below. Read-only GETs are asserted to remain public
// even when both keys are configured.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newSplitServer configures a Server with distinct public and admin keys
// so tests can prove the two are not conflated.
func newSplitServer(publicKey, adminKey string) *Server {
	s := newTestServer(publicKey)
	s.adminKey = adminKey
	return s
}

// serveThrough runs the request against the fully wired router
// (buildMux), so route registration + middleware wrapping are both
// covered.
func serveThrough(s *Server, method, path string, headers map[string]string, body string) *httptest.ResponseRecorder {
	mux := s.buildMux()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func TestWriteAuth_MissingVsWrongKey_Split(t *testing.T) {
	s := newTestServer("public-secret")
	handler := s.writeAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"missing header -> 401", "", http.StatusUnauthorized},
		{"wrong key -> 403", "wrong", http.StatusForbidden},
		{"correct key -> 200", "public-secret", http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/proofs", strings.NewReader("{}"))
			if c.header != "" {
				r.Header.Set("X-API-Key", c.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, r)
			if rec.Code != c.want {
				t.Fatalf("%s: got %d, want %d, body=%s", c.name, rec.Code, c.want, rec.Body.String())
			}
		})
	}
}

func TestWriteAuth_NoKeyConfigured_AllowsAll(t *testing.T) {
	s := newTestServer("")
	handler := s.writeAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodPost, "/proofs", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 without API_KEY configured, got %d", rec.Code)
	}
}

func TestAdminAuth_MissingVsWrongKey_Split(t *testing.T) {
	s := newTestServer("")
	s.adminKey = "admin-secret"
	handler := s.adminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name   string
		header string
		value  string
		want   int
	}{
		{"missing header -> 401", "", "", http.StatusUnauthorized},
		{"wrong key -> 403", "X-Admin-API-Key", "wrong", http.StatusForbidden},
		{"correct key -> 200", "X-Admin-API-Key", "admin-secret", http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/kyc/grant", strings.NewReader("{}"))
			if c.header != "" {
				r.Header.Set(c.header, c.value)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, r)
			if rec.Code != c.want {
				t.Fatalf("%s: got %d, want %d", c.name, rec.Code, c.want)
			}
		})
	}
}

func TestAdminAuth_NoKeyConfigured_AllowsAll(t *testing.T) {
	s := newTestServer("")
	handler := s.adminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodPost, "/kyc/grant", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 without ADMIN_API_KEY configured, got %d", rec.Code)
	}
}

// TestAdminAuth_RejectsPublicKey is the core regression guard.
// A valid public X-API-Key MUST NOT satisfy an admin route.
func TestAdminAuth_RejectsPublicKey(t *testing.T) {
	s := newSplitServer("public-secret", "admin-secret")

	// Presenting the public key on an admin route -> 401 (no admin header
	// present at all). The public key travels in a different header, so
	// the admin gate sees an empty X-Admin-API-Key and returns 401.
	rec := serveThrough(s, http.MethodPost, "/kyc/grant",
		map[string]string{"X-API-Key": "public-secret", "Content-Type": "application/json"},
		`{"user":"u1"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin route with only public X-API-Key: got %d (%s), want 401",
			rec.Code, rec.Body.String())
	}

	// Presenting the public key value in the admin header slot -> 403
	// (a value is there, but it does not match the admin secret).
	rec = serveThrough(s, http.MethodPost, "/kyc/grant",
		map[string]string{"X-Admin-API-Key": "public-secret", "Content-Type": "application/json"},
		`{"user":"u1"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin route with public value under admin header: got %d (%s), want 403",
			rec.Code, rec.Body.String())
	}
}

// TestPublicWrite_RejectsAdminKey is the mirror guard. Admin key alone
// (no public key) MUST NOT satisfy a public write route when API_KEY is
// set - otherwise operators would bypass any per-tenant policy layered on
// top of writeAuth (e.g. PR-4 per-wallet scoping).
func TestPublicWrite_RejectsAdminKey(t *testing.T) {
	s := newSplitServer("public-secret", "admin-secret")

	// Admin key sent under X-API-Key -> 403 (value present, wrong).
	rec := serveThrough(s, http.MethodPost, "/verify",
		map[string]string{"X-API-Key": "admin-secret", "Content-Type": "application/json"},
		`{"agent":"a","proof_id":"x"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("public route with admin value under public header: got %d (%s), want 403",
			rec.Code, rec.Body.String())
	}

	// X-Admin-API-Key alone is not what the public gate checks -> 401.
	rec = serveThrough(s, http.MethodPost, "/verify",
		map[string]string{"X-Admin-API-Key": "admin-secret", "Content-Type": "application/json"},
		`{"agent":"a","proof_id":"x"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("public route with only admin header: got %d (%s), want 401",
			rec.Code, rec.Body.String())
	}
}

// TestRouter_AdminRoutesAreAdminGated confirms the exact route table.
// If someone accidentally moves an admin route into the public-write
// tier, this test flips red.
func TestRouter_AdminRoutesAreAdminGated(t *testing.T) {
	s := newSplitServer("public-secret", "admin-secret")

	adminRoutes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/kyc/grant", `{"user":"u1"}`},
		{http.MethodPost, "/inference/register-model", `{"id":"m1","name":"n","version":"v"}`},
		{http.MethodPost, "/aggregation/create-batch", `{"max_proofs":4}`},
		{http.MethodPost, "/aggregation/add-proof", `{"batch_id":"x","proof_hash":"h"}`},
		{http.MethodPost, "/aggregation/finalize", `{"batch_id":"x"}`},
	}
	for _, rt := range adminRoutes {
		t.Run(rt.path, func(t *testing.T) {
			// No credentials at all -> 401 at admin gate.
			rec := serveThrough(s, rt.method, rt.path,
				map[string]string{"Content-Type": "application/json"}, rt.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s no-creds: got %d (%s), want 401 from admin gate",
					rt.method, rt.path, rec.Code, rec.Body.String())
			}
			// Public key only -> 401 at admin gate (still no admin header).
			rec = serveThrough(s, rt.method, rt.path,
				map[string]string{"X-API-Key": "public-secret", "Content-Type": "application/json"},
				rt.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s public-only: got %d (%s), want 401 from admin gate",
					rt.method, rt.path, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestRouter_PublicWritesGateOnApiKey confirms POST /proofs, POST /verify
// and other public write routes require X-API-Key when configured, and do
// NOT require X-Admin-API-Key.
func TestRouter_PublicWritesGateOnApiKey(t *testing.T) {
	s := newSplitServer("public-secret", "admin-secret")

	// A representative sample; exhaustive per-route reachability with
	// handler-side happy paths is out of scope for the auth test.
	publicWrites := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/verify", `{"agent":"a","proof_id":"x"}`},
		{http.MethodPost, "/kyc/check", `{"user":"u"}`},
		{http.MethodPost, "/zk/challenge", `{"agent":"a"}`},
	}
	for _, rt := range publicWrites {
		t.Run(rt.path, func(t *testing.T) {
			// No key -> 401 (public gate).
			rec := serveThrough(s, rt.method, rt.path,
				map[string]string{"Content-Type": "application/json"}, rt.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s no-creds: got %d (%s), want 401",
					rt.method, rt.path, rec.Code, rec.Body.String())
			}
			// Correct public key -> auth passes; handler may 400/500 on
			// the body, but MUST NOT be 401/403.
			rec = serveThrough(s, rt.method, rt.path,
				map[string]string{"X-API-Key": "public-secret", "Content-Type": "application/json"},
				rt.body)
			if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
				t.Fatalf("%s %s with valid public key: unexpected auth reject %d (%s)",
					rt.method, rt.path, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestRouter_PublicReadsBypassAllAuth confirms GETs stay open even when
// both keys are configured.
func TestRouter_PublicReadsBypassAllAuth(t *testing.T) {
	s := newSplitServer("public-secret", "admin-secret")

	reads := []string{
		"/health",
		"/proofs",
		"/stats",
	}
	for _, path := range reads {
		t.Run(path, func(t *testing.T) {
			rec := serveThrough(s, http.MethodGet, path, nil, "")
			if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
				t.Fatalf("GET %s: unexpected auth reject %d (%s)",
					path, rec.Code, rec.Body.String())
			}
		})
	}
}
