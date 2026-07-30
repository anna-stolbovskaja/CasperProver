package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func buildMLAttestMux(t *testing.T) *http.ServeMux {
	t.Helper()
	s := &Server{}
	mux := http.NewServeMux()
	s.registerMLAttestRoutes(mux)
	return mux
}

func mkHex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func mlInputBody(t *testing.T, modelID string) map[string]any {
	t.Helper()
	return map[string]any{
		"input": map[string]any{
			"model_id":           modelID,
			"weights_digest_hex": mkHex("weights-" + modelID),
			"inputs_digest_hex":  mkHex("x-tensor-" + modelID),
			"outputs_digest_hex": mkHex("y-tensor-" + modelID),
		},
	}
}

func TestMLAttest_AttestAndVerify_RoundTrip(t *testing.T) {
	mux := buildMLAttestMux(t)
	body := mlInputBody(t, "mnist-mlp-v0")

	resp := keyringDoJSON(t, mux, "POST", "/v1/ml/attest", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("attest: %d body=%s", resp.Code, resp.Body.String())
	}
	var att struct {
		Scheme        string `json:"scheme"`
		ModelID       string `json:"model_id"`
		WeightsDigest string `json:"weights_digest_hex"`
		InputsDigest  string `json:"inputs_digest_hex"`
		OutputsDigest string `json:"outputs_digest_hex"`
		Commit        string `json:"commit_hex"`
		Disclosure    string `json:"disclosure"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &att); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if att.Scheme != "ml-attest-v0" {
		t.Fatalf("wrong scheme label: %q", att.Scheme)
	}
	if len(att.Commit) != 64 {
		t.Fatalf("commit must be 64-hex, got %d chars", len(att.Commit))
	}
	if !strings.Contains(att.Disclosure, "NOT a cryptographic proof") {
		t.Fatalf("disclosure missing honesty clause: %q", att.Disclosure)
	}

	// Verify.
	verifyBody := map[string]any{
		"input":       body["input"],
		"attestation": att,
	}
	resp = keyringDoJSON(t, mux, "POST", "/v1/ml/verify-attest", verifyBody)
	if resp.Code != http.StatusOK {
		t.Fatalf("verify: %d body=%s", resp.Code, resp.Body.String())
	}
	var vr struct {
		Valid  bool   `json:"valid"`
		Scheme string `json:"scheme"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &vr)
	if !vr.Valid || vr.Scheme != "ml-attest-v0" {
		t.Fatalf("verify must return valid:true scheme:ml-attest-v0, got %s", resp.Body.String())
	}
}

func TestMLAttest_RejectsReservedZKMLScheme(t *testing.T) {
	mux := buildMLAttestMux(t)
	body := mlInputBody(t, "m1")
	body["scheme"] = "zkml-fixed-v0"

	resp := keyringDoJSON(t, mux, "POST", "/v1/ml/attest", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for reserved scheme, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "unsupported scheme") {
		t.Fatalf("expected 'unsupported scheme' hint, got %s", resp.Body.String())
	}
}

func TestMLAttest_VerifyRejectsTamperedCommit(t *testing.T) {
	mux := buildMLAttestMux(t)
	body := mlInputBody(t, "m1")
	resp := keyringDoJSON(t, mux, "POST", "/v1/ml/attest", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("attest: %d %s", resp.Code, resp.Body.String())
	}
	var att map[string]any
	_ = json.Unmarshal(resp.Body.Bytes(), &att)

	// Flip one nibble in the commit.
	commit := att["commit_hex"].(string)
	buf := []byte(commit)
	if buf[0] == '0' {
		buf[0] = '1'
	} else {
		buf[0] = '0'
	}
	att["commit_hex"] = string(buf)

	verifyBody := map[string]any{
		"input":       body["input"],
		"attestation": att,
	}
	resp = keyringDoJSON(t, mux, "POST", "/v1/ml/verify-attest", verifyBody)
	if resp.Code != http.StatusOK {
		t.Fatalf("verify http: %d %s", resp.Code, resp.Body.String())
	}
	var vr struct {
		Valid bool   `json:"valid"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &vr)
	if vr.Valid {
		t.Fatalf("tampered commit must yield valid:false, got %s", resp.Body.String())
	}
	if vr.Error == "" {
		t.Fatalf("expected error string on tamper, got %s", resp.Body.String())
	}
}

func TestMLAttest_VerifyRejectsRelabelledAsZKML(t *testing.T) {
	// This is the key laundering-guard: even a well-formed
	// ml-attest-v0 commit MUST NOT verify if the envelope claims
	// to be a real ZK-ML circuit proof (zkml-fixed-v0).
	mux := buildMLAttestMux(t)
	body := mlInputBody(t, "m1")
	resp := keyringDoJSON(t, mux, "POST", "/v1/ml/attest", body)
	if resp.Code != http.StatusOK {
		t.Fatalf("attest: %d", resp.Code)
	}
	var att map[string]any
	_ = json.Unmarshal(resp.Body.Bytes(), &att)
	att["scheme"] = "zkml-fixed-v0"

	verifyBody := map[string]any{
		"input":       body["input"],
		"attestation": att,
	}
	resp = keyringDoJSON(t, mux, "POST", "/v1/ml/verify-attest", verifyBody)
	if resp.Code != http.StatusOK {
		t.Fatalf("verify http: %d", resp.Code)
	}
	var vr struct {
		Valid bool `json:"valid"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &vr)
	if vr.Valid {
		t.Fatalf("relabelled attestation must NOT verify — this is the laundering guard")
	}
}

func TestMLAttest_RejectsBadHex(t *testing.T) {
	mux := buildMLAttestMux(t)
	body := map[string]any{
		"input": map[string]any{
			"model_id":           "m1",
			"weights_digest_hex": "not-hex!!",
			"inputs_digest_hex":  mkHex("x"),
			"outputs_digest_hex": mkHex("y"),
		},
	}
	resp := keyringDoJSON(t, mux, "POST", "/v1/ml/attest", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad hex, got %d body=%s", resp.Code, resp.Body.String())
	}
}
