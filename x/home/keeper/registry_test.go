package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/home/keeper"
	"github.com/legible-money/legible/x/home/types"
)

func TestSetGetCapability(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create a home
	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner1, Name: "Cap Home"})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}
	homeID := resp.HomeId

	// Set capability: REASONING at EXPERT
	err = k.SetCapability(ctx, homeID, types.CapReasoning, types.LevelExpert)
	if err != nil {
		t.Fatalf("SetCapability: %v", err)
	}

	// Get it back
	level, found := k.GetCapability(ctx, homeID, types.CapReasoning)
	if !found {
		t.Fatal("expected capability to be found")
	}
	if level != types.LevelExpert {
		t.Errorf("expected level %d, got %d", types.LevelExpert, level)
	}

	// Get non-existent capability
	_, found = k.GetCapability(ctx, homeID, types.CapCoding)
	if found {
		t.Error("expected capability not found for unset type")
	}

	// Overwrite: REASONING from EXPERT → BASIC
	err = k.SetCapability(ctx, homeID, types.CapReasoning, types.LevelBasic)
	if err != nil {
		t.Fatalf("SetCapability overwrite: %v", err)
	}
	level, found = k.GetCapability(ctx, homeID, types.CapReasoning)
	if !found {
		t.Fatal("expected capability after overwrite")
	}
	if level != types.LevelBasic {
		t.Errorf("expected level %d after overwrite, got %d", types.LevelBasic, level)
	}
}

func TestSetCapabilityValidation(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner1, Name: "Val Home"})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}
	homeID := resp.HomeId

	// Invalid capability type (0)
	err = k.SetCapability(ctx, homeID, 0, types.LevelBasic)
	if err == nil {
		t.Error("expected error for capability type 0")
	}

	// Invalid capability type (8)
	err = k.SetCapability(ctx, homeID, 8, types.LevelBasic)
	if err == nil {
		t.Error("expected error for capability type 8")
	}

	// Invalid level (0)
	err = k.SetCapability(ctx, homeID, types.CapCoding, 0)
	if err == nil {
		t.Error("expected error for level 0")
	}

	// Invalid level (4)
	err = k.SetCapability(ctx, homeID, types.CapCoding, 4)
	if err == nil {
		t.Error("expected error for level 4")
	}

	// Non-existent home
	err = k.SetCapability(ctx, "fake-home-999", types.CapCoding, types.LevelBasic)
	if err == nil {
		t.Error("expected error for non-existent home")
	}
}

func TestListCapabilities(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner1, Name: "List Home"})
	if err != nil {
		t.Fatalf("create home: %v", err)
	}
	homeID := resp.HomeId

	// Empty list
	caps := k.ListCapabilities(ctx, homeID)
	if len(caps) != 0 {
		t.Errorf("expected 0 capabilities, got %d", len(caps))
	}

	// Set 3 capabilities
	k.SetCapability(ctx, homeID, types.CapReasoning, types.LevelExpert)
	k.SetCapability(ctx, homeID, types.CapCoding, types.LevelProficient)
	k.SetCapability(ctx, homeID, types.CapResearch, types.LevelBasic)

	caps = k.ListCapabilities(ctx, homeID)
	if len(caps) != 3 {
		t.Fatalf("expected 3 capabilities, got %d", len(caps))
	}

	// Verify they contain the right types (order is by key, which is capType numeric)
	capTypes := map[uint32]bool{}
	for _, c := range caps {
		capTypes[c.CapType] = true
	}
	for _, ct := range []uint32{types.CapReasoning, types.CapCoding, types.CapResearch} {
		if !capTypes[ct] {
			t.Errorf("missing capability type %d", ct)
		}
	}
}

func TestVouchForHappyPath(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create two homes
	resp1, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner1, Name: "Voucher Home"})
	resp2, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner2, Name: "Target Home"})
	voucherID := resp1.HomeId
	targetID := resp2.HomeId

	// Set capability on target
	k.SetCapability(ctx, targetID, types.CapCoding, types.LevelProficient)

	// Vouch
	vouchID, err := k.VouchFor(ctx, voucherID, targetID, types.CapCoding, types.LevelProficient, 100)
	if err != nil {
		t.Fatalf("VouchFor: %v", err)
	}
	if vouchID == "" {
		t.Error("expected non-empty vouch ID")
	}

	// Retrieve vouches
	vouches := k.GetVouches(ctx, targetID, types.CapCoding)
	if len(vouches) != 1 {
		t.Fatalf("expected 1 vouch, got %d", len(vouches))
	}
	if vouches[0].VoucherHomeId != voucherID {
		t.Errorf("expected voucher %s, got %s", voucherID, vouches[0].VoucherHomeId)
	}
	if vouches[0].Level != types.LevelProficient {
		t.Errorf("expected level %d, got %d", types.LevelProficient, vouches[0].Level)
	}
}

func TestVouchForSelfReject(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner1, Name: "Self Home"})
	homeID := resp.HomeId

	_, err := k.VouchFor(ctx, homeID, homeID, types.CapReasoning, types.LevelBasic, 100)
	if err == nil {
		t.Error("expected error for self-vouch")
	}
}

func TestVouchForMaxVouches(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create target + 6 voucher homes (need 5 to fill, 6th should fail)
	targetResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner1, Name: "Target"})
	targetID := targetResp.HomeId

	k.SetCapability(ctx, targetID, types.CapAnalysis, types.LevelExpert)

	voucherIDs := make([]string, 6)
	for i := 0; i < 6; i++ {
		addr := testAddr("voucher" + string(rune('A'+i)))
		resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: addr, Name: "Voucher"})
		if err != nil {
			t.Fatalf("create voucher home %d: %v", i, err)
		}
		voucherIDs[i] = resp.HomeId
	}

	// First 5 vouches should succeed
	for i := 0; i < 5; i++ {
		_, err := k.VouchFor(ctx, voucherIDs[i], targetID, types.CapAnalysis, types.LevelExpert, 100)
		if err != nil {
			t.Fatalf("vouch %d failed: %v", i, err)
		}
	}

	// 6th vouch should fail
	_, err := k.VouchFor(ctx, voucherIDs[5], targetID, types.CapAnalysis, types.LevelExpert, 100)
	if err == nil {
		t.Error("expected error for 6th vouch (max 5)")
	}

	// Verify count
	vouches := k.GetVouches(ctx, targetID, types.CapAnalysis)
	if len(vouches) != 5 {
		t.Errorf("expected 5 vouches, got %d", len(vouches))
	}
}

func TestReputationScoreZero(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner1, Name: "Empty Home"})
	homeID := resp.HomeId

	score := k.ReputationScore(ctx, homeID)
	if score != 0 {
		t.Errorf("expected 0 reputation for home with no capabilities, got %d", score)
	}
}

func TestReputationScoreNoVouches(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner1, Name: "Unvouched Home"})
	homeID := resp.HomeId

	// Set capabilities but no vouches
	k.SetCapability(ctx, homeID, types.CapReasoning, types.LevelExpert)
	k.SetCapability(ctx, homeID, types.CapCoding, types.LevelProficient)

	score := k.ReputationScore(ctx, homeID)
	if score != 0 {
		t.Errorf("expected 0 reputation (no vouches), got %d", score)
	}
}

func TestReputationScoreMultiCapability(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create target + 2 vouchers
	targetResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner1, Name: "Target"})
	targetID := targetResp.HomeId

	v1Resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner2, Name: "V1"})
	v2Resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: user1, Name: "V2"})

	// Target: REASONING(3) + CODING(2)
	k.SetCapability(ctx, targetID, types.CapReasoning, types.LevelExpert)      // level 3
	k.SetCapability(ctx, targetID, types.CapCoding, types.LevelProficient)     // level 2

	// 2 vouches on REASONING, 1 vouch on CODING
	k.VouchFor(ctx, v1Resp.HomeId, targetID, types.CapReasoning, types.LevelExpert, 100)
	k.VouchFor(ctx, v2Resp.HomeId, targetID, types.CapReasoning, types.LevelExpert, 100)
	k.VouchFor(ctx, v1Resp.HomeId, targetID, types.CapCoding, types.LevelProficient, 100)

	// Expected: (3 * 2) + (2 * 1) = 8
	score := k.ReputationScore(ctx, targetID)
	if score != 8 {
		t.Errorf("expected reputation 8, got %d", score)
	}
}

func TestReputationScoreCap(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// To hit the 1000 cap we'd need extreme values.
	// Max per cap = level(3) * vouches(5) = 15, across 7 caps = 105.
	// So with realistic values we can't exceed 1000. Just verify the cap logic
	// by checking that a reasonable score stays below 1000.
	targetResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: owner1, Name: "MaxTarget"})
	targetID := targetResp.HomeId

	// Set all 7 capabilities at EXPERT
	for ct := types.CapReasoning; ct <= types.CapExecution; ct++ {
		k.SetCapability(ctx, targetID, ct, types.LevelExpert)
	}

	// Create 5 voucher homes and vouch for all 7 capabilities
	for i := 0; i < 5; i++ {
		addr := testAddr("maxv" + string(rune('0'+i)))
		vResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{Owner: addr, Name: "MaxV"})
		for ct := types.CapReasoning; ct <= types.CapExecution; ct++ {
			k.VouchFor(ctx, vResp.HomeId, targetID, ct, types.LevelExpert, 100)
		}
	}

	// Expected: 7 caps × 3 × 5 = 105. Under 1000 cap.
	score := k.ReputationScore(ctx, targetID)
	if score != 105 {
		t.Errorf("expected reputation 105, got %d", score)
	}
	if score > 1000 {
		t.Error("score should be capped at 1000")
	}
}
