package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/compute_pool/types"
)

// SubmitTask creates a new compute task, escrowing the reward from the sender.
func (k Keeper) SubmitTask(ctx sdk.Context, msg *types.MsgSubmitTask) (*types.ComputeTask, error) {
	height := uint64(ctx.BlockHeight())

	// Validate service type
	if !types.IsValidServiceType(msg.Spec.ServiceType) {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidComputeSpec, msg.Spec.ServiceType)
	}

	// Validate reward amount
	rewardAmt := new(big.Int)
	if _, ok := rewardAmt.SetString(msg.Reward, 10); !ok || rewardAmt.Sign() <= 0 {
		return nil, types.ErrInvalidReward
	}

	// Escrow reward: sender → module account
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	rewardCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(rewardAmt)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, rewardCoins); err != nil {
		return nil, fmt.Errorf("failed to escrow reward: %w", err)
	}

	taskId := k.GetNextTaskId(ctx)
	task := &types.ComputeTask{
		Id:            taskId,
		RequesterDID:  msg.RequesterDID,
		RequesterAddr: msg.Sender,
		Spec:          msg.Spec,
		Reward:        msg.Reward,
		Status:        string(types.TaskStatusPending),
		SubmittedAt:   height,
		ExpiresAt:     height + types.DefaultTaskExpiryBlocks,
	}

	k.SetTask(ctx, task)
	k.SetTaskRequesterIndex(ctx, msg.RequesterDID, taskId)

	// Update agent usage tracking
	usage := k.GetOrCreateAgentUsage(ctx, msg.RequesterDID)
	usage.TotalTasks++
	usage.LastActivityAt = height
	k.SetAgentUsage(ctx, usage)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_task_submitted",
			sdk.NewAttribute("task_id", fmt.Sprintf("%d", taskId)),
			sdk.NewAttribute("requester_did", msg.RequesterDID),
			sdk.NewAttribute("service_type", msg.Spec.ServiceType),
			sdk.NewAttribute("max_cu", fmt.Sprintf("%d", msg.Spec.MaxCU)),
			sdk.NewAttribute("reward", msg.Reward),
		),
	)

	return task, nil
}

// AssignTask assigns a pending task to a provider.
// The provider self-assigns by calling this method.
func (k Keeper) AssignTask(ctx sdk.Context, providerAddr string, taskId uint64) error {
	height := uint64(ctx.BlockHeight())

	task, found := k.GetTask(ctx, taskId)
	if !found {
		return types.ErrTaskNotFound
	}

	if task.Status != string(types.TaskStatusPending) {
		return types.ErrTaskAlreadyAssigned
	}

	if height > task.ExpiresAt {
		return types.ErrTaskExpired
	}

	// Validate provider is active
	provider, found := k.GetProvider(ctx, providerAddr)
	if !found {
		return types.ErrProviderNotFound
	}
	if provider.Status != string(types.ProviderStatusActive) {
		return types.ErrProviderNotActive
	}

	// Validate provider offers the required service type
	if provider.ServiceType != task.Spec.ServiceType {
		return fmt.Errorf("%w: provider offers %s, task requires %s",
			types.ErrInvalidComputeSpec, provider.ServiceType, task.Spec.ServiceType)
	}

	task.Status = string(types.TaskStatusAssigned)
	task.Provider = providerAddr
	task.AssignedAt = height
	// Extend expiry for result submission
	task.ExpiresAt = height + types.DefaultTaskResultExpiryBlocks

	k.SetTask(ctx, task)
	k.SetTaskProviderIndex(ctx, providerAddr, taskId)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_task_assigned",
			sdk.NewAttribute("task_id", fmt.Sprintf("%d", taskId)),
			sdk.NewAttribute("provider", providerAddr),
		),
	)

	return nil
}

// SubmitResult records a provider's result for an assigned task.
func (k Keeper) SubmitResult(ctx sdk.Context, providerAddr string, taskId uint64, outputHash string, actualCU uint64, resultMetadata string) error {
	height := uint64(ctx.BlockHeight())

	task, found := k.GetTask(ctx, taskId)
	if !found {
		return types.ErrTaskNotFound
	}

	if task.Status != string(types.TaskStatusAssigned) {
		return types.ErrTaskNotAssigned
	}

	if task.Provider != providerAddr {
		return types.ErrUnauthorized
	}

	if height > task.ExpiresAt {
		return types.ErrTaskExpired
	}

	if task.OutputHash != "" {
		return types.ErrResultAlreadySubmitted
	}

	// Cap actualCU at the task's max
	if actualCU > task.Spec.MaxCU {
		actualCU = task.Spec.MaxCU
	}

	task.Status = string(types.TaskStatusResultPending)
	task.OutputHash = outputHash
	task.ActualCU = actualCU
	task.ResultAt = height
	task.ResultMetadata = resultMetadata
	// Give requester time to verify
	task.ExpiresAt = height + types.DefaultTaskResultExpiryBlocks

	k.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_result_submitted",
			sdk.NewAttribute("task_id", fmt.Sprintf("%d", taskId)),
			sdk.NewAttribute("provider", providerAddr),
			sdk.NewAttribute("output_hash", outputHash),
			sdk.NewAttribute("actual_cu", fmt.Sprintf("%d", actualCU)),
		),
	)

	return nil
}

// VerifyResult accepts or rejects a task result.
// On accept: reward is paid to provider, usage metered.
// On reject: reward returned to requester, provider stake slashed.
func (k Keeper) VerifyResult(ctx sdk.Context, requesterAddr string, taskId uint64, accept bool, reason string) error {
	height := uint64(ctx.BlockHeight())

	task, found := k.GetTask(ctx, taskId)
	if !found {
		return types.ErrTaskNotFound
	}

	if task.Status != string(types.TaskStatusResultPending) {
		return types.ErrTaskNotPendingVerify
	}

	if task.RequesterAddr != requesterAddr {
		return types.ErrUnauthorized
	}

	rewardAmt := new(big.Int)
	rewardAmt.SetString(task.Reward, 10)

	if accept {
		// Pay provider from module escrow
		providerAccAddr, err := sdk.AccAddressFromBech32(task.Provider)
		if err != nil {
			return fmt.Errorf("invalid provider address: %w", err)
		}

		// Calculate payment proportional to actualCU vs maxCU
		actualPayment := new(big.Int)
		if task.ActualCU >= task.Spec.MaxCU {
			actualPayment.Set(rewardAmt)
		} else {
			// payment = reward × actualCU / maxCU
			actualPayment.Mul(rewardAmt, big.NewInt(int64(task.ActualCU)))
			actualPayment.Div(actualPayment, big.NewInt(int64(task.Spec.MaxCU)))
		}

		if actualPayment.Sign() > 0 {
			paymentCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(actualPayment)))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAccAddr, paymentCoins); err != nil {
				return fmt.Errorf("failed to pay provider: %w", err)
			}
		}

		// Refund excess if actualCU < maxCU
		refund := new(big.Int).Sub(rewardAmt, actualPayment)
		if refund.Sign() > 0 {
			requesterAccAddr, _ := sdk.AccAddressFromBech32(requesterAddr)
			refundCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(refund)))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, requesterAccAddr, refundCoins); err != nil {
				k.Logger(ctx).Error("failed to refund excess to requester", "error", err)
			}
		}

		task.Status = string(types.TaskStatusAccepted)
		task.CompletedAt = height

		// Update provider metrics
		provider, found := k.GetProvider(ctx, task.Provider)
		if found {
			provider.TasksServed++
			k.SetProvider(ctx, provider)
		}

		// Update agent usage
		usage := k.GetOrCreateAgentUsage(ctx, task.RequesterDID)
		usage.TotalCU += task.ActualCU
		usage.AcceptedTasks++
		usage.LastActivityAt = height
		totalReward := new(big.Int)
		totalReward.SetString(usage.TotalReward, 10)
		totalReward.Add(totalReward, actualPayment)
		usage.TotalReward = totalReward.String()
		k.SetAgentUsage(ctx, usage)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"compute_result_accepted",
				sdk.NewAttribute("task_id", fmt.Sprintf("%d", taskId)),
				sdk.NewAttribute("provider", task.Provider),
				sdk.NewAttribute("actual_cu", fmt.Sprintf("%d", task.ActualCU)),
				sdk.NewAttribute("payment", actualPayment.String()),
			),
		)
	} else {
		// Reject: return reward to requester
		requesterAccAddr, _ := sdk.AccAddressFromBech32(requesterAddr)
		rewardCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(rewardAmt)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, requesterAccAddr, rewardCoins); err != nil {
			return fmt.Errorf("failed to refund reward: %w", err)
		}

		// Slash provider stake (DefaultSlashOnRejectBps = 500 = 0.5%)
		provider, found := k.GetProvider(ctx, task.Provider)
		if found {
			provider.TasksFailed++
			providerStake := new(big.Int)
			providerStake.SetString(provider.Stake, 10)
			slashAmt := new(big.Int).Mul(providerStake, big.NewInt(int64(types.DefaultSlashOnRejectBps)))
			slashAmt.Div(slashAmt, big.NewInt(1000000))
			if slashAmt.Sign() > 0 {
				providerStake.Sub(providerStake, slashAmt)
				if providerStake.Sign() < 0 {
					providerStake.SetInt64(0)
				}
				provider.Stake = providerStake.String()
			}
			k.SetProvider(ctx, provider)
		}

		task.Status = string(types.TaskStatusRejected)
		task.CompletedAt = height

		// Update agent usage
		usage := k.GetOrCreateAgentUsage(ctx, task.RequesterDID)
		usage.RejectedTasks++
		usage.LastActivityAt = height
		k.SetAgentUsage(ctx, usage)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"compute_result_rejected",
				sdk.NewAttribute("task_id", fmt.Sprintf("%d", taskId)),
				sdk.NewAttribute("provider", task.Provider),
				sdk.NewAttribute("reason", reason),
			),
		)
	}

	k.SetTask(ctx, task)
	return nil
}

// CancelTask cancels a pending (unassigned) task and refunds the reward.
func (k Keeper) CancelTask(ctx sdk.Context, requesterAddr string, taskId uint64) error {
	task, found := k.GetTask(ctx, taskId)
	if !found {
		return types.ErrTaskNotFound
	}

	if task.RequesterAddr != requesterAddr {
		return types.ErrUnauthorized
	}

	if task.Status != string(types.TaskStatusPending) {
		return types.ErrTaskAlreadyAssigned
	}

	// Refund reward
	rewardAmt := new(big.Int)
	rewardAmt.SetString(task.Reward, 10)
	if rewardAmt.Sign() > 0 {
		requesterAccAddr, _ := sdk.AccAddressFromBech32(requesterAddr)
		rewardCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(rewardAmt)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, requesterAccAddr, rewardCoins); err != nil {
			return fmt.Errorf("failed to refund reward: %w", err)
		}
	}

	task.Status = string(types.TaskStatusCancelled)
	task.CompletedAt = uint64(ctx.BlockHeight())
	k.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_task_cancelled",
			sdk.NewAttribute("task_id", fmt.Sprintf("%d", taskId)),
			sdk.NewAttribute("requester", requesterAddr),
		),
	)

	return nil
}

// ExpireUnassignedTasks expires pending tasks past their ExpiresAt block.
// Called in BeginBlock.
func (k Keeper) ExpireUnassignedTasks(ctx sdk.Context) {
	height := uint64(ctx.BlockHeight())

	var toExpire []*types.ComputeTask

	k.IterateTasks(ctx, func(task *types.ComputeTask) bool {
		if height > task.ExpiresAt {
			switch task.Status {
			case string(types.TaskStatusPending), string(types.TaskStatusAssigned), string(types.TaskStatusResultPending):
				toExpire = append(toExpire, task)
			}
		}
		return false
	})

	for _, task := range toExpire {
		// Refund reward to requester
		rewardAmt := new(big.Int)
		rewardAmt.SetString(task.Reward, 10)
		if rewardAmt.Sign() > 0 {
			requesterAccAddr, err := sdk.AccAddressFromBech32(task.RequesterAddr)
			if err == nil {
				rewardCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(rewardAmt)))
				if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, requesterAccAddr, rewardCoins); err != nil {
					k.Logger(ctx).Error("failed to refund expired task reward", "task_id", task.Id, "error", err)
					continue
				}
			}
		}

		// If assigned, penalize the provider
		if task.Provider != "" {
			provider, found := k.GetProvider(ctx, task.Provider)
			if found {
				provider.TasksFailed++
				k.SetProvider(ctx, provider)
			}
		}

		task.Status = string(types.TaskStatusExpired)
		task.CompletedAt = height
		k.SetTask(ctx, task)

		k.Logger(ctx).Debug("expired compute task",
			"task_id", task.Id,
			"previous_status", task.Status,
		)
	}
}
