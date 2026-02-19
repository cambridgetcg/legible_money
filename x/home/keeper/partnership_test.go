package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/keeper"
	"github.com/legible-money/legible/x/home/types"
)

// ---------- Mock PartnershipKeeper ----------

type mockPartnershipKeeper struct {
	partnerships map[string]*types.PartnershipInfo
}

func newMockPartnershipKeeper() *mockPartnershipKeeper {
	return &mockPartnershipKeeper{
		partnerships: make(map[string]*types.PartnershipInfo),
	}
}

func (m *mockPartnershipKeeper) GetPartnership(_ sdk.Context, id string) (*types.PartnershipInfo, bool) {
	p, ok := m.partnerships[id]
	return p, ok
}

func (m *mockPartnershipKeeper) CalculateAmountTier(amount string, potBalance string) string {
	// Simplified tier calculation for tests
	// Real implementation uses big.Int BPS arithmetic
	if potBalance == "0" || potBalance == "" {
		return "major"
	}
	// Parse amounts as integers for simple ratio check
	var amtVal, potVal int64
	for _, c := range amount {
		if c >= '0' && c <= '9' {
			amtVal = amtVal*10 + int64(c-'0')
		}
	}
	for _, c := range potBalance {
		if c >= '0' && c <= '9' {
			potVal = potVal*10 + int64(c-'0')
		}
	}
	if potVal == 0 {
		return "major"
	}
	pctBps := amtVal * 1_000_000 / potVal
	switch {
	case pctBps < 10000:
		return "micro"
	case pctBps < 50000:
		return "small"
	case pctBps < 220000:
		return "medium"
	case pctBps < 550000:
		return "large"
	default:
		return "major"
	}
}

// ---------- Tests ----------

func TestHomeGuardianPartnerConsistency(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set up mock partnership keeper BEFORE creating msgServer
	// (msgServer embeds Keeper by value, so the keeper must be configured first)
	mpk := newMockPartnershipKeeper()
	mpk.partnerships["partnership-1"] = &types.PartnershipInfo{
		Id:               "partnership-1",
		HumanAddr:        owner2,
		AgentAddr:        owner1,
		Status:           "active",
		CommonPotBalance: "1000000",
	}
	k.SetPartnershipKeeper(mpk)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create a home for an agent
	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Agent Home",
	})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}

	// Link the home to a partnership
	k.LinkHomeToPartnership(ctx, resp.HomeId, "partnership-1")

	// Setting guardian_address to the human partner should succeed
	_, err = msgServer.ConfigureGuardian(ctx, &types.MsgConfigureGuardian{
		Owner:           owner1,
		HomeId:          resp.HomeId,
		GuardianAddress: owner2,
	})
	if err != nil {
		t.Fatalf("expected guardian address matching partner to succeed: %v", err)
	}

	// Verify guardian_address was stored
	home, found := k.GetHome(ctx, resp.HomeId)
	if !found {
		t.Fatal("home not found")
	}
	if home.Guardian.GuardianAddress != owner2 {
		t.Errorf("expected guardian address %s, got %s", owner2, home.Guardian.GuardianAddress)
	}

	// Setting guardian_address to a different address should fail
	wrongAddr := testAddr("wrong001")
	_, err = msgServer.ConfigureGuardian(ctx, &types.MsgConfigureGuardian{
		Owner:           owner1,
		HomeId:          resp.HomeId,
		GuardianAddress: wrongAddr,
	})
	if err == nil {
		t.Fatal("expected error when guardian address doesn't match partner")
	}
	if !strings.Contains(err.Error(), "guardian must be partnership partner") {
		t.Errorf("expected partner consistency error, got: %v", err)
	}
}

func TestHomeGuardianPartnerConsistency_NoPartnership(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create a home with no partnership link
	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Unlinked Home",
	})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}

	// Setting guardian_address without a partnership link should succeed (no constraint)
	anyAddr := testAddr("anyone01")
	_, err = msgServer.ConfigureGuardian(ctx, &types.MsgConfigureGuardian{
		Owner:           owner1,
		HomeId:          resp.HomeId,
		GuardianAddress: anyAddr,
	})
	if err != nil {
		t.Fatalf("expected no error without partnership link: %v", err)
	}
}

func TestHomeSpendingLimitDeliberation(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set up mock partnership keeper BEFORE creating msgServer
	mpk := newMockPartnershipKeeper()
	mpk.partnerships["partnership-2"] = &types.PartnershipInfo{
		Id:               "partnership-2",
		HumanAddr:        owner2,
		AgentAddr:        owner1,
		Status:           "active",
		CommonPotBalance: "1000000",
	}
	k.SetPartnershipKeeper(mpk)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create home
	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Agent Home",
	})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}

	// Set a generous spending limit
	_, err = msgServer.SetSpendingLimit(ctx, &types.MsgSetSpendingLimit{
		Owner:        owner1,
		HomeId:       resp.HomeId,
		KeyType:      "operator",
		MaxAmount:    "999999999",
		PeriodBlocks: 10000,
	})
	if err != nil {
		t.Fatalf("set spending limit: %v", err)
	}

	// Link to partnership with pot balance of 1000000
	k.LinkHomeToPartnership(ctx, resp.HomeId, "partnership-2")

	// Micro spend (<1% of pot = <10000) should pass
	err = k.CheckAndDebitSpendingLimit(ctx, resp.HomeId, "operator", "5000")
	if err != nil {
		t.Fatalf("micro spend should pass: %v", err)
	}

	// Small spend (1-5% of pot = 10000-50000) should pass
	err = k.CheckAndDebitSpendingLimit(ctx, resp.HomeId, "operator", "30000")
	if err != nil {
		t.Fatalf("small spend should pass: %v", err)
	}

	// Large spend (22-55% of pot = 220000-550000) should be blocked
	err = k.CheckAndDebitSpendingLimit(ctx, resp.HomeId, "operator", "300000")
	if err == nil {
		t.Fatal("expected large spend to be blocked for deliberation")
	}
	if !strings.Contains(err.Error(), "requires partnership deliberation") {
		t.Errorf("expected deliberation error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "large") {
		t.Errorf("expected tier 'large' in error, got: %v", err)
	}

	// Major spend (>55% of pot = >550000) should be blocked
	err = k.CheckAndDebitSpendingLimit(ctx, resp.HomeId, "operator", "700000")
	if err == nil {
		t.Fatal("expected major spend to be blocked for deliberation")
	}
	if !strings.Contains(err.Error(), "major") {
		t.Errorf("expected tier 'major' in error, got: %v", err)
	}
}

func TestHomeSpendingLimitDeliberation_NoPartnership(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create home with no partnership link
	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Solo Agent Home",
	})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}

	// Set a spending limit
	_, err = msgServer.SetSpendingLimit(ctx, &types.MsgSetSpendingLimit{
		Owner:        owner1,
		HomeId:       resp.HomeId,
		KeyType:      "operator",
		MaxAmount:    "999999999",
		PeriodBlocks: 10000,
	})
	if err != nil {
		t.Fatalf("set spending limit: %v", err)
	}

	// Without partnership link, any amount should pass (subject to spending limit only)
	err = k.CheckAndDebitSpendingLimit(ctx, resp.HomeId, "operator", "700000")
	if err != nil {
		t.Fatalf("expected no deliberation check without partnership: %v", err)
	}
}

func TestHomePartnershipLinkAndReverseLookup(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create home
	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Agent Home",
	})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}

	// Link to partnership
	k.LinkHomeToPartnership(ctx, resp.HomeId, "partnership-3")

	// Verify partnership_id is stored on the home
	home, found := k.GetHome(ctx, resp.HomeId)
	if !found {
		t.Fatal("home not found")
	}
	if home.PartnershipId != "partnership-3" {
		t.Errorf("expected partnership_id 'partnership-3', got '%s'", home.PartnershipId)
	}

	// Verify reverse lookup works
	homeId, found := k.GetHomeByPartnership(ctx, "partnership-3")
	if !found {
		t.Fatal("reverse lookup failed: home not found by partnership")
	}
	if homeId != resp.HomeId {
		t.Errorf("expected home ID %s, got %s", resp.HomeId, homeId)
	}

	// Non-existent partnership returns false
	_, found = k.GetHomeByPartnership(ctx, "nonexistent")
	if found {
		t.Error("expected false for non-existent partnership")
	}
}
