package submitter

import (
	"os"
	"testing"
)

// These tests exercise the env-var contract of the zk-verifier submitters
// without hitting a live Casper RPC endpoint. Missing CONTRACT_ZK_VERIFIER
// must fail fast with a clear message - this is the guardrail that stops
// a mis-configured engine from silently NOT anchoring verdicts.

func TestRegisterZkVk_MissingEnvFails(t *testing.T) {
	os.Unsetenv("CONTRACT_ZK_VERIFIER")
	s := &CasperSubmitter{}
	_, err := s.RegisterZkVk("mimc_v1", "aa", "bn254", "groth16", 0)
	if err == nil {
		t.Fatal("expected error when CONTRACT_ZK_VERIFIER unset")
	}
	if err.Error() != "CONTRACT_ZK_VERIFIER env var not set" {
		t.Fatalf("unexpected err message: %v", err)
	}
}

func TestRecordZkVerdict_MissingEnvFails(t *testing.T) {
	os.Unsetenv("CONTRACT_ZK_VERIFIER")
	s := &CasperSubmitter{}
	_, err := s.RecordZkVerdict("mimc_v1", "aa", "bb", "gpt-4o-mini", true)
	if err == nil {
		t.Fatal("expected error when CONTRACT_ZK_VERIFIER unset")
	}
}

func TestAddZkVerifier_MissingEnvFails(t *testing.T) {
	os.Unsetenv("CONTRACT_ZK_VERIFIER")
	s := &CasperSubmitter{}
	_, err := s.AddZkVerifier("00000000000000000000000000000000000000000000000000000000deadbeef")
	if err == nil {
		t.Fatal("expected error when CONTRACT_ZK_VERIFIER unset")
	}
}

func TestAddZkVerifier_BadAccountHash(t *testing.T) {
	os.Setenv("CONTRACT_ZK_VERIFIER", "00")
	defer os.Unsetenv("CONTRACT_ZK_VERIFIER")
	s := &CasperSubmitter{}
	_, err := s.AddZkVerifier("not-hex")
	if err == nil {
		t.Fatal("expected error on malformed account hash")
	}
}
