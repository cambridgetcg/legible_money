package keeper_test

// ==================================================================================
// OpenClaw Compute Pool Adversarial Probe Suite (OC-COMP-1 through OC-COMP-3)
//
// Attack surfaces tested:
// - Duplicate provider registration
// - Heartbeat from non-registered provider
// - UpdateParams authorization bypass
// ==================================================================================

import (
	"testing"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/compute_pool/keeper"
	"github.com/legible-money/legible/x/compute_pool/types"
)

// ==================================================================================
// OC-COMP-1: Duplicate Provider Registration
//
// ATTACK: Register as a provider, then register again to reset stats or double-stake.
// If unguarded, a provider could get back into pristine state after slashing.
//
// DEFENSE: RegisterProvider checks if provider already exists.
// ==================================================================================

func TestOC_COMP1_DuplicateProviderRegistration(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)
	addr := providerAddr1.String()
	bk.setBalance(addr, "ulgm", sdkmath.NewInt(500000000))

	srv := keeper.NewMsgServerImpl(k)

	// First registration
	_, err := srv.RegisterComputeProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender:      addr,
		ServiceType: "ai_inference",
		Endpoint:    "https://api.example.com",
		PricePerCu:  "1000",
		StakeAmount: "100000000",
	})
	if err != nil {
		t.Fatalf("first registration: %v", err)
	}

	// ATTACK: Second registration (duplicate)
	_, err = srv.RegisterComputeProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender:      addr,
		ServiceType: "gpu_compute",
		Endpoint:    "https://api2.example.com",
		PricePerCu:  "2000",
		StakeAmount: "100000000",
	})
	if err == nil {
		t.Fatal("OC-COMP-1 FAIL: duplicate provider registration allowed!")
	}

	t.Logf("OC-COMP-1 PASS: duplicate provider registration correctly rejected")
}

// ==================================================================================
// OC-COMP-2: Heartbeat from Unregistered Provider
//
// ATTACK: Non-registered address sends heartbeat to appear as active provider.
// If unguarded, ghost providers could appear in the active pool.
//
// DEFENSE: ProcessHeartbeat checks if provider exists.
// ==================================================================================

func TestOC_COMP2_UnregisteredHeartbeat(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)

	// ATTACK: Unregistered address sends heartbeat
	ghostAddr := sdk.AccAddress([]byte("ghost_provider______")).String()
	_, err := srv.ProviderHeartbeat(ctx, &types.MsgProviderHeartbeat{
		Sender: ghostAddr,
	})
	if err == nil {
		t.Fatal("OC-COMP-2 FAIL: heartbeat from unregistered provider accepted!")
	}

	t.Logf("OC-COMP-2 PASS: heartbeat from unregistered provider correctly rejected")
}

// ==================================================================================
// OC-COMP-3: UpdateParams Authorization Bypass
//
// ATTACK: Non-authority tries to update compute pool parameters.
//
// DEFENSE: UpdateComputeParams checks msg.Authority == k.GetAuthority().
// ==================================================================================

func TestOC_COMP3_UpdateParamsAuthBypass(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)

	// ATTACK: Random user tries to update params
	attackerAddr := sdk.AccAddress([]byte("attacker____________")).String()
	_, err := srv.UpdateComputeParams(ctx, &types.MsgUpdateComputeParams{
		Authority: attackerAddr,
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Fatal("OC-COMP-3 FAIL: unauthorized user was able to update params!")
	}

	t.Logf("OC-COMP-3 PASS: param update authorization enforced")
}
