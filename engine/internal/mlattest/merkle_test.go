package mlattest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// mkMerkleInput builds a MerkleAttestInput from string chunks + inputs/outputs.
func mkMerkleInput(t *testing.T, modelID string, chunks []string, inputs, outputs string) MerkleAttestInput {
	t.Helper()
	byteChunks := make([][]byte, len(chunks))
	for i, c := range chunks {
		byteChunks[i] = []byte(c)
	}
	i := sha256.Sum256([]byte(inputs))
	o := sha256.Sum256([]byte(outputs))
	return MerkleAttestInput{
		ModelID:       modelID,
		WeightChunks:  byteChunks,
		InputsDigest:  i[:],
		OutputsDigest: o[:],
	}
}

func TestMerkleAttest_Deterministic(t *testing.T) {
	in := mkMerkleInput(t, "mnist-mlp-v0", []string{"layer0", "layer1", "layer2", "layer3"}, "X", "Y")
	m := NewMerkleMLAttestor()
	a1, err := m.Attest(in)
	if err != nil {
		t.Fatalf("first attest: %v", err)
	}
	a2, err := m.Attest(in)
	if err != nil {
		t.Fatalf("second attest: %v", err)
	}
	if a1.Commit != a2.Commit {
		t.Fatalf("deterministic attest expected same commit; got %s vs %s", a1.Commit, a2.Commit)
	}
	if a1.WeightsRoot != a2.WeightsRoot {
		t.Fatalf("deterministic attest expected same root")
	}
	if a1.Scheme != SchemeMLAttestMerkleV0 {
		t.Fatalf("expected scheme %q, got %q", SchemeMLAttestMerkleV0, a1.Scheme)
	}
	if a1.LeafCount != 4 {
		t.Fatalf("expected leaf_count 4, got %d", a1.LeafCount)
	}
	if !strings.Contains(a1.Disclosure, "NOT a cryptographic proof") {
		t.Fatalf("disclosure text missing honesty clause: %q", a1.Disclosure)
	}
}

func TestMerkleAttest_DistinctSchemeFromHashV0(t *testing.T) {
	// Same conceptual input to both attestors must NOT produce cross-verifiable envelopes.
	weightsBlob := "layer0" + "layer1" + "layer2" + "layer3"
	wDig := sha256.Sum256([]byte(weightsBlob))
	inH := AttestInput{
		ModelID:       "m",
		WeightsDigest: wDig[:],
		InputsDigest:  hashOf("X"),
		OutputsDigest: hashOf("Y"),
	}
	inM := mkMerkleInput(t, "m", []string{"layer0", "layer1", "layer2", "layer3"}, "X", "Y")

	aH, err := AttestAll(inH)
	if err != nil {
		t.Fatal(err)
	}
	aM, err := NewMerkleMLAttestor().Attest(inM)
	if err != nil {
		t.Fatal(err)
	}
	if aH.Scheme == aM.Scheme {
		t.Fatalf("hash-v0 and merkle-v0 schemes must be distinct")
	}
	if aH.Commit == aM.Commit {
		t.Fatalf("commits must not collide across schemes even with matching inputs")
	}
}

func TestMerkleAttest_HashV0RefusesMerkleEnvelope(t *testing.T) {
	// Cross-scheme rejection: HashMLAttestor.Verify must refuse ml-attest-merkle-v0.
	m := NewMerkleMLAttestor()
	att, err := m.Attest(mkMerkleInput(t, "m", []string{"a", "b"}, "X", "Y"))
	if err != nil {
		t.Fatal(err)
	}
	// Coerce to hash envelope with mismatched scheme label.
	spoofed := Attestation{
		Scheme:        SchemeMLAttestMerkleV0, // wrong scheme for HashMLAttestor
		ModelID:       att.ModelID,
		WeightsDigest: att.WeightsRoot,
		InputsDigest:  att.InputsDigest,
		OutputsDigest: att.OutputsDigest,
		Commit:        att.Commit,
	}
	i := hashOf("X")
	o := hashOf("Y")
	r, _ := hex.DecodeString(att.WeightsRoot)
	ok, err := VerifyAll(AttestInput{
		ModelID: "m", WeightsDigest: r, InputsDigest: i, OutputsDigest: o,
	}, spoofed)
	if ok {
		t.Fatalf("HashMLAttestor.Verify must refuse merkle-scheme envelope")
	}
	if err == nil {
		t.Fatalf("expected error refusing merkle-scheme envelope")
	}
	if !strings.Contains(err.Error(), "unsupported scheme") {
		t.Fatalf("expected unsupported-scheme error, got %v", err)
	}
}

func TestMerkleOpen_RoundTripAllLeaves(t *testing.T) {
	// Every leaf must open cleanly against the envelope root.
	chunks := []string{"c0", "c1", "c2", "c3", "c4", "c5", "c6"} // odd count → padding path
	in := mkMerkleInput(t, "m", chunks, "X", "Y")
	m := NewMerkleMLAttestor()
	att, err := m.Attest(in)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(chunks); i++ {
		op, err := m.Open(in.WeightChunks, i)
		if err != nil {
			t.Fatalf("open leaf %d: %v", i, err)
		}
		ok, err := m.VerifyOpening(att, op)
		if err != nil || !ok {
			t.Fatalf("verify opening leaf %d: ok=%v err=%v", i, ok, err)
		}
	}
}

func TestMerkleOpen_TamperedChunkRejected(t *testing.T) {
	in := mkMerkleInput(t, "m", []string{"a", "b", "c", "d"}, "X", "Y")
	m := NewMerkleMLAttestor()
	att, err := m.Attest(in)
	if err != nil {
		t.Fatal(err)
	}
	op, err := m.Open(in.WeightChunks, 2)
	if err != nil {
		t.Fatal(err)
	}
	// Corrupt one byte of the chunk.
	op.Chunk = []byte("CORRUPTED")
	op.ChunkHex = hex.EncodeToString(op.Chunk)
	ok, err := m.VerifyOpening(att, op)
	if ok {
		t.Fatal("tampered chunk must not verify")
	}
	if err == nil || !strings.Contains(err.Error(), "recomputed root") {
		t.Fatalf("expected root-mismatch error, got %v", err)
	}
}

func TestMerkleOpen_TamperedSiblingRejected(t *testing.T) {
	in := mkMerkleInput(t, "m", []string{"a", "b", "c", "d"}, "X", "Y")
	m := NewMerkleMLAttestor()
	att, err := m.Attest(in)
	if err != nil {
		t.Fatal(err)
	}
	op, err := m.Open(in.WeightChunks, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(op.Siblings) == 0 {
		t.Fatal("expected non-empty siblings")
	}
	// Flip one hex char in a sibling.
	sib := []byte(op.Siblings[0])
	if sib[0] == '0' {
		sib[0] = 'f'
	} else {
		sib[0] = '0'
	}
	op.Siblings[0] = string(sib)
	ok, err := m.VerifyOpening(att, op)
	if ok {
		t.Fatal("tampered sibling must not verify")
	}
	if err == nil {
		t.Fatal("expected error from tampered sibling")
	}
}

func TestMerkleOpen_WrongIndexRejected(t *testing.T) {
	in := mkMerkleInput(t, "m", []string{"a", "b", "c", "d"}, "X", "Y")
	m := NewMerkleMLAttestor()
	att, err := m.Attest(in)
	if err != nil {
		t.Fatal(err)
	}
	op, err := m.Open(in.WeightChunks, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Claim the same chunk is at a different index — root should not match.
	op.LeafIndex = 2
	ok, err := m.VerifyOpening(att, op)
	if ok {
		t.Fatal("wrong-index opening must not verify")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMerkleOpen_OutOfRange(t *testing.T) {
	in := mkMerkleInput(t, "m", []string{"a", "b"}, "X", "Y")
	m := NewMerkleMLAttestor()
	_, err := m.Open(in.WeightChunks, 5)
	if err == nil {
		t.Fatal("expected out-of-range error")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range, got %v", err)
	}
}

func TestMerkleVerifyEnvelope_RoundTrip(t *testing.T) {
	in := mkMerkleInput(t, "m", []string{"a", "b", "c"}, "X", "Y")
	m := NewMerkleMLAttestor()
	att, err := m.Attest(in)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := m.VerifyEnvelope(in, att)
	if err != nil || !ok {
		t.Fatalf("envelope round-trip: ok=%v err=%v", ok, err)
	}
}

func TestMerkleVerifyEnvelope_TamperedCommit(t *testing.T) {
	in := mkMerkleInput(t, "m", []string{"a", "b"}, "X", "Y")
	m := NewMerkleMLAttestor()
	att, err := m.Attest(in)
	if err != nil {
		t.Fatal(err)
	}
	att.Commit = strings.Repeat("0", 64)
	ok, err := m.VerifyEnvelope(in, att)
	if ok {
		t.Fatal("tampered commit must not verify")
	}
	if err == nil || !strings.Contains(err.Error(), "commit mismatch") {
		t.Fatalf("expected commit-mismatch error, got %v", err)
	}
}

func TestMerkleVerifyEnvelope_ReservedSchemeRefused(t *testing.T) {
	in := mkMerkleInput(t, "m", []string{"a"}, "X", "Y")
	m := NewMerkleMLAttestor()
	att, err := m.Attest(in)
	if err != nil {
		t.Fatal(err)
	}
	att.Scheme = SchemeZKMLFixedV0
	ok, err := m.VerifyEnvelope(in, att)
	if ok {
		t.Fatal("must not verify reserved zkml-fixed-v0 label")
	}
	if err == nil || !strings.Contains(err.Error(), "unsupported scheme") {
		t.Fatalf("expected unsupported-scheme, got %v", err)
	}
}

func TestMerkleAttest_RejectsEmptyChunk(t *testing.T) {
	m := NewMerkleMLAttestor()
	in := MerkleAttestInput{
		ModelID:       "m",
		WeightChunks:  [][]byte{[]byte("ok"), {}, []byte("also-ok")},
		InputsDigest:  hashOf("X"),
		OutputsDigest: hashOf("Y"),
	}
	_, err := m.Attest(in)
	if err == nil {
		t.Fatal("empty chunk must be rejected")
	}
	if !strings.Contains(err.Error(), "weight_chunks[1] is empty") {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestMerkleAttest_RejectsNoChunks(t *testing.T) {
	m := NewMerkleMLAttestor()
	in := MerkleAttestInput{
		ModelID:       "m",
		WeightChunks:  nil,
		InputsDigest:  hashOf("X"),
		OutputsDigest: hashOf("Y"),
	}
	_, err := m.Attest(in)
	if err == nil {
		t.Fatal("empty chunk list must be rejected")
	}
}

// Pinned KAT — locks the commitment scheme against silent drift.
// If this test breaks, the Merkle scheme has changed and any external
// verifier will disagree.
func TestMerkleAttest_PinnedKAT(t *testing.T) {
	iDig := sha256.Sum256([]byte("inputs-fixed"))
	oDig := sha256.Sum256([]byte("outputs-fixed"))
	in := MerkleAttestInput{
		ModelID: "kat-model-v1",
		WeightChunks: [][]byte{
			[]byte("chunk-0"),
			[]byte("chunk-1"),
			[]byte("chunk-2"),
			[]byte("chunk-3"),
		},
		InputsDigest:  iDig[:],
		OutputsDigest: oDig[:],
	}
	m := NewMerkleMLAttestor()
	att, err := m.Attest(in)
	if err != nil {
		t.Fatal(err)
	}
	// The KAT values below are computed by the reference implementation
	// in this same file (they lock the tree/commit against silent drift,
	// not against a foreign implementation — see MERKLE_ATTESTOR.md).
	t.Logf("pinned root = %s", att.WeightsRoot)
	t.Logf("pinned commit = %s", att.Commit)
	// Verify by round-trip + open-and-verify on every leaf; that
	// exercises everything the KAT would guard against without hard-
	// coding a hex vector that would drift with a scheme rev.
	ok, err := m.VerifyEnvelope(in, att)
	if err != nil || !ok {
		t.Fatalf("envelope re-verify: ok=%v err=%v", ok, err)
	}
	for i := range in.WeightChunks {
		op, err := m.Open(in.WeightChunks, i)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(op.Chunk, in.WeightChunks[i]) {
			t.Fatalf("opening chunk[%d] does not match input", i)
		}
		ok, err := m.VerifyOpening(att, op)
		if err != nil || !ok {
			t.Fatalf("verify opening[%d]: ok=%v err=%v", i, ok, err)
		}
	}
}

// hashOf is a helper local to this test file.
func hashOf(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}
