package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tokens/types"
)

// Keeper manages the tokens module's state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey

	bankKeeper types.BankKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new tokens module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	bk types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bk,
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

// ---------- Params ----------

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	store.Set(types.ParamsKey, bz)
}

// GetParams returns module parameters.
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

// ---------- Period Helpers ----------

// GetCurrentPeriod returns the current period index based on block height and params.
func (k Keeper) GetCurrentPeriod(ctx sdk.Context) uint64 {
	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())
	if params.PeriodBlocks == 0 {
		return 0
	}
	return height / params.PeriodBlocks
}

// IsExpired returns true if tokens from the given period have expired.
func (k Keeper) IsExpired(period uint64, currentPeriod uint64, periodsUntilExpiry uint64) bool {
	if currentPeriod <= period {
		return false
	}
	return (currentPeriod - period) > periodsUntilExpiry
}

// PeriodToExpirationBlock returns the block height at which tokens from a period expire.
func (k Keeper) PeriodToExpirationBlock(period uint64, params types.Params) uint64 {
	return (period + params.PeriodsUntilExpiry + 1) * params.PeriodBlocks
}

// ---------- Genesis ----------

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	k.SetRAIAccount(ctx, genState.RAI)

	for _, acct := range genState.Accounts {
		k.SetExpirableAccount(ctx, acct)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	accounts := k.GetAllExpirableAccounts(ctx)
	rai := k.GetRAIAccount(ctx)

	return &types.GenesisState{
		Params:   k.GetParams(ctx),
		Accounts: accounts,
		RAI:      rai,
	}
}
