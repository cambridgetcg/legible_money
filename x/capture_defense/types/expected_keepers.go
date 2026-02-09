package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingKeeper defines the expected interface for the legible staking module.
type StakingKeeper interface {
	GetEffectiveStake(ctx context.Context, operatorAddr string) uint64
	GetTotalStake(ctx context.Context) uint64
	SlashValidator(ctx context.Context, operatorAddr string, amount string, reason string) error
}

// OntologyKeeper defines the expected interface for the ontology module.
type OntologyKeeper interface {
	GetDomainDepth(ctx sdk.Context, domain string) (uint32, bool)
	GetDomainStratum(ctx sdk.Context, domain string) (uint32, bool)
	DomainExists(ctx sdk.Context, domain string) bool
	GetParentDomain(ctx sdk.Context, domain string) (string, bool)
	GetStratumName(ctx sdk.Context, stratum uint32) string
}

// BankKeeper defines the expected bank keeper interface.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}
