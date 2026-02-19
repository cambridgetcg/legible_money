package keeper_test

import (
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/partnerships/keeper"
	"github.com/legible-money/legible/x/partnerships/types"
)

// ---------- Mock HomeKeeper ----------

type mockHomeKeeper struct {
	homesByOwner     map[string][]string          // ownerAddr → homeIds
	guardianAddrs    map[string]string            // homeId → guardianAddr
	partnershipLinks map[string]string            // homeId → partnershipId
	homeStatuses     map[string]string            // homeId → status
	partnershipHomes map[string]string            // partnershipId → homeId (reverse)
}

func newMockHomeKeeper() *mockHomeKeeper {
	return &mockHomeKeeper{
		homesByOwner:     make(map[string][]string),
		guardianAddrs:    make(map[string]string),
		partnershipLinks: make(map[string]string),
		homeStatuses:     make(map[string]string),
		partnershipHomes: make(map[string]string),
	}
}

func (m *mockHomeKeeper) GetHomesByOwner(_ sdk.Context, ownerAddr string) []string {
	return m.homesByOwner[ownerAddr]
}

func (m *mockHomeKeeper) GetHomeGuardianAddress(_ sdk.Context, homeId string) string {
	return m.guardianAddrs[homeId]
}

func (m *mockHomeKeeper) LinkHomeToPartnership(_ sdk.Context, homeId string, partnershipId string) {
	m.partnershipLinks[homeId] = partnershipId
	m.partnershipHomes[partnershipId] = homeId
}

func (m *mockHomeKeeper) SetHomeStatus(_ sdk.Context, homeId string, status string) {
	m.homeStatuses[homeId] = status
}

func (m *mockHomeKeeper) GetHomeByPartnership(_ sdk.Context, partnershipId string) (string, bool) {
	homeId, ok := m.partnershipHomes[partnershipId]
	return homeId, ok
}

// ---------- Tests ----------

func TestHomePartnershipAutoLink(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	human := testAddr(100)
	agent := testAddr(200)

	// Set up mock home keeper: agent has a home with human as guardian
	mhk := newMockHomeKeeper()
	mhk.homesByOwner[agent] = []string{"home-1"}
	mhk.guardianAddrs["home-1"] = human
	mhk.homeStatuses["home-1"] = "active"
	k.SetHomeKeeper(mhk)

	// Initiate partnership: human → agent
	msgServer := keeper.NewMsgServerImpl(k)
	initResp, err := msgServer.InitiatePartnership(ctx, &types.MsgInitiatePartnership{
		Initiator: human,
		Partner:   agent,
	})
	if err != nil {
		t.Fatalf("initiate partnership: %v", err)
	}

	// Accept partnership as agent
	_, err = msgServer.AcceptPartnership(ctx, &types.MsgAcceptPartnership{
		PartnershipId: initResp.PartnershipId,
		Acceptor:      agent,
	})
	if err != nil {
		t.Fatalf("accept partnership: %v", err)
	}

	// Verify home was auto-linked
	linkedPartnership, ok := mhk.partnershipLinks["home-1"]
	if !ok {
		t.Fatal("home-1 was not auto-linked to partnership")
	}
	if linkedPartnership != initResp.PartnershipId {
		t.Errorf("expected partnership %s, got %s", initResp.PartnershipId, linkedPartnership)
	}
}

func TestHomePartnershipAutoLink_NoGuardianMatch(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	human := testAddr(100)
	agent := testAddr(200)
	otherGuardian := testAddr(300)

	// Set up mock home keeper: agent has a home but guardian is someone else
	mhk := newMockHomeKeeper()
	mhk.homesByOwner[agent] = []string{"home-2"}
	mhk.guardianAddrs["home-2"] = otherGuardian
	k.SetHomeKeeper(mhk)

	msgServer := keeper.NewMsgServerImpl(k)
	initResp, err := msgServer.InitiatePartnership(ctx, &types.MsgInitiatePartnership{
		Initiator: human,
		Partner:   agent,
	})
	if err != nil {
		t.Fatalf("initiate partnership: %v", err)
	}

	_, err = msgServer.AcceptPartnership(ctx, &types.MsgAcceptPartnership{
		PartnershipId: initResp.PartnershipId,
		Acceptor:      agent,
	})
	if err != nil {
		t.Fatalf("accept partnership: %v", err)
	}

	// Home should NOT be linked (guardian doesn't match)
	if _, ok := mhk.partnershipLinks["home-2"]; ok {
		t.Error("home-2 should not be linked — guardian doesn't match human partner")
	}
}

func TestHomePartnershipAutoLink_NoHome(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	human := testAddr(100)
	agent := testAddr(200)

	// Agent has no homes
	mhk := newMockHomeKeeper()
	k.SetHomeKeeper(mhk)

	msgServer := keeper.NewMsgServerImpl(k)
	initResp, err := msgServer.InitiatePartnership(ctx, &types.MsgInitiatePartnership{
		Initiator: human,
		Partner:   agent,
	})
	if err != nil {
		t.Fatalf("initiate partnership: %v", err)
	}

	_, err = msgServer.AcceptPartnership(ctx, &types.MsgAcceptPartnership{
		PartnershipId: initResp.PartnershipId,
		Acceptor:      agent,
	})
	if err != nil {
		t.Fatalf("accept partnership: %v", err)
	}

	// No links should exist
	if len(mhk.partnershipLinks) > 0 {
		t.Error("no links should exist when agent has no homes")
	}
}

func TestHomeFreezeSync(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	human := testAddr(100)
	agent := testAddr(200)

	// Set up mock home keeper with auto-linked home
	mhk := newMockHomeKeeper()
	mhk.homesByOwner[agent] = []string{"home-3"}
	mhk.guardianAddrs["home-3"] = human
	mhk.homeStatuses["home-3"] = "active"
	mhk.partnershipHomes["partnership-1"] = "home-3"
	k.SetHomeKeeper(mhk)

	// Create an active partnership
	p := &types.Partnership{
		Id:               "partnership-1",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		CommonPotBalance: "1000000",
		CooperationScore: 5000,
	}
	k.SetPartnership(ctx, p)

	// Freeze the partnership
	_, err := k.HandleSafetyFreeze(ctx, "partnership-1", human)
	if err != nil {
		t.Fatalf("handle safety freeze: %v", err)
	}

	// Verify home status was set to "guarded"
	if mhk.homeStatuses["home-3"] != "guarded" {
		t.Errorf("expected home status 'guarded', got '%s'", mhk.homeStatuses["home-3"])
	}
}

func TestHomeFreezeSync_LiftRestoresActive(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	human := testAddr(100)
	agent := testAddr(200)

	// Set up mock home keeper with linked home
	mhk := newMockHomeKeeper()
	mhk.homeStatuses["home-4"] = "guarded"
	mhk.partnershipHomes["partnership-2"] = "home-4"
	k.SetHomeKeeper(mhk)

	params := k.GetParams(ctx)

	// Create a suspended partnership with an active freeze
	p := &types.Partnership{
		Id:               "partnership-2",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusSuspended,
		CommonPotBalance: "1000000",
		CooperationScore: 5000,
	}
	k.SetPartnership(ctx, p)

	// Create a freeze that will expire at current block
	freezeExpiresAt := uint64(ctx.BlockHeight()) // expires now
	sf := &types.SafetyFreeze{
		PartnershipId:        "partnership-2",
		FrozenBy:             human,
		FrozenAt:             freezeExpiresAt - params.SafetyFreezeDurationBlocks,
		ExpiresAt:            freezeExpiresAt,
		FreezeCountThisEpoch: 1,
	}
	k.SetSafetyFreeze(ctx, sf)

	// Advance to a block where the freeze has expired
	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: int64(freezeExpiresAt) + 1})

	// Lift expired freezes
	k.LiftExpiredFreezes(ctx)

	// Verify home status was restored to "active"
	if mhk.homeStatuses["home-4"] != "active" {
		t.Errorf("expected home status 'active' after freeze lift, got '%s'", mhk.homeStatuses["home-4"])
	}
}
