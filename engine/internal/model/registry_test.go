package model

import (
	"testing"
	"time"
)

func TestModelVersion_String(t *testing.T) {
	tests := []struct {
		name     string
		version  ModelVersion
		expected string
	}{
		{"empty version", ModelVersion{}, "0.0.0"},
		{"major only", ModelVersion{Major: 1}, "1.0.0"},
		{"minor only", ModelVersion{Minor: 2}, "0.2.0"},
		{"patch only", ModelVersion{Patch: 3}, "0.0.3"},
		{"full version", ModelVersion{Major: 1, Minor: 2, Patch: 3}, "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.String()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestComputeModelHash(t *testing.T) {
	tests := []struct {
		name     string
		arch     []byte
		weights  []byte
		params   []byte
		expected string
	}{
		{"empty inputs", nil, nil, nil, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b85"},
		{"single byte", []byte{1}, nil, nil, "5feceb66ffc86f38d952786c6d696c79c2dbc239dd4e91b46729d73a27fb57e"},
		{"different bytes", []byte{1, 2, 3}, []byte{4, 5, 6}, []byte{7, 8, 9}, "3c08c4a64f7d7f4d3f4a3f4d3f4a3f4d3f4a3f4d3f4a3f4d3f4a3f4d3f4a3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeModelHash(tt.arch, tt.weights, tt.params)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("creates empty registry", func(t *testing.T) {
		r := New()
		if r == nil {
			t.Fatal("expected non-nil registry")
		}
		if len(r.models) != 0 {
			t.Errorf("expected empty models map, got %d entries", len(r.models))
		}
		if len(r.attestations) != 0 {
			t.Errorf("expected empty attestations map, got %d entries", len(r.attestations))
		}
		if len(r.byHash) != 0 {
			t.Errorf("expected empty byHash map, got %d entries", len(r.byHash))
		}
	})
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name        string
		regName     string
		regOwner    string
		regHash     string
		regIPFSCID  string
		regSchemas  []string
		expectError bool
		validate    func(*testing.T, *ModelEntry, error)
	}{
		{"valid registration", "test-model", "user1", "hash1", "ipfs123", nil, false, func(t *testing.T, m *ModelEntry, err error) {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if m == nil {
				t.Fatal("expected non-nil model entry")
			}
			if m.Name != "test-model" {
				t.Errorf("expected name 'test-model', got %q", m.Name)
			}
			if m.Owner != "user1" {
				t.Errorf("expected owner 'user1', got %q", m.Owner)
			}
			if m.Status != StatusActive {
				t.Errorf("expected status 'active', got %q", m.Status)
			}
			if m.AttestationCount != 0 {
				t.Errorf("expected attestation count 0, got %d", m.AttestationCount)
			}
		}},
		{"empty name", "", "user1", "hash1", "ipfs123", nil, true, nil},
		{"empty owner", "test-model", "", "hash1", "ipfs123", nil, true, nil},
		{"empty hash", "test-model", "user1", "", "ipfs123", nil, true, nil},
		{"duplicate hash", "test-model2", "user2", "hash1", "ipfs456", nil, true, nil},
		{"with schemas", "test-model", "user1", "hash2", "ipfs789", []string{"input-schema", "output-schema"}, false, func(t *testing.T, m *ModelEntry, err error) {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if m.InputSchema != "input-schema" {
				t.Errorf("expected input schema 'input-schema', got %q", m.InputSchema)
			}
			if m.OutputSchema != "output-schema" {
				t.Errorf("expected output schema 'output-schema', got %q", m.OutputSchema)
			}
		}},
	}

	r := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := r.Register(tt.regName, tt.regOwner, tt.regHash, tt.regIPFSCID, tt.regSchemas...)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err != nil)
			}
			if tt.validate != nil {
				tt.validate(t, m, err)
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	r := New()
	m1, _ := r.Register("model1", "user1", "hash1", "ipfs1")
	m2, _ := r.Register("model2", "user2", "hash2", "ipfs2")

	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{"existing model", m1.ID, true},
		{"another model", m2.ID, true},
		{"non-existent model", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := r.Get(tt.id)
			if ok != tt.expected {
				t.Errorf("expected ok=%v, got %v", tt.expected, ok)
			}
			if ok && got == nil {
				t.Error("expected non-nil model when ok is true")
			}
		})
	}
}

func TestRegistry_GetByHash(t *testing.T) {
	r := New()
	m1, _ := r.Register("model1", "user1", "hash1", "ipfs1")
	m2, _ := r.Register("model2", "user2", "hash2", "ipfs2")

	tests := []struct {
		name     string
		hash     string
		expected bool
	}{
		{"existing hash", "hash1", true},
		{"another hash", "hash2", true},
		{"non-existent hash", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := r.GetByHash(tt.hash)
			if ok != tt.expected {
				t.Errorf("expected ok=%v, got %v", tt.expected, ok)
			}
			if ok && got == nil {
				t.Error("expected non-nil model when ok is true")
			}
		})
	}
}

func TestRegistry_Attest(t *testing.T) {
	r := New()
	m1, _ := r.Register("model1", "user1", "hash1", "ipfs1")
	m2, _ := r.Register("model2", "user2", "hash2", "ipfs2")
	m3, _ := r.Register("model3", "user3", "hash3", "ipfs3")
	_ = r.Deprecate(m3.ID, "test deprecation")

	tests := []struct {
		name        string
		modelID     string
		attesterID  string
		evidence    string
		expectError bool
		validate    func(*testing.T, *ModelAttestation, error)
	}{
		{"valid attestation", m1.ID, "attester1", "evidence1", false, func(t *testing.T, a *ModelAttestation, err error) {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if a == nil {
				t.Fatal("expected non-nil attestation")
			}
			if a.ModelID != m1.ID {
				t.Errorf("expected model ID %q, got %q", m1.ID, a.ModelID)
			}
			if a.AttesterID != "attester1" {
				t.Errorf("expected attester ID 'attester1', got %q", a.AttesterID)
			}
			if a.Hash != m1.Hash {
				t.Errorf("expected hash %q, got %q", m1.Hash, a.Hash)
			}
			if !a.Timestamp.Before(time.Now()) {
				t.Error("expected timestamp to be in the past")
			}
			if !a.Valid {
				t.Error("expected attestation to be valid")
			}
			if a.Evidence != "evidence1" {
				t.Errorf("expected evidence 'evidence1', got %q", a.Evidence)
			}
		}},
		{"non-existent model", "nonexistent", "attester1", "evidence1", true, nil},
		{"deprecated model", m3.ID, "attester1", "evidence1", true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := r.Attest(tt.modelID, tt.attesterID, tt.evidence)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err != nil)
			}
			if tt.validate != nil {
				tt.validate(t, a, err)
			}
		})
	}
}

func TestRegistry_ListAttestations(t *testing.T) {
	r := New()
	m1, _ := r.Register("model1", "user1", "hash1", "ipfs1")

	a1, _ := r.Attest(m1.ID, "attester1", "evidence1")
	a2, _ := r.Attest(m1.ID, "attester2", "evidence2")

	tests := []struct {
		name       string
		modelID    string
		expectNil  bool
		expectLen  int
		validate   func(*testing.T, []*ModelAttestation)
	}{
		{"existing model", m1.ID, false, 2, func(t *testing.T, atts []*ModelAttestation) {
			if len(atts) != 2 {
				t.Errorf("expected 2 attestations, got %d", len(atts))
			}
			if atts[0].AttesterID != "attester1" && atts[1].AttesterID != "attester1" {
				t.Error("expected to find attester1 in attestations")
			}
			if atts[0].AttesterID != "attester2" && atts[1].AttesterID != "attester2" {
				t.Error("expected to find attester2 in attestations")
			}
		}},
		{"non-existent model", "nonexistent", true, 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atts := r.ListAttestations(tt.modelID)
			if (atts == nil) != tt.expectNil {
				t.Errorf("expected nil=%v, got %v", tt.expectNil, atts == nil)
			}
			if !tt.expectNil && len(atts) != tt.expectLen {
				t.Errorf("expected %d attestations, got %d", tt.expectLen, len(atts))
			}
			if tt.validate != nil {
				tt.validate(t, atts)
			}
		})
	}
}

func TestRegistry_Deprecate(t *testing.T) {
	r := New()
	m1, _ := r.Register("model1", "user1", "hash1", "ipfs1")
	m2, _ := r.Register("model2", "user2", "hash2", "ipfs2")

	tests := []struct {
		name        string
		modelID     string
		reason      string
		expectError bool
		validate    func(*testing.T, error)
	}{
		{"valid deprecation", m1.ID, "test reason", false, func(t *testing.T, err error) {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			m, ok := r.Get(m1.ID)
			if !ok {
				t.Fatal("expected to find model")
			}
			if m.Status != StatusDeprecated {
				t.Errorf("expected status 'deprecated', got %q", m.Status)
			}
		}},
		{"non-existent model", "nonexistent", "reason", true, nil},
		{"already deprecated", m1.ID, "another reason", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Deprecate(tt.modelID, tt.reason)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err != nil)
			}
			if tt.validate != nil {
				tt.validate(t, err)
			}
		})
	}
}

func TestRegistry_Revoke(t *testing.T) {
	r := New()
	m1, _ := r.Register("model1", "user1", "hash1", "ipfs1")
	m2, _ := r.Register("model2", "user2", "hash2", "ipfs2")

	tests := []struct {
		name        string
		modelID     string
		reason      string
		expectError bool
		validate    func(*testing.T, error)
	}{
		{"valid revocation", m1.ID, "test reason", false, func(t *testing.T, err error) {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			m, ok := r.Get(m1.ID)
			if !ok {
				t.Fatal("expected to find model")
			}
			if m.Status != StatusRevoked {
				t.Errorf("expected status 'revoked', got %q", m.Status)
			}
		}},
		{"non-existent model", "nonexistent", "reason", true, nil},
		{"already revoked", m1.ID, "another reason", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Revoke(tt.modelID, tt.reason)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err != nil)
			}
			if tt.validate != nil {
				tt.validate(t, err)
			}
		})
	}
}

func TestModelStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   ModelStatus
		expected string
	}{
		{"active status", StatusActive, "active"},
		{"deprecated status", StatusDeprecated, "deprecated"},
		{"revoked status", StatusRevoked, "revoked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.status)
			}
		})
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := New()
	m1, _ := r.Register("model1", "user1", "hash1", "ipfs1")

	done := make(chan bool)
	go func() {
		for i := 0; i < 10; i++ {
			_, _ = r.Register("model", "user", "hash", "ipfs")
			_, _ = r.Get(m1.ID)
			_, _ = r.GetByHash(m1.Hash)
			_, _ = r.Attest(m1.ID, "attester", "evidence")
			_ = r.ListAttestations(m1.ID)
			_ = r.Deprecate(m1.ID, "reason")
			_ = r.Revoke(m1.ID, "reason")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_, _ = r.Register("model", "user", "hash", "ipfs")
			_, _ = r.Get(m1.ID)
			_, _ = r.GetByHash(m1.Hash)
			_, _ = r.Attest(m1.ID, "attester", "evidence")
			_ = r.ListAttestations(m1.ID)
			_ = r.Deprecate(m1.ID, "reason")
			_ = r.Revoke(m1.ID, "reason")
		}
		done <- true
	}()

	<-done
	<-done
}

func TestRegistry_ModelEntry_Fields(t *testing.T) {
	now := time.Now()
	m := &ModelEntry{
		ID:               "id1",
		Name:             "test-model",
		Hash:             "hash1",
		Owner:            "user1",
		Version:          ModelVersion{Major: 1, Minor: 2, Patch: 3},
		IPFSCID:          "ipfs123",
		InputSchema:       "input-schema",
		OutputSchema:      "output-schema",
		RegisteredAt:      now,
		UpdatedAt:         now,
		Status:           StatusActive,
		AttestationCount:  5,
		LastAttestationAt: &now,
	}

	t.Run("model entry fields", func(t *testing.T) {
		if m.ID != "id1" {
			t.Errorf("expected ID 'id1', got %q", m.ID)
		}
		if m.Name != "test-model" {
			t.Errorf("expected Name 'test-model', got %q", m.Name)
		}
		if m.Hash != "hash1" {
			t.Errorf("expected Hash 'hash1', got %q", m.Hash)
		}
		if m.Owner != "user1" {
			t.Errorf("expected Owner 'user1', got %q", m.Owner)
		}
		if m.Version.String() != "1.2.3" {
			t.Errorf("expected Version '1.2.3', got %q", m.Version.String())
		}
		if m.IPFSCID != "ipfs123" {
			t.Errorf("expected IPFSCID 'ipfs123', got %q", m.IPFSCID)
		}
		if m.InputSchema != "input-schema" {
			t.Errorf("expected InputSchema 'input-schema', got %q", m.InputSchema)
		}
		if m.OutputSchema != "output-schema" {
			t.Errorf("expected OutputSchema 'output-schema', got %q", m.OutputSchema)
		}
		if !m.RegisteredAt.Equal(now) {
			t.Error("expected RegisteredAt to match")
		}
		if !m.UpdatedAt.Equal(now) {
			t.Error("expected UpdatedAt to match")
		}
		if m.Status != StatusActive {
			t.Errorf("expected Status 'active', got %q", m.Status)
		}
		if m.AttestationCount != 5 {
			t.Errorf("expected AttestationCount 5, got %d", m.AttestationCount)
		}
		if m.LastAttestationAt == nil || !m.LastAttestationAt.Equal(now) {
			t.Error("expected LastAttestationAt to match")
		}
	})
}

func TestRegistry_ModelAttestation_Fields(t *testing.T) {
	now := time.Now()
	a := &ModelAttestation{
		ModelID:    "model1",
		AttesterID: "attester1",
		Hash:       "hash1",
		Timestamp:  now,
		Valid:      true,
		Evidence:   "evidence1",
	}

	t.Run("model attestation fields", func(t *testing.T) {
		if a.ModelID != "model1" {
			t.Errorf("expected ModelID 'model1', got %q", a.ModelID)
		}
		if a.AttesterID != "attester1" {
			t.Errorf("expected AttesterID 'attester1', got %q", a.AttesterID)
		}
		if a.Hash != "hash1" {
			t.Errorf("expected Hash 'hash1', got %q", a.Hash)
		}
		if !a.Timestamp.Equal(now) {
			t.Error("expected Timestamp to match")
		}
		if !a.Valid {
			t.Error("expected Valid to be true")
		}
		if a.Evidence != "evidence1" {
			t.Errorf("expected Evidence 'evidence1', got %q", a.Evidence)
		}
	})
}

func TestRegistry_UpdateModel(t *testing.T) {
	r := New()
	m, _ := r.Register("model1", "user1", "hash1", "ipfs1")

	t.Run("update model fields", func(t *testing.T) {
		m.Name = "updated-model"
		m.Owner = "user2"
		m.Status = StatusDeprecated
		m.AttestationCount = 10
		now := time.Now()
		m.UpdatedAt = now
		m.LastAttestationAt = &now

		got, ok := r.Get(m.ID)
		if !ok {
			t.Fatal("expected to find model")
		}
		if got.Name != "updated-model" {
			t.Errorf("expected Name 'updated-model', got %q", got.Name)
		}
		if got.Owner != "user2" {
			t.Errorf("expected Owner 'user2', got %q", got.Owner)
		}
		if got.Status != StatusDeprecated {
			t.Errorf("expected Status 'deprecated', got %q", got.Status)
		}
		if got.AttestationCount != 10 {
			t.Errorf("expected AttestationCount 10, got %d", got.AttestationCount)
		}
		if !got.UpdatedAt.Equal(now) {
			t.Error("expected UpdatedAt to match")
		}
		if got.LastAttestationAt == nil || !got.LastAttestationAt.Equal(now) {
			t.Error("expected LastAttestationAt to match")
		}
	})
}

func TestRegistry_MultipleAttestations(t *testing.T) {
	r := New()
	m, _ := r.Register("model1", "user1", "hash1", "ipfs1")

	// Add multiple attestations
	for i := 0; i < 5; i++ {
		_, err := r.Attest(m.ID, fmt.Sprintf("attester%d", i), fmt.Sprintf("evidence%d", i))
		if err != nil {
			t.Fatalf("failed to attest: %v", err)
		}
	}

	t.Run("verify multiple attestations", func(t *testing.T) {
		atts := r.ListAttestations(m.ID)
		if len(atts) != 5 {
			t.Errorf("expected 5 attestations, got %d", len(atts))
		}

		// Check that all attestations are present
		attesterIDs := make(map[string]bool)
		for _, a := range atts {
			attesterIDs[a.AttesterID] = true
		}

		for i := 0; i < 5; i++ {
			id := fmt.Sprintf("attester%d", i)
			if !attesterIDs[id] {
				t.Errorf("expected to find attester %s", id)
			}
		}
	})
}
