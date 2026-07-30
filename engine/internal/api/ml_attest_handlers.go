package api

// ML attestation HTTP surface — hash-only stand-in.
//
// Endpoints:
//
//   POST /v1/ml/attest         — commit to (model_id, weights, inputs, outputs)
//   POST /v1/ml/verify-attest  — recompute and check an Attestation
//
// EVERY emitted response labels its scheme as "ml-attest-v0" and carries
// a disclosure string. This is NOT a cryptographic proof of ML inference.
// See docs/ZKML_HONEST_VERDICT.md for the durable decision record and
// docs/roadmap/ML_ATTESTATION_HARNESS.md for the disclosure gate that
// blocks any relabel to a real ZK-ML scheme.

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/mlattest"
)

type mlAttestDTO struct {
	ModelID       string `json:"model_id"`
	WeightsDigest string `json:"weights_digest_hex"`
	InputsDigest  string `json:"inputs_digest_hex"`
	OutputsDigest string `json:"outputs_digest_hex"`
}

func decodeMLAttestInput(d mlAttestDTO) (mlattest.AttestInput, error) {
	if d.ModelID == "" {
		return mlattest.AttestInput{}, errors.New("model_id is required")
	}
	w, err := hex.DecodeString(d.WeightsDigest)
	if err != nil {
		return mlattest.AttestInput{}, errors.New("weights_digest_hex must be hex-encoded SHA-256")
	}
	i, err := hex.DecodeString(d.InputsDigest)
	if err != nil {
		return mlattest.AttestInput{}, errors.New("inputs_digest_hex must be hex-encoded SHA-256")
	}
	o, err := hex.DecodeString(d.OutputsDigest)
	if err != nil {
		return mlattest.AttestInput{}, errors.New("outputs_digest_hex must be hex-encoded SHA-256")
	}
	return mlattest.AttestInput{
		ModelID:       d.ModelID,
		WeightsDigest: w,
		InputsDigest:  i,
		OutputsDigest: o,
	}, nil
}

// mlAttest emits a ml-attest-v0 Attestation.
//
// The optional "scheme" field, if present, must equal "ml-attest-v0".
// Any other value (including the reserved "zkml-fixed-v0") is refused
// with 400 — this is the gate against silent laundering of a hash
// attestation as a ZK-ML proof.
func (s *Server) mlAttest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scheme string      `json:"scheme,omitempty"`
		Input  mlAttestDTO `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	scheme := mlattest.AttestationScheme(req.Scheme)
	if scheme == "" {
		scheme = mlattest.SchemeMLAttestV0
	}
	if scheme != mlattest.SchemeMLAttestV0 {
		s.jsonError(w, "unsupported scheme: "+string(scheme)+" (this endpoint only emits ml-attest-v0; ZK-ML circuits are gated by docs/ZKML_HONEST_VERDICT.md)", http.StatusBadRequest)
		return
	}
	in, err := decodeMLAttestInput(req.Input)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	att, err := mlattest.AttestAll(in)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(att)
}

// mlVerifyAttest recomputes the commit from the supplied input and
// compares it against the envelope. Returns {"valid": true|false,
// "scheme": "...", "error"?: "..."}.
func (s *Server) mlVerifyAttest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input       mlAttestDTO          `json:"input"`
		Attestation mlattest.Attestation `json:"attestation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	in, err := decodeMLAttestInput(req.Input)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	valid, verr := mlattest.VerifyAll(in, req.Attestation)
	resp := map[string]any{
		"valid":  valid,
		"scheme": string(req.Attestation.Scheme),
	}
	if verr != nil {
		resp["error"] = verr.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) registerMLAttestRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/ml/attest", s.mlAttest)
	mux.HandleFunc("POST /v1/ml/verify-attest", s.mlVerifyAttest)
}
