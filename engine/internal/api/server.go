package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/verifier"
)

type Server struct {
	eng  *prover.ProofEngine
	ver  *verifier.LocalVerifier
	port int
}

func New(eng *prover.ProofEngine, port int) *Server {
	return &Server{eng: eng, ver: verifier.New(), port: port}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /proofs", s.listProofs)
	mux.HandleFunc("GET /proofs/{id}", s.getProof)
	mux.HandleFunc("POST /proofs", s.submitProof)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("api listening on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "0.1.0"})
}

func (s *Server) listProofs(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(s.eng.List())
}

func (s *Server) getProof(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("id")
	p, ok := s.eng.Get(pid)
	if !ok {
		http.Error(w, "not found", 404)
		return
	}
	json.NewEncoder(w).Encode(p)
}

func (s *Server) submitProof(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Agent   string `json:"agent"`
		Input   string `json:"input"`
		Output  string `json:"output"`
		Model   string `json:"model"`
		UseCase string `json:"use_case"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	p := s.eng.Generate(req.Agent, []byte(req.Input), []byte(req.Output), []byte(req.Model), req.UseCase)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(p)
}
