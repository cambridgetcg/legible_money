package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// Keeper manages the home module's state.
type Keeper struct {
	cdc          codec.Codec
	storeKey     storetypes.StoreKey
	bankKeeper   types.BankKeeper
	tokensKeeper types.TokensKeeper // nil-safe

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new home module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	tokensKeeper types.TokensKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeKey:     storeKey,
		bankKeeper:   bankKeeper,
		tokensKeeper: tokensKeeper,
		authority:    authority,
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
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	k.SetParams(ctx, gs.Params)

	for _, home := range gs.Homes {
		k.SetHome(ctx, home)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var homes []types.AgentHome
	k.IterateHomes(ctx, func(home types.AgentHome) bool {
		homes = append(homes, home)
		return false
	})

	// Export all alerts across homes
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.AlertKeyPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		// Alerts are not included in genesis for simplicity (ephemeral state).
		// Include only homes and params.
	}

	return &types.GenesisState{
		Params: k.GetParams(ctx),
		Homes:  homes,
	}
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
