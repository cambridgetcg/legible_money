package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

// Keeper manages the vesting_rewards module's state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new vesting_rewards module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	authority string,
) Keeper {
	return Keeper{
		cdc:       cdc,
		storeKey:  storeKey,
		authority: authority,
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

// GetCategoryConfig returns the release curve config for a vesting category.
func (k Keeper) GetCategoryConfig(ctx sdk.Context, category types.VestingCategory) (types.CategoryConfig, bool) {
	store := ctx.KVStore(k.storeKey)
	key := append(types.CategoryConfigKeyPrefix, []byte(category)...)
	bz := store.Get(key)
	if bz == nil {
		// Fall back to defaults
		for _, cfg := range types.DefaultCategoryConfigs() {
			if cfg.Category == category {
				return cfg, true
			}
		}
		return types.CategoryConfig{}, false
	}
	var cfg types.CategoryConfig
	if err := json.Unmarshal(bz, &cfg); err != nil {
		return types.CategoryConfig{}, false
	}
	return cfg, true
}

// SetCategoryConfig stores a category config.
func (k Keeper) SetCategoryConfig(ctx sdk.Context, cfg types.CategoryConfig) {
	store := ctx.KVStore(k.storeKey)
	key := append(types.CategoryConfigKeyPrefix, []byte(cfg.Category)...)
	bz, err := json.Marshal(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal category config: %v", err))
	}
	store.Set(key, bz)
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	k.SetParams(ctx, gs.Params)

	for _, cfg := range gs.CategoryConfigs {
		k.SetCategoryConfig(ctx, cfg)
	}

	for _, schedule := range gs.VestingSchedules {
		k.SetVestingSchedule(ctx, schedule)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	return types.GenesisState{
		Params:           k.GetParams(ctx),
		CategoryConfigs:  types.DefaultCategoryConfigs(), // TODO: iterate store
		VestingSchedules: k.GetAllActiveVestingSchedules(ctx),
	}
}
