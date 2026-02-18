package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/compute_pool/types"
)

// RegisterProvider handles new provider registration.
func (k Keeper) RegisterProvider(ctx sdk.Context, msg *types.MsgRegisterComputeProvider) error {
	params := k.GetParams(ctx)

	// Check if already registered
	if _, exists := k.GetProvider(ctx, msg.Sender); exists {
		return types.ErrProviderExists
	}

	// Validate stake meets minimum
	stakeAmt := new(big.Int)
	stakeAmt.SetString(msg.StakeAmount, 10)
	minStake := new(big.Int)
	minStake.SetString(params.MinProviderStake, 10)
	if stakeAmt.Cmp(minStake) < 0 {
		return fmt.Errorf("%w: need %s, got %s", types.ErrInsufficientStake, params.MinProviderStake, msg.StakeAmount)
	}

	// Validate price doesn't exceed ceiling
	price := new(big.Int)
	price.SetString(msg.PricePerCu, 10)
	maxPrice := new(big.Int)
	maxPrice.SetString(params.MaxPricePerCu, 10)
	if price.Cmp(maxPrice) > 0 {
		return fmt.Errorf("%w: max %s, got %s", types.ErrPriceExceedsCeiling, params.MaxPricePerCu, msg.PricePerCu)
	}

	// Transfer stake to module account
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}

	stakeCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmt)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, stakeCoins); err != nil {
		return fmt.Errorf("failed to transfer stake: %w", err)
	}

	height := uint64(ctx.BlockHeight())

	provider := types.ComputeProvider{
		Address:       msg.Sender,
		Did:           msg.Did,
		ServiceType:   msg.ServiceType,
		Endpoint:      msg.Endpoint,
		PricePerCu:    msg.PricePerCu,
		Stake:         msg.StakeAmount,
		Status:        string(types.ProviderStatusActive),
		RegisteredAt:  height,
		LastHeartbeat: height,
		UptimeBps:     1000000, // start at 100%
	}

	k.SetProvider(ctx, provider)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_provider_registered",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("service_type", msg.ServiceType),
			sdk.NewAttribute("stake", msg.StakeAmount),
			sdk.NewAttribute("price_per_cu", msg.PricePerCu),
		),
	)

	return nil
}

// UpdateProviderState handles provider updates (endpoint, price, pause/unpause).
func (k Keeper) UpdateProviderState(ctx sdk.Context, msg *types.MsgUpdateProvider) error {
	params := k.GetParams(ctx)

	provider, found := k.GetProvider(ctx, msg.Sender)
	if !found {
		return types.ErrProviderNotFound
	}

	if provider.Status == string(types.ProviderStatusDeregistered) || provider.Status == string(types.ProviderStatusUnbonding) {
		return types.ErrProviderUnbonding
	}

	if msg.Endpoint != "" {
		provider.Endpoint = msg.Endpoint
	}

	if msg.Price != "" {
		// Validate price ceiling
		price := new(big.Int)
		price.SetString(msg.Price, 10)
		maxPrice := new(big.Int)
		maxPrice.SetString(params.MaxPricePerCu, 10)
		if price.Cmp(maxPrice) > 0 {
			return types.ErrPriceExceedsCeiling
		}

		// Schedule price change with delay
		height := uint64(ctx.BlockHeight())
		provider.PendingPrice = msg.Price
		provider.PriceChangeAt = height + params.PriceChangeDelayBlocks
	}

	if msg.Pause {
		provider.Status = string(types.ProviderStatusPaused)
	}
	if msg.Unpause {
		if provider.Status == string(types.ProviderStatusPaused) {
			provider.Status = string(types.ProviderStatusActive)
		}
	}

	k.SetProvider(ctx, provider)
	return nil
}

// DeregisterProviderState begins the provider unbonding process.
func (k Keeper) DeregisterProviderState(ctx sdk.Context, address string) error {
	params := k.GetParams(ctx)

	provider, found := k.GetProvider(ctx, address)
	if !found {
		return types.ErrProviderNotFound
	}

	if provider.Status == string(types.ProviderStatusUnbonding) || provider.Status == string(types.ProviderStatusDeregistered) {
		return types.ErrProviderUnbonding
	}

	height := uint64(ctx.BlockHeight())
	provider.Status = string(types.ProviderStatusUnbonding)
	provider.UnbondingAt = height + params.ProviderUnbondingBlocks

	k.SetProvider(ctx, provider)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_provider_unbonding",
			sdk.NewAttribute("address", address),
			sdk.NewAttribute("unbonds_at", fmt.Sprintf("%d", provider.UnbondingAt)),
		),
	)

	return nil
}

// ProcessHeartbeat records a provider heartbeat.
func (k Keeper) ProcessHeartbeat(ctx sdk.Context, address string) error {
	provider, found := k.GetProvider(ctx, address)
	if !found {
		return types.ErrProviderNotFound
	}

	if provider.Status == string(types.ProviderStatusDeregistered) || provider.Status == string(types.ProviderStatusUnbonding) {
		return types.ErrProviderUnbonding
	}

	height := uint64(ctx.BlockHeight())

	// Re-activate suspended providers on heartbeat
	if provider.Status == string(types.ProviderStatusSuspended) {
		provider.Status = string(types.ProviderStatusActive)
	}

	provider.LastHeartbeat = height
	k.SetProvider(ctx, provider)

	return nil
}

// CheckProviderLiveness suspends providers that haven't sent a heartbeat.
// Called in BeginBlock.
func (k Keeper) CheckProviderLiveness(ctx sdk.Context) {
	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())

	var toSuspend []string
	var toComplete []string

	k.IterateProviders(ctx, func(p types.ComputeProvider) bool {
		// Check liveness for active providers
		if p.Status == string(types.ProviderStatusActive) {
			if height-p.LastHeartbeat > params.HeartbeatIntervalBlocks {
				toSuspend = append(toSuspend, p.Address)
			}
		}

		// Complete unbonding for providers past their unbonding period
		if p.Status == string(types.ProviderStatusUnbonding) && height >= p.UnbondingAt {
			toComplete = append(toComplete, p.Address)
		}

		return false
	})

	// Suspend inactive providers
	for _, addr := range toSuspend {
		provider, _ := k.GetProvider(ctx, addr)
		provider.Status = string(types.ProviderStatusSuspended)
		k.SetProvider(ctx, provider)

		k.Logger(ctx).Info("suspended compute provider for missed heartbeat",
			"address", addr,
			"last_heartbeat", provider.LastHeartbeat,
		)
	}

	// Complete unbonding — return stake
	for _, addr := range toComplete {
		k.completeUnbonding(ctx, addr)
	}
}

// ApplyPendingPriceChanges activates pending price changes that have passed their delay.
// Called in BeginBlock.
func (k Keeper) ApplyPendingPriceChanges(ctx sdk.Context) {
	height := uint64(ctx.BlockHeight())

	k.IterateProviders(ctx, func(p types.ComputeProvider) bool {
		if p.PendingPrice != "" && height >= p.PriceChangeAt {
			p.PricePerCu = p.PendingPrice
			p.PendingPrice = ""
			p.PriceChangeAt = 0
			k.SetProvider(ctx, p)
		}
		return false
	})
}

// completeUnbonding returns the provider's stake and marks them as deregistered.
func (k Keeper) completeUnbonding(ctx sdk.Context, address string) {
	provider, found := k.GetProvider(ctx, address)
	if !found {
		return
	}

	// Return stake
	stakeAmt := new(big.Int)
	stakeAmt.SetString(provider.Stake, 10)

	if stakeAmt.Sign() > 0 {
		providerAddr, err := sdk.AccAddressFromBech32(address)
		if err == nil {
			stakeCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmt)))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAddr, stakeCoins); err != nil {
				k.Logger(ctx).Error("failed to return provider stake", "address", address, "error", err)
				return
			}
		}
	}

	provider.Status = string(types.ProviderStatusDeregistered)
	provider.Stake = "0"
	k.SetProvider(ctx, provider)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_provider_deregistered",
			sdk.NewAttribute("address", address),
		),
	)

	k.Logger(ctx).Info("completed provider unbonding", "address", address)
}

// GetActiveProvidersByType returns all active providers of a given service type.
func (k Keeper) GetActiveProvidersByType(ctx sdk.Context, serviceType string) []types.ComputeProvider {
	var result []types.ComputeProvider
	k.IterateProviders(ctx, func(p types.ComputeProvider) bool {
		if p.Status == string(types.ProviderStatusActive) {
			if serviceType == "" || p.ServiceType == serviceType {
				result = append(result, p)
			}
		}
		return false
	})
	return result
}
