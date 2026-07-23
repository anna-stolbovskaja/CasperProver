package verifier

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/anna-stolbovskaja/CasperProver/engine/internal/prover"
)

func randBytes(n int, seed int64) []byte {
	r := rand.New(rand.NewSource(seed))
	buf := make([]byte, n)
	_, _ = r.Read(buf)
	return buf
}

// payloadSizes covers the same complexity-class boundaries as
// prover.ClassifierConfig defaults (trivial/low/medium) so the benchmark
// tracks verification cost across realistic proof sizes.
var payloadSizes = []int{64, 1024, 65536}

// BenchmarkVerifyProof measures LocalVerifier.VerifyProof end-to-end
// (generate a proof, then verify it) across payload sizes.
func BenchmarkVerifyProof(b *testing.B) {
	v := New()
	for _, sz := range payloadSizes {
		input := randBytes(sz, int64(sz))
		output := randBytes(sz, int64(sz)+1)
		model := randBytes(sz, int64(sz)+2)

		engine := prover.New()
		p := engine.Generate("bench-agent", input, output, model, "bench-uc")

		b.Run(fmt.Sprintf("payload=%dB", sz), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := v.VerifyProof(p, input, output, model); err != nil {
					b.Fatalf("unexpected verify error: %v", err)
				}
			}
		})
	}
}

// BenchmarkGenerateProof measures ProofEngine.Generate cost across payload
// sizes — the write-path counterpart to BenchmarkVerifyProof.
func BenchmarkGenerateProof(b *testing.B) {
	for _, sz := range payloadSizes {
		input := randBytes(sz, int64(sz)+10)
		output := randBytes(sz, int64(sz)+11)
		model := randBytes(sz, int64(sz)+12)

		b.Run(fmt.Sprintf("payload=%dB", sz), func(b *testing.B) {
			engine := prover.New()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = engine.Generate("bench-agent", input, output, model, "bench-uc")
			}
		})
	}
}
