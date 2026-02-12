package evaluation

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// EvaluateClaim runs the 8-stage deterministic evaluation pipeline.
// All validators produce identical results for identical on-chain state.
//
// Stages:
//  1. Category base confidence
//  2. Submitter calibration (acceptance rate modifier)
//  3. Evidence quality scoring
//  4. Fact graph analysis (references, contradictions, cycles)
//  5. Logic zone evaluation (Godelian handling)
//  6. Ceiling application (zone, stratum, category)
//  7. Edge case overrides
//  8. Counter-fact analysis (negative knowledge penalties)
func EvaluateClaim(
	ctx sdk.Context,
	evalCtx *EvaluationContext,
	round types.VerificationRound,
	claim types.Claim,
) (EvaluationResult, error) {
	if evalCtx == nil {
		return EvaluationResult{Verdict: "abstain", Confidence: 0}, fmt.Errorf("nil evaluation context")
	}

	result := EvaluationResult{
		Stages: make([]StageResult, 0, 8),
	}

	// Stage 1: Category base confidence
	stage1 := evaluateCategory(claim.EpistemicCategory)
	result.Stages = append(result.Stages, stage1)
	confidence := uint64(0)
	if stage1.Adjustment > 0 {
		confidence = uint64(stage1.Adjustment)
	}

	// Stage 2: Submitter calibration (acceptance rate modifier)
	stage2 := evaluateSubmitterCalibration(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage2)
	confidence = applyAdjustment(confidence, stage2.Adjustment)

	// Stage 3: Evidence quality
	stage3 := evaluateEvidence(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage3)
	confidence = applyAdjustment(confidence, stage3.Adjustment)

	// Stage 4: Fact graph analysis
	stage4 := evaluateFactGraph(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage4)
	confidence = applyAdjustment(confidence, stage4.Adjustment)

	// Stage 5: Logic zone evaluation
	stage5 := evaluateLogicZone(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage5)

	// Stage 6: Ceiling application (zone + stratum + category)
	stage6 := applyCeilings(ctx, evalCtx, claim, stage5.Ceiling)
	result.Stages = append(result.Stages, stage6)
	if stage6.Ceiling > 0 && confidence > stage6.Ceiling {
		confidence = stage6.Ceiling
	}

	// Clamp to valid range
	if confidence > 1_000_000 {
		confidence = 1_000_000
	}

	// Stage 7: Edge cases (may force verdict)
	stage7 := evaluateEdgeCases(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage7)

	// Stage 8: Counter-fact analysis (negative knowledge penalties)
	stage8 := evaluateCounterFacts(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage8)
	confidence = applyAdjustment(confidence, stage8.Adjustment)

	// Determine verdict
	result.Confidence = confidence
	result.Verdict, result.Justification = determineVerdict(result)

	return result, nil
}

// determineVerdict applies forced verdicts from any stage, then threshold logic.
func determineVerdict(result EvaluationResult) (string, string) {
	// Check for forced verdicts (later stages override earlier ones)
	for i := len(result.Stages) - 1; i >= 0; i-- {
		stage := result.Stages[i]
		if stage.ForceVerdict == "reject" {
			return "reject", stage.Justification
		}
	}
	for i := len(result.Stages) - 1; i >= 0; i-- {
		stage := result.Stages[i]
		if stage.ForceVerdict == "abstain" {
			return "abstain", stage.Justification
		}
	}

	// Threshold-based verdict
	if result.Confidence >= AcceptThreshold {
		return "accept", fmt.Sprintf("confidence %d meets acceptance threshold %d", result.Confidence, AcceptThreshold)
	}
	if result.Confidence < RejectThreshold {
		return "reject", fmt.Sprintf("confidence %d below rejection threshold %d", result.Confidence, RejectThreshold)
	}
	return "abstain", fmt.Sprintf("confidence %d in inconclusive range [%d, %d)", result.Confidence, RejectThreshold, AcceptThreshold)
}

// applyAdjustment safely applies a signed adjustment to a uint64 confidence,
// clamping to [0, 1_000_000].
func applyAdjustment(confidence uint64, adjustment int64) uint64 {
	if adjustment >= 0 {
		confidence += uint64(adjustment)
		if confidence > 1_000_000 {
			confidence = 1_000_000
		}
		return confidence
	}
	// Negative adjustment
	neg := uint64(-adjustment)
	if neg > confidence {
		return 0
	}
	return confidence - neg
}
