package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/evidence_mgmt/keeper"
	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

// ============ OpenClaw Adversarial Tests: Evidence Management Module ============
//
// These tests probe evidence submission, audit integrity, and oracle operations
// for authorization gaps, duplicate submissions, and state manipulation.

// OC-EV-1: Duplicate Evidence Submission — Submit same evidence twice.
func TestOC_EV1_DuplicateEvidenceSubmission(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	submitter := testAddr(1)

	// First submission — succeeds
	resp1, err := ms.SubmitEvidence(sdk.WrapSDKContext(ctx), &types.MsgSubmitEvidence{
		Submitter:    submitter,
		ClaimId:      "claim-1",
		EvidenceType: string(types.EvidenceTypeDocument),
		ContentHash:  "abc123hash",
		Description:  "Test evidence",
		SourceUri:    "https://example.com/evidence",
	})
	if err != nil {
		t.Fatalf("OC-EV-1: first submission failed: %v", err)
	}
	if resp1.EvidenceId == "" {
		t.Fatal("OC-EV-1: expected evidence ID in response")
	}

	// Second submission with same content hash — keeper should reject duplicate
	_, err = ms.SubmitEvidence(sdk.WrapSDKContext(ctx), &types.MsgSubmitEvidence{
		Submitter:    submitter,
		ClaimId:      "claim-1",
		EvidenceType: string(types.EvidenceTypeDocument),
		ContentHash:  "abc123hash",
		Description:  "Duplicate evidence",
		SourceUri:    "https://example.com/evidence2",
	})
	if err == nil {
		t.Error("OC-EV-1: duplicate content hash should be rejected — attacker can spam duplicate evidence")
	}
	// DEFENDED: Keeper rejects duplicate content hashes per claim
}

// OC-EV-2: Audit Score Bounds — Submit audit with out-of-range score.
func TestOC_EV2_AuditScoreBounds(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// First create evidence to audit
	submitter := testAddr(1)
	resp, _ := ms.SubmitEvidence(sdk.WrapSDKContext(ctx), &types.MsgSubmitEvidence{
		Submitter:    submitter,
		ClaimId:      "claim-audit",
		EvidenceType: string(types.EvidenceTypeDocument),
		ContentHash:  "auditme",
		Description:  "Test",
		SourceUri:    "https://example.com",
	})

	auditor := testAddr(2)

	// Score of 0 — valid minimum
	_, err := ms.AuditEvidence(sdk.WrapSDKContext(ctx), &types.MsgAuditEvidence{
		Auditor:    auditor,
		EvidenceId: resp.EvidenceId,
		Score:      0,
		Stake:      "1000000",
		Rationale:  "zero score",
	})
	// 0 is a valid score for worthless evidence
	_ = err

	// Score of 101 (>100) — should fail if max is 100
	auditor2 := testAddr(3)
	_, err = ms.AuditEvidence(sdk.WrapSDKContext(ctx), &types.MsgAuditEvidence{
		Auditor:    auditor2,
		EvidenceId: resp.EvidenceId,
		Score:      101,
		Stake:      "1000000",
		Rationale:  "too high",
	})
	if err == nil {
		t.Log("OC-EV-2: score 101 accepted — verify score bounds in keeper.SubmitAudit()")
	}
}

// OC-EV-3: Oracle Registration — Verify operator-only registration with required fields.
func TestOC_EV3_OracleRegistrationValidation(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	operator := testAddr(10)
	bk.fundAccount(operator, sdkmath.NewInt(500000000))

	// Register oracle with valid data types and sufficient stake
	_, err := ms.RegisterOracle(sdk.WrapSDKContext(ctx), &types.MsgRegisterOracle{
		Operator:    operator,
		Name:        "test-oracle",
		Description: "A test oracle",
		DataTypes:   []string{string(types.EvidenceTypeOracleFeed)},
		Stake:       "100000000",
	})
	if err != nil {
		t.Fatalf("OC-EV-3: valid registration failed: %v", err)
	}

	// Register oracle with invalid data type (fund this operator too)
	badOp := testAddr(11)
	bk.fundAccount(badOp, sdkmath.NewInt(500000000))
	_, err = ms.RegisterOracle(sdk.WrapSDKContext(ctx), &types.MsgRegisterOracle{
		Operator:    badOp,
		Name:        "bad-oracle",
		Description: "Bad type",
		DataTypes:   []string{"invalid_type"},
		Stake:       "100000000",
	})
	if err == nil {
		t.Error("OC-EV-3: oracle with invalid data type should fail — attacker can register rogue oracle types")
	}
}
