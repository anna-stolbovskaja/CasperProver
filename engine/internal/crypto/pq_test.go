package crypto

import "testing"

// NOTE: package crypto is not currently wired into any HTTP endpoint or the
// main binary (see docs/KNOWN_LIMITATIONS.md) - it's scaffolding for the
// roadmap's "post-quantum proof signing" item, not a live feature. These are
// honest tests of its actual current behavior. A prior version of this file
// referenced functions (GenerateKeyPair, SignMessage, VerifySignature,
// HashMessage, Sha256Hash, HexEncode, HexDecode, GenerateRandomBytes,
// GenerateRandomString) that never existed anywhere in this package and had
// never been compiled; it was replaced with this file.

func TestGenerateSimulatedKeyPair_Lengths(t *testing.T) {
	for _, keyLen := range []int{simulatedSPHINCSKeyLen, simulatedMLDSAKeyLen, simulatedClassicSigLen} {
		priv, pub, err := generateSimulatedKeyPair(keyLen)
		if err != nil {
			t.Fatalf("keyLen=%d: unexpected error: %v", keyLen, err)
		}
		if len(priv) != keyLen || len(pub) != keyLen {
			t.Errorf("keyLen=%d: got priv=%d pub=%d", keyLen, len(priv), len(pub))
		}
	}
}

func TestSignSPHINCS_RejectsWrongKeyLength(t *testing.T) {
	if _, err := SignSPHINCS([]byte("too-short"), []byte("msg")); err == nil {
		t.Error("expected error for wrong-length private key")
	}
}

func TestSignSPHINCS_ProducesFixedLengthSignature(t *testing.T) {
	priv, _, err := generateSimulatedKeyPair(simulatedSPHINCSKeyLen)
	if err != nil {
		t.Fatalf("keygen failed: %v", err)
	}
	sig, err := SignSPHINCS(priv, []byte("message"))
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	if len(sig) != simulatedSPHINCSSigLen {
		t.Errorf("expected signature length %d, got %d", simulatedSPHINCSSigLen, len(sig))
	}
}

func TestSignMLDSA_ProducesFixedLengthSignature(t *testing.T) {
	priv, _, err := generateSimulatedKeyPair(simulatedMLDSAKeyLen)
	if err != nil {
		t.Fatalf("keygen failed: %v", err)
	}
	sig, err := SignMLDSA(priv, []byte("message"))
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	if len(sig) != simulatedMLDSASigLen {
		t.Errorf("expected signature length %d, got %d", simulatedMLDSASigLen, len(sig))
	}
}

func TestHybridSignature_MarshalUnmarshalRoundTrip(t *testing.T) {
	hs := &HybridSignature{ClassicSig: []byte("classic-sig"), PQSig: []byte("pq-sig-bytes")}
	data, err := hs.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out HybridSignature
	if err := out.UnmarshalBinary(data); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if string(out.ClassicSig) != string(hs.ClassicSig) || string(out.PQSig) != string(hs.PQSig) {
		t.Error("round-tripped hybrid signature does not match original")
	}
}

func TestHybridSign_ProducesUnmarshalableOutput(t *testing.T) {
	classicPriv, _, _ := generateSimulatedKeyPair(simulatedClassicSigLen)
	mldsaPriv, _, _ := generateSimulatedKeyPair(simulatedMLDSAKeyLen)

	out, err := HybridSign(classicPriv, mldsaPriv, []byte("message"))
	if err != nil {
		t.Fatalf("HybridSign failed: %v", err)
	}
	var hs HybridSignature
	if err := hs.UnmarshalBinary(out); err != nil {
		t.Fatalf("output of HybridSign did not unmarshal: %v", err)
	}
}
