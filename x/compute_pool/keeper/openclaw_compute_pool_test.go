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
	k, ctx, bk := setupKeeper(t)
	addr := testAddr(1)
	bk.setBalance(addr, "ulgm", 500000000)

	srv := keeper.NewMsgServerImpl(k)

	// First registration
	_, err := srv.RegisterComputeProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender:      addr,
		ServiceType: "inference",
		PricePerCU:  "1000",
		Stake:       "100000000",
	})
	if err != nil {
		t.Fatalf("first registration: %v", err)
	}

	// ATTACK: Second registration (duplicate)
	_, err = srv.RegisterComputeProvider(ctx, &types.MsgRegisterComputeProvider{
		Sender:      addr,
		ServiceType: "training",
		PricePerCU:  "2000",
		Stake:       "100000000",
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
	k, ctx, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)

	// ATTACK: Unregistered address sends heartbeat
	ghostAddr := testAddr(99)
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
	k, ctx, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)

	// ATTACK: Random user tries to update params
	_, err := srv.UpdateComputeParams(ctx, &types.MsgUpdateComputeParams{
		Authority: testAddr(99),
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Fatal("OC-COMP-3 FAIL: unauthorized user was able to update params!")
	}

	t.Logf("OC-COMP-3 PASS: param update authorization enforced")
}
