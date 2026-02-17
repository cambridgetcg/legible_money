package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/disputes/types"
)

// ============ OpenClaw Adversarial Tests: Disputes Module ============
//
// These tests probe the dispute lifecycle, arbiter voting, evidence commit/reveal,
// and escalation system for authorization bypasses, temporal manipulation,
// state re-entry, and economic exploits.

// OC-DS-1: Double-Vote Injection — Verify an arbiter cannot vote twice.
func TestOC_DS1_DoubleVoteInjection(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	arbiter := testAddr(10)

	bk.balances[challenger] = 999999999999
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	ms := newMsgServer(k)
	resp, err := ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger,
		TargetId:   "fact-1",
		Reason:     "test double vote",
		Stake:      "111000000",
	})
	if err != nil {
		t.Fatalf("OC-DS-1: initiate dispute failed: %v", err)
	}

	// Advance to voting phase
	dispute, _ := k.GetDispute(ctx, resp.DisputeId)
	dispute.Status = string(types.DisputeStatusVoting)
	dispute.Arbiters = []string{arbiter, testAddr(11), testAddr(12)}
	dispute.VotingDeadline = 500
	k.SetDispute(ctx, dispute)

	// First vote — succeeds
	err = k.CastVote(ctx, resp.DisputeId, arbiter, "uphold", "agreed")
	if err != nil {
		t.Fatalf("OC-DS-1: first vote failed: %v", err)
	}

	// Second vote — must fail
	err = k.CastVote(ctx, resp.DisputeId, arbiter, "overturn", "changed mind")
	if err == nil {
		t.Error("OC-DS-1: duplicate vote should fail — attacker can double-count votes")
	}
}

// OC-DS-2: Non-Arbiter Vote — Unauthorized voter injection.
func TestOC_DS2_NonArbiterVote(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	outsider := testAddr(99)

	bk.balances[challenger] = 999999999999
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	ms := newMsgServer(k)
	resp, _ := ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger, TargetId: "fact-1", Reason: "auth test", Stake: "111000000",
	})

	dispute, _ := k.GetDispute(ctx, resp.DisputeId)
	dispute.Status = string(types.DisputeStatusVoting)
	dispute.Arbiters = []string{testAddr(10), testAddr(11), testAddr(12)}
	dispute.VotingDeadline = 500
	k.SetDispute(ctx, dispute)

	// Outsider tries to vote
	err := k.CastVote(ctx, resp.DisputeId, outsider, "uphold", "sneaky")
	if err == nil {
		t.Error("OC-DS-2: non-arbiter vote should fail — attacker can inject unauthorized votes")
	}
}

// OC-DS-3: Vote After Deadline — Temporal bypass attempt.
func TestOC_DS3_VoteAfterDeadline(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	arbiter := testAddr(10)

	bk.balances[challenger] = 999999999999
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	ms := newMsgServer(k)
	resp, _ := ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger, TargetId: "fact-1", Reason: "deadline test", Stake: "111000000",
	})

	dispute, _ := k.GetDispute(ctx, resp.DisputeId)
	dispute.Status = string(types.DisputeStatusVoting)
	dispute.Arbiters = []string{arbiter}
	dispute.VotingDeadline = 150
	k.SetDispute(ctx, dispute)

	// Vote at exact deadline — should succeed (currentHeight <= deadline)
	ctxAtDeadline := ctx.WithBlockHeight(150)
	err := k.CastVote(ctxAtDeadline, resp.DisputeId, arbiter, "uphold", "at deadline")
	// Whether this succeeds depends on > vs >= comparison
	_ = err

	// Vote after deadline — must fail
	ctxPast := ctx.WithBlockHeight(151)
	err = k.CastVote(ctxPast, resp.DisputeId, arbiter, "uphold", "too late")
	if err == nil {
		t.Error("OC-DS-3: vote after deadline should fail — attacker can vote after period ends")
	}
}

// OC-DS-4: Evidence After Deadline — Phase bypass attempt.
func TestOC_DS4_EvidenceAfterDeadline(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)

	bk.balances[challenger] = 999999999999
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	ms := newMsgServer(k)
	resp, _ := ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger, TargetId: "fact-1", Reason: "evidence deadline test", Stake: "111000000",
	})

	dispute, _ := k.GetDispute(ctx, resp.DisputeId)
	// Ensure evidence deadline is in the past
	dispute.EvidenceDeadline = 90
	k.SetDispute(ctx, dispute)

	// Try to commit evidence after deadline — must fail
	_, err := ms.CommitEvidence(sdk.WrapSDKContext(ctx), &types.MsgCommitEvidence{
		Submitter:      challenger,
		DisputeId:      resp.DisputeId,
		CommitmentHash: "deadbeef",
	})
	if err == nil {
		t.Error("OC-DS-4: evidence commit after deadline should fail — attacker can submit late evidence")
	}
}

// OC-DS-5: Unauthorized Submitter Evidence — Evidence masquerading.
func TestOC_DS5_UnauthorizedSubmitterEvidence(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	outsider := testAddr(99)

	bk.balances[challenger] = 999999999999
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	ms := newMsgServer(k)
	resp, _ := ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger, TargetId: "fact-1", Reason: "auth test", Stake: "111000000",
	})

	// Outsider tries to commit evidence — must fail
	_, err := ms.CommitEvidence(sdk.WrapSDKContext(ctx), &types.MsgCommitEvidence{
		Submitter:      outsider,
		DisputeId:      resp.DisputeId,
		CommitmentHash: "deadbeef",
	})
	if err == nil {
		t.Error("OC-DS-5: evidence from non-party should fail — attacker can inject fake evidence")
	}
}

// OC-DS-6: Commitment-Reveal Mismatch — Hash manipulation.
func TestOC_DS6_CommitmentRevealMismatch(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)

	bk.balances[challenger] = 999999999999
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	ms := newMsgServer(k)
	resp, _ := ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger, TargetId: "fact-1", Reason: "hash test", Stake: "111000000",
	})

	// Commit with hash of "evidence1" + nonce "abc"
	content := "evidence1"
	nonce := "abc"
	hash := sha256.Sum256([]byte(content + nonce))
	commitHash := hex.EncodeToString(hash[:])

	_, err := ms.CommitEvidence(sdk.WrapSDKContext(ctx), &types.MsgCommitEvidence{
		Submitter:      challenger,
		DisputeId:      resp.DisputeId,
		CommitmentHash: commitHash,
	})
	if err != nil {
		t.Fatalf("OC-DS-6: commit failed: %v", err)
	}

	// Reveal with different content — must fail
	_, err = ms.RevealEvidence(sdk.WrapSDKContext(ctx), &types.MsgRevealEvidence{
		Submitter: challenger,
		DisputeId: resp.DisputeId,
		Evidence:  []*types.EvidenceItem{{Content: "DIFFERENT_CONTENT"}},
		Nonce:     nonce,
	})
	if err == nil {
		t.Error("OC-DS-6: reveal with mismatched content should fail — attacker can swap evidence post-commit")
	}
}

// OC-DS-7: Bond Insufficiency Attack — Minimum bond bypass.
func TestOC_DS7_BondInsufficiency(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)

	bk.balances[challenger] = 999999999999
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	ms := newMsgServer(k)

	// Try with insufficient stake (below tier 1 minimum)
	_, err := ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger,
		TargetId:   "fact-1",
		Reason:     "cheapskate",
		Stake:      "100", // way below minimum bond
	})
	if err == nil {
		t.Error("OC-DS-7: dispute with insufficient bond should fail — attacker can spam disputes cheaply")
	}

	// Try with zero stake
	_, err = ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger,
		TargetId:   "fact-1",
		Reason:     "free dispute",
		Stake:      "0",
	})
	if err == nil {
		t.Error("OC-DS-7: zero-stake dispute should fail — zero-cost attack vector")
	}

	// Try with negative stake
	_, err = ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger,
		TargetId:   "fact-1",
		Reason:     "negative",
		Stake:      "-100",
	})
	if err == nil {
		t.Error("OC-DS-7: negative stake should fail — input validation bypass")
	}
}

// OC-DS-8: Max Active Disputes Circumvention — Dispute flooding.
func TestOC_DS8_MaxActiveDisputesFlooding(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	defender := testAddr(2)
	setupValidators(sk, testAddr(1), defender)
	setupVerifiedFact(kk, "fact-flood", defender)

	ms := newMsgServer(k)

	params := k.GetParams(ctx)
	maxActive := params.MaxActiveDisputesPerFact

	// Create disputes up to the limit
	for i := uint64(0); i < maxActive; i++ {
		challenger := testAddr(int(100 + i))
		bk.balances[challenger] = 999999999999

		_, err := ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
			Challenger: challenger,
			TargetId:   "fact-flood",
			Reason:     "flood",
			Stake:      "111000000",
		})
		if err != nil {
			t.Fatalf("OC-DS-8: dispute #%d failed: %v", i+1, err)
		}
	}

	// One more should fail
	extra := testAddr(999)
	bk.balances[extra] = 999999999999
	_, err := ms.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: extra,
		TargetId:   "fact-flood",
		Reason:     "overflow",
		Stake:      "111000000",
	})
	if err == nil {
		t.Errorf("OC-DS-8: dispute beyond max active (%d) should fail — attacker can flood disputes", maxActive)
	}
}

// newMsgServer helper wraps keeper into a msg server.
func newMsgServer(k keeper.Keeper) types.LegacyMsgServer {
	return keeper.NewMsgServerImpl(k)
}
