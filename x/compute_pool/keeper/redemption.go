package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/legible-money/legible/x/compute_pool/types"
)

// CreateRedemptionTicket validates and creates a new redemption ticket.
//
// Flow:
//  1. Validate credits sufficient and not expired
//  2. Validate provider is active
//  3. Check provider epoch cap (max 30% of pool per provider per epoch)
//  4. Deduct credits
//  5. Create ticket (pending settlement)
func (k Keeper) CreateRedemptionTicket(ctx sdk.Context, validatorAddr, providerAddr string, cuAmount uint64) (*types.RedemptionTicket, error) {
	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())
	epoch := height / types.BlocksPerEpoch

	// Cannot redeem to own provider
	if validatorAddr == providerAddr {
		return nil, types.ErrSelfRedemption
	}

	// Validate provider exists and is active
	provider, found := k.GetProvider(ctx, providerAddr)
	if !found {
		return nil, types.ErrProviderNotFound
	}
	if provider.Status != string(types.ProviderStatusActive) {
		return nil, types.ErrProviderNotActive
	}

	// Check provider epoch cap
	if err := k.checkProviderEpochCap(ctx, params, epoch, providerAddr, cuAmount, provider.PricePerCU); err != nil {
		return nil, err
	}

	// Deduct credits from validator
	if err := k.DeductCredits(ctx, validatorAddr, cuAmount); err != nil {
		return nil, err
	}

	// Calculate LGM cost: cuAmount × pricePerCU
	pricePerCU := new(big.Int)
	pricePerCU.SetString(provider.PricePerCU, 10)
	lgmAmount := new(big.Int).Mul(big.NewInt(int64(cuAmount)), pricePerCU)

	// Verify pool has sufficient funds
	poolBalance := k.GetPoolBalance(ctx)
	if poolBalance.Cmp(lgmAmount) < 0 {
		// Refund credits
		credit, _ := k.GetCredit(ctx, validatorAddr)
		credit.Balance += cuAmount
		credit.RedeemedTotal -= cuAmount
		k.SetCredit(ctx, credit)
		return nil, types.ErrPoolInsufficient
	}

	// Create ticket
	ticketId := k.GetNextTicketId(ctx)
	ticket := types.RedemptionTicket{
		Id:        ticketId,
		Validator: validatorAddr,
		Provider:  providerAddr,
		CUAmount:  cuAmount,
		LGMAmount: lgmAmount.String(),
		Status:    string(types.TicketStatusPending),
		CreatedAt: height,
		ExpiresAt: height + params.RedemptionExpiryBlocks,
	}

	k.SetTicket(ctx, ticket)

	// Update epoch stats
	stats := k.GetOrCreateEpochStats(ctx, epoch)
	stats.CreditsRedeemed += cuAmount
	k.SetEpochStats(ctx, stats)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_credits_redeemed",
			sdk.NewAttribute("ticket_id", fmt.Sprintf("%d", ticketId)),
			sdk.NewAttribute("validator", validatorAddr),
			sdk.NewAttribute("provider", providerAddr),
			sdk.NewAttribute("cu_amount", fmt.Sprintf("%d", cuAmount)),
			sdk.NewAttribute("lgm_amount", lgmAmount.String()),
		),
	)

	return &ticket, nil
}

// ConfirmRedemption settles a redemption ticket — transfers LGM to provider.
func (k Keeper) ConfirmRedemption(ctx sdk.Context, validatorAddr string, ticketId uint64, actualCU uint64) error {
	height := uint64(ctx.BlockHeight())
	epoch := height / types.BlocksPerEpoch

	ticket, found := k.GetTicket(ctx, ticketId)
	if !found {
		return types.ErrTicketNotFound
	}

	// Only the ticket's validator can confirm
	if ticket.Validator != validatorAddr {
		return types.ErrUnauthorized
	}

	if ticket.Status != string(types.TicketStatusPending) {
		return types.ErrTicketAlreadyConfirmed
	}

	if height > ticket.ExpiresAt {
		return types.ErrTicketExpired
	}

	provider, found := k.GetProvider(ctx, ticket.Provider)
	if !found {
		return types.ErrProviderNotFound
	}

	// Calculate actual payment (may be less if actualCU < CUAmount)
	effectiveCU := actualCU
	if effectiveCU > ticket.CUAmount {
		effectiveCU = ticket.CUAmount
	}

	pricePerCU := new(big.Int)
	pricePerCU.SetString(provider.PricePerCU, 10)
	actualPayment := new(big.Int).Mul(big.NewInt(int64(effectiveCU)), pricePerCU)

	// Refund excess credits if actual < requested
	if effectiveCU < ticket.CUAmount {
		excess := ticket.CUAmount - effectiveCU
		credit, _ := k.GetCredit(ctx, validatorAddr)
		credit.Balance += excess
		credit.RedeemedTotal -= excess
		k.SetCredit(ctx, credit)
	}

	// Transfer LGM from compute_pool module to provider
	if actualPayment.Sign() > 0 {
		providerAccAddr, err := sdk.AccAddressFromBech32(ticket.Provider)
		if err != nil {
			return fmt.Errorf("invalid provider address: %w", err)
		}

		paymentCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(actualPayment)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAccAddr, paymentCoins); err != nil {
			return fmt.Errorf("failed to pay provider: %w", err)
		}
	}

	// Update ticket
	ticket.Status = string(types.TicketStatusConfirmed)
	ticket.ActualCU = effectiveCU
	k.SetTicket(ctx, ticket)

	// Create usage report
	report := types.UsageReport{
		TicketId:    ticketId,
		Validator:   validatorAddr,
		Provider:    ticket.Provider,
		RequestedCU: ticket.CUAmount,
		ActualCU:    effectiveCU,
		LGMPaid:     actualPayment.String(),
		ConfirmedAt: height,
	}
	k.SetUsageReport(ctx, report)

	// Update provider metrics
	provider.TasksServed++
	k.SetProvider(ctx, provider)

	// Update provider epoch flow
	flow := k.GetOrCreateProviderEpochFlow(ctx, epoch, ticket.Provider)
	received := new(big.Int)
	received.SetString(flow.LGMReceived, 10)
	received.Add(received, actualPayment)
	flow.LGMReceived = received.String()
	flow.TasksServed++
	k.SetProviderEpochFlow(ctx, flow)

	// Update epoch stats
	stats := k.GetOrCreateEpochStats(ctx, epoch)
	distributed := new(big.Int)
	distributed.SetString(stats.LGMDistributed, 10)
	distributed.Add(distributed, actualPayment)
	stats.LGMDistributed = distributed.String()
	k.SetEpochStats(ctx, stats)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_usage_confirmed",
			sdk.NewAttribute("ticket_id", fmt.Sprintf("%d", ticketId)),
			sdk.NewAttribute("actual_cu", fmt.Sprintf("%d", effectiveCU)),
			sdk.NewAttribute("lgm_paid", actualPayment.String()),
		),
	)

	return nil
}

// DisputeRedemption marks a ticket as disputed.
func (k Keeper) DisputeRedemption(ctx sdk.Context, validatorAddr string, ticketId uint64, reason string) error {
	height := uint64(ctx.BlockHeight())

	ticket, found := k.GetTicket(ctx, ticketId)
	if !found {
		return types.ErrTicketNotFound
	}

	if ticket.Validator != validatorAddr {
		return types.ErrUnauthorized
	}

	if ticket.Status != string(types.TicketStatusPending) {
		if ticket.Status == string(types.TicketStatusConfirmed) {
			return types.ErrTicketAlreadyConfirmed
		}
		return types.ErrTicketAlreadyDisputed
	}

	if height > ticket.ExpiresAt {
		return types.ErrTicketExpired
	}

	// Mark as disputed — credits are returned to validator
	ticket.Status = string(types.TicketStatusDisputed)
	k.SetTicket(ctx, ticket)

	// Return credits
	credit, _ := k.GetCredit(ctx, validatorAddr)
	credit.Balance += ticket.CUAmount
	credit.RedeemedTotal -= ticket.CUAmount
	// Extend expiry since credits were unused
	params := k.GetParams(ctx)
	credit.ExpiresAt = height + params.CreditExpiryBlocks
	k.SetCredit(ctx, credit)

	// Decrement provider reliability
	provider, found := k.GetProvider(ctx, ticket.Provider)
	if found {
		provider.TasksFailed++
		k.SetProvider(ctx, provider)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_usage_disputed",
			sdk.NewAttribute("ticket_id", fmt.Sprintf("%d", ticketId)),
			sdk.NewAttribute("reason", reason),
			sdk.NewAttribute("provider", ticket.Provider),
		),
	)

	return nil
}

// ExpireRedemptionTickets expires unconfirmed tickets past their ExpiresAt.
// Called in BeginBlock.
func (k Keeper) ExpireRedemptionTickets(ctx sdk.Context) {
	height := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)

	var toExpire []types.RedemptionTicket

	k.IterateTickets(ctx, func(ticket types.RedemptionTicket) bool {
		if ticket.Status == string(types.TicketStatusPending) && height > ticket.ExpiresAt {
			toExpire = append(toExpire, ticket)
		}
		return false
	})

	for _, ticket := range toExpire {
		// Expire ticket
		ticket.Status = string(types.TicketStatusExpired)
		k.SetTicket(ctx, ticket)

		// Return credits to validator
		credit, _ := k.GetCredit(ctx, ticket.Validator)
		credit.Balance += ticket.CUAmount
		credit.RedeemedTotal -= ticket.CUAmount
		credit.ExpiresAt = height + params.CreditExpiryBlocks
		k.SetCredit(ctx, credit)

		// Decrement provider reliability
		provider, found := k.GetProvider(ctx, ticket.Provider)
		if found {
			provider.TasksFailed++
			k.SetProvider(ctx, provider)
		}

		k.Logger(ctx).Debug("expired redemption ticket",
			"ticket_id", ticket.Id,
			"validator", ticket.Validator,
			"provider", ticket.Provider,
		)
	}
}

// checkProviderEpochCap ensures a provider doesn't receive more than MaxProviderShareBps
// of the pool's epoch inflow.
func (k Keeper) checkProviderEpochCap(ctx sdk.Context, params types.Params, epoch uint64, providerAddr string, cuAmount uint64, pricePerCU string) error {
	// Get epoch pool inflow
	stats := k.GetOrCreateEpochStats(ctx, epoch)
	poolInflow := new(big.Int)
	poolInflow.SetString(stats.PoolInflow, 10)

	if poolInflow.Sign() <= 0 {
		// No inflow yet this epoch — allow the first transaction
		return nil
	}

	// Calculate max allowed for this provider
	maxAllowed := new(big.Int).Mul(poolInflow, big.NewInt(int64(params.MaxProviderShareBps)))
	maxAllowed.Div(maxAllowed, big.NewInt(1000000))

	// Get current provider flow
	flow := k.GetOrCreateProviderEpochFlow(ctx, epoch, providerAddr)
	currentFlow := new(big.Int)
	currentFlow.SetString(flow.LGMReceived, 10)

	// Calculate this redemption's cost
	price := new(big.Int)
	price.SetString(pricePerCU, 10)
	thisCost := new(big.Int).Mul(big.NewInt(int64(cuAmount)), price)

	// Check cap
	total := new(big.Int).Add(currentFlow, thisCost)
	if total.Cmp(maxAllowed) > 0 {
		return fmt.Errorf("%w: provider %s would exceed %d bps of epoch inflow", types.ErrProviderCapExceeded, providerAddr, params.MaxProviderShareBps)
	}

	return nil
}

// GetPoolBalance returns the compute pool module account balance in ulgm.
func (k Keeper) GetPoolBalance(ctx sdk.Context) *big.Int {
	if k.bankKeeper == nil {
		return new(big.Int)
	}
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName)
	balance := k.bankKeeper.GetBalance(ctx, moduleAddr, "ulgm")
	return balance.Amount.BigInt()
}
