package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// idempotencyStore is a small in-memory cache of finalized responses keyed
// by (client-id, X-Idempotency-Key). It exists to make POST retries safe
// against network flakiness and client-side re-submits: a retry with the
// same key returns the exact same response the first request produced,
// instead of re-running the mutation.
//
// Design constraints:
//   - Only POST is cached (GET is idempotent by definition; PUT/DELETE are
//     not currently used on any mutating endpoint that needs this).
//   - Empty X-Idempotency-Key header bypasses the middleware entirely —
//     callers opt in per request.
//   - Entries expire after 24h; a background sweeper reclaims memory.
//   - The cache key includes a hash of the caller's API key (or a
//     placeholder for unauthenticated dev traffic) so two clients that
//     happen to pick the same idempotency key cannot see each other's
//     responses.
//   - We also bind the entry to a hash of the request body so a client
//     that reuses a key with a different body gets a 409 Conflict rather
//     than a stale response.
//
// This is deliberately in-memory and single-node: it survives process
// lifetime, not restarts. A durable variant belongs in Postgres and is
// out of scope for the deadline MVP.
type idempotencyStore struct {
	mu      sync.Mutex
	entries map[string]*idempotencyEntry
	ttl     time.Duration
}

type idempotencyEntry struct {
	status    int
	headers   http.Header
	body      []byte
	bodyHash  string
	createdAt time.Time
}

func newIdempotencyStore(ttl time.Duration) *idempotencyStore {
	s := &idempotencyStore{
		entries: make(map[string]*idempotencyEntry),
		ttl:     ttl,
	}
	go s.sweep()
	return s
}

func (s *idempotencyStore) sweep() {
	t := time.NewTicker(s.ttl / 4)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		now := time.Now()
		for k, e := range s.entries {
			if now.Sub(e.createdAt) > s.ttl {
				delete(s.entries, k)
			}
		}
		s.mu.Unlock()
	}
}

func (s *idempotencyStore) get(key string) (*idempotencyEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if !ok {
		return nil, false
	}
	if time.Since(e.createdAt) > s.ttl {
		delete(s.entries, key)
		return nil, false
	}
	return e, true
}

func (s *idempotencyStore) put(key string, e *idempotencyEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = e
}

// idempotencyRecorder is an http.ResponseWriter that captures the status,
// headers, and body of a downstream handler so the middleware can persist
// the response for future retries.
type idempotencyRecorder struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (r *idempotencyRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *idempotencyRecorder) Write(b []byte) (int, error) {
	r.buf.Write(b)
	return r.ResponseWriter.Write(b)
}

// idempotencyMiddleware wraps every mutating request that carries an
// X-Idempotency-Key header. See idempotencyStore doc for the contract.
func (s *Server) idempotencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		key := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}
		// Bind to the caller so keys from different API clients cannot
		// collide. Unauthenticated dev traffic falls back to a shared
		// "anon" bucket — good enough for local demos.
		callerID := r.Header.Get("X-API-Key")
		if callerID == "" {
			callerID = "anon"
		}
		callerHash := sha256.Sum256([]byte(callerID))

		// Read the whole body up front so we can (a) hash it for
		// conflict detection and (b) put it back for the downstream
		// handler to consume.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.jsonError(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(body))

		bodyHash := sha256.Sum256(body)
		storeKey := r.URL.Path + "|" + hex.EncodeToString(callerHash[:8]) + "|" + key

		if cached, ok := idempotencyCache.get(storeKey); ok {
			if cached.bodyHash != hex.EncodeToString(bodyHash[:]) {
				// Same key, different body — the client made a mistake.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":            "idempotency key already used with a different request body",
					"idempotency_key":  key,
					"conflicting_hash": cached.bodyHash,
				})
				return
			}
			// Cache hit: replay the stored response verbatim.
			for k, v := range cached.headers {
				for _, vv := range v {
					w.Header().Add(k, vv)
				}
			}
			w.Header().Set("Idempotent-Replayed", "true")
			w.WriteHeader(cached.status)
			_, _ = w.Write(cached.body)
			return
		}

		rec := &idempotencyRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		// Only cache success / client-error responses; 5xx is transient
		// and the client SHOULD retry with the same key expecting a
		// fresh attempt.
		if rec.status >= 200 && rec.status < 500 {
			hdrs := make(http.Header)
			for k, v := range w.Header() {
				hdrs[k] = append([]string(nil), v...)
			}
			idempotencyCache.put(storeKey, &idempotencyEntry{
				status:    rec.status,
				headers:   hdrs,
				body:      append([]byte(nil), rec.buf.Bytes()...),
				bodyHash:  hex.EncodeToString(bodyHash[:]),
				createdAt: time.Now(),
			})
		}
	})
}

// idempotencyCache is the process-wide singleton store. 24h TTL matches
// what most external providers (Stripe et al) advertise as a client's
// reasonable retry horizon.
var idempotencyCache = newIdempotencyStore(24 * time.Hour)
