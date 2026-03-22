package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/billing/types"
)

// ---------- Subscription Operations ----------

// SetSubscription stores a subscription record for a project.
func (k Keeper) SetSubscription(ctx sdk.Context, projectID string, sub *types.SubscriptionRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := subscriptionKey(projectID)
	bz, err := json.Marshal(sub)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal subscription: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetSubscription retrieves a subscription record for a project.
func (k Keeper) GetSubscription(ctx sdk.Context, projectID string) (*types.SubscriptionRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := subscriptionKey(projectID)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var sub types.SubscriptionRecord
	if err := json.Unmarshal(bz, &sub); err != nil {
		return nil, false
	}
	return &sub, true
}

// DeleteSubscription removes a subscription record.
func (k Keeper) DeleteSubscription(ctx sdk.Context, projectID string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(subscriptionKey(projectID))
}

// IterateSubscriptions iterates over all subscriptions. Return true from cb to stop.
func (k Keeper) IterateSubscriptions(ctx sdk.Context, cb func(*types.SubscriptionRecord) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.SubscriptionPrefix, prefixEndBytes(types.SubscriptionPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var sub types.SubscriptionRecord
		if err := json.Unmarshal(iter.Value(), &sub); err != nil {
			continue
		}
		if cb(&sub) {
			break
		}
	}
}

// ---------- Usage Counter Operations ----------

// IncrementUsage increments a resource counter for a project and returns
// (currentUsed, limit, allowed). allowed is false if the limit is exceeded.
func (k Keeper) IncrementUsage(ctx sdk.Context, projectID string, resource string) (uint64, int64, bool) {
	sub, found := k.GetSubscription(ctx, projectID)
	if !found {
		// No subscription = free plan defaults
		sub = &types.SubscriptionRecord{
			ProjectID: projectID,
			Plan:      types.PlanFree,
			Status:    types.StatusActive,
		}
	}

	// Cancelled/past_due subscriptions fall back to free limits
	plan := sub.Plan
	if sub.Status != types.StatusActive {
		plan = types.PlanFree
	}

	limits := types.DefaultPlanLimits()
	planLimit, ok := limits[plan]
	if !ok {
		planLimit = limits[types.PlanFree]
	}
	limit := planLimit.GetLimit(resource)

	usage := k.getUsageCounter(ctx, projectID)
	if usage.Counters == nil {
		usage.Counters = make(map[string]uint64)
	}

	current := usage.Counters[resource]

	// Check limit before incrementing (-1 = unlimited)
	if limit >= 0 && int64(current) >= limit {
		return current, limit, false
	}

	usage.Counters[resource] = current + 1
	k.setUsageCounter(ctx, projectID, usage)

	return current + 1, limit, true
}

// GetUsage returns the current usage counters for a project.
func (k Keeper) GetUsage(ctx sdk.Context, projectID string) map[string]uint64 {
	usage := k.getUsageCounter(ctx, projectID)
	if usage.Counters == nil {
		return make(map[string]uint64)
	}
	return usage.Counters
}

// ResetUsage clears all usage counters for a project.
func (k Keeper) ResetUsage(ctx sdk.Context, projectID string) {
	usage := &types.UsageCounter{
		ProjectID:   projectID,
		Counters:    make(map[string]uint64),
		PeriodStart: uint64(ctx.BlockHeight()),
	}
	k.setUsageCounter(ctx, projectID, usage)
}

// ResetAllActiveUsage resets usage counters for all active subscriptions.
// Called by BeginBlock at the start of each new day.
func (k Keeper) ResetAllActiveUsage(ctx sdk.Context) int {
	var toReset []string
	k.IterateSubscriptions(ctx, func(sub *types.SubscriptionRecord) bool {
		if sub.Status == types.StatusActive {
			toReset = append(toReset, sub.ProjectID)
		}
		return false
	})

	for _, pid := range toReset {
		k.ResetUsage(ctx, pid)
	}
	return len(toReset)
}

// ---------- Last Reset Block ----------

// SetLastResetBlock stores the block height of the last daily usage reset.
func (k Keeper) SetLastResetBlock(ctx sdk.Context, height uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, height)
	_ = kvStore.Set(types.LastResetBlockKey, bz)
}

// GetLastResetBlock returns the block height of the last daily usage reset.
func (k Keeper) GetLastResetBlock(ctx sdk.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.LastResetBlockKey)
	if err != nil || bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// ---------- Internal helpers ----------

func (k Keeper) getUsageCounter(ctx sdk.Context, projectID string) *types.UsageCounter {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := usageKey(projectID)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return &types.UsageCounter{
			ProjectID: projectID,
			Counters:  make(map[string]uint64),
		}
	}
	var usage types.UsageCounter
	if err := json.Unmarshal(bz, &usage); err != nil {
		return &types.UsageCounter{
			ProjectID: projectID,
			Counters:  make(map[string]uint64),
		}
	}
	return &usage
}

func (k Keeper) setUsageCounter(ctx sdk.Context, projectID string, usage *types.UsageCounter) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := usageKey(projectID)
	bz, err := json.Marshal(usage)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal usage counter: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// ---------- Key Construction ----------

func subscriptionKey(projectID string) []byte {
	return append(types.SubscriptionPrefix, []byte(projectID)...)
}

func usageKey(projectID string) []byte {
	return append(types.UsagePrefix, []byte(projectID)...)
}
