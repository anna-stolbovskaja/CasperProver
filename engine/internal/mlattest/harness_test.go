package mlattest

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func mkInput(t *testing.T, modelID, weights, inputs, outputs string) AttestInput {
	t.Helper()
	w := sha256.Sum256([]byte(weights))
	i := sha256.Sum256([]byte(inputs))
	o := sha256.Sum256([]byte(outputs))
	return AttestInput{
		ModelID:       modelID,
		WeightsDigest: w[:],
		InputsDigest:  i[:],
		OutputsDigest: o[:],
	}
}

func TestAttest_Deterministic(t *testing.T) {
	in := mkInput(t, "mnist-mlp-v0", "W", "X", "Y")
	a1, err := AttestAll(in)
	if err != nil {
		t.Fatalf("first attest: %v", err)
	}
	a2, err := AttestAll(in)
	if err != nil {
		t.Fatalf("second attest: %v", err)
	}
	if a1.Commit != a2.Commit {
		t.Fatalf("deterministic attest expected same commit; got %s vs %s", a1.Commit, a2.Commit)
	}
	if a1.Scheme != SchemeMLAttestV0 {
		t.Fatalf("expected scheme %q, got %q", SchemeMLAttestV0, a1.Scheme)
	}
	if !strings.Contains(a1.Disclosure, "NOT a cryptographic proof") {
		t.Fatalf("disclosure text missing honesty clause: %q", a1.Disclosure)
	}
}

func TestVerify_RoundTrip(t *testing.T) {
	in := mkInput(t, "m1", "w", "i", "o")
	att, err := AttestAll(in)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyAll(in, att)
	if err != nil || !ok {
		t.Fatalf("round-trip verify: ok=%v err=%v", ok, err)
	}
}

func TestVerify_TamperCommit(t *testing.T) {
	in := mkInput(t, "m1", "w", "i", "o")
	att, err := AttestAll(in)
	if err != nil {
		t.Fatal(err)
	}
	tampered := att
	// Flip one hex nibble in the commit.
	buf := []byte(tampered.Commit)
	if buf[0] == '0' {
		buf[0] = '1'
	} else {
		buf[0] = '0'
	}
	tampered.Commit = string(buf)
	ok, err := VerifyAll(in, tampered)
	if ok || err == nil {
		t.Fatalf("tampered commit must fail verify, got ok=%v err=%v", ok, err)
	}
}

func TestVerify_TamperInputsDigest(t *testing.T) {
	in := mkInput(t, "m1", "w", "i", "o")
	att, err := AttestAll(in)
	if err != nil {
		t.Fatal(err)
	}
	// Caller supplies a *different* input at verify time.
	tamperedIn := mkInput(t, "m1", "w", "X-DIFFERENT", "o")
	ok, err := VerifyAll(tamperedIn, att)
	if ok || err == nil {
		t.Fatalf("mismatched inputs must fail verify, got ok=%v err=%v", ok, err)
	}
}

func TestVerify_TamperModelID(t *testing.T) {
	in := mkInput(t, "m1", "w", "i", "o")
	att, err := AttestAll(in)
	if err != nil {
		t.Fatal(err)
	}
	tamperedIn := in
	tamperedIn.ModelID = "m2"
	ok, err := VerifyAll(tamperedIn, att)
	if ok || err == nil {
		t.Fatalf("model_id swap must fail verify, got ok=%v err=%v", ok, err)
	}
}

func TestVerify_RejectsReservedScheme(t *testing.T) {
	// Attacker relabels a hash-only attestation as if it were a real
	// ZK-ML circuit proof. The harness MUST refuse it — the whole
	// point of the disclosure gate.
	in := mkInput(t, "m1", "w", "i", "o")
	att, err := AttestAll(in)
	if err != nil {
		t.Fatal(err)
	}
	att.Scheme = SchemeZKMLFixedV0
	ok, err := VerifyAll(in, att)
	if ok || err == nil {
		t.Fatalf("reserved scheme label must refuse verify, got ok=%v err=%v", ok, err)
	}
	if !strings.Contains(err.Error(), "unsupported scheme") {
		t.Fatalf("expected 'unsupported scheme' in error, got %v", err)
	}
}

func TestAttest_RejectsShortDigest(t *testing.T) {
	in := AttestInput{
		ModelID:       "m1",
		WeightsDigest: []byte("too-short"),
		InputsDigest:  make([]byte, sha256.Size),
		OutputsDigest: make([]byte, sha256.Size),
	}
	if _, err := AttestAll(in); err == nil {
		t.Fatalf("short weights_digest must be rejected")
	}
}

func TestAttest_RejectsEmptyModelID(t *testing.T) {
	in := mkInput(t, "  ", "w", "i", "o")
	if _, err := AttestAll(in); err == nil {
		t.Fatalf("empty model_id must be rejected")
	}
}

func TestAttest_DifferentInputs_DifferentCommits(t *testing.T) {
	a, _ := AttestAll(mkInput(t, "m1", "w", "i1", "o"))
	b, _ := AttestAll(mkInput(t, "m1", "w", "i2", "o"))
	if a.Commit == b.Commit {
		t.Fatalf("different inputs must yield different commits (got %s twice)", a.Commit)
	}
	// Ensure hex encoding is correct length.
	if _, err := hex.DecodeString(a.Commit); err != nil {
		t.Fatalf("commit must be valid hex: %v", err)
	}
	if len(a.Commit) != 2*sha256.Size {
		t.Fatalf("commit hex length must be %d, got %d", 2*sha256.Size, len(a.Commit))
	}
}
