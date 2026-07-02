// Package sdk provides an MCP server for CasperProver.
//
// The server exposes proof generation, verification, and KYC flow
// as tools that any MCP-compatible LLM can invoke.
//
// Start:
//
//	go run sdk/mcp_server.go
//
// or via the main binary:
//
//	casperprover serve --mcp
package sdk

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// MCPTool describes an MCP-compatible tool.
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPManifest is returned by tools/list.
type MCPManifest struct {
	Name    string    `json:"name"`
	Version string    `json:"version"`
	Tools   []MCPTool `json:"tools"`
}

var mcpTools = []MCPTool{
	{
		Name:        "generate_proof",
		Description: "Generate a cryptographic proof of AI inference (input → output).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"agent_id": map[string]string{"type": "string", "description": "Agent identifier"},
				"input":    map[string]string{"type": "string", "description": "Inference input data"},
				"output":   map[string]string{"type": "string", "description": "Inference output data"},
				"model":    map[string]string{"type": "string", "description": "Model identifier"},
				"use_case": map[string]string{"type": "string", "description": "Use case tag (inference, kyc, etc.)"},
			},
			"required": []string{"agent_id", "input", "output", "model"},
		},
	},
	{
		Name:        "verify_proof",
		Description: "Verify that a proof is valid and not revoked.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"proof_id": map[string]string{"type": "string", "description": "Proof identifier (e.g. P-1)"},
			},
			"required": []string{"proof_id"},
		},
	},
	{
		Name:        "get_proof",
		Description: "Fetch full details of a proof by ID.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"proof_id": map[string]string{"type": "string"},
			},
			"required": []string{"proof_id"},
		},
	},
	{
		Name:        "list_proofs",
		Description: "List all generated proofs.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "revoke_proof",
		Description: "Revoke a proof, marking it as invalid.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"proof_id": map[string]string{"type": "string"},
				"reason":   map[string]string{"type": "string", "description": "Reason for revocation"},
			},
			"required": []string{"proof_id", "reason"},
		},
	},
	{
		Name:        "batch_proofs",
		Description: "Generate multiple proofs in a single batch request.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"proofs": map[string]interface{}{
					"type":        "array",
					"description": "Array of proof requests",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"agent_id": map[string]string{"type": "string"},
							"input":    map[string]string{"type": "string"},
							"output":   map[string]string{"type": "string"},
							"model":    map[string]string{"type": "string"},
						},
					},
				},
			},
			"required": []string{"proofs"},
		},
	},
	{
		Name:        "export_proof",
		Description: "Export a proof as a portable JSON bundle for external verification.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"proof_id": map[string]string{"type": "string"},
			},
			"required": []string{"proof_id"},
		},
	},
	{
		Name:        "get_stats",
		Description: "Get aggregate proof statistics: total proofs, verified count, models.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "kyc_check",
		Description: "Check whether an address passes KYC whitelist verification.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"address": map[string]string{"type": "string", "description": "Address to verify"},
			},
			"required": []string{"address"},
		},
	},
	{
		Name:        "kyc_grant",
		Description: "Grant KYC whitelist access to an address (admin only).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"address": map[string]string{"type": "string"},
				"admin":   map[string]string{"type": "string", "description": "Admin account hash"},
			},
			"required": []string{"address", "admin"},
		},
	},
	{
		Name:        "kyc_whitelist",
		Description: "Check KYC whitelist status for a specific user.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user": map[string]string{"type": "string"},
			},
			"required": []string{"user"},
		},
	},
	{
		Name:        "health_check",
		Description: "Check API server, database, and blockchain connection health.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "get_model_info",
		Description: "Get information about a registered model by its hash.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"model_hash": map[string]string{"type": "string"},
			},
			"required": []string{"model_hash"},
		},
	},
	{
		Name:        "list_models",
		Description: "List all registered models used in proof generation.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "get_merkle_root",
		Description: "Get the current Merkle root of all proofs for on-chain verification.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	// --- Model Registry tools ---
	{
		Name:        "register_model",
		Description: "Register a new AI model in the on-chain registry with its metadata and hash.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"model_name":    map[string]interface{}{"type": "string", "description": "Human-readable model name"},
				"model_hash":    map[string]interface{}{"type": "string", "description": "SHA-256 of model weights"},
				"framework":     map[string]interface{}{"type": "string", "description": "ML framework (pytorch, tensorflow, onnx)"},
				"version":       map[string]interface{}{"type": "string", "description": "Semantic version"},
				"max_input_size": map[string]interface{}{"type": "integer", "description": "Max input size in bytes"},
			},
			"required": []string{"model_name", "model_hash", "framework"},
		},
	},
	{
		Name:        "get_model_registry",
		Description: "Query a model from the registry by name or hash, including all versions.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"model_id": map[string]interface{}{"type": "string", "description": "Model name or hash"},
			},
			"required": []string{"model_id"},
		},
	},
	{
		Name:        "deprecate_model",
		Description: "Mark a model version as deprecated in the registry.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"model_id": map[string]interface{}{"type": "string"},
				"version":  map[string]interface{}{"type": "string"},
				"reason":   map[string]interface{}{"type": "string", "description": "Deprecation reason"},
			},
			"required": []string{"model_id", "version"},
		},
	},
	// --- Complexity Analyzer tools ---
	{
		Name:        "estimate_complexity",
		Description: "Estimate proof generation complexity for given model and input parameters.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"model_hash":  map[string]interface{}{"type": "string", "description": "Model identifier"},
				"input_size":  map[string]interface{}{"type": "integer", "description": "Input size in bytes"},
				"proof_type":  map[string]interface{}{"type": "string", "enum": []string{"full", "compact", "aggregated"}},
			},
			"required": []string{"model_hash", "input_size"},
		},
	},
	{
		Name:        "get_complexity_report",
		Description: "Get a detailed complexity breakdown for a completed proof.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"proof_id": map[string]interface{}{"type": "string"},
			},
			"required": []string{"proof_id"},
		},
	},
	// --- Distributed Worker tools ---
	{
		Name:        "submit_batch_task",
		Description: "Submit a batch of proofs for distributed generation across worker nodes.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"model_hash": map[string]interface{}{"type": "string"},
				"inputs":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Array of input hashes"},
				"priority":   map[string]interface{}{"type": "string", "enum": []string{"low", "normal", "high"}, "default": "normal"},
			},
			"required": []string{"model_hash", "inputs"},
		},
	},
	{
		Name:        "get_task_status",
		Description: "Check the status and progress of a distributed batch task.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id": map[string]interface{}{"type": "string"},
			},
			"required": []string{"task_id"},
		},
	},
	{
		Name:        "list_worker_nodes",
		Description: "List all active worker nodes with their capacity and current load.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
}

// Manifest returns the MCP tool manifest.
func Manifest() MCPManifest {
	return MCPManifest{
		Name:    "casperprover",
		Version: "0.3.0",
		Tools:   mcpTools,
	}
}

type jsonrpcRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result"`
}

// RunStdio starts a JSON-RPC stdio loop for MCP.
// Tool calls are forwarded to the given handler function.
func RunStdio(handler func(name string, args map[string]interface{}) (string, error)) {
	fmt.Fprintln(os.Stderr, "CasperProver MCP server (stdio)")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		var req jsonrpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
			continue
		}

		var resp jsonrpcResponse
		resp.JSONRPC = "2.0"
		resp.ID = req.ID

		switch req.Method {
		case "tools/list":
			resp.Result = map[string]interface{}{"tools": mcpTools}
		case "tools/call":
			name, _ := req.Params["name"].(string)
			args, _ := req.Params["arguments"].(map[string]interface{})
			text, err := handler(name, args)
			if err != nil {
				text = fmt.Sprintf(`{"error":"%s"}`, err.Error())
			}
			resp.Result = map[string]interface{}{
				"content": []map[string]string{{"type": "text", "text": text}},
			}
		default:
			resp.Result = map[string]string{"error": "unknown method"}
		}

		out, _ := json.Marshal(resp)
		fmt.Println(string(out))
	}
}
