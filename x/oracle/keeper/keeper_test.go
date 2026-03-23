package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/oracle/keeper"
	"github.com/legible-money/legible/x/oracle/types"
)

// ── Test Helpers ─────────────────────────────────────────────────────────────

func setupTest(t *testing.T) (sdk.Context, keeper.Keeper) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := sdk.AccAddress([]byte("oracle-authority----")).String()
	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), &mockBankKeeper{}, authority)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())
	return ctx, k
}

type mockBankKeeper struct{}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	return nil
}
func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}

func testAddr(i int) string {
	return sdk.AccAddress([]byte(fmt.Sprintf("oracle-test-%010d", i))).String()
}

// ── Prediction Tests ──────────────────────────────────────────────────────────

func TestSubmitPrediction_Valid(t *testing.T) {
	ctx, k := setupTest(t)
	height := uint64(ctx.BlockHeight())
	msg := types.MsgSubmitPrediction{
		Submitter:          testAddr(1),
		Claim:              "The S&P 500 will be above 5500 by end of 2026",
		Domain:             "economics.regime",
		Probability:        700_000,
		StakeAmount:        1_000_000,
		ResolutionType:     string(types.ResolutionOracle),
		ResolutionRef:      "oracle:spx_price",
		ResolutionCriteria: "SPX > 5500 on 2026-12-31",
		ResolutionBlock:    height + 50_000,
	}
	p, err := k.SubmitPrediction(ctx, msg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if p.Status != types.PredictionActive {
		t.Errorf("expected active, got %s", p.Status)
	}
	t.Logf("PASS: prediction submitted: %s", p.PredictionId)
}

func TestSubmitPrediction_ZeroProbForbidden(t *testing.T) {
	ctx, k := setupTest(t)
	height := uint64(ctx.BlockHeight())
	msg := types.MsgSubmitPrediction{
		Submitter: testAddr(1), Probability: 0,
		StakeAmount: 1_000_000, ResolutionBlock: height + 50_000,
	}
	_, err := k.SubmitPrediction(ctx, msg)
	if err == nil {
		t.Error("expected error for prob=0")
	}
	t.Logf("PASS: prob=0 rejected: %v", err)
}

func TestSubmitPrediction_CertaintyForbidden(t *testing.T) {
	ctx, k := setupTest(t)
	height := uint64(ctx.BlockHeight())
	msg := types.MsgSubmitPrediction{
		Submitter: testAddr(1), Probability: 1_000_000,
		StakeAmount: 1_000_000, ResolutionBlock: height + 50_000,
	}
	_, err := k.SubmitPrediction(ctx, msg)
	if err == nil {
		t.Error("expected error for prob=1M (certainty)")
	}
	t.Logf("PASS: certainty rejected: %v", err)
}

func TestResolvePrediction_Correct(t *testing.T) {
	ctx, k := setupTest(t)
	height := uint64(ctx.BlockHeight())
	msg := types.MsgSubmitPrediction{
		Submitter: testAddr(1), Claim: "Inflation below 3%",
		Domain: "economics.regime", Probability: 700_000,
		StakeAmount: 1_000_000, ResolutionBlock: height + 50_000,
	}
	p, _ := k.SubmitPrediction(ctx, msg)
	ctx = ctx.WithBlockHeight(int64(height + 30_000))
	err := k.ResolvePrediction(ctx, p.PredictionId, true)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	resolved, _ := k.GetPrediction(ctx, p.PredictionId)
	if resolved.Status != types.PredictionResolved {
		t.Errorf("expected resolved, got %s", resolved.Status)
	}
	if resolved.BrierScore == 0 {
		t.Error("expected brier score")
	}
	t.Logf("PASS: resolved, brier=%d bps (lower=better, perfect=0)", resolved.BrierScore)
}

func TestResolvePrediction_OverconfidentPenalised(t *testing.T) {
	ctx, k := setupTest(t)
	height := uint64(ctx.BlockHeight())
	// 90% confident but WRONG
	msg := types.MsgSubmitPrediction{
		Submitter: testAddr(2), Claim: "Market rises",
		Domain: "economics.regime", Probability: 900_000,
		StakeAmount: 1_000_000, ResolutionBlock: height + 50_000,
	}
	p, _ := k.SubmitPrediction(ctx, msg)
	ctx = ctx.WithBlockHeight(int64(height + 30_000))
	_ = k.ResolvePrediction(ctx, p.PredictionId, false)
	resolved, _ := k.GetPrediction(ctx, p.PredictionId)
	if resolved.BrierScore == 0 {
		t.Error("expected non-zero brier for overconfident wrong prediction")
	}
	t.Logf("PASS: overconfident wrong prediction penalised: score=%d", resolved.BrierScore)
}

// ── Scoring Formula Tests ─────────────────────────────────────────────────────

func TestBrierWeights(t *testing.T) {
	tests := []struct {
		days uint64; wantMin, wantMax uint64; label string
	}{
		{1, 1000, 1999, "1 day ≈ log2(2)*1000=1000"},
		{7, 3000, 3999, "7 days = log2(8)*1000=3000"},
		{30, 4000, 4999, "30 days ≈ log2(31)*1000≈4954"},
		{365, 8000, 9000, "365 days ≈ log2(366)*1000≈8517"},
	}
	for _, tt := range tests {
		w := types.ComputeBrierWeight(tt.days)
		if w < tt.wantMin || w > tt.wantMax {
			t.Errorf("%s: got %d, want [%d,%d]", tt.label, w, tt.wantMin, tt.wantMax)
		} else {
			t.Logf("PASS: %s: %d", tt.label, w)
		}
	}
}

func TestHorizonClassification(t *testing.T) {
	tests := []struct {
		days uint64; want types.HorizonTier
	}{
		{1, types.HorizonFlash}, {7, types.HorizonFlash},
		{8, types.HorizonTactical}, {90, types.HorizonTactical},
		{91, types.HorizonStrategic}, {365, types.HorizonStrategic},
		{366, types.HorizonEpochal},
	}
	for _, tt := range tests {
		if got := types.ClassifyHorizonDays(tt.days); got != tt.want {
			t.Errorf("ClassifyHorizon(%d): got %s, want %s", tt.days, got, tt.want)
		}
	}
	t.Log("PASS: all horizon classifications correct")
}

func TestRecencyWeights(t *testing.T) {
	tests := []struct {
		months uint64; wantMin, wantMax uint64
	}{
		{0, 1_000_000, 1_000_000},
		{6, 700_000, 780_000},
		{12, 500_000, 600_000},
		{24, 200_000, 350_000},
	}
	for _, tt := range tests {
		w := types.RecencyWeightBps(tt.months)
		if w < tt.wantMin || w > tt.wantMax {
			t.Errorf("RecencyWeightBps(%d): got %d, want [%d,%d]", tt.months, w, tt.wantMin, tt.wantMax)
		} else {
			t.Logf("PASS: %d months → %d bps (%.3f)", tt.months, w, float64(w)/1_000_000)
		}
	}
}
