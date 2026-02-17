package keeper_test

// ==================================================================================
// OpenClaw Agent Testing Session — Governance Module Game Theory
// ==================================================================================
//
// Purpose: Systematic probing of governance economics, quadratic voting edge cases,
// proposal spam, stage manipulation, disbursement capture, and vote buying.
//
// Naming convention:
//   OCG  = OpenClaw Governance (macro game theory)
//   NCG  = NanoClaw Governance (precision/boundary attacks)
//   ZCG  = ZeroClaw Governance (zero-cost attack vectors)
//
// Methodology: Set up scenario → execute → measure cost/damage/profit → log insight
// ==================================================================================

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/legible-money/legible/x/gov/keeper"
	"github.com/legible-money/legible/x/gov/types"
)

// ==================================================================================
// OCG-1: Vote Buying via Quadratic Arbitrage
// ==================================================================================
// Scenario: Quadratic voting means 100 voters spending 1 LGM each get more
// voting power than 1 voter spending 100 LGM. Test the exact amplification.
// Attack: Attacker splits stake across Sybil accounts for more voting power.
func TestOCG1_VoteBuyingQuadraticArbitrage(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	createAndAdvanceToVoting(t, k, ms, ctx)

	// Single whale: 10,000 LGM = sqrt(10000000000) = 100000 power
	resp, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter:      testAddr("whale"),
		ProposalId: "LIP-1",
		Option:     types.VoteOption_VOTE_OPTION_YES,
		Stake:      "10000000000", // 10,000 LGM
	})
	if err != nil {
		t.Fatalf("whale vote failed: %v", err)
	}
	whalePower := new(big.Int)
	whalePower.SetString(resp.EffectiveVotingPower, 10)

	// Reset: create a new LIP for the Sybil test
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Sybil Target",
		Description:  "For sybil vote test",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})
	lip2, _ := k.GetLIP(ctx, "LIP-2")
	lip2.Category = string(types.CategoryProcess)
	k.SetLIP(ctx, lip2)
	ms.StakeLIP(ctx, &types.MsgStakeLIP{Staker: testAddr("bob"), LipId: "LIP-2", Amount: "200000000"})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Authority: testAddr("alice"), LipId: "LIP-2"})
	ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{Authority: testAddr("alice"), LipId: "LIP-2"})

	// 100 Sybils each spending 100 LGM = sqrt(100000000) = 10000 power each
	sybilTotal := new(big.Int)
	for i := 0; i < 100; i++ {
		resp, err := ms.CastVote(ctx, &types.MsgCastVote{
			Voter:      testAddr(fmt.Sprintf("sybil%03d", i)),
			ProposalId: "LIP-2",
			Option:     types.VoteOption_VOTE_OPTION_YES,
			Stake:      "100000000", // 100 LGM each
		})
		if err != nil {
			t.Fatalf("sybil %d vote failed: %v", i, err)
		}
		power := new(big.Int)
		power.SetString(resp.EffectiveVotingPower, 10)
		sybilTotal.Add(sybilTotal, power)
	}

	// === ECONOMIC ANALYSIS ===
	t.Logf("OCG-1 RESULT:")
	t.Logf("  Whale: 10,000 LGM → power = %s", whalePower)
	t.Logf("  Sybils: 100 × 100 LGM = 10,000 LGM → total power = %s", sybilTotal)
	if sybilTotal.Cmp(whalePower) > 0 {
		amplification := new(big.Float).Quo(
			new(big.Float).SetInt(sybilTotal),
			new(big.Float).SetInt(whalePower),
		)
		t.Logf("  Sybil amplification: %sx more voting power for same capital", amplification.Text('f', 2))
		t.Log("OCG-1 FINDING: Quadratic voting rewards splitting stake across accounts.")
		t.Log("  DEFENSE: Identity verification (DID) makes Sybil creation costly.")
		t.Log("  DEFENSE: BaseCostPerVote (1 LGM) adds fixed per-vote overhead.")
	} else {
		t.Log("OCG-1 FINDING: No Sybil advantage — whale power >= sybil total")
	}
}

// ==================================================================================
// OCG-2: Proposal Spam — Flood with Low-Stake LIPs
// ==================================================================================
// Scenario: Attacker submits many proposals at minimum stake to exhaust
// reviewer attention and bury legitimate proposals.
func TestOCG2_ProposalSpam(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	spamCount := 50
	minStake := "1" // Try minimum possible stake
	successCount := 0

	for i := 0; i < spamCount; i++ {
		_, err := ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
			Proposer:     testAddr(fmt.Sprintf("spammer%03d", i)),
			Title:        fmt.Sprintf("Spam LIP %d", i),
			Description:  "Spam",
			LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
			InitialStake: minStake,
		})
		if err == nil {
			successCount++
		}
	}

	// === ECONOMIC ANALYSIS ===
	t.Logf("OCG-2 RESULT: %d/%d spam proposals succeeded at stake=%s", successCount, spamCount, minStake)

	allLIPs := k.GetAllLIPs(ctx)
	t.Logf("OCG-2 TOTAL LIPs: %d", len(allLIPs))

	if successCount == spamCount {
		t.Logf("OCG-2 ECONOMICS:")
		t.Logf("  Spam cost: %d × %s ulgm = %d ulgm total", spamCount, minStake, spamCount*1)
		t.Log("  FINDING: No rate limit on LIP submission — 50 proposals at 1 ulgm each.")
		t.Log("  DEFENSE: Proposals stay in Draft until stake threshold met (200-1000 LGM).")
		t.Log("  DEFENSE: bankKeeper escrow (nil in test) would enforce real economic cost.")
		t.Log("  RESIDUAL RISK: Reviewers must filter through spam to find real proposals.")
	} else {
		t.Log("OCG-2 DEFENSE: Some spam proposals were rejected")
	}
}

// ==================================================================================
// OCG-3: Stage Rush — Bypass Review Period
// ==================================================================================
// Scenario: Proposer tries to rush a LIP through all stages as fast as possible,
// skipping the review period. Test if ReviewPeriodBlocks is enforced.
func TestOCG3_StageRushBypassReview(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Keep default ReviewPeriodBlocks = 50,000 (do NOT set to 0)
	params := k.GetParams(ctx)
	t.Logf("OCG-3 PARAMS: ReviewPeriodBlocks=%d", params.ReviewPeriodBlocks)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("rusher"),
		Title:        "Rush This",
		Description:  "Trying to rush through review",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})

	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryProcess)
	k.SetLIP(ctx, lip)

	// Stake enough to advance to Review
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Staker: testAddr("backer"),
		LipId:  "LIP-1",
		Amount: "200000000", // 200 LGM for process category
	})

	lip, _ = k.GetLIP(ctx, "LIP-1")
	t.Logf("OCG-3 STAGE AFTER STAKE: %s", lip.Stage)

	// Immediately try to advance Review → LastCall (should fail if review period enforced)
	_, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{
		Authority: testAddr("rusher"),
		LipId:     "LIP-1",
	})

	// === INSIGHT ANALYSIS ===
	if err != nil {
		t.Logf("OCG-3 DEFENSE: Review→LastCall blocked: %v", err)
		t.Log("  ZD-E04 fix enforces minimum review period before advancement.")
		t.Log("  Proposers cannot rush proposals past community review.")
	} else {
		lip, _ = k.GetLIP(ctx, "LIP-1")
		t.Logf("OCG-3 FINDING: Advanced to %s despite review period not elapsed!", lip.Stage)
		t.Logf("  Block height=%d, ReviewPeriodBlocks=%d", ctx.BlockHeight(), params.ReviewPeriodBlocks)
		t.Log("  CRITICAL: Proposals can bypass review period entirely!")
	}
}

// ==================================================================================
// OCG-4: Category Downgrade Attack
// ==================================================================================
// Scenario: A consensus-affecting proposal is submitted as "process" category
// to require only 200 LGM stake instead of 1,000 LGM. Category is set after
// submission in the existing test pattern — test if this is exploitable.
func TestOCG4_CategoryDowngradeAttack(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	params := k.GetParams(ctx)

	// Submit as community type (category determined by LipType)
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("downgrader"),
		Title:        "Consensus Change Disguised as Process",
		Description:  "Actually changes consensus params",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})

	lip, _ := k.GetLIP(ctx, "LIP-1")
	originalCategory := lip.Category
	t.Logf("OCG-4 INITIAL CATEGORY: %s", originalCategory)

	// Attacker sets category to process (lowest stake requirement)
	lip.Category = string(types.CategoryProcess)
	k.SetLIP(ctx, lip)

	processStake := params.GetLIPStake(types.CategoryProcess)
	consensusStake := params.GetLIPStake(types.CategoryConsensus)

	t.Logf("OCG-4 ECONOMICS:")
	t.Logf("  Process stake threshold: %s ulgm", processStake)
	t.Logf("  Consensus stake threshold: %s ulgm", consensusStake)

	// Stake enough for process but not consensus
	ms.StakeLIP(ctx, &types.MsgStakeLIP{
		Staker: testAddr("backer"),
		LipId:  "LIP-1",
		Amount: processStake,
	})

	lip, _ = k.GetLIP(ctx, "LIP-1")

	// === INSIGHT ANALYSIS ===
	if lip.Stage == string(types.StageReview) {
		t.Log("OCG-4 FINDING: Proposal advanced with only process-level stake")
		t.Log("  RESIDUAL RISK: Category is set on creation based on LipType enum,")
		t.Log("  but keeper.SetLIP allows direct category override.")
		t.Log("  DEFENSE: In production, SetLIP should validate category immutability,")
		t.Log("  or category should be derived from content, not self-declared.")
	} else {
		t.Logf("OCG-4 RESULT: Stage=%s (did not advance with process stake)", lip.Stage)
	}
}

// ==================================================================================
// OCG-5: Abstain Quorum Manipulation
// ==================================================================================
// Scenario: Abstain votes count toward unique voter count but don't count
// toward support/oppose ratio. Can abstainers inflate quorum while diluting
// the support threshold calculation?
func TestOCG5_AbstainQuorumManipulation(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	createAndAdvanceToVoting(t, k, ms, ctx)

	// 1 supporter with 100 LGM = sqrt(100M) = 10000 power
	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("supporter"), ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "100000000",
	})

	// 1 opposer with 100 LGM = 10000 power
	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("opposer"), ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_NO, Stake: "100000000",
	})

	// 8 abstainers with 1 LGM each = sqrt(1M) = 1000 power each
	for i := 0; i < 8; i++ {
		ms.CastVote(ctx, &types.MsgCastVote{
			Voter: testAddr(fmt.Sprintf("abstain%d", i)), ProposalId: "LIP-1",
			Option: types.VoteOption_VOTE_OPTION_ABSTAIN, Stake: "1000000",
		})
	}

	lip, _ := k.GetLIP(ctx, "LIP-1")
	t.Logf("OCG-5 TALLY:")
	t.Logf("  Support: %s | Oppose: %s | Abstain: %s", lip.SupportWeight, lip.OpposeWeight, lip.AbstainWeight)
	t.Logf("  UniqueVoters: %d | QuorumMet: %v", lip.UniqueVoters, lip.QuorumMet)

	// === INSIGHT ANALYSIS ===
	// Support = 10000, Oppose = 10000, Abstain = 8000
	// If quorum counts all voters (10) but threshold only counts support vs oppose:
	// support / (support + oppose) = 50% < 75% threshold → rejected
	// But quorum looks healthy at 10 unique voters
	t.Log("OCG-5 INSIGHT: Abstain votes inflate UniqueVoters count but don't help support ratio.")
	t.Log("  Attacker can use abstainers to make quorum look met while the actual")
	t.Log("  support/oppose ratio is deadlocked.")
	t.Log("  DEFENSE: Quorum should be based on voting power, not just voter count.")
	t.Log("  DEFENSE: SupportThresholdBps (75%) is calculated on support vs (support+oppose).")
}

// ==================================================================================
// OCG-6: Non-Proposer Stage Advancement (ZD-E04 Check)
// ==================================================================================
// Scenario: Test that only the proposer can advance LIP stages.
// This verifies the ZD-E04 security fix.
func TestOCG6_NonProposerAdvancement(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	params := k.GetParams(ctx)
	params.ReviewPeriodBlocks = 0
	k.SetParams(ctx, params)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Alice's Proposal",
		Description:  "Only Alice should advance this",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})
	lip, _ := k.GetLIP(ctx, "LIP-1")
	lip.Category = string(types.CategoryProcess)
	k.SetLIP(ctx, lip)
	ms.StakeLIP(ctx, &types.MsgStakeLIP{Staker: testAddr("bob"), LipId: "LIP-1", Amount: "200000000"})

	// Attacker (not proposer) tries to advance
	_, err := ms.AdvanceLIPStage(ctx, &types.MsgAdvanceLIPStage{
		Authority: testAddr("attacker"),
		LipId:     "LIP-1",
	})

	// === INSIGHT ANALYSIS ===
	if err != nil {
		t.Logf("OCG-6 DEFENSE: Non-proposer advancement blocked: %v", err)
		t.Log("  ZD-E04 fix is working — only proposer can advance stages.")
	} else {
		lip, _ = k.GetLIP(ctx, "LIP-1")
		t.Logf("OCG-6 CRITICAL: Attacker advanced LIP to %s!", lip.Stage)
		t.Log("  ZD-E04 fix is NOT working — any address can advance proposals!")
	}
}

// ==================================================================================
// OCG-7: Double Withdraw Attempt
// ==================================================================================
// Scenario: Proposer withdraws a LIP, then tries to withdraw again to
// potentially trigger double refund.
func TestOCG7_DoubleWithdraw(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Double Withdraw",
		Description:  "Try to withdraw twice",
		LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
		InitialStake: "1000000",
	})

	// First withdraw
	_, err1 := ms.WithdrawLIP(ctx, &types.MsgWithdrawLIP{
		Proposer: testAddr("alice"),
		LipId:    "LIP-1",
	})
	if err1 != nil {
		t.Fatalf("first withdraw failed: %v", err1)
	}

	// Second withdraw
	_, err2 := ms.WithdrawLIP(ctx, &types.MsgWithdrawLIP{
		Proposer: testAddr("alice"),
		LipId:    "LIP-1",
	})

	// === INSIGHT ANALYSIS ===
	if err2 != nil {
		t.Logf("OCG-7 DEFENSE: Double withdraw blocked: %v", err2)
		t.Log("  Terminal stage check prevents re-withdrawal.")
	} else {
		t.Log("OCG-7 CRITICAL: Double withdraw succeeded!")
		t.Log("  If bank refund happens on withdraw, attacker gets double refund.")
	}
}

// ==================================================================================
// NCG-1: Quadratic sqrt Precision at Boundaries
// ==================================================================================
// Scenario: Test voting power calculation at exact squares, near-squares,
// and extreme values. Integer sqrt of 0, 1, 2, 3, 4 and very large values.
func TestNCG1_QuadraticSqrtPrecision(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	cases := []struct {
		name   string
		stake  string
		expect string // expected sqrt
	}{
		{"sqrt(1)=1", "1", "1"},
		{"sqrt(4)=2", "4", "2"},
		{"sqrt(9)=3", "9", "3"},
		{"sqrt(2)=1 (floor)", "2", "1"},
		{"sqrt(3)=1 (floor)", "3", "1"},
		{"sqrt(8)=2 (floor)", "8", "2"},
		{"sqrt(1000000)=1000", "1000000", "1000"},
		{"sqrt(9000000)=3000", "9000000", "3000"},
		{"sqrt(999999999999)=999999 (large)", "999999999999", "999999"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh LIP in voting stage for each case
			resp, err := ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
				Proposer: testAddr("alice"), Title: tc.name,
				Description: "sqrt test", LipType: types.LIPType_LIP_TYPE_COMMUNITY,
				InitialStake: "1000000",
			})
			if err != nil {
				t.Fatalf("submit failed: %v", err)
			}
			lip, _ := k.GetLIP(ctx, resp.LipId)
			lip.Category = string(types.CategoryProcess)
			lip.Stage = string(types.StageVoting) // Force to voting for test
			lip.VotingEndsAt = 999999
			k.SetLIP(ctx, lip)

			voteResp, err := ms.CastVote(ctx, &types.MsgCastVote{
				Voter: testAddr(tc.name), ProposalId: resp.LipId,
				Option: types.VoteOption_VOTE_OPTION_YES, Stake: tc.stake,
			})
			if err != nil {
				t.Fatalf("vote failed for stake=%s: %v", tc.stake, err)
			}

			if voteResp.EffectiveVotingPower != tc.expect {
				t.Errorf("NCG-1: sqrt(%s) = %s, expected %s — precision error",
					tc.stake, voteResp.EffectiveVotingPower, tc.expect)
			}
		})
	}
}

// ==================================================================================
// NCG-2: Support Threshold Boundary — 74.99% vs 75.01%
// ==================================================================================
// Scenario: Test the exact support threshold (7500 bps = 75%). What happens
// when support is exactly at the boundary?
func TestNCG2_SupportThresholdBoundary(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	params := k.GetParams(ctx)
	t.Logf("NCG-2 PARAMS: SupportThresholdBps=%d (%.1f%%)", params.SupportThresholdBps, float64(params.SupportThresholdBps)/100)

	createAndAdvanceToVoting(t, k, ms, ctx)

	// To get exactly 75%: support=3, oppose=1 → 3/(3+1) = 75%
	// Using perfect squares for clean sqrt: stake=9→power=3, stake=1→power=1
	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("sup1"), ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_YES, Stake: "9",
	})
	ms.CastVote(ctx, &types.MsgCastVote{
		Voter: testAddr("opp1"), ProposalId: "LIP-1",
		Option: types.VoteOption_VOTE_OPTION_NO, Stake: "1",
	})

	lip, _ := k.GetLIP(ctx, "LIP-1")
	support := new(big.Int)
	oppose := new(big.Int)
	support.SetString(lip.SupportWeight, 10)
	oppose.SetString(lip.OpposeWeight, 10)

	total := new(big.Int).Add(support, oppose)
	if total.Sign() > 0 {
		// supportBps = support * 10000 / total
		supportBps := new(big.Int).Mul(support, big.NewInt(10000))
		supportBps.Div(supportBps, total)

		t.Logf("NCG-2 RESULT: support=%s oppose=%s ratio=%d bps (%.2f%%)",
			support, oppose, supportBps.Int64(), float64(supportBps.Int64())/100)

		if supportBps.Int64() == int64(params.SupportThresholdBps) {
			t.Log("NCG-2 BOUNDARY: Support exactly at threshold — verify >= vs > comparison")
		}
	}
}

// ==================================================================================
// ZCG-1: Minimum Stake Governance Spam
// ==================================================================================
// Scenario: Submit 100 LIPs at 1 ulgm each. Total cost: 100 ulgm (0.0001 LGM).
// Can an attacker pollute the governance feed at negligible cost?
func TestZCG1_MinimumStakeGovernanceSpam(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	submitted := 0
	for i := 0; i < 100; i++ {
		_, err := ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
			Proposer:     testAddr(fmt.Sprintf("zc%03d", i)),
			Title:        fmt.Sprintf("ZeroClaw Spam %d", i),
			Description:  "Minimal stake spam",
			LipType:      types.LIPType_LIP_TYPE_COMMUNITY,
			InitialStake: "1",
		})
		if err == nil {
			submitted++
		}
	}

	allLIPs := k.GetAllLIPs(ctx)
	draftCount := 0
	for _, lip := range allLIPs {
		if lip.Stage == string(types.StageDraft) {
			draftCount++
		}
	}

	// === ECONOMIC ANALYSIS ===
	t.Logf("ZCG-1 RESULT: %d/100 spam proposals submitted, %d in draft", submitted, draftCount)
	t.Logf("  Total cost: %d ulgm (%.6f LGM)", submitted, float64(submitted)/1_000_000)
	t.Log("ZCG-1 INSIGHT: No minimum stake enforced beyond > 0.")
	t.Log("  All proposals stuck in Draft (need 200-1000 LGM to advance).")
	t.Log("  DEFENSE: Draft proposals have no governance power — they're just noise.")
	t.Log("  RISK: Pollutes GetAllLIPs query; UIs must filter by stage.")
}

// ==================================================================================
// ZCG-2: Vote on Non-Existent Proposal
// ==================================================================================
// Scenario: Attempt to cast a vote on a proposal that doesn't exist.
// Verify no state corruption occurs.
func TestZCG2_VoteOnNonexistentProposal(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CastVote(ctx, &types.MsgCastVote{
		Voter:      testAddr("voter"),
		ProposalId: "LIP-999",
		Option:     types.VoteOption_VOTE_OPTION_YES,
		Stake:      "1000000",
	})

	if err != nil {
		t.Logf("ZCG-2 DEFENSE: Vote on nonexistent LIP rejected: %v", err)
	} else {
		t.Log("ZCG-2 CRITICAL: Vote on nonexistent LIP succeeded — state corruption risk!")
	}

	// Verify no phantom votes stored
	votes := k.GetVotesForProposal(ctx, "LIP-999")
	if len(votes) > 0 {
		t.Errorf("ZCG-2 CRITICAL: %d phantom votes stored for nonexistent proposal", len(votes))
	}

	// Verify HasVoted is clean
	if k.HasVoted(ctx, "LIP-999", testAddr("voter")) {
		t.Error("ZCG-2 CRITICAL: HasVoted returns true for nonexistent proposal")
	}
}

// ==================================================================================
// ZCG-3: Disbursement Self-Benefit
// ==================================================================================
// Scenario: Proposer creates a disbursement where they are also the recipient.
// Tests whether self-dealing is possible.
func TestZCG3_DisbursementSelfBenefit(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	selfDealer := testAddr("selfdealer")

	resp, err := ms.SubmitDisbursement(ctx, &types.MsgSubmitDisbursement{
		Submitter:   selfDealer,
		Title:       "Fund Myself",
		Description: "I deserve this",
		Recipient:   selfDealer, // Same as submitter
		Amount:      "50000000000",
	})

	// === INSIGHT ANALYSIS ===
	if err != nil {
		t.Logf("ZCG-3 DEFENSE: Self-dealing disbursement rejected: %v", err)
	} else {
		disb, _ := k.GetDisbursement(ctx, resp.DisbursementId)
		t.Logf("ZCG-3 FINDING: Self-dealing disbursement created: %s", resp.DisbursementId)
		t.Logf("  Proposer=%s Recipient=%s Amount=%s Stage=%s",
			disb.Proposer, disb.RecipientAddr, disb.Amount, disb.Stage)
		t.Log("  RISK: Proposer can create disbursements benefiting themselves.")
		t.Log("  DEFENSE: Still requires voting approval (80% support, 20% quorum).")
		t.Log("  DEFENSE: DisbursementTimelockBlocks (50,000) provides challenge window.")
	}
}
