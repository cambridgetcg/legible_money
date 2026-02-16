package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/home/types"
)

// Keeper manages the home module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService
	bankKeeper   types.BankKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new home module Keeper.
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

// prefixEndBytes returns the end key for a prefix scan (exclusive upper bound).
// It increments the last byte of the prefix; if it overflows, it tries the
// previous byte, and so on. Returns nil if all bytes overflow (prefix is all 0xff).
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
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if gs.Params != nil {
		k.SetParams(ctx, *gs.Params)
	} else {
		k.SetParams(ctx, types.DefaultParams())
	}

	for _, home := range gs.Homes {
		if home != nil {
			k.SetHome(ctx, *home)
		}
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var homes []*types.AgentHome
	k.IterateHomes(ctx, func(home types.AgentHome) bool {
		homes = append(homes, &home)
		return false
	})

	// Export all alerts across homes
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.AlertKeyPrefix, prefixEndBytes(types.AlertKeyPrefix))
	if err == nil {
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			// Alerts are not included in genesis for simplicity (ephemeral state).
			// Include only homes and params.
		}
	}

	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params: &params,
		Homes:  homes,
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
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}
