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
		process.NextExecuteAt = calculateNextPeriodicExecution(process.Condition, currentHeight)
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
	// Non-periodic: check next block
	return currentHeight + 1
}
