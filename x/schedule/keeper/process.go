package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/schedule/types"
)

// ProcessDueSchedules executes schedules that are due at or before the current block.
// Returns the number of schedules executed.
func (k Keeper) ProcessDueSchedules(ctx sdk.Context) int {
	currentHeight := uint64(ctx.BlockHeight())
	dueIds := k.GetDueProcesses(ctx, currentHeight)

	executed := 0
	for _, processId := range dueIds {
		process, found := k.GetProcess(ctx, processId)
		if !found {
			continue
		}
		if process.Status != types.StatusActive {
			// Clean up stale index entry
			k.RemoveTimeIndex(ctx, process.NextExecuteAt, process.Id)
			continue
		}

		// Check fee
		remaining := new(big.Int)
		remaining.SetString(process.RemainingFee, 10)
		feePerExec := new(big.Int)
		feePerExec.SetString(process.FeePerExecution, 10)

		if remaining.Cmp(feePerExec) < 0 {
			// Insufficient fee — mark as exhausted
			k.RemoveStatusIndex(ctx, string(process.Status), process.Id)
			k.RemoveTimeIndex(ctx, process.NextExecuteAt, process.Id)
			process.Status = types.StatusExhausted
			process.NextExecuteAt = 0
			k.SetProcess(ctx, process)

			ctx.EventManager().EmitEvents(sdk.Events{
				sdk.NewEvent("schedule_exhausted",
					sdk.NewAttribute("process_id", process.Id),
				),
			})
			continue
		}

		// Evaluate non-time conditions before executing.
		// Logic and compound conditions are checked each block but only fire when satisfied.
		if process.Condition != nil && !evaluateCondition(ctx, *process.Condition, k) {
			// Condition not met — re-index for next block check without deducting fee
			k.RemoveTimeIndex(ctx, process.NextExecuteAt, process.Id)
			process.NextExecuteAt = currentHeight + 1
			k.SetProcess(ctx, process)
			continue
		}

		// Execute: deduct fee
		remaining.Sub(remaining, feePerExec)

		// Remove old indexes before mutation
		k.RemoveStatusIndex(ctx, string(process.Status), process.Id)
		k.RemoveTimeIndex(ctx, process.NextExecuteAt, process.Id)

		process.RemainingFee = remaining.String()
		process.ExecutionCount++

		// Execute transfer if configured
		if process.TransferValue != "" && process.TargetAddress != "" {
			transferAmt := new(big.Int)
			if _, ok := transferAmt.SetString(process.TransferValue, 10); ok && transferAmt.Sign() > 0 {
				targetAddr, err := sdk.AccAddressFromBech32(process.TargetAddress)
				if err == nil {
					coin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(transferAmt))
					if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, targetAddr, sdk.NewCoins(coin)); err != nil {
						k.Logger(ctx).Error("schedule transfer failed",
							"process_id", process.Id,
							"error", err,
						)
					}
				}
			}
		}

		// Emit execution event
		ctx.EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent("schedule_executed",
				sdk.NewAttribute("process_id", process.Id),
				sdk.NewAttribute("execution_count", fmt.Sprintf("%d", process.ExecutionCount)),
				sdk.NewAttribute("block_height", fmt.Sprintf("%d", currentHeight)),
			),
		})

		// Check if completed
		if process.MaxExecutions > 0 && process.ExecutionCount >= process.MaxExecutions {
			process.Status = types.StatusCompleted
			process.NextExecuteAt = 0
			k.SetProcess(ctx, process)
			executed++
			continue
		}

		// Check if expired
		if process.ExpiresAtBlock > 0 && currentHeight >= process.ExpiresAtBlock {
			process.Status = types.StatusCompleted
			process.NextExecuteAt = 0
			k.SetProcess(ctx, process)
			executed++
			continue
		}

		// Schedule next execution
		if process.Condition != nil {
			process.NextExecuteAt = calculateNextPeriodicExecution(*process.Condition, currentHeight)
		}
		k.SetProcess(ctx, process)
		executed++
	}

	return executed
}

// CleanupExpiredProcesses removes completed/cancelled processes past retention period.
func (k Keeper) CleanupExpiredProcesses(ctx sdk.Context) int {
	params := k.GetParams(ctx)
	currentHeight := uint64(ctx.BlockHeight())
	cleaned := 0

	k.IterateProcesses(ctx, func(p types.ScheduleProcess) bool {
		if p.Status != types.StatusCompleted && p.Status != types.StatusCancelled {
			return false
		}
		// Use CreatedAtBlock + retention as a simple heuristic
		if currentHeight > p.CreatedAtBlock+params.ExpiredRetentionBlocks+p.ExecutionCount*100 {
			k.DeleteProcess(ctx, p)
			cleaned++
		}
		return false
	})

	return cleaned
}

// calculateNextPeriodicExecution computes the next fire time for periodic schedules.
func calculateNextPeriodicExecution(cond types.ScheduleCondition, currentHeight uint64) uint64 {
	if cond.TimeCondition != nil && cond.TimeCondition.Type == types.TimePeriodic {
		tc := cond.TimeCondition
		next := currentHeight + tc.IntervalBlocks
		if tc.EndBlock > 0 && next > tc.EndBlock {
			return 0 // past end
		}
		return next
	}
	// Logic/compound conditions: check every block
	return currentHeight + 1
}

// evaluateCondition checks whether a schedule's condition is currently satisfied.
// Time conditions are implicitly satisfied (they fire when due via the time index).
// Logic conditions compare on-chain state against a target value.
// Compound conditions combine sub-conditions with AND/OR.
func evaluateCondition(ctx sdk.Context, cond types.ScheduleCondition, k Keeper) bool {
	// Pure time conditions: always satisfied when due (time index handles scheduling)
	if cond.TimeCondition != nil && cond.LogicCondition == nil && cond.CompoundCondition == nil {
		return true
	}

	// Logic condition evaluation
	if cond.LogicCondition != nil {
		return evaluateLogicCondition(ctx, cond.LogicCondition, k)
	}

	// Compound condition evaluation
	if cond.CompoundCondition != nil {
		return evaluateCompoundCondition(ctx, cond.CompoundCondition, k)
	}

	// No condition set — default to satisfied
	return true
}

// evaluateLogicCondition reads the state key from the module's own store and compares it.
// StateKey format: raw KV store key (hex-encoded or plain string).
// Comparator: "eq", "neq", "gt", "gte", "lt", "lte".
// CompareValue: string to compare against (numeric comparison for gt/gte/lt/lte).
func evaluateLogicCondition(ctx sdk.Context, lc *types.LogicCondition, k Keeper) bool {
	if lc == nil || lc.StateKey == "" {
		return false
	}

	// Read the state key from the schedule module's own KV store
	store := k.storeService.OpenKVStore(ctx)
	val, err := store.Get([]byte(lc.StateKey))
	if err != nil || val == nil {
		return false // Key not found — condition not met
	}
	stateValue := string(val)

	switch lc.Comparator {
	case "eq":
		return stateValue == lc.CompareValue
	case "neq":
		return stateValue != lc.CompareValue
	case "gt", "gte", "lt", "lte":
		stateNum := new(big.Int)
		compareNum := new(big.Int)
		if _, ok := stateNum.SetString(stateValue, 10); !ok {
			return false
		}
		if _, ok := compareNum.SetString(lc.CompareValue, 10); !ok {
			return false
		}
		cmp := stateNum.Cmp(compareNum)
		switch lc.Comparator {
		case "gt":
			return cmp > 0
		case "gte":
			return cmp >= 0
		case "lt":
			return cmp < 0
		case "lte":
			return cmp <= 0
		}
	}

	return false
}

// evaluateCompoundCondition evaluates AND/OR combinations of sub-conditions.
func evaluateCompoundCondition(ctx sdk.Context, cc *types.CompoundCondition, k Keeper) bool {
	if cc == nil || len(cc.Conditions) == 0 {
		return false
	}

	switch cc.Operator {
	case types.CompoundAnd:
		for _, sub := range cc.Conditions {
			if !evaluateCondition(ctx, *sub, k) {
				return false
			}
		}
		return true
	case types.CompoundOr:
		for _, sub := range cc.Conditions {
			if evaluateCondition(ctx, *sub, k) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
