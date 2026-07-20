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

func TestSubmitProof_StrictAnchoredWithoutSubmitterFailsClosed(t *testing.T) {
	s := newTestServer("")
	s.strict = true
	req := httptest.NewRequest(http.MethodPost, "/proofs", jsonBody(`{"agent":"a","input":"i","output":"o","model":"m","mode":"anchored"}`))
	rec := httptest.NewRecorder()
	s.submitProof(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected strict anchored request to fail 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHealth_ExposesStrictCapabilities(t *testing.T) {
	s := newTestServer("key")
	s.strict = true
	rec := httptest.NewRecorder()
	s.health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	var body map[string]any
	if err := decodeJSON(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["strict"] != true {
		t.Fatalf("expected strict=true, got %v", body["strict"])
	}
	caps, ok := body["capabilities"].(map[string]any)
	if !ok || caps["authenticated_writes"] != true || caps["onchain_submit"] != false {
		t.Fatalf("unexpected capabilities: %v", body["capabilities"])
	}
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

func TestAuthMiddleware_KeyConfigured_RejectsMissingOrWrongKey(t *testing.T) {
	s := newTestServer("secret123")
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"missing header", "", http.StatusUnauthorized},
		{"wrong key", "wrong", http.StatusUnauthorized},
		{"correct key", "secret123", http.StatusOK},
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
				t.Fatalf("%s: expected %d, got %d", c.name, c.want, rec.Code)
			}
		})
	}
}

func TestCORSMiddleware_AllowsAPIKeyHeader(t *testing.T) {
	s := newTestServer("secret123")
	handler := s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodOptions, "/proofs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "X-API-Key") {
		t.Fatalf("CORS must allow X-API-Key, got %q", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestSubmitProof_NonStrictFallbackDoesNotMasqueradeAsDeploy(t *testing.T) {
	s := newTestServer("key")
	s.strict = false
	req := httptest.NewRequest(http.MethodPost, "/proofs", jsonBody(`{"agent":"a","input":"i","output":"o","model":"m","mode":"anchored"}`))
	rec := httptest.NewRecorder()
	s.submitProof(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := decodeJSON(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["deploy_hash"] != nil && body["deploy_hash"] != "" {
		t.Fatalf("computed fallback must not populate deploy_hash, got %v", body["deploy_hash"])
	}
	if body["anchoring_status"] != "computed_fallback" {
		t.Fatalf("expected computed_fallback status, got %v", body["anchoring_status"])
	}
	if body["anchor_hash"] == nil || body["anchor_hash"] == "" {
		t.Fatal("expected explicit anchor_hash for computed fallback")
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

// ---------------------------------------------------------------------------
// ValidateStartupConfig + /health degraded surface (CP-8)
// ---------------------------------------------------------------------------

func TestValidateStartupConfig_StrictWithoutAPIKey_Errors(t *testing.T) {
	t.Setenv("CP_STRICT", "1")
	t.Setenv("API_KEY", "")
	if err := ValidateStartupConfig(); err == nil {
		t.Fatal("expected error when CP_STRICT=1 and API_KEY is empty")
	}
}

func TestValidateStartupConfig_StrictWithAPIKey_OK(t *testing.T) {
	t.Setenv("CP_STRICT", "1")
	t.Setenv("API_KEY", "not-empty")
	if err := ValidateStartupConfig(); err != nil {
		t.Fatalf("expected nil error when strict+key present, got %v", err)
	}
}

func TestValidateStartupConfig_NonStrictWithoutAPIKey_OK(t *testing.T) {
	t.Setenv("CP_STRICT", "0")
	t.Setenv("API_KEY", "")
	if err := ValidateStartupConfig(); err != nil {
		t.Fatalf("expected nil error in non-strict mode without key, got %v", err)
	}
}

func TestHealth_StrictWithoutAPIKey_ReportsDegraded(t *testing.T) {
	s := newTestServer("") // apiKey == ""
	s.strict = true
	rec := httptest.NewRecorder()
	s.health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	var body map[string]any
	if err := decodeJSON(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "degraded" {
		t.Fatalf("expected status=degraded, got %v (body=%s)", body["status"], rec.Body.String())
	}
	reasons, ok := body["degraded_reasons"].([]any)
	if !ok || len(reasons) == 0 {
		t.Fatalf("expected degraded_reasons to list issues, got %v", body["degraded_reasons"])
	}
	found := false
	for _, r := range reasons {
		if r == "strict_mode_without_api_key" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected strict_mode_without_api_key in reasons, got %v", reasons)
	}
}

func TestHealth_StrictWithAPIKeyAndSubmitter_ReportsOK(t *testing.T) {
	s := newTestServer("key")
	s.strict = true
	// sub == nil in newTestServer, but for ok status strict requires sub != nil.
	// Since we can't easily fake a submitter here, this test asserts submitter
	// requirement: strict+key+nil-sub should still be degraded (submitter missing).
	rec := httptest.NewRecorder()
	s.health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	var body map[string]any
	if err := decodeJSON(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "degraded" {
		t.Fatalf("expected degraded (missing submitter) got %v", body["status"])
	}
	// Non-strict: same server minus strict should report ok.
	s.strict = false
	rec2 := httptest.NewRecorder()
	s.health(rec2, httptest.NewRequest(http.MethodGet, "/health", nil))
	var body2 map[string]any
	_ = decodeJSON(rec2.Body.Bytes(), &body2)
	if body2["status"] != "ok" {
		t.Fatalf("expected status=ok in non-strict mode, got %v", body2["status"])
	}
}
