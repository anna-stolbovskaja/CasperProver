package api

import (
	"context"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/store"
)

// storeAPIKeyRecord mirrors store.APIKeyRecord one-to-one. Duplicating
// the struct lets api unit-tests build a fake apiKeyStore without
// importing the Postgres-bound store package, and lets store return
// its own type without the api package leaking into it. The mapping
// helpers below keep the two in lockstep.
type storeAPIKeyRecord struct {
	ID        string
	KeyHash   string
	Wallet    string
	Scope     string
	CreatedAt int64
	Revoked   bool
	RevokedAt int64
}

func fromStoreRecord(r *store.APIKeyRecord) *storeAPIKeyRecord {
	if r == nil {
		return nil
	}
	return &storeAPIKeyRecord{
		ID:        r.ID,
		KeyHash:   r.KeyHash,
		Wallet:    r.Wallet,
		Scope:     r.Scope,
		CreatedAt: r.CreatedAt,
		Revoked:   r.Revoked,
		RevokedAt: r.RevokedAt,
	}
}

func toStoreRecord(r *storeAPIKeyRecord) *store.APIKeyRecord {
	if r == nil {
		return nil
	}
	return &store.APIKeyRecord{
		ID:        r.ID,
		KeyHash:   r.KeyHash,
		Wallet:    r.Wallet,
		Scope:     r.Scope,
		CreatedAt: r.CreatedAt,
		Revoked:   r.Revoked,
		RevokedAt: r.RevokedAt,
	}
}

// storeWalletChallengeRecord mirrors store.WalletChallengeRecord for
// the same reason as storeAPIKeyRecord.
type storeWalletChallengeRecord struct {
	Nonce     string // hex-encoded 32 random bytes
	Wallet    string
	CreatedAt int64
	ExpiresAt int64
	ConsumedAt int64 // 0 = not consumed yet
}

func fromStoreChallenge(r *store.WalletChallengeRecord) *storeWalletChallengeRecord {
	if r == nil {
		return nil
	}
	return &storeWalletChallengeRecord{
		Nonce:      r.Nonce,
		Wallet:     r.Wallet,
		CreatedAt:  r.CreatedAt,
		ExpiresAt:  r.ExpiresAt,
		ConsumedAt: r.ConsumedAt,
	}
}

func toStoreChallenge(r *storeWalletChallengeRecord) *store.WalletChallengeRecord {
	if r == nil {
		return nil
	}
	return &store.WalletChallengeRecord{
		Nonce:      r.Nonce,
		Wallet:     r.Wallet,
		CreatedAt:  r.CreatedAt,
		ExpiresAt:  r.ExpiresAt,
		ConsumedAt: r.ConsumedAt,
	}
}

// apiKeyStore is the tiny subset of the store surface the API layer
// needs for per-wallet keys and their signed-challenge lifecycle.
// Extracted so unit tests can inject an in-memory fake.
type apiKeyStore interface {
	InsertAPIKey(ctx context.Context, rec *store.APIKeyRecord) error
	LookupAPIKeyByHash(ctx context.Context, keyHash string) (*store.APIKeyRecord, error)
	RevokeAPIKey(ctx context.Context, id string, revokedAt int64) error

	InsertWalletChallenge(ctx context.Context, rec *store.WalletChallengeRecord) error
	LookupWalletChallenge(ctx context.Context, nonce string) (*store.WalletChallengeRecord, error)
	MarkWalletChallengeConsumed(ctx context.Context, nonce string, consumedAt int64) error
}

// keyStore returns the interface-typed handle. If s.keys is nil the
// Postgres-backed s.db is used. Tests set s.keys directly.
func (s *Server) keyStore() apiKeyStore {
	if s.keys != nil {
		return s.keys
	}
	if s.db != nil {
		return s.db
	}
	return nil
}

func (s *Server) insertAPIKey(ctx context.Context, rec *storeAPIKeyRecord) error {
	ks := s.keyStore()
	if ks == nil {
		return errNoKeyStore
	}
	return ks.InsertAPIKey(ctx, toStoreRecord(rec))
}

func (s *Server) lookupAPIKey(ctx context.Context, keyHash string) (*storeAPIKeyRecord, error) {
	ks := s.keyStore()
	if ks == nil {
		return nil, errNoKeyStore
	}
	r, err := ks.LookupAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}
	return fromStoreRecord(r), nil
}

func (s *Server) revokeAPIKey(ctx context.Context, id string, revokedAt int64) error {
	ks := s.keyStore()
	if ks == nil {
		return errNoKeyStore
	}
	return ks.RevokeAPIKey(ctx, id, revokedAt)
}

func (s *Server) insertWalletChallenge(ctx context.Context, rec *storeWalletChallengeRecord) error {
	ks := s.keyStore()
	if ks == nil {
		return errNoKeyStore
	}
	return ks.InsertWalletChallenge(ctx, toStoreChallenge(rec))
}

func (s *Server) lookupWalletChallenge(ctx context.Context, nonce string) (*storeWalletChallengeRecord, error) {
	ks := s.keyStore()
	if ks == nil {
		return nil, errNoKeyStore
	}
	r, err := ks.LookupWalletChallenge(ctx, nonce)
	if err != nil {
		return nil, err
	}
	return fromStoreChallenge(r), nil
}

func (s *Server) markWalletChallengeConsumed(ctx context.Context, nonce string, consumedAt int64) error {
	ks := s.keyStore()
	if ks == nil {
		return errNoKeyStore
	}
	return ks.MarkWalletChallengeConsumed(ctx, nonce, consumedAt)
}

// errNoKeyStore is returned when neither an injected fake nor a
// Postgres connection is available (e.g. DATABASE_URL not set).
var errNoKeyStore = errNoKeyStoreVal{}

type errNoKeyStoreVal struct{}

func (errNoKeyStoreVal) Error() string { return "api key store unavailable" }
