package keeper_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/keeper"
	"github.com/legible-money/legible/x/knowledge/types"
)

// ---------- Mock PartnershipsKeeper ----------

type mockPartnershipsKeeper struct {
	partnerships map[string]*types.Partnership // addr → partnership
	distributed  []distributeCall
	failDistrib  bool
}

type distributeCall struct {
	PartnershipID string
	GrossAmount   string
	SourceType    string
}

func newMockPartnershipsKeeper() *mockPartnershipsKeeper {
	return &mockPartnershipsKeeper{
		partnerships: make(map[string]*types.Partnership),
	}
}

func (m *mockPartnershipsKeeper) addPartnership(humanAddr, agentAddr string, p *types.Partnership) {
	m.partnerships[humanAddr] = p
	m.partnerships[agentAddr] = p
}

func (m *mockPartnershipsKeeper) GetActivePartnershipForAddress(_ sdk.Context, addr string) (*types.Partnership, bool) {
	p, found := m.partnerships[addr]
	return p, found
}

func (m *mockPartnershipsKeeper) DistributeReward(
	_ sdk.Context,
	partnership *types.Partnership,
	grossAmount string,
	sourceType string,
) (humanShare, agentShare, commonPotAdd string, err error) {
	m.distributed = append(m.distributed, distributeCall{
		PartnershipID: partnership.Id,
		GrossAmount:   grossAmount,
		SourceType:    sourceType,
	})
	if m.failDistrib {
		return "", "", "", fmt.Errorf("mock distribution error")
	}

	// Simulate 22% pot, 78% split 50/50
	gross, _ := sdkmath.NewIntFromString(grossAmount)
	potShare := gross.Mul(sdkmath.NewInt(220000)).Quo(sdkmath.NewInt(1000000))
	remaining := gross.Sub(potShare)
	half := remaining.Quo(sdkmath.NewInt(2))
	otherHalf := remaining.Sub(half)

	return half.String(), otherHalf.String(), potShare.String(), nil
}

// ---------- RouteReward Tests ----------

func TestRouteReward_NoPartnership(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	// No partnerships keeper wired — direct transfer
	addr := sdk.AccAddress([]byte("test_validator_addr_"))
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(1000000)))

	err := k.RouteReward(ctx, addr, coins, "verification")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// With mockBankKeeper, this silently succeeds (direct transfer path)
}

func TestRouteReward_NoPartnershipKeeper(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	// partnershipsKeeper is nil by default — should go direct
	addr := sdk.AccAddress([]byte("test_validator_addr_"))
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(500000)))

	err := k.RouteReward(ctx, addr, coins, "citation")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRouteReward_WithPartnership(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_partner_addr__")).String()
	agentAddr := sdk.AccAddress([]byte("agent_partner_addr__")).String()

	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "partnership-1",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active",
		LockTier:      0,
		SplitHumanBps: 500000,
		SplitAgentBps: 500000,
	})
	k.SetPartnershipsKeeper(mpk)

	// Route reward for the human address
	addr, _ := sdk.AccAddressFromBech32(humanAddr)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(1000000)))

	err := k.RouteReward(ctx, addr, coins, "verification")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify partnership distribution was called
	if len(mpk.distributed) != 1 {
		t.Fatalf("expected 1 distribution call, got %d", len(mpk.distributed))
	}
	if mpk.distributed[0].PartnershipID != "partnership-1" {
		t.Errorf("expected partnership-1, got %s", mpk.distributed[0].PartnershipID)
	}
	if mpk.distributed[0].GrossAmount != "1000000" {
		t.Errorf("expected gross 1000000, got %s", mpk.distributed[0].GrossAmount)
	}
	if mpk.distributed[0].SourceType != "verification" {
		t.Errorf("expected source verification, got %s", mpk.distributed[0].SourceType)
	}
}

func TestRouteReward_WithPartnershipAgentSide(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_partner_addr__")).String()
	agentAddr := sdk.AccAddress([]byte("agent_partner_addr__")).String()

	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "partnership-2",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active",
		LockTier:      1,
		SplitHumanBps: 600000,
		SplitAgentBps: 400000,
	})
	k.SetPartnershipsKeeper(mpk)

	// Route reward for the agent address (should still find partnership)
	addr, _ := sdk.AccAddressFromBech32(agentAddr)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(2000000)))

	err := k.RouteReward(ctx, addr, coins, "citation")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mpk.distributed) != 1 {
		t.Fatalf("expected 1 distribution call, got %d", len(mpk.distributed))
	}
	if mpk.distributed[0].GrossAmount != "2000000" {
		t.Errorf("expected gross 2000000, got %s", mpk.distributed[0].GrossAmount)
	}
	if mpk.distributed[0].SourceType != "citation" {
		t.Errorf("expected source citation, got %s", mpk.distributed[0].SourceType)
	}
}

func TestRouteReward_FallbackOnError(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_partner_addr__")).String()
	agentAddr := sdk.AccAddress([]byte("agent_partner_addr__")).String()

	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "partnership-fail",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active",
		LockTier:      0,
		SplitHumanBps: 500000,
		SplitAgentBps: 500000,
	})
	mpk.failDistrib = true
	k.SetPartnershipsKeeper(mpk)

	// Route reward — distribution fails, should fall back to direct
	addr, _ := sdk.AccAddressFromBech32(humanAddr)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(1000000)))

	err := k.RouteReward(ctx, addr, coins, "verification")
	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}

	// Distribution was attempted
	if len(mpk.distributed) != 1 {
		t.Fatalf("expected 1 distribution attempt, got %d", len(mpk.distributed))
	}
}

func TestRouteReward_NoActivePartnership(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	// Empty — no partnerships registered
	k.SetPartnershipsKeeper(mpk)

	addr := sdk.AccAddress([]byte("test_no_partner_addr"))
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(500000)))

	err := k.RouteReward(ctx, addr, coins, "falsification")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// No distribution calls — went direct
	if len(mpk.distributed) != 0 {
		t.Errorf("expected 0 distribution calls, got %d", len(mpk.distributed))
	}
}

func TestRouteReward_FrozenPartnership(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_frozen_addr___")).String()
	agentAddr := sdk.AccAddress([]byte("agent_frozen_addr___")).String()

	// Frozen (suspended) partnership — per spec, rewards still split
	// (freeze blocks spending, not earning)
	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "partnership-frozen",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active", // Still "active" status — freeze is tracked separately
		LockTier:      2,
		SplitHumanBps: 500000,
		SplitAgentBps: 500000,
	})
	k.SetPartnershipsKeeper(mpk)

	addr, _ := sdk.AccAddressFromBech32(humanAddr)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(3000000)))

	err := k.RouteReward(ctx, addr, coins, "verification")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Partnership distribution was called (freeze doesn't block earning)
	if len(mpk.distributed) != 1 {
		t.Fatalf("expected 1 distribution call, got %d", len(mpk.distributed))
	}
	if mpk.distributed[0].GrossAmount != "3000000" {
		t.Errorf("expected gross 3000000, got %s", mpk.distributed[0].GrossAmount)
	}
}

func TestRouteReward_FalsificationSource(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	mpk := newMockPartnershipsKeeper()
	humanAddr := sdk.AccAddress([]byte("human_falsif_addr___")).String()
	agentAddr := sdk.AccAddress([]byte("agent_falsif_addr___")).String()

	mpk.addPartnership(humanAddr, agentAddr, &types.Partnership{
		Id:            "partnership-falsif",
		HumanAddr:     humanAddr,
		AgentAddr:     agentAddr,
		Status:        "active",
		LockTier:      0,
		SplitHumanBps: 500000,
		SplitAgentBps: 500000,
	})
	k.SetPartnershipsKeeper(mpk)

	addr, _ := sdk.AccAddressFromBech32(agentAddr)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(750000)))

	err := k.RouteReward(ctx, addr, coins, "falsification")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mpk.distributed) != 1 {
		t.Fatalf("expected 1 distribution call, got %d", len(mpk.distributed))
	}
	if mpk.distributed[0].SourceType != "falsification" {
		t.Errorf("expected source falsification, got %s", mpk.distributed[0].SourceType)
	}
}

// ---------- GetResearchTaxBps Tests ----------

func TestGetResearchTaxBps_DefaultFallback(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	// vestingRewardsKeeper is nil by default — should return DefaultResearchTaxBps
	bps := k.GetResearchTaxBps(ctx)
	if bps != keeper.DefaultResearchTaxBps {
		t.Errorf("expected default %d, got %d", keeper.DefaultResearchTaxBps, bps)
	}
	if bps != 33300 {
		t.Errorf("expected 33300 (3.33%%), got %d", bps)
	}
}

func TestGetResearchTaxBps_FromVestingKeeper(t *testing.T) {
	k, ctx, _, _, mvk := setupKeeperWithTracking(t)
	// The mock returns 33300 (3.33%)
	_ = mvk // verify it's wired
	bps := k.GetResearchTaxBps(ctx)
	if bps != 33300 {
		t.Errorf("expected 33300 from mock vesting keeper, got %d", bps)
	}
}

func TestResearchTaxRateConsistency(t *testing.T) {
	// Verify the constant matches the canonical vesting_rewards default
	if keeper.DefaultResearchTaxBps != 33300 {
		t.Errorf("DefaultResearchTaxBps should be 33300, got %d", keeper.DefaultResearchTaxBps)
	}
}

// ---------- Verification: No Double Tax ----------

func TestNoDoubleTaxOnVerification(t *testing.T) {
	k, ctx, sk, tbk, _ := setupKeeperWithTracking(t)

	// Set up a validator
	validatorAddr := sdk.AccAddress([]byte("test_validator_addr_")).String()
	sk.addValidator(validatorAddr, 10_000_000, 2) // Bonded: 1.0x multiplier

	// Create result with reward
	result := &types.VerificationResult{
		Verdict:    "accepted",
		Confidence: 800000,
		Rewards:    map[string]uint64{validatorAddr: 1_000_000},
		Slashes:    make(map[string]uint64),
	}

	err := k.ApplyRewardsAndSlashes(ctx, result, false)
	if err != nil {
		t.Fatalf("ApplyRewardsAndSlashes failed: %v", err)
	}

	// Verify the FULL reward amount was sent to the validator (not reduced by tax).
	// Before this fix, 7% was deducted as double-tax. Now the full amount goes through.
	valAccAddr, _ := sdk.AccAddressFromBech32(validatorAddr)
	received := tbk.sentToAccount[valAccAddr.String()]
	if received.IsZero() {
		t.Fatal("expected reward sent to validator account, got nothing")
	}
	receivedUlgm := received.AmountOf("ulgm")
	if !receivedUlgm.Equal(sdkmath.NewInt(1_000_000)) {
		t.Errorf("expected full 1,000,000 ulgm reward (no double tax), got %s", receivedUlgm.String())
	}
}
