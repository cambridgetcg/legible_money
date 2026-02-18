package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/ibcratelimit/types"
)

// --- Flow CRUD ---

// SetFlow stores a channel flow record.
func (k Keeper) SetFlow(ctx sdk.Context, flow *types.ChannelFlow) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(flow)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal channel flow: %v", err))
	}
	if err := store.Set(types.FlowKey(flow.ChannelId), bz); err != nil {
		panic(fmt.Sprintf("failed to set channel flow: %v", err))
	}
}

// GetFlow retrieves a channel flow record by channel ID.
func (k Keeper) GetFlow(ctx sdk.Context, channelID string) (*types.ChannelFlow, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.FlowKey(channelID))
	if err != nil || bz == nil {
		return nil, false
	}
	var flow types.ChannelFlow
	if err := proto.Unmarshal(bz, &flow); err != nil {
		return nil, false
	}
	return &flow, true
}

// DeleteFlow removes a channel flow record.
func (k Keeper) DeleteFlow(ctx sdk.Context, channelID string) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.FlowKey(channelID))
}

// IterateFlows iterates over all flow records. Return true from cb to stop.
func (k Keeper) IterateFlows(ctx sdk.Context, cb func(*types.ChannelFlow) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.FlowPrefix, prefixEndBytes(types.FlowPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var flow types.ChannelFlow
		if err := proto.Unmarshal(iter.Value(), &flow); err != nil {
			continue
		}
		if cb(&flow) {
			break
		}
	}
}

// --- Previous Flow (for sliding window) ---

// SetPreviousFlow stores the previous epoch's flow record for a channel.
func (k Keeper) SetPreviousFlow(ctx sdk.Context, flow *types.ChannelFlow) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(flow)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal previous channel flow: %v", err))
	}
	if err := store.Set(types.PreviousFlowKey(flow.ChannelId), bz); err != nil {
		panic(fmt.Sprintf("failed to set previous channel flow: %v", err))
	}
}

// GetPreviousFlow retrieves the previous epoch's flow record for a channel.
func (k Keeper) GetPreviousFlow(ctx sdk.Context, channelID string) (*types.ChannelFlow, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.PreviousFlowKey(channelID))
	if err != nil || bz == nil {
		return nil, false
	}
	var flow types.ChannelFlow
	if err := proto.Unmarshal(bz, &flow); err != nil {
		return nil, false
	}
	return &flow, true
}

// DeletePreviousFlow removes the previous epoch's flow record for a channel.
func (k Keeper) DeletePreviousFlow(ctx sdk.Context, channelID string) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.PreviousFlowKey(channelID))
}

// GetEffectiveOutflow returns the effective outflow for a channel using the
// sliding window formula: effective = current + (1 - elapsed_fraction) * previous.
// This prevents the 2x capacity burst at epoch boundaries.
func (k Keeper) GetEffectiveOutflow(ctx sdk.Context, channelID string) *big.Int {
	return k.getEffectiveValue(ctx, channelID, true)
}

// GetEffectiveInflow returns the effective inflow for a channel using the
// sliding window formula.
func (k Keeper) GetEffectiveInflow(ctx sdk.Context, channelID string) *big.Int {
	return k.getEffectiveValue(ctx, channelID, false)
}

func (k Keeper) getEffectiveValue(ctx sdk.Context, channelID string, isOutflow bool) *big.Int {
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	flow, found := k.GetFlow(ctx, channelID)
	if !found {
		return new(big.Int)
	}

	var currentStr string
	if isOutflow {
		currentStr = flow.OutflowTotal
	} else {
		currentStr = flow.InflowTotal
	}
	current, ok := new(big.Int).SetString(currentStr, 10)
	if !ok {
		current = new(big.Int)
	}

	// Look up previous epoch flow for carry-over
	prevFlow, prevFound := k.GetPreviousFlow(ctx, channelID)
	if !prevFound {
		return current
	}

	var prevStr string
	if isOutflow {
		prevStr = prevFlow.OutflowTotal
	} else {
		prevStr = prevFlow.InflowTotal
	}
	prev, ok := new(big.Int).SetString(prevStr, 10)
	if !ok || prev.Sign() == 0 {
		return current
	}

	// Compute elapsed fraction within current epoch
	// elapsed_fraction = blocks_into_epoch / epoch_length
	// carry = (1 - elapsed_fraction) * previous = (epoch_length - elapsed) * previous / epoch_length
	epochLen := params.EpochLengthBlocks
	if epochLen == 0 {
		return current
	}
	elapsed := currentBlock - flow.EpochStartBlock
	if elapsed >= epochLen {
		// Full epoch has passed since reset — no carry-over from previous
		return current
	}

	remaining := epochLen - elapsed
	carry := new(big.Int).Mul(prev, new(big.Int).SetUint64(remaining))
	carry.Div(carry, new(big.Int).SetUint64(epochLen))

	return new(big.Int).Add(current, carry)
}

// --- Params ---

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		p := types.DefaultParams()
		return &p
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		p := types.DefaultParams()
		return &p
	}
	return &params
}

// SetParams sets the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
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
func (k Keeper) GetOrResetFlow(ctx sdk.Context, channelID string) *types.ChannelFlow {
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	flow, found := k.GetFlow(ctx, channelID)
	if !found {
		// Initialize new flow record
		flow = &types.ChannelFlow{
			ChannelId:       channelID,
			Denom:           params.TrackedDenom,
			OutflowTotal:    "0",
			InflowTotal:     "0",
			EpochStartBlock: currentBlock,
		}
		k.SetFlow(ctx, flow)
		return flow
	}

	// Check if the epoch has expired and reset counters.
	// Save the previous epoch's flow for the sliding window calculation
	// before resetting, so epoch boundaries don't create a 2x capacity burst.
	if currentBlock >= flow.EpochStartBlock+params.EpochLengthBlocks {
		epochsPassed := (currentBlock - flow.EpochStartBlock) / params.EpochLengthBlocks
		if epochsPassed == 1 {
			// Adjacent epoch — save previous flow for sliding window carry-over
			k.SetPreviousFlow(ctx, flow)
		} else {
			// Multiple epochs passed — previous flow is completely stale, no carry-over
			k.DeletePreviousFlow(ctx, flow.ChannelId)
		}
		flow.OutflowTotal = "0"
		flow.InflowTotal = "0"
		// Align to deterministic epoch boundary (not current block) so elapsed
		// time within the new epoch is calculated correctly for sliding window decay
		flow.EpochStartBlock = flow.EpochStartBlock + epochsPassed*params.EpochLengthBlocks
		k.SetFlow(ctx, flow)
	}

	return flow
}

// ResetExpiredFlows iterates all flows and resets those past their epoch boundary.
func (k Keeper) ResetExpiredFlows(ctx sdk.Context) {
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	k.IterateFlows(ctx, func(flow *types.ChannelFlow) bool {
		if currentBlock >= flow.EpochStartBlock+params.EpochLengthBlocks {
			epochsPassed := (currentBlock - flow.EpochStartBlock) / params.EpochLengthBlocks
			if epochsPassed == 1 {
				k.SetPreviousFlow(ctx, flow)
			} else {
				k.DeletePreviousFlow(ctx, flow.ChannelId)
			}
			flow.OutflowTotal = "0"
			flow.InflowTotal = "0"
			flow.EpochStartBlock = flow.EpochStartBlock + epochsPassed*params.EpochLengthBlocks
			k.SetFlow(ctx, flow)
		}
		return false
	})
}
