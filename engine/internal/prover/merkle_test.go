package prover

import "testing"

func TestBuildTree(t *testing.T) {
	leaves := [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")}
	tree := BuildTree(leaves)
	if tree == nil {
		t.Fatal("nil tree")
	}
	if tree.Left == nil || tree.Right == nil {
		t.Fatal("missing children")
	}
}

func TestBuildTreeEmpty(t *testing.T) {
	tree := BuildTree(nil)
	if tree != nil {
		t.Fatal("expected nil for empty")
	}
}

func TestRoot(t *testing.T) {
	leaves := [][]byte{[]byte("a"), []byte("b")}
	r := Root(leaves)
	if len(r) != 64 {
		t.Fatalf("want 64 hex chars, got %d", len(r))
	}
}

func TestRootDeterministic(t *testing.T) {
	leaves := [][]byte{[]byte("x"), []byte("y"), []byte("z")}
	r1 := Root(leaves)
	r2 := Root(leaves)
	if r1 != r2 {
		t.Fatal("root not deterministic")
	}
}

func TestGetPath(t *testing.T) {
	leaves := [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")}
	path := GetPath(leaves, 0)
	if len(path) == 0 {
		t.Fatal("empty path")
	}
}

func TestVerifyPath(t *testing.T) {
	leaves := [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")}
	root := Root(leaves)
	path := GetPath(leaves, 0)
	if !VerifyPath([]byte("a"), path, root, 0) {
		t.Fatal("valid path failed")
	}
}

func TestVerifyPathTampered(t *testing.T) {
	leaves := [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")}
	root := Root(leaves)
	path := GetPath(leaves, 0)
	if VerifyPath([]byte("tampered"), path, root, 0) {
		t.Fatal("tampered leaf should fail")
	}
}

func TestOddLeaves(t *testing.T) {
	leaves := [][]byte{[]byte("a"), []byte("b"), []byte("c")}
	r := Root(leaves)
	if len(r) != 64 {
		t.Fatal("odd leaves failed")
	}
}
