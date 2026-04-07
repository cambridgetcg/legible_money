package keeper_test

// ==================================================================================
// Dokimance S1: Adversarial Proof of Truth Verification Tests
// ==================================================================================
//
// These tests attempt to BREAK the chain's truth verification via five critical
// failure modes. Each test sets up an attack scenario, executes it, checks whether
// defenses activate, and documents findings. Tests PASS regardless of whether the
// attack succeeds — the goal is to characterize the defense surface.
//
// Failure modes tested:
//   1. Cartel Coordination Attack (FARM-1)
//   2. Selective Reveal Withholding (commit-but-no-reveal slash)
//   3. Method Diversity Bypass (undeclared VerificationMethod)
//   4. Evidence Quality Gaming (junk evidence flooding)
//   5. Fact Cascade Poisoning (foundation falsification)
// ==================================================================================

import (
	"fmt"
	"testing"

	"github.com/legible-money/legible/x/knowledge/types"
)

// ==================================================================================
// TEST 1: Cartel Coordination Attack
// ==================================================================================
// THREAT: 5+ validators with >60% combined stake collude to always vote "accept"
// DEFENSE: FARM-1 (confidence.go:293-298) penalizes >95% conformity with up to 20%
//          reward reduction.
// QUESTION: Does the conformity penalty actually deter cartels, or can they still
//           control consensus outcome despite the penalty?
func TestAdversarial_CartelCoordination(t *testing.T) {
	k, ctx, mockSK := setupKeeper(t)

	// Set up 7 validators: 5 cartel members (70% stake) + 2 honest (30% stake)
	cartelAddrs := []string{
		"lgmvaloper1cartel1",
		"lgmvaloper1cartel2",
		"lgmvaloper1cartel3",
		"lgmvaloper1cartel4",
		"lgmvaloper1cartel5",
	}
	honestAddrs := []string{
		"lgmvaloper1honest1",
		"lgmvaloper1honest2",
	}

	// Cartel: 14% each = 70% total
	for _, addr := range cartelAddrs {
		mockSK.addValidator(addr, 14_000_000, 2) // 14 LGM each, Bonded tier
	}
	// Honest: 15% each = 30% total
	for _, addr := range honestAddrs {
		mockSK.addValidator(addr, 15_000_000, 2)
	}

	// Create a dubious claim the cartel wants to push through
	claim := types.Claim{
		Id:                "cartel-claim-1",
		ClaimText:         "Dubious claim that cartel wants accepted",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, err := k.CreateVerificationRound(ctx, &claim)
	if err != nil {
		t.Fatalf("CreateVerificationRound failed: %v", err)
	}

	allValidators := append(cartelAddrs, honestAddrs...)

	// Seed conformity records: cartel members have voted with majority 100% of time
	// across 25 rounds (above ConformityMinSamples=20)
	for _, addr := range cartelAddrs {
		k.SetVerifierConformity(ctx, types.VerifierConformityRecord{
			ValidatorAddr:    addr,
			TotalVotes:       25,
			ConformingVotes:  25, // 100% conformity
			LastUpdateHeight: 5,
		})
	}

	// Build round: cartel votes "accept", honest votes "reject"
	round.SelectedValidators = allValidators
	for _, addr := range cartelAddrs {
		round.Commitments[addr] = &types.VerificationCommitment{Validator: addr}
		round.Reveals[addr] = &types.VerificationReveal{
			Validator:  addr,
			Verdict:    "accept",
			Confidence: 950_000, // All identical — suspicious
		}
	}
	for _, addr := range honestAddrs {
		round.Commitments[addr] = &types.VerificationCommitment{Validator: addr}
		round.Reveals[addr] = &types.VerificationReveal{
			Validator:  addr,
			Verdict:    "reject",
			Confidence: 800_000,
		}
	}
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// --- Check 1: Does the cartel control the consensus outcome? ---
	t.Logf("Verdict: %s (agreement ratio: %d)", result.Verdict, result.AgreementRatio)

	// With 70% stake all voting accept, the cartel should win
	cartelWins := result.Verdict == "accepted"
	t.Logf("Cartel controls outcome: %v", cartelWins)

	// --- Check 2: Are FARM-1 conformity penalties applied? ---
	cartelRewards := uint64(0)
	honestSlashes := uint64(0)
	for _, addr := range cartelAddrs {
		if r, ok := result.Rewards[addr]; ok {
			cartelRewards += r
		}
	}
	for _, addr := range honestAddrs {
		if s, ok := result.Slashes[addr]; ok {
			honestSlashes += s
		}
	}
	t.Logf("Total cartel rewards: %d", cartelRewards)
	t.Logf("Total honest slashes: %d", honestSlashes)

	// --- Check 3: Compare penalized vs unpenalized rewards ---
	// A cartel member with 100% conformity and threshold=950_000 has:
	//   excess = 1_000_000 - 950_000 = 50_000
	//   penaltyRange = 1_000_000 - 950_000 = 50_000
	//   penalty = 50_000 * 200_000 / 50_000 = 200_000 (full 20%)
	// So each cartel member gets reward * (1_000_000 - 200_000) / 1_000_000 = 80%
	//
	// But the cartel STILL controls the verdict. The penalty only reduces rewards.
	if cartelWins {
		// FINDING: FARM-1 conformity penalty reduces cartel rewards by 20%, but
		// does NOT prevent the cartel from controlling the consensus outcome.
		t.Log("FINDING: Cartel with >60% stake controls consensus outcome despite FARM-1 penalty.")
		t.Log("FINDING: FARM-1 is economic deterrence only (20% reward reduction), not a consensus block.")
		t.Log("FINDING: Defense gap — no mechanism prevents a staking majority from dictating truth.")
	} else {
		// The 77% acceptance threshold acts as an additional defense layer.
		// With 70% cartel stake vs 30% honest, the agreement ratio may fall below
		// AcceptanceThreshold (770_000), producing "contested" instead of "accepted".
		t.Logf("DEFENSE WORKS: Cartel with 70%% stake got verdict '%s' — acceptance threshold (77%%) blocked it.", result.Verdict)
		t.Log("FINDING: The acceptance threshold provides a second line of defense beyond FARM-1.")
		t.Log("FINDING: However, a cartel with >77%% combined stake weight would overcome this threshold.")
		// Note: honest validators got slashed because they voted "reject" against the majority
		if honestSlashes > 0 {
			t.Logf("FINDING: Honest minority validators were slashed %d ulgm for disagreeing with 'contested' majority.", honestSlashes)
			t.Log("FINDING: In a contested outcome, minority validators still lose stake — potential honest-suppression vector.")
		}
	}

	// Verify that at least some penalty was actually applied (rewards < unpenalized baseline)
	penaltyApplied := false
	for _, addr := range cartelAddrs {
		if r, ok := result.Rewards[addr]; ok {
			if r < 3_000_000 {
				penaltyApplied = true
			}
		}
	}

	if penaltyApplied {
		t.Log("DEFENSE WORKS: FARM-1 conformity penalty was applied to cartel members.")
	} else if !cartelWins {
		// Contested verdict means no rewards/slashes for the "winning" side
		t.Log("NOTE: No cartel rewards to penalize — verdict was contested/inconclusive, no rewards distributed.")
	}

	// Test passes regardless — we're documenting the vulnerability
}

// ==================================================================================
// TEST 2: Selective Reveal Withholding
// ==================================================================================
// THREAT: Validator commits but doesn't reveal when the losing side becomes evident.
//         This lets them avoid wrong-verdict slashing at the cost of a smaller
//         missed-reveal slash.
// DEFENSE: 1% slash for missed reveal (confidence.go:257-265, fallback 10000 bps)
// QUESTION: Is the 1% missed-reveal slash large enough to make withholding unprofitable
//           compared to revealing and eating the 6% wrong-verdict slash?
func TestAdversarial_SelectiveRevealWithholding(t *testing.T) {
	k, ctx, mockSK := setupKeeper(t)

	// 3 honest validators + 1 strategic withholder
	mockSK.addValidator("lgmvaloper1withhold", 10_000_000, 2) // 10 LGM

	claim := types.Claim{
		Id:                "withhold-claim-1",
		ClaimText:         "Claim where withholder committed reject but majority accepts",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, err := k.CreateVerificationRound(ctx, &claim)
	if err != nil {
		t.Fatalf("CreateVerificationRound failed: %v", err)
	}

	allValidators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi", "lgmvaloper1withhold"}
	round.SelectedValidators = allValidators

	// All 4 commit...
	for _, v := range allValidators {
		round.Commitments[v] = &types.VerificationCommitment{
			Validator:         v,
			CommitmentHash:    "fake-hash-" + v,
			SubmittedAtHeight: 11,
		}
	}

	// ...but only 3 reveal (the withholder stays silent)
	for _, v := range []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"} {
		round.Reveals[v] = &types.VerificationReveal{
			Validator:  v,
			Verdict:    "accept",
			Confidence: 850_000,
		}
	}
	// lgmvaloper1withhold: committed but does NOT reveal

	k.SetVerificationRound(ctx, round)
	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	t.Logf("Verdict: %s, Confidence: %d", result.Verdict, result.Confidence)

	// --- Check 1: Is the withholder slashed? ---
	withholdSlash, slashed := result.Slashes["lgmvaloper1withhold"]
	if !slashed {
		t.Fatal("FINDING: Withholder was NOT slashed for missed reveal — defense broken!")
	}
	t.Logf("Withhold slash amount: %d", withholdSlash)

	// --- Check 2: Is the slash large enough to deter the strategy? ---
	// Withholder stake: 10_000_000
	// Missed reveal slash: 10_000_000 * 10_000 / 1_000_000 = 100_000 (1% of stake)
	// Wrong verdict slash: 10_000_000 * 60_000 / 1_000_000 = 600_000 (6% of stake)
	//
	// Savings from withholding: 600_000 - 100_000 = 500_000
	// The withholder saves 500_000 ulgm by not revealing.

	wrongSlash := uint64(10_000_000) * 60_000 / 1_000_000 // 600,000
	expectedMissedSlash := uint64(10_000_000) * 10_000 / 1_000_000 // 100,000

	t.Logf("Wrong verdict slash would be: %d", wrongSlash)
	t.Logf("Missed reveal slash is: %d", withholdSlash)
	t.Logf("Savings from withholding: %d", wrongSlash-withholdSlash)

	if withholdSlash < wrongSlash {
		// FINDING: Withholding is PROFITABLE compared to honest revealing on wrong side
		savings := wrongSlash - withholdSlash
		savingsPercent := (savings * 100) / wrongSlash
		t.Logf("FINDING: Selective reveal withholding is profitable.")
		t.Logf("FINDING: Withholder saves %d ulgm (%d%% of wrong-verdict slash) by not revealing.", savings, savingsPercent)
		t.Logf("FINDING: The 1%% missed-reveal slash is 6x smaller than the 6%% wrong-verdict slash.")
		t.Logf("FINDING: Rational validators will always withhold when they expect to be on the losing side.")
	}

	if withholdSlash != expectedMissedSlash {
		t.Logf("NOTE: Actual missed slash (%d) differs from expected (%d) — check fallback defaults", withholdSlash, expectedMissedSlash)
	}

	// Verify defense activates (slash exists)
	if slashed {
		t.Log("DEFENSE WORKS: Missed reveal slash IS applied.")
	}
}

// ==================================================================================
// TEST 3: Method Diversity Bypass
// ==================================================================================
// THREAT: All validators use the same deterministic method but don't declare
//         VerificationMethod, bypassing the uniformity penalty entirely.
// DEFENSE: Method diversity penalty (confidence.go:553-597) — only applies when
//          at least one reveal declares a VerificationMethod.
// QUESTION: Can validators avoid the penalty by simply omitting the method field?
func TestAdversarial_MethodDiversityBypass(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id:                "diversity-bypass-1",
		ClaimText:         "Claim where all validators use same method but don't declare it",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, err := k.CreateVerificationRound(ctx, &claim)
	if err != nil {
		t.Fatalf("CreateVerificationRound failed: %v", err)
	}

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	round.SelectedValidators = validators

	// Attack: all validators submit identical verdicts and confidence
	// with NO VerificationMethod declared
	for _, v := range validators {
		round.Commitments[v] = &types.VerificationCommitment{Validator: v}
		round.Reveals[v] = &types.VerificationReveal{
			Validator:          v,
			Verdict:            "accept",
			Confidence:         900_000,
			VerificationMethod: "", // Deliberately omitted
		}
	}
	k.SetVerificationRound(ctx, round)

	resultNoDeclare, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult (no method) failed: %v", err)
	}

	t.Logf("No method declared — Confidence: %d, DiversityPenalty: %d, UniqueMethodCount: %d",
		resultNoDeclare.Confidence, resultNoDeclare.MethodDiversityPenalty, resultNoDeclare.UniqueMethodCount)

	// Now test with the SAME scenario but with methods declared (should trigger penalty)
	claim2 := types.Claim{
		Id:                "diversity-declared-1",
		ClaimText:         "Same claim but validators declare their method",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim2)
	round2, err := k.CreateVerificationRound(ctx, &claim2)
	if err != nil {
		t.Fatalf("CreateVerificationRound (declared) failed: %v", err)
	}

	round2.SelectedValidators = validators
	for _, v := range validators {
		round2.Commitments[v] = &types.VerificationCommitment{Validator: v}
		round2.Reveals[v] = &types.VerificationReveal{
			Validator:          v,
			Verdict:            "accept",
			Confidence:         900_000,
			VerificationMethod: "llm_query", // All same method — should trigger penalty
		}
	}
	k.SetVerificationRound(ctx, round2)

	resultDeclared, err := k.AggregateVerificationResult(ctx, round2)
	if err != nil {
		t.Fatalf("AggregateVerificationResult (declared) failed: %v", err)
	}

	t.Logf("Method declared (all same) — Confidence: %d, DiversityPenalty: %d, UniqueMethodCount: %d",
		resultDeclared.Confidence, resultDeclared.MethodDiversityPenalty, resultDeclared.UniqueMethodCount)

	// --- Analysis ---
	noPenalty := resultNoDeclare.MethodDiversityPenalty == 0
	hasPenalty := resultDeclared.MethodDiversityPenalty != 0

	if noPenalty {
		// FINDING: No diversity penalty when methods are undeclared
		t.Log("FINDING: Method diversity penalty is NOT applied when VerificationMethod is omitted.")
		t.Log("FINDING: Validators can bypass the uniformity detection by simply not declaring their method.")
	}

	if hasPenalty {
		t.Logf("DEFENSE WORKS (when declared): Penalty of %d applied for uniform methods.", resultDeclared.MethodDiversityPenalty)
	}

	if noPenalty && hasPenalty {
		confidenceDiff := int64(resultNoDeclare.Confidence) - int64(resultDeclared.Confidence)
		t.Logf("FINDING: Confidence gap between bypass and honest declaration: %d basis points.", confidenceDiff)
		t.Log("FINDING: This creates a perverse incentive — validators are penalized for transparency.")
		t.Log("FINDING: Recommended fix: require VerificationMethod field or apply default penalty when omitted.")
	}
}

// ==================================================================================
// TEST 4: Evidence Quality Gaming
// ==================================================================================
// THREAT: Submit 100 low-quality evidence pieces; each defaults to 10% reliability,
//         but aggregate may still boost confidence via the averaging formula.
// DEFENSE: Evidence bonus capped at MaxEvidenceBonus = 15% (confidence.go:452-519).
// QUESTION: Does the cap hold? Does junk evidence still produce a meaningful bonus?
func TestAdversarial_EvidenceQualityGaming(t *testing.T) {
	k, ctx, _, mockEM := setupKeeperWithEvidence(t)

	claimId := "evidence-gaming-1"

	// Attack: flood the claim with 100 pieces of junk evidence
	// All have reliability=0, which defaults to 100_000 (10%) in computeEvidenceBonus
	for i := 0; i < 100; i++ {
		mockEM.addEvidence(claimId, types.EvidenceRecord{
			Id:           fmt.Sprintf("junk-ev-%d", i),
			ClaimId:      claimId,
			EvidenceType: "document",  // document type strength = 400_000 (conservative default)
			Status:       "submitted", // Not verified, just submitted
			Reliability:  0,           // Unaudited — will default to 100_000 (10%)
		})
	}

	claim := types.Claim{
		Id:                claimId,
		ClaimText:         "Claim flooded with 100 pieces of junk evidence",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claim)
	round, err := k.CreateVerificationRound(ctx, &claim)
	if err != nil {
		t.Fatalf("CreateVerificationRound failed: %v", err)
	}

	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	round.SelectedValidators = validators
	for _, v := range validators {
		round.Commitments[v] = &types.VerificationCommitment{Validator: v}
		round.Reveals[v] = &types.VerificationReveal{
			Validator:  v,
			Verdict:    "accept",
			Confidence: 700_000, // Moderate baseline confidence
		}
	}
	k.SetVerificationRound(ctx, round)

	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	t.Logf("Evidence bonus: %d (cap is 150_000)", result.EvidenceBonus)
	t.Logf("Final confidence: %d (base would be ~700_000)", result.Confidence)

	// --- Check 1: Is the cap enforced? ---
	if result.EvidenceBonus > 150_000 {
		t.Fatalf("FINDING: Evidence bonus %d EXCEEDS cap of 150_000 — cap is broken!", result.EvidenceBonus)
	}
	t.Log("DEFENSE WORKS: Evidence bonus cap (15%) is enforced.")

	// --- Check 2: Does junk evidence still produce a bonus? ---
	// Each piece: typeStrength=400_000 (default for unknown), reliability=100_000 (10% default)
	// Score per piece: 400_000 * 100_000 / 1_000_000 = 40_000
	// Average across 100 pieces: 40_000 (same — averaging doesn't increase it)
	// 40_000 < 150_000 cap, so bonus = 40_000
	expectedJunkBonus := uint64(400_000) * 100_000 / 1_000_000 // = 40_000

	if result.EvidenceBonus > 0 {
		t.Logf("FINDING: Junk evidence (100 unaudited documents) produces a %d bps (%d.%d%%) confidence bonus.",
			result.EvidenceBonus, result.EvidenceBonus/10_000, (result.EvidenceBonus%10_000)/1000)
		t.Log("FINDING: Submitting unaudited evidence provides a free confidence boost with zero epistemic value.")
		t.Log("FINDING: The averaging formula means quantity doesn't help, but even 1 junk item gives a bonus.")
	}

	if result.EvidenceBonus == expectedJunkBonus {
		t.Logf("CONFIRMED: Bonus matches expected junk value of %d (typeStrength * defaultReliability).", expectedJunkBonus)
	}

	// --- Check 3: Compare with zero evidence ---
	claimNoEv := types.Claim{
		Id:                "no-evidence-baseline",
		ClaimText:         "Same claim, no evidence",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, &claimNoEv)
	roundNoEv, _ := k.CreateVerificationRound(ctx, &claimNoEv)
	roundNoEv.SelectedValidators = validators
	for _, v := range validators {
		roundNoEv.Commitments[v] = &types.VerificationCommitment{Validator: v}
		roundNoEv.Reveals[v] = &types.VerificationReveal{
			Validator:  v,
			Verdict:    "accept",
			Confidence: 700_000,
		}
	}
	k.SetVerificationRound(ctx, roundNoEv)

	resultNoEv, err := k.AggregateVerificationResult(ctx, roundNoEv)
	if err != nil {
		t.Fatalf("AggregateVerificationResult (no evidence) failed: %v", err)
	}

	if result.Confidence > resultNoEv.Confidence {
		boost := result.Confidence - resultNoEv.Confidence
		t.Logf("FINDING: Junk evidence boosted confidence by %d bps over the no-evidence baseline.", boost)
	}
}

// ==================================================================================
// TEST 5: Fact Cascade Poisoning
// ==================================================================================
// THREAT: A false foundational claim gets verified, then many claims cite it.
//         When the foundation is falsified, 22% cascade reduction hits all dependents.
// DEFENSE: Cascade reduction (fact_status.go:131-185) at CascadeReductionBps = 220_000 (22%).
// QUESTION: Do dependents drop below the provisional threshold (10%)? Does cascade
//           propagate correctly?
func TestAdversarial_FactCascadePoisoning(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Step 1: Create a "foundation" fact that will later be falsified
	foundationFact := &types.Fact{
		Id:         "foundation-fact-1",
		ClaimText:  "False foundational claim that initially passes verification",
		Domain:     "physics",
		Confidence: 800_000, // 80% — high confidence
		Status:     "verified",
	}
	k.SetFact(ctx, foundationFact)

	// Step 2: Create 5 dependent facts that cite the foundation
	dependentIds := make([]string, 5)
	for i := 0; i < 5; i++ {
		depId := fmt.Sprintf("dependent-fact-%d", i)
		dependentIds[i] = depId

		depFact := &types.Fact{
			Id:         depId,
			ClaimText:  fmt.Sprintf("Dependent claim #%d citing foundation", i),
			Domain:     "physics",
			Confidence: 750_000, // 75% confidence
			Status:     "verified",
			References: []string{"foundation-fact-1"},
		}
		k.SetFact(ctx, depFact)

		// Create the bidirectional reference
		ref := &types.FactReference{
			FromFactId:    depId,
			ToFactId:      "foundation-fact-1",
			ReferenceType: "supports",
			Weight:        1_000_000,
		}
		k.SetFactReference(ctx, ref)
	}

	// Step 3: Falsify the foundation via dispute resolution
	// Use a 55% reduction (strong overturn)
	newConf, err := k.ResolveFactDispute(ctx, "foundation-fact-1", "overturned", 550_000)
	if err != nil {
		t.Fatalf("ResolveFactDispute failed: %v", err)
	}

	t.Logf("Foundation after overturn — confidence: %d, expected: %d",
		newConf, 800_000*(1_000_000-550_000)/1_000_000) // 800_000 * 45% = 360_000

	// Check foundation status
	foundationAfter, found := k.GetFact(ctx, "foundation-fact-1")
	if !found {
		t.Fatal("Foundation fact not found after dispute")
	}
	t.Logf("Foundation status: %s, confidence: %d", foundationAfter.Status, foundationAfter.Confidence)

	// Step 4: Apply cascade reduction to dependents
	affected, err := k.CascadeFactConfidence(ctx, "foundation-fact-1", 220_000) // CascadeReductionBps
	if err != nil {
		t.Fatalf("CascadeFactConfidence failed: %v", err)
	}

	t.Logf("Cascade affected %d facts", len(affected))

	// --- Check each dependent ---
	provisionalThreshold := uint64(100_000) // 10% — below this, fact is considered unreliable
	_ = uint64(100_000) // RetractionThresholdBps = 10% (used in analysis below)
	revokedCount := 0
	contestedCount := 0

	for _, depId := range dependentIds {
		dep, found := k.GetFact(ctx, depId)
		if !found {
			t.Errorf("Dependent fact %s not found after cascade", depId)
			continue
		}

		// Expected: 750_000 * (1_000_000 - 220_000) / 1_000_000 = 750_000 * 780_000 / 1_000_000 = 585_000
		expectedConf := uint64(750_000) * (1_000_000 - 220_000) / 1_000_000
		t.Logf("Dependent %s — confidence: %d (expected ~%d), status: %s",
			depId, dep.Confidence, expectedConf, dep.Status)

		if dep.Status == "revoked" {
			revokedCount++
		}
		if dep.Status == "contested" {
			contestedCount++
		}

		if dep.Confidence < provisionalThreshold {
			t.Logf("FINDING: Dependent %s dropped below provisional threshold (%d < %d)", depId, dep.Confidence, provisionalThreshold)
		}
	}

	t.Logf("After cascade: %d revoked, %d contested out of %d dependents", revokedCount, contestedCount, len(dependentIds))

	// --- Check cascade defense effectiveness ---
	if len(affected) == len(dependentIds) {
		t.Log("DEFENSE WORKS: Cascade reduction applied to all dependent facts.")
	} else if len(affected) == 0 {
		t.Log("FINDING: Cascade reduction did NOT affect any dependents — references may not be indexed.")
	} else {
		t.Logf("FINDING: Cascade only affected %d of %d dependents.", len(affected), len(dependentIds))
	}

	// A single 22% cascade from 75% yields 58.5%, which stays above retraction threshold.
	// FINDING: A single cascade is not enough to revoke dependents.
	// It would take multiple cascades or a very low starting confidence.
	if revokedCount == 0 {
		t.Log("FINDING: Single 22% cascade from 75% confidence does NOT revoke dependents (they stay at ~58.5%).")
		t.Log("FINDING: An attacker could build a dependency tree, get foundation falsified, and dependents")
		t.Log("         still survive with 58.5% confidence — enough to remain 'contested' but not revoked.")
	}

	// --- Check: What if dependents had lower starting confidence? ---
	// At what starting confidence does a single 22% cascade push below retraction (10%)?
	// startConf * 0.78 < 100_000 → startConf < 128_205
	// So only facts already below ~12.8% confidence would be revoked by cascade.
	t.Log("FINDING: 22% cascade only revokes facts already below ~12.8% confidence.")
	t.Log("FINDING: For poisoning to cascade-revoke dependents, multiple rounds of dispute are needed,")
	t.Log("         or the cascade rate needs to be significantly higher.")
}
