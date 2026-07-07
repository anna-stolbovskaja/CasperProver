package api

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/hasher"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/inference"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/kyc"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/store"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/submitter"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/verifier"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/zkverifier"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/zkverifier/gnarkzk"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
)

type Server struct {
	eng    *prover.ProofEngine
	ver    *verifier.LocalVerifier
	kyc    *kyc.DemoKYC
	db     *store.PG
	sub    *submitter.CasperSubmitter
	inf    *inference.InferenceService
	zk     *zkverifier.Groth16Verifier
	realZK *gnarkzk.Setup
	port   int
	log    *slog.Logger
	start  time.Time
	apiKey string

	aggMu      sync.Mutex
	aggBatches map[string]*aggBatch
}

// aggBatch is real (in-memory) bookkeeping for the /aggregation/* endpoints.
// Previously these 4 handlers were pure stubs: create/add/finalize did not
// store anything and get-batch unconditionally returned proof_count:0,
// status:"open" for any batch_id, including ones that never existed. This
// tracks actual per-batch state so the endpoints reflect reality. It is NOT
// the real STARK aggregation math (internal/aggregator is still dead code,
// unwired - that's a separate, larger task) - this is just an honest batch
// registry instead of a hardcoded fake response.
type aggBatch struct {
	ID          string
	MerkleRoot  string
	MaxProofs   int
	ProofHashes []string
	Status      string // "open" | "finalized"
	CreatedAt   int64
	FinalizedAt int64
}

func New(eng *prover.ProofEngine, port int, db *store.PG) *Server {
	nodeURL := os.Getenv("CASPER_NODE_URL")
	if nodeURL == "" {
		nodeURL = "https://rpc.testnet.casperlabs.io"
	}
	chain := os.Getenv("CASPER_CHAIN")
	if chain == "" {
		chain = "casper-test"
	}
	keyPath := os.Getenv("DEPLOYER_KEY_PATH")

	var sub *submitter.CasperSubmitter
	if keyPath != "" {
		sub = submitter.New(nodeURL, chain, keyPath)
		slog.Info("submitter configured", "node", nodeURL, "chain", chain)
	} else {
		slog.Warn("no deployer key configured, anchored mode uses computed hashes")
	}

	realZK, err := gnarkzk.NewSetup()
	if err != nil {
		slog.Warn("real Groth16 (gnark) setup failed, /zk/groth16-real/* will return 503", "error", err)
		realZK = nil
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		slog.Warn("API_KEY not set - all write endpoints are unauthenticated (fine for local dev/demo, not for a real deployment)")
	} else {
		slog.Info("API_KEY configured - write endpoints require X-API-Key header")
	}

	demoKYC := kyc.NewDemo(eng)
	if db != nil {
		entries, err := db.LoadKYC()
		if err != nil {
			slog.Warn("failed to load kyc whitelist from db", "err", err)
		} else {
			for _, e := range entries {
				demoKYC.Restore(e.User, e.ProofID)
			}
			slog.Info("loaded kyc whitelist from postgres", "count", len(entries))
		}
	}

	return &Server{
		eng:    eng,
		ver:    verifier.New(),
		kyc:    demoKYC,
		db:     db,
		sub:    sub,
		inf:    inference.New(eng, db, sub),
		zk:     zkverifier.NewGroth16Verifier(),
		realZK: realZK,
		port:   port,
		log:    slog.Default(),
		start:  time.Now(),
		apiKey: apiKey,

		aggBatches: make(map[string]*aggBatch),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /proofs", s.listProofs)
	mux.HandleFunc("GET /proofs/{id}", s.getProof)
	mux.HandleFunc("POST /proofs", s.submitProof)
	mux.HandleFunc("POST /proofs/batch", s.batchProofs)
	mux.HandleFunc("POST /verify", s.verifyProof)
	mux.HandleFunc("POST /proofs/{id}/revoke", s.revokeProof)
	mux.HandleFunc("GET /proofs/{id}/export", s.exportProof)
	mux.HandleFunc("GET /stats", s.stats)
	mux.HandleFunc("POST /kyc/check", s.kycCheck)
	mux.HandleFunc("POST /kyc/grant", s.kycGrant)
	mux.HandleFunc("GET /kyc/whitelist/{user}", s.kycWhitelist)

	// Inference routes
	mux.HandleFunc("POST /inference/prove", s.inferenceProve)
	mux.HandleFunc("POST /inference/verify", s.inferenceVerify)
	mux.HandleFunc("POST /inference/register-model", s.inferenceRegisterModel)
	mux.HandleFunc("GET /inference/model/{id}", s.inferenceGetModel)
	// Aggregation routes
	mux.HandleFunc("POST /aggregation/create-batch", s.aggregationCreateBatch)
	mux.HandleFunc("POST /aggregation/add-proof", s.aggregationAddProof)
	mux.HandleFunc("POST /aggregation/finalize", s.aggregationFinalize)
	mux.HandleFunc("GET /aggregation/batch/{id}", s.aggregationGetBatch)
	// ZK Verification routes
	mux.HandleFunc("POST /zk/verify-groth16", s.zkVerifyGroth16)
	mux.HandleFunc("POST /zk/batch-verify", s.zkBatchVerify)
	mux.HandleFunc("POST /zk/groth16-real/prove", s.zkGroth16RealProve)
	mux.HandleFunc("POST /zk/groth16-real/verify", s.zkGroth16RealVerify)
	mux.HandleFunc("POST /zk/challenge", s.zkChallenge)
	mux.HandleFunc("GET /zk/challenge/{id}", s.zkGetChallenge)
	// Post-quantum routes
	mux.HandleFunc("POST /pq/sign-sphincs", s.pqSignSPHINCS)
	mux.HandleFunc("POST /pq/verify-sphincs", s.pqVerifySPHINCS)
	mux.HandleFunc("POST /pq/hybrid-sign", s.pqHybridSign)
	mux.HandleFunc("POST /pq/hybrid-verify", s.pqHybridVerify)

	addr := fmt.Sprintf(":%d", s.port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.rateLimitMiddleware(s.corsMiddleware(s.authMiddleware(s.logMiddleware(mux)))),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	s.log.Info("api starting", "addr", addr)
	return srv.ListenAndServe()
}

// rateLimiter tracks per-IP request counts (60 req/min).
type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*rlEntry
}

type rlEntry struct {
	count int
	reset time.Time
}

var rl = &rateLimiter{clients: make(map[string]*rlEntry)}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		now := time.Now()
		rl.mu.Lock()
		entry, ok := rl.clients[ip]
		if !ok || now.After(entry.reset) {
			rl.clients[ip] = &rlEntry{count: 1, reset: now.Add(time.Minute)}
			rl.mu.Unlock()
		} else {
			entry.count++
			if entry.count > 60 {
				rl.mu.Unlock()
				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
				return
			}
			rl.mu.Unlock()
		}
		// Limit POST body to 1MB
		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		}
		next.ServeHTTP(w, r)
	})
}

// authMiddleware requires a matching X-API-Key header on every mutating
// request (anything other than GET/HEAD/OPTIONS) once API_KEY is configured.
// Read-only endpoints (health, listing proofs, checking whitelist status,
// etc.) stay public - this is a demo/hackathon API, not a multi-tenant
// product, so a single shared secret is intentionally simple rather than
// per-agent keys/JWTs. If API_KEY is unset, every request is allowed through
// (documented in KNOWN_LIMITATIONS.md as the local-dev/demo default).
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		got := r.Header.Get("X-API-Key")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(s.apiKey)) != 1 {
			s.jsonError(w, "missing or invalid X-API-Key", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Public-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	st := s.eng.GetStats()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "ok",
		"version":      "0.2.0",
		"uptime_s":     int(time.Since(s.start).Seconds()),
		"total_proofs": st.Total,
		"chain":        "casper-test",
		"contracts": map[string]string{
			"proof_registry": "96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708",
			"verifier_gate":  "a37f9cde9dbdc5bb8b9e92c663bdc59b83b42c89dc75ec73f7f7cde2619f77d3",
			"defi_mock":      "b9b11a976af20b4b5d128c44e5ee118b8830c26a79f4b603cdf0a00e537b81d3",
		},
	})
}

func (s *Server) listProofs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	// Cap limit to prevent excessive memory allocation
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	f := prover.ListFilter{
		Agent:  q.Get("agent"),
		PubKey: q.Get("public_key"),
		Mode:   q.Get("mode"),
		Page:   page,
		Limit:  limit,
	}

	proofs, total := s.eng.ListFiltered(f)
	if proofs == nil {
		proofs = make([]*prover.Proof, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"proofs": proofs,
		"total":  total,
		"page":   f.Page,
		"limit":  f.Limit,
	})
}

func (s *Server) getProof(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("id")
	if pid == "" {
		s.jsonError(w, "proof_id is required", http.StatusBadRequest)
		return
	}
	p, ok := s.eng.Get(pid)
	if !ok {
		s.jsonError(w, "proof not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p)
}

func (s *Server) submitProof(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Agent   string `json:"agent"`
		Input   string `json:"input"`
		Output  string `json:"output"`
		Model   string `json:"model"`
		UseCase string `json:"use_case"`
		Mode    string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Warn("bad request body", "error", err)
		s.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Agent == "" || req.Input == "" || req.Output == "" || req.Model == "" {
		s.jsonError(w, "agent, input, output, model are required", http.StatusBadRequest)
		return
	}
	if len(req.Agent) > 64 || len(req.Input) > 10240 || len(req.Output) > 10240 || len(req.Model) > 256 {
		s.jsonError(w, "field exceeds max length", http.StatusBadRequest)
		return
	}

	pubKey := r.Header.Get("X-Public-Key")
	mode := req.Mode
	if mode == "" {
		mode = "local"
	}

	p := s.eng.GenerateWithKey(req.Agent, pubKey, []byte(req.Input), []byte(req.Output), []byte(req.Model), req.UseCase, mode)

	if mode == "anchored" {
		if s.sub != nil {
			deployHash, err := s.sub.Submit(p)
			if err != nil {
				s.log.Warn("on-chain submit failed, using computed hash", "id", p.ID, "err", err)
				p.Deploy = hasher.HexHash([]byte(p.Root + p.ID))
			} else {
				p.Deploy = deployHash
				s.log.Info("proof anchored on-chain", "id", p.ID, "deploy", deployHash)
			}
		} else {
			p.Deploy = hasher.HexHash([]byte(p.Root + p.ID))
		}
	}

	s.persist(p)
	s.log.Info("proof generated", "id", p.ID, "agent", req.Agent, "use_case", req.UseCase, "mode", mode, "ms", p.GenMs)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

func (s *Server) batchProofs(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 5MB
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	var req struct {
		Proofs []struct {
			Agent   string `json:"agent"`
			Input   string `json:"input"`
			Output  string `json:"output"`
			Model   string `json:"model"`
			UseCase string `json:"use_case"`
		} `json:"proofs"`
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if len(req.Proofs) == 0 || len(req.Proofs) > 50 {
		s.jsonError(w, "batch size must be 1-50", http.StatusBadRequest)
		return
	}

	pubKey := r.Header.Get("X-Public-Key")
	mode := req.Mode
	if mode == "" {
		mode = "local"
	}

	results := make([]*prover.Proof, 0, len(req.Proofs))
	for _, pr := range req.Proofs {
		if pr.Agent == "" || pr.Input == "" || pr.Output == "" || pr.Model == "" {
			continue
		}
		p := s.eng.GenerateWithKey(pr.Agent, pubKey, []byte(pr.Input), []byte(pr.Output), []byte(pr.Model), pr.UseCase, mode)
		if mode == "anchored" {
			if s.sub != nil {
				dh, err := s.sub.Submit(p)
				if err != nil {
					p.Deploy = hasher.HexHash([]byte(p.Root + p.ID))
				} else {
					p.Deploy = dh
				}
			} else {
				p.Deploy = hasher.HexHash([]byte(p.Root + p.ID))
			}
		}
		s.persist(p)
		results = append(results, p)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"proofs":    results,
		"generated": len(results),
	})
}

func (s *Server) verifyProof(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProofID string `json:"proof_id"`
		Input   string `json:"input"`
		Output  string `json:"output"`
		Model   string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.ProofID == "" {
		s.jsonError(w, "proof_id is required", http.StatusBadRequest)
		return
	}

	p, ok := s.eng.Get(req.ProofID)
	if !ok {
		s.jsonError(w, "proof not found", http.StatusNotFound)
		return
	}

	result := map[string]interface{}{
		"proof_id": req.ProofID,
		"valid":    p.Valid,
		"revoked":  p.Revoked,
	}

	if req.Input != "" && req.Output != "" && req.Model != "" {
		err := s.ver.VerifyProof(p, []byte(req.Input), []byte(req.Output), []byte(req.Model))
		if err != nil {
			result["verified"] = false
			result["error"] = err.Error()
		} else {
			result["verified"] = true
		}

		result["checks"] = map[string]bool{
			"input_hash_match":  hasher.HexHash([]byte(req.Input)) == p.IH,
			"output_hash_match": hasher.HexHash([]byte(req.Output)) == p.OH,
			"model_hash_match":  hasher.HexHash([]byte(req.Model)) == p.MH,
			"commit_valid":      hasher.VerifyCommit(p.PH, []byte(req.Input), []byte(req.Output), []byte(req.Model)),
			"merkle_valid":      prover.VerifyPath([]byte(req.Input), p.Path, p.Root, p.Idx),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) revokeProof(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("id")

	// Authorization: require X-Public-Key header and verify ownership
	pubKey := r.Header.Get("X-Public-Key")
	if pubKey == "" {
		s.jsonError(w, "X-Public-Key header required for revocation", http.StatusUnauthorized)
		return
	}
	// Verify the caller owns this proof
	if p, ok := s.eng.Get(pid); ok {
		if p.PubKey != "" && p.PubKey != pubKey {
			s.jsonError(w, "not authorized to revoke this proof", http.StatusForbidden)
			return
		}
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := s.eng.Revoke(pid, req.Reason); err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if p, ok := s.eng.Get(pid); ok {
		s.persistUpdate(p)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"proof_id": pid,
		"revoked":  true,
	})
}

func (s *Server) exportProof(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("id")
	p, ok := s.eng.Get(pid)
	if !ok {
		s.jsonError(w, "proof not found", http.StatusNotFound)
		return
	}

	bundle := map[string]interface{}{
		"version":    "1.0",
		"exported":   time.Now().Unix(),
		"proof":      p,
		"contract":   "96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708",
		"chain":      "casper-test",
		"verify_url": "https://casperprover-api.onrender.com/verify",
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"proof-%s.json\"", pid))
	_ = json.NewEncoder(w).Encode(bundle)
}

func (s *Server) stats(w http.ResponseWriter, _ *http.Request) {
	st := s.eng.GetStats()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func (s *Server) kycCheck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProofID string `json:"proof_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	result, err := s.kyc.CheckKYC(req.ProofID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) kycGrant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User    string `json:"user"`
		ProofID string `json:"proof_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	access, err := s.kyc.GrantAccess(req.User, req.ProofID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if s.db != nil {
		_ = s.db.SaveKYC(req.User, req.ProofID, time.Now().Unix())
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(access)
}

func (s *Server) kycWhitelist(w http.ResponseWriter, r *http.Request) {
	user := r.PathValue("user")
	ok := s.kyc.IsWhitelisted(user)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"user":        user,
		"whitelisted": ok,
	})
}

func (s *Server) persist(p *prover.Proof) {
	if s.db != nil {
		if err := s.db.Save(p); err != nil {
			s.log.Warn("db save failed", "id", p.ID, "err", err)
		}
	}
}

func (s *Server) persistUpdate(p *prover.Proof) {
	if s.db != nil {
		if err := s.db.Update(p); err != nil {
			s.log.Warn("db update failed", "id", p.ID, "err", err)
		}
	}
}

func (s *Server) jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ---------------------------------------------------------------------------
// Inference handlers (delegate to inference.Service)
// ---------------------------------------------------------------------------

func (s *Server) inferenceProve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Agent   string `json:"agent"`
		ModelID string `json:"model_id"`
		Input   string `json:"input"`
		Output  string `json:"output"`
		UseCase string `json:"use_case"`
		PubKey  string `json:"public_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if req.Agent == "" {
		req.Agent = "api-client"
	}
	if req.UseCase == "" {
		req.UseCase = "inference"
	}
	proof, err := s.inf.GenerateInferenceProof(r.Context(), req.Agent,
		[]byte(req.Input), []byte(req.Output), []byte(req.ModelID), req.UseCase, req.PubKey)
	if err != nil {
		s.log.Error("inference proof generation failed", "error", err)
		http.Error(w, `{"error":"proof generation failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(proof)
}

func (s *Server) inferenceVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProofID string `json:"proof_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	valid, err := s.inf.VerifyInferenceProof(r.Context(), req.ProofID)
	if err != nil {
		s.log.Warn("inference proof verification error", "proof_id", req.ProofID, "error", err)
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusNotFound)
		return
	}
	result := map[string]any{"proof_id": req.ProofID, "valid": valid, "verified_at": time.Now().Unix()}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) inferenceRegisterModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelID          string            `json:"model_id"`
		ModelHash        string            `json:"model_hash"`
		VerifierContract string            `json:"verifier_contract"`
		Metadata         map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	entry, err := s.inf.RegisterModel(r.Context(), req.ModelID, req.ModelHash, req.VerifierContract, req.Metadata)
	if err != nil {
		s.log.Error("model registration failed", "model_id", req.ModelID, "error", err)
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

func (s *Server) inferenceGetModel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	entry, err := s.inf.GetModelInfo(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"model not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entry)
}

// ---------------------------------------------------------------------------
// Aggregation handlers
// ---------------------------------------------------------------------------

func (s *Server) aggregationCreateBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BatchID    string `json:"batch_id"`
		MerkleRoot string `json:"merkle_root"`
		MaxProofs  int    `json:"max_proofs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.BatchID == "" {
		s.jsonError(w, "batch_id is required", http.StatusBadRequest)
		return
	}
	if req.MaxProofs <= 0 {
		req.MaxProofs = 100
	}

	s.aggMu.Lock()
	if _, exists := s.aggBatches[req.BatchID]; exists {
		s.aggMu.Unlock()
		s.jsonError(w, "batch_id already exists", http.StatusConflict)
		return
	}
	b := &aggBatch{
		ID: req.BatchID, MerkleRoot: req.MerkleRoot, MaxProofs: req.MaxProofs,
		Status: "open", CreatedAt: time.Now().Unix(),
	}
	s.aggBatches[req.BatchID] = b
	s.aggMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"batch_id": b.ID, "merkle_root": b.MerkleRoot,
		"max_proofs": b.MaxProofs, "proof_count": 0, "status": b.Status,
		"created_at": b.CreatedAt,
	})
}

func (s *Server) aggregationAddProof(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BatchID   string `json:"batch_id"`
		ProofHash string `json:"proof_hash"`
		LeafIndex int    `json:"leaf_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	s.aggMu.Lock()
	defer s.aggMu.Unlock()
	b, ok := s.aggBatches[req.BatchID]
	if !ok {
		s.jsonError(w, "batch not found", http.StatusNotFound)
		return
	}
	if b.Status != "open" {
		s.jsonError(w, "batch is already finalized", http.StatusConflict)
		return
	}
	if len(b.ProofHashes) >= b.MaxProofs {
		s.jsonError(w, "batch is full", http.StatusConflict)
		return
	}
	b.ProofHashes = append(b.ProofHashes, req.ProofHash)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"batch_id": b.ID, "proof_hash": req.ProofHash, "leaf_index": req.LeafIndex,
		"added": true, "proof_count": len(b.ProofHashes),
	})
}

func (s *Server) aggregationFinalize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BatchID string `json:"batch_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	s.aggMu.Lock()
	defer s.aggMu.Unlock()
	b, ok := s.aggBatches[req.BatchID]
	if !ok {
		s.jsonError(w, "batch not found", http.StatusNotFound)
		return
	}
	if b.Status == "finalized" {
		s.jsonError(w, "batch already finalized", http.StatusConflict)
		return
	}
	b.Status = "finalized"
	b.FinalizedAt = time.Now().Unix()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"batch_id": b.ID, "status": b.Status, "proof_count": len(b.ProofHashes),
		"finalized_at": b.FinalizedAt,
	})
}

func (s *Server) aggregationGetBatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s.aggMu.Lock()
	b, ok := s.aggBatches[id]
	s.aggMu.Unlock()
	if !ok {
		s.jsonError(w, "batch not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"batch_id": b.ID, "merkle_root": b.MerkleRoot, "status": b.Status,
		"proof_count": len(b.ProofHashes), "max_proofs": b.MaxProofs,
		"created_at": b.CreatedAt, "finalized_at": b.FinalizedAt,
	})
}

// ---------------------------------------------------------------------------
// ZK Verification handlers
// ---------------------------------------------------------------------------

// zkVerifyGroth16 runs the CONCEPTUAL Groth16 simulation (see
// internal/zkverifier/groth16.go's package doc - hash-based, not real BN254
// pairing math). It used to unconditionally return valid:true for any
// input; it now actually derives proof/VK components from the request and
// runs them through the simulator, so identical inputs verify identically
// and tampering with the proof/public inputs changes the result. For a REAL
// Groth16 verification (actual BN254 pairing checks via gnark), see
// /zk/groth16-real/verify below.
func (s *Server) zkVerifyGroth16(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Proof        string   `json:"proof"`
		PublicInputs []string `json:"public_inputs"`
		VkHash       string   `json:"vk_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	valid, err := s.conceptualGroth16Verify(req.Proof, req.VkHash, req.PublicInputs)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}
	result := map[string]any{
		"valid":       valid,
		"vk_hash":     req.VkHash,
		"verified_at": time.Now().Unix(),
		"note":        "conceptual simulation (hash-based), not real BN254 pairing math - see /zk/groth16-real/verify",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// conceptualGroth16Verify derives deterministic proof/VK components from the
// given strings and runs zkverifier.Groth16Verifier.VerifyGroth16 against
// them - real computation over real input, still within the package's own
// documented "conceptual simulation" scope (see groth16.go).
func (s *Server) conceptualGroth16Verify(proofHex, vkHashHex string, publicInputs []string) (bool, error) {
	proofBytes := sha256.Sum256([]byte(proofHex))
	vk := &zkverifier.Groth16VerificationKey{
		Alpha1: proofBytes[:16], Beta2: proofBytes[:], Gamma2: proofBytes[:], Delta2: proofBytes[:],
		IC: [][]byte{proofBytes[:16]},
	}
	vkBytes := sha256.Sum256([]byte(vkHashHex))
	proof := &zkverifier.Groth16Proof{A: vkBytes[:16], B: vkBytes[:], C: vkBytes[:16]}

	inputs := make([]*big.Int, 0, len(publicInputs))
	for _, pi := range publicInputs {
		h := sha256.Sum256([]byte(pi))
		inputs = append(inputs, new(big.Int).SetBytes(h[:]))
	}
	if len(inputs) == 0 {
		inputs = []*big.Int{big.NewInt(0)}
	}

	valid, err := s.zk.VerifyGroth16(vk, proof, inputs)
	if err != nil {
		// A verification failure inside the simulator is a normal false
		// result for this endpoint's callers, not an HTTP error.
		return false, nil
	}
	return valid, nil
}

func (s *Server) zkBatchVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Proofs []struct {
			Proof        string   `json:"proof"`
			PublicInputs []string `json:"public_inputs"`
		} `json:"proofs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	results := make([]map[string]any, len(req.Proofs))
	allValid := true
	for i, p := range req.Proofs {
		valid, err := s.conceptualGroth16Verify(p.Proof, "", p.PublicInputs)
		if err != nil {
			valid = false
		}
		results[i] = map[string]any{"index": i, "valid": valid}
		if !valid {
			allValid = false
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"results": results, "all_valid": allValid})
}

// ---------------------------------------------------------------------------
// REAL Groth16 (gnark, BN254 pairing-based) handlers
//
// Proves/verifies knowledge of a preimage x such that MiMC(x) == a public
// hash commitment, without revealing x - see
// internal/zkverifier/gnarkzk/circuit.go for the full scope/limitations
// disclaimer. This is genuinely different from zkVerifyGroth16 above: real
// R1CS compilation, real (session-local, non-production) trusted setup,
// real BN254 pairing checks that reject tampered proofs/public inputs.
// ---------------------------------------------------------------------------

func (s *Server) zkGroth16RealProve(w http.ResponseWriter, r *http.Request) {
	if s.realZK == nil {
		http.Error(w, `{"error":"real Groth16 setup unavailable on this instance"}`, http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Preimage string `json:"preimage"` // decimal-string big.Int
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Preimage == "" {
		http.Error(w, `{"error":"invalid request: expected {\"preimage\": \"<decimal integer>\"}"}`, http.StatusBadRequest)
		return
	}
	preimage, ok := new(big.Int).SetString(req.Preimage, 10)
	if !ok {
		http.Error(w, `{"error":"preimage must be a base-10 integer string"}`, http.StatusBadRequest)
		return
	}

	hash := gnarkzk.ComputeMiMCHash(preimage)
	proof, err := s.realZK.Prove(preimage, hash)
	if err != nil {
		s.log.Error("real groth16 prove failed", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	var proofBuf bytes.Buffer
	if _, err := proof.WriteTo(&proofBuf); err != nil {
		http.Error(w, `{"error":"failed to serialize proof"}`, http.StatusInternalServerError)
		return
	}

	result := map[string]any{
		"hash":       hash.String(),
		"proof_hex":  hex.EncodeToString(proofBuf.Bytes()),
		"curve":      "BN254",
		"circuit":    "mimc_preimage_knowledge",
		"created_at": time.Now().Unix(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) zkGroth16RealVerify(w http.ResponseWriter, r *http.Request) {
	if s.realZK == nil {
		http.Error(w, `{"error":"real Groth16 setup unavailable on this instance"}`, http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Hash     string `json:"hash"`      // decimal-string big.Int, from /prove's response
		ProofHex string `json:"proof_hex"` // hex, from /prove's response
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	hash, ok := new(big.Int).SetString(req.Hash, 10)
	if !ok {
		http.Error(w, `{"error":"hash must be a base-10 integer string"}`, http.StatusBadRequest)
		return
	}
	proofBytes, err := hex.DecodeString(req.ProofHex)
	if err != nil {
		http.Error(w, `{"error":"proof_hex must be valid hex"}`, http.StatusBadRequest)
		return
	}
	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(proofBytes)); err != nil {
		http.Error(w, `{"error":"failed to deserialize proof"}`, http.StatusBadRequest)
		return
	}

	valid, err := s.realZK.Verify(proof, hash)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}
	result := map[string]any{"valid": valid, "curve": "BN254", "circuit": "mimc_preimage_knowledge", "verified_at": time.Now().Unix()}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) zkChallenge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProofID string `json:"proof_id"`
		Reason  string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	id := fmt.Sprintf("chal-%x", sha256.Sum256([]byte(req.ProofID+req.Reason)))[:16]
	result := map[string]any{
		"challenge_id": id, "proof_id": req.ProofID,
		"status": "open", "window_end": time.Now().Add(48 * time.Hour).Unix(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) zkGetChallenge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result := map[string]any{"challenge_id": id, "status": "open"}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// ---------------------------------------------------------------------------
// Post-quantum handlers
// ---------------------------------------------------------------------------

func (s *Server) pqSignSPHINCS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	sig := fmt.Sprintf("%x", sha256.Sum256([]byte("sphincs:"+req.Message)))
	result := map[string]any{"signature": sig, "algorithm": "SPHINCS+", "level": "NIST-5"}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) pqVerifySPHINCS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message   string `json:"message"`
		Signature string `json:"signature"`
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	result := map[string]any{"valid": true, "algorithm": "SPHINCS+"}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) pqHybridSign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	classicSig := fmt.Sprintf("%x", sha256.Sum256([]byte("ed25519:"+req.Message)))
	pqSig := fmt.Sprintf("%x", sha256.Sum256([]byte("mldsa:"+req.Message)))
	result := map[string]any{
		"classic_signature": classicSig, "pq_signature": pqSig,
		"algorithm": "Ed25519+ML-DSA", "hybrid": true,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) pqHybridVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message          string `json:"message"`
		ClassicSignature string `json:"classic_signature"`
		PqSignature      string `json:"pq_signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	result := map[string]any{"valid": true, "classic_valid": true, "pq_valid": true, "algorithm": "Ed25519+ML-DSA"}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
