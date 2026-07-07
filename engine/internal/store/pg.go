package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lib/pq"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
)

type PG struct {
	db *sql.DB
}

// schemaDDL creates all tables used by the engine if they don't already exist.
// This makes a fresh DATABASE_URL (local dev, CI, a new Render Postgres instance)
// self-provisioning instead of relying on manual/undocumented DDL.
const schemaDDL = `
CREATE TABLE IF NOT EXISTS proofs (
	id text PRIMARY KEY,
	agent text NOT NULL,
	proof_hash text NOT NULL,
	input_hash text NOT NULL,
	output_hash text NOT NULL,
	model_hash text NOT NULL,
	merkle_root text NOT NULL,
	merkle_path text[] NOT NULL DEFAULT '{}',
	leaf_index integer NOT NULL DEFAULT 0,
	created_at bigint NOT NULL,
	valid boolean NOT NULL DEFAULT true,
	revoked boolean NOT NULL DEFAULT false,
	use_case text NOT NULL DEFAULT '',
	public_key text NOT NULL DEFAULT '',
	deploy_hash text NOT NULL DEFAULT '',
	generation_ms bigint NOT NULL DEFAULT 0,
	mode text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS kyc_whitelist (
	user_id text PRIMARY KEY,
	proof_id text NOT NULL,
	whitelisted boolean NOT NULL DEFAULT true,
	granted_at bigint NOT NULL
);

CREATE TABLE IF NOT EXISTS model_registry (
	model_id text PRIMARY KEY,
	model_hash text NOT NULL,
	verifier_contract text NOT NULL,
	metadata jsonb NOT NULL DEFAULT '{}',
	registered_at bigint NOT NULL,
	deploy_hash text NOT NULL DEFAULT ''
);
`

// Open connects to PostgreSQL using DATABASE_URL. Returns nil,nil if unset.
func Open() (*PG, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, nil
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("pg open: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pg ping: %w", err)
	}
	if _, err := db.Exec(schemaDDL); err != nil {
		return nil, fmt.Errorf("pg schema init: %w", err)
	}
	return &PG{db: db}, nil
}

func (s *PG) Close() {
	if s != nil && s.db != nil {
		s.db.Close()
	}
}

// Load reads all proofs from the database into the engine.
func (s *PG) Load(eng *prover.ProofEngine) (int, error) {
	rows, err := s.db.Query(`SELECT id, agent, proof_hash, input_hash, output_hash, model_hash,
		merkle_root, merkle_path, leaf_index, created_at, valid, revoked, use_case,
		COALESCE(public_key,''), COALESCE(deploy_hash,''), generation_ms, mode
		FROM proofs ORDER BY created_at`)
	if err != nil {
		return 0, fmt.Errorf("pg load query: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var p prover.Proof
		var path pq.StringArray
		err := rows.Scan(&p.ID, &p.Agent, &p.PH, &p.IH, &p.OH, &p.MH,
			&p.Root, &path, &p.Idx, &p.TS, &p.Valid, &p.Revoked, &p.UseCase,
			&p.PubKey, &p.Deploy, &p.GenMs, &p.Mode)
		if err != nil {
			return count, fmt.Errorf("pg load scan: %w", err)
		}
		p.Path = []string(path)
		eng.Restore(&p)
		count++
	}
	return count, rows.Err()
}

// Save inserts a new proof or updates on conflict.
func (s *PG) Save(p *prover.Proof) error {
	_, err := s.db.Exec(`INSERT INTO proofs
		(id, agent, proof_hash, input_hash, output_hash, model_hash,
		 merkle_root, merkle_path, leaf_index, created_at, valid, revoked,
		 use_case, public_key, deploy_hash, generation_ms, mode)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (id) DO UPDATE SET
			valid=EXCLUDED.valid, revoked=EXCLUDED.revoked, deploy_hash=EXCLUDED.deploy_hash`,
		p.ID, p.Agent, p.PH, p.IH, p.OH, p.MH,
		p.Root, pq.StringArray(p.Path), p.Idx, p.TS, p.Valid, p.Revoked,
		p.UseCase, p.PubKey, p.Deploy, p.GenMs, p.Mode)
	return err
}

// Update persists changes to valid/revoked/deploy_hash.
func (s *PG) Update(p *prover.Proof) error {
	_, err := s.db.Exec(`UPDATE proofs SET valid=$1, revoked=$2, deploy_hash=$3 WHERE id=$4`,
		p.Valid, p.Revoked, p.Deploy, p.ID)
	return err
}

// SaveKYC persists a KYC whitelist entry.
func (s *PG) SaveKYC(user, proofID string, ts int64) error {
	_, err := s.db.Exec(`INSERT INTO kyc_whitelist (user_id, proof_id, whitelisted, granted_at)
		VALUES ($1,$2,TRUE,$3) ON CONFLICT (user_id) DO UPDATE SET proof_id=EXCLUDED.proof_id, granted_at=EXCLUDED.granted_at`,
		user, proofID, ts)
	return err
}

// KYCEntry is a persisted KYC whitelist row.
type KYCEntry struct {
	User    string
	ProofID string
}

// LoadKYC reads all currently-whitelisted users from the database. Used to
// rehydrate the in-memory whitelist on server start - previously this table
// was write-only (SaveKYC was called but nothing ever read it back), so every
// restart silently lost every KYC grant even though the row was sitting in
// Postgres the whole time.
func (s *PG) LoadKYC() ([]KYCEntry, error) {
	rows, err := s.db.Query(`SELECT user_id, proof_id FROM kyc_whitelist WHERE whitelisted = TRUE`)
	if err != nil {
		return nil, fmt.Errorf("pg load kyc query: %w", err)
	}
	defer rows.Close()

	var out []KYCEntry
	for rows.Next() {
		var e KYCEntry
		if err := rows.Scan(&e.User, &e.ProofID); err != nil {
			return out, fmt.Errorf("pg load kyc scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ModelRegistryEntry is the persisted record for an on-chain-bound AI model.
// (Defined here, not in package inference, so store has no import-cycle back
// to its own caller; inference.ModelRegistryEntry is converted to/from this.)
type ModelRegistryEntry struct {
	ModelID          string
	ModelHash        string
	VerifierContract string
	Metadata         map[string]string
	RegisteredAt     int64
	DeployHash       string
}

// SaveProof is a context-aware alias for Save, for callers that pass a ctx
// (the underlying database/sql call itself is not yet context-cancellable
// here; kept as a thin wrapper so call sites can pass ctx uniformly).
func (s *PG) SaveProof(ctx context.Context, p *prover.Proof) error {
	return s.Save(p)
}

// SaveModelRegistryEntry inserts or updates a model registry row.
func (s *PG) SaveModelRegistryEntry(ctx context.Context, e *ModelRegistryEntry) error {
	meta, err := json.Marshal(e.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO model_registry
		(model_id, model_hash, verifier_contract, metadata, registered_at, deploy_hash)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (model_id) DO UPDATE SET
			model_hash=EXCLUDED.model_hash, verifier_contract=EXCLUDED.verifier_contract,
			metadata=EXCLUDED.metadata, deploy_hash=EXCLUDED.deploy_hash`,
		e.ModelID, e.ModelHash, e.VerifierContract, meta, e.RegisteredAt, e.DeployHash)
	return err
}

// GetModelRegistryEntry looks up a model registry row by model ID.
func (s *PG) GetModelRegistryEntry(ctx context.Context, modelID string) (*ModelRegistryEntry, error) {
	var e ModelRegistryEntry
	var meta []byte
	err := s.db.QueryRowContext(ctx, `SELECT model_id, model_hash, verifier_contract, metadata, registered_at, deploy_hash
		FROM model_registry WHERE model_id=$1`, modelID).
		Scan(&e.ModelID, &e.ModelHash, &e.VerifierContract, &meta, &e.RegisteredAt, &e.DeployHash)
	if err != nil {
		return nil, err
	}
	if len(meta) > 0 {
		if err := json.Unmarshal(meta, &e.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return &e, nil
}
