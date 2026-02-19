package keeper_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

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
