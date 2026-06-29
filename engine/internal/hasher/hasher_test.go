package hasher

import "testing"

func TestHash(t *testing.T) {
	h := Hash([]byte("hello"))
	if h == [32]byte{} {
		t.Fatal("empty hash")
	}
}

func TestHexHash(t *testing.T) {
	h := HexHash([]byte("hello"))
	if len(h) != 64 {
		t.Fatalf("want 64 hex chars, got %d", len(h))
	}
}

func TestCommitHash(t *testing.T) {
	h1 := CommitHash([]byte("in"), []byte("out"), []byte("model"))
	h2 := CommitHash([]byte("in"), []byte("out"), []byte("model"))
	if h1 != h2 {
		t.Fatal("commit hash not deterministic")
	}
}

func TestVerifyCommit(t *testing.T) {
	in, out, m := []byte("a"), []byte("b"), []byte("c")
	c := CommitHash(in, out, m)
	if !VerifyCommit(c, in, out, m) {
		t.Fatal("valid commit failed")
	}
	if VerifyCommit(c, []byte("x"), out, m) {
		t.Fatal("tampered input should fail")
	}
}

func TestDifferentInputs(t *testing.T) {
	h1 := CommitHash([]byte("a"), []byte("b"), []byte("c"))
	h2 := CommitHash([]byte("x"), []byte("b"), []byte("c"))
	if h1 == h2 {
		t.Fatal("different inputs produced same hash")
	}
}
