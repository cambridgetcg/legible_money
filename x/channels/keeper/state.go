package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/channels/types"
)

// --- Channel CRUD ---

// SetChannel stores a payment channel and updates indexes.
func (k Keeper) SetChannel(ctx sdk.Context, ch types.PaymentChannel) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(ch)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal payment channel: %v", err))
	}
	store.Set(types.ChannelKey(ch.ChannelId), bz)

	// Update payer and provider indexes
	store.Set(types.PayerChannelKey(ch.Payer, ch.ChannelId), []byte(ch.ChannelId))
	store.Set(types.ProviderChannelKey(ch.Provider, ch.ChannelId), []byte(ch.ChannelId))
}

// GetChannel retrieves a payment channel by ID.
func (k Keeper) GetChannel(ctx sdk.Context, channelId string) (types.PaymentChannel, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ChannelKey(channelId))
	if bz == nil {
		return types.PaymentChannel{}, false
	}
	var ch types.PaymentChannel
	if err := json.Unmarshal(bz, &ch); err != nil {
		return types.PaymentChannel{}, false
	}
	return ch, true
}

// DeleteChannel removes a payment channel and its indexes.
func (k Keeper) DeleteChannel(ctx sdk.Context, ch types.PaymentChannel) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ChannelKey(ch.ChannelId))
	store.Delete(types.PayerChannelKey(ch.Payer, ch.ChannelId))
	store.Delete(types.ProviderChannelKey(ch.Provider, ch.ChannelId))
}

// IterateChannels iterates over all channels. Return true from cb to stop.
func (k Keeper) IterateChannels(ctx sdk.Context, cb func(types.PaymentChannel) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ChannelKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var ch types.PaymentChannel
		if err := json.Unmarshal(iter.Value(), &ch); err != nil {
			continue
		}
		if cb(ch) {
			break
		}
	}
}

// GetChannelsByPayer returns all channels for a payer via index.
func (k Keeper) GetChannelsByPayer(ctx sdk.Context, payer string) []types.PaymentChannel {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.PayerChannelPrefix(payer))
	defer iter.Close()

	var channels []types.PaymentChannel
	for ; iter.Valid(); iter.Next() {
		channelId := string(iter.Value())
		ch, found := k.GetChannel(ctx, channelId)
		if found {
			channels = append(channels, ch)
		}
	}
	return channels
}

// GetChannelsByProvider returns all channels for a provider via index.
func (k Keeper) GetChannelsByProvider(ctx sdk.Context, provider string) []types.PaymentChannel {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ProviderChannelPrefix(provider))
	defer iter.Close()

	var channels []types.PaymentChannel
	for ; iter.Valid(); iter.Next() {
		channelId := string(iter.Value())
		ch, found := k.GetChannel(ctx, channelId)
		if found {
			channels = append(channels, ch)
		}
	}
	return channels
}

// GetOpenChannelCountForPayer returns the number of open channels for a payer.
func (k Keeper) GetOpenChannelCountForPayer(ctx sdk.Context, payer string) uint64 {
	channels := k.GetChannelsByPayer(ctx, payer)
	var count uint64
	for _, ch := range channels {
		if ch.Status == types.ChannelStatusOpen {
			count++
		}
	}
	return count
}

// GetChannelsByStatus returns all channels with a given status.
func (k Keeper) GetChannelsByStatus(ctx sdk.Context, status types.ChannelStatus) []types.PaymentChannel {
	var channels []types.PaymentChannel
	k.IterateChannels(ctx, func(ch types.PaymentChannel) bool {
		if ch.Status == status {
			channels = append(channels, ch)
		}
		return false
	})
	return channels
}

// --- Dispute CRUD ---

// SetDispute stores a channel dispute.
func (k Keeper) SetDispute(ctx sdk.Context, d types.ChannelDispute) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(d)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal channel dispute: %v", err))
	}
	store.Set(types.DisputeKey(d.ChannelId), bz)
}

// GetDispute retrieves a dispute by channel ID.
func (k Keeper) GetDispute(ctx sdk.Context, channelId string) (types.ChannelDispute, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.DisputeKey(channelId))
	if bz == nil {
		return types.ChannelDispute{}, false
	}
	var d types.ChannelDispute
	if err := json.Unmarshal(bz, &d); err != nil {
		return types.ChannelDispute{}, false
	}
	return d, true
}

// DeleteDispute removes a dispute.
func (k Keeper) DeleteDispute(ctx sdk.Context, channelId string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.DisputeKey(channelId))
}

// IterateDisputes iterates over all disputes. Return true from cb to stop.
func (k Keeper) IterateDisputes(ctx sdk.Context, cb func(types.ChannelDispute) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.DisputeKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var d types.ChannelDispute
		if err := json.Unmarshal(iter.Value(), &d); err != nil {
			continue
		}
		if cb(d) {
			break
		}
	}
}

// GetExpiredDisputes returns disputes past their deadline.
func (k Keeper) GetExpiredDisputes(ctx sdk.Context, currentBlock uint64) []types.ChannelDispute {
	var disputes []types.ChannelDispute
	k.IterateDisputes(ctx, func(d types.ChannelDispute) bool {
		if !d.Resolved && d.DeadlineBlock > 0 && currentBlock > d.DeadlineBlock {
			disputes = append(disputes, d)
		}
		return false
	})
	return disputes
}

// --- Params ---

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
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
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	store.Set(types.ParamsKey, bz)
}

// --- Channel ID Counter ---

var channelCounterKey = []byte{0x20}

// GetNextChannelId returns a unique channel ID and increments the counter.
func (k Keeper) GetNextChannelId(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(channelCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(channelCounterKey, newBz)
	return counter
}
