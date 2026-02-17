package keeper_test

// ==================================================================================
// OpenClaw Capture Challenge Adversarial Probe Suite (OC-CC-1 through OC-CC-3)
//
// Attack surfaces tested:
// - Unauthorized resolver (non-governance tries to resolve)
// - Double resolution (resolve already-resolved challenge)
// - Evidence after deadline
// ==================================================================================

import (
	"testing"

	"github.com/legible-money/legible/x/capture_challenge/types"
)

// ==================================================================================
// OC-CC-1: Unauthorized Challenge Resolution
//
// ATTACK: Non-authority address tries to resolve a challenge.
// If unguarded, any user could resolve challenges in their favor.
//
// DEFENSE: ResolveChallenge checks msg.Resolver == k.authority.
// ==================================================================================

func TestOC_CC1_UnauthorizedResolution(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Submit a challenge
	submitResp, err := k.SubmitChallenge(ctx, &types.MsgSubmitChallenge{
		Challenger:        testAuthority,
		Domain:            "physics",
		Evidence:          makeEvidence(3),
		Stake:             "77000000",
		AccusedValidators: []string{"lgmvaloper1abc"},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	// ATTACK: Random user tries to resolve as "upheld" (get reward)
	attackerAddr := "lgm1attacker_addr_paddin"
	_, err = k.ResolveChallenge(ctx, &types.MsgResolveChallenge{
		Resolver:      attackerAddr,
		ChallengeId:   submitResp.ChallengeId,
		Outcome:       "upheld",
		Justification: "trust me bro",
	})
	if err == nil {
		t.Fatal("OC-CC-1 FAIL: unauthorized resolver was able to resolve challenge!")
	}

	// Verify challenge still in original state
	ch, _ := k.GetChallenge(ctx, submitResp.ChallengeId)
	if ch.Status != string(types.ChallengeEvidencePeriod) {
		t.Fatalf("OC-CC-1 FAIL: challenge status changed by unauthorized resolver: %s", ch.Status)
	}

	t.Logf("OC-CC-1 PASS: unauthorized resolver correctly rejected")
}

// ==================================================================================
// OC-CC-2: Double Resolution Attack
//
// ATTACK: Resolve a challenge, then try to resolve it again with different outcome.
// If unguarded, rewards could be double-claimed or outcomes reversed.
//
// DEFENSE: ResolveChallenge checks status is evidence_period or under_review.
// ==================================================================================

func TestOC_CC2_DoubleResolution(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Submit and resolve as rejected
	submitResp, _ := k.SubmitChallenge(ctx, &types.MsgSubmitChallenge{
		Challenger:        testAuthority,
		Domain:            "biology",
		Evidence:          makeEvidence(3),
		Stake:             "77000000",
		AccusedValidators: []string{"lgmvaloper1abc"},
	})

	_, err := k.ResolveChallenge(ctx, &types.MsgResolveChallenge{
		Resolver:      testAuthority,
		ChallengeId:   submitResp.ChallengeId,
		Outcome:       "rejected",
		Justification: "insufficient evidence",
	})
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}

	// ATTACK: Try to re-resolve as upheld (flip outcome for reward)
	_, err = k.ResolveChallenge(ctx, &types.MsgResolveChallenge{
		Resolver:      testAuthority,
		ChallengeId:   submitResp.ChallengeId,
		Outcome:       "upheld",
		Justification: "changed my mind",
	})
	if err == nil {
		t.Fatal("OC-CC-2 FAIL: double resolution allowed! Outcome could be reversed.")
	}

	// Verify original outcome preserved
	ch, _ := k.GetChallenge(ctx, submitResp.ChallengeId)
	if ch.Status != string(types.ChallengeRejected) {
		t.Fatalf("OC-CC-2 FAIL: original outcome was modified: %s", ch.Status)
	}

	t.Logf("OC-CC-2 PASS: double resolution correctly rejected")
}

// ==================================================================================
// OC-CC-3: Evidence Submission by Non-Challenger
//
// ATTACK: Someone other than the original challenger tries to add evidence.
// If unguarded, attackers could pollute evidence with misleading data.
//
// DEFENSE: AddEvidence checks msg.Challenger == challenge.Challenger.
// ==================================================================================

func TestOC_CC3_EvidenceByNonChallenger(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	submitResp, err := k.SubmitChallenge(ctx, &types.MsgSubmitChallenge{
		Challenger:        testAuthority,
		Domain:            "chemistry",
		Evidence:          makeEvidence(3),
		Stake:             "77000000",
		AccusedValidators: []string{"lgmvaloper1abc"},
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	// ATTACK: Impersonator tries to add evidence
	impersonator := "lgm1impersonator_pad___"
	_, err = k.AddEvidence(ctx, &types.MsgAddEvidence{
		ChallengeId: submitResp.ChallengeId,
		Challenger:  impersonator,
		Evidence: &types.CaptureEvidence{
			Id:          "ev-fake",
			Type:        "fabricated",
			Description: "Fake evidence to sway outcome",
			DataHash:    "fake_hash",
			SubmittedAt: 10,
			SubmittedBy: impersonator,
		},
	})
	if err == nil {
		t.Fatal("OC-CC-3 FAIL: non-challenger was able to add evidence!")
	}

	// Verify evidence count unchanged
	ch, _ := k.GetChallenge(ctx, submitResp.ChallengeId)
	if len(ch.Evidence) != 3 {
		t.Fatalf("OC-CC-3 FAIL: evidence count changed: %d (expected 3)", len(ch.Evidence))
	}

	t.Logf("OC-CC-3 PASS: non-challenger evidence submission correctly rejected")
}
