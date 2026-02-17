package keeper_test

// ==================================================================================
// OpenClaw Claiming Pot Adversarial Probe Suite (OC-CP-1 through OC-CP-3)
//
// Attack surfaces tested:
// - Double claim from same account
// - Non-creator closing pot (fund theft)
// - Claim from expired pot
// ==================================================================================

import (
	"testing"

	"github.com/legible-money/legible/x/claiming_pot/keeper"
	"github.com/legible-money/legible/x/claiming_pot/types"
)

// ==================================================================================
// OC-CP-1: Double Claim Attack
//
// ATTACK: Claim from a pot, then try to claim again from the same account.
// If unguarded, a single claimer could drain the entire pot.
//
// DEFENSE: ClaimFromPot checks record.ClaimCount >= config.MaxClaimsPerAccount.
// ==================================================================================

func TestOC_CP1_DoubleClaim(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 100000000)
	bk.setModuleBalance(types.ModuleName, "ulgm", 100000000)

	srv := keeper.NewMsgServerImpl(k)

	// Create a pot
	resp, err := srv.CreatePot(ctx, &types.MsgCreatePot{
		Creator:        creator1,
		InitialFunding: "50000000",
	})
	if err != nil {
		t.Fatalf("create pot: %v", err)
	}

	// First claim — should succeed
	_, err = srv.ClaimFromPot(ctx, &types.MsgClaimFromPot{
		PotId:   resp.PotId,
		Claimer: claimer1,
	})
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}

	// ATTACK: Second claim from same account
	_, err = srv.ClaimFromPot(ctx, &types.MsgClaimFromPot{
		PotId:   resp.PotId,
		Claimer: claimer1,
	})
	if err == nil {
		t.Fatal("OC-CP-1 FAIL: double claim allowed! Same account could drain pot.")
	}

	t.Logf("OC-CP-1 PASS: double claim correctly rejected")
}

// ==================================================================================
// OC-CP-2: Non-Creator Close Attack
//
// ATTACK: Someone other than the pot creator tries to close it and steal the refund.
// If unguarded, anyone could close pots and redirect funds.
//
// DEFENSE: ClosePot checks msg.Authority == pot.Creator.
// ==================================================================================

func TestOC_CP2_NonCreatorClose(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 100000000)

	srv := keeper.NewMsgServerImpl(k)

	resp, err := srv.CreatePot(ctx, &types.MsgCreatePot{
		Creator:        creator1,
		InitialFunding: "50000000",
	})
	if err != nil {
		t.Fatalf("create pot: %v", err)
	}

	// ATTACK: Non-creator tries to close and steal refund
	_, err = srv.ClosePot(ctx, &types.MsgClosePot{
		PotId:     resp.PotId,
		Authority: claimer1, // not the creator
	})
	if err == nil {
		t.Fatal("OC-CP-2 FAIL: non-creator was able to close pot and steal funds!")
	}

	// Verify pot still active
	pot, found := k.GetPot(ctx, resp.PotId)
	if !found {
		t.Fatal("pot not found")
	}
	if pot.Status != string(types.PotStatusActive) {
		t.Fatalf("OC-CP-2 FAIL: pot status changed by non-creator: %s", pot.Status)
	}

	t.Logf("OC-CP-2 PASS: non-creator close correctly rejected")
}

// ==================================================================================
// OC-CP-3: Claim from Expired Pot
//
// ATTACK: Pot expires, but someone tries to claim from it.
// If unguarded, expired pots could continue paying out.
//
// DEFENSE: ClaimFromPot checks expiration and transitions to expired status.
// ==================================================================================

func TestOC_CP3_ClaimFromExpiredPot(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 100000000)
	bk.setModuleBalance(types.ModuleName, "ulgm", 100000000)

	srv := keeper.NewMsgServerImpl(k)

	// Create a pot with expiry at block 150
	resp, err := srv.CreatePot(ctx, &types.MsgCreatePot{
		Creator:        creator1,
		InitialFunding: "50000000",
		ExpiryHeight:   150,
	})
	if err != nil {
		t.Fatalf("create pot: %v", err)
	}

	// Advance past expiry
	expiredCtx := ctx.WithBlockHeight(200)

	// ATTACK: Claim from expired pot
	_, err = srv.ClaimFromPot(expiredCtx, &types.MsgClaimFromPot{
		PotId:   resp.PotId,
		Claimer: claimer1,
	})
	if err == nil {
		t.Fatal("OC-CP-3 FAIL: claim from expired pot allowed!")
	}

	t.Logf("OC-CP-3 PASS: claim from expired pot correctly rejected")
}
