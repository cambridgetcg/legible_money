package keeper

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// DefaultResearchTaxBps is the fallback research tax rate when the vesting
// rewards keeper is not wired. Matches the canonical 3.33% rate.
const DefaultResearchTaxBps = 33300

// GetResearchTaxBps returns the unified research tax rate in BPS (1,000,000 scale).
// Reads from the canonical source (vesting_rewards params) if available,
// otherwise uses the default fallback.
func (k Keeper) GetResearchTaxBps(ctx sdk.Context) uint64 {
	if k.vestingRewardsKeeper != nil {
		return k.vestingRewardsKeeper.GetResearchShareBps(ctx)
	}
	return DefaultResearchTaxBps
}

// RouteReward sends reward to addr, routing through partnership if one exists.
// If addr is in an active partnership, the reward is split per partnership economics
// (22% common pot, 78% split per partnership terms). If not, the full reward goes
// directly to addr. Falls back to direct transfer on any partnership error.
func (k Keeper) RouteReward(ctx sdk.Context, addr sdk.AccAddress, amount sdk.Coins, sourceType string) error {
	if k.partnershipsKeeper == nil {
		// No partnership module — direct transfer
		return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, amount)
	}

	// Module accounts (genesis submitter = "genesis") should not be looked up
	// in partnerships. AccAddressFromBech32 would have failed for "genesis" before
	// reaching here, but guard against module account addresses too.
	if k.accountKeeper != nil {
		acc := k.accountKeeper.GetAccount(ctx, addr)
		if acc != nil {
			// Check if it's a module account by seeing if it implements ModuleAccountI
			if _, isModule := acc.(sdk.ModuleAccountI); isModule {
				return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, amount)
			}
		}
	}

	partnership, found := k.partnershipsKeeper.GetActivePartnershipForAddress(ctx, addr.String())
	if !found {
		// No active partnership — direct transfer
		return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, amount)
	}

	// Route through partnership: 22% pot, 78% split
	grossAmount := amount.AmountOf("ulgm").String()
	humanShare, agentShare, _, err := k.partnershipsKeeper.DistributeReward(
		ctx, partnership, grossAmount, sourceType,
	)
	if err != nil {
		// Partnership distribution failed — fall back to direct
		k.Logger(ctx).Error("partnership reward routing failed, falling back to direct",
			"addr", addr.String(), "error", err)
		return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, amount)
	}

	// Send individual shares
	humanAddr, errH := sdk.AccAddressFromBech32(partnership.HumanAddr)
	agentAddr, errA := sdk.AccAddressFromBech32(partnership.AgentAddr)
	if errH != nil || errA != nil {
		k.Logger(ctx).Error("invalid partnership addresses, falling back to direct",
			"human", partnership.HumanAddr, "agent", partnership.AgentAddr)
		return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, amount)
	}

	if humanAmt, ok := sdkmath.NewIntFromString(humanShare); ok && humanAmt.IsPositive() {
		humanCoins := sdk.NewCoins(sdk.NewCoin("ulgm", humanAmt))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, humanAddr, humanCoins); err != nil {
			return err
		}
	}
	if agentAmt, ok := sdkmath.NewIntFromString(agentShare); ok && agentAmt.IsPositive() {
		agentCoins := sdk.NewCoins(sdk.NewCoin("ulgm", agentAmt))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, agentAddr, agentCoins); err != nil {
			return err
		}
	}

	// Common pot share stays in partnership module account (handled by DistributeReward)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"partnership_reward_routed",
		sdk.NewAttribute("partnership_id", partnership.Id),
		sdk.NewAttribute("source", sourceType),
		sdk.NewAttribute("gross", grossAmount),
		sdk.NewAttribute("human_share", humanShare),
		sdk.NewAttribute("agent_share", agentShare),
	))

	return nil
}
