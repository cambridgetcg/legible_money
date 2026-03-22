package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// Guardian maintenance task constants.
const (
	// GuardianMaintenanceTaskType matches schedule.TaskTypeGuardianMaintenance (1).
	GuardianMaintenanceTaskType uint32 = 1

	// GuardianMaintenanceInterval is one day in blocks (~86400s / 2.521s).
	GuardianMaintenanceInterval uint64 = 34272
)

// ScheduleGuardianMaintenance registers a recurring guardian maintenance task
// for the given home via the schedule module.
// Nil-safe: if scheduleKeeper is nil, this is a no-op (returns nil).
// Idempotent: if a task is already registered, cancels it first before re-registering.
func (k Keeper) ScheduleGuardianMaintenance(ctx sdk.Context, homeID string) error {
	if k.scheduleKeeper == nil {
		return nil
	}

	if homeID == "" {
		return fmt.Errorf("homeID cannot be empty")
	}

	// Verify home exists
	if _, found := k.GetHome(ctx, homeID); !found {
		return types.ErrHomeNotFound.Wrapf("home %s not found", homeID)
	}

	// If a task already exists, cancel it first (idempotent re-schedule)
	if existingID, exists := k.GetGuardianTaskID(ctx, homeID); exists {
		k.scheduleKeeper.CancelTask(ctx, homeID, existingID)
		k.deleteGuardianTaskID(ctx, homeID)
	}

	// Schedule the first execution one interval from now
	scheduledHeight := uint64(ctx.BlockHeight()) + GuardianMaintenanceInterval

	taskID, err := k.scheduleKeeper.RegisterTask(
		ctx,
		homeID,
		GuardianMaintenanceTaskType,
		scheduledHeight,
		GuardianMaintenanceInterval,
		nil, // no payload needed
	)
	if err != nil {
		return fmt.Errorf("failed to register guardian maintenance task: %w", err)
	}

	// Store the taskID under prefix 0x15
	k.setGuardianTaskID(ctx, homeID, taskID)

	return nil
}

// GetGuardianTaskID returns the guardian maintenance taskID for a home, if any.
func (k Keeper) GetGuardianTaskID(ctx sdk.Context, homeID string) (taskID string, exists bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.GuardianTaskKey(homeID))
	if err != nil || bz == nil {
		return "", false
	}
	return string(bz), true
}

// CancelGuardianMaintenance cancels the guardian maintenance task for a home.
// No-op if scheduleKeeper is nil or no task is registered.
func (k Keeper) CancelGuardianMaintenance(ctx sdk.Context, homeID string) {
	if k.scheduleKeeper == nil {
		return
	}

	taskID, exists := k.GetGuardianTaskID(ctx, homeID)
	if !exists {
		return
	}

	k.scheduleKeeper.CancelTask(ctx, homeID, taskID)
	k.deleteGuardianTaskID(ctx, homeID)
}

// setGuardianTaskID stores the guardian maintenance taskID for a home.
func (k Keeper) setGuardianTaskID(ctx sdk.Context, homeID, taskID string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Set(types.GuardianTaskKey(homeID), []byte(taskID))
}

// deleteGuardianTaskID removes the guardian maintenance taskID for a home.
func (k Keeper) deleteGuardianTaskID(ctx sdk.Context, homeID string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(types.GuardianTaskKey(homeID))
}
