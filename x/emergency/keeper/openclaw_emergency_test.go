package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/emergency/keeper"
	"github.com/legible-money/legible/x/emergency/types"
)

// ============ OpenClaw Adversarial Tests: Emergency Module ============
//
// These tests probe the 3-phase BFT kill switch for access control bypasses,
// quorum manipulation, timing attacks, and ceremony lifecycle edge cases
// that could allow unauthorized halts or prevent legitimate emergency responses.

// OC-EM-1: Non-Guardian Halt Attempt — A tier-2 bonded validator (not guardian)
// tries to propose halt. Verify ErrNotGuardian.
func TestOC_EM1_NonGuardianHaltAttempt(t *testing.T) {
	_, ctx, _ := setupKeeper(t)
	k, ctx2, _ := setupKeeper(t)
	_ = ctx
	msgServer := keeper.NewMsgServerImpl(k)

	// bonded1 is tier 2 (bonded), not guardian
	_, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx2), &types.MsgProposeHalt{
		Proposer: "bonded1",
		Reason:   "I want to halt the chain",
	})
	if err == nil {
		t.Error("OC-EM-1: non-guardian halt proposal should fail — attacker can halt chain without guardian status")
	}

	// Also test a completely unknown address
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx2), &types.MsgProposeHalt{
		Proposer: "random_attacker",
		Reason:   "chaos",
	})
	if err == nil {
		t.Error("OC-EM-1: unknown address halt proposal should fail — no access control on halt proposals")
	}
}

// OC-EM-2: Cooldown Bypass — After a ceremony completes/fails, immediately try
// another halt. Verify cooldown enforcement and boundary at CooldownBlocks=111.
func TestOC_EM2_CooldownBypass(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// First proposal — succeeds
	resp, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "first ceremony",
	})
	if err != nil {
		t.Fatalf("OC-EM-2: first proposal failed: %v", err)
	}

	// Advance past timeout so ceremony fails
	ctx = ctx.WithBlockHeight(200)
	k.CheckCeremonyProgress(ctx, resp.ProposalId)
	k.HandleCeremonyFailure(ctx, resp.ProposalId)

	// Immediately try again — should fail (cooldown from block 100, ends at 100+111=211)
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian2",
		Reason:   "too soon",
	})
	if err == nil {
		t.Error("OC-EM-2: proposal during cooldown should fail — attacker can spam ceremonies")
	}

	// At block 210 — still in cooldown (100 + 111 = 211)
	ctx = ctx.WithBlockHeight(210)
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian3",
		Reason:   "off by one",
	})
	if err == nil {
		t.Error("OC-EM-2: proposal at cooldown boundary-1 should fail")
	}

	// At block 211 or beyond — cooldown elapsed
	// Need to advance epoch counters since guardian1 used their per-epoch limit
	k.ResetEpochCounters(ctx)
	ctx = ctx.WithBlockHeight(212)
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian3",
		Reason:   "cooldown elapsed",
	})
	if err != nil {
		t.Errorf("OC-EM-2: proposal after cooldown should succeed: %v", err)
	}
}

// OC-EM-3: Proposal Spam Per Epoch — Same guardian tries to propose twice in
// one epoch. Verify MaxProposalsPerGuardianPerEpoch=1 is enforced.
func TestOC_EM3_ProposalSpamPerEpoch(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// First proposal from guardian1 — succeeds
	resp, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "first",
	})
	if err != nil {
		t.Fatalf("OC-EM-3: first proposal failed: %v", err)
	}

	// Fail the ceremony and pass cooldown
	ctx = ctx.WithBlockHeight(400)
	k.CheckCeremonyProgress(ctx, resp.ProposalId)
	k.HandleCeremonyFailure(ctx, resp.ProposalId)

	// guardian1 tries again in same epoch — should fail
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "second attempt",
	})
	if err == nil {
		t.Error("OC-EM-3: same guardian proposing twice per epoch should fail — attacker can spam halt ceremonies")
	}

	// Different guardian — should succeed (if cooldown elapsed)
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian2",
		Reason:   "different guardian",
	})
	if err != nil {
		t.Errorf("OC-EM-3: different guardian should be able to propose: %v", err)
	}
}

// OC-EM-4: Precommit Without Prevote — Start halt ceremony. Guardian tries to
// precommit without having prevoted YES. Verify ErrPrevoteRequired.
func TestOC_EM4_PrecommitWithoutPrevote(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "precommit test",
	})

	// Get 4 guardians to prevote (advances to precommit phase)
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: resp.ProposalId, Approve: true,
		})
	}

	// Verify phase is precommit
	ceremony, _ := k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrecommit) {
		t.Fatalf("OC-EM-4: expected precommit phase, got %s", ceremony.Phase)
	}

	// guardian5 did NOT prevote — their precommit should fail
	_, err := msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian5", ProposalId: resp.ProposalId, Approve: true,
	})
	if err == nil {
		t.Error("OC-EM-4: precommit without prevote should fail — attacker can skip prevote phase")
	}
}

// OC-EM-5: Duplicate Vote in Phase — Guardian prevotes YES, then tries to
// prevote again. Verify ErrDuplicateVote.
func TestOC_EM5_DuplicateVoteInPhase(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "duplicate test",
	})

	// First prevote — succeeds
	_, err := msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian2", ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Fatalf("OC-EM-5: first prevote failed: %v", err)
	}

	// Duplicate prevote — should fail
	_, err = msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian2", ProposalId: resp.ProposalId, Approve: true,
	})
	if err == nil {
		t.Error("OC-EM-5: duplicate prevote should fail — attacker can double-count votes")
	}

	// Also test vote-flipping: prevote NO after YES
	_, err = msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian2", ProposalId: resp.ProposalId, Approve: false,
	})
	if err == nil {
		t.Error("OC-EM-5: vote flip (YES→NO) should fail — attacker can manipulate quorum")
	}
}

// OC-EM-6: Halt from Wrong Status — Set status to "halted" and try to propose
// another halt. Verify ErrStatusConflict.
func TestOC_EM6_HaltFromWrongStatus(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set to halted
	k.SetEmergencyStatus(ctx, types.StatusHalted)

	_, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "already halted",
	})
	if err == nil {
		t.Error("OC-EM-6: halt proposal while already halted should fail — status conflict not enforced")
	}

	// Set to resume_voting and try halt
	k.SetEmergencyStatus(ctx, types.StatusResumeVoting)
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian2",
		Reason:   "during resume voting",
	})
	if err == nil {
		t.Error("OC-EM-6: halt proposal during resume_voting should fail")
	}

	// Set to revert_voting and try halt
	k.SetEmergencyStatus(ctx, types.StatusRevertVoting)
	_, err = msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian3",
		Reason:   "during revert voting",
	})
	if err == nil {
		t.Error("OC-EM-6: halt proposal during revert_voting should fail")
	}
}

// OC-EM-7: Revert Depth Exceeded — Chain halted at high block. Guardian proposes
// revert to very old block exceeding MaxRevertDepth. Verify boundary handling.
func TestOC_EM7_RevertDepthExceeded(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Execute halt
	haltResp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "depth test",
	})
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: haltResp.ProposalId, Approve: true,
		})
	}
	for _, g := range []string{"guardian1", "guardian2", "guardian3", "guardian4"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: haltResp.ProposalId, Approve: true,
		})
	}

	if k.GetEmergencyStatus(ctx) != types.StatusHalted {
		t.Fatal("OC-EM-7: chain should be halted")
	}

	params := k.GetParams(ctx)
	maxDepth := params.MaxRevertDepth // 111111

	// Set high block height and propose revert beyond MaxRevertDepth
	highBlock := int64(200000)
	ctx = ctx.WithBlockHeight(highBlock)
	targetBlock := highBlock - int64(maxDepth) - 1 // depth = maxDepth + 1

	_, err := msgServer.ProposeRevert(sdk.WrapSDKContext(ctx), &types.MsgProposeRevert{
		Proposer:       "guardian1",
		RevertToHeight: uint64(targetBlock),
		Justification:  "too deep",
	})
	if err == nil {
		t.Errorf("OC-EM-7: revert depth %d > max %d should fail — attacker can revert arbitrarily deep", highBlock-targetBlock, maxDepth)
	}

	// Revert to block 0 — boundary test
	_, err = msgServer.ProposeRevert(sdk.WrapSDKContext(ctx), &types.MsgProposeRevert{
		Proposer:       "guardian2",
		RevertToHeight: 0,
		Justification:  "genesis revert",
	})
	// Should fail: either because depth > max, or height 0 validation, or ValidateBasic
	// (ValidateBasic requires height > 0)
	if err == nil {
		t.Error("OC-EM-7: revert to block 0 should fail — genesis revert not allowed")
	}

	// Revert within allowed depth — should succeed
	safeTarget := highBlock - int64(maxDepth) + 1
	if safeTarget > 0 {
		_, err = msgServer.ProposeRevert(sdk.WrapSDKContext(ctx), &types.MsgProposeRevert{
			Proposer:       "guardian3",
			RevertToHeight: uint64(safeTarget),
			Justification:  "within bounds",
		})
		if err != nil {
			t.Errorf("OC-EM-7: revert within depth limit should succeed: %v", err)
		}
	}
}

// OC-EM-8: Resume from Normal State — Status is "normal". Guardian proposes
// resume. Verify ErrHaltRequired.
func TestOC_EM8_ResumeFromNormalState(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Status is normal (default)
	if k.GetEmergencyStatus(ctx) != types.StatusNormal {
		t.Fatal("expected normal status")
	}

	_, err := msgServer.ProposeResume(sdk.WrapSDKContext(ctx), &types.MsgProposeResume{
		Proposer: "guardian1",
	})
	if err == nil {
		t.Error("OC-EM-8: resume from normal state should fail — can resume when not halted")
	}

	// Also try revert from normal state
	_, err = msgServer.ProposeRevert(sdk.WrapSDKContext(ctx), &types.MsgProposeRevert{
		Proposer:       "guardian1",
		RevertToHeight: 50,
		Justification:  "not halted",
	})
	if err == nil {
		t.Error("OC-EM-8: revert from normal state should fail — can revert when not halted")
	}
}

// OC-EM-9: Quorum Manipulation — Start halt ceremony. Incrementally add guardian
// votes and verify quorum advances correctly at the 75% threshold.
func TestOC_EM9_QuorumManipulation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "quorum test",
	})

	// 2 guardians = 240M / 600M = 40% — not enough for 75%
	for _, g := range []string{"guardian1", "guardian2"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: resp.ProposalId, Approve: true,
		})
	}
	ceremony, _ := k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrevote) {
		t.Errorf("OC-EM-9: 40%% quorum should stay in prevote, got %s", ceremony.Phase)
	}

	// 3 guardians = 360M / 600M = 60% — still not enough for 75%
	msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian3", ProposalId: resp.ProposalId, Approve: true,
	})
	ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrevote) {
		t.Errorf("OC-EM-9: 60%% quorum should stay in prevote, got %s", ceremony.Phase)
	}

	// 4 guardians = 480M / 600M = 80% — exceeds 75% threshold
	msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian4", ProposalId: resp.ProposalId, Approve: true,
	})
	ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrecommit) {
		t.Errorf("OC-EM-9: 80%% quorum should advance to precommit, got %s — quorum check broken", ceremony.Phase)
	}

	// Now verify precommit quorum similarly
	// Need all 4 who prevoted to precommit
	for _, g := range []string{"guardian1", "guardian2", "guardian3"} {
		msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
			Voter: g, ProposalId: resp.ProposalId, Approve: true,
		})
	}
	ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase == string(types.PhaseFinalized) {
		t.Log("OC-EM-9: 3 precommits (60%) finalized — checking if 75% is required for precommit too")
	}

	// 4th precommit should finalize
	msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian4", ProposalId: resp.ProposalId, Approve: true,
	})

	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusHalted {
		t.Errorf("OC-EM-9: expected halted after full quorum, got %s — quorum not enforced in precommit", status)
	}
}

// OC-EM-10: Ceremony Timeout Enforcement — Start halt ceremony, don't complete
// voting. Advance past HaltTimeoutBlocks. Verify ceremony status is PhaseFailed.
func TestOC_EM10_CeremonyTimeoutEnforcement(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.ProposeHalt(sdk.WrapSDKContext(ctx), &types.MsgProposeHalt{
		Proposer: "guardian1",
		Reason:   "timeout enforcement test",
	})
	if err != nil {
		t.Fatalf("OC-EM-10: propose failed: %v", err)
	}

	// Add partial votes (not enough for quorum)
	msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian1", ProposalId: resp.ProposalId, Approve: true,
	})
	msgServer.VoteHalt(sdk.WrapSDKContext(ctx), &types.MsgVoteHalt{
		Voter: "guardian2", ProposalId: resp.ProposalId, Approve: true,
	})

	// PrevoteDeadline = startBlock(100) + HaltPrevoteBlocks(11) = 111
	// CheckCeremonyProgress checks currentBlock > PrevoteDeadline for prevote phase timeout.
	// At the deadline boundary (block 111): 111 > 111 is false — still active
	ctx111 := ctx.WithBlockHeight(111)
	k.CheckCeremonyProgress(ctx111, resp.ProposalId)
	ceremony, _ := k.GetCeremony(ctx111, resp.ProposalId)
	if ceremony.Phase == string(types.PhaseFailed) {
		t.Error("OC-EM-10: ceremony should not timeout at prevote deadline boundary (block 111)")
	}

	// Just past prevote deadline (block 112): 112 > 111 is true — should fail
	ctx112 := ctx.WithBlockHeight(112)
	k.CheckCeremonyProgress(ctx112, resp.ProposalId)
	ceremony, _ = k.GetCeremony(ctx112, resp.ProposalId)
	if ceremony.Phase != string(types.PhaseFailed) {
		t.Errorf("OC-EM-10: ceremony should fail after prevote deadline, got %s — attacker can keep ceremony open past deadline", ceremony.Phase)
	}

	// Status should revert to normal (or at least not halted)
	// After failure, need HandleCeremonyFailure to clean up
	status := k.GetEmergencyStatus(ctx.WithBlockHeight(200))
	if status == types.StatusHalted {
		t.Error("OC-EM-10: status should not be halted after ceremony timeout")
	}
}
