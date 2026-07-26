// Quickstart example for the CasperProver Go SDK.
//
// Usage: CP_API_URL=http://localhost:9090 CP_API_KEY=... go run ./sdk/examples/go/quickstart
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/anna-stolbovskaja/CasperProver/sdk"
)

func main() {
	baseURL := getenv("CP_API_URL", "http://localhost:9090")
	apiKey := os.Getenv("CP_API_KEY")

	c := sdk.NewClient(
		sdk.WithBaseURL(baseURL),
		sdk.WithAuthToken(apiKey),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Health.
	h, err := c.Health(ctx)
	if err != nil {
		log.Fatalf("/health failed: %v", err)
	}
	fmt.Printf("health: %+v\n", h)

	// 2. Prove.
	proof, err := c.Prove(ctx, sdk.ProveRequest{
		Agent:   "example-agent",
		Model:   "gpt-toy-v1",
		Input:   "hello world",
		Output:  "42",
		UseCase: "quickstart",
	}, sdk.WithIdempotencyKey("quickstart-1"))
	if err != nil {
		log.Fatalf("prove failed: %v", err)
	}
	fmt.Printf("proof: id=%s vk_hash=%s\n", proof.ID, proof.VKHash)

	// 3. Verify.
	v, err := c.Verify(ctx, proof.ID)
	if err != nil {
		log.Fatalf("verify failed: %v", err)
	}
	fmt.Printf("verify: valid=%v\n", v.Valid)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
