package app

import (
	"context"
	"math/big"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	alignmenttypes "github.com/legible-money/legible/x/alignment/types"
	autopoiesiskeeper "github.com/legible-money/legible/x/autopoiesis/keeper"
	autopoiesistypes "github.com/legible-money/legible/x/autopoiesis/types"
	emergencykeeper "github.com/legible-money/legible/x/emergency/keeper"
	knowledgekeeper "github.com/legible-money/legible/x/knowledge/keeper"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
	ontologykeeper "github.com/legible-money/legible/x/ontology/keeper"
	legiblestakingkeeper "github.com/legible-money/legible/x/staking/keeper"
	vestingrewardskeeper "github.com/legible-money/legible/x/vesting_rewards/keeper"
)

// AlignmentAutopoiesisAdapter bridges the alignment module's AutopoiesisKeeper
// interface to the actual autopoiesis keeper. The alignment module uses this to
// read and write multiplier paths, and to observe autopoiesis internal state
// for the Harmony sensor.
type AlignmentAutopoiesisAdapter struct {
	keeper autopoiesiskeeper.Keeper
}

func NewAlignmentAutopoiesisAdapter(k autopoiesiskeeper.Keeper) *AlignmentAutopoiesisAdapter {
	return &AlignmentAutopoiesisAdapter{keeper: k}
}

var _ alignmenttypes.AutopoiesisKeeper = (*AlignmentAutopoiesisAdapter)(nil)

func (a *AlignmentAutopoiesisAdapter) GetMultiplier(ctx context.Context, path string) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetMultiplier(sdkCtx, path)
}

func (a *AlignmentAutopoiesisAdapter) SetMultiplier(ctx context.Context, entry alignmenttypes.MultiplierEntry) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	a.keeper.SetMultiplier(sdkCtx, autopoiesistypes.MultiplierEntry{
		Path:    entry.Path,
		Value:   entry.Value,
		Hormone: entry.Hormone,
		Epoch:   entry.Epoch,
	})
}

func (a *AlignmentAutopoiesisAdapter) GetMultiplierEntry(ctx context.Context, path string) *alignmenttypes.MultiplierEntry {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	entry := a.keeper.GetMultiplierEntry(sdkCtx, path)
	if entry == nil {
		return nil
	}
	return &alignmenttypes.MultiplierEntry{
		Path:    entry.Path,
		Value:   entry.Value,
		Hormone: entry.Hormone,
		Epoch:   entry.Epoch,
	}
}

func (a *AlignmentAutopoiesisAdapter) GetAllMultiplierEntries(ctx context.Context) []alignmenttypes.MultiplierEntry {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	entries := a.keeper.GetAllMultiplierEntries(sdkCtx)
	result := make([]alignmenttypes.MultiplierEntry, len(entries))
	for i, e := range entries {
		result[i] = alignmenttypes.MultiplierEntry{
			Path:    e.Path,
			Value:   e.Value,
			Hormone: e.Hormone,
			Epoch:   e.Epoch,
		}
	}
	return result
}

func (a *AlignmentAutopoiesisAdapter) GetAdaptationCount(ctx context.Context, epoch uint64) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetAdaptationCount(sdkCtx, epoch)
}

func (a *AlignmentAutopoiesisAdapter) GetConflictCount(ctx context.Context, epoch uint64) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetConflictCount(sdkCtx, epoch)
}

func (a *AlignmentAutopoiesisAdapter) GetDirectionChanges(ctx context.Context, lastNEpochs uint64) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetDirectionChanges(sdkCtx, lastNEpochs)
}

func (a *AlignmentAutopoiesisAdapter) IsActive(ctx context.Context) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.IsActive(sdkCtx)
}
