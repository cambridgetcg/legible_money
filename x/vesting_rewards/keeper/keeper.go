package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

// Keeper manages the vesting_rewards module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	// Dependency keepers
	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new vesting_rewards module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bk types.BankKeeper,
	sk types.StakingKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bk,
		stakingKeeper: sk,
		authority:    authority,
	}
}

// prefixEndBytes returns the end key for a prefix scan (exclusive).
// It increments the last byte of the prefix; if overflow occurs on all bytes, returns nil.
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

// GetStakingKeeper returns the staking keeper for external use.
func (k Keeper) GetStakingKeeper() types.StakingKeeper {
	return k.stakingKeeper
}

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil {
		return types.DefaultParams()
	}
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// SetParams sets the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// GetCategoryConfig returns the release curve config for a vesting category.
func (k Keeper) GetCategoryConfig(ctx sdk.Context, category types.VestingCategoryStr) (*types.CategoryConfig, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(types.CategoryConfigKeyPrefix, []byte(category)...)
	bz, err := store.Get(key)
	if err != nil {
		return nil, false
	}
	if bz == nil {
		// Fall back to defaults
		for _, cfg := range types.DefaultCategoryConfigs() {
			if cfg.Category == string(category) {
				return cfg, true
			}
		}
		return nil, false
	}
	var cfg types.CategoryConfig
	if err := json.Unmarshal(bz, &cfg); err != nil {
		return nil, false
	}
	return &cfg, true
}

// SetCategoryConfig stores a category config.
func (k Keeper) SetCategoryConfig(ctx sdk.Context, cfg *types.CategoryConfig) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(types.CategoryConfigKeyPrefix, []byte(cfg.Category)...)
	bz, err := json.Marshal(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal category config: %v", err))
	}
	if err := store.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set category config: %v", err))
	}
}

// GetFundBalance returns the current block reward fund balance.
func (k Keeper) GetFundBalance(ctx sdk.Context) *big.Int {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.BlockRewardFundBalanceKey)
	if err != nil || bz == nil {
		return new(big.Int)
	}
	balance := new(big.Int)
	if _, ok := balance.SetString(string(bz), 10); !ok {
		return new(big.Int)
	}
	return balance
}

// SetFundBalance stores the current block reward fund balance.
func (k Keeper) SetFundBalance(ctx sdk.Context, balance *big.Int) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.BlockRewardFundBalanceKey, []byte(balance.String())); err != nil {
		panic(fmt.Sprintf("failed to set fund balance: %v", err))
	}
}

// DeductFromFund atomically deducts an amount from the fund balance.
// Returns false if insufficient balance (fund exhausted).
func (k Keeper) DeductFromFund(ctx sdk.Context, amount *big.Int) bool {
	if amount.Sign() <= 0 {
		return true
	}
	balance := k.GetFundBalance(ctx)
	if balance.Cmp(amount) < 0 {
		return false
	}
	balance.Sub(balance, amount)
	k.SetFundBalance(ctx, balance)
	return true
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	if gs.Params != nil {
		k.SetParams(ctx, gs.Params)
	}

	for _, cfg := range gs.CategoryConfigs {
		if cfg != nil {
			k.SetCategoryConfig(ctx, cfg)
		}
	}

	for _, schedule := range gs.VestingSchedules {
		if schedule != nil {
			k.SetVestingSchedule(ctx, schedule)
		}
	}

	// Initialize fund balance and mint into module account
	if gs.Params != nil && gs.Params.InitialFundBalance != "" {
		initialFund := new(big.Int)
		if _, ok := initialFund.SetString(gs.Params.InitialFundBalance, 10); ok && initialFund.Sign() > 0 {
			k.SetFundBalance(ctx, initialFund)
			// Mint initial fund into module account so it can be distributed
			if k.bankKeeper != nil {
				mintCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(initialFund)))
				if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
					panic(fmt.Sprintf("failed to mint initial block reward fund: %v", err))
				}
			}
		}
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	// Overwrite InitialFundBalance with current balance so chain restart preserves remaining fund
	currentBalance := k.GetFundBalance(ctx)
	params.InitialFundBalance = currentBalance.String()
	return &types.GenesisState{
		Params:           params,
		CategoryConfigs:  k.GetAllCategoryConfigPtrs(ctx),
		VestingSchedules: k.GetAllActiveVestingSchedulePtrs(ctx),
	}
}

// GetAllCategoryConfigs returns all stored category configs as pointers, falling back to defaults.
func (k Keeper) GetAllCategoryConfigs(ctx sdk.Context) []*types.CategoryConfig {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.CategoryConfigKeyPrefix, prefixEndBytes(types.CategoryConfigKeyPrefix))
	if err != nil {
		return types.DefaultCategoryConfigs()
	}
	defer iter.Close()

	var configs []*types.CategoryConfig
	for ; iter.Valid(); iter.Next() {
		var cfg types.CategoryConfig
		if err := json.Unmarshal(iter.Value(), &cfg); err != nil {
			continue
		}
		configs = append(configs, &cfg)
	}

	// Fall back to defaults if store is empty
	if len(configs) == 0 {
		return types.DefaultCategoryConfigs()
	}
	return configs
}

// GetAllCategoryConfigPtrs returns all stored category configs as pointers for genesis export.
func (k Keeper) GetAllCategoryConfigPtrs(ctx sdk.Context) []*types.CategoryConfig {
	return k.GetAllCategoryConfigs(ctx)
}

// GetBlockRewardDistribution retrieves the block reward distribution for a specific height.
func (k Keeper) GetBlockRewardDistribution(ctx sdk.Context, blockHeight uint64) (*types.BlockRewardDistribution, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(types.BlockRewardKeyPrefix, sdk.Uint64ToBigEndian(blockHeight)...)
	bz, err := store.Get(key)
	if err != nil {
		return nil, false
	}
	if bz == nil {
		return nil, false
	}
	var dist types.BlockRewardDistribution
	if err := json.Unmarshal(bz, &dist); err != nil {
		return nil, false
	}
	return &dist, true
}
