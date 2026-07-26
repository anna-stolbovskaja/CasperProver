// Package submitter builds and signs Casper 2.0 (Condor) TransactionV1
// payloads using the official casper-go-sdk/v2 and submits them to a
// Casper node via JSON-RPC.
package submitter

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/make-software/casper-go-sdk/v2/casper"
	"github.com/make-software/casper-go-sdk/v2/rpc"
	"github.com/make-software/casper-go-sdk/v2/types"
	"github.com/make-software/casper-go-sdk/v2/types/clvalue"
	"github.com/make-software/casper-go-sdk/v2/types/key"
	"github.com/make-software/casper-go-sdk/v2/types/keypair"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
)

const (
	defaultTTL       = 30 * time.Minute
	defaultGasAmount = 3_000_000_000 // 3 CSPR
)

// CasperSubmitter signs and submits TransactionV1 calls to on-chain
// CasperProver contracts via the casper-go-sdk RPC client.
type CasperSubmitter struct {
	chain  string
	keys   keypair.PrivateKey
	client rpc.Client
}

// New creates a CasperSubmitter. keyPath must point to a PEM file
// (ED25519 or SECP256K1) whose corresponding account is the contract
// deployer / has the named keys for the target contracts.
func New(nodeURL, chain, keyPath string) *CasperSubmitter {
	// Try secp256k1 first (the project's deployer keys are secp256k1),
	// then fall back to ed25519.
	keys, err := casper.NewSECP256k1PrivateKeyFromPEMFile(keyPath)
	if err != nil {
		keys, err = casper.NewED25519PrivateKeyFromPEMFile(keyPath)
		if err != nil {
			slog.Error("failed to load deployer key", "path", keyPath, "err", err)
			return nil
		}
		slog.Info("submitter loaded ED25519 key", "path", keyPath)
	} else {
		slog.Info("submitter loaded SECP256K1 key", "path", keyPath)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	rpcClient := rpc.NewClient(rpc.NewHttpHandler(nodeURL+"/rpc", httpClient))

	return &CasperSubmitter{
		chain:  chain,
		keys:   keys,
		client: rpcClient,
	}
}

// putTransaction builds, signs, and submits a TransactionV1 that calls
// the given entry point on a contract identified by its hex-encoded hash.
func (s *CasperSubmitter) putTransaction(contractHash, entryPoint string, args *types.Args) (string, error) {
	pubKey := s.keys.PublicKey()

	hashBytes, err := hex.DecodeString(contractHash)
	if err != nil {
		return "", fmt.Errorf("decode contract hash: %w", err)
	}
	var hash key.Hash
	copy(hash[:], hashBytes)

	ep := entryPoint
	payload, err := types.NewTransactionV1Payload(
		types.InitiatorAddr{PublicKey: &pubKey},
		types.Timestamp(time.Now().UTC()),
		types.Duration(defaultTTL),
		s.chain,
		types.PricingMode{
			Limited: &types.LimitedMode{
				GasPriceTolerance: 1,
				StandardPayment:   true,
				PaymentAmount:     defaultGasAmount,
			},
		},
		types.NewNamedArgs(args),
		types.TransactionTarget{
			Stored: &types.StoredTarget{
				ID:      types.TransactionInvocationTarget{ByHash: &hash},
				Runtime: types.NewVmCasperV1TransactionRuntime(),
			},
		},
		types.TransactionEntryPoint{Custom: &ep},
		types.TransactionScheduling{Standard: &struct{}{}},
	)
	if err != nil {
		return "", fmt.Errorf("build payload: %w", err)
	}

	tx, err := types.MakeTransactionV1(payload)
	if err != nil {
		return "", fmt.Errorf("make transaction: %w", err)
	}

	if err := tx.Sign(s.keys); err != nil {
		return "", fmt.Errorf("sign transaction: %w", err)
	}

	res, err := s.client.PutTransactionV1(context.Background(), *tx)
	if err != nil {
		return "", fmt.Errorf("put transaction: %w", err)
	}

	txHash := res.TransactionHash.TransactionV1.ToHex()
	slog.Info("transaction submitted", "hash", txHash, "entry_point", entryPoint)
	return txHash, nil
}

// Submit anchors a proof to the on-chain proof_registry contract.
// The contract hash is read from CONTRACT_PROOF_REGISTRY env var.
func (s *CasperSubmitter) Submit(p *prover.Proof) (string, error) {
	contractHash := os.Getenv("CONTRACT_PROOF_REGISTRY")
	if contractHash == "" {
		return "", fmt.Errorf("CONTRACT_PROOF_REGISTRY env var not set")
	}

	args := &types.Args{}
	args.AddArgument("proof_hash", *clvalue.NewCLString(p.PH)).
		AddArgument("input_hash", *clvalue.NewCLString(p.IH)).
		AddArgument("output_hash", *clvalue.NewCLString(p.OH)).
		AddArgument("model_hash", *clvalue.NewCLString(p.MH))

	return s.putTransaction(contractHash, "submit_proof", args)
}

// Revoke marks a proof as revoked on-chain.
func (s *CasperSubmitter) Revoke(pid, reason string) (string, error) {
	contractHash := os.Getenv("CONTRACT_PROOF_REGISTRY")
	if contractHash == "" {
		return "", fmt.Errorf("CONTRACT_PROOF_REGISTRY env var not set")
	}

	args := &types.Args{}
	args.AddArgument("proof_id", *clvalue.NewCLString(pid)).
		AddArgument("reason", *clvalue.NewCLString(reason))

	return s.putTransaction(contractHash, "revoke_proof", args)
}

// SubmitModelRegistration registers a model on the model_registry contract.
func (s *CasperSubmitter) SubmitModelRegistration(modelID, modelHash, verifierContract string, metadata map[string]string) (string, error) {
	contractHash := os.Getenv("CONTRACT_MODEL_REGISTRY")
	if contractHash == "" {
		return "", fmt.Errorf("CONTRACT_MODEL_REGISTRY env var not set")
	}

	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("marshal metadata: %w", err)
	}

	args := &types.Args{}
	args.AddArgument("model_id", *clvalue.NewCLString(modelID)).
		AddArgument("model_hash", *clvalue.NewCLString(modelHash)).
		AddArgument("verifier_contract", *clvalue.NewCLString(verifierContract)).
		AddArgument("metadata", *clvalue.NewCLString(string(metaJSON)))

	return s.putTransaction(contractHash, "register_model", args)
}

// RegisterZkVk pins a verifying-key digest for a circuit_id on the on-chain
// zk-verifier contract (BACKLOG 1.8 + 2.6 anchor). governanceApproved=0 for
// direct-owner calls, 1 for calls proved off-chain against governance's
// is_executed(proposal_id)==1.
func (s *CasperSubmitter) RegisterZkVk(circuitID, vkHash, curve, backend string, governanceApproved uint64) (string, error) {
	contractHash := os.Getenv("CONTRACT_ZK_VERIFIER")
	if contractHash == "" {
		return "", fmt.Errorf("CONTRACT_ZK_VERIFIER env var not set")
	}
	args := &types.Args{}
	args.AddArgument("governance_approved", *clvalue.NewCLUInt64(governanceApproved)).
		AddArgument("circuit_id", *clvalue.NewCLString(circuitID)).
		AddArgument("vk_hash", *clvalue.NewCLString(vkHash)).
		AddArgument("curve", *clvalue.NewCLString(curve)).
		AddArgument("backend", *clvalue.NewCLString(backend))
	return s.putTransaction(contractHash, "register_vk", args)
}

// RecordZkVerdict anchors an off-chain Groth16 verification verdict on-chain
// against a specific (circuit_id, proof_hash). Public inputs are hashed off-chain
// (sha256 of their canonical encoding) so the on-chain record stays fixed-size.
func (s *CasperSubmitter) RecordZkVerdict(circuitID, proofHash, publicInputsHash, modelID string, valid bool) (string, error) {
	contractHash := os.Getenv("CONTRACT_ZK_VERIFIER")
	if contractHash == "" {
		return "", fmt.Errorf("CONTRACT_ZK_VERIFIER env var not set")
	}
	var verdict uint64
	if valid {
		verdict = 1
	}
	args := &types.Args{}
	args.AddArgument("circuit_id", *clvalue.NewCLString(circuitID)).
		AddArgument("proof_hash", *clvalue.NewCLString(proofHash)).
		AddArgument("public_inputs_hash", *clvalue.NewCLString(publicInputsHash)).
		AddArgument("model_id", *clvalue.NewCLString(modelID)).
		AddArgument("verdict", *clvalue.NewCLUInt64(verdict))
	return s.putTransaction(contractHash, "record_verdict", args)
}

// AddZkVerifier authorizes an account hash to record verdicts.
func (s *CasperSubmitter) AddZkVerifier(verifierAccountHash string) (string, error) {
	contractHash := os.Getenv("CONTRACT_ZK_VERIFIER")
	if contractHash == "" {
		return "", fmt.Errorf("CONTRACT_ZK_VERIFIER env var not set")
	}
	acc, err := key.NewAccountHash(verifierAccountHash)
	if err != nil {
		return "", fmt.Errorf("parse verifier account hash: %w", err)
	}
	args := &types.Args{}
	args.AddArgument("verifier", clvalue.NewCLByteArray(acc.Bytes()))
	return s.putTransaction(contractHash, "add_verifier", args)
}
