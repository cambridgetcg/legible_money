package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected Cosmos SDK bank keeper.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
}

// StakingKeeper defines the expected staking keeper (used for epoch detection).
type StakingKeeper interface {
	GetLastValidatorPower(ctx context.Context, operator sdk.ValAddress) (int64, error)
}

// HomeKeeper defines the expected home module interface.
// Used for auto-linking homes to partnerships and syncing freeze status.
type HomeKeeper interface {
	// GetHomesByOwner returns home IDs for an owner address.
	GetHomesByOwner(ctx sdk.Context, ownerAddr string) []string
	// GetHomeGuardianAddress returns the guardian_address for a home.
	GetHomeGuardianAddress(ctx sdk.Context, homeId string) string
	// LinkHomeToPartnership sets the partnership_id on a home.
	LinkHomeToPartnership(ctx sdk.Context, homeId string, partnershipId string)
	// SetHomeStatus sets the status of a home.
	SetHomeStatus(ctx sdk.Context, homeId string, status string)
	// GetHomeByPartnership returns the home ID linked to a partnership.
	GetHomeByPartnership(ctx sdk.Context, partnershipId string) (string, bool)
}
