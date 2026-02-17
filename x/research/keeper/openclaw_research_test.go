package keeper_test

// ==================================================================================
// OpenClaw Research Adversarial Probe Suite (OC-RES-1 through OC-RES-4)
//
// Attack surfaces tested:
// - Double review by same reviewer
// - Unauthorized research resolution
// - Challenge on already-accepted research
// - Bounty claim after expiry
// ==================================================================================

import (
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/legible-money/legible/x/research/types"
)

// ==================================================================================
// OC-RES-1: Double Review Attack
//
// ATTACK: Reviewer submits a review, then submits again to double their influence.
// If unguarded, a single reviewer could dominate the outcome.
//
// DEFENSE: ReviewResearch checks if reviewer already reviewed.
// ==================================================================================

func TestOC_RES1_DoubleReview(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	reviewer := testAddr("rev1")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:   submitter,
		Title:       "Test Research",
		Description: "Desc",
		Domain:      "physics",
		Stake:       "100000000",
	})

	// First review
	_, err := ms.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer:     reviewer,
		ResearchId:   "RES-1",
		Verdict:      types.ReviewVerdict_REVIEW_VERDICT_APPROVE,
		QualityScore: 80,
		Reasoning:    "Good",
	})
	if err != nil {
		t.Fatalf("first review: %v", err)
	}

	// ATTACK: Same reviewer submits again
	_, err = ms.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer:     reviewer,
		ResearchId:   "RES-1",
		Verdict:      types.ReviewVerdict_REVIEW_VERDICT_REJECT,
		QualityScore: 10,
		Reasoning:    "Changed mind",
	})
	if err == nil {
		t.Fatal("OC-RES-1 FAIL: double review allowed!")
	}

	// Verify original review preserved
	r, _ := k.GetResearch(ctx, "RES-1")
	if r.ReviewCount != 1 {
		t.Fatalf("OC-RES-1 FAIL: review count changed: %d", r.ReviewCount)
	}
	if r.AggregateScore != 80 {
		t.Fatalf("OC-RES-1 FAIL: aggregate score changed: %d", r.AggregateScore)
	}

	t.Logf("OC-RES-1 PASS: double review correctly rejected")
}

// ==================================================================================
// OC-RES-2: Unauthorized Research Resolution
//
// ATTACK: Non-authority address tries to resolve a research submission.
// If unguarded, anyone could accept/reject research and control stake distribution.
//
// DEFENSE: ResolveResearch checks msg.Authority == k.authority.
// ==================================================================================

func TestOC_RES2_UnauthorizedResolution(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	bk.fundAccount(submitter, "ulgm", largeBalance)

	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:   submitter,
		Title:       "Research",
		Description: "Desc",
		Domain:      "physics",
		Stake:       "100000000",
	})

	// Add enough reviews to make resolvable
	for i := 0; i < 3; i++ {
		ms.ReviewResearch(ctx, &types.MsgReviewResearch{
			Reviewer:     testAddr("rev" + string(rune('a'+i))),
			ResearchId:   "RES-1",
			Verdict:      types.ReviewVerdict_REVIEW_VERDICT_APPROVE,
			QualityScore: 80,
			Reasoning:    "Good",
		})
	}

	// ATTACK: Random user tries to resolve
	_, err := ms.ResolveResearch(ctx, &types.MsgResolveResearch{
		Authority:  testAddr("rando"),
		ResearchId: "RES-1",
	})
	if err == nil {
		t.Fatal("OC-RES-2 FAIL: unauthorized user was able to resolve research!")
	}

	// Verify still under review
	r, _ := k.GetResearch(ctx, "RES-1")
	if r.Status == string(types.ResearchStatusAccepted) || r.Status == string(types.ResearchStatusRejected) {
		t.Fatalf("OC-RES-2 FAIL: research was resolved by unauthorized party: %s", r.Status)
	}

	t.Logf("OC-RES-2 PASS: unauthorized resolution correctly rejected")
}

// ==================================================================================
// OC-RES-3: Challenge on Already-Accepted Research
//
// ATTACK: Research is accepted, then someone tries to challenge it.
// If unguarded, accepted research could be put back into dispute.
//
// DEFENSE: ChallengeResearch checks status is challengeable (submitted/under_review).
// ==================================================================================

func TestOC_RES3_ChallengeAcceptedResearch(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	submitter := testAddr("alice")
	challenger := testAddr("bob")
	bk.fundAccount(submitter, "ulgm", largeBalance)
	bk.fundAccount(challenger, "ulgm", largeBalance)

	// Submit and manually accept
	ms.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:   submitter,
		Title:       "Research",
		Description: "Desc",
		Domain:      "physics",
		Stake:       "100000000",
	})
	r, _ := k.GetResearch(ctx, "RES-1")
	r.Status = string(types.ResearchStatusAccepted)
	k.SetResearch(ctx, r)

	// ATTACK: Challenge accepted research
	_, err := ms.ChallengeResearch(ctx, &types.MsgChallengeResearch{
		Challenger: challenger,
		ResearchId: "RES-1",
		Reason:     "Retroactive challenge",
		Stake:      "100000000",
	})
	if err == nil {
		t.Fatal("OC-RES-3 FAIL: challenging accepted research allowed!")
	}

	// Verify status unchanged
	r, _ = k.GetResearch(ctx, "RES-1")
	if r.Status != string(types.ResearchStatusAccepted) {
		t.Fatalf("OC-RES-3 FAIL: accepted status changed to: %s", r.Status)
	}

	t.Logf("OC-RES-3 PASS: challenge on accepted research correctly rejected")
}

// ==================================================================================
// OC-RES-4: Bounty Claim After Expiry
//
// ATTACK: Bounty deadline passes, then someone tries to claim it.
// If unguarded, expired bounties could still be claimed and funds extracted.
//
// DEFENSE: ClaimBounty checks currentBlock <= bounty.DeadlineHeight.
// ==================================================================================

func TestOC_RES4_BountyClaimAfterExpiry(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := newMsgServer(k)

	creator := testAddr("alice")
	claimer := testAddr("bob")
	bk.fundAccount(creator, "ulgm", sdkmath.NewIntWithDecimal(1, 24))

	ms.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator:        creator,
		Title:          "Find error",
		Description:    "Desc",
		Reward:         "1000000000",
		DeadlineHeight: 100 + 34272 + 1,
	})

	// Advance past deadline
	expiredCtx := ctx.WithBlockHeight(100 + 34272 + 2)

	// ATTACK: Claim expired bounty
	_, err := ms.ClaimBounty(expiredCtx, &types.MsgClaimBounty{
		Claimer:  claimer,
		BountyId: "BOUNTY-1",
	})
	if err == nil {
		t.Fatal("OC-RES-4 FAIL: claiming expired bounty allowed!")
	}

	// Verify bounty status unchanged (still open, not claimed)
	b, _ := k.GetBounty(ctx, "BOUNTY-1")
	if b.ClaimedBy != "" {
		t.Fatalf("OC-RES-4 FAIL: expired bounty was claimed by: %s", b.ClaimedBy)
	}

	t.Logf("OC-RES-4 PASS: expired bounty claim correctly rejected")
}
