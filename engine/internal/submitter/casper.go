package submitter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
)

type CasperSubmitter struct {
	nodeURL  string
	chain    string
	keyPath  string
	client   *http.Client
}

func New(nodeURL, chain, keyPath string) *CasperSubmitter {
	return &CasperSubmitter{
		nodeURL: nodeURL,
		chain:   chain,
		keyPath: keyPath,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
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

	resp, err := s.client.Post(s.nodeURL+"/rpc", "application/json", bytes.NewReader(body))
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

	resp, err := s.client.Post(s.nodeURL+"/rpc", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Result struct {
			DeployHash string `json:"deploy_hash"`
		} `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Result.DeployHash, nil
}
