package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capturedefensetypes "github.com/legible-money/legible/x/capture_defense/types"
)

// CaptureDefenseKeeper defines the expected interface from the capture defense module.
type CaptureDefenseKeeper interface {
	GetCaptureMetrics(ctx sdk.Context, domain string) (capturedefensetypes.CaptureMetrics, bool)
	AnalyzeCaptureRisk(ctx sdk.Context, domain string, currentBlock uint64) capturedefensetypes.CaptureMetrics
}

// StakingKeeper defines the expected staking keeper interface.
type StakingKeeper interface {
	GetEffectiveStake(ctx context.Context, operatorAddr string) uint64
	SlashValidator(ctx context.Context, operatorAddr string, amount string, reason string) error
}

// BankKeeper defines the expected bank keeper interface.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}
