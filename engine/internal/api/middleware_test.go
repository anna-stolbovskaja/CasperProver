package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// helper: assemble the middleware chain the same way Server.Start does,
// but around a fake terminal handler so we can inspect what came
// through without booting the full server.
func newMiddlewareTestHarness(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	s := &Server{apiKey: "", strict: false}
	mux := http.NewServeMux()
	// A single POST route that echoes its body back — mimics a real
	// mutation endpoint (submitProof etc.) closely enough for the
	// middleware tests.
	mux.HandleFunc("POST /proofs", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	})
	// A GET route for the versioning tests.
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"stats":true}`))
	})
	chain := s.versionRewriteMiddleware(
		s.idempotencyMiddleware(
			s.deprecationMiddleware(mux)))
	return s, chain
}

func TestVersionRewrite_V1PathReachesUnversionedHandler(t *testing.T) {
	_, h := newMiddlewareTestHarness(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/stats", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-API-Version"); got != "1" {
		t.Fatalf("expected X-API-Version=1, got %q", got)
	}
	// /v1/ requests must NOT get Deprecation headers.
	if got := rec.Header().Get("Deprecation"); got != "" {
		t.Fatalf("expected no Deprecation on /v1 request, got %q", got)
	}
	if got := rec.Header().Get("Sunset"); got != "" {
		t.Fatalf("expected no Sunset on /v1 request, got %q", got)
	}
}

func TestDeprecation_LegacyPathGetsSunsetHeaders(t *testing.T) {
	_, h := newMiddlewareTestHarness(t)
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Deprecation"); got != "true" {
		t.Fatalf("expected Deprecation=true, got %q", got)
	}
	sunset := rec.Header().Get("Sunset")
	if sunset == "" {
		t.Fatalf("expected Sunset header on legacy path")
	}
	// Sunset date must be a valid IMF-fixdate parseable by time.Parse.
	if _, err := time.Parse(http.TimeFormat, sunset); err != nil {
		t.Fatalf("Sunset %q not IMF-fixdate: %v", sunset, err)
	}
	link := rec.Header().Get("Link")
	if !strings.Contains(link, "/v1/stats") || !strings.Contains(link, `rel="successor-version"`) {
		t.Fatalf("expected Link to point at /v1/stats successor, got %q", link)
	}
}

func TestDeprecation_HealthNotFlagged(t *testing.T) {
	// /health is intentionally OUTSIDE the versioned surface so that
	// probe consumers don't get a sunset warning on their liveness check.
	s := &Server{apiKey: ""}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	h := s.versionRewriteMiddleware(s.deprecationMiddleware(mux))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Deprecation") != "" {
		t.Fatalf("health probe should not be deprecation-tagged")
	}
}

func TestIdempotency_RetrySameKeyReplaysResponse(t *testing.T) {
	_, h := newMiddlewareTestHarness(t)

	body := []byte(`{"proof":"deadbeef"}`)
	makeReq := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/proofs", bytes.NewReader(body))
		req.Header.Set("X-Idempotency-Key", "abc-123")
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, makeReq())
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first: expected 201, got %d body=%s", rec1.Code, rec1.Body.String())
	}
	if rec1.Header().Get("Idempotent-Replayed") == "true" {
		t.Fatalf("first request must not be marked as replayed")
	}

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, makeReq())
	if rec2.Code != http.StatusCreated {
		t.Fatalf("replay: expected 201, got %d", rec2.Code)
	}
	if rec2.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("replay must set Idempotent-Replayed=true")
	}
	if !bytes.Equal(rec1.Body.Bytes(), rec2.Body.Bytes()) {
		t.Fatalf("replay body diverged:\n first=%s\nreplay=%s",
			rec1.Body.String(), rec2.Body.String())
	}
}

func TestIdempotency_SameKeyDifferentBodyConflicts(t *testing.T) {
	_, h := newMiddlewareTestHarness(t)

	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, httptest.NewRequest(
		http.MethodPost, "/proofs", bytes.NewReader([]byte(`{"a":1}`))))
	// The first request has no key — it should NOT populate the cache.
	if rec1.Code != http.StatusCreated {
		t.Fatalf("no-key request should pass through, got %d", rec1.Code)
	}

	// Now do two requests with the same key + different bodies.
	req1 := httptest.NewRequest(http.MethodPost, "/proofs", bytes.NewReader([]byte(`{"a":1}`)))
	req1.Header.Set("X-Idempotency-Key", "key-conflict")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req1)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("first keyed request: got %d", rec2.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/proofs", bytes.NewReader([]byte(`{"a":2}`)))
	req2.Header.Set("X-Idempotency-Key", "key-conflict")
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req2)
	if rec3.Code != http.StatusConflict {
		t.Fatalf("conflicting body reuse: expected 409, got %d body=%s",
			rec3.Code, rec3.Body.String())
	}
	var payload map[string]string
	if err := json.NewDecoder(rec3.Body).Decode(&payload); err != nil {
		t.Fatalf("decode conflict body: %v", err)
	}
	if payload["idempotency_key"] != "key-conflict" {
		t.Fatalf("expected idempotency_key echoed, got %+v", payload)
	}
}

func TestIdempotency_EmptyKeyBypassesCache(t *testing.T) {
	_, h := newMiddlewareTestHarness(t)

	// Two identical requests with NO key — each should hit the handler
	// afresh (Idempotent-Replayed must never be set).
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/proofs", bytes.NewReader([]byte(`{"x":1}`)))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Header().Get("Idempotent-Replayed") == "true" {
			t.Fatalf("no-key request #%d should not be replayed", i+1)
		}
	}
}
