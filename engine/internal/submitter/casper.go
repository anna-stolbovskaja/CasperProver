// Package submitter builds and signs Casper 2.0 (Condor) TransactionV1
// payloads using the official casper-go-sdk/v2 and submits them to a
// Casper node via JSON-RPC.
package submitter

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
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

	// Retry defaults for transient RPC failures (network timeouts, 5xx,
	// connection resets). Values chosen for interactive-latency submits:
	// at most 3 attempts with 200ms/400ms backoff between attempts, so
	// worst case adds ~600ms before surfacing a terminal error.
	defaultMaxAttempts   = 3
	defaultInitialBackoff = 200 * time.Millisecond
	defaultBackoffFactor = 2.0
)

// txPutter is the subset of the casper-go-sdk rpc.Client surface the
// submitter depends on. Extracting an interface lets unit tests inject
// a mock RPC that returns programmed errors / successes without hitting
// a real Casper node.
type txPutter interface {
	PutTransactionV1(ctx context.Context, tx types.TransactionV1) (rpc.PutTransactionResult, error)
}

// retryConfig controls how putTransaction retries a transient failure.
type retryConfig struct {
	maxAttempts    int
	initialBackoff time.Duration
	backoffFactor  float64
}

// CasperSubmitter signs and submits TransactionV1 calls to on-chain
// CasperProver contracts via the casper-go-sdk RPC client.
type CasperSubmitter struct {
	chain  string
	keys   keypair.PrivateKey
	client txPutter
	retry  retryConfig
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
		retry: retryConfig{
			maxAttempts:    defaultMaxAttempts,
			initialBackoff: defaultInitialBackoff,
			backoffFactor:  defaultBackoffFactor,
		},
	}
}

// isRetryableRPCError returns true for errors that indicate a transient
// failure of the deploy-submission RPC call and are worth retrying:
//   - context.DeadlineExceeded / net.Error with Timeout==true
//   - connection resets / EOF / broken pipe / "connection refused"
//   - HTTP 5xx responses (surfaced by the SDK as textual errors)
//
// Terminal errors (bad payload, signing failure, 4xx from the node,
// invalid arguments) are NOT retried — retrying them would just waste
// a slot and delay the real error reaching the caller.
func isRetryableRPCError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	transientMarkers := []string{
		"connection reset",
		"connection refused",
		"broken pipe",
		"eof",
		"i/o timeout",
		"timeout",
		"temporarily unavailable",
		"service unavailable", // 503
		"bad gateway",         // 502
		"gateway timeout",     // 504
		"internal server error", // 500
		"status 5", // catch-all for "HTTP status 5xx" style formatting
	}
	for _, m := range transientMarkers {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
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

	res, err := s.putWithRetry(context.Background(), *tx, entryPoint)
	if err != nil {
		return "", fmt.Errorf("put transaction: %w", err)
	}

	txHash := res.TransactionHash.TransactionV1.ToHex()
	slog.Info("transaction submitted", "hash", txHash, "entry_point", entryPoint)
	return txHash, nil
}

// putWithRetry calls PutTransactionV1 with exponential-backoff retries
// on transient RPC errors. Returns the RPC result of the first
// successful call, or the last error after maxAttempts.
func (s *CasperSubmitter) putWithRetry(ctx context.Context, tx types.TransactionV1, entryPoint string) (rpc.PutTransactionResult, error) {
	maxAttempts := s.retry.maxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}
	backoff := s.retry.initialBackoff
	if backoff <= 0 {
		backoff = defaultInitialBackoff
	}
	factor := s.retry.backoffFactor
	if factor <= 0 {
		factor = defaultBackoffFactor
	}

	var lastErr error
	var zero rpc.PutTransactionResult
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		res, err := s.client.PutTransactionV1(ctx, tx)
		if err == nil {
			if attempt > 1 {
				slog.Info("deploy submission succeeded after retry",
					"attempt", attempt,
					"entry_point", entryPoint)
			}
			return res, nil
		}
		lastErr = err
		if !isRetryableRPCError(err) {
			slog.Warn("deploy submission failed with non-retryable error",
				"attempt", attempt,
				"entry_point", entryPoint,
				"err", err)
			return zero, err
		}
		if attempt == maxAttempts {
			break
		}
		slog.Warn("deploy submission transient failure, retrying",
			"attempt", attempt,
			"next_backoff", backoff,
			"entry_point", entryPoint,
			"err", err)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return zero, ctx.Err()
		}
		backoff = time.Duration(float64(backoff) * factor)
	}
	return zero, fmt.Errorf("exhausted %d attempts: %w", maxAttempts, lastErr)
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
