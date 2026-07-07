package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
)

func jsonBody(s string) *strings.Reader {
	return strings.NewReader(s)
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
