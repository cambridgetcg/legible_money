package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/channels/types"
)

// Keeper manages the channels module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService
	bankKeeper   types.BankKeeper
	authority    string
}

// NewKeeper creates a new channels module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bankKeeper,
		authority:    authority,
	}
}

// prefixEndBytes returns the end key for prefix iteration.
// It increments the last byte of the prefix. If overflow, returns nil (iterate to end).
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	k.SetParams(ctx, *gs.Params)
	for _, ch := range gs.Channels {
		k.SetChannel(ctx, *ch)
	}
	for _, d := range gs.Disputes {
		k.SetDispute(ctx, *d)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var channels []*types.PaymentChannel
	k.IterateChannels(ctx, func(ch types.PaymentChannel) bool {
		c := ch
		channels = append(channels, &c)
		return false
	})

	var disputes []*types.ChannelDispute
	k.IterateDisputes(ctx, func(d types.ChannelDispute) bool {
		dd := d
		disputes = append(disputes, &dd)
		return false
	})

	params := k.GetParams(ctx)
	gs := &types.GenesisState{
		Params:   &params,
		Channels: channels,
		Disputes: disputes,
	}
	return gs
}

// ExportGenesisJSON exports genesis as raw JSON.
func (k Keeper) ExportGenesisJSON(ctx sdk.Context) json.RawMessage {
	gs := k.ExportGenesis(ctx)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal channels genesis: %v", err))
	}
	return bz
}
