package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// CheckDeadmanSwitches scans all homes and triggers deadman for inactive agents.
// Called from BeginBlocker.
func (k Keeper) CheckDeadmanSwitches(ctx sdk.Context) {
	height := uint64(ctx.BlockHeight())
	k.IterateHomes(ctx, func(home *types.AgentHome) bool {
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
func (k Keeper) triggerDeadman(ctx sdk.Context, home *types.AgentHome) {
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

	k.IterateHomes(ctx, func(home *types.AgentHome) bool {
		k.IterateSessions(ctx, home.HomeId, func(session *types.ActiveSession) bool {
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

	k.IterateHomes(ctx, func(home *types.AgentHome) bool {
		k.IterateAlerts(ctx, home.HomeId, func(alert *types.Alert) bool {
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
//
// If the home is linked to a partnership, large/major amounts are blocked and
// must go through partnership deliberation instead.
func (k Keeper) CheckAndDebitSpendingLimit(ctx sdk.Context, homeId, keyType, amount string) error {
	// Check partnership deliberation tier for linked homes
	if k.partnershipKeeper != nil {
		home, found := k.GetHome(ctx, homeId)
		if found && home.PartnershipId != "" {
			if p, pFound := k.partnershipKeeper.GetPartnership(ctx, home.PartnershipId); pFound {
				tier := k.partnershipKeeper.CalculateAmountTier(amount, p.CommonPotBalance)
				if tier == "large" || tier == "major" {
					return fmt.Errorf("amount %s requires partnership deliberation (tier: %s)", amount, tier)
				}
			}
		}
	}

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
	k.IterateHomes(ctx, func(home *types.AgentHome) bool {
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
func (k Keeper) autoCompoundRewards(ctx sdk.Context, home *types.AgentHome) {
	// Emit event signaling auto-compound intent.
	// The staking module's BeginBlocker handles actual reward distribution.
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"treasury_auto_compound",
		sdk.NewAttribute("home_id", home.HomeId),
		sdk.NewAttribute("owner", home.OwnerAddress),
	))
}

// checkMinLiquidBalance alerts if the home owner's balance drops below the configured minimum.
func (k Keeper) checkMinLiquidBalance(ctx sdk.Context, home *types.AgentHome) {
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
	pendingCount := k.CountPendingAlerts(ctx, homeId)
	if pendingCount >= params.MaxAlertsPerHome {
		return // Silently drop if at max alerts
	}

	// Use homeId + blockHeight + pendingCount for unique alert IDs within a block.
	// Previous formula used BlockTime().UnixNano() which collides for multiple alerts
	// created in the same block (P1 fix).
	alertId := fmt.Sprintf("alert-%s-%d-%d", homeId, ctx.BlockHeight(), pendingCount)

	alert := &types.Alert{
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

// --- Defense Strategy Enactment ---

// defenseRules defines which enforcement mechanisms are active for a strategy.
type defenseRules struct {
	RateLimitPerWindow uint64
	RateWindowBlocks   uint64
	CooldownBlocks     uint64
	Require2FA         bool
	UseWhitelist       bool
	UseBlacklist       bool
}

func defenseRulesForStrategy(strategy types.DefenseStrategy) defenseRules {
	switch strategy {
	case types.DefenseAggressive:
		return defenseRules{
			RateLimitPerWindow: 5,
			RateWindowBlocks:   100,
			CooldownBlocks:     10,
			Require2FA:         true,
			UseBlacklist:       true,
		}
	case types.DefenseModerate:
		return defenseRules{
			RateLimitPerWindow: 20,
			RateWindowBlocks:   100,
			CooldownBlocks:     3,
			UseBlacklist:       true,
		}
	case types.DefenseConservative:
		return defenseRules{
			RateLimitPerWindow: 10,
			RateWindowBlocks:   100,
			CooldownBlocks:     5,
			UseWhitelist:       true,
			UseBlacklist:       true,
		}
	case types.DefenseDiplomatic:
		return defenseRules{
			UseBlacklist: true,
		}
	default:
		return defenseRules{}
	}
}

// EnforceDefenseStrategy checks that the operation is allowed under the home's
// configured defense strategy. Returns nil if allowed.
//
// Enforcement mechanisms by strategy:
//   - aggressive: rate_limit(5/100blk), cooldown(10blk), require_2fa, blacklist
//   - moderate:   rate_limit(20/100blk), cooldown(3blk), blacklist
//   - conservative: rate_limit(10/100blk), cooldown(5blk), whitelist, blacklist
//   - diplomatic: blacklist only
func (k Keeper) EnforceDefenseStrategy(ctx sdk.Context, homeID, keyID, operationType string, metadata map[string]string) error {
	home, found := k.GetHome(ctx, homeID)
	if !found {
		return types.ErrHomeNotFound
	}
	if home.Guardian == nil || home.Guardian.DefenseStrategy == "" {
		return nil // No guardian configured — allow
	}

	strategy := types.DefenseStrategy(home.Guardian.DefenseStrategy)
	rules := defenseRulesForStrategy(strategy)

	height := uint64(ctx.BlockHeight())

	// --- Stateless checks first (no side-effects on failure) ---

	// 2FA check
	if rules.Require2FA {
		if metadata == nil || metadata["2fa_verified"] != "true" {
			return types.ErrDefense2FARequired
		}
	}

	// Address-based checks
	targetAddr := ""
	if metadata != nil {
		targetAddr = metadata["target_address"]
	}

	// Blacklist check
	if rules.UseBlacklist && targetAddr != "" {
		blacklist := k.GetDefenseList(ctx, homeID, "blacklist")
		for _, addr := range blacklist {
			if addr == targetAddr {
				return types.ErrDefenseBlacklisted.Wrapf("address %s", targetAddr)
			}
		}
	}

	// Whitelist check (only when strategy uses whitelist AND list is non-empty)
	if rules.UseWhitelist && targetAddr != "" {
		whitelist := k.GetDefenseList(ctx, homeID, "whitelist")
		if len(whitelist) > 0 {
			allowed := false
			for _, addr := range whitelist {
				if addr == targetAddr {
					allowed = true
					break
				}
			}
			if !allowed {
				return types.ErrDefenseNotWhitelisted.Wrapf("address %s", targetAddr)
			}
		}
	}

	// --- Stateful checks (increment counters / update timestamps) ---

	// Rate limit check
	if rules.RateLimitPerWindow > 0 && rules.RateWindowBlocks > 0 {
		if err := k.checkAndIncrementRate(ctx, homeID, keyID, height, rules); err != nil {
			return err
		}
	}

	// Cooldown check
	if rules.CooldownBlocks > 0 {
		if err := k.checkAndSetCooldown(ctx, homeID, keyID, height, rules.CooldownBlocks); err != nil {
			return err
		}
	}

	return nil
}

// checkAndIncrementRate increments the rate counter for the current window and
// returns ErrDefenseRateLimited if the limit is exceeded.
func (k Keeper) checkAndIncrementRate(ctx sdk.Context, homeID, keyID string, height uint64, rules defenseRules) error {
	kvStore := k.storeService.OpenKVStore(ctx)
	window := height / rules.RateWindowBlocks
	key := types.DefenseRateKey(homeID, keyID, window)

	var count uint64
	bz, err := kvStore.Get(key)
	if err == nil && bz != nil && len(bz) == 8 {
		count = binary.BigEndian.Uint64(bz)
	}

	if count >= rules.RateLimitPerWindow {
		return types.ErrDefenseRateLimited.Wrapf(
			"key %s: %d ops in window %d (limit %d)", keyID, count, window, rules.RateLimitPerWindow)
	}

	count++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, count)
	_ = kvStore.Set(key, newBz)
	return nil
}

// checkAndSetCooldown checks if the key is in a cooldown period. If not, it
// records the current block as the last operation time.
func (k Keeper) checkAndSetCooldown(ctx sdk.Context, homeID, keyID string, currentBlock, cooldownBlocks uint64) error {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := types.DefenseCooldownKey(homeID, keyID)

	bz, err := kvStore.Get(key)
	if err == nil && bz != nil && len(bz) == 8 {
		lastOp := binary.BigEndian.Uint64(bz)
		if currentBlock < lastOp+cooldownBlocks {
			return types.ErrDefenseCooldown.Wrapf(
				"key %s: %d blocks remaining", keyID, lastOp+cooldownBlocks-currentBlock)
		}
	}

	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, currentBlock)
	_ = kvStore.Set(key, newBz)
	return nil
}

// GetDefenseList returns the whitelist or blacklist addresses for a home.
func (k Keeper) GetDefenseList(ctx sdk.Context, homeID, listType string) []string {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.DefenseListKey(homeID, listType))
	if err != nil || bz == nil {
		return nil
	}
	var addresses []string
	if err := json.Unmarshal(bz, &addresses); err != nil {
		return nil
	}
	return addresses
}

// SetDefenseList stores a whitelist or blacklist for a home.
func (k Keeper) SetDefenseList(ctx sdk.Context, homeID, listType string, addresses []string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, _ := json.Marshal(addresses)
	_ = kvStore.Set(types.DefenseListKey(homeID, listType), bz)
}
