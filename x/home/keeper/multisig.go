package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// --- MultiSigConfig CRUD ---

// SetMultiSigConfig stores the multi-sig configuration for a home.
func (k Keeper) SetMultiSigConfig(ctx sdk.Context, config *types.MultiSigConfig) error {
	if config.RequiredApprovals == 0 {
		return types.ErrInvalidMultiSigConfig.Wrap("required_approvals must be > 0")
	}
	if uint32(len(config.Approvers)) < config.RequiredApprovals {
		return types.ErrInvalidMultiSigConfig.Wrapf(
			"required_approvals (%d) exceeds approver count (%d)",
			config.RequiredApprovals, len(config.Approvers))
	}
	if config.DeadlineBlocks == 0 {
		return types.ErrInvalidMultiSigConfig.Wrap("deadline_blocks must be > 0")
	}

	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal multi-sig config: %w", err)
	}
	if err := store.Set(types.MultiSigConfigKey(config.HomeId), bz); err != nil {
		return fmt.Errorf("failed to set multi-sig config: %w", err)
	}
	return nil
}

// GetMultiSigConfig retrieves the multi-sig configuration for a home.
func (k Keeper) GetMultiSigConfig(ctx sdk.Context, homeId string) (*types.MultiSigConfig, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.MultiSigConfigKey(homeId))
	if err != nil || bz == nil {
		return nil, false
	}
	var config types.MultiSigConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return nil, false
	}
	return &config, true
}

// DeleteMultiSigConfig removes the multi-sig configuration for a home.
func (k Keeper) DeleteMultiSigConfig(ctx sdk.Context, homeId string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.MultiSigConfigKey(homeId)); err != nil {
		panic(fmt.Sprintf("failed to delete multi-sig config: %v", err))
	}
}

// --- PendingApproval CRUD ---

// SetPendingApproval stores a pending approval.
func (k Keeper) SetPendingApproval(ctx sdk.Context, approval *types.PendingApproval) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(approval)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal pending approval: %v", err))
	}
	if err := store.Set(types.PendingApprovalKey(approval.Id), bz); err != nil {
		panic(fmt.Sprintf("failed to set pending approval: %v", err))
	}
}

// GetPendingApproval retrieves a pending approval by ID.
func (k Keeper) GetPendingApproval(ctx sdk.Context, approvalId string) (*types.PendingApproval, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.PendingApprovalKey(approvalId))
	if err != nil || bz == nil {
		return nil, false
	}
	var approval types.PendingApproval
	if err := json.Unmarshal(bz, &approval); err != nil {
		return nil, false
	}
	return &approval, true
}

// DeletePendingApproval removes a pending approval.
func (k Keeper) DeletePendingApproval(ctx sdk.Context, approvalId string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.PendingApprovalKey(approvalId)); err != nil {
		panic(fmt.Sprintf("failed to delete pending approval: %v", err))
	}
}

// --- Multi-Sig Operations ---

// ProposePendingApproval creates a new pending approval for a high-value operation.
// Returns (approvalId, error).
func (k Keeper) ProposePendingApproval(ctx sdk.Context, homeId, opType string, payloadHash []byte, proposer string) (string, error) {
	config, found := k.GetMultiSigConfig(ctx, homeId)
	if !found {
		return "", types.ErrInvalidMultiSigConfig.Wrap("no multi-sig config for home")
	}

	currentHeight := ctx.BlockHeight()

	// Deterministic ID: homeId + height + opType
	approvalId := fmt.Sprintf("approval-%s-%d-%s", homeId, currentHeight, opType)

	approval := &types.PendingApproval{
		Id:              approvalId,
		HomeId:          homeId,
		OperationType:   opType,
		PayloadHash:     payloadHash,
		Approvals:       []string{},
		CreatedAtHeight: currentHeight,
		ExpiresAtHeight: currentHeight + int64(config.DeadlineBlocks),
	}

	// If the proposer is an authorized approver, auto-approve
	for _, addr := range config.Approvers {
		if addr == proposer {
			approval.Approvals = []string{proposer}
			break
		}
	}

	k.SetPendingApproval(ctx, approval)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"multisig_proposal_created",
		sdk.NewAttribute("approval_id", approvalId),
		sdk.NewAttribute("home_id", homeId),
		sdk.NewAttribute("operation_type", opType),
		sdk.NewAttribute("proposer", proposer),
	))

	return approvalId, nil
}

// ApprovePendingApproval adds an approver's signature to a pending approval.
// Returns (thresholdMet bool, error).
func (k Keeper) ApprovePendingApproval(ctx sdk.Context, approvalId, approverAddr string) (bool, error) {
	approval, found := k.GetPendingApproval(ctx, approvalId)
	if !found {
		return false, types.ErrApprovalNotFound
	}

	// Check expiry
	currentHeight := ctx.BlockHeight()
	if currentHeight > approval.ExpiresAtHeight {
		k.DeletePendingApproval(ctx, approvalId)
		return false, types.ErrApprovalExpired
	}

	// Load config to validate approver
	config, configFound := k.GetMultiSigConfig(ctx, approval.HomeId)
	if !configFound {
		return false, types.ErrInvalidMultiSigConfig.Wrap("multi-sig config removed")
	}

	// Check approver is authorized
	isApprover := false
	for _, addr := range config.Approvers {
		if addr == approverAddr {
			isApprover = true
			break
		}
	}
	if !isApprover {
		return false, types.ErrNotApprover.Wrapf("address %s", approverAddr)
	}

	// Check not already approved
	for _, addr := range approval.Approvals {
		if addr == approverAddr {
			return false, types.ErrAlreadyApproved.Wrapf("address %s", approverAddr)
		}
	}

	// Add approval
	approval.Approvals = append(approval.Approvals, approverAddr)
	k.SetPendingApproval(ctx, approval)

	thresholdMet := uint32(len(approval.Approvals)) >= config.RequiredApprovals

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"multisig_approval_added",
		sdk.NewAttribute("approval_id", approvalId),
		sdk.NewAttribute("approver", approverAddr),
		sdk.NewAttribute("current_approvals", fmt.Sprintf("%d", len(approval.Approvals))),
		sdk.NewAttribute("threshold_met", fmt.Sprintf("%t", thresholdMet)),
	))

	if thresholdMet {
		// Clean up the approval once threshold is met
		k.DeletePendingApproval(ctx, approvalId)
	}

	return thresholdMet, nil
}

// CleanExpiredApprovals removes all pending approvals past their deadline.
// Called from BeginBlocker.
func (k Keeper) CleanExpiredApprovals(ctx sdk.Context) {
	currentHeight := ctx.BlockHeight()
	store := k.storeService.OpenKVStore(ctx)

	iter, err := store.Iterator(types.PendingApprovalPrefix, prefixEndBytes(types.PendingApprovalPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	var expiredKeys [][]byte
	for ; iter.Valid(); iter.Next() {
		var approval types.PendingApproval
		if err := json.Unmarshal(iter.Value(), &approval); err != nil {
			continue
		}
		if currentHeight > approval.ExpiresAtHeight {
			expiredKeys = append(expiredKeys, append([]byte{}, iter.Key()...))
		}
	}

	for _, key := range expiredKeys {
		_ = store.Delete(key)
	}
}

// CheckNeedsMultiSig returns true if the amount exceeds the home's multi-sig threshold.
func (k Keeper) CheckNeedsMultiSig(ctx sdk.Context, homeId string, amount sdk.Coins) bool {
	config, found := k.GetMultiSigConfig(ctx, homeId)
	if !found {
		return false
	}

	if config.ThresholdAmount == "" || config.ThresholdAmount == "0" {
		return false
	}

	threshold := new(big.Int)
	if _, ok := threshold.SetString(config.ThresholdAmount, 10); !ok {
		return false
	}

	// Sum all coin amounts
	total := new(big.Int)
	for _, coin := range amount {
		total.Add(total, coin.Amount.BigInt())
	}

	return total.Cmp(threshold) >= 0
}
