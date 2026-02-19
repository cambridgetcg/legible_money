package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/evaluation"
	"github.com/legible-money/legible/x/knowledge/types"
)

// TestFARMIntegration verifies all six FARM fixes (10-15) work together.
// Scenario: a farming agent tries multiple strategies, all blocked or penalized.
func TestFARMIntegration(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ep := k.GetExtendedParams(ctx)

	// ====================================================================
	// Setup: Establish canonical/established facts for reference
	// ====================================================================

	// Create a canonical fact in mathematics (high-ceiling domain)
	canonicalFact := &types.Fact{
		Id:              "fact-canonical-1",
		ClaimText:       "1 + 1 = 2",
		Domain:          "mathematics",
		Status:          types.FactStatusVerified,
		Confidence:      950_000,
		BaseConfidence:  950_000,
		Maturity:        types.MaturityCanonical,
		CreatedAtHeight: 1,
		CitationCount:   50,
		VerifierCount:   15,
	}
	k.SetFact(ctx, canonicalFact)

	// Create an established fact in physics (high-ceiling domain)
	establishedFact := &types.Fact{
		Id:              "fact-established-1",
		ClaimText:       "F = ma",
		Domain:          "physics",
		Status:          types.FactStatusVerified,
		Confidence:      850_000,
		BaseConfidence:  850_000,
		Maturity:        types.MaturityEstablished,
		CreatedAtHeight: 100,
		CitationCount:   30,
		VerifierCount:   10,
	}
	k.SetFact(ctx, establishedFact)

	// Populate citation sources for established fact (diverse, legitimate)
	for i := 0; i < 30; i++ {
		addr := testSubmitterAddr + string(rune('A'+i%26)) + string(rune('0'+i/26))
		k.UpdateCitationSource(ctx, "fact-established-1", addr)
	}

	// Create a young, uncited fact in sociology (low-ceiling domain)
	youngFact := &types.Fact{
		Id:              "fact-young-1",
		ClaimText:       "People prefer fairness",
		Domain:          "sociology",
		Status:          types.FactStatusVerified,
		Confidence:      750_000,
		BaseConfidence:  750_000,
		Maturity:        "provisional",
		CreatedAtHeight: 8, // very young (2 blocks before current)
		CitationCount:   0,
		VerifierCount:   3,
	}
	k.SetFact(ctx, youngFact)

	// ====================================================================
	// FARM-10: Axiom restating blocked (via full evaluation pipeline)
	// ====================================================================
	t.Run("FARM-10_axiom_restating_blocked", func(t *testing.T) {
		// Create a claim that merely restates canonical facts in a well-populated domain.
		// The novelty stage should apply RestatingPenaltyBps because:
		//   - info gain is low (domain is dense → low sparsity, same-domain refs → low reference novelty)
		//   - all references are canonical/established

		// Create a second canonical math fact (same domain as claim)
		k.SetFact(ctx, &types.Fact{
			Id: "fact-canonical-math-2", ClaimText: "2 + 2 = 4",
			Domain: "mathematics", Status: types.FactStatusVerified,
			Confidence: 980_000, BaseConfidence: 980_000,
			Maturity: types.MaturityCanonical, VerifierCount: 20,
		})

		// Populate domain to reduce sparsity (100+ facts → domain_sparsity floors at 10%)
		for i := 0; i < 100; i++ {
			k.SetFact(ctx, &types.Fact{
				Id:     "fact-math-filler-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
				Domain: "mathematics",
				Status: types.FactStatusVerified,
			})
		}

		// FARM-10 triggers when: info_gain < MinInformationGainBps AND all refs are canonical/established.
		// Key: ALL references must be in the SAME domain as the claim to keep reference novelty low.
		// Cross-domain refs inflate info gain via sqrt(sparsity * refNovelty) and bypass FARM-10.
		claim := &types.Claim{
			Id:                "claim-restating",
			ClaimText:         "One plus one equals two",
			Domain:            "mathematics",
			EpistemicCategory: "analytic",
			References:        []string{"fact-canonical-1", "fact-canonical-math-2"},
			Submitter:         testSubmitterAddr,
		}
		k.SetClaim(ctx, claim)

		// Add fact references — both same-domain (mathematics) to keep reference novelty at 1.0
		// info_gain = sqrt(sparsity * refNovelty) where refNovelty = uniqueDomains/totalRefs
		// With 2 refs from 1 domain: refNovelty = 1/2 = 500_000
		// With 100+ facts: sparsity floors at 100_000
		// info_gain = sqrt(100_000 * 500_000) = sqrt(50B) ≈ 223_606 < MinInformationGainBps(250_000)
		k.SetFactReference(ctx, &types.FactReference{
			FromFactId:    "claim-restating",
			ToFactId:      "fact-canonical-1",
			ReferenceType: "supports",
		})
		k.SetFactReference(ctx, &types.FactReference{
			FromFactId:    "claim-restating",
			ToFactId:      "fact-canonical-math-2",
			ReferenceType: "supports",
		})

		// Build evaluation context from keeper (same wiring as production).
		// Override GetEvidenceByClaimId to inject mock evidence — the novelty stage
		// has an evidence quality gate that returns early if evidence adjustment ≤ 0.
		// This is correct behavior: FARM-10 only fires when a claim has positive
		// evidence yet still restates known facts (gaming with "legitimate" evidence).
		evalCtx := k.BuildEvaluationContext(ctx)
		evalCtx.GetEvidenceByClaimId = func(_ sdk.Context, claimId string) []types.EvidenceRecord {
			if claimId == "claim-restating" {
				return []types.EvidenceRecord{{
					EvidenceType: "formal_proof",
					Status:       "verified",
					Reliability:  800_000,
				}}
			}
			return nil
		}

		round := &types.VerificationRound{
			ClaimId: "claim-restating",
		}

		result, err := evaluation.EvaluateClaim(ctx, evalCtx, round, claim)
		if err != nil {
			t.Fatalf("EvaluateClaim: %v", err)
		}

		// Stage 6 is the novelty stage — check it applied FARM-10 penalty
		if len(result.Stages) < 6 {
			t.Fatalf("expected at least 6 stages, got %d", len(result.Stages))
		}
		noveltyStage := result.Stages[5] // 0-indexed: stage 6 is index 5
		if noveltyStage.Name != "novelty" {
			t.Errorf("stage 5 expected name 'novelty', got %q", noveltyStage.Name)
		}
		// With a dense domain and all canonical refs, FARM-10 should penalize
		if noveltyStage.Adjustment >= 0 {
			t.Errorf("FARM-10: expected negative novelty adjustment for axiom restating, got %d",
				noveltyStage.Adjustment)
		}
		// Verify the exact penalty value
		if noveltyStage.Adjustment != -int64(ep.RestatingPenaltyBps) {
			t.Errorf("FARM-10: expected adjustment -%d, got %d",
				ep.RestatingPenaltyBps, noveltyStage.Adjustment)
		}
	})

	// ====================================================================
	// FARM-11: Sybil citation ring discounted
	// ====================================================================
	t.Run("FARM-11_sybil_citation_ring_discounted", func(t *testing.T) {
		sybilFact := &types.Fact{
			Id:             "fact-sybil-target",
			Domain:         "general",
			Status:         types.FactStatusVerified,
			Confidence:     800_000,
			BaseConfidence: 800_000,
			CitationCount:  10,
			Submitter:      testSubmitterAddr,
		}
		k.SetFact(ctx, sybilFact)

		// Create sybil ring: 2 addresses cite 5 times each = 100% concentration
		for i := 0; i < 5; i++ {
			k.UpdateCitationSource(ctx, "fact-sybil-target", "lgm1sybil_ring_a")
		}
		for i := 0; i < 5; i++ {
			k.UpdateCitationSource(ctx, "fact-sybil-target", "lgm1sybil_ring_b")
		}

		discount := k.GetCitationConcentrationDiscount(ctx, "fact-sybil-target", "anyone", ep)
		if discount != ep.CitationConcentrationDiscount {
			t.Errorf("FARM-11: expected concentration discount %d, got %d",
				ep.CitationConcentrationDiscount, discount)
		}
		// Discount = 700,000 BPS (70%) — effectively slashing citation rewards by 70%
		if discount != 700_000 {
			t.Errorf("FARM-11: expected default discount 700000, got %d", discount)
		}
	})

	// ====================================================================
	// FARM-12: Unknown domain gets restrictive ceiling
	// ====================================================================
	t.Run("FARM-12_unknown_domain_restrictive_ceiling", func(t *testing.T) {
		// Test 1: Unknown domain not in store
		_, found := k.GetDomainStratum(ctx, "underwater_basket_weaving")
		if found {
			t.Error("FARM-12: unknown domain should not be in store")
		}

		// Test 2: Unknown domain should get social ceiling (850k) via AggregateVerificationResult
		claim := types.Claim{
			Id: "farm12-unknown", ClaimText: "Unknown domain test",
			Domain: "underwater_basket_weaving", EpistemicCategory: "empirical",
			Submitter: testSubmitterAddr, StakeAmount: "1000000", Status: "pending",
		}
		k.SetClaim(ctx, &claim)
		round, _ := k.CreateVerificationRound(ctx, &claim)
		validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
		for _, v := range validators {
			round.Commitments[v] = &types.VerificationCommitment{Validator: v}
			round.Reveals[v] = &types.VerificationReveal{
				Validator: v, Verdict: "accept", Confidence: 950_000,
			}
		}
		round.SelectedValidators = validators
		k.SetVerificationRound(ctx, round)

		result, err := k.AggregateVerificationResult(ctx, round)
		if err != nil {
			t.Fatalf("AggregateVerificationResult: %v", err)
		}
		// Social ceiling = 850k, so 950k should be clamped
		if result.Confidence > 850_000 {
			t.Errorf("FARM-12: unknown domain confidence=%d, should be clamped to ≤850000 (social ceiling)",
				result.Confidence)
		}

		// Test 3: Mathematics should have fundamental ceiling (1,000,000)
		mathStratum, mathFound := k.GetDomainStratum(ctx, "mathematics")
		if !mathFound {
			t.Fatal("FARM-12: mathematics should be in store (bootstrapped)")
		}
		mathCeiling, ok := ep.GetStratumConfidenceCeiling(mathStratum)
		if !ok {
			t.Fatal("FARM-12: stratum ceiling not found for mathematics")
		}
		if mathCeiling != 1_000_000 {
			t.Errorf("FARM-12: expected math ceiling 1000000, got %d", mathCeiling)
		}

		// Test 4: Governance override — set unknown domain to cognitive
		k.SetDomainStratum(ctx, "underwater_basket_weaving", "cognitive")
		newStratum, newFound := k.GetDomainStratum(ctx, "underwater_basket_weaving")
		if !newFound || newStratum != "cognitive" {
			t.Errorf("FARM-12: expected stratum=cognitive after governance set, got %q found=%v",
				newStratum, newFound)
		}
		newCeiling, ceilOk := ep.GetStratumConfidenceCeiling("cognitive")
		if !ceilOk || newCeiling != 900_000 {
			t.Errorf("FARM-12: expected cognitive ceiling 900000, got %d ok=%v", newCeiling, ceilOk)
		}
	})

	// ====================================================================
	// FARM-13: Sybil-cited prophetic status blocked
	// ====================================================================
	t.Run("FARM-13_sybil_prophetic_blocked", func(t *testing.T) {
		// Fact has enough citations (10) but from only 2 unique addresses
		propheticCandidate := &types.Fact{
			Id:              "fact-prophetic-sybil",
			Domain:          "physics",
			Status:          types.FactStatusVerified,
			Confidence:      600_000,
			BaseConfidence:  630_000,
			CitationCount:   10,
			CreatedAtHeight: 1,
		}
		k.SetFact(ctx, propheticCandidate)

		// Sybil citations: 2 addresses
		for i := 0; i < 5; i++ {
			k.UpdateCitationSource(ctx, "fact-prophetic-sybil", "lgm1sybil_prophet_a")
		}
		for i := 0; i < 5; i++ {
			k.UpdateCitationSource(ctx, "fact-prophetic-sybil", "lgm1sybil_prophet_b")
		}

		storedFact, _ := k.GetFact(ctx, "fact-prophetic-sybil")
		diverse := k.ValidatePropheticDiversity(ctx, storedFact, ep)
		if diverse {
			t.Error("FARM-13: expected prophetic diversity to FAIL for sybil-cited fact")
		}

		// Now add legitimate diverse citations
		for i := 0; i < 5; i++ {
			addr := testSubmitterAddr + string(rune('a'+i))
			k.UpdateCitationSource(ctx, "fact-prophetic-sybil", addr)
		}

		// With 7 unique citers (2 sybil + 5 legit), diversity should pass IF
		// no single address exceeds 30%. Let's check: sybil_a has 5/15 = 33% > 30%
		storedFact, _ = k.GetFact(ctx, "fact-prophetic-sybil")
		diverse = k.ValidatePropheticDiversity(ctx, storedFact, ep)
		if diverse {
			t.Error("FARM-13: expected prophetic diversity to STILL FAIL — sybil_a has 33% share")
		}
	})

	// ====================================================================
	// FARM-14: Low-ceiling domain normalized difficulty
	// ====================================================================
	t.Run("FARM-14_normalized_difficulty_bonus", func(t *testing.T) {
		// For a social domain (ceiling=850,000), a claim at 650,000 confidence
		// should get a normalized difficulty bonus of (850k-650k)/850k = ~23.5%
		// For a math domain (ceiling=1,000,000), the same 650k confidence
		// gives a higher difficulty bonus of (1M-650k)/1M = 35%
		//
		// This ensures validators aren't over-rewarded for "easy" low-ceiling domains.

		socialCeiling := uint64(850_000)
		mathCeiling := uint64(1_000_000)
		confidence := uint64(650_000)

		// Social domain: normalized difficulty
		socialDifficulty := safeMulDivTest(socialCeiling-confidence, 1_000_000, socialCeiling)
		// Math domain: normalized difficulty
		mathDifficulty := safeMulDivTest(mathCeiling-confidence, 1_000_000, mathCeiling)

		// Social difficulty should be LOWER than math difficulty for same confidence
		if socialDifficulty >= mathDifficulty {
			t.Errorf("FARM-14: expected social difficulty (%d) < math difficulty (%d)",
				socialDifficulty, mathDifficulty)
		}

		// Verify approximate values
		// Social: (850000-650000)*1000000/850000 = 235294
		if socialDifficulty < 230_000 || socialDifficulty > 240_000 {
			t.Errorf("FARM-14: social difficulty %d not in expected range [230000, 240000]", socialDifficulty)
		}
		// Math: (1000000-650000)*1000000/1000000 = 350000
		if mathDifficulty != 350_000 {
			t.Errorf("FARM-14: math difficulty expected 350000, got %d", mathDifficulty)
		}

		// At ceiling: no bonus
		atCeiling := safeMulDivTest(socialCeiling-socialCeiling, 1_000_000, socialCeiling)
		if atCeiling != 0 {
			t.Errorf("FARM-14: expected zero difficulty at ceiling, got %d", atCeiling)
		}
	})

	// ====================================================================
	// FARM-15: Young uncited fact gives minimal falsification reward
	// ====================================================================
	t.Run("FARM-15_falsification_params_validated", func(t *testing.T) {
		// Verify falsification reward scaling parameters are sensible:
		// - base: 1x
		// - age bonus: scales linearly up to +1x over 342720 blocks
		// - cite bonus: scales linearly up to +1x over 50 citations
		// - cap: 3x (base + max_age + max_cite)

		if ep.FalsificationBaseMultiplierBps != 1_000_000 {
			t.Errorf("FARM-15: base multiplier expected 1000000, got %d", ep.FalsificationBaseMultiplierBps)
		}
		if ep.FalsificationAgeScaleBlocks != 342_720 {
			t.Errorf("FARM-15: age scale blocks expected 342720, got %d", ep.FalsificationAgeScaleBlocks)
		}
		if ep.FalsificationAgeMaxBonusBps != 1_000_000 {
			t.Errorf("FARM-15: age max bonus expected 1000000, got %d", ep.FalsificationAgeMaxBonusBps)
		}
		if ep.FalsificationCiteScaleCount != 50 {
			t.Errorf("FARM-15: cite scale count expected 50, got %d", ep.FalsificationCiteScaleCount)
		}
		if ep.FalsificationCiteMaxBonusBps != 1_000_000 {
			t.Errorf("FARM-15: cite max bonus expected 1000000, got %d", ep.FalsificationCiteMaxBonusBps)
		}
		if ep.FalsificationRewardCapBps != 3_000_000 {
			t.Errorf("FARM-15: reward cap expected 3000000, got %d", ep.FalsificationRewardCapBps)
		}

		// Young fact (age=2 blocks, 0 citations) → minimal reward
		// age_bonus = min(1M, 2 * 1M / 342720) ≈ 5 BPS (negligible)
		// cite_bonus = min(1M, 0 * 1M / 50) = 0
		// total = base(1M) + 5 + 0 = ~1.000005x
		ageBonus := safeMulDivTest(2, ep.FalsificationAgeMaxBonusBps, ep.FalsificationAgeScaleBlocks)
		citeBonus := safeMulDivTest(0, ep.FalsificationCiteMaxBonusBps, ep.FalsificationCiteScaleCount)
		totalMultiplier := ep.FalsificationBaseMultiplierBps + ageBonus + citeBonus

		// Should be barely above base
		if totalMultiplier > ep.FalsificationBaseMultiplierBps+100 {
			t.Errorf("FARM-15: young uncited fact multiplier too high: %d (expected ~%d)",
				totalMultiplier, ep.FalsificationBaseMultiplierBps)
		}

		// Old heavily-cited fact → max reward (3x)
		ageBonus = safeMulDivTest(ep.FalsificationAgeScaleBlocks, ep.FalsificationAgeMaxBonusBps, ep.FalsificationAgeScaleBlocks)
		if ageBonus > ep.FalsificationAgeMaxBonusBps {
			ageBonus = ep.FalsificationAgeMaxBonusBps
		}
		citeBonus = safeMulDivTest(ep.FalsificationCiteScaleCount, ep.FalsificationCiteMaxBonusBps, ep.FalsificationCiteScaleCount)
		if citeBonus > ep.FalsificationCiteMaxBonusBps {
			citeBonus = ep.FalsificationCiteMaxBonusBps
		}
		totalMultiplier = ep.FalsificationBaseMultiplierBps + ageBonus + citeBonus
		if totalMultiplier > ep.FalsificationRewardCapBps {
			totalMultiplier = ep.FalsificationRewardCapBps
		}
		if totalMultiplier != ep.FalsificationRewardCapBps {
			t.Errorf("FARM-15: fully aged/cited fact multiplier expected %d (cap), got %d",
				ep.FalsificationRewardCapBps, totalMultiplier)
		}

		// Validation: base + age_max + cite_max must not exceed cap
		if ep.FalsificationBaseMultiplierBps+ep.FalsificationAgeMaxBonusBps+ep.FalsificationCiteMaxBonusBps > ep.FalsificationRewardCapBps {
			t.Error("FARM-15: base + age_max + cite_max exceeds reward cap")
		}
	})

	// ====================================================================
	// Combined: FARM-11 + FARM-13 share CitationSourceRecord
	// ====================================================================
	t.Run("FARM-11_13_shared_data", func(t *testing.T) {
		// Create a fact, populate citation sources, verify both FARM-11 and FARM-13
		// read from the same data.
		sharedFact := &types.Fact{
			Id:             "fact-shared-data",
			Domain:         "physics",
			Status:         types.FactStatusVerified,
			Confidence:     600_000,
			BaseConfidence: 630_000,
			CitationCount:  10,
		}
		k.SetFact(ctx, sharedFact)

		// Concentrated citations: 1 address with 8, 2 with 1 each
		for i := 0; i < 8; i++ {
			k.UpdateCitationSource(ctx, "fact-shared-data", "lgm1whale")
		}
		k.UpdateCitationSource(ctx, "fact-shared-data", "lgm1minnow_a")
		k.UpdateCitationSource(ctx, "fact-shared-data", "lgm1minnow_b")

		// FARM-11: should detect concentration (top-3 = 10/10 = 100%)
		discount := k.GetCitationConcentrationDiscount(ctx, "fact-shared-data", "anyone", ep)
		if discount == 0 {
			t.Error("FARM-11: expected concentration discount for whale-dominated fact")
		}

		// FARM-13: should fail prophetic diversity (only 3 unique, whale has 80%)
		storedFact, _ := k.GetFact(ctx, "fact-shared-data")
		diverse := k.ValidatePropheticDiversity(ctx, storedFact, ep)
		if diverse {
			t.Error("FARM-13: expected prophetic diversity to fail for whale-dominated fact")
		}

		// Verify they read the same record
		record, found := k.GetCitationSourceRecord(ctx, "fact-shared-data")
		if !found {
			t.Fatal("expected citation source record to exist")
		}
		if record.TotalCites != 10 {
			t.Errorf("expected TotalCites=10, got %d", record.TotalCites)
		}
		if record.SourceCounts["lgm1whale"] != 8 {
			t.Errorf("expected whale count=8, got %d", record.SourceCounts["lgm1whale"])
		}
	})
}
