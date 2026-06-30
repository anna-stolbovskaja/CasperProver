package store

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/lib/pq"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
)

type PG struct {
	db *sql.DB
}

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
