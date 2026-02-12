package app

import (
	"context"

	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	lgmgovtypes "github.com/legible-money/legible/x/gov/types"
)

// LGMGovUpgradeAdapter bridges the LGM governance module's UpgradeKeeper
// interface to the real Cosmos SDK x/upgrade keeper. When a consensus-category
// LIP is accepted with an attached upgrade plan, the governance module calls
// ScheduleUpgrade through this adapter.
type LGMGovUpgradeAdapter struct {
	keeper *upgradekeeper.Keeper
}

// NewLGMGovUpgradeAdapter creates a new adapter wrapping the Cosmos SDK upgrade keeper.
func NewLGMGovUpgradeAdapter(k *upgradekeeper.Keeper) *LGMGovUpgradeAdapter {
	return &LGMGovUpgradeAdapter{keeper: k}
}

// ScheduleUpgrade converts the LGM gov UpgradePlan to a Cosmos SDK Plan
// and delegates to the real upgrade keeper.
func (a *LGMGovUpgradeAdapter) ScheduleUpgrade(ctx context.Context, plan lgmgovtypes.UpgradePlan) error {
	return a.keeper.ScheduleUpgrade(ctx, upgradetypes.Plan{
		Name:   plan.Name,
		Height: plan.Height,
		Info:   plan.Info,
	})
}

// Compile-time interface check.
var _ lgmgovtypes.UpgradeKeeper = (*LGMGovUpgradeAdapter)(nil)
