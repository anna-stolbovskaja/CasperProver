package mlattest

// ML attestation harness.
//
// This package exposes the interface an ML-attestation surface would use,
// with a hash-based default implementation. It DELIBERATELY does not
// claim to be a cryptographic proof of ML inference. Every claim in the
// CasperProver tree that implies a cryptographic proof of a model's
// inference — as opposed to an attestation of inputs, outputs, and a
// model identifier — is labelled `SIMULATION`. See
// `docs/ZKML_HONEST_VERDICT.md` for the durable decision record and the
// four conditions that must hold before any relabel to `REAL` is
// authorised.
//
// This harness sits inside the boundary that the honest verdict permits:
// it exposes only attestation semantics ("this model id, over these
// inputs, produced this output, signed as a SHA-256 chain"), never
// inference proof semantics. Downstream code that wants a real ZK-ML
// proof must match on a specific scheme label — not on the presence of
// this type.
//
// The public envelope labels itself `"scheme":"ml-attest-v0"` so no
// downstream consumer can silently mistake it for a cryptographic ML
// inference proof.
//
// The design intentionally mirrors internal/aggregator (Nova harness):
// same disclosure discipline, same "refuse unknown scheme" verify, same
// roadmap gating for a future real implementation.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// AttestationScheme is the label reported inside Attestation.Scheme.
type AttestationScheme string

const (
	// SchemeMLAttestV0 is the hash-only attestation stand-in.
	// Genuinely deterministic, genuinely verifiable — but ONLY as an
	// attestation of the inputs/outputs/model-id triple. Not a proof
	// that the named model was actually executed on the named inputs.
	SchemeMLAttestV0 AttestationScheme = "ml-attest-v0"

	// SchemeZKMLFixedV0 is RESERVED for a future named-circuit ZK-ML
	// implementation. It MUST NOT be emitted by any code in this
	// package until all four gating conditions in
	// docs/ZKML_HONEST_VERDICT.md are met. Verify() refuses this label.
	SchemeZKMLFixedV0 AttestationScheme = "zkml-fixed-v0"
)

// Attestation is the public envelope. Inputs, outputs and the model
// identifier are hex-encoded byte strings the caller supplies; the
// attestor commits to their hashes but does not interpret them.
type Attestation struct {
	Scheme        AttestationScheme `json:"scheme"`
	ModelID       string            `json:"model_id"`
	WeightsDigest string            `json:"weights_digest_hex"` // SHA-256 hex of weights
	InputsDigest  string            `json:"inputs_digest_hex"`  // SHA-256 hex of inputs
	OutputsDigest string            `json:"outputs_digest_hex"` // SHA-256 hex of outputs
	Commit        string            `json:"commit_hex"`         // final chain commitment
	Disclosure    string            `json:"disclosure"`
}

// DisclosureText is embedded in every emitted Attestation so a consumer
// reading the JSON envelope alone cannot mistake it for a cryptographic
// ML inference proof.
const DisclosureText = "This is an attestation of (model_id, weights_digest, inputs_digest, outputs_digest) — NOT a cryptographic proof that the named model was executed on the named inputs. See docs/ZKML_HONEST_VERDICT.md."

// Attestor is the pluggable interface. A future real ZK-ML implementation
// would implement this on top of a fixed named circuit and expose the
// same signatures, but under a distinct scheme label.
type Attestor interface {
	Attest(input AttestInput) (Attestation, error)
	Verify(input AttestInput, att Attestation) (bool, error)
}

// AttestInput is the raw material the attestor commits to. All four
// fields are required; empty strings are rejected.
type AttestInput struct {
	ModelID       string // opaque identifier ("mnist-mlp-8x8-v0")
	WeightsDigest []byte // SHA-256 of the weights blob
	InputsDigest  []byte // SHA-256 of the input tensor
	OutputsDigest []byte // SHA-256 of the output tensor
}

// HashMLAttestor is the hash-only stand-in. Computes:
//
//	step_a = SHA256( model_id || weights_digest )
//	step_b = SHA256( inputs_digest || outputs_digest )
//	commit = SHA256( domain_seed || step_a || step_b )
//
// where domain_seed = SHA256("ml-attest-v0") so future schemes do not
// collide with this one on identical inputs.
type HashMLAttestor struct{}

// NewHashMLAttestor returns a stateless attestor. Safe for concurrent use.
func NewHashMLAttestor() *HashMLAttestor { return &HashMLAttestor{} }

func (h *HashMLAttestor) validate(in AttestInput) error {
	if strings.TrimSpace(in.ModelID) == "" {
		return errors.New("mlattest: model_id is required")
	}
	if len(in.WeightsDigest) != sha256.Size {
		return fmt.Errorf("mlattest: weights_digest must be %d bytes", sha256.Size)
	}
	if len(in.InputsDigest) != sha256.Size {
		return fmt.Errorf("mlattest: inputs_digest must be %d bytes", sha256.Size)
	}
	if len(in.OutputsDigest) != sha256.Size {
		return fmt.Errorf("mlattest: outputs_digest must be %d bytes", sha256.Size)
	}
	return nil
}

func (h *HashMLAttestor) commit(in AttestInput) [sha256.Size]byte {
	stepA := sha256.Sum256(append([]byte(in.ModelID), in.WeightsDigest...))
	stepB := sha256.Sum256(append(append([]byte{}, in.InputsDigest...), in.OutputsDigest...))
	seed := sha256.Sum256([]byte(SchemeMLAttestV0))
	h256 := sha256.New()
	h256.Write(seed[:])
	h256.Write(stepA[:])
	h256.Write(stepB[:])
	var out [sha256.Size]byte
	copy(out[:], h256.Sum(nil))
	return out
}

// Attest produces the envelope. Deterministic — same input → same commit.
func (h *HashMLAttestor) Attest(in AttestInput) (Attestation, error) {
	if err := h.validate(in); err != nil {
		return Attestation{}, err
	}
	c := h.commit(in)
	return Attestation{
		Scheme:        SchemeMLAttestV0,
		ModelID:       in.ModelID,
		WeightsDigest: hex.EncodeToString(in.WeightsDigest),
		InputsDigest:  hex.EncodeToString(in.InputsDigest),
		OutputsDigest: hex.EncodeToString(in.OutputsDigest),
		Commit:        hex.EncodeToString(c[:]),
		Disclosure:    DisclosureText,
	}, nil
}

// Verify recomputes the chain from the input and compares the commit.
//
// Returns (false, error) on:
//   - unknown/reserved scheme label (zkml-fixed-v0 is deliberately rejected here)
//   - digest length mismatch
//   - hex decode error
//   - commit mismatch (tampering)
//
// Returns (true, nil) iff the recomputed commit equals att.Commit AND
// every hex-encoded digest in att equals the caller-supplied digest.
func (h *HashMLAttestor) Verify(in AttestInput, att Attestation) (bool, error) {
	if att.Scheme != SchemeMLAttestV0 {
		return false, fmt.Errorf("mlattest: unsupported scheme %q (this harness only verifies %q)", att.Scheme, SchemeMLAttestV0)
	}
	if err := h.validate(in); err != nil {
		return false, err
	}
	// Cross-check that the envelope's own digests match the caller's.
	if hex.EncodeToString(in.WeightsDigest) != att.WeightsDigest {
		return false, errors.New("mlattest: weights_digest envelope mismatch")
	}
	if hex.EncodeToString(in.InputsDigest) != att.InputsDigest {
		return false, errors.New("mlattest: inputs_digest envelope mismatch")
	}
	if hex.EncodeToString(in.OutputsDigest) != att.OutputsDigest {
		return false, errors.New("mlattest: outputs_digest envelope mismatch")
	}
	if in.ModelID != att.ModelID {
		return false, errors.New("mlattest: model_id envelope mismatch")
	}
	got := h.commit(in)
	if hex.EncodeToString(got[:]) != att.Commit {
		return false, errors.New("mlattest: commit mismatch — attestation tampered or scheme divergent")
	}
	return true, nil
}

// AttestAll is a stateless convenience for HTTP handlers.
func AttestAll(in AttestInput) (Attestation, error) {
	return NewHashMLAttestor().Attest(in)
}

// VerifyAll is the stateless counterpart to AttestAll.
func VerifyAll(in AttestInput, att Attestation) (bool, error) {
	return NewHashMLAttestor().Verify(in, att)
}
