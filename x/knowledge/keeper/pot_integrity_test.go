package keeper_test

// ==================================================================================
// R2-Session-1: Proof of Truth — End-to-End Integrity Tests
// ==================================================================================
//
// Covers: commit-reveal-finalize lifecycle, proposer adversarial scenarios,
// cross-round replay protection, duplicate/equivocation handling.
//
// Trust boundaries verified:
//   B1: ExtendVote → VoteExtension (validator-local, VRF-gated)
//   B2: VoteExtension → Injection tx (proposer-controlled — censorship, duplication)
//   B3: Injection tx → Keeper state (PreBlocker — commitment hash verified)
//   B4: Keeper state → Aggregation (BeginBlocker — deterministic, sorted)
//   B5: Aggregation → Fact/Reward (CompleteRound — idempotent guard)
// ==================================================================================

import (
	"fmt"
	"testing"

	"github.com/legible-money/legible/x/knowledge/types"
)

// ==================================================================================
// TEST 1: Happy Path — honest commit → reveal → finalize → rewards
// ==================================================================================
// Verifies the full lifecycle for 3 validators all voting "accept":
//   1. Commitments stored with correct hash
//   2. Reveals verified against commitment hash
//   3. Aggregation produces "accepted" verdict
//   4. Fact created, claim updated
func TestPoT_HappyPath_CommitRevealFinalize(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id:                "pot-happy-claim",
		ClaimText:         "Water is composed of hydrogen and oxygen atoms (H2O)",
		Domain:            "physics",
		EpistemicCategory: "empirical",
		Submitter:         testSubmitterAddr,
		StakeAmount:       "1000000",
		Status:            "pending",
	}
	k.SetClaim(ctx, claim)

	round, err := k.CreateVerificationRound(ctx, claim)
	if err != nil {
		t.Fatalf("CreateVerificationRound failed: %v", err)
	}

	// === COMMIT PHASE ===
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	verdicts := map[string]string{"lgmvaloper1abc": "accept", "lgmvaloper1def": "accept", "lgmvaloper1ghi": "accept"}
	confs := map[string]uint64{"lgmvaloper1abc": 850000, "lgmvaloper1def": 900000, "lgmvaloper1ghi": 800000}
	salts := map[string]string{"lgmvaloper1abc": "salt-abc", "lgmvaloper1def": "salt-def", "lgmvaloper1ghi": "salt-ghi"}

	// Compute canonical commitment hashes (same function used in production)
	for _, v := range validators {
		hash := types.ComputeCommitmentHash(round.Id, verdicts[v], confs[v], salts[v])
		if len(hash) != 64 {
			t.Fatalf("commitment hash for %s has wrong length: %d (expected 64)", v, len(hash))
		}
		if err := k.StoreCommitmentInRound(ctx, round.Id, types.VerificationCommitment{
			Validator:         v,
			CommitmentHash:    hash,
			SubmittedAtHeight: 11,
		}); err != nil {
			t.Fatalf("StoreCommitmentInRound failed for %s: %v", v, err)
		}
	}

	// Verify commitments stored
	commitments, err := k.GetCommitmentsForRound(ctx, round.Id)
	if err != nil {
		t.Fatalf("GetCommitmentsForRound failed: %v", err)
	}
	if len(commitments) != 3 {
		t.Fatalf("expected 3 commitments, got %d", len(commitments))
	}

	// === REVEAL PHASE ===
	// Advance round to reveal phase
	updatedRound, _ := k.GetVerificationRound(ctx, round.Id)
	updatedRound.Phase = "reveal"
	k.SetVerificationRound(ctx, updatedRound)

	// Reveals must use the same verdict/confidence/salt as the commitment
	for _, v := range validators {
		if err := k.StoreRevealInRound(ctx, round.Id, types.VerificationReveal{
			Validator:         v,
			Verdict:           verdicts[v],
			Confidence:        confs[v],
			Salt:              salts[v],
			SubmittedAtHeight: 15,
		}); err != nil {
			t.Fatalf("StoreRevealInRound failed for %s: %v", v, err)
		}
	}

	reveals, err := k.GetRevealsForRound(ctx, round.Id)
	if err != nil {
		t.Fatalf("GetRevealsForRound failed: %v", err)
	}
	if len(reveals) != 3 {
		t.Fatalf("expected 3 reveals, got %d", len(reveals))
	}

	// === AGGREGATION ===
	finalRound, _ := k.GetVerificationRound(ctx, round.Id)
	result, err := k.AggregateVerificationResult(ctx, finalRound)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	if result.Verdict != "accepted" {
		t.Errorf("expected verdict=accepted, got %s", result.Verdict)
	}
	if result.ParticipantCount != 3 {
		t.Errorf("expected 3 participants, got %d", result.ParticipantCount)
	}
	if result.Confidence < 770000 {
		t.Errorf("expected confidence >= 770000, got %d", result.Confidence)
	}
	t.Logf("HAPPY PATH: verdict=%s confidence=%d rewards=%v",
		result.Verdict, result.Confidence, result.Rewards)

	// === FINALIZE ===
	if err := k.CompleteRound(ctx, finalRound, result); err != nil {
		t.Fatalf("CompleteRound failed: %v", err)
	}

	// Claim updated
	updatedClaim, found := k.GetClaim(ctx, claim.Id)
	if !found {
		t.Fatal("claim should exist after round completion")
	}
	if updatedClaim.Status != "accepted" {
		t.Errorf("expected claim status=accepted, got %s", updatedClaim.Status)
	}

	// Fact created
	factCount := k.GetFactCount(ctx)
	if factCount == 0 {
		t.Error("expected fact to be created after acceptance")
	}
}

// ==================================================================================
// TEST 2: Proposer Omits One Vote (Censorship)
// ==================================================================================
// Simulates a malicious proposer who silently drops validator C's vote extension.
// With 2/3 validators participating, check quorum behavior.
//
// Trust boundary: B2 (proposer-controlled injection)
func TestPoT_ProposerOmitsOneVote_ReducedParticipation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id:        "pot-censorship-claim",
		ClaimText: "The speed of light is approximately 299792458 m/s in vacuum",
		Domain:    "physics",
		Status:    "pending",
	}
	k.SetClaim(ctx, claim)

	round, _ := k.CreateVerificationRound(ctx, claim)

	// 3 validators selected, but proposer only injects 2 commitments (censors lgmvaloper1ghi)
	selected := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	included := []string{"lgmvaloper1abc", "lgmvaloper1def"}

	// Store commitments for only the 2 included validators
	for _, v := range included {
		hash := types.ComputeCommitmentHash(round.Id, "accept", 800000, "salt-"+v)
		if err := k.StoreCommitmentInRound(ctx, round.Id, types.VerificationCommitment{
			Validator:         v,
			CommitmentHash:    hash,
			SubmittedAtHeight: 11,
		}); err != nil {
			t.Fatalf("StoreCommitmentInRound failed for %s: %v", v, err)
		}
	}

	// Set selected validators (what the round knows about)
	updatedRound, _ := k.GetVerificationRound(ctx, round.Id)
	updatedRound.SelectedValidators = selected
	updatedRound.Phase = "reveal"
	k.SetVerificationRound(ctx, updatedRound)

	// Only 2 reveals (censored validator never reveals)
	for _, v := range included {
		if err := k.StoreRevealInRound(ctx, round.Id, types.VerificationReveal{
			Validator:  v,
			Verdict:    "accept",
			Confidence: 800000,
			Salt:       "salt-" + v,
		}); err != nil {
			t.Fatalf("StoreRevealInRound failed for %s: %v", v, err)
		}
	}

	finalRound, _ := k.GetVerificationRound(ctx, round.Id)
	result, err := k.AggregateVerificationResult(ctx, finalRound)
	if err != nil {
		t.Fatalf("AggregateVerificationResult failed: %v", err)
	}

	// 2/3 selected validators = 66.6% participation. Quorum threshold defaults to 66%.
	// With 66.6% participation the result should be processed (not insufficient).
	t.Logf("CENSORSHIP: participation=%d/%d verdict=%s quorum_reached=%v",
		len(finalRound.Reveals), len(finalRound.SelectedValidators),
		result.Verdict, result.QuorumReached)

	// The censored validator (lgmvaloper1ghi) is assigned but didn't reveal.
	// calculateRewardsAndSlashes should slash them for missed reveal.
	slashedForMissed := false
	for v, slash := range result.Slashes {
		if v == "lgmvaloper1ghi" {
			slashedForMissed = true
			t.Logf("CENSORSHIP: censored validator slashed %d ulgm for missed reveal", slash)
		}
	}
	if slashedForMissed {
		t.Logf("CENSORSHIP: slashing correctly applied to censored validator")
	} else {
		// lgmvaloper1ghi committed (no commitment stored, so not in commitments map)
		// Without a commitment, the slash-for-missed-reveal won't fire.
		t.Logf("CENSORSHIP: no slash (validator had no commitment on-chain — proposer omitted it)")
		t.Logf("CENSORSHIP: key insight: proposer can censor votes without triggering on-chain detection")
	}
}

// ==================================================================================
// TEST 3: Tampered Reveal Rejected by Commitment Hash Check
// ==================================================================================
// Simulates a proposer who tries to change a validator's reveal verdict after
// the commitment phase. The commitment hash verification (S6 fix) must catch this.
//
// Trust boundary: B3 (injection → keeper state)
func TestPoT_TamperedRevealRejectedByHashCheck(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id:        "pot-tamper-claim",
		ClaimText: "Quantum entanglement allows faster-than-light information transfer",
		Domain:    "physics",
		Status:    "pending",
	}
	k.SetClaim(ctx, claim)

	round, _ := k.CreateVerificationRound(ctx, claim)

	// Validator committed to "reject" with confidence 850000
	originalVerdict := "reject"
	originalConf := uint64(850000)
	salt := "honest-salt-abc"
	hash := types.ComputeCommitmentHash(round.Id, originalVerdict, originalConf, salt)

	if err := k.StoreCommitmentInRound(ctx, round.Id, types.VerificationCommitment{
		Validator:         "lgmvaloper1abc",
		CommitmentHash:    hash,
		SubmittedAtHeight: 11,
	}); err != nil {
		t.Fatalf("StoreCommitmentInRound failed: %v", err)
	}

	// Advance to reveal phase
	updatedRound, _ := k.GetVerificationRound(ctx, round.Id)
	updatedRound.Phase = "reveal"
	k.SetVerificationRound(ctx, updatedRound)

	// === ATTACK: proposer changes verdict from "reject" to "accept" ===
	tamperedVerdict := "accept"
	err := k.StoreRevealInRound(ctx, round.Id, types.VerificationReveal{
		Validator:  "lgmvaloper1abc",
		Verdict:    tamperedVerdict, // Changed by proposer
		Confidence: originalConf,
		Salt:       salt,
	})
	if err == nil {
		t.Error("SECURITY FAILURE: tampered reveal (changed verdict) should be rejected by commitment hash check")
	} else {
		t.Logf("TAMPER REJECTED: %v", err)
		t.Logf("ANTI-TAMPER: commitment hash verification (S6 fix) correctly prevents verdict manipulation")
	}

	// Verify no reveal was stored
	finalRound, _ := k.GetVerificationRound(ctx, round.Id)
	if _, hasReveal := finalRound.Reveals["lgmvaloper1abc"]; hasReveal {
		t.Error("tampered reveal should not be stored")
	}

	// === VERIFY: correct reveal (matching commitment) is accepted ===
	err = k.StoreRevealInRound(ctx, round.Id, types.VerificationReveal{
		Validator:  "lgmvaloper1abc",
		Verdict:    originalVerdict, // Original, unmodified verdict
		Confidence: originalConf,
		Salt:       salt,
	})
	if err != nil {
		t.Errorf("honest reveal should be accepted: %v", err)
	}
	t.Logf("HONEST REVEAL: correctly accepted with matching commitment hash")
}

// ==================================================================================
// TEST 4: Cross-Round Replay Protection
// ==================================================================================
// A commitment from round A cannot be replayed in round B because the roundID
// is included in the canonical commitment hash (domain separation).
//
// Trust boundary: B3 (injection → keeper state) — roundID-keyed hash
func TestPoT_CrossRoundReplay_CommitmentHashMismatch(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim1 := types.Claim{
		Id:        "pot-replay-claim-1",
		ClaimText: "Absolute zero is approximately -273.15 degrees Celsius",
		Domain:    "physics",
		Status:    "pending",
	}
	claim2 := types.Claim{
		Id:        "pot-replay-claim-2",
		ClaimText: "The Bohr radius is approximately 0.529 angstroms",
		Domain:    "physics",
		Status:    "pending",
	}
	k.SetClaim(ctx, claim1)
	k.SetClaim(ctx, claim2)

	round1, _ := k.CreateVerificationRound(ctx, claim1)

	// Advance height so round2 gets different start height
	ctx2 := ctx.WithBlockHeight(20)
	round2, _ := k.CreateVerificationRound(ctx2, claim2)

	// Commitment generated for round1 with specific verdict/salt
	verdict := "accept"
	confidence := uint64(800000)
	salt := "replay-salt"

	hashForRound1 := types.ComputeCommitmentHash(round1.Id, verdict, confidence, salt)
	hashForRound2 := types.ComputeCommitmentHash(round2.Id, verdict, confidence, salt)

	// Hashes must differ (cross-round domain separation)
	if hashForRound1 == hashForRound2 {
		t.Errorf("SECURITY FAILURE: same hash for different rounds — roundID not providing domain separation")
	} else {
		t.Logf("DOMAIN SEPARATION: round1 hash ≠ round2 hash (first 8 chars: %s vs %s)",
			hashForRound1[:8], hashForRound2[:8])
	}

	// Store commitment for round1
	if err := k.StoreCommitmentInRound(ctx, round1.Id, types.VerificationCommitment{
		Validator:         "lgmvaloper1abc",
		CommitmentHash:    hashForRound1,
		SubmittedAtHeight: 11,
	}); err != nil {
		t.Fatalf("StoreCommitmentInRound for round1 failed: %v", err)
	}

	// Advance to reveal phase for round1
	updatedRound1, _ := k.GetVerificationRound(ctx, round1.Id)
	updatedRound1.Phase = "reveal"
	k.SetVerificationRound(ctx, updatedRound1)

	// === ATTACK: replay round1's reveal data in round2 (store commitment for round2 with round1 hash) ===
	// Store commitment for round2 using round1's hash
	if err := k.StoreCommitmentInRound(ctx, round2.Id, types.VerificationCommitment{
		Validator:         "lgmvaloper1abc",
		CommitmentHash:    hashForRound1, // Using round1's hash in round2
		SubmittedAtHeight: 11,
	}); err != nil {
		t.Fatalf("StoreCommitmentInRound for round2 failed: %v", err)
	}

	// Advance round2 to reveal
	updatedRound2, _ := k.GetVerificationRound(ctx, round2.Id)
	updatedRound2.Phase = "reveal"
	k.SetVerificationRound(ctx, updatedRound2)

	// Attacker tries to reveal in round2 with the same data as round1
	// This should fail: StoreRevealInRound verifies hash(round2.Id, verdict, confidence, salt)
	// but the stored commitment hash is hash(round1.Id, verdict, confidence, salt)
	err := k.StoreRevealInRound(ctx, round2.Id, types.VerificationReveal{
		Validator:  "lgmvaloper1abc",
		Verdict:    verdict,
		Confidence: confidence,
		Salt:       salt,
	})
	if err == nil {
		t.Error("SECURITY FAILURE: cross-round replay should be rejected — roundID not in reveal verification")
	} else {
		t.Logf("REPLAY REJECTED: %v", err)
		t.Logf("ANTI-REPLAY: roundID included in commitment hash prevents cross-round replay attacks")
	}
}

// ==================================================================================
// TEST 5: Duplicate Commitment — Idempotent (same hash) vs Equivocation (different hash)
// ==================================================================================
// Tests that:
//   - Same commitment hash twice = silently dropped (idempotent)
//   - Different commitment hash = equivocation → slash
//
// Trust boundary: B2 (proposer can inject duplicates) + B3 (keeper deduplication)
func TestPoT_DuplicateCommitment_IdempotentVsEquivocation(t *testing.T) {
	k, ctx, mockSK := setupKeeper(t)

	claim := types.Claim{
		Id:        "pot-dup-claim",
		ClaimText: "Pi is approximately 3.14159265358979",
		Domain:    "mathematics",
		Status:    "pending",
	}
	k.SetClaim(ctx, claim)

	// === SCENARIO A: Same hash twice (idempotent) ===
	round1, _ := k.CreateVerificationRound(ctx, claim)

	hash := types.ComputeCommitmentHash(round1.Id, "accept", 800000, "salt-a")

	// First commitment succeeds
	if err := k.StoreCommitmentInRound(ctx, round1.Id, types.VerificationCommitment{
		Validator: "lgmvaloper1abc", CommitmentHash: hash, SubmittedAtHeight: 11,
	}); err != nil {
		t.Fatalf("first commitment should succeed: %v", err)
	}

	// Same hash again — should be idempotent (not equivocation)
	err := k.StoreCommitmentInRound(ctx, round1.Id, types.VerificationCommitment{
		Validator: "lgmvaloper1abc", CommitmentHash: hash, SubmittedAtHeight: 12,
	})
	if err == nil {
		t.Error("duplicate commitment should return error (ErrDuplicateCommitment)")
	} else {
		t.Logf("DUPLICATE (same hash): correctly rejected with: %v", err)
		t.Logf("IDEMPOTENT: no slash — same hash is not equivocation")
	}

	// Verify no slash was recorded
	if len(mockSK.slashes) > 0 {
		t.Errorf("no slash expected for idempotent duplicate, got: %v", mockSK.slashes)
	}

	// === SCENARIO B: Different hash = equivocation ===
	claim2 := types.Claim{Id: "pot-equivoc-claim", ClaimText: "Equivocation test", Domain: "general", Status: "pending"}
	k.SetClaim(ctx, claim2)
	round2, _ := k.CreateVerificationRound(ctx, claim2)

	hash1 := types.ComputeCommitmentHash(round2.Id, "accept", 800000, "salt-first")
	hash2 := types.ComputeCommitmentHash(round2.Id, "reject", 800000, "salt-second") // Different

	// First commitment
	if err := k.StoreCommitmentInRound(ctx, round2.Id, types.VerificationCommitment{
		Validator: "lgmvaloper1abc", CommitmentHash: hash1, SubmittedAtHeight: 11,
	}); err != nil {
		t.Fatalf("first commitment for equivocation test should succeed: %v", err)
	}

	slashCountBefore := len(mockSK.slashes)

	// Conflicting commitment = equivocation
	err = k.StoreCommitmentInRound(ctx, round2.Id, types.VerificationCommitment{
		Validator: "lgmvaloper1abc", CommitmentHash: hash2, SubmittedAtHeight: 12,
	})
	if err == nil {
		t.Error("conflicting commitment should be rejected as equivocation")
	} else {
		t.Logf("EQUIVOCATION: correctly rejected with: %v", err)
	}

	// Verify slash was recorded
	if len(mockSK.slashes) <= slashCountBefore {
		t.Error("equivocation should trigger a validator slash")
	} else {
		for _, s := range mockSK.slashes[slashCountBefore:] {
			t.Logf("EQUIVOCATION SLASH: validator=%s amount=%s reason=%s", s.Addr, s.Amount, s.Reason)
		}
	}
}

// ==================================================================================
// TEST 6: Proposer Cannot Fabricate Reveals — Hash Must Match Commitment
// ==================================================================================
// Tests all reveal tampering vectors: changed verdict, changed confidence, wrong salt.
// Each should be rejected independently by the commitment hash check.
//
// Trust boundary: B3 (ProtoVoteExtInjection → StoreRevealInRound)
func TestPoT_AllRevealTamperingVectorsRejected(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id:        "pot-reveal-tamper",
		ClaimText: "The electron has a negative elementary charge of approximately -1.6e-19 coulombs",
		Domain:    "physics",
		Status:    "pending",
	}
	k.SetClaim(ctx, claim)

	round, _ := k.CreateVerificationRound(ctx, claim)

	// Honest commitment: accept, 800000, "honest-salt"
	trueVerdict := "accept"
	trueConf := uint64(800000)
	trueSalt := "honest-salt"
	hash := types.ComputeCommitmentHash(round.Id, trueVerdict, trueConf, trueSalt)

	if err := k.StoreCommitmentInRound(ctx, round.Id, types.VerificationCommitment{
		Validator: "lgmvaloper1abc", CommitmentHash: hash, SubmittedAtHeight: 11,
	}); err != nil {
		t.Fatalf("StoreCommitmentInRound failed: %v", err)
	}

	updatedRound, _ := k.GetVerificationRound(ctx, round.Id)
	updatedRound.Phase = "reveal"
	k.SetVerificationRound(ctx, updatedRound)

	tamperings := []struct {
		name       string
		verdict    string
		confidence uint64
		salt       string
	}{
		{"changed verdict", "reject", trueConf, trueSalt},
		{"changed confidence", trueVerdict, 999000, trueSalt},
		{"wrong salt", trueVerdict, trueConf, "wrong-salt"},
		{"all changed", "reject", 999000, "wrong-salt"},
	}

	for _, tc := range tamperings {
		// Use a fresh round for each tamper test to avoid phase errors
		claimN := types.Claim{
			Id:        fmt.Sprintf("pot-tamper-%s", tc.name),
			ClaimText: "Tamper test " + tc.name,
			Domain:    "general",
			Status:    "pending",
		}
		k.SetClaim(ctx, claimN)
		rN, _ := k.CreateVerificationRound(ctx, claimN)

		hashN := types.ComputeCommitmentHash(rN.Id, trueVerdict, trueConf, trueSalt)
		if err := k.StoreCommitmentInRound(ctx, rN.Id, types.VerificationCommitment{
			Validator: "lgmvaloper1abc", CommitmentHash: hashN, SubmittedAtHeight: 11,
		}); err != nil {
			t.Fatalf("[%s] store commitment failed: %v", tc.name, err)
		}
		// Fetch the updated round (StoreCommitmentInRound mutates state) before
		// advancing phase, to avoid overwriting the stored commitment.
		rNUpdated, _ := k.GetVerificationRound(ctx, rN.Id)
		rNUpdated.Phase = "reveal"
		k.SetVerificationRound(ctx, rNUpdated)

		err := k.StoreRevealInRound(ctx, rN.Id, types.VerificationReveal{
			Validator:  "lgmvaloper1abc",
			Verdict:    tc.verdict,
			Confidence: tc.confidence,
			Salt:       tc.salt,
		})
		if err == nil {
			t.Errorf("[%s] SECURITY FAILURE: tampered reveal should be rejected", tc.name)
		} else {
			t.Logf("[%s] TAMPER REJECTED: %v", tc.name, err)
		}
	}
}

// ==================================================================================
// TEST 7: Round Idempotency — CompleteRound called twice does nothing
// ==================================================================================
// The idempotency guard in CompleteRound (PP-CONSENSUS-2) ensures that state
// replay or migration cannot double-create facts or double-return stakes.
func TestPoT_RoundCompletionIdempotency(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id:        "pot-idempotent-claim",
		ClaimText: "Sodium chloride (NaCl) is common table salt",
		Domain:    "physics",
		Status:    "pending",
	}
	k.SetClaim(ctx, claim)

	round, _ := k.CreateVerificationRound(ctx, claim)

	// Set up reveals directly (bypassing commit for speed)
	validators := []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	for _, v := range validators {
		round.Commitments[v] = &types.VerificationCommitment{Validator: v, CommitmentHash: "hash"}
		round.Reveals[v] = &types.VerificationReveal{Validator: v, Verdict: "accept", Confidence: 900000}
	}
	round.SelectedValidators = validators
	k.SetVerificationRound(ctx, round)

	result, _ := k.AggregateVerificationResult(ctx, round)

	// First completion
	if err := k.CompleteRound(ctx, round, result); err != nil {
		t.Fatalf("first CompleteRound failed: %v", err)
	}

	factCountAfterFirst := k.GetFactCount(ctx)

	// Second completion (simulates replay) — should be no-op
	completedRound, _ := k.GetVerificationRound(ctx, round.Id)
	if err := k.CompleteRound(ctx, completedRound, result); err != nil {
		t.Fatalf("second CompleteRound should not error: %v", err)
	}

	factCountAfterSecond := k.GetFactCount(ctx)

	if factCountAfterFirst != factCountAfterSecond {
		t.Errorf("idempotency failure: fact count changed on second completion (%d → %d)",
			factCountAfterFirst, factCountAfterSecond)
	} else {
		t.Logf("IDEMPOTENCY: fact count unchanged on second completion (%d facts)", factCountAfterFirst)
	}
}

// ==================================================================================
// TEST 8: Aggregation Determinism — Reordering reveals cannot change outcome
// ==================================================================================
// The sorted iteration in AggregateVerificationResult guarantees that
// map iteration order does not affect the final verdict or confidence.
func TestPoT_AggregationDeterminism_ReorderingCannotChangeOutcome(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	claim := types.Claim{
		Id:        "pot-determinism-claim",
		ClaimText: "The gravitational constant G is approximately 6.674e-11 N m^2 kg^-2",
		Domain:    "physics",
		Status:    "pending",
	}
	k.SetClaim(ctx, claim)

	// === ARRANGEMENT 1: abc, def, ghi ===
	round1, _ := k.CreateVerificationRound(ctx, claim)
	round1.SelectedValidators = []string{"lgmvaloper1abc", "lgmvaloper1def", "lgmvaloper1ghi"}
	round1.Commitments["lgmvaloper1abc"] = &types.VerificationCommitment{Validator: "lgmvaloper1abc"}
	round1.Commitments["lgmvaloper1def"] = &types.VerificationCommitment{Validator: "lgmvaloper1def"}
	round1.Commitments["lgmvaloper1ghi"] = &types.VerificationCommitment{Validator: "lgmvaloper1ghi"}
	round1.Reveals["lgmvaloper1abc"] = &types.VerificationReveal{Validator: "lgmvaloper1abc", Verdict: "accept", Confidence: 800000}
	round1.Reveals["lgmvaloper1def"] = &types.VerificationReveal{Validator: "lgmvaloper1def", Verdict: "accept", Confidence: 900000}
	round1.Reveals["lgmvaloper1ghi"] = &types.VerificationReveal{Validator: "lgmvaloper1ghi", Verdict: "reject", Confidence: 700000}
	k.SetVerificationRound(ctx, round1)

	// === ARRANGEMENT 2: Same data, "inserted" in reverse order ===
	claim2 := types.Claim{Id: "pot-determinism-2", ClaimText: claim.ClaimText, Domain: claim.Domain, Status: "pending"}
	k.SetClaim(ctx, claim2)
	round2, _ := k.CreateVerificationRound(ctx, claim2)
	round2.SelectedValidators = []string{"lgmvaloper1ghi", "lgmvaloper1def", "lgmvaloper1abc"}
	round2.Commitments["lgmvaloper1ghi"] = &types.VerificationCommitment{Validator: "lgmvaloper1ghi"}
	round2.Commitments["lgmvaloper1def"] = &types.VerificationCommitment{Validator: "lgmvaloper1def"}
	round2.Commitments["lgmvaloper1abc"] = &types.VerificationCommitment{Validator: "lgmvaloper1abc"}
	round2.Reveals["lgmvaloper1ghi"] = &types.VerificationReveal{Validator: "lgmvaloper1ghi", Verdict: "reject", Confidence: 700000}
	round2.Reveals["lgmvaloper1def"] = &types.VerificationReveal{Validator: "lgmvaloper1def", Verdict: "accept", Confidence: 900000}
	round2.Reveals["lgmvaloper1abc"] = &types.VerificationReveal{Validator: "lgmvaloper1abc", Verdict: "accept", Confidence: 800000}
	k.SetVerificationRound(ctx, round2)

	result1, _ := k.AggregateVerificationResult(ctx, round1)
	result2, _ := k.AggregateVerificationResult(ctx, round2)

	if result1.Verdict != result2.Verdict {
		t.Errorf("DETERMINISM FAILURE: different verdicts for same data in different order (%s vs %s)",
			result1.Verdict, result2.Verdict)
	}
	if result1.Confidence != result2.Confidence {
		t.Errorf("DETERMINISM FAILURE: different confidence for same data in different order (%d vs %d)",
			result1.Confidence, result2.Confidence)
	}
	t.Logf("DETERMINISM: verdict=%s confidence=%d (identical regardless of insertion order)",
		result1.Verdict, result1.Confidence)
}
