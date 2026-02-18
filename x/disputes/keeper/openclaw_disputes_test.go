package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/disputes/keeper"
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

	bk.fundAccount(challenger, sdkmath.NewIntWithDecimal(1, 12))
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
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

	bk.fundAccount(challenger, sdkmath.NewIntWithDecimal(1, 12))
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	srv := keeper.NewMsgServerImpl(k)
	resp, _ := srv.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger, TargetId: "fact-1", Reason: "auth test", Stake: "111000000",
	})

	dispute, _ := k.GetDispute(ctx, resp.DisputeId)
	dispute.Status = string(types.DisputeStatusVoting)
	dispute.Arbiters = []string{testAddr(10), testAddr(11), testAddr(12)}
	dispute.VotingDeadline = 500
	k.SetDispute(ctx, dispute)

	// Outsider tries to vote — must fail
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

	bk.fundAccount(challenger, sdkmath.NewIntWithDecimal(1, 12))
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	srv := keeper.NewMsgServerImpl(k)
	resp, _ := srv.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger, TargetId: "fact-1", Reason: "deadline test", Stake: "111000000",
	})

	dispute, _ := k.GetDispute(ctx, resp.DisputeId)
	dispute.Status = string(types.DisputeStatusVoting)
	dispute.Arbiters = []string{arbiter}
	dispute.VotingDeadline = 150
	k.SetDispute(ctx, dispute)

	// Vote after deadline — must fail
	ctxPast := ctx.WithBlockHeight(151)
	err := k.CastVote(ctxPast, resp.DisputeId, arbiter, "uphold", "too late")
	if err == nil {
		t.Error("OC-DS-3: vote after deadline should fail — attacker can vote after period ends")
	}
}

// OC-DS-4: Evidence After Deadline — Phase bypass attempt.
func TestOC_DS4_EvidenceAfterDeadline(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)

	bk.fundAccount(challenger, sdkmath.NewIntWithDecimal(1, 12))
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	srv := keeper.NewMsgServerImpl(k)
	resp, _ := srv.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger, TargetId: "fact-1", Reason: "evidence deadline test", Stake: "111000000",
	})

	// Set evidence deadline in the past
	dispute, _ := k.GetDispute(ctx, resp.DisputeId)
	dispute.EvidenceDeadline = 90
	k.SetDispute(ctx, dispute)

	// Try to commit evidence after deadline — must fail
	_, err := srv.CommitEvidence(sdk.WrapSDKContext(ctx), &types.MsgCommitEvidence{
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

	bk.fundAccount(challenger, sdkmath.NewIntWithDecimal(1, 12))
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	srv := keeper.NewMsgServerImpl(k)
	resp, _ := srv.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger, TargetId: "fact-1", Reason: "auth test", Stake: "111000000",
	})

	// Outsider tries to commit evidence — must fail
	_, err := srv.CommitEvidence(sdk.WrapSDKContext(ctx), &types.MsgCommitEvidence{
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

	bk.fundAccount(challenger, sdkmath.NewIntWithDecimal(1, 12))
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	srv := keeper.NewMsgServerImpl(k)
	resp, _ := srv.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger, TargetId: "fact-1", Reason: "hash test", Stake: "111000000",
	})

	// Commit with hash of "evidence1" + nonce "abc"
	content := "evidence1"
	nonce := "abc"
	hash := sha256.Sum256([]byte(content + nonce))
	commitHash := hex.EncodeToString(hash[:])

	_, err := srv.CommitEvidence(sdk.WrapSDKContext(ctx), &types.MsgCommitEvidence{
		Submitter:      challenger,
		DisputeId:      resp.DisputeId,
		CommitmentHash: commitHash,
	})
	if err != nil {
		t.Fatalf("OC-DS-6: commit failed: %v", err)
	}

	// Reveal with different content — must fail
	_, err = srv.RevealEvidence(sdk.WrapSDKContext(ctx), &types.MsgRevealEvidence{
		Submitter: challenger,
		DisputeId: resp.DisputeId,
		Evidence:  []*types.DisputeEvidence{{Content: "DIFFERENT_CONTENT"}},
		Nonce:     nonce,
	})
	if err == nil {
		t.Error("OC-DS-6: reveal with mismatched content should fail — attacker can swap evidence post-commit")
	}
}

// OC-DS-7: Bond Insufficiency Attack — Minimum bond bypass.
func TestOC_DS7_BondInsufficiency(t *testing.T) {
	_, ctx, bk, kk, sk := setupKeeper(t)
	k2, _, _, _, _ := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)

	bk.fundAccount(challenger, sdkmath.NewIntWithDecimal(1, 12))
	setupValidators(sk, challenger, defender)
	setupVerifiedFact(kk, "fact-1", defender)

	srv := keeper.NewMsgServerImpl(k2)
	_ = srv
	_ = ctx

	// Try with zero stake — must fail (invalid bond)
	srv2 := keeper.NewMsgServerImpl(k2)
	_, err := srv2.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: challenger,
		TargetId:   "fact-1",
		Reason:     "free dispute",
		Stake:      "0",
	})
	if err == nil {
		t.Error("OC-DS-7: zero-stake dispute should fail — zero-cost attack vector")
	}

	// Try with negative stake — must fail
	_, err = srv2.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
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

	srv := keeper.NewMsgServerImpl(k)

	params := k.GetParams(ctx)
	maxActive := params.MaxActiveDisputes

	// Create disputes up to the limit
	for i := uint32(0); i < maxActive; i++ {
		challenger := testAddr(int(100 + i))
		bk.fundAccount(challenger, sdkmath.NewIntWithDecimal(1, 12))

		_, err := srv.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
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
	bk.fundAccount(extra, sdkmath.NewIntWithDecimal(1, 12))
	_, err := srv.InitiateDispute(sdk.WrapSDKContext(ctx), &types.MsgInitiateDispute{
		Challenger: extra,
		TargetId:   "fact-flood",
		Reason:     "overflow",
		Stake:      "111000000",
	})
	if err == nil {
		t.Errorf("OC-DS-8: dispute beyond max active (%d) should fail — attacker can flood disputes", maxActive)
	}
}

// ==================================================================================
// OCD-2: Arbiter Collusion — All Vote to Overturn
// ==================================================================================
// Scenario: 3 arbiters collude to overturn a legitimate fact, triggering
// confidence reduction and vesting clawback for the original submitter.
func TestOCD2_ArbiterCollusion(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	factId := "fact-collude"
	challenger := testAddr(1)
	defender := testAddr(2)
	setupVerifiedFact(kk, factId, defender)
	setupValidators(sk, challenger, defender)
	bk.fundAccount(challenger, sdkmath.NewInt(200_000_000))

	dispute, err := k.OpenDispute(ctx, factId, challenger, "Collusion test", "111000000")
	if err != nil {
		t.Fatalf("failed to open dispute: %v", err)
	}

	// Transition to voting
	dispute.Status = string(types.DisputeStatusVoting)
	dispute.VotingDeadline = 500
	k.SetDispute(ctx, dispute)

	// All 3 arbiters collude to overturn
	for _, arbiter := range dispute.Arbiters {
		// Fund staking keeper mock for effective stake
		sk.validators = append(sk.validators, types.ValidatorInfo{
			Address: arbiter, Stake: 5000, Tier: 2,
		})
		err := k.CastVote(ctx, dispute.Id, arbiter, "overturn", "Colluding to overturn")
		if err != nil {
			t.Logf("OCD-2 VOTE: arbiter %s failed: %v", arbiter[:20], err)
		}
	}

	// Tally
	outcome, err := k.TallyVotes(ctx, dispute)
	if err != nil {
		t.Fatalf("tally failed: %v", err)
	}

	// === INSIGHT ANALYSIS ===
	t.Logf("OCD-2 OUTCOME: %s", outcome)
	if outcome == types.OutcomeOverturned {
		t.Log("OCD-2 FINDING: 3 colluding arbiters successfully overturn verified fact")
		t.Log("  ECONOMIC IMPACT:")
		t.Log("    - Challenger gets full bond + 15% of slash back")
		t.Log("    - Fact confidence reduced by 55%")
		t.Log("    - All referencing facts cascade 22% confidence reduction")
		t.Log("    - Original submitter's vesting clawed back (33% released, 100% unvested)")
		t.Log("  DEFENSE: Arbiter selection is pseudo-random (SHA256 seed from block hash)")
		t.Log("  DEFENSE: Escalation to tier 2 (7 arbiters) or tier 3 (15) dilutes collusion")
		t.Log("  RESIDUAL RISK: With only 3 arbiters at tier 1, collusion is economically feasible")
	}
}

// ==================================================================================
// OCD-3: Escalation as Delay Weapon
// ==================================================================================
// Scenario: Challenger opens dispute, waits for escalation delay, then escalates.
// This resets evidence + voting deadlines, creating indefinite delay.
func TestOCD3_EscalationAsDelayWeapon(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	factId := "fact-delay"
	challenger := testAddr(1)
	defender := testAddr(2)
	setupVerifiedFact(kk, factId, defender)
	setupValidators(sk, challenger, defender)
	bk.fundAccount(challenger, sdkmath.NewInt(1_000_000_000))

	dispute, err := k.OpenDispute(ctx, factId, challenger, "Delay tactic", "111000000")
	if err != nil {
		t.Fatalf("failed to open dispute: %v", err)
	}

	originalEvDeadline := dispute.EvidenceDeadline
	t.Logf("OCD-3 INITIAL: evidenceDeadline=%d tier=%d", originalEvDeadline, dispute.Tier)

	// Wait for escalation delay
	params := k.GetParams(ctx)
	escalationDelay := params.EscalationDelay
	t.Logf("OCD-3 ESCALATION DELAY: %d blocks", escalationDelay)

	// Escalate: tier 1 → tier 2
	escalateCtx := ctx.WithBlockHeight(int64(100 + escalationDelay + 1))
	err = k.EscalateDispute(escalateCtx, dispute.Id, challenger, "555000000")
	if err != nil {
		t.Logf("OCD-3 ESCALATION: failed — %v", err)
	} else {
		dispute, _ = k.GetDispute(escalateCtx, dispute.Id)
		t.Logf("OCD-3 AFTER ESCALATION: tier=%d evidenceDeadline=%d status=%s",
			dispute.Tier, dispute.EvidenceDeadline, dispute.Status)
	}

	// === ECONOMIC ANALYSIS ===
	tier1Bond := int64(111_000_000) // 111 LGM
	tier2Bond := int64(555_000_000) // 555 LGM
	tier3Bond := int64(1_111_000_000)

	t.Logf("OCD-3 ECONOMICS:")
	t.Logf("  Tier 1 bond: %d ulgm (111 LGM), evidence+voting: ~3 days", tier1Bond)
	t.Logf("  Tier 2 bond: %d ulgm (555 LGM), evidence+voting: ~6 days", tier2Bond)
	t.Logf("  Tier 3 bond: %d ulgm (1111 LGM), evidence+voting: ~10 days", tier3Bond)
	t.Logf("  Total delay (all 3 tiers): ~19 days")
	t.Logf("  Total bond: %d ulgm (1777 LGM)", tier1Bond+tier2Bond+tier3Bond)
	t.Log("OCD-3 INSIGHT: Escalation resets deadlines, creating extended lockup.")
	t.Log("  Attacker pays escalating bonds but can lock a fact for ~3 weeks.")
	t.Log("  DEFENSE: Each escalation requires additional bond (111 → 555 → 1111 LGM)")
	t.Log("  DEFENSE: If fact is upheld, challenger loses 50% of all bonds")
	t.Log("  COST OF GRIEFING: ~888 LGM (50% of 1777) for 19 days of lockup")
}

// ==================================================================================
// OCD-4: Bond Recovery Arbitrage
// ==================================================================================
// Scenario: Challenge a fact that is already "contested" (low confidence).
// If overturned, challenger gets bond + reward. If upheld, challenger loses 50%.
// Expected value calculation: is it profitable to challenge low-confidence facts?
func TestOCD4_BondRecoveryArbitrage(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	setupValidators(sk, challenger, defender)
	bk.fundAccount(challenger, sdkmath.NewInt(200_000_000))

	// Set up a low-confidence "contested" fact
	kk.facts["fact-weak"] = types.Fact{
		Id:         "fact-weak",
		ClaimText:  "Controversial claim",
		Domain:     "physics",
		Status:     "contested",
		Confidence: 400_000, // 40% — below typical threshold
		Submitter:  defender,
	}

	dispute, err := k.OpenDispute(ctx, "fact-weak", challenger, "This fact is weak", "111000000")
	if err != nil {
		t.Fatalf("failed to open dispute: %v", err)
	}

	bond := int64(111_000_000)
	params := k.GetParams(ctx)
	slashRate := int64(params.SlashRateLoser)
	rewardRate := int64(params.RewardRateWinner)

	// === ECONOMIC ANALYSIS ===
	slash := bond * slashRate / 1_000_000
	winnerReward := slash * rewardRate / 1_000_000
	lossChallengerUpheld := slash
	gainChallengerOverturned := winnerReward

	t.Logf("OCD-4 BOND ECONOMICS (dispute %s):", dispute.Id)
	t.Logf("  Bond: %d ulgm (%.0f LGM)", bond, float64(bond)/1_000_000)
	t.Logf("  If UPHELD (challenger loses):  -%d ulgm (%.1f LGM)", lossChallengerUpheld, float64(lossChallengerUpheld)/1_000_000)
	t.Logf("  If OVERTURNED (challenger wins): +%d ulgm (%.1f LGM)", gainChallengerOverturned, float64(gainChallengerOverturned)/1_000_000)

	// Expected value at 50/50 probability
	ev := (gainChallengerOverturned - lossChallengerUpheld) / 2
	t.Logf("  Expected value (50/50): %d ulgm", ev)

	if ev > 0 {
		t.Log("OCD-4 FINDING: Positive EV — challenging any fact is profitable at 50% success rate")
	} else {
		t.Log("OCD-4 FINDING: Negative EV — challenger needs >50% success rate to profit")
		breakeven := float64(lossChallengerUpheld) / float64(lossChallengerUpheld+gainChallengerOverturned) * 100
		t.Logf("  Breakeven win rate: %.1f%%", breakeven)
	}
	t.Log("  INSIGHT: SlashRateLoser (50%) >> RewardRateWinner (30% of slash = 15% of bond)")
	t.Log("  Challenger risks 50% to gain 15% — needs ~77% win rate to break even.")
	t.Log("  DEFENSE: High asymmetry deters speculative challenges.")
}

// ==================================================================================
// OCD-5: Arbiter Abstention to Force Inconclusive
// ==================================================================================
// Scenario: 1 of 3 arbiters abstains, preventing the 77% majority threshold.
// With 2 votes: uphold+overturn → 50% < 77% → inconclusive.
func TestOCD5_ArbiterAbstentionForcesInconclusive(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	arbiter1 := testAddr(10)
	arbiter2 := testAddr(11)
	arbiter3 := testAddr(12)

	dispute := types.Dispute{
		Id:             "dispute-abstain",
		FactId:         "fact-001",
		Status:         string(types.DisputeStatusVoting),
		Tier:           uint32(types.TierBasic),
		VotingDeadline: 500,
		Arbiters:       []string{arbiter1, arbiter2, arbiter3},
	}
	k.SetDispute(ctx, dispute)

	// Arbiter 1: uphold, Arbiter 2: overturn, Arbiter 3: abstain
	k.SetVote(ctx, types.DisputeVote{DisputeId: dispute.Id, Arbiter: arbiter1, Vote: "uphold", Stake: "5000"})
	k.SetVote(ctx, types.DisputeVote{DisputeId: dispute.Id, Arbiter: arbiter2, Vote: "overturn", Stake: "5000"})
	k.SetVote(ctx, types.DisputeVote{DisputeId: dispute.Id, Arbiter: arbiter3, Vote: "abstain", Stake: "5000"})

	outcome, err := k.TallyVotes(ctx, dispute)
	if err != nil {
		t.Fatalf("tally failed: %v", err)
	}

	t.Logf("OCD-5 OUTCOME: %s", outcome)
	t.Log("OCD-5 ANALYSIS:")
	t.Log("  Votes: uphold=5000, overturn=5000, abstain=5000")
	t.Log("  Quorum: 3/3 voted (100%) >= 66% threshold ✓")
	t.Log("  Majority: uphold/(uphold+overturn) = 50% < 77% threshold → inconclusive")

	if outcome == types.OutcomeInconclusive {
		t.Log("OCD-5 FINDING: Single abstention + split creates inconclusive")
		t.Log("  ECONOMIC IMPACT:")
		t.Log("    - Challenger gets full bond back (no penalty!)")
		t.Log("    - Fact moved to 'contested' status")
		t.Log("    - No rewards/slashes for arbiters")
		t.Log("  RESIDUAL RISK: Challenger can repeatedly force inconclusive at zero net cost")
	}
}

// ==================================================================================
// OCD-6: Cascade Attack — Overturn Foundation Fact
// ==================================================================================
// Scenario: Overturn a fact that other facts reference. The 22% confidence
// cascade could bring multiple facts below the revocation threshold (10%).
func TestOCD6_CascadeAttack(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	challenger := testAddr(1)
	defender := testAddr(2)
	setupValidators(sk, challenger, defender)
	bk.fundAccount(challenger, sdkmath.NewInt(200_000_000))

	// Set up fact network: foundation → child1, child2, child3
	kk.facts["fact-foundation"] = types.Fact{
		Id: "fact-foundation", ClaimText: "Foundation claim",
		Domain: "physics", Status: "verified", Confidence: 800_000, Submitter: defender,
	}
	for i := 1; i <= 3; i++ {
		kk.facts[fmt.Sprintf("fact-child-%d", i)] = types.Fact{
			Id:         fmt.Sprintf("fact-child-%d", i),
			ClaimText:  fmt.Sprintf("Child claim %d referencing foundation", i),
			Domain:     "physics",
			Status:     "verified",
			Confidence: 600_000, // 60% — cascade of 22% → 468,000 (46.8%)
			Submitter:  defender,
		}
	}

	// Open dispute against foundation
	dispute, err := k.OpenDispute(ctx, "fact-foundation", challenger, "Foundation is wrong", "111000000")
	if err != nil {
		t.Fatalf("failed to open dispute: %v", err)
	}

	// Force overturn for cascade test
	dispute.Status = string(types.DisputeStatusVoting)
	dispute.VotingDeadline = 500
	k.SetDispute(ctx, dispute)

	// All arbiters vote overturn
	for _, arbiter := range dispute.Arbiters {
		k.SetVote(ctx, types.DisputeVote{
			DisputeId: dispute.Id, Arbiter: arbiter, Vote: "overturn", Stake: "5000",
		})
	}

	outcome, _ := k.TallyVotes(ctx, dispute)
	t.Logf("OCD-6 TALLY: %s", outcome)

	if outcome == types.OutcomeOverturned {
		// Resolve with cascade
		err = k.ResolveDispute(ctx, dispute, outcome)
		if err != nil {
			t.Logf("OCD-6 RESOLVE: %v", err)
		}

		// Check cascade damage
		foundation, _ := kk.GetFact(ctx, "fact-foundation")
		t.Logf("OCD-6 CASCADE:")
		t.Logf("  Foundation: confidence %d → %d (status=%s)", 800_000, foundation.Confidence, foundation.Status)

		for i := 1; i <= 3; i++ {
			child, _ := kk.GetFact(ctx, fmt.Sprintf("fact-child-%d", i))
			t.Logf("  Child %d: confidence %d (status=%s)", i, child.Confidence, child.Status)
		}

		t.Log("OCD-6 INSIGHT: Overturning a foundation fact cascades 22% reduction to dependents.")
		t.Log("  Facts near the revocation threshold (10%) can be revoked by cascading.")
		t.Log("  DEFENSE: Cascade is limited to direct references (no recursive cascade).")
		t.Log("  DEFENSE: 111 LGM bond required per dispute (expensive for multi-fact attack).")
	}
}

// ==================================================================================
// NCD-1: Bond Slash Integer Truncation
// ==================================================================================
// Scenario: With SlashRateLoser=500000 (50%) and a very small bond, does
// integer truncation create a zero-slash scenario?
func TestNCD1_BondSlashIntegerTruncation(t *testing.T) {
	params := types.DefaultGenesis().Params
	slashRate := int64(params.SlashRateLoser) // 500000 = 50%

	bonds := []int64{1, 2, 3, 100, 999, 1000, 1001}

	for _, bond := range bonds {
		slash := bond * slashRate / 1_000_000
		winnerReward := slash * int64(params.RewardRateWinner) / 1_000_000
		arbiterReward := slash * int64(params.ArbiterRewardRate) / 1_000_000
		burn := slash - winnerReward - arbiterReward

		t.Logf("NCD-1: bond=%d → slash=%d reward=%d arbiter=%d burn=%d",
			bond, slash, winnerReward, arbiterReward, burn)

		if slash == 0 && bond > 0 {
			t.Logf("  TRUNCATION: bond=%d produces zero slash at %d bps rate", bond, slashRate)
		}
		if burn < 0 {
			t.Errorf("  CRITICAL: negative burn at bond=%d — reward+arbiter > slash", bond)
		}
	}

	t.Log("NCD-1 INSIGHT: At 50% slash rate, bond=1 → slash=0 (truncated).")
	t.Log("  DEFENSE: MinBond is 111 LGM (111,000,000 ulgm) — well above truncation zone.")
	t.Log("  At min bond: slash=55,500,000, reward=16,650,000, burn=11,100,000.")
}

// ==================================================================================
// NCD-2: Quorum Boundary — Exactly 66%
// ==================================================================================
// Scenario: Test the quorum threshold at exact boundary. With 3 arbiters,
// 2/3 = 66.67% > 66% (passes), but what about exact 66%?
func TestNCD2_QuorumBoundary(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// With 3 arbiters and 66% quorum, need 2/3 votes (66.7%)
	// Test: only 1 of 3 votes → 33.3% < 66% → should fail quorum
	dispute := types.Dispute{
		Id:             "dispute-quorum",
		FactId:         "fact-q",
		Status:         string(types.DisputeStatusVoting),
		Tier:           uint32(types.TierBasic),
		Arbiters:       []string{testAddr(10), testAddr(11), testAddr(12)},
		VotingDeadline: 500,
	}
	k.SetDispute(ctx, dispute)

	// Only 1 arbiter votes
	k.SetVote(ctx, types.DisputeVote{
		DisputeId: dispute.Id, Arbiter: testAddr(10), Vote: "uphold", Stake: "5000",
	})

	outcome, err := k.TallyVotes(ctx, dispute)
	t.Logf("NCD-2 ONE VOTER: outcome=%s err=%v", outcome, err)

	// Now test 2 of 3 (should meet quorum)
	k.SetVote(ctx, types.DisputeVote{
		DisputeId: dispute.Id, Arbiter: testAddr(11), Vote: "uphold", Stake: "5000",
	})

	outcome2, err2 := k.TallyVotes(ctx, dispute)
	t.Logf("NCD-2 TWO VOTERS: outcome=%s err=%v", outcome2, err2)

	if outcome == types.OutcomeInconclusive && outcome2 == types.OutcomeUpheld {
		t.Log("NCD-2 BOUNDARY: 1/3 → inconclusive, 2/3 → upheld (correct quorum behavior)")
	}

	// Edge: test with 7 arbiters (tier 2). 5/7 = 71.4% > 66%, 4/7 = 57.1% < 66%
	dispute2 := types.Dispute{
		Id:       "dispute-quorum-7",
		FactId:   "fact-q2",
		Status:   string(types.DisputeStatusVoting),
		Tier:     uint32(types.TierEscalated),
		Arbiters: []string{testAddr(20), testAddr(21), testAddr(22), testAddr(23), testAddr(24), testAddr(25), testAddr(26)},
		VotingDeadline: 500,
	}
	k.SetDispute(ctx, dispute2)

	// 4 of 7 vote → 57.1% < 66%
	for i := 20; i < 24; i++ {
		k.SetVote(ctx, types.DisputeVote{
			DisputeId: dispute2.Id, Arbiter: testAddr(i), Vote: "uphold", Stake: "5000",
		})
	}
	outcome3, _ := k.TallyVotes(ctx, dispute2)
	t.Logf("NCD-2 TIER 2 (4/7 = 57.1%%): outcome=%s", outcome3)

	// 5 of 7 vote → 71.4% > 66%
	k.SetVote(ctx, types.DisputeVote{
		DisputeId: dispute2.Id, Arbiter: testAddr(24), Vote: "uphold", Stake: "5000",
	})
	outcome4, _ := k.TallyVotes(ctx, dispute2)
	t.Logf("NCD-2 TIER 2 (5/7 = 71.4%%): outcome=%s", outcome4)
}

// ==================================================================================
// ZCD-1: Inconclusive Loop — Repeated Free Challenges
// ==================================================================================
// Scenario: Challenger forces inconclusive outcomes (via abstaining arbiters)
// and gets full bond back each time, effectively locking the fact at zero cost.
func TestZCD1_InconclusiveLoop(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	factId := "fact-looped"
	challenger := testAddr(1)
	defender := testAddr(2)
	setupVerifiedFact(kk, factId, defender)
	setupValidators(sk, challenger, defender)
	bk.fundAccount(challenger, sdkmath.NewInt(200_000_000))

	// First challenge
	dispute1, err := k.OpenDispute(ctx.WithBlockHeight(100), factId, challenger, "Round 1", "111000000")
	if err != nil {
		t.Fatalf("dispute 1 failed: %v", err)
	}

	// Force inconclusive
	dispute1.Status = string(types.DisputeStatusVoting)
	dispute1.VotingDeadline = 500
	k.SetDispute(ctx, dispute1)

	for _, a := range dispute1.Arbiters[:2] {
		k.SetVote(ctx, types.DisputeVote{DisputeId: dispute1.Id, Arbiter: a, Vote: "uphold", Stake: "5000"})
	}
	if len(dispute1.Arbiters) > 2 {
		k.SetVote(ctx, types.DisputeVote{DisputeId: dispute1.Id, Arbiter: dispute1.Arbiters[2], Vote: "overturn", Stake: "5000"})
	}

	// With 2 uphold + 1 overturn: uphold/(uphold+overturn)=66.7% < 77% → inconclusive
	outcome, _ := k.TallyVotes(ctx, dispute1)
	t.Logf("ZCD-1 ROUND 1: outcome=%s", outcome)

	// Resolve as inconclusive
	if outcome == types.OutcomeInconclusive {
		err = k.ResolveDispute(ctx, dispute1, outcome)
		if err != nil {
			t.Logf("ZCD-1 RESOLVE 1: %v", err)
		}
	}

	// Check: is the fact still challengeable?
	fact, _ := kk.GetFact(ctx, factId)
	t.Logf("ZCD-1 FACT AFTER ROUND 1: status=%s confidence=%d", fact.Status, fact.Confidence)

	// Can challenger open another dispute?
	// Reset active dispute count (the first is resolved)
	activeCount := k.GetActiveDisputeCountForFact(ctx, factId)
	t.Logf("ZCD-1 ACTIVE DISPUTES: %d", activeCount)

	// === ECONOMIC ANALYSIS ===
	t.Log("ZCD-1 INSIGHT:")
	t.Log("  Inconclusive outcome → challenger gets full bond back")
	t.Log("  Fact remains in 'contested' status")
	t.Log("  Challenger can re-challenge indefinitely at zero net cost")
	t.Log("  DEFENSE: Each challenge requires a new 111 LGM bond (temporarily locked)")
	t.Log("  DEFENSE: Arbiter fees and gas costs add friction")
	t.Log("  RESIDUAL RISK: Wealthy attacker can loop indefinitely, keeping fact contested")
}

// ==================================================================================
// ZCD-2: Non-Arbiter Vote Attempt
// ==================================================================================
// Scenario: Non-selected validator tries to vote on a dispute.
func TestZCD2_NonArbiterVoteAttempt(t *testing.T) {
	k, ctx, _, _, sk := setupKeeper(t)

	arbiter := testAddr(10)
	attacker := testAddr(99)
	sk.validators = []types.ValidatorInfo{
		{Address: arbiter, Stake: 5000, Tier: 2},
		{Address: attacker, Stake: 10000, Tier: 3}, // Higher tier/stake but not selected
	}

	dispute := types.Dispute{
		Id:             "dispute-nonarbiter",
		Status:         string(types.DisputeStatusVoting),
		VotingDeadline: 500,
		Arbiters:       []string{arbiter}, // Only arbiter is selected
	}
	k.SetDispute(ctx, dispute)

	err := k.CastVote(ctx, dispute.Id, attacker, "overturn", "I want to vote too")

	if err != nil {
		t.Logf("ZCD-2 DEFENSE: Non-arbiter vote rejected: %v", err)
	} else {
		t.Log("ZCD-2 CRITICAL: Non-arbiter vote accepted! Attacker can influence dispute outcomes!")
	}

	// Verify no phantom vote stored
	_, found := k.GetVote(ctx, dispute.Id, attacker)
	if found {
		t.Error("ZCD-2 CRITICAL: Phantom vote stored for non-arbiter")
	}
}

// ==================================================================================
// ZCD-3: Dispute Against Own Fact
// ==================================================================================
// Scenario: Submitter challenges their own fact to trigger vesting pause
// or manipulate confidence scores.
func TestZCD3_DisputeOwnFact(t *testing.T) {
	k, ctx, bk, kk, sk := setupKeeper(t)

	selfDisputer := testAddr(1)
	kk.facts["fact-self"] = types.Fact{
		Id: "fact-self", ClaimText: "My own fact",
		Domain: "physics", Status: "verified", Confidence: 900_000,
		Submitter: selfDisputer,
	}
	setupValidators(sk, selfDisputer, selfDisputer)
	bk.fundAccount(selfDisputer, sdkmath.NewInt(200_000_000))

	_, err := k.OpenDispute(ctx, "fact-self", selfDisputer, "Challenging my own fact", "111000000")

	// === INSIGHT ANALYSIS ===
	if err != nil {
		t.Logf("ZCD-3 DEFENSE: Self-dispute rejected: %v", err)
	} else {
		t.Log("ZCD-3 FINDING: Self-dispute allowed!")
		t.Log("  EXPLOITATION:")
		t.Log("    - Submitter pauses their own vesting (strategic timing)")
		t.Log("    - Force inconclusive → fact becomes 'contested' → resubmit improved version")
		t.Log("    - Use arbiter selection to probe which validators would review the claim")
		t.Log("  DEFENSE: If upheld, self-disputer loses 50% of bond (55 LGM)")
	}
}
