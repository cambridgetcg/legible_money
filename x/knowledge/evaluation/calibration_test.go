package evaluation

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

func TestCalibration_NoKeeper(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: nil,
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)
	if result.Adjustment != 0 {
		t.Errorf("expected 0 adjustment without calibration keeper, got %d", result.Adjustment)
	}
}

func TestCalibration_EmptySubmitter(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{}, false
		},
	}
	claim := types.Claim{Id: "c1", Submitter: ""}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)
	if result.Adjustment != 0 {
		t.Errorf("expected 0 adjustment for empty submitter, got %d", result.Adjustment)
	}
}

func TestCalibration_Unrated(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{
				TotalClaims:    3, // below MinCalibrationSamples (5)
				AcceptedClaims: 2,
			}, true
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)
	if result.Adjustment != 0 {
		t.Errorf("expected 0 adjustment for unrated submitter, got %d", result.Adjustment)
	}
}

func TestCalibration_PerfectSubmitter(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{
				TotalClaims:    10,
				AcceptedClaims: 10, // 100% acceptance
			}, true
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)

	// FARM-2 fix: 100% acceptance is suspiciously trivial.
	// rate = 1,000,000, excess = 50,000, penalty = -(50000 * 100000) / 50000 = -100,000
	expected := -int64(types.MaxCalibrationTrivialPenalty)
	if result.Adjustment != expected {
		t.Errorf("expected %d (trivial penalty), got %d", expected, result.Adjustment)
	}
}

func TestCalibration_NeutralSubmitter(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{
				TotalClaims:    10,
				AcceptedClaims: 7, // 70% acceptance = neutral
			}, true
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)

	if result.Adjustment != 0 {
		t.Errorf("expected 0 for neutral rate, got %d", result.Adjustment)
	}
}

func TestCalibration_SerialHallucinator(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{
				TotalClaims:    10,
				AcceptedClaims: 0, // 0% acceptance
				RejectedClaims: 10,
			}, true
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)

	// rate = 0, deviation = -700,000, /5 = -140,000
	if result.Adjustment != -140_000 {
		t.Errorf("expected -140000, got %d", result.Adjustment)
	}
}

func TestCalibration_LowAcceptance(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{
				TotalClaims:    10,
				AcceptedClaims: 3, // 30% acceptance
				RejectedClaims: 7,
			}, true
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)

	// rate = 300,000, deviation = -400,000, /5 = -80,000
	if result.Adjustment != -80_000 {
		t.Errorf("expected -80000, got %d", result.Adjustment)
	}
}

func TestCalibration_HighAcceptance(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{
				TotalClaims:    10,
				AcceptedClaims: 9, // 90% acceptance
				RejectedClaims: 1,
			}, true
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)

	// rate = 900,000, deviation = 200,000, /5 = 40,000
	if result.Adjustment != 40_000 {
		t.Errorf("expected 40000, got %d", result.Adjustment)
	}
}

func TestCalibration_NotFound(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{}, false
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)
	if result.Adjustment != 0 {
		t.Errorf("expected 0 for not-found submitter, got %d", result.Adjustment)
	}
}

func TestCalibration_ExactMinSamples(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{
				TotalClaims:    5, // exactly MinCalibrationSamples
				AcceptedClaims: 5,
			}, true
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)

	// FARM-2 fix: 100% acceptance triggers trivial penalty even at min samples
	expected := -int64(types.MaxCalibrationTrivialPenalty)
	if result.Adjustment != expected {
		t.Errorf("expected %d (trivial penalty) at exact min samples, got %d", expected, result.Adjustment)
	}
}

func TestCalibration_TrivialThresholdBoundary(t *testing.T) {
	ctx := mockCtx()

	// 96% acceptance = just above trivial threshold (950,000)
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{
				TotalClaims:    100,
				AcceptedClaims: 96, // 96% acceptance
			}, true
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)

	// rate = 960,000, excess = 10,000, penalty = -(10000 * 100000) / 50000 = -20,000
	if result.Adjustment != -20_000 {
		t.Errorf("expected -20000 for 96%% acceptance, got %d", result.Adjustment)
	}
}

func TestCalibration_JustBelowTrivialThreshold(t *testing.T) {
	ctx := mockCtx()

	// 94% acceptance = below trivial threshold, gets normal bonus
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{
				TotalClaims:    100,
				AcceptedClaims: 94, // 94% acceptance — just below threshold
			}, true
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)

	// rate = 940,000, deviation from 700k = 240,000, /5 = 48,000 (bonus)
	if result.Adjustment != 48_000 {
		t.Errorf("expected 48000 bonus for 94%% acceptance, got %d", result.Adjustment)
	}
}

func TestCalibration_BelowMinSamples(t *testing.T) {
	ctx := mockCtx()
	evalCtx := &EvaluationContext{
		GetSubmitterCalibration: func(_ sdk.Context, _ string) (types.CalibrationRecord, bool) {
			return types.CalibrationRecord{
				TotalClaims:    4, // below MinCalibrationSamples
				AcceptedClaims: 0,
				RejectedClaims: 4,
			}, true
		},
	}
	claim := types.Claim{Id: "c1", Submitter: "lgm1abc"}

	result := evaluateSubmitterCalibration(ctx, evalCtx, claim)

	// Even with 0% acceptance, below min samples → no adjustment
	if result.Adjustment != 0 {
		t.Errorf("expected 0 below min samples, got %d", result.Adjustment)
	}
}
