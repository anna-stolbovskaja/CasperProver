package inference

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelRegistryEntryUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    ModelRegistryEntry
	}{
		{
			name: "valid json",
			jsonStr: `{
				"model_id": "model-1",
				"model_hash": "hash-1",
				"verifier_contract": "contract-1"
			}`,
			want: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
		},
		{
			name: "invalid json",
			jsonStr: `{
				"model_id": "model-1",
				"model_hash": "hash-1"
			}`,
			want: ModelRegistryEntry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m ModelRegistryEntry
			err := json.Unmarshal([]byte(tt.jsonStr), &m)
			if tt.name == "invalid json" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, m)
			}
		})
	}
}

func TestModelRegistryEntryMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		m    ModelRegistryEntry
		want string
	}{
		{
			name: "valid model",
			m: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
			want: `{"model_id":"model-1","model_hash":"hash-1","verifier_contract":"contract-1"}`,
		},
		{
			name: "empty model",
			m:   ModelRegistryEntry{},
			want: `{"model_id":"","model_hash":"","verifier_contract":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr, err := json.Marshal(tt.m)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(jsonStr))
		})
	}
}

func TestNewModelRegistryEntry(t *testing.T) {
	tests := []struct {
		name string
		modelID          string
		modelHash        string
		verifierContract string
		want    ModelRegistryEntry
	}{
		{
			name: "valid model",
			modelID:          "model-1",
			modelHash:        "hash-1",
			verifierContract: "contract-1",
			want: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
		},
		{
			name: "empty model",
			modelID:          "",
			modelHash:        "",
			verifierContract: "",
			want: ModelRegistryEntry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModelRegistryEntry(tt.modelID, tt.modelHash, tt.verifierContract)
			assert.Equal(t, tt.want, m)
		})
	}
}

func TestModelRegistryEntryValidate(t *testing.T) {
	tests := []struct {
		name    string
		m       ModelRegistryEntry
		wantErr bool
	}{
		{
			name: "valid model",
			m: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
			wantErr: false,
		},
		{
			name: "empty model",
			m:   ModelRegistryEntry{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.m.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestModelRegistryEntryString(t *testing.T) {
	tests := []struct {
		name string
		m    ModelRegistryEntry
		want string
	}{
		{
			name: "valid model",
			m: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
			want: "ModelRegistryEntry{ModelID:model-1, ModelHash:hash-1, VerifierContract:contract-1}",
		},
		{
			name: "empty model",
			m:   ModelRegistryEntry{},
			want: "ModelRegistryEntry{ModelID:, ModelHash:, VerifierContract:}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.m.String())
		})
	}
}

func TestModelRegistryEntryEqual(t *testing.T) {
	tests := []struct {
		name string
		m1   ModelRegistryEntry
		m2   ModelRegistryEntry
		want bool
	}{
		{
			name: "equal models",
			m1: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
			m2: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
			want: true,
		},
		{
			name: "different models",
			m1: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
			m2: ModelRegistryEntry{
				ModelID:          "model-2",
				ModelHash:        "hash-2",
				VerifierContract: "contract-2",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.m1.Equal(tt.m2))
		})
	}
}

func TestModelRegistryEntryGetModelID(t *testing.T) {
	tests := []struct {
		name string
		m    ModelRegistryEntry
		want string
	}{
		{
			name: "valid model",
			m: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
			want: "model-1",
		},
		{
			name: "empty model",
			m:   ModelRegistryEntry{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.m.GetModelID())
		})
	}
}

func TestModelRegistryEntryGetModelHash(t *testing.T) {
	tests := []struct {
		name string
		m    ModelRegistryEntry
		want string
	}{
		{
			name: "valid model",
			m: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
			want: "hash-1",
		},
		{
			name: "empty model",
			m:   ModelRegistryEntry{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.m.GetModelHash())
		})
	}
}

func TestModelRegistryEntryGetVerifierContract(t *testing.T) {
	tests := []struct {
		name string
		m    ModelRegistryEntry
		want string
	}{
		{
			name: "valid model",
			m: ModelRegistryEntry{
				ModelID:          "model-1",
				ModelHash:        "hash-1",
				VerifierContract: "contract-1",
			},
			want: "contract-1",
		},
		{
			name: "empty model",
			m:   ModelRegistryEntry{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.m.GetVerifierContract())
		})
	}
}
