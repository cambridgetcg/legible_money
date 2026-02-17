package keeper

import (
	"encoding/json"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

// ============ Meta-Learning (Loop 2 — Level 3 Recursion) ============
//
// RunMetaLearning tracks which adaptation rules actually improve their target
// metrics. Called every LearningIntervalEpochs (10) from CollectAndAdapt.
// Effective rules get rate multipliers boosted; ineffective rules are dampened.

// RunMetaLearning evaluates adaptation effectiveness over the last learning window.
// It compares metrics at window start vs window end for each active adaptation rule.
func (k Keeper) RunMetaLearning(ctx sdk.Context, currentEpoch uint64, hormones types.HormoneValues) {
	mlState := k.GetMetaLearningState(ctx)

	// Guard: minimum epochs of data before activation
	if currentEpoch < types.MinLearningEpochs {
		return
	}

	// Guard: only run at learning interval boundaries
	if mlState.LastLearningEpoch > 0 && currentEpoch-mlState.LastLearningEpoch < uint64(types.LearningIntervalEpochs) {
		return
	}

	windowStart := mlState.WindowStartEpoch
	if windowStart == 0 && currentEpoch > uint64(types.LearningIntervalEpochs) {
		windowStart = currentEpoch - uint64(types.LearningIntervalEpochs)
	}

	// Collect metrics from window boundaries
	startObs := k.GetObservation(ctx, windowStart)
	endObs := k.GetObservation(ctx, currentEpoch)
	if startObs == nil || endObs == nil {
		// Not enough data — update state and return
		k.SetMetaLearningState(ctx, types.MetaLearningState{
			WindowStartEpoch:  currentEpoch,
			LastLearningEpoch: currentEpoch,
		})
		return
	}

	// Build metric snapshots from observations
	startMetrics := k.extractMetrics(startObs, windowStart)
	endMetrics := k.extractMetrics(endObs, currentEpoch)

	// Collect all adaptations that fired during this window
	adaptationSummary := k.collectWindowAdaptations(ctx, windowStart, currentEpoch)

	// For each adaptation rule that fired, evaluate effectiveness
	for _, summary := range adaptationSummary {
		record := k.GetLearningRecord(ctx, summary.Path)
		if record == nil {
			record = &types.LearningRecord{
				HormoneType:    summary.HormoneType,
				MultiplierPath: summary.Path,
				RateMultiplier: types.DefaultRateMultiplier,
			}
		}

		improved := k.evaluateImprovement(summary.HormoneType, startMetrics, endMetrics)

		record.ApplicationCount += summary.FireCount
		if improved {
			record.ImprovementCount += summary.FireCount
		} else {
			record.DegradationCount += summary.FireCount
		}

		// Compute effectiveness BPS
		if record.ApplicationCount > 0 {
			record.EffectivenessBps = (record.ImprovementCount * types.BasisPoints) / record.ApplicationCount
		}

		// Adjust the adaptation rate multiplier
		k.adjustAdaptationRate(record)
		record.LastUpdatedEpoch = currentEpoch

		k.SetLearningRecord(ctx, *record)
	}

	// Save learning window for audit
	window := types.LearningWindow{
		StartEpoch:         windowStart,
		EndEpoch:           currentEpoch,
		InitialMetrics:     startMetrics,
		FinalMetrics:       endMetrics,
		AdaptationsApplied: adaptationSummary,
	}
	k.SetLearningWindow(ctx, window)

	// Update state: next window starts at current epoch
	k.SetMetaLearningState(ctx, types.MetaLearningState{
		WindowStartEpoch:  currentEpoch,
		LastLearningEpoch: currentEpoch,
	})

	k.Logger(ctx).Info("meta-learning completed",
		"epoch", currentEpoch,
		"window_start", windowStart,
		"rules_evaluated", len(adaptationSummary),
	)
}

// extractMetrics builds a metric map from an EpochObservation.
// Keys correspond to HormoneTargetMetric values.
func (k Keeper) extractMetrics(obs *types.EpochObservation, epoch uint64) map[string]uint64 {
	metrics := make(map[string]uint64)

	// System stress (SSI)
	ssi := types.ComputeSSI(*obs)
	metrics["system_stress"] = ssi

	// Retention (average CRI across domains)
	cri := types.ComputeCRI(obs.Retentions)
	if len(cri) > 0 {
		var sum uint64
		for _, c := range cri {
			sum += c.Value
		}
		metrics["retention"] = sum / uint64(len(cri))
	} else {
		metrics["retention"] = types.BasisPoints // perfect if no data
	}

	// Domain vitality (average absolute DVI across domains)
	dvi := types.ComputeDVI(obs.Domains, nil)
	if len(dvi) > 0 {
		var sum int64
		for _, d := range dvi {
			sum += d.Value
		}
		avg := sum / int64(len(dvi))
		if avg >= 0 {
			metrics["domain_vitality"] = uint64(avg)
		} else {
			metrics["domain_vitality"] = 0 // negative DVI maps to 0 for improvement tracking
		}
	}

	// Cascade factor (average VCF)
	vcf := types.ComputeVCF(obs.Domains)
	if len(vcf) > 0 {
		var sum uint64
		for _, v := range vcf {
			sum += v.Value
		}
		metrics["cascade_factor"] = sum / uint64(len(vcf))
	}

	// Growth utilization (GRM)
	metrics["growth_utilization"] = types.ComputeGRM(*obs)

	return metrics
}

// collectWindowAdaptations gathers all adaptations that fired between windowStart and currentEpoch.
func (k Keeper) collectWindowAdaptations(ctx sdk.Context, windowStart, currentEpoch uint64) []types.AdaptationSummary {
	pathSummary := make(map[string]*types.AdaptationSummary)

	for epoch := windowStart; epoch <= currentEpoch; epoch++ {
		adaptations := k.GetAdaptations(ctx, epoch)
		for _, a := range adaptations {
			if a.Delta == 0 {
				continue
			}
			key := a.Path
			if existing, ok := pathSummary[key]; ok {
				existing.TotalDelta += a.Delta
				existing.FireCount++
			} else {
				pathSummary[key] = &types.AdaptationSummary{
					Path:        a.Path,
					HormoneType: a.Hormone,
					TotalDelta:  a.Delta,
					FireCount:   1,
				}
			}
		}
	}

	// Convert to sorted slice for deterministic ordering
	result := make([]types.AdaptationSummary, 0, len(pathSummary))
	for _, s := range pathSummary {
		result = append(result, *s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	return result
}

// evaluateImprovement checks whether a hormone's target metric improved.
func (k Keeper) evaluateImprovement(hormoneType string, startMetrics, endMetrics map[string]uint64) bool {
	metricKey, ok := types.HormoneTargetMetric[hormoneType]
	if !ok {
		return false
	}

	startVal, startOk := startMetrics[metricKey]
	endVal, endOk := endMetrics[metricKey]
	if !startOk || !endOk {
		return false
	}

	if types.MetricDirectionLowerIsBetter(metricKey) {
		return endVal < startVal
	}
	return endVal > startVal
}

// adjustAdaptationRate modifies the rate multiplier based on effectiveness.
// Effective rules (>60%): rate *= 1.1 (capped at 4.0x)
// Ineffective rules (<40%): rate *= 0.9 (floored at 0.25x)
// Neutral (40-60%): no change.
func (k Keeper) adjustAdaptationRate(record *types.LearningRecord) {
	if record.EffectivenessBps > types.NeutralHighBps {
		// Effective: boost rate by 10%
		newRate := record.RateMultiplier + (record.RateMultiplier * types.EffectiveBoostBps / types.BasisPoints)
		if newRate > types.BoundsAdaptationRate.Max {
			newRate = types.BoundsAdaptationRate.Max
		}
		record.RateMultiplier = newRate
	} else if record.EffectivenessBps < types.NeutralLowBps {
		// Ineffective: dampen rate by 10%
		reduction := record.RateMultiplier * types.IneffectiveDampenBps / types.BasisPoints
		if reduction >= record.RateMultiplier {
			record.RateMultiplier = types.BoundsAdaptationRate.Min
		} else {
			newRate := record.RateMultiplier - reduction
			if newRate < types.BoundsAdaptationRate.Min {
				newRate = types.BoundsAdaptationRate.Min
			}
			record.RateMultiplier = newRate
		}
	}
	// Neutral: no change
}

// GetAdaptationRate returns the learned rate multiplier for a path (BPS, default 1M).
// Consumer modules use this to scale rawDelta in adaptation functions.
func (k Keeper) GetAdaptationRate(ctx sdk.Context, path string) uint64 {
	record := k.GetLearningRecord(ctx, path)
	if record == nil {
		return types.DefaultRateMultiplier
	}
	return record.RateMultiplier
}

// ============ Learning Store Methods ============

func learningRecordKey(path string) []byte {
	return append(types.LearningRecordKeyPrefix, []byte(path)...)
}

func learningWindowKey(epoch uint64) []byte {
	return append(types.LearningWindowKeyPrefix, []byte(fmt.Sprintf("%016d", epoch))...)
}

// SetLearningRecord stores a learning record for a multiplier path.
func (k Keeper) SetLearningRecord(ctx sdk.Context, record types.LearningRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(record)
	if err != nil {
		return
	}
	_ = kvStore.Set(learningRecordKey(record.MultiplierPath), bz)
}

// GetLearningRecord returns the learning record for a path, or nil.
func (k Keeper) GetLearningRecord(ctx sdk.Context, path string) *types.LearningRecord {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(learningRecordKey(path))
	if err != nil || bz == nil {
		return nil
	}
	var record types.LearningRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return nil
	}
	return &record
}

// SetLearningWindow stores a learning window for an epoch.
func (k Keeper) SetLearningWindow(ctx sdk.Context, window types.LearningWindow) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(window)
	if err != nil {
		return
	}
	_ = kvStore.Set(learningWindowKey(window.EndEpoch), bz)
}

// GetLearningWindow returns the learning window for an epoch, or nil.
func (k Keeper) GetLearningWindow(ctx sdk.Context, epoch uint64) *types.LearningWindow {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(learningWindowKey(epoch))
	if err != nil || bz == nil {
		return nil
	}
	var window types.LearningWindow
	if err := json.Unmarshal(bz, &window); err != nil {
		return nil
	}
	return &window
}

// SetMetaLearningState stores the meta-learning state.
func (k Keeper) SetMetaLearningState(ctx sdk.Context, state types.MetaLearningState) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(state)
	if err != nil {
		return
	}
	_ = kvStore.Set(types.MetaLearningStateKey, bz)
}

// GetMetaLearningState returns the meta-learning state.
func (k Keeper) GetMetaLearningState(ctx sdk.Context) types.MetaLearningState {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.MetaLearningStateKey)
	if err != nil || bz == nil {
		return types.MetaLearningState{}
	}
	var state types.MetaLearningState
	if err := json.Unmarshal(bz, &state); err != nil {
		return types.MetaLearningState{}
	}
	return state
}

// GetAdaptations returns the adaptation records for an epoch, or nil.
func (k Keeper) GetAdaptations(ctx sdk.Context, epoch uint64) []types.Adaptation {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(adaptationKey(epoch))
	if err != nil || bz == nil {
		return nil
	}
	var adaptations []types.Adaptation
	if err := json.Unmarshal(bz, &adaptations); err != nil {
		return nil
	}
	return adaptations
}
