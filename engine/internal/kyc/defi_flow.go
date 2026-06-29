package kyc

import (
	"fmt"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
)

type DeFiFlow struct {
	eng  *prover.ProofEngine
	demo *DemoKYC
}

func NewDeFiFlow(eng *prover.ProofEngine) *DeFiFlow {
	return &DeFiFlow{eng: eng, demo: NewDemo(eng)}
}

func (f *DeFiFlow) RunDemo(agentID string) error {
	input := []byte(`{"user":"alice","doc_type":"passport"}`)
	output := []byte(`{"verified":true,"score":95}`)
	model := []byte("kyc-model-v1")

	p := f.eng.Generate(agentID, input, output, model, "kyc")
	fmt.Printf("proof generated: %s\n", p.ID)

	_, err := f.demo.GrantAccess("alice", p.ID)
	if err != nil {
		return fmt.Errorf("grant: %w", err)
	}

	ok := f.demo.IsWhitelisted("alice")
	fmt.Printf("alice whitelisted: %v\n", ok)
	return nil
}
