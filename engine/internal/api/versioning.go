package api

import (
	"net/http"
	"strings"
	"time"
)

// API versioning strategy — path-first, header-negotiated.
//
// Rationale: the current `/proofs`, `/verify`, `/zk/*` etc. routes are
// unversioned and callers baked their URLs into deployment configs a
// while ago. Renaming them outright would break every existing client.
// So this middleware layer does two things:
//
//   1. Accepts requests at `/v1/…` and rewrites the path to the
//      unversioned handler underneath. `/v1/proofs` → `/proofs`,
//      `/v1/zk/verify-groth16` → `/zk/verify-groth16`, and so on for
//      every route registered on the mux. Callers can migrate to
//      versioned URLs at their own pace.
//
//   2. Adds Deprecation / Sunset / Link:successor-version response
//      headers to responses served on the LEGACY (unversioned) path.
//      RFC 8594 / RFC 9745 headers, so a well-behaved client library
//      surfaces the upgrade path without any backend change. Callers on
//      `/v1/…` get no Deprecation headers.
//
// The Accept header is not yet enforced — spec is
// `Accept: application/vnd.cp+json; version=1` and future major bumps
// (breaking wire changes) will negotiate on that header. For v1 the
// header is optional and ignored; documenting it in OpenAPI now avoids
// a breaking client change when v2 lands.

// versionRewriteMiddleware turns `/v1/anything` into `/anything` so all
// existing handlers keep working. It also tags the request so the
// deprecation middleware knows NOT to attach sunset headers on the
// return trip (the client has already migrated).
func (s *Server) versionRewriteMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/v1")
			// RawPath mirrors Path when there's no percent-encoding; we
			// clear it so downstream mux matching uses the rewritten
			// Path. (Setting RawPath to the trimmed form would be
			// equally correct.)
			r.URL.RawPath = ""
			r.Header.Set("X-API-Version", "1")
			w.Header().Set("X-API-Version", "1")
		}
		next.ServeHTTP(w, r)
	})
}

// deprecationMiddleware attaches RFC 8594 sunset metadata to every
// response that was NOT served via the /v1/ path. The values are
// deliberately conservative — Sunset is 90 days out, which is
// long enough for a hackathon submission window and short enough to
// signal "please migrate soon" to any downstream reader.
//
// The Deprecation field is a bare "true" (RFC 9745 §2 — "Deprecation:
// true" is a valid value meaning "already deprecated, no future date").
// The Link header points at the versioned equivalent so the client
// library can auto-upgrade requests if it wants to.
func (s *Server) deprecationMiddleware(next http.Handler) http.Handler {
	// Compute the sunset date once at startup; it's a policy constant,
	// not a per-request field.
	sunset := time.Now().Add(90 * 24 * time.Hour).UTC().Format(http.TimeFormat)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only tag legacy paths. If the rewrite middleware already set
		// X-API-Version, this request came in on /v1/.
		if w.Header().Get("X-API-Version") == "" && r.Header.Get("X-API-Version") == "" {
			// Only tag paths that are actually part of the versioned
			// surface — skip /health, static assets, etc.
			if isVersionedRoute(r.URL.Path) {
				w.Header().Set("Deprecation", "true")
				w.Header().Set("Sunset", sunset)
				w.Header().Set("Link", `</v1`+r.URL.Path+`>; rel="successor-version"`)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// isVersionedRoute returns true for paths that are part of the /v1
// contract — mutation endpoints and content endpoints, not `/health`
// or CORS preflight surfaces.
func isVersionedRoute(path string) bool {
	if path == "/health" {
		return false
	}
	for _, prefix := range versionedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// versionedPrefixes enumerates the route families that live under
// `/v1/…`. Kept as a slice (not a set) because the list is small and
// occasionally scanned by hand when adding a new endpoint family.
var versionedPrefixes = []string{
	"/proofs",
	"/verify",
	"/stats",
	"/kyc/",
	"/inference/",
	"/aggregation/",
	"/zk/",
	"/proof-chain/",
	"/pq/",
}
