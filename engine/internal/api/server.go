package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/hasher"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/kyc"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/store"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/verifier"
)

type Server struct {
	eng   *prover.ProofEngine
	ver   *verifier.LocalVerifier
	kyc   *kyc.DemoKYC
	db    *store.PG
	port  int
	log   *slog.Logger
	start time.Time
}

func New(eng *prover.ProofEngine, port int, db *store.PG) *Server {
	return &Server{
		eng:   eng,
		ver:   verifier.New(),
		kyc:   kyc.NewDemo(eng),
		db:    db,
		port:  port,
		log:   slog.Default(),
		start: time.Now(),
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

	addr := fmt.Sprintf(":%d", s.port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.corsMiddleware(s.logMiddleware(mux)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	s.log.Info("api starting", "addr", addr)
	return srv.ListenAndServe()
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
		p.Deploy = hasher.HexHash([]byte(p.Root + p.ID))
	}

	s.persist(p)
	s.log.Info("proof generated", "id", p.ID, "agent", req.Agent, "use_case", req.UseCase, "mode", mode, "ms", p.GenMs)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

func (s *Server) batchProofs(w http.ResponseWriter, r *http.Request) {
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
			p.Deploy = hasher.HexHash([]byte(p.Root + p.ID))
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
		"version":  "1.0",
		"exported": time.Now().Unix(),
		"proof":    p,
		"contract": "96e97c4d564fe7374ba4e938355fb89f5be2f448decbe9b7727bd3c978a10708",
		"chain":    "casper-test",
		"verify_url": fmt.Sprintf("https://casperprover-api.onrender.com/verify"),
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
