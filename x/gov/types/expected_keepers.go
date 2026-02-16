package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingKeeper defines the expected staking module keeper interface.
type StakingKeeper interface {
	GetValidatorStake(ctx context.Context, address string) (string, error)
}

// BankKeeper defines the expected bank module keeper interface for token transfers.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// UpgradePlan mirrors cosmossdk.io/x/upgrade/types.Plan to avoid a direct
// module-level import in the types package.
type UpgradePlan struct {
	Name   string `json:"name"`
	Height int64  `json:"height"`
	Info   string `json:"info"`
}

// UpgradeKeeper defines the expected upgrade module keeper interface.
// When a consensus-category LIP is accepted with an attached upgrade plan,
// the gov module calls ScheduleUpgrade to halt the chain at the target height.
type UpgradeKeeper interface {
	ScheduleUpgrade(ctx context.Context, plan UpgradePlan) error
}
