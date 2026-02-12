package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/ibcratelimit/types"
)

// --- Flow CRUD ---

// SetFlow stores a channel flow record.
func (k Keeper) SetFlow(ctx sdk.Context, flow types.ChannelFlow) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(flow)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal channel flow: %v", err))
	}
	if err := store.Set(types.FlowKey(flow.ChannelID), bz); err != nil {
		panic(fmt.Sprintf("failed to set channel flow: %v", err))
	}
}

// GetFlow retrieves a channel flow record by channel ID.
func (k Keeper) GetFlow(ctx sdk.Context, channelID string) (types.ChannelFlow, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.FlowKey(channelID))
	if err != nil || bz == nil {
		return types.ChannelFlow{}, false
	}
	var flow types.ChannelFlow
	if err := json.Unmarshal(bz, &flow); err != nil {
		return types.ChannelFlow{}, false
	}
	return flow, true
}

// DeleteFlow removes a channel flow record.
func (k Keeper) DeleteFlow(ctx sdk.Context, channelID string) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.FlowKey(channelID))
}

// IterateFlows iterates over all flow records. Return true from cb to stop.
func (k Keeper) IterateFlows(ctx sdk.Context, cb func(types.ChannelFlow) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.FlowPrefix, prefixEndBytes(types.FlowPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var flow types.ChannelFlow
		if err := json.Unmarshal(iter.Value(), &flow); err != nil {
			continue
		}
		if cb(flow) {
			break
		}
	}
}

// --- Params ---

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// --- Epoch Management ---

// GetOrResetFlow retrieves a channel's flow record, resetting counters if the epoch has expired.
// If no flow record exists, a new one is initialized with zero counters.
func (k Keeper) GetOrResetFlow(ctx sdk.Context, channelID string) types.ChannelFlow {
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	flow, found := k.GetFlow(ctx, channelID)
	if !found {
		// Initialize new flow record
		flow = types.ChannelFlow{
			ChannelID:       channelID,
			Denom:           params.TrackedDenom,
			OutflowTotal:    "0",
			InflowTotal:     "0",
			EpochStartBlock: currentBlock,
		}
		k.SetFlow(ctx, flow)
		return flow
	}

	// Check if the epoch has expired and reset counters
	if currentBlock >= flow.EpochStartBlock+params.EpochLengthBlocks {
		flow.OutflowTotal = "0"
		flow.InflowTotal = "0"
		flow.EpochStartBlock = currentBlock
		k.SetFlow(ctx, flow)
	}

	return flow
}

// ResetExpiredFlows iterates all flows and resets those past their epoch boundary.
func (k Keeper) ResetExpiredFlows(ctx sdk.Context) {
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	k.IterateFlows(ctx, func(flow types.ChannelFlow) bool {
		if currentBlock >= flow.EpochStartBlock+params.EpochLengthBlocks {
			flow.OutflowTotal = "0"
			flow.InflowTotal = "0"
			flow.EpochStartBlock = currentBlock
			k.SetFlow(ctx, flow)
		}
		return false
	})
}
