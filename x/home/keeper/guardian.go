package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// CheckDeadmanSwitches scans all homes and triggers deadman for inactive agents.
// Called from BeginBlocker.
func (k Keeper) CheckDeadmanSwitches(ctx sdk.Context) {
	height := uint64(ctx.BlockHeight())
	k.IterateHomes(ctx, func(home types.AgentHome) bool {
		if home.Guardian == nil || home.Guardian.Deadman == nil || !home.Guardian.Deadman.Enabled {
			return false
		}
		if home.Status == string(types.HomeStatusArchived) {
			return false
		}
		threshold := home.Guardian.Deadman.InactivityThreshold
		if height > home.LastActiveBlock+threshold {
			k.triggerDeadman(ctx, home)
		}
		return false
	})
}

// triggerDeadman executes the configured deadman action.
func (k Keeper) triggerDeadman(ctx sdk.Context, home types.AgentHome) {
	threshold := uint64(0)
	action := ""
	if home.Guardian != nil && home.Guardian.Deadman != nil {
		threshold = home.Guardian.Deadman.InactivityThreshold
		action = home.Guardian.Deadman.Action
	}

	// Create critical alert
	k.CreateAlert(ctx, home.HomeId, types.AlertDeadmanTriggered, types.AlertPriorityCritical,
		fmt.Sprintf("Deadman switch triggered: inactive for %d blocks", threshold),
		"")

	// Set home to guarded status
	home.Status = string(types.HomeStatusGuarded)
	k.SetHome(ctx, home)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"deadman_triggered",
		sdk.NewAttribute("home_id", home.HomeId),
		sdk.NewAttribute("owner_did", home.OwnerDid),
		sdk.NewAttribute("action", action),
	))

	// Phase 2: transfer_to_beneficiary and donate_to_research actions
}

// CleanupExpiredSessions removes sessions past their expiration.
// Called from BeginBlocker.
func (k Keeper) CleanupExpiredSessions(ctx sdk.Context) {
	height := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)

	k.IterateHomes(ctx, func(home types.AgentHome) bool {
		k.IterateSessions(ctx, home.HomeId, func(session types.ActiveSession) bool {
			maxExpiry := session.StartedAt + params.SessionMaxDuration
			if height > session.ExpiresAt || height > maxExpiry {
				k.DeleteSession(ctx, home.HomeId, session.SessionId)

				// Create alert for expired session
				k.CreateAlert(ctx, home.HomeId, types.AlertSessionExpired, types.AlertPriorityLow,
					fmt.Sprintf("Session %s expired", session.SessionId), "")
			}
			return false
		})
		return false
	})
}

// CleanupOldAlerts removes acknowledged alerts past retention period.
// Called from EndBlocker.
func (k Keeper) CleanupOldAlerts(ctx sdk.Context) {
	height := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)

	k.IterateHomes(ctx, func(home types.AgentHome) bool {
		k.IterateAlerts(ctx, home.HomeId, func(alert types.Alert) bool {
			if alert.Acknowledged && height > alert.CreatedAt+params.AlertRetentionBlocks {
				k.DeleteAlert(ctx, home.HomeId, alert.AlertId)
			}
			return false
		})
		return false
	})
}

// --- Spending Limit Enforcement ---

// CheckAndDebitSpendingLimit checks if a spend of `amount` is within the limit
// for the given keyType on the given home. If allowed, it debits the amount
// from the limit's spent-in-period counter. Returns nil if allowed, error if exceeded.
// If no spending limit is configured for the keyType, the spend is allowed (no limit).
func (k Keeper) CheckAndDebitSpendingLimit(ctx sdk.Context, homeId, keyType, amount string) error {
	limit, found := k.GetSpendingLimit(ctx, homeId, keyType)
	if !found {
		return nil // No limit configured — allow
	}

	height := uint64(ctx.BlockHeight())

	// Reset period if elapsed
	if height >= limit.PeriodStart+limit.PeriodBlocks {
		limit.SpentInPeriod = "0"
		limit.PeriodStart = height
	}

	// Parse amounts
	spendAmount := new(big.Int)
	if _, ok := spendAmount.SetString(amount, 10); !ok || spendAmount.Sign() <= 0 {
		return fmt.Errorf("invalid spend amount: %s", amount)
	}

	spent := new(big.Int)
	spent.SetString(limit.SpentInPeriod, 10)

	maxAmount := new(big.Int)
	maxAmount.SetString(limit.MaxAmount, 10)

	// Check: spent + amount <= max
	newSpent := new(big.Int).Add(spent, spendAmount)
	if newSpent.Cmp(maxAmount) > 0 {
		// Create alert and reject
		k.CreateAlert(ctx, homeId, types.AlertSpendingLimitHit, types.AlertPriorityHigh,
			fmt.Sprintf("Spending limit exceeded for key type %s: tried %s, limit %s, already spent %s",
				keyType, amount, limit.MaxAmount, limit.SpentInPeriod), "")
		return types.ErrSpendingLimitExceeded.Wrapf(
			"key type %s: spent %s + %s exceeds limit %s",
			keyType, limit.SpentInPeriod, amount, limit.MaxAmount)
	}

	// Debit
	limit.SpentInPeriod = newSpent.String()
	k.SetSpendingLimit(ctx, homeId, limit)
	return nil
}

// GetRemainingSpendingLimit returns how much can still be spent in the current period.
// Returns -1 (as string) if no limit is configured.
func (k Keeper) GetRemainingSpendingLimit(ctx sdk.Context, homeId, keyType string) string {
	limit, found := k.GetSpendingLimit(ctx, homeId, keyType)
	if !found {
		return "-1" // No limit
	}

	height := uint64(ctx.BlockHeight())
	if height >= limit.PeriodStart+limit.PeriodBlocks {
		return limit.MaxAmount // Period reset, full limit available
	}

	maxAmount := new(big.Int)
	maxAmount.SetString(limit.MaxAmount, 10)
	spent := new(big.Int)
	spent.SetString(limit.SpentInPeriod, 10)

	remaining := new(big.Int).Sub(maxAmount, spent)
	if remaining.Sign() < 0 {
		remaining.SetInt64(0)
	}
	return remaining.String()
}

// --- Treasury Automation ---

// ProcessTreasuryAutomation runs automated treasury actions for homes with automation enabled.
// Called from BeginBlocker.
func (k Keeper) ProcessTreasuryAutomation(ctx sdk.Context) {
	k.IterateHomes(ctx, func(home types.AgentHome) bool {
		if home.Status != string(types.HomeStatusActive) {
			return false
		}
		if home.Treasury == nil || home.Treasury.Automation == nil {
			return false
		}
		auto := home.Treasury.Automation

		// Auto-compound rewards: if enabled, compound any pending rewards into stake
		if auto.AutoCompoundRewards {
			k.autoCompoundRewards(ctx, home)
		}

		// Min liquid balance check: alert if balance drops below threshold
		if auto.MinLiquidBalance != "" && auto.MinLiquidBalance != "0" {
			k.checkMinLiquidBalance(ctx, home)
		}

		return false
	})
}

// autoCompoundRewards compounds pending rewards by emitting an event.
// Actual reward claiming requires x/staking integration.
func (k Keeper) autoCompoundRewards(ctx sdk.Context, home types.AgentHome) {
	// Emit event signaling auto-compound intent.
	// The staking module's BeginBlocker handles actual reward distribution.
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"treasury_auto_compound",
		sdk.NewAttribute("home_id", home.HomeId),
		sdk.NewAttribute("owner", home.OwnerAddress),
	))
}

// checkMinLiquidBalance alerts if the home owner's balance drops below the configured minimum.
func (k Keeper) checkMinLiquidBalance(ctx sdk.Context, home types.AgentHome) {
	if k.bankKeeper == nil {
		return
	}
	ownerAddr, err := sdk.AccAddressFromBech32(home.OwnerAddress)
	if err != nil {
		return
	}

	balance := k.bankKeeper.GetBalance(ctx, ownerAddr, "ulgm")

	minBalance := new(big.Int)
	minBalance.SetString(home.Treasury.Automation.MinLiquidBalance, 10)

	balanceInt := balance.Amount.BigInt()
	if balanceInt.Cmp(minBalance) < 0 {
		k.CreateAlert(ctx, home.HomeId, types.AlertTokenExpiring, types.AlertPriorityMedium,
			fmt.Sprintf("Balance %s below minimum %s", balance.Amount.String(), home.Treasury.Automation.MinLiquidBalance),
			"")
	}
}

// CreateAlert creates a new alert for a home.
func (k Keeper) CreateAlert(ctx sdk.Context, homeId string, alertType types.AlertType, priority types.AlertPriority, message, data string) {
	params := k.GetParams(ctx)
	if k.CountPendingAlerts(ctx, homeId) >= params.MaxAlertsPerHome {
		return // Silently drop if at max alerts
	}

	alertId := fmt.Sprintf("alert-%d-%d", ctx.BlockHeight(), ctx.BlockTime().UnixNano())

	alert := types.Alert{
		AlertId:      alertId,
		HomeId:       homeId,
		Type:         string(alertType),
		Priority:     string(priority),
		Message:      message,
		Data:         data,
		CreatedAt:    uint64(ctx.BlockHeight()),
		Acknowledged: false,
	}
	k.SetAlert(ctx, alert)
}
