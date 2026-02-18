package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"

	legibleauthtypes "github.com/legible-money/legible/x/auth/types"
)

// LegibleAuthKeeper defines the expected interface for the x/auth (Legible) keeper.
type LegibleAuthKeeper interface {
	GetAccount(ctx sdk.Context, address string) (*legibleauthtypes.LegibleAccount, bool)
	GetAccountByDID(ctx sdk.Context, did string) (*legibleauthtypes.LegibleAccount, bool)
	GetAddressForDID(ctx sdk.Context, did string) (string, bool)
}

// ICAControllerKeeper defines the expected interface for the ICA controller keeper.
type ICAControllerKeeper interface {
	RegisterInterchainAccount(ctx sdk.Context, connectionID, owner, version string) error
	GetInterchainAccountAddress(ctx sdk.Context, connectionID, portID string) (string, bool)
	GetOpenActiveChannel(ctx sdk.Context, connectionID, portID string) (string, bool)
	SendTx(ctx sdk.Context, chanCap *capabilitytypes.Capability, connectionID, portID string, icaPacketData icatypes.InterchainAccountPacketData, timeoutTimestamp uint64) (uint64, error)
}
