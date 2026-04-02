package keeper

import (
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

// MeterTotals holds cumulative token-flow counters for the consumption meter.
type MeterTotals struct {
	TotalMinted string `json:"total_minted"`
	TotalBurned string `json:"total_burned"`
	TotalLocked string `json:"total_locked"`
	TotalRouted string `json:"total_routed"`
	TotalFees   string `json:"total_fees"`
	BlockHeight int64  `json:"block_height"`
}

// incrementCounter is the shared helper that adds `amount` to the counter
// stored at `key`. The value is stored as a decimal string (same pattern as
// TotalMintedKey in keeper.go).
func (k Keeper) incrementCounter(ctx sdk.Context, key []byte, amount *big.Int) {
	if amount == nil || amount.Sign() <= 0 {
		return
	}
	store := k.storeService.OpenKVStore(ctx)
	current := new(big.Int)
	bz, err := store.Get(key)
	if err == nil && bz != nil {
		current.SetString(string(bz), 10)
	}
	current.Add(current, amount)
	if err := store.Set(key, []byte(current.String())); err != nil {
		panic("failed to set meter counter: " + err.Error())
	}
}

// getCounter reads a single cumulative counter from the store.
func (k Keeper) getCounter(ctx sdk.Context, key []byte) *big.Int {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return new(big.Int)
	}
	v := new(big.Int)
	if _, ok := v.SetString(string(bz), 10); !ok {
		return new(big.Int)
	}
	return v
}

// IncrementMinted records newly minted tokens.
func (k Keeper) IncrementMinted(ctx sdk.Context, amount sdkmath.Int) {
	k.incrementCounter(ctx, types.MeterMintedKey, amount.BigInt())
}

// IncrementBurned records burned tokens.
func (k Keeper) IncrementBurned(ctx sdk.Context, amount sdkmath.Int) {
	k.incrementCounter(ctx, types.MeterBurnedKey, amount.BigInt())
}

// IncrementLocked records tokens locked in vesting schedules.
func (k Keeper) IncrementLocked(ctx sdk.Context, amount sdkmath.Int) {
	k.incrementCounter(ctx, types.MeterLockedKey, amount.BigInt())
}

// IncrementRouted records tokens routed through research/pool splits.
func (k Keeper) IncrementRouted(ctx sdk.Context, amount sdkmath.Int) {
	k.incrementCounter(ctx, types.MeterRoutedKey, amount.BigInt())
}

// IncrementFees records transaction fees swept to the research fund.
func (k Keeper) IncrementFees(ctx sdk.Context, amount sdkmath.Int) {
	k.incrementCounter(ctx, types.MeterFeesKey, amount.BigInt())
}

// GetMeterTotals returns all cumulative token-flow counters plus the current
// block height. This provides the aggregated view required by WI-2.
func (k Keeper) GetMeterTotals(ctx sdk.Context) MeterTotals {
	return MeterTotals{
		TotalMinted: k.getCounter(ctx, types.MeterMintedKey).String(),
		TotalBurned: k.getCounter(ctx, types.MeterBurnedKey).String(),
		TotalLocked: k.getCounter(ctx, types.MeterLockedKey).String(),
		TotalRouted: k.getCounter(ctx, types.MeterRoutedKey).String(),
		TotalFees:   k.getCounter(ctx, types.MeterFeesKey).String(),
		BlockHeight: ctx.BlockHeight(),
	}
}
