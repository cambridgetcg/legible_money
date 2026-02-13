package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/schedule/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreateSchedule creates a new scheduled process.
func (k msgServer) CreateSchedule(goCtx context.Context, msg *types.MsgCreateSchedule) (*types.MsgCreateScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	// Check account limit
	activeCount := k.CountActiveByCreator(ctx, msg.Creator)
	if activeCount >= params.MaxActivePerAccount {
		return nil, types.ErrMaxSchedulesExceeded
	}

	// Validate condition
	currentHeight := uint64(ctx.BlockHeight())
	if msg.Condition == nil {
		return nil, types.ErrInvalidCondition
	}
	if err := validateCondition(*msg.Condition, currentHeight, params); err != nil {
		return nil, err
	}

	// Parse and validate prepaid fee
	prepaidFee := new(big.Int)
	if _, ok := prepaidFee.SetString(msg.PrepaidFee, 10); !ok || prepaidFee.Sign() <= 0 {
		return nil, fmt.Errorf("invalid prepaid fee: %s", msg.PrepaidFee)
	}

	feePerExec := new(big.Int)
	if _, ok := feePerExec.SetString(msg.FeePerExecution, 10); !ok || feePerExec.Sign() <= 0 {
		return nil, fmt.Errorf("invalid fee per execution: %s", msg.FeePerExecution)
	}

	// Check minimum prepaid executions
	minPrepaid := new(big.Int).Mul(feePerExec, new(big.Int).SetUint64(params.MinPrepaidExecutions))
	if prepaidFee.Cmp(minPrepaid) < 0 {
		return nil, types.ErrInsufficientFee
	}

	// Deduct prepaid fee from creator
	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}

	feeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(prepaidFee))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, types.ModuleName, sdk.NewCoins(feeCoin)); err != nil {
		return nil, fmt.Errorf("insufficient funds for prepaid fee: %w", err)
	}

	// Generate process ID
	seq := k.NextSequence(ctx)
	processId := fmt.Sprintf("sched-%d", seq)

	// Calculate next execution block
	nextExecAt := calculateNextExecuteAt(*msg.Condition, currentHeight)

	process := types.ScheduleProcess{
		Id:               processId,
		Creator:          msg.Creator,
		Condition:        msg.Condition,
		Status:           types.StatusActive,
		ExecutionCount:   0,
		MaxExecutions:    msg.MaxExecutions,
		RemainingFee:     prepaidFee.String(),
		FeePerExecution:  msg.FeePerExecution,
		TargetAddress:    msg.TargetAddress,
		CallData:         msg.CallData,
		TransferValue:    msg.TransferValue,
		LinkedEntityType: msg.LinkedEntityType,
		LinkedEntityId:   msg.LinkedEntityId,
		CreatedAtBlock:   currentHeight,
		NextExecuteAt:    nextExecAt,
	}

	k.SetProcess(ctx, process)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"schedule_created",
			sdk.NewAttribute("process_id", processId),
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("next_execute_at", fmt.Sprintf("%d", nextExecAt)),
		),
	})

	k.Logger(ctx).Info("schedule created",
		"process_id", processId,
		"creator", msg.Creator,
	)

	return &types.MsgCreateScheduleResponse{ProcessId: processId}, nil
}

// PauseSchedule pauses an active schedule.
func (k msgServer) PauseSchedule(goCtx context.Context, msg *types.MsgPauseSchedule) (*types.MsgPauseScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	process, found := k.GetProcess(ctx, msg.ProcessId)
	if !found {
		return nil, types.ErrProcessNotFound
	}
	if process.Creator != msg.Creator {
		return nil, types.ErrUnauthorized
	}
	if process.Status != types.StatusActive {
		return nil, types.ErrProcessNotActive
	}

	// Remove old indexes before status change
	k.RemoveStatusIndex(ctx, string(process.Status), process.Id)
	if process.NextExecuteAt > 0 {
		k.RemoveTimeIndex(ctx, process.NextExecuteAt, process.Id)
	}

	process.Status = types.StatusPaused
	k.SetProcess(ctx, process)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent("schedule_paused", sdk.NewAttribute("process_id", msg.ProcessId)),
	})

	return &types.MsgPauseScheduleResponse{}, nil
}

// ResumeSchedule resumes a paused schedule.
func (k msgServer) ResumeSchedule(goCtx context.Context, msg *types.MsgResumeSchedule) (*types.MsgResumeScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	process, found := k.GetProcess(ctx, msg.ProcessId)
	if !found {
		return nil, types.ErrProcessNotFound
	}
	if process.Creator != msg.Creator {
		return nil, types.ErrUnauthorized
	}
	if process.Status != types.StatusPaused {
		return nil, types.ErrProcessNotPaused
	}

	// Remove old status index
	k.RemoveStatusIndex(ctx, string(process.Status), process.Id)

	process.Status = types.StatusActive
	// Recalculate next execution from current height
	currentHeight := uint64(ctx.BlockHeight())
	if process.Condition != nil {
		process.NextExecuteAt = calculateNextExecuteAt(*process.Condition, currentHeight)
	}

	k.SetProcess(ctx, process)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent("schedule_resumed", sdk.NewAttribute("process_id", msg.ProcessId)),
	})

	return &types.MsgResumeScheduleResponse{}, nil
}

// CancelSchedule cancels a schedule and refunds remaining fee.
func (k msgServer) CancelSchedule(goCtx context.Context, msg *types.MsgCancelSchedule) (*types.MsgCancelScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	process, found := k.GetProcess(ctx, msg.ProcessId)
	if !found {
		return nil, types.ErrProcessNotFound
	}
	if process.Creator != msg.Creator {
		return nil, types.ErrUnauthorized
	}
	if process.Status == types.StatusCompleted {
		return nil, types.ErrProcessCompleted
	}
	if process.Status == types.StatusCancelled {
		return nil, types.ErrProcessCancelled
	}

	// Refund remaining fee
	refundAmount := new(big.Int)
	refundAmount.SetString(process.RemainingFee, 10)

	refundedStr := "0"
	if refundAmount.Sign() > 0 {
		creatorAddr, err := sdk.AccAddressFromBech32(process.Creator)
		if err != nil {
			return nil, fmt.Errorf("invalid creator address: %w", err)
		}
		refundCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(refundAmount))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, sdk.NewCoins(refundCoin)); err != nil {
			return nil, fmt.Errorf("failed to refund: %w", err)
		}
		refundedStr = refundAmount.String()
	}

	// Remove old indexes
	k.RemoveStatusIndex(ctx, string(process.Status), process.Id)
	if process.NextExecuteAt > 0 {
		k.RemoveTimeIndex(ctx, process.NextExecuteAt, process.Id)
	}

	process.Status = types.StatusCancelled
	process.RemainingFee = "0"
	process.NextExecuteAt = 0
	k.SetProcess(ctx, process)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent("schedule_cancelled",
			sdk.NewAttribute("process_id", msg.ProcessId),
			sdk.NewAttribute("refunded", refundedStr),
		),
	})

	return &types.MsgCancelScheduleResponse{RefundedAmount: refundedStr}, nil
}

// AddFee adds prepaid fee to an existing schedule.
func (k msgServer) AddFee(goCtx context.Context, msg *types.MsgAddFee) (*types.MsgAddFeeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	process, found := k.GetProcess(ctx, msg.ProcessId)
	if !found {
		return nil, types.ErrProcessNotFound
	}
	if process.Creator != msg.Creator {
		return nil, types.ErrUnauthorized
	}

	addAmount := new(big.Int)
	if _, ok := addAmount.SetString(msg.Amount, 10); !ok || addAmount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	// Transfer fee from creator to module
	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}
	feeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(addAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, types.ModuleName, sdk.NewCoins(feeCoin)); err != nil {
		return nil, fmt.Errorf("insufficient funds: %w", err)
	}

	// Update remaining fee
	remaining := new(big.Int)
	remaining.SetString(process.RemainingFee, 10)
	remaining.Add(remaining, addAmount)
	process.RemainingFee = remaining.String()

	// Reactivate if exhausted
	if process.Status == types.StatusExhausted {
		k.RemoveStatusIndex(ctx, string(process.Status), process.Id)
		process.Status = types.StatusActive
		currentHeight := uint64(ctx.BlockHeight())
		if process.Condition != nil {
			process.NextExecuteAt = calculateNextExecuteAt(*process.Condition, currentHeight)
		}
	}

	k.SetProcess(ctx, process)

	return &types.MsgAddFeeResponse{}, nil
}

// UpdateParams updates module parameters.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != k.GetAuthority() {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	k.SetParams(ctx, msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

// ---------- Helpers ----------

// validateCondition validates a schedule condition against module params.
func validateCondition(cond types.ScheduleCondition, currentHeight uint64, params types.Params) error {
	if cond.TimeCondition == nil && cond.LogicCondition == nil && cond.CompoundCondition == nil {
		return types.ErrInvalidCondition
	}

	if cond.TimeCondition != nil {
		tc := cond.TimeCondition
		switch tc.Type {
		case types.TimeAbsolute:
			if tc.ExecuteAtBlock <= currentHeight {
				return types.ErrInvalidExecuteAt
			}
			if tc.ExecuteAtBlock > currentHeight+params.MaxFutureBlocks {
				return types.ErrInvalidFutureBlock
			}
		case types.TimeRelative:
			if tc.ExecuteAtBlock == 0 {
				return types.ErrInvalidExecuteAt
			}
		case types.TimePeriodic:
			if tc.IntervalBlocks < params.MinIntervalBlocks {
				return types.ErrInvalidInterval
			}
		default:
			return types.ErrInvalidCondition
		}
	}

	if cond.CompoundCondition != nil {
		if cond.CompoundCondition.Depth() > 3 {
			return types.ErrCompoundDepthExceeded
		}
	}

	return nil
}

// calculateNextExecuteAt determines the next block height for execution.
func calculateNextExecuteAt(cond types.ScheduleCondition, currentHeight uint64) uint64 {
	if cond.TimeCondition != nil {
		tc := cond.TimeCondition
		switch tc.Type {
		case types.TimeAbsolute:
			return tc.ExecuteAtBlock
		case types.TimeRelative:
			return currentHeight + tc.ExecuteAtBlock
		case types.TimePeriodic:
			start := tc.StartBlock
			if start <= currentHeight {
				start = currentHeight + 1
			}
			return start
		}
	}
	// For logic-only conditions, check every block
	return currentHeight + 1
}
