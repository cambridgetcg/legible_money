package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/alignment/types"
)

// computeCorrections determines corrective actions based on dimension scores.
// Each dimension is checked against its threshold ladder: mild → moderate → severe → critical.
// Corrections flow through autopoiesis multiplier paths at priority 5.
func (k Keeper) computeCorrections(
	ctx sdk.Context,
	scores types.DimensionScores,
	state types.AlignmentState,
	params types.Params,
) []types.CorrectionRecord {
	var corrections []types.CorrectionRecord

	if params.TruthEnabled {
		corrections = append(corrections, k.computeTruthCorrections(ctx, scores.TDI, state)...)
	}
	if params.UnderstandingEnabled {
		corrections = append(corrections, k.computeUnderstandingCorrections(ctx, scores.DCI, state)...)
	}
	if params.BeautyEnabled {
		corrections = append(corrections, k.computeBeautyCorrections(ctx, scores.HI, state)...)
	}
	if params.JusticeEnabled {
		corrections = append(corrections, k.computeJusticeCorrections(ctx, scores.FI, state)...)
	}
	if params.CreativityEnabled {
		corrections = append(corrections, k.computeCreativityCorrections(ctx, scores.NI, state)...)
	}

	// Enforce max corrections limit
	if uint64(len(corrections)) > params.MaxCorrections {
		// Sort by severity: critical > severe > moderate > mild
		// Then truncate. Use stable sort to preserve dimension order on ties.
		sortBySeverity(corrections)
		corrections = corrections[:params.MaxCorrections]
	}

	return corrections
}

// computeTruthCorrections handles TDI-based corrections.
// TDI > threshold → corrective actions to restore truth.
func (k Keeper) computeTruthCorrections(ctx sdk.Context, tdi uint64, state types.AlignmentState) []types.CorrectionRecord {
	if k.autopoiesisKeeper == nil {
		return nil
	}

	var corrections []types.CorrectionRecord
	damping := state.CorrectionDamping

	// Critical (TDI > 80%): ALIGNMENT OVERRIDE
	if tdi > types.TDICriticalThreshold {
		// Increase challenge reward substantially
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionTruth, types.PathChallengeReward,
				int64(types.MaxChangePerEpochBPS*3), types.BoundsChallengeReward, damping, "critical"),
		)
		// Decrease unanimity confidence substantially
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionTruth, types.PathUnanimityPenalty,
				int64(types.MaxChangePerEpochBPS*2), types.BoundsUnanimityPenalty, damping, "critical"),
		)
		return corrections
	}

	// Severe (TDI > 60%): Strong corrections
	if tdi > types.TDISevereThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionTruth, types.PathUnanimityPenalty,
				int64(types.MaxChangePerEpochBPS*2), types.BoundsUnanimityPenalty, damping, "severe"),
		)
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionTruth, types.PathChallengeReward,
				int64(types.MaxChangePerEpochBPS*2), types.BoundsChallengeReward, damping, "severe"),
		)
		return corrections
	}

	// Moderate (TDI > 40%): Increase challenge reward
	if tdi > types.TDIModerateThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionTruth, types.PathChallengeReward,
				int64(types.MaxChangePerEpochBPS), types.BoundsChallengeReward, damping, "moderate"),
		)
		return corrections
	}

	// Mild (TDI > 20%): Minor adjustment to min verifiers
	if tdi > types.TDIMildThreshold {
		// Use existing autopoiesis path for min verifiers
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionTruth, "minverifiers.system",
				int64(types.MaxChangePerEpochBPS/2),
				types.MultiplierBounds{Min: 0, Max: 5_000_000}, damping, "mild"),
		)
		return corrections
	}

	// Healthy: relax truth corrections toward default
	corrections = append(corrections, k.relaxCorrections(ctx, types.DimensionTruth,
		[]pathBounds{
			{types.PathChallengeReward, types.BoundsChallengeReward},
			{types.PathUnanimityPenalty, types.BoundsUnanimityPenalty},
		}, damping)...)

	return corrections
}

// computeUnderstandingCorrections handles DCI-based corrections.
// DCI < threshold → corrective actions to deepen understanding.
func (k Keeper) computeUnderstandingCorrections(ctx sdk.Context, dci uint64, state types.AlignmentState) []types.CorrectionRecord {
	if k.autopoiesisKeeper == nil {
		return nil
	}

	var corrections []types.CorrectionRecord
	damping := state.CorrectionDamping

	// Severe (DCI < 20%): Strong corrections on orphan cap + cross-domain bonus
	if dci < types.DCISevereThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionUnderstanding, types.PathOrphanCap,
				-int64(types.MaxChangePerEpochBPS*2), types.BoundsOrphanCap, damping, "severe"),
		)
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionUnderstanding, types.PathCrossDomainBonus,
				int64(types.MaxChangePerEpochBPS*2), types.BoundsCrossDomainBonus, damping, "severe"),
		)
		return corrections
	}

	// Moderate (DCI < 40%): Decrease orphan cap + increase cross-domain bonus
	if dci < types.DCIModerateThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionUnderstanding, types.PathOrphanCap,
				-int64(types.MaxChangePerEpochBPS), types.BoundsOrphanCap, damping, "moderate"),
		)
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionUnderstanding, types.PathCrossDomainBonus,
				int64(types.MaxChangePerEpochBPS), types.BoundsCrossDomainBonus, damping, "moderate"),
		)
		return corrections
	}

	// Mild (DCI < 60%): Increase citation reward weight
	if dci < types.DCIMildThreshold {
		// Use existing autopoiesis citation path pattern
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionUnderstanding, types.PathCrossDomainBonus,
				int64(types.MaxChangePerEpochBPS/2), types.BoundsCrossDomainBonus, damping, "mild"),
		)
		return corrections
	}

	// Healthy: relax understanding corrections toward default
	corrections = append(corrections, k.relaxCorrections(ctx, types.DimensionUnderstanding,
		[]pathBounds{
			{types.PathOrphanCap, types.BoundsOrphanCap},
			{types.PathCrossDomainBonus, types.BoundsCrossDomainBonus},
		}, damping)...)

	return corrections
}

// computeBeautyCorrections handles HI-based corrections.
// HI < threshold → corrective actions to restore harmony.
func (k Keeper) computeBeautyCorrections(ctx sdk.Context, hi uint64, state types.AlignmentState) []types.CorrectionRecord {
	if k.autopoiesisKeeper == nil {
		return nil
	}

	var corrections []types.CorrectionRecord
	damping := state.CorrectionDamping

	// Severe (HI < 20%): BEAUTY ALARM — increase damping substantially
	if hi < types.HISevereThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionBeauty, types.PathAlignmentDamping,
				int64(types.MaxChangePerEpochBPS*3), types.BoundsAlignmentDamping, damping, "severe"),
		)
		return corrections
	}

	// Moderate (HI < 40%): Increase damping
	if hi < types.HIModerateThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionBeauty, types.PathAlignmentDamping,
				int64(types.MaxChangePerEpochBPS*2), types.BoundsAlignmentDamping, damping, "moderate"),
		)
		return corrections
	}

	// Mild (HI < 60%): Small damping increase
	if hi < types.HIMildThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionBeauty, types.PathAlignmentDamping,
				int64(types.MaxChangePerEpochBPS), types.BoundsAlignmentDamping, damping, "mild"),
		)
		return corrections
	}

	// Healthy: relax damping toward default
	corrections = append(corrections, k.relaxCorrections(ctx, types.DimensionBeauty,
		[]pathBounds{
			{types.PathAlignmentDamping, types.BoundsAlignmentDamping},
		}, damping)...)

	return corrections
}

// computeJusticeCorrections handles FI-based corrections.
// FI < threshold → corrective actions to redistribute opportunity.
func (k Keeper) computeJusticeCorrections(ctx sdk.Context, fi uint64, state types.AlignmentState) []types.CorrectionRecord {
	if k.autopoiesisKeeper == nil {
		return nil
	}

	var corrections []types.CorrectionRecord
	damping := state.CorrectionDamping

	// Severe (FI < 20%): JUSTICE ALARM — boost quadratic voting + reduce claim stake
	if fi < types.FISevereThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionJustice, types.PathQuadraticBoost,
				int64(types.MaxChangePerEpochBPS*3), types.BoundsQuadraticBoost, damping, "severe"),
		)
		return corrections
	}

	// Moderate (FI < 40%): Reduce new entrant barrier
	if fi < types.FIModerateThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionJustice, types.PathQuadraticBoost,
				int64(types.MaxChangePerEpochBPS*2), types.BoundsQuadraticBoost, damping, "moderate"),
		)
		return corrections
	}

	// Mild (FI < 60%): Small quadratic boost
	if fi < types.FIMildThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionJustice, types.PathQuadraticBoost,
				int64(types.MaxChangePerEpochBPS), types.BoundsQuadraticBoost, damping, "mild"),
		)
		return corrections
	}

	// Healthy: relax justice corrections toward default
	corrections = append(corrections, k.relaxCorrections(ctx, types.DimensionJustice,
		[]pathBounds{
			{types.PathQuadraticBoost, types.BoundsQuadraticBoost},
		}, damping)...)

	return corrections
}

// computeCreativityCorrections handles NI-based corrections.
// NI < threshold → corrective actions to incentivize exploration.
func (k Keeper) computeCreativityCorrections(ctx sdk.Context, ni uint64, state types.AlignmentState) []types.CorrectionRecord {
	if k.autopoiesisKeeper == nil {
		return nil
	}

	var corrections []types.CorrectionRecord
	damping := state.CorrectionDamping

	// Severe (NI < 20%): STAGNATION ALARM
	if ni < types.NISevereThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionCreativity, types.PathCrossDomainBonus,
				int64(types.MaxChangePerEpochBPS*2), types.BoundsCrossDomainBonus, damping, "severe"),
		)
		return corrections
	}

	// Moderate (NI < 40%): Boost cross-domain incentives
	if ni < types.NIModerateThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionCreativity, types.PathCrossDomainBonus,
				int64(types.MaxChangePerEpochBPS), types.BoundsCrossDomainBonus, damping, "moderate"),
		)
		return corrections
	}

	// Mild (NI < 60%): Small incentive
	if ni < types.NIMildThreshold {
		corrections = append(corrections,
			k.makeCorrection(ctx, types.DimensionCreativity, types.PathCrossDomainBonus,
				int64(types.MaxChangePerEpochBPS/2), types.BoundsCrossDomainBonus, damping, "mild"),
		)
		return corrections
	}

	return corrections
}

// ============ Correction Application ============

// applyCorrections writes corrective multiplier entries to autopoiesis store.
func (k Keeper) applyCorrections(ctx sdk.Context, corrections []types.CorrectionRecord, epoch uint64) {
	if k.autopoiesisKeeper == nil {
		return
	}

	for i := range corrections {
		corrections[i].Epoch = epoch
		if corrections[i].Delta != 0 {
			k.autopoiesisKeeper.SetMultiplier(ctx, types.MultiplierEntry{
				Path:    corrections[i].Path,
				Value:   corrections[i].NewValue,
				Hormone: "ALIGNMENT:" + string(corrections[i].Dimension),
				Epoch:   epoch,
			})
		}
	}
}

// ============ Helpers ============

type pathBounds struct {
	Path   string
	Bounds types.MultiplierBounds
}

// makeCorrection creates a CorrectionRecord by reading current value and applying damped delta.
func (k Keeper) makeCorrection(
	ctx sdk.Context,
	dim types.Dimension,
	path string,
	rawDelta int64,
	bounds types.MultiplierBounds,
	dampingBPS uint64,
	severity string,
) types.CorrectionRecord {
	current := k.autopoiesisKeeper.GetMultiplier(ctx, path)
	effectiveDelta := types.ApplyDampedDelta(rawDelta, current, bounds, types.MaxChangePerEpochBPS, dampingBPS)

	return types.CorrectionRecord{
		Dimension: dim,
		Path:      path,
		OldValue:  current,
		NewValue:  types.ApplySignedDelta(current, effectiveDelta),
		Delta:     effectiveDelta,
		Severity:  severity,
		Priority:  types.AlignmentPriority,
	}
}

// relaxCorrections generates small corrections toward DefaultMultiplier for paths that have drifted.
func (k Keeper) relaxCorrections(
	ctx sdk.Context,
	dim types.Dimension,
	paths []pathBounds,
	dampingBPS uint64,
) []types.CorrectionRecord {
	var corrections []types.CorrectionRecord
	for _, pb := range paths {
		current := k.autopoiesisKeeper.GetMultiplier(ctx, pb.Path)
		if current == types.DefaultMultiplier {
			continue
		}
		var rawDelta int64
		if current > types.DefaultMultiplier {
			rawDelta = -int64(types.MaxChangePerEpochBPS / 4) // relax slowly
		} else {
			rawDelta = int64(types.MaxChangePerEpochBPS / 4)
		}
		effectiveDelta := types.ApplyDampedDelta(rawDelta, current, pb.Bounds, types.MaxChangePerEpochBPS, dampingBPS)
		if effectiveDelta == 0 {
			continue
		}
		corrections = append(corrections, types.CorrectionRecord{
			Dimension: dim,
			Path:      pb.Path,
			OldValue:  current,
			NewValue:  types.ApplySignedDelta(current, effectiveDelta),
			Delta:     effectiveDelta,
			Severity:  "relax",
			Priority:  types.AlignmentPriority,
		})
	}
	return corrections
}

// sortBySeverity sorts corrections by severity (critical first, mild last).
func sortBySeverity(corrections []types.CorrectionRecord) {
	severityOrder := map[string]int{
		"critical": 0,
		"severe":   1,
		"moderate": 2,
		"mild":     3,
		"relax":    4,
	}
	// Stable sort preserves dimension ordering on ties
	n := len(corrections)
	for i := 1; i < n; i++ {
		for j := i; j > 0; j-- {
			ai := severityOrder[corrections[j].Severity]
			bi := severityOrder[corrections[j-1].Severity]
			if ai < bi {
				corrections[j], corrections[j-1] = corrections[j-1], corrections[j]
			} else {
				break
			}
		}
	}
}
