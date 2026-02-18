package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	gogoproto "github.com/cosmos/gogoproto/proto"

	legibleauthtypes "github.com/legible-money/legible/x/auth/types"
	"github.com/legible-money/legible/x/icaauth/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
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

	// P0-4: Enforce per-account AllowedMsgTypes and global AllowedHostMsgTypes.
	// Deserialize the CosmosTx to extract message type URLs.
	var cosmosTx icatypes.CosmosTx
	if err := gogoproto.Unmarshal(msg.MsgBytes, &cosmosTx); err != nil {
		return nil, fmt.Errorf("failed to decode ICA messages: %w", err)
	}
	if len(cosmosTx.Messages) == 0 {
		return nil, fmt.Errorf("ICA transaction must contain at least one message")
	}

	params := ms.k.GetParams(ctx)
	for _, anyMsg := range cosmosTx.Messages {
		typeURL := anyMsg.TypeUrl

		// Check per-account allowlist (if set)
		if len(remoteAccount.AllowedMsgTypes) > 0 {
			if !containsStr(remoteAccount.AllowedMsgTypes, typeURL) {
				return nil, types.ErrMsgTypeNotAllowed.Wrapf(
					"message type %s not in account allowlist", typeURL)
			}
		}

		// Check global allowlist
		if !containsStr(params.AllowedHostMsgTypes, typeURL) {
			return nil, types.ErrMsgTypeNotAllowed.Wrapf(
				"message type %s not in global AllowedHostMsgTypes", typeURL)
		}
	}

	// P0-5: Actually send the ICA transaction.
	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: msg.MsgBytes,
	}

	seq, err := ms.k.icaController.SendTx(
		ctx, nil, msg.ConnectionId, remoteAccount.PortId, packetData, msg.TimeoutNanos,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send ICA transaction: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"send_ica_transaction",
		sdk.NewAttribute("owner", msg.Owner),
		sdk.NewAttribute("did", lgmAccount.Did),
		sdk.NewAttribute("connection_id", msg.ConnectionId),
		sdk.NewAttribute("sequence", fmt.Sprintf("%d", seq)),
	))

	return &types.MsgSendICATransactionResponse{Sequence: seq}, nil
}

// containsStr checks if a string slice contains a specific string.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}


// UpdateParams handles MsgUpdateParams -- governance-gated parameter update.
func (ms *msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", ms.k.GetAuthority(), msg.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := ms.k.SetParams(ctx, *msg.Params); err != nil {
		return nil, fmt.Errorf("failed to set params: %w", err)
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"icaauth_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)
	return &types.MsgUpdateParamsResponse{}, nil
}
