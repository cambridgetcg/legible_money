package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"

	icaauthtypes "github.com/legible-money/legible/x/icaauth/types"
)

// ICAControllerAdapter wraps the ibc-go ICA controller keeper to satisfy
// the x/icaauth expected keeper interface.
type ICAControllerAdapter struct {
	keeper icacontrollerkeeper.Keeper
}

func NewICAControllerAdapter(keeper icacontrollerkeeper.Keeper) *ICAControllerAdapter {
	return &ICAControllerAdapter{keeper: keeper}
}

var _ icaauthtypes.ICAControllerKeeper = (*ICAControllerAdapter)(nil)

func (a *ICAControllerAdapter) RegisterInterchainAccount(ctx sdk.Context, connectionID, owner, version string) error {
	return a.keeper.RegisterInterchainAccount(ctx, connectionID, owner, version)
}

func (a *ICAControllerAdapter) GetInterchainAccountAddress(ctx sdk.Context, connectionID, portID string) (string, bool) {
	return a.keeper.GetInterchainAccountAddress(ctx, connectionID, portID)
}

func (a *ICAControllerAdapter) GetOpenActiveChannel(ctx sdk.Context, connectionID, portID string) (string, bool) {
	return a.keeper.GetOpenActiveChannel(ctx, connectionID, portID)
}

func (a *ICAControllerAdapter) SendTx(ctx sdk.Context, chanCap *capabilitytypes.Capability, connectionID, portID string, icaPacketData icatypes.InterchainAccountPacketData, timeoutTimestamp uint64) (uint64, error) {
	return a.keeper.SendTx(ctx, chanCap, connectionID, portID, icaPacketData, timeoutTimestamp)
}
