package app

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	lgmgovkeeper "github.com/legible-money/legible/x/gov/keeper"
	lgmgovtypes "github.com/legible-money/legible/x/gov/types"

	autopoiesistypes "github.com/legible-money/legible/x/autopoiesis/types"
)

// AutopoiesisGovAdapter bridges the autopoiesis module's GovKeeper interface
// to the LGM governance keeper. The autopoiesis module uses this to observe
// governance activity for auto-proposal rate limiting.
type AutopoiesisGovAdapter struct {
	keeper lgmgovkeeper.Keeper
}

// NewAutopoiesisGovAdapter creates a new adapter wrapping the LGM gov keeper.
func NewAutopoiesisGovAdapter(k lgmgovkeeper.Keeper) *AutopoiesisGovAdapter {
	return &AutopoiesisGovAdapter{keeper: k}
}

// GetActiveProposalCount returns the number of LIPs currently in voting stage.
func (a *AutopoiesisGovAdapter) GetActiveProposalCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	lips := a.keeper.GetLIPsByStage(sdkCtx, lgmgovtypes.StageVoting)
	return uint64(len(lips))
}

// Compile-time interface check.
var _ autopoiesistypes.GovKeeper = (*AutopoiesisGovAdapter)(nil)
