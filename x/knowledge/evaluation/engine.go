package evaluation

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// EvaluateClaim runs the 10-stage deterministic evaluation pipeline.
// All validators produce identical results for identical on-chain state.
//
// Stages:
//  1. Category base confidence
//  2. Submitter calibration (acceptance rate modifier)
//  3. Participant context (epistemic domain: derivable vs observable)
//  4. Evidence quality scoring
//  5. Fact graph analysis (references, contradictions, cycles)
//  6. Novelty assessment (information gain + bridge score)
//  7. Logic zone evaluation (Godelian handling)
//  8. Ceiling application (zone, stratum, category)
//  9. Edge case overrides
//  10. Counter-fact analysis (negative knowledge penalties)
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
		Stages: make([]StageResult, 0, 10),
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

	// Stage 3: Participant context (epistemic domain check)
	stage3 := evaluateParticipantContext(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage3)
	confidence = applyAdjustment(confidence, stage3.Adjustment)
	// Apply participant context ceiling (observable + agent-only evidence)
	if stage3.Ceiling > 0 && confidence > stage3.Ceiling {
		confidence = stage3.Ceiling
	}

	// Stage 4: Evidence quality
	stage4 := evaluateEvidence(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage4)
	confidence = applyAdjustment(confidence, stage4.Adjustment)

	// Stage 5: Fact graph analysis
	stage5 := evaluateFactGraph(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage5)
	confidence = applyAdjustment(confidence, stage5.Adjustment)

	// Stage 6: Novelty assessment (information gain + bridge score)
	// Passes the evidence stage adjustment as a quality gate
	stage6 := evaluateNovelty(ctx, evalCtx, claim, stage4.Adjustment)
	result.Stages = append(result.Stages, stage6)
	confidence = applyAdjustment(confidence, stage6.Adjustment)

	// Stage 7: Logic zone evaluation
	stage7 := evaluateLogicZone(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage7)

	// Stage 8: Ceiling application (zone + stratum + category)
	stage8 := applyCeilings(ctx, evalCtx, claim, stage7.Ceiling)
	result.Stages = append(result.Stages, stage8)
	if stage8.Ceiling > 0 && confidence > stage8.Ceiling {
		confidence = stage8.Ceiling
	}

	// Clamp to valid range
	if confidence > 1_000_000 {
		confidence = 1_000_000
	}

	// Stage 9: Edge cases (may force verdict)
	stage9 := evaluateEdgeCases(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage9)

	// Stage 10: Counter-fact analysis (negative knowledge penalties)
	stage10 := evaluateCounterFacts(ctx, evalCtx, claim)
	result.Stages = append(result.Stages, stage10)
	confidence = applyAdjustment(confidence, stage10.Adjustment)

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
