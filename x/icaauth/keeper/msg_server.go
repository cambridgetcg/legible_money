package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	legibleauthtypes "github.com/legible-money/legible/x/auth/types"
	"github.com/legible-money/legible/x/icaauth/types"
)

type msgServer struct {
	k Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

// RegisterInterchainAccount registers a new ICA on a remote chain.
// Only the operational key (or identity key) may call this — session keys are denied
// by the LegibleCapabilityDecorator in the ante handler.
func (ms *msgServer) RegisterInterchainAccount(goCtx context.Context, msg *types.MsgRegisterInterchainAccount) (*types.MsgRegisterInterchainAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify the sender has a Legible account
	lgmAccount, found := ms.k.legibleAuth.GetAccount(ctx, msg.Owner)
	if !found {
		return nil, types.ErrNoLegibleAccount
	}

	// Validate DID exists
	if err := legibleauthtypes.ValidateDID(lgmAccount.Did); err != nil {
		return nil, fmt.Errorf("invalid DID on account: %w", err)
	}

	// Check max accounts
	params := ms.k.GetParams(ctx)
	existing := ms.k.GetRemoteAccountsForDID(ctx, lgmAccount.Did)
	if uint32(len(existing)) >= params.MaxRemoteAccountsPerDid {
		return nil, types.ErrMaxAccountsReached
	}

	// Check for duplicate connection
	for _, acc := range existing {
		if acc.ConnectionId == msg.ConnectionId {
			return nil, types.ErrAlreadyRegistered
		}
	}

	// Register with the ICA controller
	version := msg.Version
	if version == "" {
		version = "" // Let ibc-go choose the default version
	}
	if err := ms.k.icaController.RegisterInterchainAccount(ctx, msg.ConnectionId, msg.Owner, version); err != nil {
		return nil, fmt.Errorf("failed to register ICA: %w", err)
	}

	// Store the pending registration
	remoteAccount := &types.RemoteAccount{
		ConnectionId:    msg.ConnectionId,
		PortId:          fmt.Sprintf("icacontroller-%s", msg.Owner),
		OwnerDid:        lgmAccount.Did,
		OwnerAddress:    msg.Owner,
		RegisteredBlock: uint64(ctx.BlockHeight()),
		AllowedMsgTypes: msg.AllowedMsgs,
		Active:          false, // becomes active when ICA channel opens
	}

	if err := ms.k.AddRemoteAccount(ctx, lgmAccount.Did, remoteAccount); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"register_interchain_account",
		sdk.NewAttribute("owner", msg.Owner),
		sdk.NewAttribute("did", lgmAccount.Did),
		sdk.NewAttribute("connection_id", msg.ConnectionId),
	))

	return &types.MsgRegisterInterchainAccountResponse{
		PortId: remoteAccount.PortId,
	}, nil
}

// SendICATransaction sends a transaction to a remote chain via ICA.
func (ms *msgServer) SendICATransaction(goCtx context.Context, msg *types.MsgSendICATransaction) (*types.MsgSendICATransactionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify the sender has a Legible account with this DID
	lgmAccount, found := ms.k.legibleAuth.GetAccount(ctx, msg.Owner)
	if !found {
		return nil, types.ErrNoLegibleAccount
	}

	// Check the ICA exists and is active
	remoteAccount, found := ms.k.GetRemoteAccountByConnection(ctx, lgmAccount.Did, msg.ConnectionId)
	if !found {
		return nil, types.ErrNotRegistered
	}
	if !remoteAccount.Active {
		return nil, fmt.Errorf("interchain account is not yet active")
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"send_ica_transaction",
		sdk.NewAttribute("owner", msg.Owner),
		sdk.NewAttribute("did", lgmAccount.Did),
		sdk.NewAttribute("connection_id", msg.ConnectionId),
	))

	return &types.MsgSendICATransactionResponse{}, nil
}
