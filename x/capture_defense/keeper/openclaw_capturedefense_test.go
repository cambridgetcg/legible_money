package keeper_test

// ==================================================================================
// OpenClaw Capture Defense Adversarial Probe Suite (OC-CD-1 through OC-CD-4)
//
// These probes systematically test the capture_defense module's attack surface:
// - Expired qualification bypass (time-based access control)
// - Below-threshold accuracy sneaking through track_record pathway
// - Duplicate qualification creation (idempotency / overwrite)
// - Self-endorsement via cross_reference pathway
// ==================================================================================

import (
	"testing"

	"github.com/legible-money/legible/x/capture_defense/types"
)

// ==================================================================================
// OC-CD-1: Expired Qualification Bypass
//
// ATTACK: Validator creates a qualification with ExpiresAtBlock=100.
// At block 101, the validator attempts to verify claims in that domain.
// IsQualifiedForDomain should return false, preventing the expired
// validator from participating in verification rounds.
//
// DEFENSE: IsQualifiedForDomain checks ExpiresAtBlock > 0 && currentBlock > ExpiresAtBlock.
// ==================================================================================

func TestOC_CD1_ExpiredQualificationCheck(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	validatorAddr := testAddr("expired_v")
	domain := "quantum_mechanics"

	// Create qualification that expires at block 100 (current block is 100)
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr:       validatorAddr,
		Domain:              domain,
		Pathway:             string(types.PathwayTrackRecord),
		Status:              string(types.StatusActive),
		EarnedAtBlock:       50,
		ExpiresAtBlock:      100,
		InheritanceDiscount: 1_000_000,
		Metrics: &types.QualificationMetrics{
			TotalVerifications:   25,
			CorrectVerifications: 20,
			Accuracy:             800_000,
			EffectiveWeight:      1_000_000,
		},
	})

	// At block 100 (boundary), should still be qualified (currentBlock == ExpiresAtBlock, not >)
	if !k.IsQualifiedForDomain(ctx, validatorAddr, domain, 100) {
		t.Error("ATTACK SURFACE: qualification should be valid AT the expiration block (boundary)")
	}

	// At block 101 (one past expiry), should NOT be qualified
	if k.IsQualifiedForDomain(ctx, validatorAddr, domain, 101) {
		t.Error("OC-CD-1 DEFENSE FAILURE: expired qualification (ExpiresAtBlock=100) " +
			"was accepted at block 101. An attacker could continue verifying claims " +
			"in a domain after their qualification has lapsed, corrupting verification " +
			"integrity and potentially earning undeserved rewards.")
	}

	// At block 200 (well past expiry), should NOT be qualified
	if k.IsQualifiedForDomain(ctx, validatorAddr, domain, 200) {
		t.Error("OC-CD-1 DEFENSE FAILURE: expired qualification accepted at block 200 " +
			"(100 blocks past expiry). Expiration enforcement is broken.")
	}

	// Verify GetQualificationWeight also reflects expiry correctly:
	// The weight method only checks status (active/probationary), not expiry.
	// This is acceptable because IsQualifiedForDomain is the gatekeeper.
	// But verify the qualification data itself still exists (not deleted).
	qual, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		t.Fatal("qualification should still exist in storage after expiry")
	}
	if qual.Status != string(types.StatusActive) {
		t.Errorf("status should remain 'active' until explicit expiration sweep; got %s", qual.Status)
	}

	t.Logf("OC-CD-1 PASS: expired qualification correctly denied at block 101 and 200")
}

// ==================================================================================
// OC-CD-2: Below-Threshold Accuracy Sneaking Through Track Record
//
// ATTACK: Validator builds a domain reputation with 76% accuracy (760_000 BPS),
// which is just below the MinAccuracyForQual threshold of 77% (770_000 BPS).
// They attempt to qualify via track_record pathway. The system must reject this.
//
// Without this defense, a validator with poor accuracy could gain qualification
// and be assigned to verification rounds, degrading knowledge quality.
//
// DEFENSE: qualifyByTrackRecord checks domainRep.Accuracy < params.MinAccuracyForQual.
// ==================================================================================

func TestOC_CD2_MinAccuracyThreshold(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "organic_chemistry"
	validatorAddr := testAddr("lowaccval")

	mockOK.domainExists[domain] = true
	mockOK.domainDepths[domain] = 3
	mockOK.domainStrata[domain] = 4

	// Set up domain reputation: 25 verifications (above MinVerificationsForQual=19),
	// but only 19 correct out of 25 = 76% accuracy = 760_000 BPS (below 770_000 threshold)
	rep := types.IsolatedReputation{
		ValidatorAddr: validatorAddr,
		GlobalReputation: &types.GlobalReputation{
			Score:                700,
			TotalVerifications:   25,
			CorrectVerifications: 19,
			Accuracy:             760_000, // 19/25 = 76%
		},
		DomainReputations: []*types.DomainReputationEntry{
			{
				Key: domain,
				Value: &types.DomainReputation{
					Domain:               domain,
					Stratum:              4,
					Score:                700,
					TotalVerifications:   25,
					CorrectVerifications: 19,
					Accuracy:             760_000, // 76% -- below 77% threshold
				},
			},
		},
		CreatedAtBlock:   10,
		LastUpdatedBlock: 90,
	}
	k.SetIsolatedReputation(ctx, rep)

	// ATTACK: attempt track_record qualification with below-threshold accuracy
	_, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayTrackRecord,
		"", nil, "",
	)
	if err == nil {
		t.Error("OC-CD-2 DEFENSE FAILURE: validator with 76% accuracy (below 77% threshold) " +
			"was granted track_record qualification. An attacker could accumulate just enough " +
			"verifications while maintaining poor accuracy, then qualify to pollute verification " +
			"rounds with unreliable judgments.")
	}
	if err != nil && err != types.ErrInsufficientTrackRecord {
		t.Errorf("expected ErrInsufficientTrackRecord, got: %v", err)
	}

	// Verify that exactly at threshold (77%) would succeed:
	// 77 correct out of 100 = 770_000 BPS = exactly MinAccuracyForQual
	repAtThreshold := types.IsolatedReputation{
		ValidatorAddr: validatorAddr,
		GlobalReputation: &types.GlobalReputation{
			Score:                750,
			TotalVerifications:   100,
			CorrectVerifications: 77,
			Accuracy:             770_000, // exactly 77%
		},
		DomainReputations: []*types.DomainReputationEntry{
			{
				Key: domain,
				Value: &types.DomainReputation{
					Domain:               domain,
					Stratum:              4,
					Score:                750,
					TotalVerifications:   100,
					CorrectVerifications: 77,
					Accuracy:             770_000, // exactly at threshold
				},
			},
		},
		CreatedAtBlock:   10,
		LastUpdatedBlock: 90,
	}
	k.SetIsolatedReputation(ctx, repAtThreshold)

	qual, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayTrackRecord,
		"", nil, "",
	)
	if err != nil {
		t.Errorf("OC-CD-2 boundary: validator at exactly 77%% threshold should qualify, got: %v", err)
	}
	if err == nil && qual.Status != string(types.StatusActive) {
		t.Errorf("OC-CD-2 boundary: expected active status, got %s", qual.Status)
	}

	t.Logf("OC-CD-2 PASS: 76%% accuracy rejected, 77%% accuracy accepted (threshold enforced)")
}

// ==================================================================================
// OC-CD-3: Duplicate Qualification Creation
//
// ATTACK: Validator already holds an active qualification for a domain.
// They submit a second qualification request for the same domain. The system
// must reject this to prevent:
// - Double-counting of voting weight
// - Resetting of probationary metrics
// - Circumventing suspension by re-qualifying
//
// DEFENSE: ProcessQualificationRequest checks for existing active/probationary
// qualification and returns ErrAlreadyQualified.
// ==================================================================================

func TestOC_CD3_DuplicateQualification(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "number_theory"
	validatorAddr := testAddr("dup_val")

	mockOK.domainExists[domain] = true
	mockOK.domainDepths[domain] = 3
	mockOK.domainStrata[domain] = 5

	// First qualification: stake commitment (succeeds)
	_, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayStakeCommitment,
		"30000000", // above 22M threshold
		nil, "",
	)
	if err != nil {
		t.Fatalf("first qualification should succeed: %v", err)
	}

	// ATTACK 1: Duplicate via same pathway (stake_commitment again)
	_, err = k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayStakeCommitment,
		"50000000", // even more stake
		nil, "",
	)
	if err == nil {
		t.Error("OC-CD-3 DEFENSE FAILURE: duplicate qualification via same pathway was accepted. " +
			"An attacker could repeatedly re-qualify to reset probationary period metrics, " +
			"effectively evading the probationary-to-active transition requirements.")
	}
	if err != nil && err != types.ErrAlreadyQualified {
		t.Errorf("expected ErrAlreadyQualified for same-pathway duplicate, got: %v", err)
	}

	// ATTACK 2: Duplicate via different pathway (cross_reference)
	endorser1, endorser2, endorser3 := testAddr("e1_dup"), testAddr("e2_dup"), testAddr("e3_dup")
	for _, e := range []string{endorser1, endorser2, endorser3} {
		k.SetQualification(ctx, types.DomainQualification{
			ValidatorAddr: e,
			Domain:        domain,
			Pathway:       string(types.PathwayStakeCommitment),
			Status:        string(types.StatusActive),
		})
	}

	_, err = k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayCrossReference,
		"",
		[]string{endorser1, endorser2, endorser3},
		"",
	)
	if err == nil {
		t.Error("OC-CD-3 DEFENSE FAILURE: duplicate qualification via different pathway was accepted. " +
			"An attacker could hold multiple qualifications for the same domain through different " +
			"pathways, potentially gaining amplified voting weight.")
	}
	if err != nil && err != types.ErrAlreadyQualified {
		t.Errorf("expected ErrAlreadyQualified for cross-pathway duplicate, got: %v", err)
	}

	// Verify: after existing qualification is expired/revoked, re-qualification should work
	qual, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		t.Fatal("qualification should exist")
	}
	qual.Status = string(types.StatusExpired)
	k.SetQualification(ctx, qual)

	// Now re-qualification should succeed (previous is expired)
	_, err = k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayStakeCommitment,
		"30000000",
		nil, "",
	)
	if err != nil {
		t.Errorf("re-qualification after expiry should succeed, got: %v", err)
	}

	t.Logf("OC-CD-3 PASS: duplicate qualification rejected (same pathway and cross-pathway), re-qualification after expiry allowed")
}

// ==================================================================================
// OC-CD-4: Self-Endorsement via Cross-Reference Pathway
//
// ATTACK: Validator lists themselves as one of their own endorsers in a
// cross_reference qualification request. If self-endorsement is counted,
// the validator only needs 2 legitimate endorsers instead of 3
// (MinEndorsersForQual=3), lowering the collusion threshold.
//
// DEFENSE: qualifyByCrossReference skips endorsers where endorser == validatorAddr.
// The filtered count must still meet MinEndorsersForQual.
// ==================================================================================

func TestOC_CD4_SelfEndorsement(t *testing.T) {
	k, ctx, _, mockOK := setupKeeper(t)

	domain := "topology"
	validatorAddr := testAddr("selfendo")

	mockOK.domainExists[domain] = true
	mockOK.domainDepths[domain] = 3
	mockOK.domainStrata[domain] = 5

	// Set up 2 legitimate endorsers (below the MinEndorsersForQual=3 threshold)
	endorser1 := testAddr("legit_e1")
	endorser2 := testAddr("legit_e2")

	for _, e := range []string{endorser1, endorser2} {
		k.SetQualification(ctx, types.DomainQualification{
			ValidatorAddr: e,
			Domain:        domain,
			Pathway:       string(types.PathwayStakeCommitment),
			Status:        string(types.StatusActive),
		})
	}

	// Also give the validator themselves a qualification so self-endorsement
	// would "look" valid if the self-check were missing
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: validatorAddr,
		Domain:        "other_domain", // Different domain so no ErrAlreadyQualified
		Pathway:       string(types.PathwayStakeCommitment),
		Status:        string(types.StatusActive),
	})
	// Note: The validator does NOT have a qualification for "topology" itself,
	// but we include them in the endorser list to test the self-endorsement filter.
	// Even if they did have one, self-endorsement should be excluded.

	// ATTACK: Include self as 3rd endorser to reach the threshold of 3
	_, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayCrossReference,
		"",
		[]string{endorser1, endorser2, validatorAddr}, // self-endorsement attempt
		"",
	)
	if err == nil {
		t.Error("OC-CD-4 DEFENSE FAILURE: self-endorsement was counted toward the endorser " +
			"threshold. An attacker only needs to collude with 2 validators instead of 3 " +
			"by including themselves as an endorser, reducing the cost of a Sybil attack " +
			"on the cross-reference pathway.")
	}
	if err != nil && err != types.ErrInsufficientEndorsers {
		t.Errorf("expected ErrInsufficientEndorsers (self filtered out, only 2 remain), got: %v", err)
	}

	// Verify: with 3 legitimate (non-self) endorsers, it should succeed
	endorser3 := testAddr("legit_e3")
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: endorser3,
		Domain:        domain,
		Pathway:       string(types.PathwayStakeCommitment),
		Status:        string(types.StatusActive),
	})

	qual, err := k.ProcessQualificationRequest(
		ctx, validatorAddr, domain,
		types.PathwayCrossReference,
		"",
		[]string{endorser1, endorser2, endorser3},
		"",
	)
	if err != nil {
		t.Errorf("3 legitimate endorsers should succeed, got: %v", err)
	}
	if err == nil {
		// Verify the self was not sneaked into the endorsers list
		for _, e := range qual.Endorsers {
			if e == validatorAddr {
				t.Error("OC-CD-4 DEFENSE FAILURE: validator's own address appears in stored endorsers list")
			}
		}
		if len(qual.Endorsers) != 3 {
			t.Errorf("expected 3 endorsers stored, got %d", len(qual.Endorsers))
		}
	}

	t.Logf("OC-CD-4 PASS: self-endorsement excluded from count, 2+self rejected, 3 legitimate accepted")
}
