package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/channels/types"
)

// Keeper manages the channels module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService
	bankKeeper   types.BankKeeper
	authority    string
}

// NewKeeper creates a new channels module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bankKeeper,
		authority:    authority,
	}
}

// prefixEndBytes returns the end key for prefix iteration.
// It increments the last byte of the prefix. If overflow, returns nil (iterate to end).
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	k.SetParams(ctx, *gs.Params)
	for _, ch := range gs.Channels {
		k.SetChannel(ctx, *ch)
	}
	for _, d := range gs.Disputes {
		k.SetDispute(ctx, *d)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var channels []*types.PaymentChannel
	k.IterateChannels(ctx, func(ch types.PaymentChannel) bool {
		c := ch
		channels = append(channels, &c)
		return false
	})

	var disputes []*types.ChannelDispute
	k.IterateDisputes(ctx, func(d types.ChannelDispute) bool {
		dd := d
		disputes = append(disputes, &dd)
		return false
	})

	params := k.GetParams(ctx)
	gs := &types.GenesisState{
		Params:   &params,
		Channels: channels,
		Disputes: disputes,
	}
	return gs
}

// GetChannelInfo returns channel details for cross-module queries.
// Returns payer, provider, available balance, status, and found.
// Accepts context.Context for cross-module interface compatibility.
func (k Keeper) GetChannelInfo(goCtx context.Context, channelId string) (payer, provider, available, status string, found bool) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ch, ok := k.GetChannel(ctx, channelId)
	if !ok {
		return "", "", "", "", false
	}
	return ch.Payer, ch.Provider, ch.Available, ch.Status, true
}

// SpendFromChannel debits amount from a channel's available balance and transfers
// the funds from the channels module account to the specified recipient module account.
// Accepts context.Context for cross-module interface compatibility.
func (k Keeper) SpendFromChannel(goCtx context.Context, channelId string, amount string, recipientModule string) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ch, found := k.GetChannel(ctx, channelId)
	if !found {
		return types.ErrChannelNotFound
	}
	if ch.Status != types.ChannelStatusOpen {
		return types.ErrChannelNotOpen
	}

	spendAmt, ok := new(big.Int).SetString(amount, 10)
	if !ok || spendAmt.Sign() <= 0 {
		return fmt.Errorf("invalid spend amount")
	}

	avail, _ := new(big.Int).SetString(ch.Available, 10)
	if avail == nil || spendAmt.Cmp(avail) > 0 {
		return types.ErrSpentExceedsDeposit.Wrap("insufficient channel balance")
	}

	// Update channel accounting
	spent, _ := new(big.Int).SetString(ch.Spent, 10)
	if spent == nil {
		spent = new(big.Int)
	}
	spent.Add(spent, spendAmt)
	avail.Sub(avail, spendAmt)
	ch.Spent = spent.String()
	ch.Available = avail.String()
	ch.Nonce++
	k.SetChannel(ctx, ch)

	// Transfer from channels module to recipient module
	if k.bankKeeper != nil {
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(spendAmt)))
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, recipientModule, coins); err != nil {
			return fmt.Errorf("failed to transfer from channel: %w", err)
		}
	}

	return nil
}

// GetExpiredChannels returns channels in closing or disputed status past their deadline.
func (k Keeper) GetExpiredChannels(ctx sdk.Context, currentBlock uint64) []types.PaymentChannel {
	var channels []types.PaymentChannel
	k.IterateChannels(ctx, func(ch types.PaymentChannel) bool {
		if (ch.Status == types.ChannelStatusClosing || ch.Status == types.ChannelStatusDisputed) &&
			ch.DisputeDeadline > 0 && currentBlock > ch.DisputeDeadline {
			channels = append(channels, ch)
		}
		return false
	})
	return channels
}

// AutoSettleChannel distributes funds and closes a channel that has passed its deadline.
// For disputed channels, uses the disputed state. For closing channels, uses the last agreed state.
func (k Keeper) AutoSettleChannel(ctx sdk.Context, ch types.PaymentChannel) {
	deposited, _ := new(big.Int).SetString(ch.Deposited, 10)
	spent, _ := new(big.Int).SetString(ch.Spent, 10)
	if spent == nil {
		spent = new(big.Int)
	}

	// If disputed, resolve dispute and use disputed state
	if ch.Status == types.ChannelStatusDisputed {
		dispute, hasDispute := k.GetDispute(ctx, ch.ChannelId)
		if hasDispute && !dispute.Resolved {
			disputedSpent, ok := new(big.Int).SetString(dispute.DisputedSpent, 10)
			if ok {
				spent = disputedSpent
			}
			ch.Spent = dispute.DisputedSpent
			ch.Nonce = dispute.DisputedNonce
			dispute.Resolved = true
			dispute.Resolution = "auto_settled"
			k.SetDispute(ctx, dispute)
		}
	}

	senderRefund := new(big.Int).Sub(deposited, spent)
	receiverPayout := new(big.Int).Set(spent)

	// Transfer from module to payer (refund)
	if senderRefund.Sign() > 0 && k.bankKeeper != nil {
		payerAddr, err := sdk.AccAddressFromBech32(ch.Payer)
		if err == nil {
			refundCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(senderRefund)))
			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payerAddr, refundCoins)
		}
	}

	// Transfer from module to provider (payout)
	if receiverPayout.Sign() > 0 && k.bankKeeper != nil {
		providerAddr, err := sdk.AccAddressFromBech32(ch.Provider)
		if err == nil {
			payoutCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(receiverPayout)))
			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAddr, payoutCoins)
		}
	}

	ch.Status = types.ChannelStatusClosed
	ch.DisputeDeadline = 0
	available := new(big.Int).Sub(deposited, spent)
	ch.Available = available.String()
	k.SetChannel(ctx, ch)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"auto_settle_channel",
		sdk.NewAttribute("channel_id", ch.ChannelId),
		sdk.NewAttribute("amount_to_payer", senderRefund.String()),
		sdk.NewAttribute("amount_to_provider", receiverPayout.String()),
	))
}

// ExportGenesisJSON exports genesis as raw JSON.
func (k Keeper) ExportGenesisJSON(ctx sdk.Context) json.RawMessage {
	gs := k.ExportGenesis(ctx)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal channels genesis: %v", err))
	}
	return bz
}
