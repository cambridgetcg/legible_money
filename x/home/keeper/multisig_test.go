package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmath "cosmossdk.io/math"

	"github.com/legible-money/legible/x/home/types"
)

// ==================== MultiSig Config ====================

func TestSetGetMultiSigConfig(t *testing.T) {
	k, ctx := setupKeeper(t)

	config := &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 2,
		Approvers:         []string{owner1, owner2, user1},
		DeadlineBlocks:    100,
		ThresholdAmount:   "1000000",
	}

	err := k.SetMultiSigConfig(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error setting config: %v", err)
	}

	got, found := k.GetMultiSigConfig(ctx, "home-1")
	if !found {
		t.Fatal("config not found after set")
	}
	if got.RequiredApprovals != 2 {
		t.Errorf("expected required_approvals=2, got %d", got.RequiredApprovals)
	}
	if len(got.Approvers) != 3 {
		t.Errorf("expected 3 approvers, got %d", len(got.Approvers))
	}
	if got.DeadlineBlocks != 100 {
		t.Errorf("expected deadline_blocks=100, got %d", got.DeadlineBlocks)
	}
	if got.ThresholdAmount != "1000000" {
		t.Errorf("expected threshold_amount=1000000, got %s", got.ThresholdAmount)
	}

	// Not found case
	_, found = k.GetMultiSigConfig(ctx, "nonexistent")
	if found {
		t.Error("expected not found for nonexistent home")
	}
}

func TestSetMultiSigConfigValidation(t *testing.T) {
	k, ctx := setupKeeper(t)

	// RequiredApprovals = 0
	err := k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 0,
		Approvers:         []string{owner1},
		DeadlineBlocks:    100,
	})
	if err == nil {
		t.Error("expected error for required_approvals=0")
	}

	// RequiredApprovals > len(Approvers)
	err = k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 3,
		Approvers:         []string{owner1, owner2},
		DeadlineBlocks:    100,
	})
	if err == nil {
		t.Error("expected error when required > approver count")
	}

	// DeadlineBlocks = 0
	err = k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 1,
		Approvers:         []string{owner1},
		DeadlineBlocks:    0,
	})
	if err == nil {
		t.Error("expected error for deadline_blocks=0")
	}
}

// ==================== Propose Pending Approval ====================

func TestProposePendingApproval(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set up config first
	err := k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 2,
		Approvers:         []string{owner1, owner2, user1},
		DeadlineBlocks:    50,
	})
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	approvalId, err := k.ProposePendingApproval(ctx, "home-1", "transfer", []byte("hash123"), owner1)
	if err != nil {
		t.Fatalf("unexpected error proposing approval: %v", err)
	}
	if approvalId == "" {
		t.Fatal("expected non-empty approval ID")
	}

	// Verify stored
	approval, found := k.GetPendingApproval(ctx, approvalId)
	if !found {
		t.Fatal("approval not found after creation")
	}
	if approval.HomeId != "home-1" {
		t.Errorf("expected home_id=home-1, got %s", approval.HomeId)
	}
	if approval.OperationType != "transfer" {
		t.Errorf("expected operation_type=transfer, got %s", approval.OperationType)
	}
	// Proposer is an approver, so auto-approved
	if len(approval.Approvals) != 1 || approval.Approvals[0] != owner1 {
		t.Errorf("expected auto-approval from proposer, got %v", approval.Approvals)
	}
	if approval.ExpiresAtHeight != approval.CreatedAtHeight+50 {
		t.Errorf("expected expiry = created+50, got %d (created=%d)", approval.ExpiresAtHeight, approval.CreatedAtHeight)
	}
}

func TestProposePendingApprovalNoConfig(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, err := k.ProposePendingApproval(ctx, "home-1", "transfer", []byte("hash"), owner1)
	if err == nil {
		t.Error("expected error when no multi-sig config exists")
	}
}

// ==================== Approve Reaches Threshold (2-of-3) ====================

func TestApproveReachesThreshold(t *testing.T) {
	k, ctx := setupKeeper(t)

	err := k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 2,
		Approvers:         []string{owner1, owner2, user1},
		DeadlineBlocks:    100,
	})
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	// Propose (non-approver proposer — no auto-approve)
	nonApprover := testAddr("outsider")
	approvalId, err := k.ProposePendingApproval(ctx, "home-1", "stake", []byte("payload"), nonApprover)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First approval
	met, err := k.ApprovePendingApproval(ctx, approvalId, owner1)
	if err != nil {
		t.Fatalf("unexpected error on first approve: %v", err)
	}
	if met {
		t.Error("threshold should NOT be met after 1 approval (need 2)")
	}

	// Second approval — threshold met
	met, err = k.ApprovePendingApproval(ctx, approvalId, owner2)
	if err != nil {
		t.Fatalf("unexpected error on second approve: %v", err)
	}
	if !met {
		t.Error("threshold SHOULD be met after 2 approvals")
	}

	// Approval should be cleaned up after threshold met
	_, found := k.GetPendingApproval(ctx, approvalId)
	if found {
		t.Error("approval should be deleted after threshold met")
	}
}

// ==================== Approve Not Yet Threshold (1-of-3) ====================

func TestApproveNotYetThreshold(t *testing.T) {
	k, ctx := setupKeeper(t)

	err := k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 3,
		Approvers:         []string{owner1, owner2, user1},
		DeadlineBlocks:    100,
	})
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	nonApprover := testAddr("outsider")
	approvalId, err := k.ProposePendingApproval(ctx, "home-1", "transfer", []byte("hash"), nonApprover)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only one approval
	met, err := k.ApprovePendingApproval(ctx, approvalId, owner1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if met {
		t.Error("threshold should NOT be met after 1 of 3 approvals")
	}

	// Approval should still exist
	approval, found := k.GetPendingApproval(ctx, approvalId)
	if !found {
		t.Error("approval should still exist")
	}
	if len(approval.Approvals) != 1 {
		t.Errorf("expected 1 approval, got %d", len(approval.Approvals))
	}
}

// ==================== Expired Approval Cleaned ====================

func TestExpiredApprovalCleaned(t *testing.T) {
	k, ctx := setupKeeper(t)

	err := k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 2,
		Approvers:         []string{owner1, owner2},
		DeadlineBlocks:    10,
	})
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	approvalId, err := k.ProposePendingApproval(ctx, "home-1", "transfer", []byte("hash"), owner1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it exists
	_, found := k.GetPendingApproval(ctx, approvalId)
	if !found {
		t.Fatal("approval should exist before expiry")
	}

	// Advance past deadline (created at height 100, deadline 10 → expires at 110)
	ctx = ctx.WithBlockHeight(111)

	// CleanExpiredApprovals should remove it
	k.CleanExpiredApprovals(ctx)

	_, found = k.GetPendingApproval(ctx, approvalId)
	if found {
		t.Error("expired approval should have been cleaned")
	}
}

func TestExpiredApprovalRejectsApprove(t *testing.T) {
	k, ctx := setupKeeper(t)

	err := k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 2,
		Approvers:         []string{owner1, owner2},
		DeadlineBlocks:    10,
	})
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	nonApprover := testAddr("outsider")
	approvalId, err := k.ProposePendingApproval(ctx, "home-1", "transfer", []byte("hash"), nonApprover)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Advance past deadline
	ctx = ctx.WithBlockHeight(111)

	_, err = k.ApprovePendingApproval(ctx, approvalId, owner1)
	if err == nil {
		t.Error("expected error approving expired approval")
	}
}

// ==================== Non-Approver Rejected ====================

func TestNonApproverRejected(t *testing.T) {
	k, ctx := setupKeeper(t)

	err := k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 2,
		Approvers:         []string{owner1, owner2},
		DeadlineBlocks:    100,
	})
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	nonApprover := testAddr("outsider")
	approvalId, err := k.ProposePendingApproval(ctx, "home-1", "transfer", []byte("hash"), nonApprover)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to approve with unauthorized address
	_, err = k.ApprovePendingApproval(ctx, approvalId, nonApprover)
	if err == nil {
		t.Error("expected error for non-approver")
	}
}

func TestAlreadyApprovedRejected(t *testing.T) {
	k, ctx := setupKeeper(t)

	err := k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 3,
		Approvers:         []string{owner1, owner2, user1},
		DeadlineBlocks:    100,
	})
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	nonApprover := testAddr("outsider")
	approvalId, err := k.ProposePendingApproval(ctx, "home-1", "transfer", []byte("hash"), nonApprover)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First approval
	_, err = k.ApprovePendingApproval(ctx, approvalId, owner1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duplicate approval
	_, err = k.ApprovePendingApproval(ctx, approvalId, owner1)
	if err == nil {
		t.Error("expected error for duplicate approval")
	}
}

// ==================== CheckNeedsMultiSig ====================

func TestCheckNeedsMultiSig(t *testing.T) {
	k, ctx := setupKeeper(t)

	// No config — should not need multi-sig
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(5000000)))
	if k.CheckNeedsMultiSig(ctx, "home-1", coins) {
		t.Error("should not require multi-sig when no config exists")
	}

	// Set config with threshold
	err := k.SetMultiSigConfig(ctx, &types.MultiSigConfig{
		HomeId:            "home-1",
		RequiredApprovals: 2,
		Approvers:         []string{owner1, owner2},
		DeadlineBlocks:    100,
		ThresholdAmount:   "1000000",
	})
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	// Below threshold
	below := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(999999)))
	if k.CheckNeedsMultiSig(ctx, "home-1", below) {
		t.Error("should not require multi-sig for amount below threshold")
	}

	// At threshold
	atThreshold := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(1000000)))
	if !k.CheckNeedsMultiSig(ctx, "home-1", atThreshold) {
		t.Error("should require multi-sig for amount at threshold")
	}

	// Above threshold
	above := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(2000000)))
	if !k.CheckNeedsMultiSig(ctx, "home-1", above) {
		t.Error("should require multi-sig for amount above threshold")
	}
}
