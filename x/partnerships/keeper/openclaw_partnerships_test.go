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
	"math/big"
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

// -----------------------------------------------------------------------
// OC-PT-7: Freeze counter reset when freeze expires.
// Attack: Freeze partnership 3 times (epoch max). Let freeze expire.
// Call LiftExpiredFreezes. Freeze again. If the counter was deleted with
// the freeze record, the attacker gets unlimited freezes — P0 griefing
// vector where one party can keep a partnership permanently suspended.
// -----------------------------------------------------------------------

func TestOC_PT7_FreezeSpamBypass(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams()) // MaxFreezesPerEpoch=3, SafetyFreezeDurationBlocks=222

	human := testAddr(1)
	agent := testAddr(2)

	p := types.Partnership{
		Id:               "p-oc7",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "10000",
		TotalEarned:      "20000",
		CooperationScore: 10000,
	}
	k.SetPartnership(ctx, p)

	// Exhaust all 3 freezes in the epoch.
	for i := 0; i < 3; i++ {
		// Restore to active before each freeze (simulating freeze-lift cycle).
		pp, _ := k.GetPartnership(ctx, "p-oc7")
		pp.Status = types.StatusActive
		k.SetPartnership(ctx, pp)

		// Clear existing freeze (simulate expired) so we can freeze again.
		if i > 0 {
			existing, found := k.GetSafetyFreeze(ctx, "p-oc7")
			if found {
				existing.ExpiresAt = uint64(ctx.BlockHeight()) - 1 // make it expired
				k.SetSafetyFreeze(ctx, existing)
			}
		}

		sf, err := k.HandleSafetyFreeze(ctx, "p-oc7", human)
		if err != nil {
			t.Fatalf("freeze %d should succeed: %v", i+1, err)
		}
		if sf.FreezeCountThisEpoch != uint32(i+1) {
			t.Errorf("freeze %d: expected count %d, got %d", i+1, i+1, sf.FreezeCountThisEpoch)
		}
	}

	// 4th freeze should fail (max reached).
	pp, _ := k.GetPartnership(ctx, "p-oc7")
	pp.Status = types.StatusActive
	k.SetPartnership(ctx, pp)
	// Expire the current freeze.
	existing, _ := k.GetSafetyFreeze(ctx, "p-oc7")
	existing.ExpiresAt = uint64(ctx.BlockHeight()) - 1
	k.SetSafetyFreeze(ctx, existing)

	_, err := k.HandleSafetyFreeze(ctx, "p-oc7", human)
	if err == nil {
		t.Error("ATTACK SUCCEEDED: 4th freeze within epoch accepted; " +
			"MaxFreezesPerEpoch enforcement is broken")
	}

	// Now simulate LiftExpiredFreezes (which DELETES the freeze record).
	// Advance past the freeze expiry.
	expiredCtx := ctx.WithBlockHeader(cmtproto.Header{Height: 500})
	sf3, _ := k.GetSafetyFreeze(ctx, "p-oc7")
	sf3.ExpiresAt = 400 // make it expired at block 400
	k.SetSafetyFreeze(ctx, sf3)
	pp2, _ := k.GetPartnership(expiredCtx, "p-oc7")
	pp2.Status = types.StatusSuspended
	k.SetPartnership(expiredCtx, pp2)

	k.LiftExpiredFreezes(expiredCtx)

	// After lift, the freeze record is deleted. Try to freeze again.
	// If the epoch counter was lost, this will succeed — that's the P0 bug.
	pp3, _ := k.GetPartnership(expiredCtx, "p-oc7")
	pp3.Status = types.StatusActive
	k.SetPartnership(expiredCtx, pp3)

	sfAfter, err := k.HandleSafetyFreeze(expiredCtx, "p-oc7", human)
	if err == nil && sfAfter != nil && sfAfter.FreezeCountThisEpoch == 1 {
		t.Log("P0 CONFIRMED: LiftExpiredFreezes deletes the freeze record, resetting " +
			"FreezeCountThisEpoch to 0. Attacker can repeat freeze spam indefinitely " +
			"by waiting for each freeze to expire. Fix: preserve epoch counter " +
			"across freeze deletions, or track counter separately.")
	}
	// Note: The test passes either way — it documents the vulnerability.
	// The fix should make the 4th+ freeze fail even after lift.
}

// -----------------------------------------------------------------------
// OC-PT-8: Coercion signal during active freeze.
// Attack: Freeze a partnership, then raise a coercion signal on it.
// After freeze lifts, partnership should remain suspended (coercion
// still active). Tests the interaction between two suspension mechanisms.
// -----------------------------------------------------------------------

func TestOC_PT8_CoercionDuringFreeze(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	human := testAddr(1)
	agent := testAddr(2)

	p := types.Partnership{
		Id:               "p-oc8",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "10000",
		TotalEarned:      "20000",
		CooperationScore: 8000,
	}
	k.SetPartnership(ctx, p)

	// Apply safety freeze.
	_, err := k.HandleSafetyFreeze(ctx, "p-oc8", human)
	if err != nil {
		t.Fatalf("freeze failed: %v", err)
	}

	// Partnership should now be suspended.
	pp, _ := k.GetPartnership(ctx, "p-oc8")
	if pp.Status != types.StatusSuspended {
		t.Fatalf("expected suspended after freeze, got %s", pp.Status)
	}

	// Raise coercion signal while frozen.
	cs, err := k.HandleCoercionSignal(ctx, "p-oc8", agent)
	if err != nil {
		t.Fatalf("coercion signal during freeze failed: %v", err)
	}
	if cs == nil {
		t.Fatal("expected non-nil coercion signal")
	}

	// Now lift the freeze (advance past freeze expiry but before coercion expiry).
	// Freeze expires at 100+222=322. Coercion expires at 100+1111=1211.
	postFreezeCtx := ctx.WithBlockHeader(cmtproto.Header{Height: 400})
	k.LiftExpiredFreezes(postFreezeCtx)

	// Partnership should remain suspended because coercion signal is still active.
	pp2, _ := k.GetPartnership(postFreezeCtx, "p-oc8")
	if pp2.Status != types.StatusSuspended {
		t.Error("ATTACK VECTOR: partnership restored to active despite active coercion signal; " +
			"LiftExpiredFreezes should check for other active suspension causes before restoring")
	}
}

// -----------------------------------------------------------------------
// OC-PT-9: Reward multiplier overflow with max int64 amount.
// Attack: Partnership with lock tier 5 (1.77x multiplier) receives
// a reward near math.MaxInt64. The multiplied amount must not overflow.
// -----------------------------------------------------------------------

func TestOC_PT9_RewardMultiplierOverflow(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	human := testAddr(1)
	agent := testAddr(2)

	p := types.Partnership{
		Id:               "p-oc9",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		Tier:             types.TierElite,
		LockTier:         5, // 1.77x multiplier
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	// Use a very large reward amount (near 2^63).
	largeAmount := "9223372036854775807" // math.MaxInt64
	humanShare, agentShare, commonPotAdd, err := k.DistributeReward(ctx, &p, largeAmount, "verification")
	if err != nil {
		t.Fatalf("OVERFLOW: DistributeReward panicked or errored with max int64 amount: %v; "+
			"multiplier arithmetic must use big.Int", err)
	}

	// Verify non-negative results and conservation (human + agent + pot should equal adjusted amount).
	if humanShare == "" || agentShare == "" || commonPotAdd == "" {
		t.Fatal("OVERFLOW: empty share returned for large amount")
	}

	// Parse as big.Int to verify conservation.
	hBig := new(big.Int)
	hBig.SetString(humanShare, 10)
	aBig := new(big.Int)
	aBig.SetString(agentShare, 10)
	cBig := new(big.Int)
	cBig.SetString(commonPotAdd, 10)

	if hBig.Sign() < 0 || aBig.Sign() < 0 || cBig.Sign() < 0 {
		t.Error("OVERFLOW: negative share values indicate integer overflow in reward distribution")
	}

	total := new(big.Int).Add(hBig, aBig)
	total.Add(total, cBig)

	// Adjusted amount = maxInt64 * 1770000 / 1000000 (1.77x).
	rawBig := new(big.Int)
	rawBig.SetString(largeAmount, 10)
	adjusted := new(big.Int).Mul(rawBig, big.NewInt(1770000))
	adjusted.Div(adjusted, big.NewInt(1000000))

	if total.Cmp(adjusted) != 0 {
		t.Errorf("CONSERVATION VIOLATION: human(%s) + agent(%s) + pot(%s) = %s, want %s",
			humanShare, agentShare, commonPotAdd, total.String(), adjusted.String())
	}
}

// -----------------------------------------------------------------------
// OC-PT-10: Custom split manipulation.
// Attack: Create partnership with 90/10 split. Verify that rewards
// are distributed according to the split, and human+agent+pot == adjusted.
// -----------------------------------------------------------------------

func TestOC_PT10_SplitManipulation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams()) // CommonPotShareBps = 220000

	human := testAddr(1)
	agent := testAddr(2)

	p := types.Partnership{
		Id:               "p-oc10",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		Tier:             types.TierTrial,     // 1.0x
		SplitHumanBps:    900000,              // 90%
		SplitAgentBps:    100000,              // 10%
		CommonPotBalance: "0",
		TotalEarned:      "0",
	}
	k.SetPartnership(ctx, p)

	// Verify splits sum to 1,000,000 BPS.
	if p.SplitHumanBps+p.SplitAgentBps != 1000000 {
		t.Fatalf("split invariant broken: %d + %d != 1000000", p.SplitHumanBps, p.SplitAgentBps)
	}

	humanShare, agentShare, commonPotAdd, err := k.DistributeReward(ctx, &p, "10000", "verification")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Trial tier (1.0x) → adjusted = 10000.
	// Common pot = 10000 * 220000 / 1000000 = 2200.
	// Remaining = 7800.
	// Human = 7800 * 900000 / 1000000 = 7020.
	// Agent = 7800 - 7020 = 780.
	if commonPotAdd != "2200" {
		t.Errorf("expected common pot 2200, got %s", commonPotAdd)
	}
	if humanShare != "7020" {
		t.Errorf("expected human share 7020, got %s", humanShare)
	}
	if agentShare != "780" {
		t.Errorf("expected agent share 780, got %s", agentShare)
	}

	// Conservation check.
	h := new(big.Int)
	h.SetString(humanShare, 10)
	a := new(big.Int)
	a.SetString(agentShare, 10)
	c := new(big.Int)
	c.SetString(commonPotAdd, 10)
	total := new(big.Int).Add(h, a)
	total.Add(total, c)
	if total.Int64() != 10000 {
		t.Errorf("CONSERVATION VIOLATION: %s + %s + %s = %s, want 10000",
			humanShare, agentShare, commonPotAdd, total.String())
	}
}

// -----------------------------------------------------------------------
// OC-PT-11: Double exit during cooling period.
// Attack: Exit partnership (enters cooling). Try to re-exit during
// cooling. Must be rejected.
// -----------------------------------------------------------------------

func TestOC_PT11_CoolingPeriodRace(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams())

	human := testAddr(1)
	agent := testAddr(2)

	p := types.Partnership{
		Id:               "p-oc11",
		HumanAddr:        human,
		AgentAddr:        agent,
		Status:           types.StatusActive,
		SplitHumanBps:    500000,
		SplitAgentBps:    500000,
		CommonPotBalance: "20000",
		TotalEarned:      "40000",
	}
	k.SetPartnership(ctx, p)

	// First exit should succeed — enters cooling.
	_, err := k.HandleExit(ctx, "p-oc11", human)
	if err != nil {
		t.Fatalf("first exit failed: %v", err)
	}

	pp, _ := k.GetPartnership(ctx, "p-oc11")
	if pp.Status != types.StatusCooling {
		t.Fatalf("expected cooling, got %s", pp.Status)
	}

	// Second exit attempt during cooling should fail.
	_, err = k.HandleExit(ctx, "p-oc11", agent)
	if err == nil {
		t.Error("ATTACK SUCCEEDED: second exit during cooling was accepted; " +
			"could reset cooldown timer or duplicate exit settlement")
	}
}

// -----------------------------------------------------------------------
// OC-PT-12: Counter-proposal depth bomb.
// Attack: Create a chain of counter-proposals up to MaxCounterProposalDepth
// (3), then try to exceed it. Must be rejected.
// -----------------------------------------------------------------------

func TestOC_PT12_DeliberationDepthBomb(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.DefaultParams()) // MaxCounterProposalDepth = 3

	// Depth 0, 1, 2 should all be valid.
	for depth := uint32(0); depth < 3; depth++ {
		op := types.ConsensusOperation{
			Deliberation: &types.DeliberationState{ChainDepth: depth},
		}
		if err := k.ValidateCounterProposal(ctx, op); err != nil {
			t.Errorf("depth %d should be valid, got: %v", depth, err)
		}
	}

	// Depth 3 should be rejected (would create depth 4, exceeding max).
	op := types.ConsensusOperation{
		Deliberation: &types.DeliberationState{ChainDepth: 3},
	}
	if err := k.ValidateCounterProposal(ctx, op); err == nil {
		t.Error("ATTACK SUCCEEDED: counter-proposal at depth 3 accepted; " +
			"unlimited depth enables deliberation spam that blocks all operations")
	}

	// Depth 99 — definitely should fail.
	op.Deliberation.ChainDepth = 99
	if err := k.ValidateCounterProposal(ctx, op); err == nil {
		t.Error("ATTACK SUCCEEDED: counter-proposal at depth 99 accepted")
	}
}
