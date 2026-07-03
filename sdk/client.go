package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "http://localhost:8080/api/v1"
	defaultTimeout = 30 * time.Second
)

// CasperProverClient is a client for interacting with the CasperProver API.
type CasperProverClient struct {
	authToken  string
	baseURL    string
	httpClient *http.Client
}

// NewCasperProverClient creates a new CasperProverClient with default or custom options.
func NewCasperProverClient(opts ...ClientOption) *CasperProverClient {
	client := &CasperProverClient{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// ClientOption defines a functional option for CasperProverClient.
type ClientOption func(*CasperProverClient)

// WithBaseURL sets the base URL for the CasperProver API.
func WithBaseURL(url string) ClientOption {
	return func(c *CasperProverClient) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *CasperProverClient) {
		c.httpClient = httpClient
	}
}

// WithAuthToken sets the authentication token for API requests.
func WithAuthToken(token string) ClientOption {
	return func(c *CasperProverClient) {
		c.authToken = token
	}
}

// SetAuthToken updates the authentication token at runtime.
func (c *CasperProverClient) SetAuthToken(token string) {
	c.authToken = token
}

// doRequest performs an HTTP request and decodes the JSON response.
func (c *CasperProverClient) doRequest(ctx context.Context, method, path string, reqBody interface{}, resp interface{}) error {
	var bodyReader io.Reader
	if reqBody != nil {
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	respHTTP, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer respHTTP.Body.Close()

	if respHTTP.StatusCode < 200 || respHTTP.StatusCode >= 300 {
		errorBody, _ := io.ReadAll(respHTTP.Body)
		return fmt.Errorf("API request failed with status %d: %s", respHTTP.StatusCode, string(errorBody))
	}

	if resp != nil {
		if err := json.NewDecoder(respHTTP.Body).Decode(resp); err != nil {
			return fmt.Errorf("failed to decode response body: %w", err)
		}
	}

	return nil
}

// SubmitProof submits a new ZK proof to the prover.
func (c *CasperProverClient) SubmitProof(ctx context.Context, proof *Proof) (*VerificationResult, error) {
	var result VerificationResult
	err := c.doRequest(ctx, http.MethodPost, "/proofs", proof, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to submit proof: %w", err)
	}
	return &result, nil
}

// VerifyProof retrieves the verification status of a specific proof.
func (c *CasperProverClient) VerifyProof(ctx context.Context, proofID string) (*VerificationResult, error) {
	var result VerificationResult
	err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/proofs/%s/verify", proofID), nil, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to verify proof %s: %w", proofID, err)
	}
	return &result, nil
}

// BatchProofs submits a batch of proof IDs for verification.
func (c *CasperProverClient) BatchProofs(ctx context.Context, proofIDs []string) (*BatchResult, error) {
	var result BatchResult
	reqBody := struct {
		ProofIDs []string `json:"proof_ids"`
	}{ProofIDs: proofIDs}
	err := c.doRequest(ctx, http.MethodPost, "/proofs/batch", reqBody, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to batch proofs: %w", err)
	}
	return &result, nil
}

// RegisterModel registers a new AI model with the prover.
func (c *CasperProverClient) RegisterModel(ctx context.Context, model *Model) (*Model, error) {
	var result Model
	err := c.doRequest(ctx, http.MethodPost, "/models", model, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to register model: %w", err)
	}
	return &result, nil
}

// AggregateSTARK requests the aggregation of STARK proofs.
func (c *CasperProverClient) AggregateSTARK(ctx context.Context, challenge *Challenge) (*Proof, error) {
	var result Proof
	err := c.doRequest(ctx, http.MethodPost, "/proofs/stark/aggregate", challenge, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate STARK: %w", err)
	}
	return &result, nil
}

// VerifyGroth16 verifies a specific Groth16 proof.
func (c *CasperProverClient) VerifyGroth16(ctx context.Context, proofID string) (*VerificationResult, error) {
	var result VerificationResult
	err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/proofs/groth16/%s/verify", proofID), nil, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to verify Groth16 proof %s: %w", proofID, err)
	}
	return &result, nil
}

// SignSPHINCS requests the prover to sign a message using SPHINCS+.
func (c *CasperProverClient) SignSPHINCS(ctx context.Context, message []byte) ([]byte, error) {
	reqBody := struct {
		Message string `json:"message"`
	}{Message: string(message)} // Assuming message is base64 encoded or similar string
	var respBody struct {
		Signature string `json:"signature"`
	}
	err := c.doRequest(ctx, http.MethodPost, "/crypto/sphincs/sign", reqBody, &respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with SPHINCS+: %w", err)
	}
	return []byte(respBody.Signature), nil // Assuming signature is returned as a string
}

// HybridSign requests the prover to sign a message using a hybrid signature scheme.
func (c *CasperProverClient) HybridSign(ctx context.Context, message []byte) ([]byte, error) {
	reqBody := struct {
		Message string `json:"message"`
	}{Message: string(message)} // Assuming message is base64 encoded or similar string
	var respBody struct {
		Signature string `json:"signature"`
	}
	err := c.doRequest(ctx, http.MethodPost, "/crypto/hybrid/sign", reqBody, &respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with hybrid scheme: %w", err)
	}
	return []byte(respBody.Signature), nil // Assuming signature is returned as a string
}

// GetPQCryptoStatus retrieves the current post-quantum cryptography status.
func (c *CasperProverClient) GetPQCryptoStatus(ctx context.Context) (*PQCryptoStatus, error) {
	var status PQCryptoStatus
	err := c.doRequest(ctx, http.MethodGet, "/crypto/pq/status", nil, &status)
	if err != nil {
		return nil, fmt.Errorf("failed to get PQ crypto status: %w", err)
	}
	return &status, nil
}
