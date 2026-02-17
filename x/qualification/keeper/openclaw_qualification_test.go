package keeper_test

// ==================================================================================
// OpenClaw Qualification Adversarial Probe Suite (OC-QUAL-1 through OC-QUAL-4)
//
// Attack surfaces tested:
// - Double qualification (same validator, same domain)
// - Stratum inheritance in wrong direction (high to low)
// - Renewing a revoked qualification
// - UpdateParams authorization bypass
// ==================================================================================

import (
	"testing"

	"github.com/legible-money/legible/x/qualification/keeper"
	"github.com/legible-money/legible/x/qualification/types"
)

// ==================================================================================
// OC-QUAL-1: Double Qualification Attack
//
// ATTACK: Validator qualifies in a domain, then tries to qualify again to
// reset metrics or re-stake at different level.
// If unguarded, a slashed validator could reset their metrics.
//
// DEFENSE: QualifyViaStakeCommitment checks if already qualified.
// ==================================================================================

func TestOC_QUAL1_DoubleQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// First qualification
	_, err := k.QualifyViaStakeCommitment(ctx, "lgmvaloper1abc", "physics", "22000000")
	if err != nil {
		t.Fatalf("first qualification: %v", err)
	}

	// ATTACK: Second qualification (duplicate)
	_, err = k.QualifyViaStakeCommitment(ctx, "lgmvaloper1abc", "physics", "22000000")
	if err == nil {
		t.Fatal("OC-QUAL-1 FAIL: double qualification allowed!")
	}

	t.Logf("OC-QUAL-1 PASS: double qualification correctly rejected")
}

// ==================================================================================
// OC-QUAL-2: Stratum Inheritance Wrong Direction
//
// ATTACK: Qualify in a higher stratum domain (physics, stratum 4), then try to
// inherit down to a lower stratum domain (mathematics, stratum 0).
// If unguarded, validators could bypass the harder lower-stratum requirements.
//
// DEFENSE: QualifyViaStratumInheritance checks source stratum < target stratum.
// ==================================================================================

func TestOC_QUAL2_StratumInheritanceWrongDirection(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Qualify in higher stratum (physics = stratum 4)
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusActive),
	})

	// ATTACK: Inherit from higher to lower stratum (physics -> mathematics)
	_, err := k.QualifyViaStratumInheritance(ctx, "lgmvaloper1abc", "mathematics", "physics")
	if err == nil {
		t.Fatal("OC-QUAL-2 FAIL: stratum inheritance from higher to lower allowed!")
	}

	// Verify no qualification was created for mathematics
	_, found := k.GetQualification(ctx, "lgmvaloper1abc", "mathematics")
	if found {
		t.Fatal("OC-QUAL-2 FAIL: qualification was created despite wrong direction!")
	}

	t.Logf("OC-QUAL-2 PASS: wrong-direction stratum inheritance correctly rejected")
}

// ==================================================================================
// OC-QUAL-3: Renewing a Revoked Qualification
//
// ATTACK: Validator's qualification is revoked (e.g., due to repeated challenge losses),
// then they try to renew it.
// If unguarded, revoked validators could bypass the revocation penalty.
//
// DEFENSE: RenewQualification checks status != revoked.
// ==================================================================================

func TestOC_QUAL3_RenewRevokedQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Set up a revoked qualification
	k.SetQualification(ctx, types.DomainQualification{
		ValidatorAddr: "lgmvaloper1abc",
		Domain:        "physics",
		Status:        string(types.StatusRevoked),
	})

	// ATTACK: Try to renew the revoked qualification
	err := k.RenewQualification(ctx, "lgmvaloper1abc", "physics")
	if err == nil {
		t.Fatal("OC-QUAL-3 FAIL: renewing revoked qualification allowed!")
	}

	// Verify still revoked
	q, _ := k.GetQualification(ctx, "lgmvaloper1abc", "physics")
	if q.Status != string(types.StatusRevoked) {
		t.Fatalf("OC-QUAL-3 FAIL: revoked status changed to %s", q.Status)
	}

	t.Logf("OC-QUAL-3 PASS: revoked qualification renewal correctly rejected")
}

// ==================================================================================
// OC-QUAL-4: UpdateParams Authorization Bypass
//
// ATTACK: Non-authority address tries to update qualification parameters.
// If unguarded, anyone could lower accuracy thresholds or min verifications.
//
// DEFENSE: UpdateParams checks msg.Authority == k.authority.
// ==================================================================================

func TestOC_QUAL4_UpdateParamsAuthBypass(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	ms := keeper.NewMsgServerImpl(k)

	// ATTACK: Random user tries to update params
	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "not_authority",
		Params: types.Params{
			MinVerifications: 1,    // dangerously low
			MinAccuracyBps:   1000, // 10% (instead of 77%)
		},
	})
	if err == nil {
		t.Fatal("OC-QUAL-4 FAIL: unauthorized user was able to update params!")
	}

	// Verify params unchanged
	params := k.GetParams(ctx)
	if params.MinVerifications != 19 {
		t.Fatalf("OC-QUAL-4 FAIL: params modified: MinVerifications=%d", params.MinVerifications)
	}

	t.Logf("OC-QUAL-4 PASS: param update authorization enforced")
}
