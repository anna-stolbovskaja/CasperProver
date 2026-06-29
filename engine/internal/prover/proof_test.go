package prover

import "testing"

func TestGenerate(t *testing.T) {
	eng := New()
	p := eng.Generate("agent-1", []byte("in"), []byte("out"), []byte("m"), "test")
	if p.ID != "P-1" {
		t.Fatalf("want P-1 got %s", p.ID)
	}
	if !p.Valid || p.Revoked {
		t.Fatal("new proof should be valid")
	}
	if p.PH == "" || p.IH == "" || p.OH == "" || p.MH == "" {
		t.Fatal("missing hashes")
	}
}

func TestGet(t *testing.T) {
	eng := New()
	eng.Generate("a", []byte("1"), []byte("2"), []byte("3"), "uc")
	p, ok := eng.Get("P-1")
	if !ok || p == nil {
		t.Fatal("proof not found")
	}
}

func TestGetMissing(t *testing.T) {
	eng := New()
	_, ok := eng.Get("P-999")
	if ok {
		t.Fatal("should not find missing")
	}
}

func TestRevoke(t *testing.T) {
	eng := New()
	eng.Generate("a", []byte("1"), []byte("2"), []byte("3"), "uc")
	err := eng.Revoke("P-1", "test")
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	p, _ := eng.Get("P-1")
	if p.Valid || !p.Revoked {
		t.Fatal("should be revoked")
	}
}

func TestDoubleRevoke(t *testing.T) {
	eng := New()
	eng.Generate("a", []byte("1"), []byte("2"), []byte("3"), "uc")
	eng.Revoke("P-1", "r1")
	err := eng.Revoke("P-1", "r2")
	if err == nil {
		t.Fatal("double revoke should fail")
	}
}

func TestVerify(t *testing.T) {
	eng := New()
	eng.Generate("a", []byte("1"), []byte("2"), []byte("3"), "uc")
	ok, err := eng.Verify("P-1")
	if err != nil || !ok {
		t.Fatal("should verify")
	}
}

func TestVerifyRevoked(t *testing.T) {
	eng := New()
	eng.Generate("a", []byte("1"), []byte("2"), []byte("3"), "uc")
	eng.Revoke("P-1", "r")
	ok, _ := eng.Verify("P-1")
	if ok {
		t.Fatal("revoked should not verify")
	}
}

func TestList(t *testing.T) {
	eng := New()
	eng.Generate("a", []byte("1"), []byte("2"), []byte("3"), "uc")
	eng.Generate("b", []byte("4"), []byte("5"), []byte("6"), "uc")
	all := eng.List()
	if len(all) != 2 {
		t.Fatalf("want 2, got %d", len(all))
	}
}

func TestSequentialIDs(t *testing.T) {
	eng := New()
	p1 := eng.Generate("a", []byte("1"), []byte("2"), []byte("3"), "uc")
	p2 := eng.Generate("a", []byte("4"), []byte("5"), []byte("6"), "uc")
	if p1.ID == p2.ID {
		t.Fatal("ids should be unique")
	}
}
