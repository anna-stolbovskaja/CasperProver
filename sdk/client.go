// Package sdk provides a Go client for the CasperProver REST API.
//
// Usage:
//
//	c := sdk.NewClient("http://localhost:9090")
//	proof, _ := c.Submit("agent-1", []byte("in"), []byte("out"), []byte("m"), "inference")
//	status, _ := c.Get(proof.ID)
//	ok, _ := c.Verify(proof.ID)
package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Proof mirrors the server's JSON representation.
type Proof struct {
	ID      string   `json:"id"`
	AgentID string   `json:"agent_id"`
	PH      string   `json:"proof_hash"`
	IH      string   `json:"input_hash"`
	OH      string   `json:"output_hash"`
	MH      string   `json:"model_hash"`
	Root    string   `json:"root"`
	Path    []string `json:"path,omitempty"`
	Idx     int      `json:"idx"`
	UseCase string   `json:"use_case"`
	Valid   bool     `json:"valid"`
	Revoked bool     `json:"revoked"`
	TS      int64    `json:"timestamp"`
}

// SubmitRequest is the body for POST /proofs.
type SubmitRequest struct {
	AgentID string `json:"agent_id"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Model   string `json:"model"`
	UseCase string `json:"use_case"`
}

// Client talks to a CasperProver API server.
type Client struct {
	base   string
	http   *http.Client
}

// NewClient creates a client pointed at the given base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		base: baseURL,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// Health returns true if the server is reachable.
func (c *Client) Health() (bool, error) {
	resp, err := c.http.Get(c.base + "/health")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, nil
}

// Submit generates a new proof on the server.
func (c *Client) Submit(agentID string, input, output, model []byte, useCase string) (*Proof, error) {
	body := SubmitRequest{
		AgentID: agentID,
		Input:   string(input),
		Output:  string(output),
		Model:   string(model),
		UseCase: useCase,
	}
	return c.postProof("/proofs", body)
}

// Get fetches a proof by ID.
func (c *Client) Get(proofID string) (*Proof, error) {
	resp, err := c.http.Get(fmt.Sprintf("%s/proofs/%s", c.base, proofID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, c.readError(resp)
	}
	var p Proof
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

// List returns all proofs.
func (c *Client) List() ([]Proof, error) {
	resp, err := c.http.Get(c.base + "/proofs")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var proofs []Proof
	if err := json.NewDecoder(resp.Body).Decode(&proofs); err != nil {
		return nil, err
	}
	return proofs, nil
}

// Verify checks a proof's validity server-side.
func (c *Client) Verify(proofID string) (bool, error) {
	p, err := c.Get(proofID)
	if err != nil {
		return false, err
	}
	return p.Valid && !p.Revoked, nil
}

func (c *Client) postProof(path string, body interface{}) (*Proof, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Post(c.base+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, c.readError(resp)
	}
	var p Proof
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) readError(resp *http.Response) error {
	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("api error %d: %s", resp.StatusCode, string(b))
}
