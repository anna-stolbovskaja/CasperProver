// Command mcpserver runs the CasperProver MCP (Model Context Protocol)
// server over stdio, backed by a real CasperProver API instance.
//
// Usage:
//
//	CASPERPROVER_API_URL=http://localhost:9090 go run ./sdk/cmd/mcpserver
//
// Only tools with a real, implemented API endpoint are executed; the rest
// return a clear "not implemented" error (see docs/KNOWN_LIMITATIONS.md)
// instead of silently returning fabricated data.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/anna-stolbovskaja/CasperProver/sdk"
)

func main() {
	baseURL := os.Getenv("CASPERPROVER_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9090"
	}
	client := sdk.NewClient(sdk.WithBaseURL(baseURL))
	if token := os.Getenv("CASPERPROVER_API_KEY"); token != "" {
		client.SetAuthToken(token)
	}

	sdk.RunStdio(func(name string, args map[string]interface{}) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return dispatch(ctx, client, name, args)
	})
}

func dispatch(ctx context.Context, c *sdk.Client, name string, args map[string]interface{}) (string, error) {
	str := func(k string) string { s, _ := args[k].(string); return s }

	var (
		result interface{}
		err    error
	)

	switch name {
	case "health_check":
		result, err = c.Health(ctx)
	case "generate_proof":
		result, err = c.SubmitProof(ctx, sdk.SubmitProofRequest{
			Agent: str("agent"), Input: str("input"), Output: str("output"),
			Model: str("model"), UseCase: str("use_case"),
		})
	case "verify_proof":
		result, err = c.VerifyProof(ctx, str("proof_id"))
	case "get_proof":
		result, err = c.GetProof(ctx, str("proof_id"))
	case "list_proofs":
		result, err = c.ListProofs(ctx)
	case "revoke_proof":
		err = c.RevokeProof(ctx, str("proof_id"), str("reason"))
		result = map[string]any{"revoked": err == nil}
	case "export_proof":
		result, err = c.ExportProof(ctx, str("proof_id"))
	case "get_stats":
		result, err = c.Stats(ctx)
	case "kyc_check":
		result, err = c.KYCCheck(ctx, str("proof_id"))
	case "kyc_grant":
		result, err = c.KYCGrant(ctx, str("user"), str("proof_id"))
	case "kyc_whitelist":
		result, err = c.KYCWhitelist(ctx, str("user"))
	case "get_model_info":
		result, err = c.GetModel(ctx, str("model_id"))
	case "register_model":
		result, err = c.RegisterModel(ctx, sdk.RegisterModelRequest{
			ModelID: str("model_id"), ModelHash: str("model_hash"),
			VerifierContract: str("verifier_contract"),
		})
	case "batch_proofs":
		// batch_proofs takes a list of proof specs (not flat args); not mapped over MCP yet.
		return "", fmt.Errorf("tool %q not implemented over MCP yet - use the REST API's POST /proofs/batch directly", name)
	default:
		return "", fmt.Errorf("tool %q has no backing API endpoint yet - see docs/KNOWN_LIMITATIONS.md", name)
	}

	if err != nil {
		return "", err
	}
	out, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return "", marshalErr
	}
	return string(out), nil
}
