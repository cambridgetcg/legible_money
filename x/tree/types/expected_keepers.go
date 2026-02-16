package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// ChannelsKeeper defines the expected channels module interface for channel-gated service calls.
type ChannelsKeeper interface {
	// GetChannelInfo returns channel details. Returns payer, provider, available balance, status, and found.
	GetChannelInfo(ctx context.Context, channelId string) (payer, provider, available, status string, found bool)
	// SpendFromChannel debits amount from a channel's available balance and transfers
	// the funds from the channels module account to the recipient module account.
	SpendFromChannel(ctx context.Context, channelId string, amount string, recipientModule string) error
}
