//go:build openclaw

package agents

import (
	"context"
	"math/rand"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// AlwaysTrueEvaluator accepts every claim with the given confidence.
type AlwaysTrueEvaluator struct {
	Confidence uint32
}

func (e AlwaysTrueEvaluator) Evaluate(_ context.Context, _ string) (agentsdk.Verdict, uint32, error) {
	return agentsdk.VerdictTrue, e.Confidence, nil
}

// AlwaysFalseEvaluator rejects every claim (adversarial behavior).
type AlwaysFalseEvaluator struct {
	Confidence uint32
}

func (e AlwaysFalseEvaluator) Evaluate(_ context.Context, _ string) (agentsdk.Verdict, uint32, error) {
	return agentsdk.VerdictFalse, e.Confidence, nil
}

// DeterministicEvaluator returns pre-configured verdicts per claim ID.
// Unknown claims get VerdictUncertain.
type DeterministicEvaluator struct {
	Verdicts map[string]VerdictEntry
}

// VerdictEntry holds a pre-configured verdict and confidence for a claim.
type VerdictEntry struct {
	Verdict    agentsdk.Verdict
	Confidence uint32
}

func (e DeterministicEvaluator) Evaluate(_ context.Context, claimID string) (agentsdk.Verdict, uint32, error) {
	if entry, ok := e.Verdicts[claimID]; ok {
		return entry.Verdict, entry.Confidence, nil
	}
	return agentsdk.VerdictUncertain, 500000, nil
}

// RandomEvaluator returns deterministic pseudo-random verdicts from a fixed seed.
// Useful for chaos testing with reproducible results.
type RandomEvaluator struct {
	rng *rand.Rand
}

// NewRandomEvaluator creates a seeded random evaluator.
func NewRandomEvaluator(seed int64) *RandomEvaluator {
	return &RandomEvaluator{rng: rand.New(rand.NewSource(seed))}
}

func (e *RandomEvaluator) Evaluate(_ context.Context, _ string) (agentsdk.Verdict, uint32, error) {
	r := e.rng.Float64()
	confidence := uint32(500000 + e.rng.Intn(500000)) // 50-100%
	if r < 0.6 {
		return agentsdk.VerdictTrue, confidence, nil
	} else if r < 0.9 {
		return agentsdk.VerdictFalse, confidence, nil
	}
	return agentsdk.VerdictUncertain, confidence, nil
}
