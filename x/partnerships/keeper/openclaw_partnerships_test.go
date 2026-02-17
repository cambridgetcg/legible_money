package keeper_test

// ==================================================================================
// OpenClaw Partnerships Adversarial Probe Suite (OC-PT-1 through OC-PT-6)
//
// These probes systematically test the partnerships module's attack surface:
// - Authorization boundary enforcement (non-participant deposit, unauthorized dissolution)
// - Input validation (zero-amount deposit)
// - Self-approval prevention (proposer cannot approve own operation)
// - Formation window expiry enforcement
// - Double-acceptance prevention
// ==================================================================================

import (
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/partnerships/keeper"
	"github.com/legible-money/legible/x/partnerships/types"
)

// -----------------------------------------------------------------------
// OC-PT-1: Non-participant tries to deposit into a partnership pot.
// Attack: Third-party address attempts DepositToPot on a partnership
// between two other addresses. Must be rejected by participant check.
// -----------------------------------------------------------------------

func TestOC_PT1_NonParticipantDeposit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	human := testAddr(1)
	agent := testAddr(2)
	outsider := testAddr(99)

	// Create an active partnership between human and agent.
	p := types.Partnership{
		Id:               "p-oc1",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "10000",
		TotalEarned:      "20000",
	}
	k.SetPartnership(ctx, p)

	// Outsider attempts to deposit into the partnership pot.
	_, err := srv.DepositToPot(ctx, &types.MsgDepositToPot{
		Depositor:     outsider,
		PartnershipId: "p-oc1",
		Amount:        "5000",
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: non-participant was able to deposit into a partnership pot; " +
			"authorization check is missing or broken, allowing anyone to inject funds")
	}

	// Verify pot balance was NOT changed.
	got, found := k.GetPartnership(ctx, "p-oc1")
	if !found {
		t.Fatal("partnership should still exist")
	}
	if got.CommonPotBalance != "10000" {
		t.Errorf("pot balance should be unchanged at 10000, got %s", got.CommonPotBalance)
	}
}

// -----------------------------------------------------------------------
// OC-PT-2: Zero-amount deposit attempt.
// Attack: Participant submits a DepositToPot with Amount "0".
// This must be rejected by validation to prevent no-op state writes
// and potential gas-griefing.
// -----------------------------------------------------------------------

func TestOC_PT2_ZeroAmountDeposit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	human := testAddr(1)
	agent := testAddr(2)

	p := types.Partnership{
		Id:               "p-oc2",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "5000",
		TotalEarned:      "10000",
	}
	k.SetPartnership(ctx, p)

	// Attempt zero-amount deposit via ValidateBasic first.
	msg := &types.MsgDepositToPot{
		Depositor:     human,
		PartnershipId: "p-oc2",
		Amount:        "0",
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Error("ATTACK SUCCEEDED: ValidateBasic accepted zero-amount deposit; " +
			"allows gas-griefing via no-op deposit transactions")
	}

	// Also verify the server handler rejects it if ValidateBasic is bypassed.
	_, err := srv.DepositToPot(ctx, msg)
	if err == nil {
		t.Error("ATTACK SUCCEEDED: server handler accepted zero-amount deposit; " +
			"amount validation is missing in the keeper-level handler")
	}

	// Verify pot balance unchanged.
	got, _ := k.GetPartnership(ctx, "p-oc2")
	if got.CommonPotBalance != "5000" {
		t.Errorf("pot balance should be unchanged at 5000, got %s", got.CommonPotBalance)
	}
}

// -----------------------------------------------------------------------
// OC-PT-3: Self-approval of a proposed operation.
// Attack: Address proposes an operation, then tries to approve it
// themselves. The consensus model requires the OTHER partner to approve.
// Self-approval would let one party unilaterally drain the pot.
// -----------------------------------------------------------------------

func TestOC_PT3_SelfApprovalOperation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	human := testAddr(1)
	agent := testAddr(2)

	p := types.Partnership{
		Id:               "p-oc3",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "50000",
		TotalEarned:      "100000",
	}
	k.SetPartnership(ctx, p)

	// Human proposes a withdrawal operation.
	propResp, err := srv.ProposeOperation(ctx, &types.MsgProposeOperation{
		Proposer:      human,
		PartnershipId: "p-oc3",
		OperationType: "withdrawal",
		Amount:        "25000",
		Rationale:     "need funds",
	})
	if err != nil {
		t.Fatalf("failed to propose operation: %v", err)
	}

	// Now the SAME human tries to approve their own proposal.
	_, err = srv.ApproveOperation(ctx, &types.MsgApproveOperation{
		Approver:    human,
		OperationId: propResp.OperationId,
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: proposer was able to approve their own operation; " +
			"this breaks the two-party consensus model and allows unilateral pot drainage")
	}

	// Verify pot balance was NOT changed (withdrawal should not have executed).
	got, _ := k.GetPartnership(ctx, "p-oc3")
	if got.CommonPotBalance != "50000" {
		t.Errorf("pot balance should be unchanged at 50000, got %s; " +
			"self-approval may have partially executed the withdrawal", got.CommonPotBalance)
	}
}

// -----------------------------------------------------------------------
// OC-PT-4: Formation window expiry bypass attempt.
// Attack: Partnership is initiated, then acceptance is attempted
// AFTER the FormationWindowBlocks has elapsed. Must be rejected.
// -----------------------------------------------------------------------

func TestOC_PT4_FormationWindowExpiry(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams()) // FormationWindowBlocks = 77
	srv := keeper.NewMsgServerImpl(k)

	human := testAddr(1)
	agent := testAddr(2)

	// Initiate partnership at block 100.
	resp, err := srv.InitiatePartnership(ctx, &types.MsgInitiatePartnership{
		Initiator: human,
		Partner:   agent,
	})
	if err != nil {
		t.Fatalf("failed to initiate partnership: %v", err)
	}

	// Advance context past the formation window (100 + 77 = 177, go to 200).
	expiredCtx := ctx.WithBlockHeader(cmtproto.Header{Height: 200})

	// Try to accept the partnership after the window has expired.
	_, err = srv.AcceptPartnership(expiredCtx, &types.MsgAcceptPartnership{
		Acceptor:      agent,
		PartnershipId: resp.PartnershipId,
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: partnership was accepted after formation window expired; " +
			"time-bombed acceptances could be used to activate stale partnerships")
	}

	// Verify partnership remains in forming status (not activated).
	p, found := k.GetPartnership(ctx, resp.PartnershipId)
	if !found {
		t.Fatal("partnership should still exist")
	}
	if p.Status != types.StatusForming {
		t.Errorf("partnership should still be forming, got %s", p.Status)
	}
}

// -----------------------------------------------------------------------
// OC-PT-5: Double-acceptance of a partnership.
// Attack: Partnership is initiated and accepted once (becoming active).
// A second acceptance is attempted. Must fail because the partnership
// is no longer in "forming" status.
// -----------------------------------------------------------------------

func TestOC_PT5_DoubleAcceptance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	human := testAddr(1)
	agent := testAddr(2)

	// Initiate and accept.
	resp, err := srv.InitiatePartnership(ctx, &types.MsgInitiatePartnership{
		Initiator: human,
		Partner:   agent,
	})
	if err != nil {
		t.Fatalf("failed to initiate: %v", err)
	}

	_, err = srv.AcceptPartnership(ctx, &types.MsgAcceptPartnership{
		Acceptor:      agent,
		PartnershipId: resp.PartnershipId,
	})
	if err != nil {
		t.Fatalf("first accept failed: %v", err)
	}

	// Verify it is active.
	p, _ := k.GetPartnership(ctx, resp.PartnershipId)
	if p.Status != types.StatusActive {
		t.Fatalf("expected active after first accept, got %s", p.Status)
	}

	// Try to accept again.
	_, err = srv.AcceptPartnership(ctx, &types.MsgAcceptPartnership{
		Acceptor:      agent,
		PartnershipId: resp.PartnershipId,
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: partnership was accepted a second time; " +
			"double-acceptance could reset formation state or re-trigger events")
	}
}

// -----------------------------------------------------------------------
// OC-PT-6: Unauthorized dissolution attempt.
// Attack: An outsider (address C) who is NOT a participant in the
// partnership between A and B tries to initiate an exit/dissolution.
// Must be rejected by participant check.
// -----------------------------------------------------------------------

func TestOC_PT6_UnauthorizedDissolution(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())
	srv := keeper.NewMsgServerImpl(k)

	human := testAddr(1)
	agent := testAddr(2)
	outsider := testAddr(99)

	p := types.Partnership{
		Id:               "p-oc6",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "30000",
		TotalEarned:      "60000",
	}
	k.SetPartnership(ctx, p)

	// Outsider tries to initiate exit.
	_, err := srv.InitiateExit(ctx, &types.MsgInitiateExit{
		Initiator:     outsider,
		PartnershipId: "p-oc6",
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: outsider was able to initiate dissolution of a partnership " +
			"they are not a member of; authorization check is missing, allowing third-party " +
			"forced dissolution and pot theft")
	}

	// Verify partnership is still active and pot untouched.
	got, found := k.GetPartnership(ctx, "p-oc6")
	if !found {
		t.Fatal("partnership should still exist")
	}
	if got.Status != types.StatusActive {
		t.Errorf("partnership should still be active, got %s", got.Status)
	}
	if got.CommonPotBalance != "30000" {
		t.Errorf("pot balance should be unchanged at 30000, got %s", got.CommonPotBalance)
	}
}
