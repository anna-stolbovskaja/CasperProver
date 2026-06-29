package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/api"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/kyc"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
	"github.com/anna-stolbovskaja/CasperProver/engine/internal/verifier"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	eng := prover.New()

	switch os.Args[1] {
	case "prove":
		demoProve(eng)
	case "verify":
		demoVerify(eng)
	case "demo":
		demoFlow(eng)
	case "serve":
		serve(eng)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: casperprover <command>\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  prove   Generate a demo proof\n")
	fmt.Fprintf(os.Stderr, "  verify  Verify a demo proof\n")
	fmt.Fprintf(os.Stderr, "  demo    Run full KYC demo flow\n")
	fmt.Fprintf(os.Stderr, "  serve   Start API server\n")
}

func demoProve(eng *prover.ProofEngine) {
	input := []byte(`{"user":"bob","doc":"passport"}`)
	output := []byte(`{"verified":true}`)
	model := []byte("model-v1")

	p := eng.Generate("demo-agent", input, output, model, "kyc")
	fmt.Printf("id:   %s\n", p.ID)
	fmt.Printf("hash: %s\n", p.PH)
	fmt.Printf("root: %s\n", p.Root)
}

func demoVerify(eng *prover.ProofEngine) {
	input := []byte(`{"user":"bob","doc":"passport"}`)
	output := []byte(`{"verified":true}`)
	model := []byte("model-v1")

	p := eng.Generate("demo-agent", input, output, model, "kyc")
	v := verifier.New()
	err := v.VerifyProof(p, input, output, model)
	if err != nil {
		slog.Error("verification failed", "error", err, "proof_id", p.ID)
		os.Exit(1)
	}
	slog.Info("verification passed", "proof_id", p.ID)
	fmt.Printf("OK: proof %s verified\n", p.ID)
}

func demoFlow(eng *prover.ProofEngine) {
	flow := kyc.NewDeFiFlow(eng)
	if err := flow.RunDemo("demo-agent"); err != nil {
		slog.Error("demo flow failed", "error", err)
		os.Exit(1)
	}
}

func serve(eng *prover.ProofEngine) {
	port := 8080
	if v := os.Getenv("API_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}
	srv := api.New(eng, port)
	if err := srv.Start(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
