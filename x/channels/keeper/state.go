package keeper

import (
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/channels/types"
)

// --- Channel CRUD ---

// SetChannel stores a payment channel and updates indexes.
func (k Keeper) SetChannel(ctx sdk.Context, ch *types.PaymentChannel) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(ch)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal payment channel: %v", err))
	}
	if err := store.Set(types.ChannelKey(ch.ChannelId), bz); err != nil {
		panic(fmt.Sprintf("failed to set payment channel: %v", err))
	}

	// Update payer and provider indexes
	if err := store.Set(types.PayerChannelKey(ch.Payer, ch.ChannelId), []byte(ch.ChannelId)); err != nil {
		panic(fmt.Sprintf("failed to set payer channel index: %v", err))
	}
	if err := store.Set(types.ProviderChannelKey(ch.Provider, ch.ChannelId), []byte(ch.ChannelId)); err != nil {
		panic(fmt.Sprintf("failed to set provider channel index: %v", err))
	}
}

// GetChannel retrieves a payment channel by ID.
func (k Keeper) GetChannel(ctx sdk.Context, channelId string) (*types.PaymentChannel, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ChannelKey(channelId))
	if err != nil || bz == nil {
		return nil, false
	}
	var ch types.PaymentChannel
	if err := proto.Unmarshal(bz, &ch); err != nil {
		return nil, false
	}
	return &ch, true
}

// DeleteChannel removes a payment channel and its indexes.
func (k Keeper) DeleteChannel(ctx sdk.Context, ch *types.PaymentChannel) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.ChannelKey(ch.ChannelId))
	_ = store.Delete(types.PayerChannelKey(ch.Payer, ch.ChannelId))
	_ = store.Delete(types.ProviderChannelKey(ch.Provider, ch.ChannelId))
}

// IterateChannels iterates over all channels. Return true from cb to stop.
func (k Keeper) IterateChannels(ctx sdk.Context, cb func(*types.PaymentChannel) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ChannelKeyPrefix, prefixEndBytes(types.ChannelKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var ch types.PaymentChannel
		if err := proto.Unmarshal(iter.Value(), &ch); err != nil {
			continue
		}
		if cb(&ch) {
			break
		}
	}
}

// GetChannelsByPayer returns all channels for a payer via index.
func (k Keeper) GetChannelsByPayer(ctx sdk.Context, payer string) []*types.PaymentChannel {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.PayerChannelPrefix(payer)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var channels []*types.PaymentChannel
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
func (k Keeper) GetChannelsByProvider(ctx sdk.Context, provider string) []*types.PaymentChannel {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.ProviderChannelPrefix(provider)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var channels []*types.PaymentChannel
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

// GetChannelsByStatus returns all channels with a given status string.
func (k Keeper) GetChannelsByStatus(ctx sdk.Context, status string) []*types.PaymentChannel {
	var channels []*types.PaymentChannel
	k.IterateChannels(ctx, func(ch *types.PaymentChannel) bool {
		if ch.Status == status {
			channels = append(channels, ch)
		}
		return false
	})
	return channels
}

// --- Dispute CRUD ---

// SetDispute stores a channel dispute.
func (k Keeper) SetDispute(ctx sdk.Context, d *types.ChannelDispute) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(d)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal channel dispute: %v", err))
	}
	if err := store.Set(types.DisputeKey(d.ChannelId), bz); err != nil {
		panic(fmt.Sprintf("failed to set channel dispute: %v", err))
	}
}

// GetDispute retrieves a dispute by channel ID.
func (k Keeper) GetDispute(ctx sdk.Context, channelId string) (*types.ChannelDispute, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.DisputeKey(channelId))
	if err != nil || bz == nil {
		return nil, false
	}
	var d types.ChannelDispute
	if err := proto.Unmarshal(bz, &d); err != nil {
		return nil, false
	}
	return &d, true
}

// DeleteDispute removes a dispute.
func (k Keeper) DeleteDispute(ctx sdk.Context, channelId string) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.DisputeKey(channelId))
}

// IterateDisputes iterates over all disputes. Return true from cb to stop.
func (k Keeper) IterateDisputes(ctx sdk.Context, cb func(*types.ChannelDispute) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.DisputeKeyPrefix, prefixEndBytes(types.DisputeKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var d types.ChannelDispute
		if err := proto.Unmarshal(iter.Value(), &d); err != nil {
			continue
		}
		if cb(&d) {
			break
		}
	}
}

// GetExpiredDisputes returns disputes past their deadline.
func (k Keeper) GetExpiredDisputes(ctx sdk.Context, currentBlock uint64) []*types.ChannelDispute {
	var disputes []*types.ChannelDispute
	k.IterateDisputes(ctx, func(d *types.ChannelDispute) bool {
		if !d.Resolved && d.DeadlineBlock > 0 && currentBlock > d.DeadlineBlock {
			disputes = append(disputes, d)
		}
		return false
	})
	return disputes
}

// --- Params ---

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
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

// --- Channel ID Counter ---

var channelCounterKey = []byte{0x20}

// GetNextChannelId returns a unique channel ID and increments the counter.
func (k Keeper) GetNextChannelId(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(channelCounterKey)
	var counter uint64
	if err == nil && bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	_ = store.Set(channelCounterKey, newBz)
	return counter
}
