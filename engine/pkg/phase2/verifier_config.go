// Package phase2 contains infrastructure stubs for planned CasperProver features.
//
// These types define interfaces and data structures for Phase 2-4 upgrades.
// Implementation will be completed incrementally.
package phase2

// VerifierMode defines the verification strategy for proofs.
type VerifierMode int

const (
	// Optimistic mode: proof is accepted unless challenged within the dispute window.
	ModeOptimistic VerifierMode = iota
	// ZK mode: full zero-knowledge proof verification on-chain.
	ModeZK
	// Hybrid mode: optimistic by default, ZK verification on challenge.
	ModeHybrid
)

// VerifierConfig holds configuration for the proof verification system.
type VerifierConfig struct {
	Mode               VerifierMode `json:"mode"`
	DisputeWindowSecs  int64        `json:"dispute_window_secs"`
	RequireModelCommit bool         `json:"require_model_commit"`
	ZKBackend          string       `json:"zk_backend"` // "groth16", "plonk", "stark"
}

// DefaultOptimistic returns a configuration for optimistic verification.
func DefaultOptimistic() VerifierConfig {
	return VerifierConfig{
		Mode:               ModeOptimistic,
		DisputeWindowSecs:  3600, // 1 hour
		RequireModelCommit: true,
		ZKBackend:          "",
	}
}
