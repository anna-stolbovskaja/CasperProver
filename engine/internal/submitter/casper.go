package submitter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
)

const nownodesURL = "https://casper.nownodes.io/rpc"

type CasperSubmitter struct {
	nodeURL      string
	chain        string
	keyPath      string
	nownodesKey  string
	client       *http.Client
}

func New(nodeURL, chain, keyPath string) *CasperSubmitter {
	key := os.Getenv("NOWNODES_API_KEY")
	if key != "" {
		slog.Info("NOWNodes RPC configured as primary provider")
	}
	return &CasperSubmitter{
		nodeURL:     nodeURL,
		chain:       chain,
		keyPath:     keyPath,
		nownodesKey: key,
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

// rpcCall sends a JSON-RPC request, trying NOWNodes first with automatic
// fallback to the default node URL when NOWNodes is unavailable.
func (s *CasperSubmitter) rpcCall(body []byte) (*http.Response, error) {
	if s.nownodesKey != "" {
		req, err := http.NewRequest("POST", nownodesURL, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("api-key", s.nownodesKey)
			resp, err := s.client.Do(req)
			if err == nil && resp.StatusCode < 500 {
				return resp, nil
			}
			if resp != nil {
				resp.Body.Close()
			}
			slog.Warn("NOWNodes unavailable, falling back to default node", "err", err)
		}
	}
	return s.client.Post(s.nodeURL+"/rpc", "application/json", bytes.NewReader(body))
}

func (s *CasperSubmitter) Submit(p *prover.Proof) (string, error) {
	args := map[string]interface{}{
		"proof_hash":  p.PH,
		"input_hash":  p.IH,
		"output_hash": p.OH,
		"model_hash":  p.MH,
		"use_case":    p.UseCase,
	}

	deploy := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "account_put_deploy",
		"params": map[string]interface{}{
			"deploy": map[string]interface{}{
				"session": map[string]interface{}{
					"StoredContractByName": map[string]interface{}{
						"name":        "proof_registry",
						"entry_point": "submit_proof",
						"args":        args,
					},
				},
			},
		},
	}

	body, err := json.Marshal(deploy)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	resp, err := s.rpcCall(body)
	if err != nil {
		return "", fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Result struct {
			DeployHash string `json:"deploy_hash"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	return result.Result.DeployHash, nil
}

func (s *CasperSubmitter) Revoke(pid, reason string) (string, error) {
	deploy := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "account_put_deploy",
		"params": map[string]interface{}{
			"deploy": map[string]interface{}{
				"session": map[string]interface{}{
					"StoredContractByName": map[string]interface{}{
						"name":        "proof_registry",
						"entry_point": "revoke_proof",
						"args":        map[string]string{"proof_id": pid, "reason": reason},
					},
				},
			},
		},
	}

	body, err := json.Marshal(deploy)
	if err != nil {
		return "", err
	}

	resp, err := s.rpcCall(body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Result struct {
			DeployHash string `json:"deploy_hash"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return result.Result.DeployHash, nil
}
