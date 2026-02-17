package evaluation

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// calibrationScalingFactor converts raw acceptance-rate deviation into a
// confidence adjustment. Derived from the desired output range:
//
//	Max deviation at 100% acceptance: +300,000 (1M - CalibrationNeutralRate 700k)
//	Desired pre-clamp output:         ~60,000  (slightly above MaxCalibrationBonus 50k)
//	Factor = 300,000 / 60,000 = 5
//
// This means:
//   - 80% acceptance (10% above neutral): deviation=+100k → adj=+20k (+2% confidence)
//   - 60% acceptance (10% below neutral): deviation=-100k → adj=-20k (-2% confidence)
//   - 100% acceptance: deviation=+300k → adj=+60k → clamped to +50k (MaxCalibrationBonus)
//   - 0% acceptance: deviation=-700k → adj=-140k → clamped to -150k (MaxCalibrationPenalty)
//
// The factor is intentionally conservative: even a perfect track record only earns +5%,
// while a terrible one penalizes up to -15%. This asymmetry discourages hallucination
// farming (submitting only "safe" claims to build a high bonus).
const calibrationScalingFactor = 5

// evaluateSubmitterCalibration applies a confidence modifier based on the
// submitter's historical claim acceptance rate.
//
// Stage 1.5 of the evaluation pipeline — inserted between category base and
// evidence quality. Rewards reliable submitters, penalizes hallucination-prone ones.
//
// - Submitters with < MinCalibrationSamples claims: no adjustment (unrated)
// - acceptance_rate = accepted * 1,000,000 / total
// - Deviation from CalibrationNeutralRate (70%) is scaled to an adjustment
// - Clamped to [-MaxCalibrationPenalty, +MaxCalibrationBonus]
func evaluateSubmitterCalibration(ctx sdk.Context, evalCtx *EvaluationContext, claim types.Claim) StageResult {
	result := StageResult{
		Name: "submitter_calibration",
	}

	if evalCtx.GetSubmitterCalibration == nil || claim.Submitter == "" {
		result.Justification = "calibration not available"
		return result
	}

	record, found := evalCtx.GetSubmitterCalibration(ctx, claim.Submitter)
	if !found || record.TotalClaims < types.MinCalibrationSamples {
		result.Justification = fmt.Sprintf("unrated submitter (%d/%d claims)",
			record.TotalClaims, types.MinCalibrationSamples)
		return result
	}

	// Calculate acceptance rate in basis points (0-1,000,000)
	acceptanceRate := (record.AcceptedClaims * 1_000_000) / record.TotalClaims

	var adjustment int64

	// FARM-2 fix: Suspiciously high acceptance rate (>95%) indicates trivial claim spam.
	// Instead of rewarding near-perfect acceptance, penalize it. The insight is that
	// genuine knowledge work has a natural rejection rate — if everything you submit
	// gets accepted, you're likely submitting only obvious truths.
	if acceptanceRate > types.CalibrationTrivialThreshold {
		// Scale penalty by how far above the trivial threshold
		excess := int64(acceptanceRate) - int64(types.CalibrationTrivialThreshold)
		// 50,000 range (950k→1M) maps to 0→MaxCalibrationTrivialPenalty
		adjustment = -(excess * int64(types.MaxCalibrationTrivialPenalty)) / 50_000
		if adjustment < -int64(types.MaxCalibrationTrivialPenalty) {
			adjustment = -int64(types.MaxCalibrationTrivialPenalty)
		}
	} else {
		// Normal calibration: Deviation from neutral point, scaled by calibrationScalingFactor.
		// Positive = above 70% → bonus, Negative = below 70% → penalty.
		deviation := int64(acceptanceRate) - int64(types.CalibrationNeutralRate)
		adjustment = deviation / calibrationScalingFactor

		// Clamp to bounds
		if adjustment > int64(types.MaxCalibrationBonus) {
			adjustment = int64(types.MaxCalibrationBonus)
		}
		if adjustment < -int64(types.MaxCalibrationPenalty) {
			adjustment = -int64(types.MaxCalibrationPenalty)
		}
	}

	result.Adjustment = adjustment
	result.Justification = fmt.Sprintf("calibration: %d/%d accepted (rate: %d, adjustment: %d)",
		record.AcceptedClaims, record.TotalClaims, acceptanceRate, adjustment)

	return result
}
