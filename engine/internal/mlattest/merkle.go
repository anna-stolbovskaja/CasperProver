package mlattest

// MerkleMLAttestor — attestation harness with selective disclosure.
//
// Sits alongside HashMLAttestor under the same disclosure discipline
// (see docs/ZKML_HONEST_VERDICT.md). Emits a DISTINCT scheme label
// ("ml-attest-merkle-v0") so downstream consumers cannot confuse the
// two envelopes and cannot mistake either for a cryptographic ML
// inference proof. The HashMLAttestor.Verify path deliberately refuses
// unknown scheme labels — including this one — as designed.
//
// What this adds over HashMLAttestor: the weights digest is replaced by
// a MERKLE ROOT over an ordered list of weight chunks. Given the root
// (published in the envelope), an auditor can request an opening for
// any single chunk index and verify it against the root — WITHOUT the
// prover disclosing the other chunks. This is the selective-disclosure
// property that a flat SHA-256 over the whole weights blob cannot
// provide.
//
// What this is NOT: a cryptographic proof of ML inference. The
// attestation still commits to (model_id, weights_root, inputs_digest,
// outputs_digest) — it says "these are the inputs, this is a Merkle
// root over the weights, this is the output". It says nothing about
// whether the named model was actually executed on the named inputs.
// The four gating conditions in ZKML_HONEST_VERDICT.md remain.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// SchemeMLAttestMerkleV0 is the label reported inside
// MerkleAttestation.Scheme. Distinct from SchemeMLAttestV0.
const SchemeMLAttestMerkleV0 AttestationScheme = "ml-attest-merkle-v0"

// Domain-separation tags. Different tag on leaf vs internal node is
// standard practice to avoid second-preimage between the two layers.
var (
	merkleLeafTag = []byte("ml-attest-merkle-v0:leaf")
	merkleNodeTag = []byte("ml-attest-merkle-v0:node")
	merkleSeedTag = []byte("ml-attest-merkle-v0:commit-seed")
)

// MerkleAttestation is the public envelope emitted by MerkleMLAttestor.
//
// WeightsRoot replaces HashMLAttestor.WeightsDigest — it is the Merkle
// root computed by CommitWeights over the caller's ordered chunk list.
// LeafCount is emitted so the verifier can range-check openings without
// re-hashing the whole tree.
type MerkleAttestation struct {
	Scheme        AttestationScheme `json:"scheme"`
	ModelID       string            `json:"model_id"`
	WeightsRoot   string            `json:"weights_root_hex"` // Merkle root over ordered chunks
	LeafCount     int               `json:"leaf_count"`
	InputsDigest  string            `json:"inputs_digest_hex"`
	OutputsDigest string            `json:"outputs_digest_hex"`
	Commit        string            `json:"commit_hex"` // envelope commitment
	Disclosure    string            `json:"disclosure"`
}

// MerkleOpening is the selective-disclosure proof for one chunk.
//
// Given (Chunk, LeafIndex, Siblings), a verifier can recompute the
// candidate root and compare against MerkleAttestation.WeightsRoot,
// with no other chunk disclosed. Siblings is ordered leaf-to-root; each
// entry is the sibling of the current node at that level.
type MerkleOpening struct {
	LeafIndex int      `json:"leaf_index"`
	LeafCount int      `json:"leaf_count"`
	Chunk     []byte   `json:"-"`             // caller supplies at verify time
	ChunkHex  string   `json:"chunk_hex"`     // hex-encoded chunk bytes
	Siblings  []string `json:"siblings_hex"`  // hex-encoded sibling hashes, leaf→root
	IsRight   []bool   `json:"is_right_path"` // true where current node is the right child
}

// MerkleAttestInput is the raw material MerkleMLAttestor commits to.
//
// WeightChunks is an ORDERED list of chunk byte-slices. Order is
// significant: the Merkle root is index-sensitive. Empty chunks and an
// empty list are both rejected.
type MerkleAttestInput struct {
	ModelID       string
	WeightChunks  [][]byte
	InputsDigest  []byte
	OutputsDigest []byte
}

// MerkleMLAttestor is stateless and safe for concurrent use.
type MerkleMLAttestor struct{}

// NewMerkleMLAttestor returns a new stateless attestor.
func NewMerkleMLAttestor() *MerkleMLAttestor { return &MerkleMLAttestor{} }

func (m *MerkleMLAttestor) validate(in MerkleAttestInput) error {
	if strings.TrimSpace(in.ModelID) == "" {
		return errors.New("mlattest.merkle: model_id is required")
	}
	if len(in.WeightChunks) == 0 {
		return errors.New("mlattest.merkle: weight_chunks must not be empty")
	}
	for i, c := range in.WeightChunks {
		if len(c) == 0 {
			return fmt.Errorf("mlattest.merkle: weight_chunks[%d] is empty", i)
		}
	}
	if len(in.InputsDigest) != sha256.Size {
		return fmt.Errorf("mlattest.merkle: inputs_digest must be %d bytes", sha256.Size)
	}
	if len(in.OutputsDigest) != sha256.Size {
		return fmt.Errorf("mlattest.merkle: outputs_digest must be %d bytes", sha256.Size)
	}
	return nil
}

// leafHash computes SHA256(leaf_tag || chunk). Domain-separated so a
// chunk can never be confused for an internal node hash.
func leafHash(chunk []byte) [sha256.Size]byte {
	h := sha256.New()
	h.Write(merkleLeafTag)
	h.Write(chunk)
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out
}

// nodeHash computes SHA256(node_tag || left || right). Domain-separated
// against leafHash and against the envelope commit-seed.
func nodeHash(left, right [sha256.Size]byte) [sha256.Size]byte {
	h := sha256.New()
	h.Write(merkleNodeTag)
	h.Write(left[:])
	h.Write(right[:])
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out
}

// buildTree returns every level of the Merkle tree, level 0 = leaves,
// last level = single root. Odd-node level is padded by duplicating
// the last node (Bitcoin-style). Deterministic — same input → same
// tree.
func (m *MerkleMLAttestor) buildTree(chunks [][]byte) [][][sha256.Size]byte {
	level := make([][sha256.Size]byte, len(chunks))
	for i, c := range chunks {
		level[i] = leafHash(c)
	}
	levels := [][][sha256.Size]byte{level}
	for len(level) > 1 {
		if len(level)%2 == 1 {
			level = append(level, level[len(level)-1])
		}
		next := make([][sha256.Size]byte, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			next[i/2] = nodeHash(level[i], level[i+1])
		}
		levels = append(levels, next)
		level = next
	}
	return levels
}

// merkleRoot returns the root of the tree built over chunks.
func (m *MerkleMLAttestor) merkleRoot(chunks [][]byte) [sha256.Size]byte {
	levels := m.buildTree(chunks)
	return levels[len(levels)-1][0]
}

// envelopeCommit binds root + input/output digests + model_id into a
// single envelope commit. Domain-separated seed prevents scheme cross-
// contamination.
func (m *MerkleMLAttestor) envelopeCommit(modelID string, root, inputs, outputs [sha256.Size]byte) [sha256.Size]byte {
	stepA := sha256.Sum256(append([]byte(modelID), root[:]...))
	stepB := sha256.Sum256(append(append([]byte{}, inputs[:]...), outputs[:]...))
	seed := sha256.Sum256(merkleSeedTag)
	h := sha256.New()
	h.Write(seed[:])
	h.Write(stepA[:])
	h.Write(stepB[:])
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out
}

// Attest builds the Merkle tree over WeightChunks and returns the
// envelope. Deterministic — same input → same commit and root.
func (m *MerkleMLAttestor) Attest(in MerkleAttestInput) (MerkleAttestation, error) {
	if err := m.validate(in); err != nil {
		return MerkleAttestation{}, err
	}
	root := m.merkleRoot(in.WeightChunks)
	var iDig, oDig [sha256.Size]byte
	copy(iDig[:], in.InputsDigest)
	copy(oDig[:], in.OutputsDigest)
	c := m.envelopeCommit(in.ModelID, root, iDig, oDig)
	return MerkleAttestation{
		Scheme:        SchemeMLAttestMerkleV0,
		ModelID:       in.ModelID,
		WeightsRoot:   hex.EncodeToString(root[:]),
		LeafCount:     len(in.WeightChunks),
		InputsDigest:  hex.EncodeToString(in.InputsDigest),
		OutputsDigest: hex.EncodeToString(in.OutputsDigest),
		Commit:        hex.EncodeToString(c[:]),
		Disclosure:    DisclosureText,
	}, nil
}

// Open produces a MerkleOpening for the chunk at leafIdx. The prover
// must retain the full ordered chunk list to answer opening requests;
// there is no way to compute an opening from the envelope alone.
//
// Returns error on out-of-range index or if chunks are invalid.
func (m *MerkleMLAttestor) Open(chunks [][]byte, leafIdx int) (MerkleOpening, error) {
	if leafIdx < 0 || leafIdx >= len(chunks) {
		return MerkleOpening{}, fmt.Errorf("mlattest.merkle: leaf_index %d out of range [0,%d)", leafIdx, len(chunks))
	}
	if len(chunks[leafIdx]) == 0 {
		return MerkleOpening{}, fmt.Errorf("mlattest.merkle: chunk[%d] is empty", leafIdx)
	}
	levels := m.buildTree(chunks)
	siblings := make([]string, 0, len(levels)-1)
	isRight := make([]bool, 0, len(levels)-1)
	idx := leafIdx
	for lvl := 0; lvl < len(levels)-1; lvl++ {
		nodes := levels[lvl]
		// Determine sibling. Odd-node padding is by duplicating last.
		var sib [sha256.Size]byte
		if idx%2 == 0 {
			if idx+1 < len(nodes) {
				sib = nodes[idx+1]
			} else {
				sib = nodes[idx] // padded — sibling is self
			}
			isRight = append(isRight, false)
		} else {
			sib = nodes[idx-1]
			isRight = append(isRight, true)
		}
		siblings = append(siblings, hex.EncodeToString(sib[:]))
		idx /= 2
	}
	// Copy chunk defensively so caller mutating input doesn't corrupt opening.
	chunkCopy := append([]byte{}, chunks[leafIdx]...)
	return MerkleOpening{
		LeafIndex: leafIdx,
		LeafCount: len(chunks),
		Chunk:     chunkCopy,
		ChunkHex:  hex.EncodeToString(chunkCopy),
		Siblings:  siblings,
		IsRight:   isRight,
	}, nil
}

// VerifyOpening recomputes the root from (chunk, index, siblings) and
// compares against the envelope's WeightsRoot. Returns (true, nil) iff
// the recomputed root equals the envelope root AND the scheme label is
// SchemeMLAttestMerkleV0 AND the opening's LeafCount matches the
// envelope's LeafCount AND leaf_index is in range.
//
// This is the selective-disclosure verification path: no other chunk
// is required.
func (m *MerkleMLAttestor) VerifyOpening(att MerkleAttestation, op MerkleOpening) (bool, error) {
	if att.Scheme != SchemeMLAttestMerkleV0 {
		return false, fmt.Errorf("mlattest.merkle: unsupported scheme %q (this harness only verifies %q)", att.Scheme, SchemeMLAttestMerkleV0)
	}
	if op.LeafCount != att.LeafCount {
		return false, fmt.Errorf("mlattest.merkle: leaf_count mismatch (envelope=%d, opening=%d)", att.LeafCount, op.LeafCount)
	}
	if op.LeafIndex < 0 || op.LeafIndex >= att.LeafCount {
		return false, fmt.Errorf("mlattest.merkle: leaf_index %d out of range [0,%d)", op.LeafIndex, att.LeafCount)
	}
	if len(op.Siblings) != len(op.IsRight) {
		return false, errors.New("mlattest.merkle: siblings and is_right_path lengths differ")
	}
	if len(op.Chunk) == 0 {
		return false, errors.New("mlattest.merkle: opening chunk is empty")
	}
	if hex.EncodeToString(op.Chunk) != op.ChunkHex {
		return false, errors.New("mlattest.merkle: opening chunk_hex disagrees with chunk bytes")
	}
	// Cryptographically bind LeafIndex: the is_right_path bit sequence
	// MUST equal the low bits of leaf_index (LSB=level 0). Without this
	// check, an opening for leaf i verifies with any claimed index that
	// happens to share the same tree path — turning LeafIndex into
	// spoofable metadata. Padding rule (duplicated last node) means the
	// expected path bit is derived from position, not from tree state.
	idx := op.LeafIndex
	for i := 0; i < len(op.IsRight); i++ {
		expectedIsRight := (idx & 1) == 1
		if op.IsRight[i] != expectedIsRight {
			return false, fmt.Errorf("mlattest.merkle: is_right_path[%d]=%v inconsistent with leaf_index %d", i, op.IsRight[i], op.LeafIndex)
		}
		idx >>= 1
	}
	// Recompute leaf and climb.
	cur := leafHash(op.Chunk)
	for i, sibHex := range op.Siblings {
		sibBytes, err := hex.DecodeString(sibHex)
		if err != nil {
			return false, fmt.Errorf("mlattest.merkle: sibling[%d] hex decode: %w", i, err)
		}
		if len(sibBytes) != sha256.Size {
			return false, fmt.Errorf("mlattest.merkle: sibling[%d] must be %d bytes", i, sha256.Size)
		}
		var sib [sha256.Size]byte
		copy(sib[:], sibBytes)
		if op.IsRight[i] {
			cur = nodeHash(sib, cur)
		} else {
			cur = nodeHash(cur, sib)
		}
	}
	if hex.EncodeToString(cur[:]) != att.WeightsRoot {
		return false, errors.New("mlattest.merkle: recomputed root does not match envelope — opening tampered or wrong tree")
	}
	return true, nil
}

// VerifyEnvelope re-derives the envelope commit from the input digests
// + envelope root and compares against att.Commit. Does NOT open any
// chunk; a separate VerifyOpening call is needed per disclosed leaf.
func (m *MerkleMLAttestor) VerifyEnvelope(in MerkleAttestInput, att MerkleAttestation) (bool, error) {
	if att.Scheme != SchemeMLAttestMerkleV0 {
		return false, fmt.Errorf("mlattest.merkle: unsupported scheme %q (this harness only verifies %q)", att.Scheme, SchemeMLAttestMerkleV0)
	}
	if err := m.validate(in); err != nil {
		return false, err
	}
	if len(in.WeightChunks) != att.LeafCount {
		return false, fmt.Errorf("mlattest.merkle: leaf_count mismatch (envelope=%d, input=%d)", att.LeafCount, len(in.WeightChunks))
	}
	if in.ModelID != att.ModelID {
		return false, errors.New("mlattest.merkle: model_id envelope mismatch")
	}
	if hex.EncodeToString(in.InputsDigest) != att.InputsDigest {
		return false, errors.New("mlattest.merkle: inputs_digest envelope mismatch")
	}
	if hex.EncodeToString(in.OutputsDigest) != att.OutputsDigest {
		return false, errors.New("mlattest.merkle: outputs_digest envelope mismatch")
	}
	root := m.merkleRoot(in.WeightChunks)
	if hex.EncodeToString(root[:]) != att.WeightsRoot {
		return false, errors.New("mlattest.merkle: recomputed root differs from envelope")
	}
	var iDig, oDig [sha256.Size]byte
	copy(iDig[:], in.InputsDigest)
	copy(oDig[:], in.OutputsDigest)
	c := m.envelopeCommit(in.ModelID, root, iDig, oDig)
	if hex.EncodeToString(c[:]) != att.Commit {
		return false, errors.New("mlattest.merkle: commit mismatch — envelope tampered or scheme divergent")
	}
	return true, nil
}
