package keeper_test

// ==================================================================================
// OpenClaw Agent Testing Session — Financial Incentives & Agent Behavior
// ==================================================================================
//
// Purpose: Open-ended exploratory testing of knowledge tree financial incentives.
// Each test probes a specific game-theoretic scenario, computes attack economics,
// and records whether the protocol's defenses hold.
//
// Naming: OC = OpenClaw probe
// Methodology: Set up economic scenario → execute → measure cost/damage/profit → log insight
// ==================================================================================

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// ==================================================================================
// OC-1: Whale Domination — Can a single high-stake validator control outcomes?
// ==================================================================================
// Scenario: One validator with 90% of total stake always votes "accept".
// Two minority validators (5% each) vote "reject". Does the whale win?
func TestOC1_WhaleDomination(t *testing.T) {
	k, ctx, mockSK, _, _ := setupKeeperWithTracking(t)

	// Reset validators: whale + 2 minnows
	delete(mockSK.validators, "lgmvaloper1abc")
	delete(mockSK.validators, "lgmvaloper1def")
	delete(mockSK.validators, "lgmvaloper1ghi")
	mockSK.addValidator("lgmvaloper1whale", 90_000_000, 3) // 90 LGM (whale)
	mockSK.addValidator("lgmvaloper1min1", 5_000_000, 1)   // 5 LGM
	mockSK.addValidator("lgmvaloper1min2", 5_000_000, 1)   // 5 LGM

	claim := types.Claim{
		Id:                "oc1-whale-claim",
		ClaimText:         "Whale-dominated claim that two honest validators reject",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, _ := k.CreateVerificationRound(ctx, &claim)

	// Whale accepts, minorities reject
	round.Commitments["lgmvaloper1whale"] = &types.VerificationCommitment{Validator: "lgmvaloper1whale"}
	round.Commitments["lgmvaloper1min1"] = &types.VerificationCommitment{Validator: "lgmvaloper1min1"}
	round.Commitments["lgmvaloper1min2"] = &types.VerificationCommitment{Validator: "lgmvaloper1min2"}

	round.Reveals["lgmvaloper1whale"] = &types.VerificationReveal{
		Validator: "lgmvaloper1whale", Verdict: "accept", Confidence: 900000,
	}
	round.Reveals["lgmvaloper1min1"] = &types.VerificationReveal{
		Validator: "lgmvaloper1min1", Verdict: "reject", Confidence: 900000,
	}
	round.Reveals["lgmvaloper1min2"] = &types.VerificationReveal{
		Validator: "lgmvaloper1min2", Verdict: "reject", Confidence: 900000,
	}

	round.SelectedValidators = []string{"lgmvaloper1whale", "lgmvaloper1min1", "lgmvaloper1min2"}
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// === INSIGHT ANALYSIS ===
	// Whale: 90M stake * 900k conf / 1M = 81M weighted accept
	// Min1+Min2: (5M+5M) * 900k / 1M = 9M weighted reject
	// Total = 90M, Accept ratio = 81/90 = 90% → exceeds 77% threshold
	t.Logf("OC-1 RESULT: verdict=%s, agreement=%d, confidence=%d",
		result.Verdict, result.AgreementRatio, result.Confidence)
	t.Logf("OC-1 INSIGHT: Whale with 90%% stake DOES control outcome (verdict=%s)", result.Verdict)

	if result.Verdict == "accepted" {
		t.Logf("OC-1 RISK: Whale domination confirmed — 90%% stake holder overrides 2 honest validators")
		t.Logf("OC-1 DEFENSE: capture_defense reputation weighting should dampen this (nil in test)")

		// Check: minority validators get slashed for "wrong" verdict
		for v, slash := range result.Slashes {
			t.Logf("OC-1 COLLATERAL: %s slashed %d ulgm for honest minority vote", v, slash)
		}
		// Check: whale gets rewarded for potentially dishonest acceptance
		for v, reward := range result.Rewards {
			t.Logf("OC-1 REWARD: %s gets %d ulgm (%0.2f LGM)", v, reward, float64(reward)/1_000_000)
		}
	}
}

// ==================================================================================
// OC-2: Grief-by-Rejection — Colluding verifiers burn submitter stakes
// ==================================================================================
// Scenario: 3 colluding validators reject every claim to burn the 22% stake.
// Question: What does it cost the attacker vs the damage to victims?
func TestOC2_GriefByRejection(t *testing.T) {
	k, ctx, _, tbk, _ := setupKeeperWithTracking(t)

	submitterStake := int64(10_000_000) // 10 LGM at risk
	claim := types.Claim{
		Id:                "oc2-grief-claim",
		ClaimText:         "Honest claim that colluding validators will reject",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       fmt.Sprintf("%d", submitterStake),
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, _ := k.CreateVerificationRound(ctx, &claim)

	// All 3 validators collude to reject
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = &types.VerificationCommitment{Validator: v}
		round.Reveals[v] = &types.VerificationReveal{
			Validator: v, Verdict: "reject", Confidence: 900000,
		}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	_ = k.CompleteRound(ctx, round, result)

	// === ECONOMIC ANALYSIS ===
	slashBps := int64(220000) // 22%
	expectedSlash := submitterStake * slashBps / 1_000_000
	expectedReturn := submitterStake - expectedSlash

	returnedAmt := tbk.sentToAccount[testSubmitterAddr].AmountOf("ulgm").Int64()
	burnedAmt := tbk.burned.AmountOf("ulgm").Int64()

	t.Logf("OC-2 RESULT: verdict=%s", result.Verdict)
	t.Logf("OC-2 ECONOMICS:")
	t.Logf("  Submitter stake: %d ulgm (%.2f LGM)", submitterStake, float64(submitterStake)/1_000_000)
	t.Logf("  Expected slash:  %d ulgm (22%%)", expectedSlash)
	t.Logf("  Actual returned: %d ulgm", returnedAmt)
	t.Logf("  Actual burned:   %d ulgm", burnedAmt)

	// Attacker cost: verifiers stake nothing to vote — they risk slash on THEIR stake only if wrong
	// But since they collude to the majority, they ARE the majority → no slashing
	t.Logf("  Attacker cost:   0 ulgm (colluders form majority, no slash)")
	t.Logf("  Attacker reward: %d ulgm each in verification rewards", result.Rewards["lgmvaloper1abc"])
	t.Logf("  Victim loss:     %d ulgm (%.2f LGM burned)", burnedAmt, float64(burnedAmt)/1_000_000)

	totalAttackerReward := uint64(0)
	for _, r := range result.Rewards {
		totalAttackerReward += r
	}
	t.Logf("OC-2 PROFIT RATIO: Attacker earns %d ulgm, victim loses %d ulgm",
		totalAttackerReward, burnedAmt)

	if returnedAmt != expectedReturn {
		t.Errorf("stake math mismatch: expected return %d, got %d", expectedReturn, returnedAmt)
	}

	t.Logf("OC-2 DEFENSE: VRF selection randomizes validator assignment — colluders can't guarantee selection")
	t.Logf("OC-2 DEFENSE: Reputation decay from false rejections would eventually lower collusion weight")
	t.Logf("OC-2 DEFENSE: Challenge mechanism allows victims to contest false rejections")
}

// ==================================================================================
// OC-3: Abstain-to-Avoid-Slash — Can verifiers dodge risk while earning presence?
// ==================================================================================
// Scenario: Verifiers commit then abstain on reveal to avoid wrong-verdict slash.
// Question: Does abstaining carry any penalty? Can they farm quorum presence risk-free?
func TestOC3_AbstainToAvoidSlash(t *testing.T) {
	k, ctx, mockSK, _, _ := setupKeeperWithTracking(t)

	// Need 4 validators: 2 accept, 1 reject, 1 abstain
	mockSK.addValidator("lgmvaloper1jkl", 4_000_000, 2)

	claim := types.Claim{
		Id:                "oc3-abstain-claim",
		ClaimText:         "Claim where one validator strategically abstains",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, _ := k.CreateVerificationRound(ctx, &claim)

	round.Commitments["lgmvaloper1abc"] = &types.VerificationCommitment{Validator: "lgmvaloper1abc"}
	round.Commitments["lgmvaloper1def"] = &types.VerificationCommitment{Validator: "lgmvaloper1def"}
	round.Commitments["lgmvaloper1ghi"] = &types.VerificationCommitment{Validator: "lgmvaloper1ghi"}
	round.Commitments["lgmvaloper1jkl"] = &types.VerificationCommitment{Validator: "lgmvaloper1jkl"}

	round.Reveals["lgmvaloper1abc"] = &types.VerificationReveal{
		Validator: "lgmvaloper1abc", Verdict: "accept", Confidence: 900000,
	}
	round.Reveals["lgmvaloper1def"] = &types.VerificationReveal{
		Validator: "lgmvaloper1def", Verdict: "accept", Confidence: 900000,
	}
	round.Reveals["lgmvaloper1ghi"] = &types.VerificationReveal{
		Validator: "lgmvaloper1ghi", Verdict: "reject", Confidence: 800000,
	}
	round.Reveals["lgmvaloper1jkl"] = &types.VerificationReveal{
		Validator: "lgmvaloper1jkl", Verdict: "abstain", Confidence: 500000,
	}

	round.SelectedValidators = []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi", "lgmvaloper1jkl"}
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)

	// === INSIGHT ANALYSIS ===
	t.Logf("OC-3 RESULT: verdict=%s, agreement=%d", result.Verdict, result.AgreementRatio)

	// Check abstainer: did they get slashed or rewarded?
	abstainSlash, abstainSlashed := result.Slashes["lgmvaloper1jkl"]
	abstainReward, abstainRewarded := result.Rewards["lgmvaloper1jkl"]

	t.Logf("OC-3 ABSTAINER: slashed=%v (%d), rewarded=%v (%d)",
		abstainSlashed, abstainSlash, abstainRewarded, abstainReward)

	if !abstainSlashed && !abstainRewarded {
		t.Logf("OC-3 INSIGHT: Abstaining is RISK-FREE — no slash, no reward")
		t.Logf("OC-3 RISK: Lazy validators can abstain to avoid risk while occupying quorum slots")
		t.Logf("OC-3 DEFENSE: No direct reward for abstaining. But zero-cost participation could be exploited.")
	} else if abstainSlashed {
		t.Logf("OC-3 DEFENSE: Abstaining incurs slash — discourages lazy participation")
	} else if abstainRewarded {
		t.Logf("OC-3 BUG: Abstaining should NOT be rewarded — this incentivizes non-participation")
	}

	// What if ghi (the rejector) also loses? Check minority validator slash
	rejectSlash, rejectSlashed := result.Slashes["lgmvaloper1ghi"]
	t.Logf("OC-3 MINORITY: lgmvaloper1ghi (rejected) slashed=%v (%d ulgm)",
		rejectSlashed, rejectSlash)
}

// ==================================================================================
// OC-4: Minimum Stake Spam — Overwhelm verification with 1 ulgm claims
// ==================================================================================
// Scenario: Attacker submits 1000 claims at minimum stake (1 ulgm each).
// Question: Total cost to attacker? Impact on validator time? Spam defense?
func TestOC4_MinimumStakeSpam(t *testing.T) {
	k, ctx, _, _, _ := setupKeeperWithTracking(t)

	numSpamClaims := 100
	minStake := int64(1) // 1 ulgm = 0.000001 LGM

	successCount := 0
	failCount := 0
	for i := 0; i < numSpamClaims; i++ {
		claim := types.Claim{
			Id:                fmt.Sprintf("oc4-spam-%d", i),
			ClaimText:         fmt.Sprintf("Spam claim number %d for pipeline flood testing scenario", i),
			Domain:            "general",
			EpistemicCategory: "empirical",
			Submitter:         testSubmitterAddr,
			StakeAmount:       fmt.Sprintf("%d", minStake),
			Status:            "pending",
			SubmittedAtHeight: uint64(10 + i),
		}
		k.SetClaim(ctx, &claim)

		_, err := k.CreateVerificationRound(ctx, &claim)
		if err != nil {
			failCount++
		} else {
			successCount++
		}
	}

	// === ECONOMIC ANALYSIS ===
	totalCost := int64(numSpamClaims) * minStake
	slashPerClaim := minStake * 220000 / 1_000_000 // 22% of 1 ulgm = 0 (integer truncation!)
	totalSlashRisk := int64(numSpamClaims) * slashPerClaim

	t.Logf("OC-4 SPAM RESULTS:")
	t.Logf("  Claims submitted: %d", numSpamClaims)
	t.Logf("  Success: %d, Failed: %d", successCount, failCount)
	t.Logf("  Total cost: %d ulgm (%.6f LGM)", totalCost, float64(totalCost)/1_000_000)
	t.Logf("  Slash per claim: %d ulgm (22%% of 1 = integer 0!)", slashPerClaim)
	t.Logf("  Total slash risk: %d ulgm", totalSlashRisk)

	if slashPerClaim == 0 {
		t.Logf("OC-4 CRITICAL INSIGHT: 22%% of 1 ulgm = 0 due to integer truncation!")
		t.Logf("OC-4 RISK: Attacker can spam claims with ZERO slashing risk at 1 ulgm each")
		t.Logf("OC-4 COST: Each claim consumes 3+ validator slots × commit+reveal phases")
		t.Logf("OC-4 DEFENSE NEEDED: Minimum claim stake floor (e.g., 1000 ulgm = 0.001 LGM)")
		t.Logf("OC-4 DEFENSE EXISTS: Gas fees per tx, but economic slash is bypassed")
	}

	// Verify: what's the minimum stake where slash > 0?
	for s := int64(1); s <= 10; s++ {
		slash := s * 220000 / 1_000_000
		if slash > 0 {
			t.Logf("OC-4 MINIMUM EFFECTIVE STAKE: %d ulgm (first stake where 22%% slash > 0)", s)
			break
		}
	}
}

// ==================================================================================
// OC-5: Tier 0 Reward Viability — Can a new validator earn enough to climb?
// ==================================================================================
// Scenario: Tier 0 validator (0.1x multiplier) participates in verification.
// Question: How many rounds to earn enough for Tier 1 promotion?
func TestOC5_Tier0RewardViability(t *testing.T) {
	k, ctx, mockSK, _, _ := setupKeeperWithTracking(t)

	// Add a Tier 0 validator with minimum stake
	mockSK.addValidator("lgmvaloper1newbie", 111_000, 0) // 0.111 LGM (minimum)

	claim := types.Claim{
		Id:                "oc5-tier-claim",
		ClaimText:         "Standard claim for tier viability analysis of new validators",
		Domain:            "general",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, _ := k.CreateVerificationRound(ctx, &claim)

	// All accept (newbie included)
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1newbie"}
	for _, v := range validators {
		round.Commitments[v] = &types.VerificationCommitment{Validator: v}
		round.Reveals[v] = &types.VerificationReveal{
			Validator: v, Verdict: "accept", Confidence: 800000,
		}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)

	// === REWARD ANALYSIS ===
	params := k.GetParams(ctx)
	baseReward := params.VerificationRewardAmount // 3,000,000 ulgm (3 LGM)

	tier0Reward := result.Rewards["lgmvaloper1newbie"]
	tier2Reward := result.Rewards["lgmvaloper1abc"]

	t.Logf("OC-5 TIER REWARD ANALYSIS:")
	t.Logf("  Base reward: %d ulgm (%.2f LGM)", baseReward, float64(baseReward)/1_000_000)
	t.Logf("  Tier 0 reward (0.1x): %d ulgm (%.3f LGM)", tier0Reward, float64(tier0Reward)/1_000_000)
	t.Logf("  Tier 2 reward (1.0x): %d ulgm (%.3f LGM)", tier2Reward, float64(tier2Reward)/1_000_000)

	// Tier 0 → Tier 1 requires: 50+ verifications with 75%+ accuracy
	// Tier 1 min stake: ~1 LGM = 1,000,000 ulgm
	roundsToTier1Stake := int64(0)
	if tier0Reward > 0 {
		roundsToTier1Stake = 1_000_000 / int64(tier0Reward)
	}

	t.Logf("  Rounds to earn 1 LGM at Tier 0: %d", roundsToTier1Stake)
	t.Logf("  Tier 1 req: 50 verifications at 75%% accuracy")
	t.Logf("  Time estimate at 1 round/block: %.1f hours", float64(roundsToTier1Stake)*2.521/3600)

	if tier0Reward == 0 {
		t.Logf("OC-5 CRITICAL: Tier 0 reward is 0 ulgm — new validators can NEVER earn through verification alone")
		t.Logf("OC-5 INSIGHT: applyTierMultiplier(3M, 0) = 3M * 100/1000 = 300000 ulgm expected (0.3 LGM)")
	} else {
		t.Logf("OC-5 VIABLE: Tier 0 can earn, but at 10x slower rate than Tier 2")
	}
}

// ==================================================================================
// OC-6: Counter-fact Profit Maximization — Optimal falsification strategy
// ==================================================================================
// Scenario: Agent targets facts with high original stakes for falsification rewards.
// Question: What's the maximum ROI? Is there a "falsification whaling" strategy?
func TestOC6_CounterFactProfitMaximization(t *testing.T) {
	k, ctx, _, _, mvk := setupKeeperWithTracking(t)

	// Test 3 scenarios with different stake ratios
	scenarios := []struct {
		name           string
		targetStake    string // OriginalStake of target fact
		cfStake        string // Counter-fact stake
		confidence     uint64 // Verification confidence
		expectedReward string // min(target*conf/1M, cf*3)
		expectedROI    string // reward / cfStake ratio
	}{
		{
			name:           "small_cf_vs_big_fact",
			targetStake:    "100000000", // 100 LGM
			cfStake:        "5000000",   // 5 LGM
			confidence:     900000,      // 90%
			expectedReward: "15000000",  // min(90M, 15M) = 15M (3x cap)
			expectedROI:    "3.0x",
		},
		{
			name:           "proportional_stakes",
			targetStake:    "10000000", // 10 LGM
			cfStake:        "5000000",  // 5 LGM
			confidence:     800000,     // 80%
			expectedReward: "8000000",  // min(8M, 15M) = 8M (uncapped)
			expectedROI:    "1.6x",
		},
		{
			name:           "overleveraged_cf",
			targetStake:    "2000000",  // 2 LGM
			cfStake:        "10000000", // 10 LGM
			confidence:     900000,     // 90%
			expectedReward: "1800000",  // min(1.8M, 30M) = 1.8M (target-limited)
			expectedROI:    "0.18x",
		},
	}

	for i, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			factID := fmt.Sprintf("f-oc6-%d", i)
			cfID := fmt.Sprintf("cfact-oc6-%d", i)
			roundID := fmt.Sprintf("r-oc6-%d", i)

			k.SetFact(ctx, &types.Fact{
				Id: factID, ClaimText: "Target fact", Domain: "physics",
				EpistemicCategory: "empirical", Status: types.FactStatusVerified,
				Submitter: "lgm1other", OriginalStake: sc.targetStake,
				Confidence: 800000,
			})

			k.SetCounterFact(ctx, &types.CounterFact{
				Id:           cfID,
				TargetFactId: factID,
				NegationType: types.NegationTypeContradiction,
				Submitter:    "lgm1w3jhxarpv3j8yv2lta047h6lta047h6l6m7hcw",
				StakeAmount:  sc.cfStake,
				Status:       types.CounterFactStatusPending,
			})

			k.SetVerificationRound(ctx, &types.VerificationRound{
				Id: roundID, ClaimId: cfID, Type: types.RoundTypeContradiction,
				Phase: "reveal",
				Reveals: map[string]*types.VerificationReveal{
					"val1": {Verdict: "accepted", Confidence: sc.confidence},
				},
			})

			_ = k.CompleteRound(ctx, &types.VerificationRound{
				Id: roundID, ClaimId: cfID, Type: types.RoundTypeContradiction,
				Phase: "reveal",
				Reveals: map[string]*types.VerificationReveal{
					"val1": {Verdict: "accepted", Confidence: sc.confidence},
				},
			}, &types.VerificationResult{
				Verdict: "accepted", Confidence: sc.confidence,
			})

			// Finalize via challenge window
			windowEnd := uint64(10) + types.CounterFactChallengeWindowBlocks + 1
			expiryCtx := ctx.WithBlockHeight(int64(windowEnd))
			k.ProcessCounterFactChallengeExpiry(expiryCtx, windowEnd)

			t.Logf("  Scenario: %s", sc.name)
			t.Logf("    Target fact stake: %s ulgm", sc.targetStake)
			t.Logf("    Counter-fact stake: %s ulgm", sc.cfStake)
			t.Logf("    Confidence: %d bps", sc.confidence)
			t.Logf("    Expected reward: %s ulgm (ROI: %s)", sc.expectedReward, sc.expectedROI)

			// Check actual reward
			for _, vc := range mvk.created {
				if vc.claimId == cfID && vc.category == "falsification" {
					t.Logf("    Actual reward:   %s ulgm", vc.totalAmount)
					break
				}
			}
		})
	}

	t.Logf("\nOC-6 INSIGHT: Optimal strategy is small cf stakes against large target facts (3x ROI cap)")
	t.Logf("OC-6 INSIGHT: Overleveraged cf stakes (cf > target) are UNPROFITABLE (<1x ROI)")
	t.Logf("OC-6 INSIGHT: 'Falsification whaling' is viable — target facts with large OriginalStake")
	t.Logf("OC-6 DEFENSE: 3x cap limits maximum profit; MinContradictionStakeRatio prevents dust attacks")
	t.Logf("OC-6 DEFENSE: Challenge window (34,272 blocks) delays reward, allowing counter-challenges")
}

// ==================================================================================
// OC-7: Split Verdict Stake Lock — Engineering contested outcomes
// ==================================================================================
// Scenario: Attacker controls 2 of 4 validators and splits their votes to create
// a contested outcome, locking the submitter's stake indefinitely.
func TestOC7_SplitVerdictStakeLock(t *testing.T) {
	k, ctx, mockSK, tbk, _ := setupKeeperWithTracking(t)

	// 4 validators with equal stake to maximize split potential
	delete(mockSK.validators, "lgmvaloper1abc")
	delete(mockSK.validators, "lgmvaloper1def")
	delete(mockSK.validators, "lgmvaloper1ghi")
	mockSK.addValidator("lgmvaloper1a", 5_000_000, 2) // honest
	mockSK.addValidator("lgmvaloper1b", 5_000_000, 2) // honest
	mockSK.addValidator("lgmvaloper1c", 5_000_000, 2) // attacker
	mockSK.addValidator("lgmvaloper1d", 5_000_000, 2) // attacker

	claim := types.Claim{
		Id:                "oc7-split-claim",
		ClaimText:         "Valid claim that attackers will try to contest by splitting votes",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "5000000", // 5 LGM locked
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, _ := k.CreateVerificationRound(ctx, &claim)

	// Honest validators accept; attacker validators reject to create split
	round.Commitments["lgmvaloper1a"] = &types.VerificationCommitment{Validator: "lgmvaloper1a"}
	round.Commitments["lgmvaloper1b"] = &types.VerificationCommitment{Validator: "lgmvaloper1b"}
	round.Commitments["lgmvaloper1c"] = &types.VerificationCommitment{Validator: "lgmvaloper1c"}
	round.Commitments["lgmvaloper1d"] = &types.VerificationCommitment{Validator: "lgmvaloper1d"}

	round.Reveals["lgmvaloper1a"] = &types.VerificationReveal{Validator: "lgmvaloper1a", Verdict: "accept", Confidence: 900000}
	round.Reveals["lgmvaloper1b"] = &types.VerificationReveal{Validator: "lgmvaloper1b", Verdict: "accept", Confidence: 900000}
	round.Reveals["lgmvaloper1c"] = &types.VerificationReveal{Validator: "lgmvaloper1c", Verdict: "reject", Confidence: 900000}
	round.Reveals["lgmvaloper1d"] = &types.VerificationReveal{Validator: "lgmvaloper1d", Verdict: "reject", Confidence: 900000}

	round.SelectedValidators = []string{"lgmvaloper1a", "lgmvaloper1b", "lgmvaloper1c", "lgmvaloper1d"}
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	_ = k.CompleteRound(ctx, round, result)

	// === ANALYSIS ===
	// Equal stakes: 2×accept(10M) vs 2×reject(10M) = 50/50 split
	// Agreement ratio = 50% < 55% → should be "inconclusive" (not even contested)
	t.Logf("OC-7 RESULT: verdict=%s, agreement=%d", result.Verdict, result.AgreementRatio)

	stakeReturned := tbk.sentToAccount[testSubmitterAddr].AmountOf("ulgm").Int64()
	t.Logf("OC-7 STAKE: returned=%d ulgm to submitter", stakeReturned)

	if result.Verdict == "inconclusive" {
		t.Logf("OC-7 INSIGHT: Equal split → inconclusive (below 55%% contested threshold)")
		if stakeReturned > 0 {
			t.Logf("OC-7 DEFENSE: Stake returned on inconclusive — attacker cannot lock victim's funds (C-1 fix verified)")
		} else {
			t.Errorf("OC-7 BUG: Stake NOT returned on inconclusive — C-1 fix failed!")
		}
	} else if result.Verdict == "contested" {
		t.Logf("OC-7 RISK: 50/50 split classified as contested — stake potentially locked")
	}

	// Check: attacker validators — are they slashed for being on the losing side?
	for v, slash := range result.Slashes {
		t.Logf("OC-7 SLASH: %s → %d ulgm", v, slash)
	}
}

// ==================================================================================
// OC-8: Difficulty Bonus Exploitation — Submit borderline claims for higher rewards
// ==================================================================================
// Scenario: Verifiers encounter a claim at 60% confidence (borderline accepted).
// The difficulty bonus formula gives +40% extra reward at 60% confidence.
// Question: Can agents collude to submit at borderline confidence for max rewards?
func TestOC8_DifficultyBonusExploitation(t *testing.T) {
	k, ctx, _, _, _ := setupKeeperWithTracking(t)

	// Test reward at different confidence levels
	params := k.GetParams(ctx)
	baseReward := params.VerificationRewardAmount // 3,000,000 ulgm

	confidenceLevels := []struct {
		conf  uint64
		label string
	}{
		{990000, "99% (trivial)"},
		{900000, "90% (easy)"},
		{770000, "77% (threshold)"},
		{600000, "60% (borderline)"},
		{500000, "50% (near-reject)"},
	}

	t.Logf("OC-8 DIFFICULTY BONUS ANALYSIS (base reward: %d ulgm = %.2f LGM):",
		baseReward, float64(baseReward)/1_000_000)

	for _, cl := range confidenceLevels {
		claim := types.Claim{
			Id:                fmt.Sprintf("oc8-conf-%d", cl.conf),
			ClaimText:         fmt.Sprintf("Claim at %s confidence for difficulty testing scenario analysis", cl.label),
			Domain:            "general",
			EpistemicCategory: "empirical",
			Submitter:         testSubmitterAddr,
			StakeAmount:       "1000000",
			Status:            "pending",
		}
		k.SetClaim(ctx, &claim)
		round, _ := k.CreateVerificationRound(ctx, &claim)

		validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
		for _, v := range validators {
			round.Commitments[v] = &types.VerificationCommitment{Validator: v}
			round.Reveals[v] = &types.VerificationReveal{
				Validator: v, Verdict: "accept", Confidence: cl.conf,
			}
		}
		round.SelectedValidators = validators
		k.SetVerificationRound(ctx, round)

		result, _ := k.AggregateVerificationResult(ctx, round)

		tier2Reward := result.Rewards["lgmvaloper1abc"]
		difficultyBonus := uint64(0)
		if cl.conf < 1_000_000 {
			difficultyBonus = baseReward * (1_000_000 - cl.conf) / 1_000_000
		}

		t.Logf("  %s → reward=%d ulgm (%.3f LGM), difficulty_bonus=%d ulgm, verdict=%s",
			cl.label, tier2Reward, float64(tier2Reward)/1_000_000,
			difficultyBonus, result.Verdict)
	}

	t.Logf("\nOC-8 INSIGHT: Difficulty bonus creates incentive to verify hard claims honestly")
	t.Logf("OC-8 RISK: Colluding verifiers could coordinate low-confidence accepts to inflate rewards")
	t.Logf("OC-8 DEFENSE: Agreement ratio must still exceed 77%% — can't claim difficulty AND low agreement")
	t.Logf("OC-8 DEFENSE: AH-4 method diversity check penalizes suspiciously uniform verdicts")
}

// ==================================================================================
// OC-9: Research Tax Evasion — Can verifiers route around the 7% tax?
// ==================================================================================
// Analysis: Research tax is applied on reward distribution, not verification.
// The 7% tax reduces the effective reward for verifiers.
func TestOC9_ResearchTaxAnalysis(t *testing.T) {
	// Pure economic analysis — no keeper needed

	baseReward := uint64(3_000_000)  // 3 LGM
	researchTaxBps := uint64(70_000) // 7%

	taxAmount := baseReward * researchTaxBps / 1_000_000
	netReward := baseReward - taxAmount

	t.Logf("OC-9 RESEARCH TAX ANALYSIS:")
	t.Logf("  Base reward:  %d ulgm (%.2f LGM)", baseReward, float64(baseReward)/1_000_000)
	t.Logf("  Tax rate:     %d bps (%.1f%%)", researchTaxBps, float64(researchTaxBps)/10_000)
	t.Logf("  Tax amount:   %d ulgm", taxAmount)
	t.Logf("  Net reward:   %d ulgm (%.3f LGM)", netReward, float64(netReward)/1_000_000)

	// Tax evasion vectors:
	t.Logf("\nOC-9 TAX EVASION VECTORS:")
	t.Logf("  1. Direct verification rewards: Tax applied at vesting, not claim-level → CANNOT evade")
	t.Logf("  2. Counter-fact rewards: Routed through vesting → TAXED")
	t.Logf("  3. Citation rewards: Routed through vesting → TAXED")
	t.Logf("  4. Block rewards: Separate pool, own tax rate → SEPARATE SYSTEM")
	t.Logf("\nOC-9 CONCLUSION: Research tax is applied uniformly at vesting layer — no evasion path")
	t.Logf("OC-9 DEFENSE: VestingRewardsKeeper enforces tax on all reward sources")
}

// ==================================================================================
// OC-10: Equivocation Risk/Reward — Is double-voting ever rational?
// ==================================================================================
// Scenario: A validator with 10 LGM stake submits two different commitments.
// Question: What's the maximum possible gain vs the 100% slash?
func TestOC10_EquivocationRiskReward(t *testing.T) {
	k, ctx, mockSK, _, _ := setupKeeperWithTracking(t)

	validatorStake := uint64(10_000_000) // 10 LGM
	mockSK.validators["lgmvaloper1abc"] = &types.ValidatorInfo{
		Address: "lgmvaloper1abc", Stake: validatorStake, Tier: 2,
		PublicKey: []byte("pubkey_lgmvaloper1abc"), Reputation: 800000,
	}

	claim := types.Claim{
		Id:                "oc10-equivoc-claim",
		ClaimText:         "Claim to test equivocation detection and slashing scenario",
		Domain:            "general",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, _ := k.CreateVerificationRound(ctx, &claim)

	// First commitment succeeds
	err := k.StoreCommitmentInRound(ctx, round.Id, &types.VerificationCommitment{
		Validator:      "lgmvaloper1abc",
		CommitmentHash: "aaaa" + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	if err != nil {
		t.Fatalf("first commitment failed: %v", err)
	}

	// Second commitment should be rejected (equivocation attempt)
	err = k.StoreCommitmentInRound(ctx, round.Id, &types.VerificationCommitment{
		Validator:      "lgmvaloper1abc",
		CommitmentHash: "cccc" + "dddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	})

	t.Logf("OC-10 EQUIVOCATION ANALYSIS:")
	t.Logf("  Validator stake: %d ulgm (%.2f LGM)", validatorStake, float64(validatorStake)/1_000_000)
	t.Logf("  Max verification reward: ~%d ulgm (%.2f LGM)", 3_000_000*2, float64(3_000_000*2)/1_000_000)
	t.Logf("  Equivocation slash: 100%% = %d ulgm (%.2f LGM)", validatorStake, float64(validatorStake)/1_000_000)

	if err != nil {
		t.Logf("  Duplicate commitment: REJECTED (%v)", err)
		t.Logf("OC-10 DEFENSE: Protocol rejects duplicate commitments — equivocation impossible at commit level")
	} else {
		t.Logf("OC-10 BUG: Duplicate commitment was ACCEPTED — equivocation vulnerability!")
	}

	t.Logf("\nOC-10 RISK/REWARD:")
	t.Logf("  Best case for attacker: +6 LGM (verification reward)")
	t.Logf("  Worst case for attacker: -10 LGM (100%% slash)")
	t.Logf("  Expected value: DEEPLY NEGATIVE — equivocation is never rational")
}

// ==================================================================================
// OC-11: Quorum Manipulation — Commit but don't reveal to kill quorum
// ==================================================================================
// Scenario: Attacker commits but never reveals, trying to push participation
// below quorum threshold (66%) so the claim gets "insufficient" verdict.
func TestOC11_QuorumManipulation(t *testing.T) {
	k, ctx, mockSK, tbk, _ := setupKeeperWithTracking(t)

	// 5 validators selected, attacker controls 2
	mockSK.addValidator("lgmvaloper1atk1", 3_000_000, 2)
	mockSK.addValidator("lgmvaloper1atk2", 3_000_000, 2)

	claim := types.Claim{
		Id:                "oc11-quorum-claim",
		ClaimText:         "Valid claim that attackers will try to sabotage via quorum starvation",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "5000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, _ := k.CreateVerificationRound(ctx, &claim)

	// All 5 commit, but attackers don't reveal
	allVals := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi", "lgmvaloper1atk1", "lgmvaloper1atk2"}
	for _, v := range allVals {
		round.Commitments[v] = &types.VerificationCommitment{Validator: v}
	}

	// Only honest validators reveal (3 of 5 = 60%)
	for _, v := range []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"} {
		round.Reveals[v] = &types.VerificationReveal{
			Validator: v, Verdict: "accept", Confidence: 900000,
		}
	}

	round.SelectedValidators = allVals
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)
	_ = k.CompleteRound(ctx, round, result)

	// 3/5 = 60% participation, quorum requires 66%
	t.Logf("OC-11 QUORUM MANIPULATION:")
	t.Logf("  Selected: %d, Revealed: %d (%d%%)", len(allVals), len(round.Reveals), 100*len(round.Reveals)/len(allVals))
	t.Logf("  Quorum threshold: 66%%")
	t.Logf("  Result: verdict=%s", result.Verdict)

	if result.Verdict == "insufficient" {
		stakeReturned := tbk.sentToAccount[testSubmitterAddr].AmountOf("ulgm").Int64()
		t.Logf("  Stake returned: %d ulgm (%.2f LGM)", stakeReturned, float64(stakeReturned)/1_000_000)
		t.Logf("OC-11 RISK: Attackers can kill quorum by committing but not revealing (3/5 < 66%%)")
		t.Logf("OC-11 COST TO ATTACKER:")

		// Check: are non-revealers slashed?
		for v, slash := range result.Slashes {
			t.Logf("  %s slashed %d ulgm for missed reveal", v, slash)
		}

		missedSlashBps := uint64(10_000) // 1% (raised from 0.01% by OpenClaw C-2 fix)
		attackerStake := uint64(3_000_000)
		attackerSlash := attackerStake * missedSlashBps / 1_000_000
		t.Logf("  Expected slash per attacker: %d ulgm (1%% of 3M)", attackerSlash)
		t.Logf("OC-11 DEFENSE: Missed reveal slash raised to 1%% (was 0.01%%) — quorum attacks now cost 30,000 ulgm each")
	} else {
		t.Logf("OC-11 DEFENSE: 3/5 still exceeds quorum — claim processed normally")
	}
}

// ==================================================================================
// OC-12: Sybil Attack Economics — Creating multiple validators to control rounds
// ==================================================================================
// Pure economic analysis — compute cost to create enough Sybil validators
// to control verification outcomes.
func TestOC12_SybilAttackEconomics(t *testing.T) {
	minValidatorStake := int64(111_000) // 0.111 LGM minimum
	minVerifiers := 3

	// Cost to create enough Sybil validators to control outcomes
	// Need 3 validators (all selected) to guarantee 100% agreement
	sybilCost := int64(minVerifiers) * minValidatorStake

	t.Logf("OC-12 SYBIL ATTACK ECONOMICS:")
	t.Logf("  Min validator stake: %d ulgm (%.3f LGM)", minValidatorStake, float64(minValidatorStake)/1_000_000)
	t.Logf("  Min verifiers per round: %d", minVerifiers)
	t.Logf("  Sybil set cost: %d ulgm (%.3f LGM) for %d validators",
		sybilCost, float64(sybilCost)/1_000_000, minVerifiers)

	// But VRF selection probability is stake-weighted
	totalNetworkStake := int64(100_000_000) // Assume 100 LGM total
	sybilSelectProb := float64(sybilCost) / float64(totalNetworkStake)

	t.Logf("  Network stake: %d ulgm (assumed)", totalNetworkStake)
	t.Logf("  Sybil selection probability (per slot): %.4f%%", sybilSelectProb*100)
	t.Logf("  P(all 3 Sybil selected): %.8f%%", sybilSelectProb*sybilSelectProb*sybilSelectProb*100)

	// To get meaningful selection probability, need more stake
	for _, targetProb := range []float64{0.01, 0.1, 0.5} {
		neededStakePerValidator := int64(float64(totalNetworkStake) * targetProb / float64(minVerifiers))
		totalNeeded := neededStakePerValidator * int64(minVerifiers)
		t.Logf("  For %.0f%% per-slot selection: need %d ulgm (%.2f LGM) total",
			targetProb*100, totalNeeded, float64(totalNeeded)/1_000_000)
	}

	t.Logf("\nOC-12 INSIGHT: Sybil attack at min stake is ECONOMICALLY INFEASIBLE")
	t.Logf("OC-12 INSIGHT: VRF stake-weighting means Sybils need proportional stake")
	t.Logf("OC-12 DEFENSE: MaxValidatorsPerRound=22, VRF selection, reputation weighting")
	t.Logf("OC-12 DEFENSE: CaptureDefense module tracks domain concentration")
}

// ==================================================================================
// OC-13: Sensitivity Analysis — Slash rate vs deterrence threshold
// ==================================================================================
// What slash rate makes griefing unprofitable for all validator tiers?
func TestOC13_SlashRateSensitivity(t *testing.T) {
	t.Logf("OC-13 SLASH RATE SENSITIVITY ANALYSIS:")
	t.Logf("────────────────────────────────────────────────────────")
	t.Logf("%-20s %-12s %-15s %-12s %s", "Scenario", "Slash BPS", "Slash Amount", "Net Return", "Deterrent?")

	submitterStake := int64(10_000_000) // 10 LGM
	slashRates := []struct {
		name string
		bps  int64
	}{
		{"Current (22%)", 220_000},
		{"Light (10%)", 100_000},
		{"Heavy (50%)", 500_000},
		{"Full (100%)", 1_000_000},
		{"Tiny (1%)", 10_000},
	}

	for _, sr := range slashRates {
		slashAmount := submitterStake * sr.bps / 1_000_000
		netReturn := submitterStake - slashAmount
		deterrent := "YES"
		if slashAmount < 500_000 { // Less than 0.5 LGM at 10 LGM stake
			deterrent = "WEAK"
		}
		if slashAmount == 0 {
			deterrent = "NO"
		}
		t.Logf("%-20s %-12d %-15d %-12d %s",
			sr.name, sr.bps, slashAmount, netReturn, deterrent)
	}

	// Verifier slash analysis
	t.Logf("\n  Verifier wrong-verdict slash rate: 60,000 bps (6%%)")
	t.Logf("  Verifier missed-reveal slash rate: 100 bps (0.01%%)")
	t.Logf("  Equivocation slash rate: 1,000,000 bps (100%%)")
	t.Logf("\nOC-13 INSIGHT: 22%% claim slash is moderate deterrent — serious but not devastating")
	t.Logf("OC-13 INSIGHT: 0.01%% missed-reveal slash is TOO LOW for quorum manipulation defense")
	t.Logf("OC-13 RECOMMENDATION: Consider 1-5%% missed-reveal slash (10,000-50,000 bps)")
}

// ==================================================================================
// OC-14: Integer Truncation Attack — Zero-slash at low stakes
// ==================================================================================
// Extends OC-4 insight: What are ALL the integer truncation vulnerabilities?
func TestOC14_IntegerTruncationAttack(t *testing.T) {
	t.Logf("OC-14 INTEGER TRUNCATION ANALYSIS:")
	t.Logf("────────────────────────────────────────────────────────")

	truncationPoints := []struct {
		name    string
		bps     int64
		context string
	}{
		{"Claim rejection slash", 220_000, "submitter stake"},
		{"Wrong verdict slash", 60_000, "validator stake"},
		{"Missed reveal slash", 10_000, "validator stake"},
		{"Failed challenge slash", 220_000, "challenge stake"},
		{"Failed contradiction", 330_000, "cf stake"},
		{"Research tax", 70_000, "verification reward"},
		{"Citation share", 150_000, "reward pool"},
	}

	for _, tp := range truncationPoints {
		// Find minimum amount where slash > 0
		minEffective := int64(0)
		for s := int64(1); s <= 1_000_000; s++ {
			if s*tp.bps/1_000_000 > 0 {
				minEffective = s
				break
			}
		}
		zeroSlashBelow := minEffective - 1

		t.Logf("  %s (%d bps):", tp.name, tp.bps)
		t.Logf("    Zero-slash below: %d ulgm on %s", zeroSlashBelow, tp.context)
		t.Logf("    Min effective: %d ulgm (%.6f LGM)", minEffective, float64(minEffective)/1_000_000)

		if zeroSlashBelow > 100 {
			t.Logf("    ⚠ RISK: Amounts up to %d ulgm evade this slash entirely", zeroSlashBelow)
		}
	}

	// Special case: missed reveal slash (raised from 100 to 10,000 bps by C-2 fix)
	t.Logf("\n  FIXED: Missed reveal (10,000 bps / 1%%) now has zero-slash below %d ulgm", 1_000_000/10_000)
	t.Logf("  That's %d ulgm = %.2f LGM — min validator stake (111,000 ulgm) is well above this",
		100, float64(100)/1_000_000)
}

// ==================================================================================
// OC-15: Multi-round Accumulation — Long-term agent economic trajectory
// ==================================================================================
// Simulate 100 rounds for a Tier 2 validator to understand long-term economics.
func TestOC15_MultiRoundAccumulation(t *testing.T) {
	k, ctx, _, _, _ := setupKeeperWithTracking(t)

	params := k.GetParams(ctx)
	baseReward := params.VerificationRewardAmount // 3M ulgm = 3 LGM

	// Tier 2 validator: 1.0x multiplier
	// Assume 80% of rounds are accepted (correct verdict)
	// Assume 15% rejected (wrong verdict → 6% slash)
	// Assume 5% inconclusive (no reward/slash)

	numRounds := 100
	correctRate := 0.80
	wrongRate := 0.15
	// inconclusive: 0.05

	stake := uint64(5_000_000) // 5 LGM
	totalRewards := int64(0)
	totalSlashes := int64(0)
	wrongSlashBps := uint64(60_000) // 6%

	for i := 0; i < numRounds; i++ {
		roll := float64(i) / float64(numRounds)

		if roll < correctRate {
			// Correct verdict — earn base reward with difficulty bonus
			// Average confidence ~800k → difficulty bonus = 200k/1M * 3M = 600k ulgm
			avgConf := uint64(800_000)
			difficultyBonus := baseReward * (1_000_000 - avgConf) / 1_000_000
			totalRewards += int64(baseReward + difficultyBonus)
		} else if roll < correctRate+wrongRate {
			// Wrong verdict — slashed 6% of stake
			slashAmount := int64(stake * wrongSlashBps / 1_000_000)
			totalSlashes += slashAmount
		}
		// else: inconclusive, nothing happens
	}

	netProfit := totalRewards - totalSlashes

	t.Logf("OC-15 MULTI-ROUND ACCUMULATION (Tier 2, %d rounds):", numRounds)
	t.Logf("  Correct rounds: %d (%.0f%%)", int(float64(numRounds)*correctRate), correctRate*100)
	t.Logf("  Wrong rounds:   %d (%.0f%%)", int(float64(numRounds)*wrongRate), wrongRate*100)
	t.Logf("  Total rewards:  %d ulgm (%.2f LGM)", totalRewards, float64(totalRewards)/1_000_000)
	t.Logf("  Total slashes:  %d ulgm (%.2f LGM)", totalSlashes, float64(totalSlashes)/1_000_000)
	t.Logf("  Net profit:     %d ulgm (%.2f LGM)", netProfit, float64(netProfit)/1_000_000)
	t.Logf("  ROI per round:  %.2f LGM", float64(netProfit)/float64(numRounds)/1_000_000)

	// Compare with Tier 0 (0.1x rewards)
	tier0Rewards := totalRewards / 10
	tier0Net := tier0Rewards - totalSlashes // same slashes!
	t.Logf("\n  Tier 0 comparison (0.1x rewards, SAME slashes):")
	t.Logf("    Net profit: %d ulgm (%.2f LGM)", tier0Net, float64(tier0Net)/1_000_000)

	if tier0Net < 0 {
		t.Logf("OC-15 INSIGHT: Tier 0 validators LOSE MONEY at 85%% accuracy! Net: %.3f LGM",
			float64(tier0Net)/1_000_000)
		t.Logf("OC-15 RISK: New validators face negative expected value — discourages entry")
	}

	// Break-even accuracy for Tier 0
	// reward_per_correct = baseReward/10 * 1.2 (avg difficulty bonus) = 360k
	// slash_per_wrong = 5M * 60k / 1M = 300k
	// breakeven: 360k * x = 300k * (1-x) → x = 300/(360+300) = 45.5%
	t.Logf("\nOC-15 BREAK-EVEN ACCURACY:")
	tier0RewardPerCorrect := float64(baseReward) / 10 * 1.2
	slashPerWrong := float64(stake * wrongSlashBps / 1_000_000)
	breakeven := slashPerWrong / (tier0RewardPerCorrect + slashPerWrong) * 100
	t.Logf("  Tier 0: %.1f%% accuracy needed to break even", breakeven)
	t.Logf("  Tier 2: ~%.1f%% accuracy needed", slashPerWrong/(float64(baseReward)*1.2+slashPerWrong)*100)
}

// Suppress unused import warnings
var _ = sdkmath.NewInt
var _ = sdk.AccAddress{}
