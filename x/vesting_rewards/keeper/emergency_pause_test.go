package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

// ---------- Mock Emergency Keeper ----------

type mockEmergencyKeeper struct {
	halted bool
}

func (m *mockEmergencyKeeper) IsHalted(_ sdk.Context) bool {
	return m.halted
}

// newVestingCtx returns a context at a high block height so vesting has released tokens.
func newVestingCtx(ctx sdk.Context) sdk.Context {
	return ctx.WithBlockHeader(cmtproto.Header{Height: 500000})
}

// makeTestSchedule creates a VestingSchedule that is well past its cliff.
// AcceptedAtBlock=10, CliffEndsAtBlock=100, so at height 500000 it's far past cliff.
func makeTestSchedule() *types.VestingSchedule {
	return &types.VestingSchedule{
		Id:               "test-vesting-1",
		Recipient:        "legible1abc",
		TotalAmount:      "1000000000000000000000",
		ReleasedAmount:   "0",
		Category:         string(types.CategoryPeerReviewed),
		Status:           string(types.VestingStatusActive),
		AcceptedAtBlock:  10,
		CliffEndsAtBlock: 100,
	}
}

// ---------- Tests ----------

func TestCalculateReleasedAmount_EmergencyHalted(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 1}
	k, ctx := setupKeeperWithBank(t, bk, sk)
	ctx = newVestingCtx(ctx)

	ek := &mockEmergencyKeeper{halted: true}
	k.SetEmergencyKeeper(ek)

	result := k.CalculateReleasedAmount(ctx, makeTestSchedule())
	if result != "0" {
		t.Errorf("expected 0 during emergency halt, got %s", result)
	}
}

func TestCalculateReleasedAmount_NotHalted(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 1}
	k, ctx := setupKeeperWithBank(t, bk, sk)
	ctx = newVestingCtx(ctx)

	ek := &mockEmergencyKeeper{halted: false}
	k.SetEmergencyKeeper(ek)

	result := k.CalculateReleasedAmount(ctx, makeTestSchedule())
	if result == "0" {
		t.Error("expected non-zero release when not halted, got 0")
	}
}

func TestCalculateReleasedAmount_NilEmergencyKeeper(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 1}
	k, ctx := setupKeeperWithBank(t, bk, sk)
	ctx = newVestingCtx(ctx)

	// Do NOT set emergency keeper — it stays nil.
	result := k.CalculateReleasedAmount(ctx, makeTestSchedule())
	if result == "0" {
		t.Error("expected non-zero release when emergency keeper is nil, got 0")
	}
}

func TestCalculateReleasedAmount_HaltToggle(t *testing.T) {
	bk := newMockBankKeeper()
	sk := &mockStakingKeeper{activeCount: 1}
	k, ctx := setupKeeperWithBank(t, bk, sk)
	ctx = newVestingCtx(ctx)

	ek := &mockEmergencyKeeper{halted: true}
	k.SetEmergencyKeeper(ek)

	schedule := makeTestSchedule()

	// Halted: should return 0
	result := k.CalculateReleasedAmount(ctx, schedule)
	if result != "0" {
		t.Errorf("expected 0 during halt, got %s", result)
	}

	// Resume: should return non-zero
	ek.halted = false
	result = k.CalculateReleasedAmount(ctx, schedule)
	if result == "0" {
		t.Error("expected non-zero after halt cleared, got 0")
	}
}
