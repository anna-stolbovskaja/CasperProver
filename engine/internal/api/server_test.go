package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
)

func jsonBody(s string) *strings.Reader {
	return strings.NewReader(s)
}

func decodeJSON(b []byte, v any) error {
	return json.Unmarshal(b, v)
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func newTestServer(apiKey string) *Server {
	eng := prover.New()
	s := New(eng, 0, nil)
	s.apiKey = apiKey
	return s
}

func TestAuthMiddleware_NoKeyConfigured_AllowsAll(t *testing.T) {
	s := newTestServer("")
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/kyc/grant", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with no API key configured, got %d", rec.Code)
	}
}

func TestAuthMiddleware_KeyConfigured_AdminRoutes(t *testing.T) {
	s := newTestServer("secret123")
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Admin-gated routes:
	// - no header             → 401 (missing)
	// - wrong header value    → 403 (invalid / forbidden)
	// - correct header        → 200 (passes through)
	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"missing header", "", http.StatusUnauthorized},
		{"wrong key", "wrong", http.StatusForbidden},
		{"correct key", "secret123", http.StatusOK},
	}
	adminRoutes := []struct{ method, path string }{
		{http.MethodPost, "/kyc/grant"},
		{http.MethodPost, "/kyc/check"},
		{http.MethodPost, "/inference/register-model"},
		{http.MethodPost, "/aggregation/finalize"},
	}
	for _, route := range adminRoutes {
		for _, c := range cases {
			t.Run(c.name+":"+route.method+" "+route.path, func(t *testing.T) {
				req := httptest.NewRequest(route.method, route.path, nil)
				if c.header != "" {
					req.Header.Set("X-API-Key", c.header)
				}
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				if rec.Code != c.want {
					t.Fatalf("%s %s [%s]: expected %d, got %d",
						route.method, route.path, c.name, c.want, rec.Code)
				}
			})
		}
	}
}

func TestAuthMiddleware_KeyConfigured_GetAlwaysAllowed(t *testing.T) {
	s := newTestServer("secret123")
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET should bypass auth even with a key configured, got %d", rec.Code)
	}
}

// TestAuthMiddleware_PublicRoutes_NoAuthRequired verifies that the
// explicit-public POST endpoints (submit_proof, verify) pass through
// without X-API-Key even when the server has one configured.
func TestAuthMiddleware_PublicRoutes_NoAuthRequired(t *testing.T) {
	s := newTestServer("secret123")
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct{ method, path string }{
		{http.MethodPost, "/proofs"},
		{http.MethodPost, "/verify"},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s %s expected 200 (public), got %d", c.method, c.path, rec.Code)
			}
		})
	}
}

// TestAuthMiddleware_ErrorBodyIsJSON verifies that both 401 and 403
// responses carry a machine-readable JSON error body — not empty, not
// HTML, not an unhandled panic.
func TestAuthMiddleware_ErrorBodyIsJSON(t *testing.T) {
	s := newTestServer("secret123")
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"401 missing", "", http.StatusUnauthorized},
		{"403 invalid", "wrong", http.StatusForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/kyc/grant", nil)
			if c.header != "" {
				req.Header.Set("X-API-Key", c.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("expected %d, got %d", c.want, rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("expected JSON error body, Content-Type=%q", ct)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v (raw=%q)", err, rec.Body.String())
			}
			if body["error"] == nil {
				t.Fatalf("expected error field in body, got %v", body)
			}
		})
	}
}

func TestAggregationBatch_FullLifecycle(t *testing.T) {
	s := newTestServer("")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /aggregation/create-batch", s.aggregationCreateBatch)
	mux.HandleFunc("POST /aggregation/add-proof", s.aggregationAddProof)
	mux.HandleFunc("POST /aggregation/finalize", s.aggregationFinalize)
	mux.HandleFunc("GET /aggregation/batch/{id}", s.aggregationGetBatch)

	post := func(path, body string) int {
		req := httptest.NewRequest(http.MethodPost, path, jsonBody(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}
	get := func(path string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := get("/aggregation/batch/unknown"); code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown batch, got %d", code)
	}
	if code := post("/aggregation/create-batch", `{"batch_id":"b1","merkle_root":"abc","max_proofs":1}`); code != http.StatusCreated {
		t.Fatalf("expected 201 creating batch, got %d", code)
	}
	if code := post("/aggregation/create-batch", `{"batch_id":"b1","merkle_root":"abc","max_proofs":1}`); code != http.StatusConflict {
		t.Fatalf("expected 409 creating duplicate batch, got %d", code)
	}
	if code := post("/aggregation/add-proof", `{"batch_id":"b1","proof_hash":"h1","leaf_index":0}`); code != http.StatusOK {
		t.Fatalf("expected 200 adding proof, got %d", code)
	}
	if code := post("/aggregation/add-proof", `{"batch_id":"b1","proof_hash":"h2","leaf_index":1}`); code != http.StatusConflict {
		t.Fatalf("expected 409 adding proof past max_proofs, got %d", code)
	}
	if code := post("/aggregation/finalize", `{"batch_id":"b1"}`); code != http.StatusOK {
		t.Fatalf("expected 200 finalizing batch, got %d", code)
	}
	if code := post("/aggregation/add-proof", `{"batch_id":"b1","proof_hash":"h3","leaf_index":2}`); code != http.StatusConflict {
		t.Fatalf("expected 409 adding proof to finalized batch, got %d", code)
	}
}

func TestAggregationVerifyBatch_RunsThroughRealAggregator(t *testing.T) {
	s := newTestServer("")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /aggregation/create-batch", s.aggregationCreateBatch)
	mux.HandleFunc("POST /aggregation/add-proof", s.aggregationAddProof)
	mux.HandleFunc("POST /aggregation/finalize", s.aggregationFinalize)
	mux.HandleFunc("GET /aggregation/verify-batch/{id}", s.aggregationVerifyBatch)

	do := func(method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, path, jsonBody(body))
		} else {
			r = httptest.NewRequest(method, path, nil)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, r)
		var resp map[string]any
		_ = decodeJSON(rec.Body.Bytes(), &resp)
		return rec, resp
	}

	if rec, _ := do(http.MethodGet, "/aggregation/verify-batch/b2", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 verifying unknown batch, got %d", rec.Code)
	}

	do(http.MethodPost, "/aggregation/create-batch", `{"batch_id":"b2","merkle_root":"r","max_proofs":5}`)
	if rec, _ := do(http.MethodGet, "/aggregation/verify-batch/b2", ""); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 verifying a non-finalized batch, got %d", rec.Code)
	}

	do(http.MethodPost, "/aggregation/add-proof", `{"batch_id":"b2","proof_hash":"p1","leaf_index":0}`)
	do(http.MethodPost, "/aggregation/add-proof", `{"batch_id":"b2","proof_hash":"p2","leaf_index":1}`)
	finRec, finResp := do(http.MethodPost, "/aggregation/finalize", `{"batch_id":"b2"}`)
	if finRec.Code != http.StatusOK {
		t.Fatalf("finalize expected 200, got %d", finRec.Code)
	}
	if _, ok := finResp["aggregate_proof_hash"]; !ok {
		t.Fatal("expected finalize response to include aggregate_proof_hash")
	}

	verifyRec, verifyResp := do(http.MethodGet, "/aggregation/verify-batch/b2", "")
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify-batch expected 200, got %d: %v", verifyRec.Code, verifyResp)
	}
	if verifyResp["valid"] != true {
		t.Fatalf("expected a freshly finalized batch to verify as valid, got %v", verifyResp)
	}
	if verifyResp["aggregate_proof_hash"] != finResp["aggregate_proof_hash"] {
		t.Fatalf("expected verify-batch to report the same aggregate hash as finalize")
	}
}

// These exercise the real PQ crypto wired into the HTTP layer (previously
// pqVerifySPHINCS/pqHybridVerify always returned valid:true regardless of
// input - see internal/crypto's package doc for the underlying bug).

func TestPQSignVerifySPHINCS_HTTPRoundTrip(t *testing.T) {
	s := newTestServer("")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /pq/sign-sphincs", s.pqSignSPHINCS)
	mux.HandleFunc("POST /pq/verify-sphincs", s.pqVerifySPHINCS)

	signReq := httptest.NewRequest(http.MethodPost, "/pq/sign-sphincs", jsonBody(`{"message":"hello"}`))
	signRec := httptest.NewRecorder()
	mux.ServeHTTP(signRec, signReq)
	if signRec.Code != http.StatusOK {
		t.Fatalf("sign expected 200, got %d: %s", signRec.Code, signRec.Body.String())
	}
	var signResp map[string]any
	if err := decodeJSON(signRec.Body.Bytes(), &signResp); err != nil {
		t.Fatalf("decode sign response: %v", err)
	}

	verifyBody := `{"message":"hello","signature":"` + asString(signResp["signature"]) + `","public_key":"` + asString(signResp["public_key"]) + `"}`
	verifyReq := httptest.NewRequest(http.MethodPost, "/pq/verify-sphincs", jsonBody(verifyBody))
	verifyRec := httptest.NewRecorder()
	mux.ServeHTTP(verifyRec, verifyReq)
	var verifyResp map[string]any
	if err := decodeJSON(verifyRec.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}
	if verifyResp["valid"] != true {
		t.Fatalf("expected valid=true for a genuine signature, got %v", verifyResp)
	}

	// Tampered message must be rejected.
	tamperedBody := `{"message":"goodbye","signature":"` + asString(signResp["signature"]) + `","public_key":"` + asString(signResp["public_key"]) + `"}`
	tamperedReq := httptest.NewRequest(http.MethodPost, "/pq/verify-sphincs", jsonBody(tamperedBody))
	tamperedRec := httptest.NewRecorder()
	mux.ServeHTTP(tamperedRec, tamperedReq)
	var tamperedResp map[string]any
	_ = decodeJSON(tamperedRec.Body.Bytes(), &tamperedResp)
	if tamperedResp["valid"] != false {
		t.Fatalf("expected valid=false for a tampered message, got %v", tamperedResp)
	}
}

func TestPQHybridSignVerify_HTTPRoundTrip(t *testing.T) {
	s := newTestServer("")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /pq/hybrid-sign", s.pqHybridSign)
	mux.HandleFunc("POST /pq/hybrid-verify", s.pqHybridVerify)

	signReq := httptest.NewRequest(http.MethodPost, "/pq/hybrid-sign", jsonBody(`{"message":"hello"}`))
	signRec := httptest.NewRecorder()
	mux.ServeHTTP(signRec, signReq)
	if signRec.Code != http.StatusOK {
		t.Fatalf("sign expected 200, got %d: %s", signRec.Code, signRec.Body.String())
	}
	var signResp map[string]any
	if err := decodeJSON(signRec.Body.Bytes(), &signResp); err != nil {
		t.Fatalf("decode sign response: %v", err)
	}

	verifyBody := `{"message":"hello","signature":"` + asString(signResp["signature"]) +
		`","classic_public_key":"` + asString(signResp["classic_public_key"]) +
		`","pq_public_key":"` + asString(signResp["pq_public_key"]) + `"}`
	verifyReq := httptest.NewRequest(http.MethodPost, "/pq/hybrid-verify", jsonBody(verifyBody))
	verifyRec := httptest.NewRecorder()
	mux.ServeHTTP(verifyRec, verifyReq)
	var verifyResp map[string]any
	if err := decodeJSON(verifyRec.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}
	if verifyResp["valid"] != true || verifyResp["classic_valid"] != true || verifyResp["pq_valid"] != true {
		t.Fatalf("expected all-valid for a genuine hybrid signature, got %v", verifyResp)
	}

	// Tampered message must be rejected on both components.
	tamperedBody := `{"message":"goodbye","signature":"` + asString(signResp["signature"]) +
		`","classic_public_key":"` + asString(signResp["classic_public_key"]) +
		`","pq_public_key":"` + asString(signResp["pq_public_key"]) + `"}`
	tamperedReq := httptest.NewRequest(http.MethodPost, "/pq/hybrid-verify", jsonBody(tamperedBody))
	tamperedRec := httptest.NewRecorder()
	mux.ServeHTTP(tamperedRec, tamperedReq)
	var tamperedResp map[string]any
	_ = decodeJSON(tamperedRec.Body.Bytes(), &tamperedResp)
	if tamperedResp["valid"] != false {
		t.Fatalf("expected valid=false for a tampered message, got %v", tamperedResp)
	}
}
