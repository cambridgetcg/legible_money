package evaluation

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

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

	// Deviation from neutral point, scaled by factor of 5
	// Positive = above 70% → bonus, Negative = below 70% → penalty
	deviation := int64(acceptanceRate) - int64(types.CalibrationNeutralRate)
	adjustment := deviation / 5

	// Clamp to bounds
	if adjustment > int64(types.MaxCalibrationBonus) {
		adjustment = int64(types.MaxCalibrationBonus)
	}
	if adjustment < -int64(types.MaxCalibrationPenalty) {
		adjustment = -int64(types.MaxCalibrationPenalty)
	}

	result.Adjustment = adjustment
	result.Justification = fmt.Sprintf("calibration: %d/%d accepted (rate: %d, adjustment: %d)",
		record.AcceptedClaims, record.TotalClaims, acceptanceRate, adjustment)

	return result
}
