package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/channels/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns a MsgServer implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// NewFullMsgServerImpl returns a FullMsgServer implementation that includes
// both protobuf service methods and supplemental methods (DepositChannel, UpdateState, ClaimExpired).
func NewFullMsgServerImpl(keeper Keeper) types.FullMsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}
var _ types.FullMsgServer = msgServer{}

// OpenChannel opens a new payment channel.
func (m msgServer) OpenChannel(goCtx context.Context, msg *types.MsgOpenChannel) (*types.MsgOpenChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	// Validate sender != receiver
	if msg.Sender == msg.Receiver {
		return nil, fmt.Errorf("sender and receiver cannot be the same address")
	}

	// Check deposit >= MinDeposit
	minDeposit, _ := new(big.Int).SetString(params.MinDeposit, 10)
	deposit, ok := new(big.Int).SetString(msg.Deposit, 10)
	if !ok || deposit.Sign() <= 0 {
		return nil, types.ErrInsufficientDeposit
	}
	if minDeposit != nil && deposit.Cmp(minDeposit) < 0 {
		return nil, types.ErrInsufficientDeposit
	}

	// Check timeout height (duration) <= MaxChannelDuration
	if msg.TimeoutHeight > params.MaxChannelDuration {
		return nil, types.ErrChannelDurationExceeded
	}

	// Check sender open channels < MaxChannelsPerPayer
	openCount := m.Keeper.GetOpenChannelCountForPayer(ctx, msg.Sender)
	if openCount >= params.MaxChannelsPerPayer {
		return nil, types.ErrMaxChannelsExceeded
	}

	// Transfer deposit from sender to module account
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}
	depositCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(deposit)))
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, depositCoins); err != nil {
			return nil, fmt.Errorf("failed to transfer deposit: %w", err)
		}
	}

	// Generate channel ID
	counter := m.Keeper.GetNextChannelId(ctx)
	channelId := fmt.Sprintf("pc-%d-%d", ctx.BlockHeight(), counter)

	// Set settlement frequency
	settlementFreq := params.DefaultSettlementFreq

	// Create channel
	ch := types.PaymentChannel{
		ChannelId:           channelId,
		Payer:               msg.Sender,
		Provider:            msg.Receiver,
		Deposited:           msg.Deposit,
		Spent:               "0",
		Available:           msg.Deposit,
		Status:              types.ChannelStatusOpen,
		OpenedAtBlock:       uint64(ctx.BlockHeight()),
		ExpiresAtBlock:      uint64(ctx.BlockHeight()) + msg.TimeoutHeight,
		Nonce:               0,
		LastStateHash:       "",
		SettlementFrequency: settlementFreq,
		LastSettlementBlock: uint64(ctx.BlockHeight()),
	}

	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"open_channel",
		sdk.NewAttribute("channel_id", channelId),
		sdk.NewAttribute("sender", msg.Sender),
		sdk.NewAttribute("receiver", msg.Receiver),
		sdk.NewAttribute("deposit", msg.Deposit),
	))

	return &types.MsgOpenChannelResponse{ChannelId: channelId}, nil
}

// UpdateChannel submits an off-chain channel state update.
func (m msgServer) UpdateChannel(goCtx context.Context, msg *types.MsgUpdateChannel) (*types.MsgUpdateChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen {
		return nil, types.ErrChannelNotOpen
	}

	// Verify sender is payer or provider
	if msg.Sender != ch.Payer && msg.Sender != ch.Provider {
		return nil, types.ErrNotChannelParty
	}

	// Verify sequence is higher
	if msg.Sequence <= ch.Nonce {
		return nil, types.ErrInvalidNonce
	}

	// Parse new balance receiver as spent
	spent, ok := new(big.Int).SetString(msg.NewBalanceReceiver, 10)
	if !ok {
		return nil, fmt.Errorf("invalid new balance receiver amount")
	}
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	if spent.Cmp(deposited) > 0 {
		return nil, types.ErrSpentExceedsDeposit
	}

	// Update channel state
	ch.Nonce = msg.Sequence
	ch.Spent = msg.NewBalanceReceiver
	available := new(big.Int).Sub(deposited, spent)
	ch.Available = available.String()

	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"update_channel",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("sequence", fmt.Sprintf("%d", msg.Sequence)),
		sdk.NewAttribute("new_balance_receiver", msg.NewBalanceReceiver),
	))

	return &types.MsgUpdateChannelResponse{}, nil
}

// DisputeChannel opens a dispute on a channel.
func (m msgServer) DisputeChannel(goCtx context.Context, msg *types.MsgDisputeChannel) (*types.MsgDisputeChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen && ch.Status != types.ChannelStatusClosing && ch.Status != types.ChannelStatusDisputed {
		return nil, types.ErrChannelNotOpen.Wrap("channel must be open, closing, or disputed to dispute")
	}

	// Verify disputer is payer or provider
	if msg.Disputer != ch.Payer && msg.Disputer != ch.Provider {
		return nil, types.ErrNotChannelParty
	}

	// Check no active dispute (or sequence must be higher)
	existingDispute, hasDispute := m.Keeper.GetDispute(ctx, msg.ChannelId)
	if hasDispute && !existingDispute.Resolved {
		if msg.Sequence <= existingDispute.DisputedNonce {
			return nil, types.ErrDisputeAlreadyActive
		}
	}

	// Set channel to disputed
	ch.Status = types.ChannelStatusDisputed
	deadline := uint64(ctx.BlockHeight()) + params.DisputePeriodBlocks
	ch.DisputeDeadline = deadline
	ch.DisputeNonce = msg.Sequence
	m.Keeper.SetChannel(ctx, ch)

	// Create dispute record
	dispute := types.ChannelDispute{
		ChannelId:        msg.ChannelId,
		Disputer:         msg.Disputer,
		DisputedNonce:    msg.Sequence,
		DisputedSpent:    msg.ClaimedBalanceReceiver,
		DisputeStateHash: "",
		DeadlineBlock:    deadline,
		Resolved:         false,
	}
	m.Keeper.SetDispute(ctx, dispute)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"dispute_channel",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("disputer", msg.Disputer),
		sdk.NewAttribute("deadline", fmt.Sprintf("%d", deadline)),
	))

	return &types.MsgDisputeChannelResponse{}, nil
}

// SettleChannel settles a channel (distributes funds).
func (m msgServer) SettleChannel(goCtx context.Context, msg *types.MsgSettleChannel) (*types.MsgSettleChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}

	// Verify authority is payer or provider
	if msg.Authority != ch.Payer && msg.Authority != ch.Provider {
		return nil, types.ErrNotChannelParty
	}

	if ch.Status == types.ChannelStatusClosed || ch.Status == types.ChannelStatusSettled {
		return nil, types.ErrChannelAlreadyClosed
	}

	// If disputed, check deadline passed
	if ch.Status == types.ChannelStatusDisputed {
		if uint64(ctx.BlockHeight()) <= ch.DisputeDeadline {
			return nil, types.ErrDisputePeriodActive
		}
		// Resolve dispute -- use disputed state
		dispute, hasDispute := m.Keeper.GetDispute(ctx, ch.ChannelId)
		if hasDispute && !dispute.Resolved {
			ch.Spent = dispute.DisputedSpent
			ch.Nonce = dispute.DisputedNonce
			dispute.Resolved = true
			dispute.Resolution = "provider_wins" // default: disputer's state accepted
			m.Keeper.SetDispute(ctx, dispute)
		}
	}

	// Calculate distribution
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	spent, _ := new(big.Int).SetString(ch.Spent, 10)
	if spent == nil {
		spent = new(big.Int)
	}
	senderRefund := new(big.Int).Sub(deposited, spent)
	receiverPayout := new(big.Int).Set(spent)

	// Transfer from module to sender (refund)
	if senderRefund.Sign() > 0 && m.Keeper.bankKeeper != nil {
		payerAddr, err := sdk.AccAddressFromBech32(ch.Payer)
		if err != nil {
			return nil, fmt.Errorf("invalid payer address: %w", err)
		}
		refundCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(senderRefund)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payerAddr, refundCoins); err != nil {
			return nil, fmt.Errorf("failed to refund payer: %w", err)
		}
	}

	// Transfer from module to receiver (payout)
	if receiverPayout.Sign() > 0 && m.Keeper.bankKeeper != nil {
		providerAddr, err := sdk.AccAddressFromBech32(ch.Provider)
		if err != nil {
			return nil, fmt.Errorf("invalid provider address: %w", err)
		}
		payoutCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(receiverPayout)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAddr, payoutCoins); err != nil {
			return nil, fmt.Errorf("failed to pay provider: %w", err)
		}
	}

	ch.Status = types.ChannelStatusClosed
	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"settle_channel",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("amount_to_sender", senderRefund.String()),
		sdk.NewAttribute("amount_to_receiver", receiverPayout.String()),
	))

	return &types.MsgSettleChannelResponse{
		AmountToSender:   senderRefund.String(),
		AmountToReceiver: receiverPayout.String(),
	}, nil
}

// CloseChannel cooperatively closes a channel.
func (m msgServer) CloseChannel(goCtx context.Context, msg *types.MsgCloseChannel) (*types.MsgCloseChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen {
		return nil, types.ErrChannelNotOpen
	}

	// Verify closer is payer or provider
	if msg.Closer != ch.Payer && msg.Closer != ch.Provider {
		return nil, types.ErrNotChannelParty
	}

	// Use final balance receiver as spent amount
	spent, ok := new(big.Int).SetString(msg.FinalBalanceReceiver, 10)
	if !ok {
		spent = new(big.Int)
	}
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	if spent.Cmp(deposited) > 0 {
		return nil, types.ErrSpentExceedsDeposit
	}

	payerRefund := new(big.Int).Sub(deposited, spent)
	providerPayout := new(big.Int).Set(spent)

	// Transfer from module to payer (refund)
	if payerRefund.Sign() > 0 && m.Keeper.bankKeeper != nil {
		payerAddr, err := sdk.AccAddressFromBech32(ch.Payer)
		if err != nil {
			return nil, fmt.Errorf("invalid payer address: %w", err)
		}
		refundCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(payerRefund)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payerAddr, refundCoins); err != nil {
			return nil, fmt.Errorf("failed to refund payer: %w", err)
		}
	}

	// Transfer from module to provider (payout)
	if providerPayout.Sign() > 0 && m.Keeper.bankKeeper != nil {
		providerAddr, err := sdk.AccAddressFromBech32(ch.Provider)
		if err != nil {
			return nil, fmt.Errorf("invalid provider address: %w", err)
		}
		payoutCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(providerPayout)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAddr, payoutCoins); err != nil {
			return nil, fmt.Errorf("failed to pay provider: %w", err)
		}
	}

	ch.Status = types.ChannelStatusClosed
	ch.Nonce = msg.Sequence
	ch.Spent = msg.FinalBalanceReceiver
	available := new(big.Int).Sub(deposited, spent)
	ch.Available = available.String()
	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"close_channel",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("payer_refund", payerRefund.String()),
		sdk.NewAttribute("provider_payout", providerPayout.String()),
	))

	return &types.MsgCloseChannelResponse{}, nil
}

// DepositChannel adds funds to an existing channel.
func (m msgServer) DepositChannel(goCtx context.Context, msg *types.MsgDepositChannel) (*types.MsgDepositChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen {
		return nil, types.ErrChannelNotOpen
	}
	if msg.Depositor != ch.Payer {
		return nil, types.ErrNotChannelParty.Wrap("only payer can deposit")
	}

	amount, ok := new(big.Int).SetString(msg.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid deposit amount")
	}

	// Transfer from depositor to module account
	depositorAddr, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		return nil, fmt.Errorf("invalid depositor address: %w", err)
	}
	depositCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amount)))
	if m.Keeper.bankKeeper != nil {
		if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, depositorAddr, types.ModuleName, depositCoins); err != nil {
			return nil, fmt.Errorf("failed to transfer deposit: %w", err)
		}
	}

	// Update channel balances
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	available, _ := new(big.Int).SetString(ch.Available, 10)
	deposited.Add(deposited, amount)
	available.Add(available, amount)
	ch.Deposited = deposited.String()
	ch.Available = available.String()

	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"deposit_channel",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("depositor", msg.Depositor),
		sdk.NewAttribute("amount", msg.Amount),
	))

	return &types.MsgDepositChannelResponse{}, nil
}

// UpdateState submits an off-chain state update using the legacy message format.
func (m msgServer) UpdateState(goCtx context.Context, msg *types.MsgUpdateState) (*types.MsgUpdateStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen {
		return nil, types.ErrChannelNotOpen
	}

	// Verify sender is payer or provider
	if msg.Sender != ch.Payer && msg.Sender != ch.Provider {
		return nil, types.ErrNotChannelParty
	}

	// Verify nonce is higher
	if msg.Update.Nonce <= ch.Nonce {
		return nil, types.ErrInvalidNonce
	}

	// Verify spent <= deposited
	spent, ok := new(big.Int).SetString(msg.Update.Spent, 10)
	if !ok {
		return nil, fmt.Errorf("invalid spent amount")
	}
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	if spent.Cmp(deposited) > 0 {
		return nil, types.ErrSpentExceedsDeposit
	}

	// Update channel state
	ch.Nonce = msg.Update.Nonce
	ch.Spent = msg.Update.Spent
	available := new(big.Int).Sub(deposited, spent)
	ch.Available = available.String()
	ch.LastStateHash = msg.Update.StateHash

	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"update_state",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("nonce", fmt.Sprintf("%d", msg.Update.Nonce)),
		sdk.NewAttribute("spent", msg.Update.Spent),
	))

	return &types.MsgUpdateStateResponse{}, nil
}

// ClaimExpired claims refund from an expired channel.
func (m msgServer) ClaimExpired(goCtx context.Context, msg *types.MsgClaimExpired) (*types.MsgClaimExpiredResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	ch, found := m.Keeper.GetChannel(ctx, msg.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen {
		return nil, types.ErrChannelNotOpen
	}

	// Verify claimer is payer
	if msg.Claimer != ch.Payer {
		return nil, types.ErrNotChannelParty.Wrap("only payer can claim expired channel")
	}

	// Verify channel is expired (past expires + grace period)
	expiryWithGrace := ch.ExpiresAtBlock + params.ExpiryGracePeriod
	if uint64(ctx.BlockHeight()) <= expiryWithGrace {
		return nil, types.ErrNotExpired
	}

	// Calculate distribution
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	spent, _ := new(big.Int).SetString(ch.Spent, 10)
	if spent == nil {
		spent = new(big.Int)
	}
	payerRefund := new(big.Int).Sub(deposited, spent)

	// Refund payer
	if payerRefund.Sign() > 0 && m.Keeper.bankKeeper != nil {
		payerAddr, err := sdk.AccAddressFromBech32(ch.Payer)
		if err != nil {
			return nil, fmt.Errorf("invalid payer address: %w", err)
		}
		refundCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(payerRefund)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payerAddr, refundCoins); err != nil {
			return nil, fmt.Errorf("failed to refund payer: %w", err)
		}
	}

	// Pay provider for spent amount
	if spent.Sign() > 0 && m.Keeper.bankKeeper != nil {
		providerAddr, err := sdk.AccAddressFromBech32(ch.Provider)
		if err != nil {
			return nil, fmt.Errorf("invalid provider address: %w", err)
		}
		payoutCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(spent)))
		if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAddr, payoutCoins); err != nil {
			return nil, fmt.Errorf("failed to pay provider: %w", err)
		}
	}

	ch.Status = types.ChannelStatusClosed
	m.Keeper.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"claim_expired",
		sdk.NewAttribute("channel_id", msg.ChannelId),
		sdk.NewAttribute("refunded", payerRefund.String()),
	))

	return &types.MsgClaimExpiredResponse{RefundedAmount: payerRefund.String()}, nil
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
			"channels_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)
	return &types.MsgUpdateParamsResponse{}, nil
}
