package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/auth/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// RotateKey handles key rotation for operational or session keys
func (ms msgServer) RotateKey(goCtx sdk.Context, msg *types.MsgRotateKey) (*types.MsgRotateKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get account
	account, found := ms.GetAccount(ctx, msg.Sender)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	// Check if account is frozen
	if account.Flags.Frozen {
		return nil, types.ErrAccountFrozen
	}

	// Check cooldown period
	params := ms.GetParams(ctx)
	lastRotation := ms.GetLastRotation(ctx, msg.Sender)
	currentHeight := uint64(ctx.BlockHeight())

	if lastRotation > 0 && currentHeight < lastRotation+params.KeyRotationCooldown {
		return nil, types.ErrKeyRotationCooldown
	}

	// TODO: Verify signature from old key (msg.Signature)
	// This would require access to the old key's public key and crypto verification

	// Update key based on type
	switch msg.KeyType {
	case "operational":
		account.OperationalKeyHash = msg.NewKeyHash
		account.OperationalKeyVersion++
		ms.SetLastRotation(ctx, msg.Sender, currentHeight)

	case "session":
		// For session keys, we don't update the account directly
		// Session keys are managed separately via CreateSession/RevokeSession
		return nil, types.ErrInvalidKeyType

	default:
		return nil, types.ErrInvalidKeyType
	}

	// Update account
	account.LastActiveBlock = currentHeight
	ms.SetAccount(ctx, account)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"key_rotated",
			sdk.NewAttribute("sender", msg.Sender),
			sdk.NewAttribute("key_type", msg.KeyType),
			sdk.NewAttribute("version", fmt.Sprintf("%d", account.OperationalKeyVersion)),
		),
	)

	return &types.MsgRotateKeyResponse{}, nil
}

// CreateSession creates a new session key with limited capabilities
func (ms msgServer) CreateSession(goCtx sdk.Context, msg *types.MsgCreateSession) (*types.MsgCreateSessionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get account
	account, found := ms.GetAccount(ctx, msg.Owner)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	// Check if account is frozen
	if account.Flags.Frozen {
		return nil, types.ErrAccountFrozen
	}

	// Get params
	params := ms.GetParams(ctx)

	// Check max session duration
	if msg.Duration > params.MaxSessionDuration {
		return nil, types.ErrSessionDurationExceeded
	}

	// Check max session keys
	activeCount := ms.CountSessionKeys(ctx, msg.Owner)
	if activeCount >= params.MaxSessionKeys {
		return nil, types.ErrMaxSessionKeys
	}

	// Calculate expiration
	currentHeight := uint64(ctx.BlockHeight())
	expiresAt := currentHeight + msg.Duration

	// Create session key
	session := types.SessionKey{
		KeyHash:        msg.SessionKeyHash,
		Owner:          msg.Owner,
		Capabilities:   msg.Capabilities,
		ExpiresAtBlock: expiresAt,
		CreatedAtBlock: currentHeight,
	}

	// Store session key
	ms.SetSessionKey(ctx, session)

	// Update account
	account.SessionKeyCount++
	account.LastActiveBlock = currentHeight
	ms.SetAccount(ctx, account)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"session_created",
			sdk.NewAttribute("owner", msg.Owner),
			sdk.NewAttribute("key_hash", msg.SessionKeyHash),
			sdk.NewAttribute("expires_at", fmt.Sprintf("%d", expiresAt)),
		),
	)

	return &types.MsgCreateSessionResponse{
		ExpiresAtBlock: expiresAt,
	}, nil
}

// RevokeSession revokes an existing session key
func (ms msgServer) RevokeSession(goCtx sdk.Context, msg *types.MsgRevokeSession) (*types.MsgRevokeSessionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get account
	account, found := ms.GetAccount(ctx, msg.Owner)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	// Check if session exists
	_, found = ms.GetSessionKey(ctx, msg.Owner, msg.SessionKeyHash)
	if !found {
		return nil, types.ErrSessionNotFound
	}

	// Delete session key
	ms.DeleteSessionKey(ctx, msg.Owner, msg.SessionKeyHash)

	// Update account
	if account.SessionKeyCount > 0 {
		account.SessionKeyCount--
	}
	account.LastActiveBlock = uint64(ctx.BlockHeight())
	ms.SetAccount(ctx, account)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"session_revoked",
			sdk.NewAttribute("owner", msg.Owner),
			sdk.NewAttribute("key_hash", msg.SessionKeyHash),
		),
	)

	return &types.MsgRevokeSessionResponse{}, nil
}
