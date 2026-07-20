package api

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/aggregator"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/config"
	pqcrypto "github.com/anna-stolbovskaja/CasperProver/engine/internal/crypto"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/hasher"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/inference"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/kyc"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/store"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/submitter"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/verifier"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/zkverifier"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/zkverifier/gnarkzk"
	"github.com/anna-stolbovskaja/CasperProver/engine/pkg/phase2"
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
)

// contractHashes holds on-chain contract hashes, configurable via env vars
// so redeployed contracts don't require a code change.
type contractHashes struct {
	ProofRegistry string
	VerifierGate  string
	DefiMock      string
	StakeSlashing string
}

type Server struct {
	eng       *prover.ProofEngine
	ver       *verifier.LocalVerifier
	kyc       *kyc.DemoKYC
	db        *store.PG
	sub       *submitter.CasperSubmitter
	inf       *inference.InferenceService
	zk        *zkverifier.Groth16Verifier
	realZK    *gnarkzk.Setup
	contracts contractHashes
	port      int
	log       *slog.Logger
	start     time.Time
	apiKey    string
	strict    bool // fail closed for requested on-chain operations

	aggMu      sync.Mutex
	aggBatches map[string]*aggBatch
}

// aggBatch tracks per-batch state for the /aggregation/* endpoints.
// Batches are persisted to Postgres (aggregation_batches table) and
// rehydrated on server startup. The aggregation uses hash-chain
// verification via internal/aggregator.
type aggBatch struct {
	ID          string
	MerkleRoot  string
	MaxProofs   int
	ProofHashes []string
	Status      string // "open" | "finalized"
	CreatedAt   int64
	FinalizedAt int64

	// Pack is populated on finalize by running the batch's proof hashes
	// through the real internal/aggregator (still a conceptual/hash-based
	// simulation of STARK aggregation, not real STARK math - see that
	// package's doc comment - but genuinely wired in and genuinely
	// verifiable, not just bookkeeping).
	Pack *aggregator.STARKPack
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ValidateStartupConfig refuses unsafe strict-mode API configuration.
func ValidateStartupConfig() error {
	if os.Getenv("CP_STRICT") == "1" && os.Getenv("API_KEY") == "" {
		return fmt.Errorf("CP_STRICT=1 requires API_KEY; refusing unauthenticated startup")
	}
	return nil
}

// manifestHashOrEnv returns env override first, then the canonical manifest,
// then an empty string. A hardcoded fallback here would silently outlive a
// redeploy (Gate 1.5 forbids it).
func manifestHashOrEnv(envKey, manifestKey string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	h, err := config.ContractHash(manifestKey)
	if err != nil {
		slog.Warn("onchain manifest not readable; contract hash will be empty until redeploy manifest is provisioned", "env_key", envKey, "manifest_key", manifestKey, "err", err)
		return ""
	}
	return h
}

func New(eng *prover.ProofEngine, port int, db *store.PG) *Server {
	contracts := contractHashes{
		ProofRegistry: manifestHashOrEnv("CONTRACT_PROOF_REGISTRY", "proof_registry"),
		VerifierGate:  manifestHashOrEnv("CONTRACT_VERIFIER_GATE", "verifier_gate"),
		DefiMock:      manifestHashOrEnv("CONTRACT_DEFI_MOCK", "defi_mock"),
		StakeSlashing: manifestHashOrEnv("CONTRACT_STAKE_SLASHING", "stake_slashing"),
	}

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

	strict := os.Getenv("CP_STRICT") == "1"
	apiKey := os.Getenv("API_KEY")
	// ValidateStartupConfig should have been called by main.go before us; if a
	// caller ignored it we still warn but never silently accept the unsafe mix.
	if apiKey == "" {
		if strict {
			slog.Error("CP_STRICT=1 with empty API_KEY - health will report degraded; writes remain unauthenticated")
		} else {
			slog.Warn("API_KEY not set - all write endpoints are unauthenticated (fine for local dev/demo, not for a real deployment)")
		}
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

	srv := &Server{
		eng:       eng,
		ver:       verifier.New(),
		kyc:       demoKYC,
		db:        db,
		sub:       sub,
		inf:       inference.New(eng, db, sub),
		zk:        zkverifier.NewGroth16Verifier(),
		realZK:    realZK,
		contracts: contracts,
		port:      port,
		log:       slog.Default(),
		start:     time.Now(),
		apiKey:    apiKey,
		strict:    strict,

		aggBatches: make(map[string]*aggBatch),
	}

	// Rehydrate aggregation batches from Postgres
	if db != nil {
		rows, err := db.LoadAggBatches()
		if err != nil {
			slog.Warn("failed to load aggregation batches from db", "err", err)
		} else {
			for _, row := range rows {
				b := &aggBatch{
					ID: row.BatchID, MerkleRoot: row.MerkleRoot, MaxProofs: row.MaxProofs,
					ProofHashes: row.ProofHashes, Status: row.Status,
					CreatedAt: row.CreatedAt, FinalizedAt: row.FinalizedAt,
				}
				if row.AggregateProofHash != "" {
					b.Pack = &aggregator.STARKPack{
						AggregateProofHash:    row.AggregateProofHash,
						IndividualProofHashes: row.IndividualProofHashes,
						ProofCount:            row.ProofCount,
						Timestamp:             row.FinalizedAt,
					}
				}
				srv.aggBatches[row.BatchID] = b
			}
			slog.Info("loaded aggregation batches from postgres", "count", len(rows))
		}
	}

	return srv
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
	mux.HandleFunc("GET /aggregation/verify-batch/{id}", s.aggregationVerifyBatch)
	// ZK Verification routes
	mux.HandleFunc("POST /zk/verify-groth16", s.zkVerifyGroth16)
	mux.HandleFunc("POST /zk/batch-verify", s.zkBatchVerify)
	mux.HandleFunc("POST /zk/groth16-real/prove", s.zkGroth16RealProve)
	mux.HandleFunc("POST /zk/groth16-real/verify", s.zkGroth16RealVerify)
	mux.HandleFunc("POST /zk/challenge", s.zkChallenge)
	mux.HandleFunc("GET /zk/challenge/{id}", s.zkGetChallenge)
	// Phase 2: proof chains (DAG validation)
	mux.HandleFunc("POST /proof-chain/validate", s.proofChainValidate)
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
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip == "" {
			ip = r.RemoteAddr
		}
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
				s.jsonError(w, "too many requests", http.StatusTooManyRequests)
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Public-Key, X-API-Key")
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

	// Compute status: OK when we can serve the mode we advertise; degraded when
	// strict mode is on but a strict-required capability is missing (no
	// authenticated writes, no on-chain submitter). Never fails the endpoint —
	// operators still need to reach /health to diagnose.
	degradedReasons := make([]string, 0, 2)
	if s.strict {
		if s.apiKey == "" {
			degradedReasons = append(degradedReasons, "strict_mode_without_api_key")
		}
		if s.sub == nil {
			degradedReasons = append(degradedReasons, "strict_mode_without_onchain_submitter")
		}
	}
	status := "ok"
	if len(degradedReasons) > 0 {
		status = "degraded"
	}

	resp := map[string]interface{}{
		"status":       status,
		"version":      "0.2.0",
		"uptime_s":     int(time.Since(s.start).Seconds()),
		"total_proofs": st.Total,
		"chain":        "casper-test",
		"strict":       s.strict,
		"capabilities": map[string]bool{
			"authenticated_writes": s.apiKey != "",
			"onchain_submit":       s.sub != nil,
			"real_groth16":         s.realZK != nil,
		},
		"contracts": map[string]string{
			"proof_registry": s.contracts.ProofRegistry,
			"verifier_gate":  s.contracts.VerifierGate,
			"defi_mock":      s.contracts.DefiMock,
			"stake_slashing": s.contracts.StakeSlashing,
		},
	}
	if len(degradedReasons) > 0 {
		resp["degraded_reasons"] = degradedReasons
	}
	_ = json.NewEncoder(w).Encode(resp)
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
	if len(req.Agent) > 128 || len(req.Input) > 10240 || len(req.Output) > 10240 || len(req.Model) > 256 {
		s.jsonError(w, "field exceeds max length", http.StatusBadRequest)
		return
	}

	pubKey := r.Header.Get("X-Public-Key")
	mode := req.Mode
	if mode == "" {
		mode = "local"
	}

	if mode == "anchored" && s.strict && s.sub == nil {
		s.jsonError(w, "anchored mode unavailable: deployer key is not configured", http.StatusServiceUnavailable)
		return
	}
	p := s.eng.GenerateWithKey(req.Agent, pubKey, []byte(req.Input), []byte(req.Output), []byte(req.Model), req.UseCase, mode)

	if mode == "anchored" {
		if s.sub == nil {
			p.AnchorHash = hasher.HexHash([]byte(p.Root + p.ID))
			p.AnchoringStatus = "computed_fallback"
		} else {
			deployHash, err := s.sub.Submit(p)
			if err != nil {
				if s.strict {
					s.log.Error("strict on-chain submit failed", "id", p.ID, "err", err)
					s.jsonError(w, "on-chain submission failed", http.StatusBadGateway)
					return
				}
				s.log.Warn("on-chain submit failed, recording computed fallback", "id", p.ID, "err", err)
				p.AnchorHash = hasher.HexHash([]byte(p.Root + p.ID))
				p.AnchoringStatus = "computed_fallback"
			} else {
				p.Deploy = deployHash
				p.AnchorHash = deployHash
				p.AnchoringStatus = "submitted"
				s.log.Info("proof submitted on-chain", "id", p.ID, "deploy", deployHash)
			}
		}
	} else {
		p.AnchoringStatus = "local"
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

	if mode == "anchored" && s.strict && s.sub == nil {
		s.jsonError(w, "anchored mode unavailable: deployer key is not configured", http.StatusServiceUnavailable)
		return
	}
	results := make([]*prover.Proof, 0, len(req.Proofs))
	for _, pr := range req.Proofs {
		if pr.Agent == "" || pr.Input == "" || pr.Output == "" || pr.Model == "" {
			continue
		}
		p := s.eng.GenerateWithKey(pr.Agent, pubKey, []byte(pr.Input), []byte(pr.Output), []byte(pr.Model), pr.UseCase, mode)
		if mode == "anchored" {
			if s.sub == nil {
				p.AnchorHash = hasher.HexHash([]byte(p.Root + p.ID))
				p.AnchoringStatus = "computed_fallback"
			} else {
				dh, err := s.sub.Submit(p)
				if err != nil {
					if s.strict {
						s.jsonError(w, "on-chain submission failed", http.StatusBadGateway)
						return
					}
					p.AnchorHash = hasher.HexHash([]byte(p.Root + p.ID))
					p.AnchoringStatus = "computed_fallback"
				} else {
					p.Deploy = dh
					p.AnchorHash = dh
					p.AnchoringStatus = "submitted"
				}
			}
		} else {
			p.AnchoringStatus = "local"
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

	// Authorization: if X-Public-Key header is present, verify ownership
	pubKey := r.Header.Get("X-Public-Key")
	if pubKey != "" {
		if p, ok := s.eng.Get(pid); ok {
			if p.PubKey != "" && p.PubKey != pubKey {
				s.jsonError(w, "not authorized to revoke this proof", http.StatusForbidden)
				return
			}
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

	verifyURL := ""
	if m, err := config.Load(); err == nil {
		verifyURL = m.Verification.APIHealth
		// APIHealth ends in /health; the verify surface is a sibling.
		if verifyURL != "" {
			verifyURL = verifyURL[:len(verifyURL)-len("/health")] + "/verify"
		}
	}
	bundle := map[string]interface{}{
		"version":    "1.0",
		"exported":   time.Now().Unix(),
		"proof":      p,
		"contract":   s.contracts.ProofRegistry,
		"chain":      "casper-test",
		"verify_url": verifyURL,
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
		s.jsonError(w, "invalid request", http.StatusBadRequest)
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
		s.jsonError(w, "proof generation failed", http.StatusInternalServerError)
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
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	valid, err := s.inf.VerifyInferenceProof(r.Context(), req.ProofID)
	if err != nil {
		s.log.Warn("inference proof verification error", "proof_id", req.ProofID, "error", err)
		s.jsonError(w, err.Error(), http.StatusNotFound)
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
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	entry, err := s.inf.RegisterModel(r.Context(), req.ModelID, req.ModelHash, req.VerifierContract, req.Metadata)
	if err != nil {
		s.log.Error("model registration failed", "model_id", req.ModelID, "error", err)
		s.jsonError(w, err.Error(), http.StatusBadRequest)
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
		s.jsonError(w, "model not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entry)
}

// ---------------------------------------------------------------------------
// Aggregation handlers
// ---------------------------------------------------------------------------

// persistAggBatch saves a batch to Postgres (if DB is configured).
// Called after every mutation (create, add-proof, finalize).
func (s *Server) persistAggBatch(b *aggBatch) {
	if s.db == nil {
		return
	}
	row := &store.AggBatchRow{
		BatchID: b.ID, MaxProofs: b.MaxProofs,
		ProofHashes: b.ProofHashes, MerkleRoot: b.MerkleRoot,
		Status: b.Status, CreatedAt: b.CreatedAt, FinalizedAt: b.FinalizedAt,
	}
	if b.Pack != nil {
		row.AggregateProofHash = b.Pack.AggregateProofHash
		row.IndividualProofHashes = b.Pack.IndividualProofHashes
		row.ProofCount = b.Pack.ProofCount
	}
	if err := s.db.SaveAggBatch(context.Background(), row); err != nil {
		s.log.Warn("failed to persist agg batch", "batch_id", b.ID, "err", err)
	}
}

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
	s.persistAggBatch(b)

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
	s.persistAggBatch(b)

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
	if len(b.ProofHashes) == 0 {
		s.jsonError(w, "cannot finalize an empty batch", http.StatusBadRequest)
		return
	}

	proofBytes := make([][]byte, len(b.ProofHashes))
	for i, h := range b.ProofHashes {
		proofBytes[i] = []byte(h)
	}
	pack, err := aggregator.NewSTARKAggregator().CreateSTARKPack(proofBytes, map[string]string{"batch_id": b.ID})
	if err != nil {
		s.jsonError(w, fmt.Sprintf("aggregation failed: %v", err), http.StatusInternalServerError)
		return
	}
	b.Pack = pack
	b.Status = "finalized"
	b.FinalizedAt = time.Now().Unix()
	s.persistAggBatch(b)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"batch_id": b.ID, "status": b.Status, "proof_count": len(b.ProofHashes),
		"finalized_at": b.FinalizedAt, "aggregate_proof_hash": pack.AggregateProofHash,
	})
}

// aggregationVerifyBatch re-runs the finalized batch's proof hashes through
// internal/aggregator.UnpackAndVerify - this is a real round trip through the
// aggregator's own verification path (still hash-based/conceptual, not real
// STARK math), not a re-derivation done inline in this handler.
func (s *Server) aggregationVerifyBatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s.aggMu.Lock()
	b, ok := s.aggBatches[id]
	s.aggMu.Unlock()
	if !ok {
		s.jsonError(w, "batch not found", http.StatusNotFound)
		return
	}
	if b.Status != "finalized" || b.Pack == nil {
		s.jsonError(w, "batch is not finalized yet", http.StatusConflict)
		return
	}

	valid, err := aggregator.NewSTARKAggregator().UnpackAndVerify(b.Pack)
	w.Header().Set("Content-Type", "application/json")
	result := map[string]any{
		"batch_id": b.ID, "valid": valid,
		"aggregate_proof_hash": b.Pack.AggregateProofHash,
		"proof_count":          b.Pack.ProofCount,
	}
	if err != nil {
		result["error"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(result)
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

	resp := map[string]any{
		"batch_id": b.ID, "merkle_root": b.MerkleRoot, "status": b.Status,
		"proof_count": len(b.ProofHashes), "max_proofs": b.MaxProofs,
		"created_at": b.CreatedAt, "finalized_at": b.FinalizedAt,
	}
	if b.Pack != nil {
		resp["aggregate_proof_hash"] = b.Pack.AggregateProofHash
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
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
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	valid, err := s.conceptualGroth16Verify(req.Proof, req.VkHash, req.PublicInputs)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
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
		s.jsonError(w, "invalid request", http.StatusBadRequest)
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
		s.jsonError(w, "real Groth16 setup unavailable on this instance", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Preimage string `json:"preimage"` // decimal-string big.Int
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Preimage == "" {
		s.jsonError(w, "invalid request: expected {preimage: <decimal integer>}", http.StatusBadRequest)
		return
	}
	preimage, ok := new(big.Int).SetString(req.Preimage, 10)
	if !ok {
		s.jsonError(w, "preimage must be a base-10 integer string", http.StatusBadRequest)
		return
	}

	hash := gnarkzk.ComputeMiMCHash(preimage)
	proof, err := s.realZK.Prove(preimage, hash)
	if err != nil {
		s.log.Error("real groth16 prove failed", "error", err)
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var proofBuf bytes.Buffer
	if _, err := proof.WriteTo(&proofBuf); err != nil {
		s.jsonError(w, "failed to serialize proof", http.StatusInternalServerError)
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
		s.jsonError(w, "real Groth16 setup unavailable on this instance", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Hash     string `json:"hash"`      // decimal-string big.Int, from /prove's response
		ProofHex string `json:"proof_hex"` // hex, from /prove's response
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	hash, ok := new(big.Int).SetString(req.Hash, 10)
	if !ok {
		s.jsonError(w, "hash must be a base-10 integer string", http.StatusBadRequest)
		return
	}
	proofBytes, err := hex.DecodeString(req.ProofHex)
	if err != nil {
		s.jsonError(w, "proof_hex must be valid hex", http.StatusBadRequest)
		return
	}
	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(proofBytes)); err != nil {
		s.jsonError(w, "failed to deserialize proof", http.StatusBadRequest)
		return
	}

	valid, err := s.realZK.Verify(proof, hash)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
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
		s.jsonError(w, "invalid request", http.StatusBadRequest)
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

// pqSignSPHINCS signs the message with a freshly generated, single-use
// Lamport one-time signature key pair (see internal/crypto's package doc for
// why this stands in for SPHINCS+). The private key is deliberately not
// returned - only the public key needed to verify - since the key must never
// be reused for a second message.
func (s *Server) pqSignSPHINCS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	priv, pub, err := pqcrypto.GenerateLamportKeyPair()
	if err != nil {
		s.jsonError(w, "key generation failed", http.StatusInternalServerError)
		return
	}
	sig, err := pqcrypto.SignSPHINCS(priv, []byte(req.Message))
	if err != nil {
		s.jsonError(w, "signing failed", http.StatusInternalServerError)
		return
	}
	result := map[string]any{
		"signature":  hex.EncodeToString(sig),
		"public_key": hex.EncodeToString(pub.Bytes()),
		"algorithm":  "Lamport-OTS",
		"note":       "single-use key: do not reuse this public_key to sign another message",
	}
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
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	sigBytes, err1 := hex.DecodeString(req.Signature)
	pubBytes, err2 := hex.DecodeString(req.PublicKey)
	if err1 != nil || err2 != nil {
		s.jsonError(w, "signature and public_key must be hex-encoded", http.StatusBadRequest)
		return
	}
	pub, err := pqcrypto.LamportPublicKeyFromBytes(pubBytes)
	if err != nil {
		s.jsonError(w, "invalid public_key length", http.StatusBadRequest)
		return
	}
	valid, err := pqcrypto.VerifySPHINCS(pub, []byte(req.Message), sigBytes)
	if err != nil {
		s.jsonError(w, "invalid signature length", http.StatusBadRequest)
		return
	}
	result := map[string]any{"valid": valid, "algorithm": "Lamport-OTS"}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// pqHybridSign signs the message with a freshly generated Ed25519 key pair
// and a freshly generated real ML-DSA-65 key pair, and returns both public
// keys plus the combined hybrid signature.
func (s *Server) pqHybridSign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	classicPriv, classicPub, err := pqcrypto.GenerateEd25519KeyPair()
	if err != nil {
		s.jsonError(w, "key generation failed", http.StatusInternalServerError)
		return
	}
	pqPriv, pqPub, err := pqcrypto.GenerateMLDSAKeyPair()
	if err != nil {
		s.jsonError(w, "key generation failed", http.StatusInternalServerError)
		return
	}
	sig, err := pqcrypto.HybridSign(classicPriv, pqPriv, []byte(req.Message))
	if err != nil {
		s.jsonError(w, "signing failed", http.StatusInternalServerError)
		return
	}
	pqPubBytes, _ := pqPub.MarshalBinary()
	result := map[string]any{
		"signature":          hex.EncodeToString(sig),
		"classic_public_key": hex.EncodeToString(classicPub),
		"pq_public_key":      hex.EncodeToString(pqPubBytes),
		"algorithm":          "Ed25519+ML-DSA-65",
		"hybrid":             true,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) pqHybridVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message          string `json:"message"`
		Signature        string `json:"signature"`
		ClassicPublicKey string `json:"classic_public_key"`
		PqPublicKey      string `json:"pq_public_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	sigBytes, e1 := hex.DecodeString(req.Signature)
	classicPubBytes, e2 := hex.DecodeString(req.ClassicPublicKey)
	pqPubBytes, e3 := hex.DecodeString(req.PqPublicKey)
	if e1 != nil || e2 != nil || e3 != nil {
		s.jsonError(w, "signature and public keys must be hex-encoded", http.StatusBadRequest)
		return
	}
	var pqPub mldsa65.PublicKey
	if err := pqPub.UnmarshalBinary(pqPubBytes); err != nil {
		s.jsonError(w, "invalid pq_public_key", http.StatusBadRequest)
		return
	}
	valid, classicValid, pqValid, err := pqcrypto.HybridVerify(ed25519.PublicKey(classicPubBytes), &pqPub, []byte(req.Message), sigBytes)
	if err != nil {
		s.jsonError(w, "invalid signature format", http.StatusBadRequest)
		return
	}
	result := map[string]any{
		"valid": valid, "classic_valid": classicValid, "pq_valid": pqValid,
		"algorithm": "Ed25519+ML-DSA-65",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// ---------------------------------------------------------------------------
// Phase 2: Proof Chain (DAG) Validation
// ---------------------------------------------------------------------------

func (s *Server) proofChainValidate(w http.ResponseWriter, r *http.Request) {
	var chain phase2.ProofChain
	if err := json.NewDecoder(r.Body).Decode(&chain); err != nil {
		s.jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if len(chain.Steps) == 0 {
		s.jsonError(w, "chain must have at least one step", http.StatusBadRequest)
		return
	}

	chain.TotalSteps = len(chain.Steps)
	if chain.ID == "" {
		chain.ID = fmt.Sprintf("chain-%x", sha256.Sum256([]byte(fmt.Sprintf("%v", chain.Steps))))[:16]
	}

	if err := phase2.ValidateDAG(&chain); err != nil {
		chain.Status = phase2.ChainBroken
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"valid":       false,
			"error":       err.Error(),
			"chain_id":    chain.ID,
			"total_steps": chain.TotalSteps,
			"status":      "broken",
		})
		return
	}

	chain.Status = phase2.ChainFullyVerified
	// Find depth (longest path from root to any leaf)
	depth := 0
	index := make(map[string]*phase2.ChainStep, len(chain.Steps))
	for i := range chain.Steps {
		index[chain.Steps[i].ProofID] = &chain.Steps[i]
	}
	var calcDepth func(id string) int
	calcDepth = func(id string) int {
		step := index[id]
		if len(step.ParentIDs) == 0 {
			return 0
		}
		maxParent := 0
		for _, pid := range step.ParentIDs {
			d := calcDepth(pid)
			if d > maxParent {
				maxParent = d
			}
		}
		return maxParent + 1
	}
	for _, step := range chain.Steps {
		d := calcDepth(step.ProofID)
		if d > depth {
			depth = d
		}
	}
	chain.Depth = depth

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"valid":         true,
		"chain_id":      chain.ID,
		"total_steps":   chain.TotalSteps,
		"depth":         chain.Depth,
		"root_proof_id": chain.RootProofID,
		"status":        "fully_verified",
	})
}
