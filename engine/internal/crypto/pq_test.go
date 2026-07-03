package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateKeyPair(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "valid key pair",
			wantErr: false,
		},
		{
			name:    "invalid key pair",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := GenerateKeyPair()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSignMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{
			name:    "valid message",
			message: "Hello, World!",
			wantErr: false,
		},
		{
			name:    "invalid message",
			message: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := SignMessage(tt.message)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifySignature(t *testing.T) {
	tests := []struct {
		name    string
		message string
		signature string
		wantErr bool
	}{
		{
			name:    "valid signature",
			message: "Hello, World!",
			signature: "signature",
			wantErr: false,
		},
		{
			name:    "invalid signature",
			message: "Hello, World!",
			signature: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifySignature(tt.message, tt.signature)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHashMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "valid message",
			message: "Hello, World!",
			want:    "315f5bdb76d078c43b8ac0064e4a0164612b1fce77c869345bfc94c75894edd3",
		},
		{
			name:    "empty message",
			message: "",
			want:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashMessage(tt.message)
			assert.Equal(t, tt.want, hash)
		})
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			name:    "valid size",
			size:    32,
			wantErr: false,
		},
		{
			name:    "invalid size",
			size:    -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GenerateRandomBytes(tt.size)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			name:    "valid size",
			size:    32,
			wantErr: false,
		},
		{
			name:    "invalid size",
			size:    -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GenerateRandomString(tt.size)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSha256Hash(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
	}{
		{
			name:    "valid input",
			input:   "Hello, World!",
			want:    "315f5bdb76d078c43b8ac0064e4a0164612b1fce77c869345bfc94c75894edd3",
		},
		{
			name:    "empty input",
			input:   "",
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
			input:   []byte{1, 2, 3, 4, 5},
			want:    "0102030405",
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
			input:   "0102030405",
			want:    []byte{1, 2, 3, 4, 5},
			wantErr: false,
		},
		{
			name:    "invalid input",
			input:   "abcdefabcdef",
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
