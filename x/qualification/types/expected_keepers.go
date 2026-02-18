package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ontologytypes "github.com/legible-money/legible/x/ontology/types"
	stakingtypes "github.com/legible-money/legible/x/staking/types"
)

// OntologyKeeper defines the expected ontology module keeper interface.
type OntologyKeeper interface {
	GetDomain(ctx sdk.Context, name string) (*ontologytypes.Domain, bool)
}

// StakingKeeper defines the expected staking module keeper interface.
type StakingKeeper interface {
	GetValidator(ctx sdk.Context, operatorAddr string) (*stakingtypes.LegibleValidator, bool)
	GetActiveValidatorSet(ctx sdk.Context) []*stakingtypes.LegibleValidator
}

// BankKeeper defines the expected bank keeper interface.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}
