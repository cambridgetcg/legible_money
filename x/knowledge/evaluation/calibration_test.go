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

	// rate = 1,000,000, deviation = 300,000, /5 = 60,000 → capped at 50,000
	if result.Adjustment != int64(types.MaxCalibrationBonus) {
		t.Errorf("expected %d (max bonus), got %d", types.MaxCalibrationBonus, result.Adjustment)
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

	// rate = 1,000,000, deviation = 300,000, /5 = 60,000 → capped at 50,000
	if result.Adjustment != int64(types.MaxCalibrationBonus) {
		t.Errorf("expected %d at exact min samples, got %d", types.MaxCalibrationBonus, result.Adjustment)
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
