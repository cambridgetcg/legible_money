package keeper

import (
	"fmt"

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
