package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected account keeper interface.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected bank keeper interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}

// StakingKeeper defines the expected staking keeper interface.
// The knowledge module uses this to query validator info for VRF selection.
type StakingKeeper interface {
	// GetActiveValidatorInfos returns all active validators as ValidatorInfo structs.
	GetActiveValidatorInfos(ctx context.Context) []ValidatorInfo

	// GetValidatorInfo returns info for a single validator by operator address.
	GetValidatorInfo(ctx context.Context, operatorAddr string) (ValidatorInfo, bool)

	// GetEffectiveStake returns the effective selection stake for a validator,
	// accounting for virtual stake at lower tiers.
	GetEffectiveStake(ctx context.Context, operatorAddr string) uint64

	// GetTotalStake returns the total stake across all active validators.
	GetTotalStake(ctx context.Context) uint64

	// RecordVerification records a verification outcome for a validator.
	RecordVerification(ctx context.Context, operatorAddr string, correct bool) error

	// SlashValidator slashes a validator's stake.
	SlashValidator(ctx context.Context, operatorAddr string, amount string, reason string) error
}
