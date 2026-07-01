package prover

import (
	"math"
)

type ComplexityClass int

const (
	TRIVIAL ComplexityClass = iota
	LOW
	MEDIUM
	HIGH
	EXTREME
)

type ComplexityMetrics struct {
	InputSize  int
	OutputSize int
	ModelSize  int
	LeafCount  int
	TreeDepth  int
	MerkleOps  int
	EstGasCSPR int64
	EstTimeMs  int64
	Class      ComplexityClass
}

type ClassifierConfig struct {
	ThresholdTrivial int
	ThresholdLow     int
	ThresholdMedium  int
	ThresholdHigh    int
}

const (
	baseGas      int64 = 21000
	perByteGas   int64 = 68
	perMerkleOp  int64 = 2000
	baseTimeMs   int64 = 50
	perKBMs      int64 = 10
)

var (
	defaultConfig = ClassifierConfig{
		ThresholdTrivial: 1024,
		ThresholdLow:     65536,
		ThresholdMedium:  1048576,
		ThresholdHigh:    16777216,
	}
)

func DefaultConfig() ClassifierConfig {
	return defaultConfig
}

func Classify(inputSize, outputSize, modelSize int) *ComplexityMetrics {
	total := inputSize + outputSize + modelSize
	leafCount := 3
	treeDepth := int(math.Ceil(math.Log2(float64(leafCount))))
	merkleOps := leafCount*2 - 1
	gasEstimate := baseGas + perByteGas*int64(total) + perMerkleOp*int64(merkleOps)
	timeEstimate := baseTimeMs + perKBMs*int64(total/1024)

	var class ComplexityClass
	switch {
	case total <= defaultConfig.ThresholdTrivial:
		class = TRIVIAL
	case total <= defaultConfig.ThresholdLow:
		class = LOW
	case total <= defaultConfig.ThresholdMedium:
		class = MEDIUM
	case total <= defaultConfig.ThresholdHigh:
		class = HIGH
	default:
		class = EXTREME
	}

	return &ComplexityMetrics{
		InputSize:  inputSize,
		OutputSize: outputSize,
		ModelSize:  modelSize,
		LeafCount:  leafCount,
		TreeDepth:  treeDepth,
		MerkleOps:  merkleOps,
		EstGasCSPR: gasEstimate,
		EstTimeMs:  timeEstimate,
		Class:      class,
	}
}

func ClassifyProof(p *Proof) *ComplexityMetrics {
	return Classify(len(p.IH), len(p.OH), len(p.MH))
}

func EstimateGas(m *ComplexityMetrics) int64 {
	if m == nil {
		return 0
	}
	return m.EstGasCSPR
}

func EstimateBatchGas(proofs []*Proof) int64 {
	var total int64
	for _, p := range proofs {
		if p == nil {
			continue
		}
		m := ClassifyProof(p)
		total += m.EstGasCSPR
	}
	return total
}

func SuggestBatchSize(proofs []*Proof, maxGas int64) int {
	if maxGas <= 0 {
		return 0
	}
	var sum int64
	var count int
	for _, p := range proofs {
		if p == nil {
			continue
		}
		m := ClassifyProof(p)
		if sum+m.EstGasCSPR > maxGas {
			break
		}
		sum += m.EstGasCSPR
		count++
	}
	return count
}

func ComplexityDistribution(proofs []*Proof) map[ComplexityClass]int {
	dist := make(map[ComplexityClass]int)
	for _, p := range proofs {
		if p == nil {
			continue
		}
		m := ClassifyProof(p)
		dist[m.Class]++
	}
	return dist
}

func (c ComplexityClass) String() string Supreme {
	switch c {
	case TRIVIAL:
		return "TRIVIAL"
	case LOW:
		return "LOW"
	case MEDIUM:
		return "MEDIUM"
	case HIGH:
		return "HIGH"
	case EXTREME:
		return "EXTREME"
	default:
		return "UNKNOWN"
	}
}
