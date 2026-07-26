// Package crypto — key rotation + versioning layer on top of pq.go primitives.
//
// KeyRing tracks signature keys for one process across:
//
//   - Multiple algorithms (ed25519, mldsa65, lamport, hybrid_ed25519_mldsa65)
//   - Monotonic versions per algorithm (v1, v2, …)
//   - Lifecycle states: active (exactly one per algo) or retired (past keys
//     kept for signature verification of already-anchored proofs)
//
// Every key has a stable ID `<algo>:v<version>:<sha256(pub)[:8]>`. Rotation is
// generate-fresh + retire-previous atomically; a signature under a retired
// key still verifies (the ring keeps public material forever), but signing
// operations without an explicit ID always use the current active key.
//
// SECURITY NOTE — private-key storage:
//
//   Private keys live IN MEMORY ONLY. There is no on-disk persistence layer
//   here. A process restart loses every private key; only the JSON snapshot
//   of PUBLIC metadata can be exported/imported (`KeyRing.MarshalPublic` /
//   `LoadPublicKeyRing`). This is deliberate: production deployments MUST
//   wire a hardware-backed key store (HSM, KMS, or a sealed enclave) —
//   see docs/roadmap/KEY_MANAGEMENT.md. Signing endpoints exposing memory-only
//   keys are demo-grade and MUST be gated by CP_KEYRING_ENABLE=1 outside
//   local dev.
package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

// Algo identifies a signature algorithm slot in the ring.
type Algo string

const (
	AlgoEd25519 Algo = "ed25519"
	AlgoMLDSA65 Algo = "mldsa65"
	AlgoLamport Algo = "lamport"
	AlgoHybrid  Algo = "hybrid_ed25519_mldsa65"
)

// SupportedAlgos returns the algorithms the ring knows how to generate,
// sign, and verify. Kept in-sync with generateForAlgo below.
func SupportedAlgos() []Algo {
	return []Algo{AlgoEd25519, AlgoMLDSA65, AlgoLamport, AlgoHybrid}
}

// KeyMeta is the public, exportable view of a key on the ring.
type KeyMeta struct {
	ID        string     `json:"id"`
	Algo      Algo       `json:"algo"`
	Version   int        `json:"version"`
	PublicKey string     `json:"public_key_hex"` // canonical bytes hex
	CreatedAt time.Time  `json:"created_at"`
	RetiredAt *time.Time `json:"retired_at,omitempty"`
	Active    bool       `json:"active"`
}

// keyEntry couples public metadata with the (memory-only) private material.
// For hybrid, we carry both classic and PQ private keys side-by-side.
type keyEntry struct {
	meta KeyMeta
	// One of these is populated depending on algo. Untouched by MarshalPublic.
	ed25519Priv ed25519.PrivateKey
	ed25519Pub  ed25519.PublicKey
	mldsaPriv   *mldsa65.PrivateKey
	mldsaPub   *mldsa65.PublicKey
	lamportPriv *LamportPrivateKey
	lamportPub  *LamportPublicKey
}

// KeyRing is thread-safe. Every mutation happens under mu. Reads take RLock.
type KeyRing struct {
	mu     sync.RWMutex
	keys   map[string]*keyEntry // by ID
	active map[Algo]string      // algo -> active ID (empty string if none)
	// perAlgoMaxVersion tracks the highest version issued for an algo so
	// rotation never re-uses a version even if a key is dropped.
	perAlgoMaxVersion map[Algo]int
	// clock is a seam for tests.
	clock func() time.Time
}

// NewKeyRing returns an empty ring.
func NewKeyRing() *KeyRing {
	return &KeyRing{
		keys:              make(map[string]*keyEntry),
		active:            make(map[Algo]string),
		perAlgoMaxVersion: make(map[Algo]int),
		clock:             time.Now,
	}
}

// SetClock overrides the ring's clock (tests only).
func (r *KeyRing) SetClock(f func() time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clock = f
}

// CreateKey generates a fresh key pair for algo, adds it to the ring, and
// makes it the active key for that algo (retiring the previous active key
// if any). Returns the new key's metadata.
func (r *KeyRing) CreateKey(algo Algo) (KeyMeta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.createLocked(algo)
}

// RotateKey is a convenience alias for CreateKey — the semantics are
// identical (fresh key becomes active, previous active gets retired).
func (r *KeyRing) RotateKey(algo Algo) (KeyMeta, error) {
	return r.CreateKey(algo)
}

func (r *KeyRing) createLocked(algo Algo) (KeyMeta, error) {
	entry, err := generateForAlgo(algo)
	if err != nil {
		return KeyMeta{}, err
	}
	nextVersion := r.perAlgoMaxVersion[algo] + 1
	r.perAlgoMaxVersion[algo] = nextVersion

	// publicBytes dispatches on entry.meta.Algo; set it before computing.
	entry.meta.Algo = algo
	pubBytes := entry.publicBytes()
	id := makeKeyID(algo, nextVersion, pubBytes)
	now := r.clock().UTC()
	entry.meta = KeyMeta{
		ID:        id,
		Algo:      algo,
		Version:   nextVersion,
		PublicKey: hex.EncodeToString(pubBytes),
		CreatedAt: now,
		Active:    true,
	}

	// Retire the previous active key for this algo.
	if prevID, ok := r.active[algo]; ok && prevID != "" {
		if prev, ok := r.keys[prevID]; ok {
			retired := now
			prev.meta.RetiredAt = &retired
			prev.meta.Active = false
		}
	}

	r.keys[id] = entry
	r.active[algo] = id
	return entry.meta, nil
}

// ActiveKeyID returns the current active key ID for algo, or "" and false
// if no key has been generated for that algo yet.
func (r *KeyRing) ActiveKeyID(algo Algo) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.active[algo]
	return id, ok && id != ""
}

// GetMeta returns the KeyMeta for a specific key ID.
func (r *KeyRing) GetMeta(id string) (KeyMeta, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.keys[id]
	if !ok {
		return KeyMeta{}, fmt.Errorf("keyring: unknown key id %q", id)
	}
	return entry.meta, nil
}

// List returns metadata for every key ever registered, sorted by
// (algo asc, version asc). Retired keys are included.
func (r *KeyRing) List() []KeyMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]KeyMeta, 0, len(r.keys))
	for _, e := range r.keys {
		out = append(out, e.meta)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Algo != out[j].Algo {
			return string(out[i].Algo) < string(out[j].Algo)
		}
		return out[i].Version < out[j].Version
	})
	return out
}

// Sign signs message with the currently active key for algo, returning the
// signature bytes and the key ID that produced them. The caller is expected
// to store the ID alongside the signature so future verification can pick
// the right (possibly-retired) public key.
func (r *KeyRing) Sign(algo Algo, message []byte) (signature []byte, keyID string, err error) {
	r.mu.RLock()
	id, ok := r.active[algo]
	r.mu.RUnlock()
	if !ok || id == "" {
		return nil, "", fmt.Errorf("keyring: no active key for algo %q", algo)
	}
	sig, err := r.SignWithKey(id, message)
	if err != nil {
		return nil, "", err
	}
	return sig, id, nil
}

// SignWithKey signs message with a specific key ID (allows re-signing under
// a retired key, e.g. for compatibility tests).
func (r *KeyRing) SignWithKey(id string, message []byte) ([]byte, error) {
	r.mu.RLock()
	entry, ok := r.keys[id]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("keyring: unknown key id %q", id)
	}
	return entry.sign(message)
}

// Verify verifies a signature against the public key of the given key ID.
// Retired keys still verify — the ring exists so anchored signatures do not
// break when the operator rotates.
func (r *KeyRing) Verify(id string, message, signature []byte) (bool, error) {
	r.mu.RLock()
	entry, ok := r.keys[id]
	r.mu.RUnlock()
	if !ok {
		return false, fmt.Errorf("keyring: unknown key id %q", id)
	}
	return entry.verify(message, signature)
}

// MigrateSignature verifies a signature produced under oldKeyID (typically
// a retired or weaker-algo key) and returns a fresh signature under the
// currently-active key of `toAlgo`, plus that new key's ID.
//
// This is the "upgrade path" endpoint: given an existing anchored proof
// signed with, say, an ed25519 v1 key, migrate it forward to a hybrid v2
// signature without re-generating the underlying proof.
func (r *KeyRing) MigrateSignature(oldKeyID string, message, oldSignature []byte, toAlgo Algo) (newSig []byte, newKeyID string, err error) {
	ok, err := r.Verify(oldKeyID, message, oldSignature)
	if err != nil {
		return nil, "", fmt.Errorf("keyring: migrate verify: %w", err)
	}
	if !ok {
		return nil, "", errors.New("keyring: old signature did not verify — refusing to migrate")
	}
	sig, newID, err := r.Sign(toAlgo, message)
	if err != nil {
		return nil, "", fmt.Errorf("keyring: migrate sign: %w", err)
	}
	return sig, newID, nil
}

// MarshalPublic serializes the public view of the ring (metadata only —
// NO private key material). Use for snapshot/audit; import with
// LoadPublicKeyRing to reconstruct a verify-only ring.
func (r *KeyRing) MarshalPublic() ([]byte, error) {
	return json.MarshalIndent(struct {
		Keys []KeyMeta `json:"keys"`
	}{Keys: r.List()}, "", "  ")
}

// LoadPublicKeyRing reconstructs a KeyRing from a MarshalPublic snapshot.
// Private keys are NOT restored — the resulting ring can Verify but not Sign.
func LoadPublicKeyRing(data []byte) (*KeyRing, error) {
	var doc struct {
		Keys []KeyMeta `json:"keys"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("keyring: parse snapshot: %w", err)
	}
	r := NewKeyRing()
	for _, m := range doc.Keys {
		pubBytes, err := hex.DecodeString(m.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("keyring: bad public_key_hex on %s: %w", m.ID, err)
		}
		entry := &keyEntry{meta: m}
		if err := entry.hydratePublicOnly(m.Algo, pubBytes); err != nil {
			return nil, fmt.Errorf("keyring: hydrate %s: %w", m.ID, err)
		}
		r.keys[m.ID] = entry
		if m.Version > r.perAlgoMaxVersion[m.Algo] {
			r.perAlgoMaxVersion[m.Algo] = m.Version
		}
		if m.Active {
			r.active[m.Algo] = m.ID
		}
	}
	return r, nil
}

// -----------------------------------------------------------------------------
// keyEntry internals — algo dispatch
// -----------------------------------------------------------------------------

func generateForAlgo(algo Algo) (*keyEntry, error) {
	e := &keyEntry{}
	switch algo {
	case AlgoEd25519:
		priv, pub, err := GenerateEd25519KeyPair()
		if err != nil {
			return nil, err
		}
		e.ed25519Priv = priv
		e.ed25519Pub = pub
	case AlgoMLDSA65:
		priv, pub, err := GenerateMLDSAKeyPair()
		if err != nil {
			return nil, err
		}
		e.mldsaPriv = priv
		e.mldsaPub = pub
	case AlgoLamport:
		priv, pub, err := GenerateLamportKeyPair()
		if err != nil {
			return nil, err
		}
		e.lamportPriv = priv
		e.lamportPub = pub
	case AlgoHybrid:
		classicPriv, classicPub, err := GenerateEd25519KeyPair()
		if err != nil {
			return nil, err
		}
		pqPriv, pqPub, err := GenerateMLDSAKeyPair()
		if err != nil {
			return nil, err
		}
		e.ed25519Priv = classicPriv
		e.ed25519Pub = classicPub
		e.mldsaPriv = pqPriv
		e.mldsaPub = pqPub
	default:
		return nil, fmt.Errorf("keyring: unsupported algo %q", algo)
	}
	return e, nil
}

func (e *keyEntry) publicBytes() []byte {
	switch e.meta.Algo {
	case AlgoEd25519:
		return []byte(e.ed25519Pub)
	case AlgoMLDSA65:
		pubBytes, _ := e.mldsaPub.MarshalBinary()
		return pubBytes
	case AlgoLamport:
		return e.lamportPub.Bytes()
	case AlgoHybrid:
		// Concatenate classic || pq, with a 4-byte length prefix on classic
		// so unmarshal can split them.
		classic := []byte(e.ed25519Pub)
		pq, _ := e.mldsaPub.MarshalBinary()
		out := make([]byte, 4+len(classic)+len(pq))
		out[0] = byte(len(classic) >> 24)
		out[1] = byte(len(classic) >> 16)
		out[2] = byte(len(classic) >> 8)
		out[3] = byte(len(classic))
		copy(out[4:], classic)
		copy(out[4+len(classic):], pq)
		return out
	}
	// Called before generateForAlgo? Shouldn't happen; return empty so ID
	// generation surfaces the mistake loudly rather than crashing.
	return nil
}

func (e *keyEntry) sign(message []byte) ([]byte, error) {
	switch e.meta.Algo {
	case AlgoEd25519:
		if e.ed25519Priv == nil {
			return nil, errors.New("keyring: private key not present (verify-only ring)")
		}
		return SignClassic(e.ed25519Priv, message)
	case AlgoMLDSA65:
		if e.mldsaPriv == nil {
			return nil, errors.New("keyring: private key not present (verify-only ring)")
		}
		return SignMLDSA(e.mldsaPriv, message)
	case AlgoLamport:
		if e.lamportPriv == nil {
			return nil, errors.New("keyring: private key not present (verify-only ring)")
		}
		return SignSPHINCS(e.lamportPriv, message)
	case AlgoHybrid:
		if e.ed25519Priv == nil || e.mldsaPriv == nil {
			return nil, errors.New("keyring: private key not present (verify-only ring)")
		}
		return HybridSign(e.ed25519Priv, e.mldsaPriv, message)
	}
	return nil, fmt.Errorf("keyring: unsupported algo %q", e.meta.Algo)
}

func (e *keyEntry) verify(message, signature []byte) (bool, error) {
	switch e.meta.Algo {
	case AlgoEd25519:
		return VerifyClassic(e.ed25519Pub, message, signature)
	case AlgoMLDSA65:
		return VerifyMLDSA(e.mldsaPub, message, signature)
	case AlgoLamport:
		return VerifySPHINCS(e.lamportPub, message, signature)
	case AlgoHybrid:
		ok, _, _, err := HybridVerify(e.ed25519Pub, e.mldsaPub, message, signature)
		return ok, err
	}
	return false, fmt.Errorf("keyring: unsupported algo %q", e.meta.Algo)
}

func (e *keyEntry) hydratePublicOnly(algo Algo, pubBytes []byte) error {
	switch algo {
	case AlgoEd25519:
		if len(pubBytes) != ed25519.PublicKeySize {
			return errInvalidKeyLength
		}
		e.ed25519Pub = ed25519.PublicKey(pubBytes)
	case AlgoMLDSA65:
		var pub mldsa65.PublicKey
		if err := pub.UnmarshalBinary(pubBytes); err != nil {
			return err
		}
		e.mldsaPub = &pub
	case AlgoLamport:
		pub, err := LamportPublicKeyFromBytes(pubBytes)
		if err != nil {
			return err
		}
		e.lamportPub = pub
	case AlgoHybrid:
		if len(pubBytes) < 4 {
			return errInvalidKeyLength
		}
		classicLen := int(pubBytes[0])<<24 | int(pubBytes[1])<<16 | int(pubBytes[2])<<8 | int(pubBytes[3])
		if 4+classicLen > len(pubBytes) {
			return errInvalidKeyLength
		}
		classic := pubBytes[4 : 4+classicLen]
		pq := pubBytes[4+classicLen:]
		if len(classic) != ed25519.PublicKeySize {
			return errInvalidKeyLength
		}
		e.ed25519Pub = ed25519.PublicKey(classic)
		var pqPub mldsa65.PublicKey
		if err := pqPub.UnmarshalBinary(pq); err != nil {
			return err
		}
		e.mldsaPub = &pqPub
	default:
		return fmt.Errorf("keyring: unsupported algo %q", algo)
	}
	return nil
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func makeKeyID(algo Algo, version int, pubBytes []byte) string {
	h := sha256.Sum256(pubBytes)
	return fmt.Sprintf("%s:v%d:%s", algo, version, hex.EncodeToString(h[:4]))
}

// ParseAlgo turns a string into an Algo, returning an error for unknown values.
func ParseAlgo(s string) (Algo, error) {
	a := Algo(strings.ToLower(s))
	for _, known := range SupportedAlgos() {
		if a == known {
			return a, nil
		}
	}
	return "", fmt.Errorf("keyring: unsupported algo %q", s)
}
