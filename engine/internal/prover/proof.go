package prover

import (
	"fmt"
	"sync"
	"time"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/hasher"
)

type ProofEngine struct {
	mu     sync.RWMutex
	proofs map[string]*Proof
	seq    int
}

func New() *ProofEngine {
	return &ProofEngine{proofs: make(map[string]*Proof)}
}

func (e *ProofEngine) Generate(agent string, input, output, model []byte, uc string) *Proof {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.seq++
	pid := fmt.Sprintf("P-%d", e.seq)
	ph := hasher.CommitHash(input, output, model)
	ih := hasher.HexHash(input)
	oh := hasher.HexHash(output)
	mh := hasher.HexHash(model)

	leaves := [][]byte{input, output, model}
	root := Root(leaves)
	path := GetPath(leaves, 0)

	p := &Proof{
		ID:      pid,
		Agent:   agent,
		PH:      ph,
		IH:      ih,
		OH:      oh,
		MH:      mh,
		Root:    root,
		Path:    path,
		Idx:     0,
		TS:      time.Now().Unix(),
		Valid:   true,
		UseCase: uc,
	}
	e.proofs[pid] = p
	return p
}

func (e *ProofEngine) Get(pid string) (*Proof, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	p, ok := e.proofs[pid]
	return p, ok
}

func (e *ProofEngine) Revoke(pid, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, ok := e.proofs[pid]
	if !ok {
		return fmt.Errorf("proof %s not found", pid)
	}
	if p.Revoked {
		return fmt.Errorf("proof %s already revoked", pid)
	}
	p.Valid = false
	p.Revoked = true
	return nil
}

func (e *ProofEngine) Verify(pid string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	p, ok := e.proofs[pid]
	if !ok {
		return false, fmt.Errorf("proof %s not found", pid)
	}
	return p.Valid && !p.Revoked, nil
}

func (e *ProofEngine) List() []*Proof {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*Proof, 0, len(e.proofs))
	for _, p := range e.proofs {
		out = append(out, p)
	}
	return out
}
