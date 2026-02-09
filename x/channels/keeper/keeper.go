package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/channels/types"
)

// Keeper manages the channels module's state.
type Keeper struct {
	cdc        codec.Codec
	storeKey   storetypes.StoreKey
	bankKeeper types.BankKeeper
	authority  string
}

// NewKeeper creates a new channels module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bankKeeper,
		authority:  authority,
	}
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
	k.SetParams(ctx, gs.Params)
	for _, ch := range gs.Channels {
		k.SetChannel(ctx, ch)
	}
	for _, d := range gs.Disputes {
		k.SetDispute(ctx, d)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var channels []types.PaymentChannel
	k.IterateChannels(ctx, func(ch types.PaymentChannel) bool {
		channels = append(channels, ch)
		return false
	})

	var disputes []types.ChannelDispute
	k.IterateDisputes(ctx, func(d types.ChannelDispute) bool {
		disputes = append(disputes, d)
		return false
	})

	gs := &types.GenesisState{
		Params:   k.GetParams(ctx),
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
