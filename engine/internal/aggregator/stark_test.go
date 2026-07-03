package aggregator

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStarkAggregator(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "valid aggregator",
			wantErr: false,
		},
		{
			name:    "invalid aggregator",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewStarkAggregator()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStarkAggregatorAggregate(t *testing.T) {
	tests := []struct {
		name    string
		inputs  [][]byte
		want    []byte
		wantErr bool
	}{
		{
			name:    "valid inputs",
			inputs:  [][]byte{{1, 2, 3}, {4, 5, 6}},
			want:    []byte{1, 2, 3, 4, 5, 6},
			wantErr: false,
		},
		{
			name:    "invalid inputs",
			inputs:  [][]byte{},
			want:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aggregator, err := NewStarkAggregator()
			if err != nil {
				t.Fatal(err)
			}
			output, err := aggregator.Aggregate(tt.inputs)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, output)
			}
		})
	}
}

func TestStarkAggregatorVerify(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		proof   []byte
		want    bool
		wantErr bool
	}{
		{
			name:    "valid input and proof",
			input:   []byte{1, 2, 3},
			proof:   []byte{4, 5, 6},
			want:    true,
			wantErr: false,
		},
		{
			name:    "invalid input and proof",
			input:   []byte{},
			proof:   []byte{},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aggregator, err := NewStarkAggregator()
			if err != nil {
				t.Fatal(err)
			}
			valid, err := aggregator.Verify(tt.input, tt.proof)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, valid)
			}
		})
	}
}

func TestSha256Hash(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
	}{
		{
			name:    "valid input",
			input:   []byte{1, 2, 3},
			want:    "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1ba9fe6b9abf4e913a71",
		},
		{
			name:    "empty input",
			input:   []byte{},
			want:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := Sha256Hash(tt.input)
			assert.Equal(t, tt.want, hash)
		})
	}
}

func TestHexEncode(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
	}{
		{
			name:    "valid input",
			input:   []byte{1, 2, 3},
			want:    "010203",
		},
		{
			name:    "empty input",
			input:   []byte{},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := HexEncode(tt.input)
			assert.Equal(t, tt.want, encoded)
		})
	}
}

func TestHexDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{
			name:    "valid input",
			input:   "010203",
			want:    []byte{1, 2, 3},
			wantErr: false,
		},
		{
			name:    "invalid input",
			input:   "abcdef",
			want:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := HexDecode(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, decoded)
			}
		})
	}
}
