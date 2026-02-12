package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/billing/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// RegisterProvider handles MsgRegisterProvider -- register as a knowledge API provider.
//
// Workflow:
// 1. Validate sender address and stake amount
// 2. Check provider doesn't already exist
// 3. Verify stake meets minimum requirement
// 4. Transfer stake from sender to module account
// 5. Create and store provider record
// 6. Update domain indexes
func (k msgServer) RegisterProvider(goCtx context.Context, msg *types.MsgRegisterProvider) (*types.MsgRegisterProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	// Check provider doesn't already exist
	if _, found := k.GetProvider(ctx, msg.Sender); found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderExists, msg.Sender)
	}

	// Validate stake amount
	stakeAmt := new(big.Int)
	if _, ok := stakeAmt.SetString(msg.StakeAmount, 10); !ok || stakeAmt.Sign() <= 0 {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, msg.StakeAmount)
	}

	params := k.GetParams(ctx)
	minStake := new(big.Int)
	minStake.SetString(params.MinProviderStake, 10)
	if stakeAmt.Cmp(minStake) < 0 {
		return nil, fmt.Errorf("%w: need %s, got %s", types.ErrInsufficientStake, params.MinProviderStake, msg.StakeAmount)
	}

	// Transfer stake to module account
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmt)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to stake: %w", err)
	}

	// Create provider record
	provider := types.Provider{
		Address:      msg.Sender,
		Domains:      msg.Domains,
		StakeAmount:  msg.StakeAmount,
		Active:       true,
		TotalCalls:   0,
		TotalRevenue: "0",
		RegisteredAt: uint64(ctx.BlockHeight()),
	}
	k.SetProvider(ctx, provider)

	// Update domain indexes
	for _, domain := range msg.Domains {
		k.SetDomainIndex(ctx, domain, msg.Sender)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_registered",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("stake", msg.StakeAmount),
			sdk.NewAttribute("domains", fmt.Sprintf("%v", msg.Domains)),
		),
	)

	k.Logger(ctx).Info("provider registered",
		"address", msg.Sender,
		"stake", msg.StakeAmount,
	)

	return &types.MsgRegisterProviderResponse{}, nil
}

// DeregisterProvider handles MsgDeregisterProvider -- remove provider and refund stake.
func (k msgServer) DeregisterProvider(goCtx context.Context, msg *types.MsgDeregisterProvider) (*types.MsgDeregisterProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	provider, found := k.GetProvider(ctx, msg.Sender)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, msg.Sender)
	}

	// Refund stake
	stakeAmt := new(big.Int)
	stakeAmt.SetString(provider.StakeAmount, 10)
	if stakeAmt.Sign() > 0 {
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmt)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, senderAddr, coins); err != nil {
			return nil, fmt.Errorf("failed to refund stake: %w", err)
		}
	}

	// Remove domain indexes
	for _, domain := range provider.Domains {
		k.DeleteDomainIndex(ctx, domain, msg.Sender)
	}

	// Delete provider
	k.DeleteProvider(ctx, msg.Sender)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_deregistered",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("stake_refunded", provider.StakeAmount),
		),
	)

	k.Logger(ctx).Info("provider deregistered",
		"address", msg.Sender,
		"stake_refunded", provider.StakeAmount,
	)

	return &types.MsgDeregisterProviderResponse{}, nil
}

// RequestQuote handles MsgRequestQuote -- generate a price quote for fact queries.
func (k msgServer) RequestQuote(goCtx context.Context, msg *types.MsgRequestQuote) (*types.MsgRequestQuoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	// Verify provider exists and is active
	provider, found := k.GetProvider(ctx, msg.Provider)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, msg.Provider)
	}
	if !provider.Active {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotActive, msg.Provider)
	}

	currentBlock := uint64(ctx.BlockHeight())
	quote := k.GenerateQuote(ctx, msg.Provider, msg.Sender, msg.FactIds, currentBlock)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"quote_generated",
			sdk.NewAttribute("quote_id", fmt.Sprintf("%d", quote.QuoteId)),
			sdk.NewAttribute("provider", msg.Provider),
			sdk.NewAttribute("caller", msg.Sender),
			sdk.NewAttribute("total_price", quote.TotalPrice),
		),
	)

	return &types.MsgRequestQuoteResponse{Quote: quote}, nil
}

// ExecutePayment handles MsgExecutePayment -- execute a quoted knowledge payment.
//
// Workflow:
// 1. Look up quote, validate not expired and not already used
// 2. Verify caller matches quote caller
// 3. Calculate distribution
// 4. Execute atomic transfers
// 5. Increment citation counts for all queried facts
// 6. Update provider stats
// 7. Record the call
func (k msgServer) ExecutePayment(goCtx context.Context, msg *types.MsgExecutePayment) (*types.MsgExecutePaymentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	callerAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	// Look up quote
	quote, found := k.GetQuote(ctx, msg.QuoteId)
	if !found {
		return nil, fmt.Errorf("%w: %d", types.ErrQuoteNotFound, msg.QuoteId)
	}

	// Validate quote
	if quote.Executed {
		return nil, fmt.Errorf("%w: %d", types.ErrQuoteAlreadyUsed, msg.QuoteId)
	}

	currentBlock := uint64(ctx.BlockHeight())
	if currentBlock > quote.ValidUntil {
		return nil, fmt.Errorf("%w: valid until %d, current %d", types.ErrQuoteExpired, quote.ValidUntil, currentBlock)
	}

	// Verify caller
	if msg.Sender != quote.Caller {
		return nil, fmt.Errorf("%w: quote caller %s, sender %s", types.ErrUnauthorized, quote.Caller, msg.Sender)
	}

	// Parse total payment
	totalPayment := new(big.Int)
	totalPayment.SetString(quote.TotalPrice, 10)

	providerAddr, err := sdk.AccAddressFromBech32(quote.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider address in quote: %w", err)
	}

	// Calculate distribution
	distribution := k.CalculateDistribution(ctx, totalPayment, quote.FactIds)

	// Execute distribution
	if err := k.ExecuteDistribution(ctx, callerAddr, providerAddr, distribution); err != nil {
		return nil, fmt.Errorf("failed to execute distribution: %w", err)
	}

	// Increment citation counts for queried facts
	for _, factId := range quote.FactIds {
		if err := k.knowledgeKeeper.IncrementCitationCount(ctx, factId); err != nil {
			k.Logger(ctx).Error("failed to increment citation count",
				"fact_id", factId,
				"error", err,
			)
		}
	}

	// Mark quote as executed
	quote.Executed = true
	k.SetQuote(ctx, quote)

	// Update provider stats
	provider, provFound := k.GetProvider(ctx, quote.Provider)
	if provFound {
		provider.TotalCalls++
		revenue := new(big.Int)
		revenue.SetString(provider.TotalRevenue, 10)
		providerShare := new(big.Int)
		providerShare.SetString(distribution.ProviderShare, 10)
		revenue.Add(revenue, providerShare)
		provider.TotalRevenue = revenue.String()
		k.SetProvider(ctx, provider)
	}

	// Record the call
	recordId := k.GetNextCallRecordId(ctx)
	record := types.CallRecord{
		RecordId:  recordId,
		QuoteId:   msg.QuoteId,
		Caller:    msg.Sender,
		Provider:  quote.Provider,
		TotalPaid: quote.TotalPrice,
		FactCount: uint32(len(quote.FactIds)),
		Block:     currentBlock,
	}
	k.SetCallRecord(ctx, record)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"payment_executed",
			sdk.NewAttribute("record_id", fmt.Sprintf("%d", recordId)),
			sdk.NewAttribute("quote_id", fmt.Sprintf("%d", msg.QuoteId)),
			sdk.NewAttribute("caller", msg.Sender),
			sdk.NewAttribute("provider", quote.Provider),
			sdk.NewAttribute("total_paid", quote.TotalPrice),
		),
	)

	k.Logger(ctx).Info("payment executed",
		"record_id", recordId,
		"quote_id", msg.QuoteId,
		"total_paid", quote.TotalPrice,
	)

	return &types.MsgExecutePaymentResponse{RecordId: recordId}, nil
}

// UpdateParams handles MsgUpdateParams -- governance-gated parameter update.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"billing_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
