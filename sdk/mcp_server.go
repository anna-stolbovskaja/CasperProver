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
