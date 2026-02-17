package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

// ---------- Meta-Learning Store Tests ----------

func TestSetGetLearningRecord(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	record := types.LearningRecord{
		HormoneType:      "SSI",
		MultiplierPath:   types.PathSlashSystem,
		ApplicationCount: 10,
		ImprovementCount: 7,
		DegradationCount: 3,
		EffectivenessBps: 700_000,
		RateMultiplier:   1_100_000,
		LastUpdatedEpoch: 25,
	}
	k.SetLearningRecord(ctx, record)

	got := k.GetLearningRecord(ctx, types.PathSlashSystem)
	if got == nil {
		t.Fatal("expected learning record, got nil")
	}
	if got.HormoneType != "SSI" {
		t.Errorf("expected hormone SSI, got %s", got.HormoneType)
	}
	if got.RateMultiplier != 1_100_000 {
		t.Errorf("expected rate 1.1M, got %d", got.RateMultiplier)
	}
	if got.EffectivenessBps != 700_000 {
		t.Errorf("expected effectiveness 700k, got %d", got.EffectivenessBps)
	}
}

func TestGetLearningRecord_NotFound(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	got := k.GetLearningRecord(ctx, "nonexistent.path")
	if got != nil {
		t.Errorf("expected nil for nonexistent path, got %+v", got)
	}
}

func TestSetGetLearningWindow(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	window := types.LearningWindow{
		StartEpoch: 10,
		EndEpoch:   20,
		InitialMetrics: map[string]uint64{
			"system_stress": 500_000,
			"retention":     800_000,
		},
		FinalMetrics: map[string]uint64{
			"system_stress": 300_000,
			"retention":     850_000,
		},
		AdaptationsApplied: []types.AdaptationSummary{
			{Path: types.PathSlashSystem, HormoneType: "SSI", TotalDelta: 50_000, FireCount: 5},
		},
	}
	k.SetLearningWindow(ctx, window)

	got := k.GetLearningWindow(ctx, 20)
	if got == nil {
		t.Fatal("expected learning window, got nil")
	}
	if got.StartEpoch != 10 || got.EndEpoch != 20 {
		t.Errorf("epoch mismatch: start=%d end=%d", got.StartEpoch, got.EndEpoch)
	}
	if len(got.AdaptationsApplied) != 1 {
		t.Errorf("expected 1 adaptation summary, got %d", len(got.AdaptationsApplied))
	}
}

func TestSetGetMetaLearningState(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	state := types.MetaLearningState{
		WindowStartEpoch:  15,
		LastLearningEpoch: 20,
	}
	k.SetMetaLearningState(ctx, state)

	got := k.GetMetaLearningState(ctx)
	if got.WindowStartEpoch != 15 {
		t.Errorf("expected window start 15, got %d", got.WindowStartEpoch)
	}
	if got.LastLearningEpoch != 20 {
		t.Errorf("expected last learning 20, got %d", got.LastLearningEpoch)
	}
}

// ---------- GetAdaptationRate Tests ----------

func TestGetAdaptationRate_DefaultWhenNoRecord(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	rate := k.GetAdaptationRate(ctx, types.PathSlashSystem)
	if rate != types.DefaultRateMultiplier {
		t.Errorf("expected default rate %d, got %d", types.DefaultRateMultiplier, rate)
	}
}

func TestGetAdaptationRate_ReturnsStoredRate(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	k.SetLearningRecord(ctx, types.LearningRecord{
		MultiplierPath: types.PathSlashSystem,
		RateMultiplier: 1_500_000, // 1.5x
	})

	rate := k.GetAdaptationRate(ctx, types.PathSlashSystem)
	if rate != 1_500_000 {
		t.Errorf("expected rate 1.5M, got %d", rate)
	}
}

// ---------- RunMetaLearning Tests ----------

func TestRunMetaLearning_SkipsBeforeMinEpochs(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Epoch 10 < MinLearningEpochs (20) — should not create any records
	hormones := types.HormoneValues{Epoch: 10, SSI: 300_000}
	k.RunMetaLearning(ctx, 10, hormones)

	state := k.GetMetaLearningState(ctx)
	if state.LastLearningEpoch != 0 {
		t.Errorf("expected no learning state update, got epoch %d", state.LastLearningEpoch)
	}
}

func TestRunMetaLearning_SetsStateOnFirstRun(t *testing.T) {
	k, ctx, mockSK, mockKK, _ := setupKeeper(t)
	_ = mockSK
	_ = mockKK

	// Seed observations at epoch 10 and 20
	k.SetObservation(ctx, types.EpochObservation{
		Epoch:                10,
		ActiveValidators:     20,
		TotalValidators:      22,
		ValidatorAccuracyBps: 900_000,
		TotalActiveFacts:     100,
		FactsCreated:         10,
		FactsExpired:         2,
	})
	k.SetObservation(ctx, types.EpochObservation{
		Epoch:                20,
		ActiveValidators:     20,
		TotalValidators:      22,
		ValidatorAccuracyBps: 950_000, // improved accuracy
		TotalActiveFacts:     110,
		FactsCreated:         15,
		FactsExpired:         5,
	})

	hormones := types.HormoneValues{Epoch: 20, SSI: 200_000}
	k.RunMetaLearning(ctx, 20, hormones)

	state := k.GetMetaLearningState(ctx)
	if state.LastLearningEpoch != 20 {
		t.Errorf("expected last learning epoch 20, got %d", state.LastLearningEpoch)
	}
	if state.WindowStartEpoch != 20 {
		t.Errorf("expected window start 20, got %d", state.WindowStartEpoch)
	}
}

func TestRunMetaLearning_TracksAdaptationEffectiveness(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Seed observations
	k.SetObservation(ctx, types.EpochObservation{
		Epoch:                10,
		ActiveValidators:     15,
		TotalValidators:      22,
		ValidatorAccuracyBps: 700_000, // low accuracy → high SSI
		TotalActiveFacts:     100,
		FactsCreated:         5,
		FactsExpired:         10,
	})
	k.SetObservation(ctx, types.EpochObservation{
		Epoch:                20,
		ActiveValidators:     20,
		TotalValidators:      22,
		ValidatorAccuracyBps: 950_000, // improved! SSI should be lower
		TotalActiveFacts:     120,
		FactsCreated:         25,
		FactsExpired:         5,
	})

	// Seed adaptations in the window (SSI rule fired 5 times)
	for e := uint64(10); e < 15; e++ {
		k.SetAdaptations(ctx, e, []types.Adaptation{
			{Path: types.PathSlashSystem, Delta: 50_000, Hormone: "SSI", Epoch: e},
		})
	}

	k.SetMetaLearningState(ctx, types.MetaLearningState{
		WindowStartEpoch:  10,
		LastLearningEpoch: 10,
	})

	hormones := types.HormoneValues{Epoch: 20, SSI: 100_000}
	k.RunMetaLearning(ctx, 20, hormones)

	// SSI should have improved (stress decreased) → rule is effective
	record := k.GetLearningRecord(ctx, types.PathSlashSystem)
	if record == nil {
		t.Fatal("expected learning record for slash.system")
	}
	if record.ApplicationCount != 5 {
		t.Errorf("expected 5 applications, got %d", record.ApplicationCount)
	}
	// SSI decreased = improvement for SSI (lower is better)
	if record.ImprovementCount != 5 {
		t.Errorf("expected 5 improvements (SSI decreased), got %d", record.ImprovementCount)
	}
	// Rate should be boosted since effectiveness > 60%
	if record.RateMultiplier <= types.DefaultRateMultiplier {
		t.Errorf("expected boosted rate > 1M, got %d", record.RateMultiplier)
	}
}

func TestRunMetaLearning_DampensIneffectiveRules(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Seed observations where SSI got WORSE despite adaptations
	k.SetObservation(ctx, types.EpochObservation{
		Epoch:                10,
		ActiveValidators:     20,
		TotalValidators:      22,
		ValidatorAccuracyBps: 900_000,
		TotalActiveFacts:     100,
		FactsCreated:         10,
		FactsExpired:         2,
	})
	k.SetObservation(ctx, types.EpochObservation{
		Epoch:                20,
		ActiveValidators:     15,
		TotalValidators:      22,
		ValidatorAccuracyBps: 700_000, // accuracy got worse → SSI increased
		TotalActiveFacts:     80,
		FactsCreated:         3,
		FactsExpired:         25,
	})

	// Seed adaptations
	for e := uint64(10); e < 15; e++ {
		k.SetAdaptations(ctx, e, []types.Adaptation{
			{Path: types.PathSlashSystem, Delta: 50_000, Hormone: "SSI", Epoch: e},
		})
	}

	k.SetMetaLearningState(ctx, types.MetaLearningState{
		WindowStartEpoch:  10,
		LastLearningEpoch: 10,
	})

	hormones := types.HormoneValues{Epoch: 20, SSI: 600_000}
	k.RunMetaLearning(ctx, 20, hormones)

	record := k.GetLearningRecord(ctx, types.PathSlashSystem)
	if record == nil {
		t.Fatal("expected learning record for slash.system")
	}
	// SSI increased = degradation for SSI rule
	if record.DegradationCount != 5 {
		t.Errorf("expected 5 degradations, got %d", record.DegradationCount)
	}
	// Rate should be dampened since effectiveness < 40%
	if record.RateMultiplier >= types.DefaultRateMultiplier {
		t.Errorf("expected dampened rate < 1M, got %d", record.RateMultiplier)
	}
}

func TestRunMetaLearning_RateMultiplierBounded(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Pre-seed a record with a very high rate (near max)
	k.SetLearningRecord(ctx, types.LearningRecord{
		MultiplierPath:   types.PathSlashSystem,
		HormoneType:      "SSI",
		ApplicationCount: 100,
		ImprovementCount: 95,
		EffectivenessBps: 950_000, // very effective
		RateMultiplier:   3_900_000, // near max of 4M
	})

	k.SetObservation(ctx, types.EpochObservation{
		Epoch: 20, ActiveValidators: 20, TotalValidators: 22,
		ValidatorAccuracyBps: 600_000, TotalActiveFacts: 100,
		FactsCreated: 5, FactsExpired: 10,
	})
	k.SetObservation(ctx, types.EpochObservation{
		Epoch: 30, ActiveValidators: 20, TotalValidators: 22,
		ValidatorAccuracyBps: 950_000, TotalActiveFacts: 120,
		FactsCreated: 30, FactsExpired: 5,
	})
	for e := uint64(20); e < 25; e++ {
		k.SetAdaptations(ctx, e, []types.Adaptation{
			{Path: types.PathSlashSystem, Delta: 50_000, Hormone: "SSI", Epoch: e},
		})
	}

	k.SetMetaLearningState(ctx, types.MetaLearningState{
		WindowStartEpoch: 20, LastLearningEpoch: 20,
	})

	k.RunMetaLearning(ctx, 30, types.HormoneValues{Epoch: 30, SSI: 100_000})

	record := k.GetLearningRecord(ctx, types.PathSlashSystem)
	if record == nil {
		t.Fatal("expected record")
	}
	if record.RateMultiplier > types.BoundsAdaptationRate.Max {
		t.Errorf("rate %d exceeds max bound %d", record.RateMultiplier, types.BoundsAdaptationRate.Max)
	}
}

func TestRunMetaLearning_RateMultiplierFloor(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Pre-seed a record with a very low rate (near min)
	k.SetLearningRecord(ctx, types.LearningRecord{
		MultiplierPath:   types.PathSlashSystem,
		HormoneType:      "SSI",
		ApplicationCount: 100,
		ImprovementCount: 5,
		DegradationCount: 95,
		EffectivenessBps: 50_000, // very ineffective
		RateMultiplier:   300_000, // near min of 250k
	})

	k.SetObservation(ctx, types.EpochObservation{
		Epoch: 20, ActiveValidators: 20, TotalValidators: 22,
		ValidatorAccuracyBps: 900_000, TotalActiveFacts: 100,
		FactsCreated: 10, FactsExpired: 2,
	})
	k.SetObservation(ctx, types.EpochObservation{
		Epoch: 30, ActiveValidators: 15, TotalValidators: 22,
		ValidatorAccuracyBps: 700_000, TotalActiveFacts: 80,
		FactsCreated: 3, FactsExpired: 25,
	})
	for e := uint64(20); e < 25; e++ {
		k.SetAdaptations(ctx, e, []types.Adaptation{
			{Path: types.PathSlashSystem, Delta: 50_000, Hormone: "SSI", Epoch: e},
		})
	}

	k.SetMetaLearningState(ctx, types.MetaLearningState{
		WindowStartEpoch: 20, LastLearningEpoch: 20,
	})

	k.RunMetaLearning(ctx, 30, types.HormoneValues{Epoch: 30, SSI: 700_000})

	record := k.GetLearningRecord(ctx, types.PathSlashSystem)
	if record == nil {
		t.Fatal("expected record")
	}
	if record.RateMultiplier < types.BoundsAdaptationRate.Min {
		t.Errorf("rate %d below min bound %d", record.RateMultiplier, types.BoundsAdaptationRate.Min)
	}
}

// ---------- Type Tests ----------

func TestMetricDirectionLowerIsBetter(t *testing.T) {
	if !types.MetricDirectionLowerIsBetter("system_stress") {
		t.Error("system_stress should be lower-is-better")
	}
	if types.MetricDirectionLowerIsBetter("retention") {
		t.Error("retention should not be lower-is-better")
	}
	if types.MetricDirectionLowerIsBetter("domain_vitality") {
		t.Error("domain_vitality should not be lower-is-better")
	}
}

func TestIsEternalCategory_IncludesProtocolObservation(t *testing.T) {
	if !types.IsEternalCategory("protocol_observation") {
		t.Error("protocol_observation should be eternal")
	}
	if !types.IsEternalCategory("computational") {
		t.Error("computational should be eternal")
	}
	if !types.IsEternalCategory("analytic") {
		t.Error("analytic should be eternal")
	}
	if types.IsEternalCategory("empirical") {
		t.Error("empirical should NOT be eternal")
	}
}

// ---------- Integration: Meta-Learning via CollectAndAdapt ----------

func TestCollectAndAdapt_MetaLearningAtEpoch20(t *testing.T) {
	k, ctx, mockSK, mockKK, _ := setupKeeper(t)
	_ = mockSK
	_ = mockKK

	// Activate and seed through 20 epochs
	params := types.DefaultParams()
	params.EpochLengthBlocks = 10 // short epochs for testing
	k.SetParams(ctx, params)
	k.SetState(ctx, types.AutopoiesisState{Active: true, CurrentEpoch: 0})

	// Run CollectAndAdapt for epochs 1-20
	for block := int64(10); block <= 200; block += 10 {
		ctx = ctx.WithBlockHeight(block)
		k.CollectAndAdapt(ctx)
	}

	// After epoch 20, meta-learning should have run
	mlState := k.GetMetaLearningState(ctx)
	if mlState.LastLearningEpoch == 0 {
		t.Error("expected meta-learning to have run by epoch 20")
	}
}
