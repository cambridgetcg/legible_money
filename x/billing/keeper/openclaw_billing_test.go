package keeper_test

// ==================================================================================
// OpenClaw Billing Adversarial Probe Suite (OC-BILL-1 through OC-BILL-4)
//
// Attack surfaces tested:
// - Quote replay (execute same quote twice)
// - Quote caller impersonation (execute someone else's quote)
// - Quote temporal bypass (execute expired quote)
// - Zero/negative stake registration
// ==================================================================================

import (
	"fmt"
	"testing"

	"github.com/legible-money/legible/x/billing/keeper"
	"github.com/legible-money/legible/x/billing/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ==================================================================================
// OC-BILL-1: Quote Replay Attack — Double Execution
//
// ATTACK: Execute a quote, then try to execute the same quote again.
// If unguarded, the caller gets double the goods for the price of one.
//
// DEFENSE: quote.Executed is set to true after first execution.
// ==================================================================================

func TestOC_BILL1_QuoteReplayAttack(t *testing.T) {
	k, ctx, bk, kk := setupKeeper(t)

	providerAddr := testAddr(1)
	callerAddr := testAddr(2)
	accProvider, _ := sdk.AccAddressFromBech32(providerAddr)
	accCaller, _ := sdk.AccAddressFromBech32(callerAddr)

	// Fund accounts
	bk.setBalance(accProvider, "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(accCaller, "ulgm", sdkmath.NewInt(200000000))

	// Register provider
	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      providerAddr,
		Domains:     []string{"mathematics"},
		Stake: "100000000",
	})
	if err != nil {
		t.Fatalf("register provider: %v", err)
	}

	// Add a fact
	kk.addFact("fact-replay", 500000, 10, 50, testAddr(10))

	// Generate and execute a quote
	quote := k.GenerateQuote(ctx, providerAddr, callerAddr, []string{"fact-replay"}, 100)

	// Fund module account for distribution
	bk.setBalance(sdk.AccAddress([]byte("billing-module------")), "ulgm", sdkmath.NewInt(100000000))

	// First execution — should succeed
	_, err = srv.ExecutePayment(ctx, &types.MsgExecutePayment{
		Sender:  callerAddr,
		QuoteId: fmt.Sprintf("%d", quote.QuoteId),
	})
	if err != nil {
		t.Fatalf("OC-BILL-1 FAIL: first execution should succeed: %v", err)
	}

	// ATTACK: Second execution — should be rejected
	_, err = srv.ExecutePayment(ctx, &types.MsgExecutePayment{
		Sender:  callerAddr,
		QuoteId: fmt.Sprintf("%d", quote.QuoteId),
	})
	if err == nil {
		t.Fatal("OC-BILL-1 FAIL: quote replay attack succeeded! Double execution allowed.")
	}

	t.Logf("OC-BILL-1 PASS: quote replay correctly rejected")
}

// ==================================================================================
// OC-BILL-2: Quote Caller Impersonation
//
// ATTACK: Alice generates a quote. Bob tries to execute it.
// If unguarded, Bob can use Alice's quotes (or vice versa).
//
// DEFENSE: ExecutePayment checks msg.Sender == quote.Caller.
// ==================================================================================

func TestOC_BILL2_QuoteCallerImpersonation(t *testing.T) {
	k, ctx, bk, kk := setupKeeper(t)

	providerAddr := testAddr(1)
	aliceAddr := testAddr(2)
	bobAddr := testAddr(3)
	accProvider, _ := sdk.AccAddressFromBech32(providerAddr)
	accAlice, _ := sdk.AccAddressFromBech32(aliceAddr)
	accBob, _ := sdk.AccAddressFromBech32(bobAddr)

	bk.setBalance(accProvider, "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(accAlice, "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(accBob, "ulgm", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)
	_, _ = srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      providerAddr,
		Domains:     []string{"mathematics"},
		Stake: "100000000",
	})

	kk.addFact("fact-impersonate", 500000, 10, 50, testAddr(10))

	// Alice generates a quote
	quote := k.GenerateQuote(ctx, providerAddr, aliceAddr, []string{"fact-impersonate"}, 100)

	// ATTACK: Bob tries to execute Alice's quote
	_, err := srv.ExecutePayment(ctx, &types.MsgExecutePayment{
		Sender:  bobAddr,
		QuoteId: fmt.Sprintf("%d", quote.QuoteId),
	})
	if err == nil {
		t.Fatal("OC-BILL-2 FAIL: Bob was able to execute Alice's quote!")
	}

	t.Logf("OC-BILL-2 PASS: caller impersonation correctly rejected")
}

// ==================================================================================
// OC-BILL-3: Expired Quote Execution
//
// ATTACK: Generate a quote, advance past its validity window, then execute.
// If unguarded, expired quotes can be used indefinitely.
//
// DEFENSE: ExecutePayment checks currentBlock > quote.ValidUntil.
// ==================================================================================

func TestOC_BILL3_ExpiredQuoteExecution(t *testing.T) {
	k, ctx, bk, kk := setupKeeper(t)

	providerAddr := testAddr(1)
	callerAddr := testAddr(2)
	accProvider, _ := sdk.AccAddressFromBech32(providerAddr)
	accCaller, _ := sdk.AccAddressFromBech32(callerAddr)

	bk.setBalance(accProvider, "ulgm", sdkmath.NewInt(200000000))
	bk.setBalance(accCaller, "ulgm", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)
	_, _ = srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      providerAddr,
		Domains:     []string{"physics"},
		Stake: "100000000",
	})

	kk.addFact("fact-expired", 500000, 10, 50, testAddr(10))

	// Generate quote at block 100 (default validity: 100 blocks => valid until 200)
	quote := k.GenerateQuote(ctx, providerAddr, callerAddr, []string{"fact-expired"}, 100)

	// Advance past validity window
	expiredCtx := ctx.WithBlockHeight(int64(quote.ValidUntil + 1))

	// ATTACK: Execute expired quote
	_, err := srv.ExecutePayment(expiredCtx, &types.MsgExecutePayment{
		Sender:  callerAddr,
		QuoteId: fmt.Sprintf("%d", quote.QuoteId),
	})
	if err == nil {
		t.Fatal("OC-BILL-3 FAIL: expired quote was accepted!")
	}

	t.Logf("OC-BILL-3 PASS: expired quote correctly rejected")
}

// ==================================================================================
// OC-BILL-4: UpdateParams Authorization Bypass
//
// ATTACK: Non-authority address tries to update billing params.
// If unguarded, anyone can change fee structures.
//
// DEFENSE: UpdateParams checks msg.Authority == k.GetAuthority().
// ==================================================================================

func TestOC_BILL4_UpdateParamsAuthBypass(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)

	// ATTACK: Random user tries to update params
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr(99),
		Params: &types.Params{
			BaseCostPerFact:  "1",        // Set absurdly low cost
			MinProviderStake: "1",        // Set absurdly low stake
			ResearchShareBps: 0,          // Zero out research funding
		},
	})
	if err == nil {
		t.Fatal("OC-BILL-4 FAIL: unauthorized user was able to update params!")
	}

	// Verify params unchanged
	params := k.GetParams(ctx)
	if params.BaseCostPerFact != "1000000" {
		t.Fatalf("OC-BILL-4 FAIL: params were modified by unauthorized party: %s", params.BaseCostPerFact)
	}

	t.Logf("OC-BILL-4 PASS: param update authorization enforced")
}
