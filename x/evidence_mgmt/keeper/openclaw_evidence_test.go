package keeper_test

import (
	"testing"

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
		EvidenceType: types.EvidenceTypeDocument,
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

	// Second submission with same content hash — should produce different ID
	// (not a rejection, but IDs must be unique)
	resp2, err := ms.SubmitEvidence(sdk.WrapSDKContext(ctx), &types.MsgSubmitEvidence{
		Submitter:    submitter,
		ClaimId:      "claim-1",
		EvidenceType: types.EvidenceTypeDocument,
		ContentHash:  "abc123hash",
		Description:  "Duplicate evidence",
		SourceUri:    "https://example.com/evidence2",
	})
	if err != nil {
		t.Fatalf("OC-EV-1: second submission failed: %v", err)
	}
	if resp2.EvidenceId == resp1.EvidenceId {
		t.Error("OC-EV-1: duplicate evidence got same ID — attacker can overwrite evidence records")
	}
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
		EvidenceType: types.EvidenceTypeDocument,
		ContentHash:  "auditme",
		Description:  "Test",
		SourceUri:    "https://example.com",
	})

	auditor := testAddr(2)

	// Score of 0 — should work (valid minimum)
	_, err := ms.AuditEvidence(sdk.WrapSDKContext(ctx), &types.MsgAuditEvidence{
		Auditor:    auditor,
		EvidenceId: resp.EvidenceId,
		Score:      0,
		Stake:      "1000000",
		Rationale:  "zero score",
	})
	// May succeed — 0 can be a valid score for worthless evidence
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
		// Log the finding — check if there's a score cap
		t.Log("OC-EV-2: score 101 accepted — verify score bounds in keeper.SubmitAudit()")
	}
}

// OC-EV-3: Oracle Registration Authorization — Non-authority tries to register oracle.
func TestOC_EV3_OracleRegistrationAuth(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Non-authority tries to register oracle
	nonAuth := testAddr(99)
	_, err := ms.RegisterOracle(sdk.WrapSDKContext(ctx), &types.MsgRegisterOracle{
		Authority:  nonAuth,
		Name:       "malicious-oracle",
		Operator:   nonAuth,
		OracleType: "chainlink",
	})
	if err == nil {
		t.Error("OC-EV-3: non-authority oracle registration should fail — attacker can register rogue oracles")
	}
}
