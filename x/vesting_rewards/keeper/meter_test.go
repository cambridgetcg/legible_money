package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
)

// TestIncrementMinted verifies that successive calls accumulate correctly.
func TestIncrementMinted(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.IncrementMinted(ctx, sdkmath.NewInt(100))
	k.IncrementMinted(ctx, sdkmath.NewInt(250))

	totals := k.GetMeterTotals(ctx)
	if totals.TotalMinted != "350" {
		t.Fatalf("expected TotalMinted=350, got %s", totals.TotalMinted)
	}
}

// TestIncrementBurned verifies burned counter.
func TestIncrementBurned(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.IncrementBurned(ctx, sdkmath.NewInt(42))
	totals := k.GetMeterTotals(ctx)
	if totals.TotalBurned != "42" {
		t.Fatalf("expected TotalBurned=42, got %s", totals.TotalBurned)
	}
}

// TestGetMeterTotals verifies all counters are returned together.
func TestGetMeterTotals(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.IncrementMinted(ctx, sdkmath.NewInt(1000))
	k.IncrementBurned(ctx, sdkmath.NewInt(200))
	k.IncrementLocked(ctx, sdkmath.NewInt(300))
	k.IncrementRouted(ctx, sdkmath.NewInt(400))
	k.IncrementFees(ctx, sdkmath.NewInt(50))

	totals := k.GetMeterTotals(ctx)

	if totals.TotalMinted != "1000" {
		t.Errorf("TotalMinted: want 1000, got %s", totals.TotalMinted)
	}
	if totals.TotalBurned != "200" {
		t.Errorf("TotalBurned: want 200, got %s", totals.TotalBurned)
	}
	if totals.TotalLocked != "300" {
		t.Errorf("TotalLocked: want 300, got %s", totals.TotalLocked)
	}
	if totals.TotalRouted != "400" {
		t.Errorf("TotalRouted: want 400, got %s", totals.TotalRouted)
	}
	if totals.TotalFees != "50" {
		t.Errorf("TotalFees: want 50, got %s", totals.TotalFees)
	}
	if totals.BlockHeight != 1000 {
		t.Errorf("BlockHeight: want 1000, got %d", totals.BlockHeight)
	}
}

// TestDistributeBlockReward_IncrementsMeter verifies that DistributeBlockReward
// records minted tokens in the meter. The test keeper has no bank keeper so
// coin transfers are skipped, but MintWithCap falls back to totalMinted tracking
// and the meter increment still fires.
func TestDistributeBlockReward_IncrementsMeter(t *testing.T) {
	k, ctx := setupKeeper(t)

	// DistributeBlockReward with hasTransactions=true so it is not short-circuited.
	producer := "lgm1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq5dwmac"
	_, err := k.DistributeBlockReward(ctx, producer, 10, true)
	if err != nil {
		t.Fatalf("DistributeBlockReward failed: %v", err)
	}

	totals := k.GetMeterTotals(ctx)
	if totals.TotalMinted == "0" || totals.TotalMinted == "" {
		t.Fatalf("expected TotalMinted > 0 after DistributeBlockReward, got %q", totals.TotalMinted)
	}
	t.Logf("Meter after block reward: minted=%s routed=%s", totals.TotalMinted, totals.TotalRouted)
}

// TestZeroAmountNoOp verifies that zero or negative amounts are ignored.
func TestZeroAmountNoOp(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.IncrementMinted(ctx, sdkmath.ZeroInt())
	k.IncrementMinted(ctx, sdkmath.NewInt(-5))

	totals := k.GetMeterTotals(ctx)
	if totals.TotalMinted != "0" {
		t.Fatalf("expected TotalMinted=0 for zero/negative increments, got %s", totals.TotalMinted)
	}
}
