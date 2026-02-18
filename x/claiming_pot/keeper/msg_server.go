package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/claiming_pot/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns a MsgServer implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreatePot creates a new claiming pot.
func (m msgServer) CreatePot(goCtx context.Context, msg *types.MsgCreatePot) (*types.MsgCreatePotResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	// Validate initial funding
	if msg.InitialFunding == "" || msg.InitialFunding == "0" {
		return nil, types.ErrInvalidConfig.Wrap("initial funding must be positive")
	}

	// Check expiry height
	if msg.ExpiryHeight > 0 && msg.ExpiryHeight <= uint64(ctx.BlockHeight()) {
		return nil, types.ErrInvalidConfig.Wrap("expiration must be in the future")
	}

	// Check creator pot count
	count := m.Keeper.GetCreatorPotCount(ctx, msg.Creator)
	if count >= params.MaxPotsPerCreator {
		return nil, types.ErrMaxPotsExceeded
	}

	// Check initial deposit >= minimum
	minDeposit, _ := new(big.Int).SetString(params.MinInitialDeposit, 10)
	deposit, ok := new(big.Int).SetString(msg.InitialFunding, 10)
	if !ok || deposit.Sign() <= 0 {
		return nil, types.ErrInsufficientDeposit.Wrap("invalid deposit amount")
	}
	if minDeposit != nil && deposit.Cmp(minDeposit) < 0 {
		return nil, types.ErrInsufficientDeposit
	}

	// Transfer initial deposit from creator to module account
	creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}
	depositCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(deposit)))
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, types.ModuleName, depositCoins); err != nil {
			return nil, fmt.Errorf("failed to transfer deposit: %w", err)
		}
	}

	// Generate pot ID
	potId := m.Keeper.GetNextPotId(ctx)

	// Build PotConfig from message fields
	config := &types.PotConfig{
		AmountPerClaim:      msg.InitialFunding, // default: entire funding is one claim
		MaxClaims:           0,                  // unlimited by default
		MaxClaimsPerAccount: 1,                  // default: one claim per account
		ExpiresAtBlock:      msg.ExpiryHeight,
		CreatorCanClose:     true,
	}

	// Create pot
	pot := types.ClaimingPot{
		PotId:          potId,
		Creator:        msg.Creator,
		Config:         config,
		TotalDeposited: msg.InitialFunding,
		TotalClaimed:   "0",
		Balance:        msg.InitialFunding,
		ClaimCount:     0,
		CreatedAtBlock: uint64(ctx.BlockHeight()),
		Status:         string(types.PotStatusActive),
	}

	m.Keeper.SetPot(ctx, pot)
	m.Keeper.IncrementCreatorPotCount(ctx, msg.Creator)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"create_pot",
		sdk.NewAttribute("pot_id", potId),
		sdk.NewAttribute("creator", msg.Creator),
		sdk.NewAttribute("deposit", msg.InitialFunding),
	))

	return &types.MsgCreatePotResponse{PotId: potId}, nil
}

// FundPot adds funds to an existing pot.
func (m msgServer) FundPot(goCtx context.Context, msg *types.MsgFundPot) (*types.MsgFundPotResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pot, found := m.Keeper.GetPot(ctx, msg.PotId)
	if !found {
		return nil, types.ErrPotNotFound
	}
	if pot.Status == string(types.PotStatusClosed) {
		return nil, types.ErrPotNotActive.Wrap("pot is closed")
	}

	amount, ok := new(big.Int).SetString(msg.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid fund amount")
	}

	// Transfer from funder to module account
	funderAddr, err := sdk.AccAddressFromBech32(msg.Funder)
	if err != nil {
		return nil, fmt.Errorf("invalid funder address: %w", err)
	}
	fundCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amount)))
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, funderAddr, types.ModuleName, fundCoins); err != nil {
			return nil, fmt.Errorf("failed to transfer funds: %w", err)
		}
	}

	// Update pot balance and total deposited
	balance, _ := new(big.Int).SetString(pot.Balance, 10)
	totalDeposited, _ := new(big.Int).SetString(pot.TotalDeposited, 10)
	balance.Add(balance, amount)
	totalDeposited.Add(totalDeposited, amount)
	pot.Balance = balance.String()
	pot.TotalDeposited = totalDeposited.String()

	// Reactivate exhausted pot if it now has enough for a claim
	if pot.Status == string(types.PotStatusExhausted) {
		amountPerClaim, _ := new(big.Int).SetString(pot.Config.AmountPerClaim, 10)
		if amountPerClaim != nil && balance.Cmp(amountPerClaim) >= 0 {
			pot.Status = string(types.PotStatusActive)
		}
	}

	m.Keeper.SetPot(ctx, pot)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"fund_pot",
		sdk.NewAttribute("pot_id", msg.PotId),
		sdk.NewAttribute("funder", msg.Funder),
		sdk.NewAttribute("amount", msg.Amount),
	))

	return &types.MsgFundPotResponse{}, nil
}

// ClaimFromPot claims from an eligible pot.
func (m msgServer) ClaimFromPot(goCtx context.Context, msg *types.MsgClaimFromPot) (*types.MsgClaimFromPotResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pot, found := m.Keeper.GetPot(ctx, msg.PotId)
	if !found {
		return nil, types.ErrPotNotFound
	}
	if pot.Status != string(types.PotStatusActive) {
		return nil, types.ErrPotNotActive
	}

	// Check expiration
	if pot.Config.ExpiresAtBlock > 0 && uint64(ctx.BlockHeight()) > pot.Config.ExpiresAtBlock {
		pot.Status = string(types.PotStatusExpired)
		m.Keeper.SetPot(ctx, pot)
		return nil, types.ErrPotExpired
	}

	// Check max total claims
	if pot.Config.MaxClaims > 0 && pot.ClaimCount >= pot.Config.MaxClaims {
		return nil, types.ErrMaxClaimsReached
	}

	// Check per-account claim limit
	record, _ := m.Keeper.GetClaimRecord(ctx, msg.PotId, msg.Claimer)
	if record.ClaimCount >= pot.Config.MaxClaimsPerAccount {
		return nil, types.ErrAlreadyClaimed
	}

	// Evaluate eligibility
	eligible, reason := m.Keeper.EvaluateEligibility(ctx, pot, msg.Claimer)
	if !eligible {
		return nil, types.ErrIneligible.Wrap(reason)
	}

	// Check pot has enough balance
	amountPerClaim, _ := new(big.Int).SetString(pot.Config.AmountPerClaim, 10)
	balance, _ := new(big.Int).SetString(pot.Balance, 10)
	if balance.Cmp(amountPerClaim) < 0 {
		return nil, types.ErrInsufficientPotFunds
	}

	// Transfer from module account to claimer
	claimerAddr, err := sdk.AccAddressFromBech32(msg.Claimer)
	if err != nil {
		return nil, fmt.Errorf("invalid claimer address: %w", err)
	}
	claimCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amountPerClaim)))
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, claimerAddr, claimCoins); err != nil {
			return nil, fmt.Errorf("failed to transfer claim: %w", err)
		}
	}

	// Update pot
	totalClaimed, _ := new(big.Int).SetString(pot.TotalClaimed, 10)
	balance.Sub(balance, amountPerClaim)
	totalClaimed.Add(totalClaimed, amountPerClaim)
	pot.Balance = balance.String()
	pot.TotalClaimed = totalClaimed.String()
	pot.ClaimCount++

	// Mark exhausted if balance < amountPerClaim
	if balance.Cmp(amountPerClaim) < 0 {
		pot.Status = string(types.PotStatusExhausted)
	}

	m.Keeper.SetPot(ctx, pot)

	// Update claim record
	record.PotId = msg.PotId
	record.Claimer = msg.Claimer
	record.ClaimCount++
	claimedSoFar, _ := new(big.Int).SetString(record.TotalAmount, 10)
	if claimedSoFar == nil {
		claimedSoFar = new(big.Int)
	}
	claimedSoFar.Add(claimedSoFar, amountPerClaim)
	record.TotalAmount = claimedSoFar.String()
	record.LastClaimBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetClaimRecord(ctx, record)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"claim_from_pot",
		sdk.NewAttribute("pot_id", msg.PotId),
		sdk.NewAttribute("claimer", msg.Claimer),
		sdk.NewAttribute("amount", pot.Config.AmountPerClaim),
	))

	return &types.MsgClaimFromPotResponse{AmountClaimed: pot.Config.AmountPerClaim}, nil
}

// ClosePot closes a pot and refunds remaining balance to creator.
func (m msgServer) ClosePot(goCtx context.Context, msg *types.MsgClosePot) (*types.MsgClosePotResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pot, found := m.Keeper.GetPot(ctx, msg.PotId)
	if !found {
		return nil, types.ErrPotNotFound
	}

	// Verify authority matches pot creator
	if msg.Authority != pot.Creator {
		return nil, types.ErrNotPotCreator
	}

	// Verify config allows close
	if !pot.Config.CreatorCanClose {
		return nil, types.ErrCreatorCannotClose
	}

	// Already closed
	if pot.Status == string(types.PotStatusClosed) {
		return nil, types.ErrPotNotActive.Wrap("pot already closed")
	}

	// Refund remaining balance to creator
	refunded := pot.Balance
	balance, _ := new(big.Int).SetString(pot.Balance, 10)
	if balance != nil && balance.Sign() > 0 && m.Keeper.bankKeeper != nil {
		creatorAddr, err := sdk.AccAddressFromBech32(pot.Creator)
		if err != nil {
			return nil, fmt.Errorf("invalid creator address: %w", err)
		}
		refundCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(balance)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, refundCoins); err != nil {
			return nil, fmt.Errorf("failed to refund: %w", err)
		}
	}

	pot.Status = string(types.PotStatusClosed)
	pot.Balance = "0"
	pot.ClosedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetPot(ctx, pot)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"close_pot",
		sdk.NewAttribute("pot_id", msg.PotId),
		sdk.NewAttribute("authority", msg.Authority),
		sdk.NewAttribute("refunded", refunded),
	))

	return &types.MsgClosePotResponse{RefundedAmount: refunded}, nil
}


// UpdateParams handles MsgUpdateParams -- governance-gated parameter update.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if m.Keeper.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", m.Keeper.GetAuthority(), msg.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	m.Keeper.SetParams(ctx, *msg.Params)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"claiming_pot_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)
	return &types.MsgUpdateParamsResponse{}, nil
}
