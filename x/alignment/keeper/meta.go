package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/alignment/types"
)

// runMetaLoop executes the recursive meta-alignment check.
// It evaluates whether past corrections were effective and adjusts
// alignment system parameters accordingly.
//
// Runs every MetaLoopInterval epochs (default 10), matching autopoiesis Level 3.
func (k Keeper) runMetaLoop(ctx sdk.Context, epoch uint64, state *types.AlignmentState) {
	// Evaluate outcomes of corrections made CorrectionWindow epochs ago
	evaluationEpoch := epoch - uint64(types.CorrectionWindow)
	if evaluationEpoch == 0 || evaluationEpoch > epoch { // underflow guard
		return
	}

	outcomes := k.evaluateCorrectionOutcomes(ctx, evaluationEpoch, epoch)
	if len(outcomes) == 0 {
		return
	}
	k.SetCorrectionOutcomes(ctx, epoch, outcomes)

	// Compute AlignmentHealthIndex from outcomes
	ahi := k.computeAHI(outcomes)
	ahi.Epoch = epoch
	k.SetMetaHealth(ctx, ahi)

	// Apply meta-corrections based on AHI
	k.applyMetaCorrections(ctx, ahi, state)

	k.Logger(ctx).Info("alignment meta-loop",
		"epoch", epoch,
		"effectiveness", ahi.CorrectionEffectiveness,
		"overcorrection", ahi.OvercorrectionRate,
		"stability", ahi.SensorStability,
		"composite", ahi.Composite,
	)
}

// evaluateCorrectionOutcomes compares dimension scores before and after corrections.
func (k Keeper) evaluateCorrectionOutcomes(ctx sdk.Context, correctionEpoch, currentEpoch uint64) []types.CorrectionOutcome {
	corrections := k.GetCorrections(ctx, correctionEpoch)
	if len(corrections) == 0 {
		return nil
	}

	scoresBefore := k.GetScores(ctx, correctionEpoch)
	scoresAfter := k.GetScores(ctx, currentEpoch)
	if scoresBefore == nil || scoresAfter == nil {
		return nil
	}

	// Track which dimensions had corrections
	dimCorrected := make(map[types.Dimension]bool)
	for _, c := range corrections {
		if c.Delta != 0 {
			dimCorrected[c.Dimension] = true
		}
	}

	var outcomes []types.CorrectionOutcome
	for dim := range dimCorrected {
		before := getDimensionScore(*scoresBefore, dim)
		after := getDimensionScore(*scoresAfter, dim)

		improved := false
		overcorrected := false

		switch dim {
		case types.DimensionTruth:
			// TDI: lower is better
			improved = after < before
			if before > 0 && after < before {
				improvement := before - after
				overcorrected = improvement > types.OvercorrectionBPS
			}
		default:
			// DCI, HI, FI, NI: higher is better
			improved = after > before
			if after > before {
				improvement := after - before
				overcorrected = improvement > types.OvercorrectionBPS
			}
		}

		outcomes = append(outcomes, types.CorrectionOutcome{
			CorrectionEpoch: correctionEpoch,
			Dimension:       dim,
			ScoreBefore:     before,
			ScoreAfter:      after,
			Improved:        improved,
			Overcorrected:   overcorrected,
		})
	}

	return outcomes
}

// computeAHI computes the Alignment Health Index from correction outcomes.
func (k Keeper) computeAHI(outcomes []types.CorrectionOutcome) types.AlignmentHealthIndex {
	if len(outcomes) == 0 {
		return types.AlignmentHealthIndex{
			CorrectionEffectiveness: types.BasisPoints, // no data = assume healthy
			Composite:               types.BasisPoints,
		}
	}

	total := uint64(len(outcomes))
	var improved, overcorrected uint64
	for _, o := range outcomes {
		if o.Improved {
			improved++
		}
		if o.Overcorrected {
			overcorrected++
		}
	}

	effectiveness := (improved * types.BasisPoints) / total
	overcorrectionRate := (overcorrected * types.BasisPoints) / total

	// Sensor stability: variance of score changes
	var totalChange uint64
	for _, o := range outcomes {
		if o.ScoreAfter >= o.ScoreBefore {
			totalChange += o.ScoreAfter - o.ScoreBefore
		} else {
			totalChange += o.ScoreBefore - o.ScoreAfter
		}
	}
	avgChange := totalChange / total
	// High change variance = unstable
	stability := types.ClampUint64(avgChange, 0, types.BasisPoints)

	// Composite AHI (higher = healthier)
	// effectiveness contributes positively, overcorrection and instability negatively
	composite := uint64(0)
	if effectiveness > overcorrectionRate+stability/2 {
		composite = effectiveness - overcorrectionRate - stability/2
	}

	return types.AlignmentHealthIndex{
		CorrectionEffectiveness: effectiveness,
		OvercorrectionRate:     overcorrectionRate,
		SensorStability:        stability,
		FalsePositiveRate:      types.BasisPoints - effectiveness, // approximation
		Composite:              types.ClampUint64(composite, 0, types.BasisPoints),
	}
}

// applyMetaCorrections adjusts alignment system parameters based on AHI.
func (k Keeper) applyMetaCorrections(ctx sdk.Context, ahi types.AlignmentHealthIndex, state *types.AlignmentState) {
	// If corrections don't work (low effectiveness) → increase correction magnitude
	if ahi.CorrectionEffectiveness < 400_000 { // < 40% effective
		// Reduce damping (= stronger corrections)
		if state.CorrectionDamping > types.MagnitudeIncrementBPS {
			state.CorrectionDamping -= types.MagnitudeIncrementBPS
		}
	}

	// If overcorrecting → increase damping
	if ahi.OvercorrectionRate > 300_000 { // > 30% overcorrection
		if state.CorrectionDamping < types.BasisPoints {
			state.CorrectionDamping += types.DampingIncrementBPS
			if state.CorrectionDamping > types.BasisPoints {
				state.CorrectionDamping = types.BasisPoints
			}
		}
	}

	// If sensors are noisy → increase smoothing window
	if ahi.SensorStability > 500_000 { // high instability
		if state.SmoothingWindow < uint64(types.SmoothingWindowMax) {
			state.SmoothingWindow++
		}
	} else if ahi.SensorStability < 100_000 { // very stable
		if state.SmoothingWindow > uint64(types.SmoothingWindowMin) {
			state.SmoothingWindow--
		}
	}

	// Clamp damping to reasonable range [50%, 100%]
	if state.CorrectionDamping < 500_000 {
		state.CorrectionDamping = 500_000
	}
}

// getDimensionScore extracts the score for a specific dimension from DimensionScores.
func getDimensionScore(scores types.DimensionScores, dim types.Dimension) uint64 {
	switch dim {
	case types.DimensionTruth:
		return scores.TDI
	case types.DimensionUnderstanding:
		return scores.DCI
	case types.DimensionBeauty:
		return scores.HI
	case types.DimensionJustice:
		return scores.FI
	case types.DimensionCreativity:
		return scores.NI
	default:
		return 0
	}
}
