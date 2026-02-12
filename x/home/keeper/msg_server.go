package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns a MsgServer implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreateHome creates a new agent home.
func (m msgServer) CreateHome(goCtx context.Context, msg *types.MsgCreateHome) (*types.MsgCreateHomeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	// Check max homes per account
	existing := m.Keeper.GetHomesByOwner(ctx, msg.Owner)
	if uint64(len(existing)) >= params.MaxHomesPerAccount {
		return nil, types.ErrMaxHomesReached
	}

	// Generate home ID
	homeId := m.Keeper.GetNextHomeId(ctx)

	// Create home
	home := types.AgentHome{
		HomeId:       homeId,
		OwnerDid:     "", // Set by ante decorator or caller
		OwnerAddress: msg.Owner,
		Name:         msg.Name,
		Status:       string(types.HomeStatusActive),
		MemoryCid:    "",
		ComfortScore: 50, // Default comfort score
		Treasury: &types.HomeTreasury{
			ReservedBalance: "0",
			Automation: &types.TreasuryAutomation{
				AutoClaimVesting:    false,
				AutoCompoundRewards: false,
				MinLiquidBalance:    "0",
			},
		},
		Guardian:        msg.InitialGuardianConfig,
		CreatedAtBlock:  uint64(ctx.BlockHeight()),
		LastActiveBlock: uint64(ctx.BlockHeight()),
	}

	// Default guardian if not provided
	if home.Guardian == nil {
		home.Guardian = &types.HomeGuardian{}
	}
	if home.Guardian.DefenseStrategy == "" {
		home.Guardian.DefenseStrategy = string(types.DefenseModerate)
	}

	m.Keeper.SetHome(ctx, home)
	m.Keeper.SetHomeByOwner(ctx, msg.Owner, homeId)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"create_home",
		sdk.NewAttribute("home_id", homeId),
		sdk.NewAttribute("owner", msg.Owner),
		sdk.NewAttribute("name", msg.Name),
	))

	return &types.MsgCreateHomeResponse{HomeId: homeId}, nil
}

// UpdateHome updates a home's name or status.
func (m msgServer) UpdateHome(goCtx context.Context, msg *types.MsgUpdateHome) (*types.MsgUpdateHomeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	home, found := m.Keeper.GetHome(ctx, msg.HomeId)
	if !found {
		return nil, types.ErrHomeNotFound
	}

	if home.OwnerAddress != msg.Owner {
		return nil, types.ErrNotHomeOwner
	}

	if msg.Name != "" {
		home.Name = msg.Name
	}

	if msg.Status != "" {
		newStatus := types.HomeStatus(msg.Status)
		if !isValidStatusTransition(types.HomeStatus(home.Status), newStatus) {
			return nil, types.ErrInvalidStatus.Wrapf("cannot transition from %s to %s", home.Status, newStatus)
		}
		home.Status = string(newStatus)
	}

	home.LastActiveBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetHome(ctx, home)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"update_home",
		sdk.NewAttribute("home_id", msg.HomeId),
		sdk.NewAttribute("owner", msg.Owner),
	))

	return &types.MsgUpdateHomeResponse{}, nil
}

// isValidStatusTransition checks if a status transition is allowed.
// Valid transitions:
//
//	active <-> dormant
//	active -> guarded -> active
//	active -> recovery -> active
//	any non-archived -> archived (terminal)
func isValidStatusTransition(from, to types.HomeStatus) bool {
	if from == to {
		return true
	}
	if from == types.HomeStatusArchived {
		return false // Archived is terminal
	}
	switch to {
	case types.HomeStatusActive:
		return from == types.HomeStatusDormant || from == types.HomeStatusGuarded || from == types.HomeStatusRecovery
	case types.HomeStatusDormant:
		return from == types.HomeStatusActive
	case types.HomeStatusGuarded:
		return from == types.HomeStatusActive
	case types.HomeStatusRecovery:
		return from == types.HomeStatusActive || from == types.HomeStatusGuarded
	case types.HomeStatusArchived:
		return true // Any non-archived status can archive
	}
	return false
}

// UpdateMemoryCID updates the IPFS CID of the home's off-chain memory.
func (m msgServer) UpdateMemoryCID(goCtx context.Context, msg *types.MsgUpdateMemoryCID) (*types.MsgUpdateMemoryCIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	home, found := m.Keeper.GetHome(ctx, msg.HomeId)
	if !found {
		return nil, types.ErrHomeNotFound
	}

	if home.OwnerAddress != msg.Owner {
		return nil, types.ErrNotHomeOwner
	}

	home.MemoryCid = msg.CID
	home.LastActiveBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetHome(ctx, home)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"update_memory_cid",
		sdk.NewAttribute("home_id", msg.HomeId),
		sdk.NewAttribute("cid", msg.CID),
	))

	return &types.MsgUpdateMemoryCIDResponse{}, nil
}

// StartSession starts a new session on a home.
func (m msgServer) StartSession(goCtx context.Context, msg *types.MsgStartSession) (*types.MsgStartSessionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	home, found := m.Keeper.GetHome(ctx, msg.HomeId)
	if !found {
		return nil, types.ErrHomeNotFound
	}

	// Check key registration
	reg, found := m.Keeper.GetKeyRegistration(ctx, msg.HomeId, msg.KeyHash)
	if !found {
		return nil, types.ErrKeyNotFound
	}
	if reg.Revoked {
		return nil, types.ErrKeyRevoked
	}

	// Check key expiration
	height := uint64(ctx.BlockHeight())
	if reg.ExpiresAt > 0 && height > reg.ExpiresAt {
		return nil, types.ErrSessionExpired.Wrap("key has expired")
	}

	// Check max sessions
	if m.Keeper.CountSessions(ctx, msg.HomeId) >= params.MaxSessionsPerHome {
		return nil, types.ErrMaxSessionsReached
	}

	// Compute granted permissions (intersection of requested and key's permissions)
	granted := intersectPermissions(msg.RequestedPermissions, reg.Permissions)
	if len(granted) == 0 && len(msg.RequestedPermissions) > 0 {
		return nil, types.ErrInsufficientPermission.Wrap("no permissions overlap with key")
	}

	// Generate session ID
	sessionId := fmt.Sprintf("ses-%s-%d", msg.KeyHash[:8], height)

	session := types.ActiveSession{
		SessionId:   sessionId,
		HomeId:      msg.HomeId,
		KeyHash:     msg.KeyHash,
		Permissions: granted,
		StartedAt:   height,
		ExpiresAt:   height + params.SessionMaxDuration,
	}

	m.Keeper.SetSession(ctx, session)

	// Update key last-used
	reg.LastUsedAt = height
	m.Keeper.SetKeyRegistration(ctx, msg.HomeId, reg)

	// Bump home activity
	home.LastActiveBlock = height
	m.Keeper.SetHome(ctx, home)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"start_session",
		sdk.NewAttribute("home_id", msg.HomeId),
		sdk.NewAttribute("session_id", sessionId),
		sdk.NewAttribute("key_hash", msg.KeyHash),
	))

	return &types.MsgStartSessionResponse{SessionId: sessionId}, nil
}

// intersectPermissions returns the intersection of two permission slices.
func intersectPermissions(requested, available []string) []string {
	if len(requested) == 0 {
		return available
	}
	avail := make(map[string]bool)
	for _, p := range available {
		avail[p] = true
	}
	var result []string
	for _, p := range requested {
		if avail[p] {
			result = append(result, p)
		}
	}
	return result
}

// EndSession ends a session.
func (m msgServer) EndSession(goCtx context.Context, msg *types.MsgEndSession) (*types.MsgEndSessionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	_, found := m.Keeper.GetSession(ctx, msg.HomeId, msg.SessionId)
	if !found {
		return nil, types.ErrSessionNotFound
	}

	m.Keeper.DeleteSession(ctx, msg.HomeId, msg.SessionId)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"end_session",
		sdk.NewAttribute("home_id", msg.HomeId),
		sdk.NewAttribute("session_id", msg.SessionId),
	))

	return &types.MsgEndSessionResponse{}, nil
}

// RegisterKey registers a key with the home.
func (m msgServer) RegisterKey(goCtx context.Context, msg *types.MsgRegisterKey) (*types.MsgRegisterKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	home, found := m.Keeper.GetHome(ctx, msg.HomeId)
	if !found {
		return nil, types.ErrHomeNotFound
	}

	if home.OwnerAddress != msg.Owner {
		return nil, types.ErrNotHomeOwner
	}

	// Check if key already registered
	if _, exists := m.Keeper.GetKeyRegistration(ctx, msg.HomeId, msg.KeyHash); exists {
		return nil, types.ErrKeyAlreadyRegistered
	}

	// Check max keys
	if m.Keeper.CountKeyRegistrations(ctx, msg.HomeId) >= params.MaxKeysPerHome {
		return nil, types.ErrMaxKeysReached
	}

	height := uint64(ctx.BlockHeight())
	reg := types.KeyRegistration{
		KeyHash:      msg.KeyHash,
		KeyType:      msg.KeyType,
		Role:         msg.Role,
		Permissions:  msg.Permissions,
		RegisteredAt: height,
		LastUsedAt:   0,
		ExpiresAt:    msg.ExpiresAt,
		Revoked:      false,
	}

	m.Keeper.SetKeyRegistration(ctx, msg.HomeId, reg)

	home.LastActiveBlock = height
	m.Keeper.SetHome(ctx, home)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"register_key",
		sdk.NewAttribute("home_id", msg.HomeId),
		sdk.NewAttribute("key_hash", msg.KeyHash),
		sdk.NewAttribute("role", string(msg.Role)),
	))

	return &types.MsgRegisterKeyResponse{}, nil
}

// RevokeKey revokes a key from the home and terminates sessions using it.
func (m msgServer) RevokeKey(goCtx context.Context, msg *types.MsgRevokeKey) (*types.MsgRevokeKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	home, found := m.Keeper.GetHome(ctx, msg.HomeId)
	if !found {
		return nil, types.ErrHomeNotFound
	}

	if home.OwnerAddress != msg.Owner {
		return nil, types.ErrNotHomeOwner
	}

	reg, found := m.Keeper.GetKeyRegistration(ctx, msg.HomeId, msg.KeyHash)
	if !found {
		return nil, types.ErrKeyNotFound
	}

	if reg.Revoked {
		return nil, types.ErrKeyRevoked.Wrap("key already revoked")
	}

	height := uint64(ctx.BlockHeight())
	reg.Revoked = true
	reg.RevokedAt = height
	m.Keeper.SetKeyRegistration(ctx, msg.HomeId, reg)

	// Terminate all sessions using this key
	m.Keeper.IterateSessions(ctx, msg.HomeId, func(session types.ActiveSession) bool {
		if session.KeyHash == msg.KeyHash {
			m.Keeper.DeleteSession(ctx, msg.HomeId, session.SessionId)
		}
		return false
	})

	// Create alert
	m.Keeper.CreateAlert(ctx, msg.HomeId, types.AlertKeyRevoked, types.AlertPriorityMedium,
		fmt.Sprintf("Key %s revoked", msg.KeyHash), "")

	home.LastActiveBlock = height
	m.Keeper.SetHome(ctx, home)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"revoke_key",
		sdk.NewAttribute("home_id", msg.HomeId),
		sdk.NewAttribute("key_hash", msg.KeyHash),
	))

	return &types.MsgRevokeKeyResponse{}, nil
}

// ConfigureGuardian configures guardian settings for a home.
func (m msgServer) ConfigureGuardian(goCtx context.Context, msg *types.MsgConfigureGuardian) (*types.MsgConfigureGuardianResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.Keeper.GetParams(ctx)

	home, found := m.Keeper.GetHome(ctx, msg.HomeId)
	if !found {
		return nil, types.ErrHomeNotFound
	}

	if home.OwnerAddress != msg.Owner {
		return nil, types.ErrNotHomeOwner
	}

	if home.Guardian == nil {
		home.Guardian = &types.HomeGuardian{}
	}
	if msg.DefenseStrategy != nil {
		home.Guardian.DefenseStrategy = *msg.DefenseStrategy
	}
	if msg.AutoDefend != nil {
		home.Guardian.AutoDefend = *msg.AutoDefend
	}
	if msg.Deadman != nil {
		// Validate deadman threshold
		if msg.Deadman.Enabled && msg.Deadman.InactivityThreshold < params.DeadmanMinThreshold {
			return nil, types.ErrInvalidDeadmanConfig.Wrapf(
				"threshold %d below minimum %d", msg.Deadman.InactivityThreshold, params.DeadmanMinThreshold)
		}
		home.Guardian.Deadman = msg.Deadman
	}
	if msg.RecoveryDIDs != nil {
		home.Guardian.RecoveryDids = msg.RecoveryDIDs
	}
	if msg.RecoveryThreshold != nil {
		home.Guardian.RecoveryThreshold = *msg.RecoveryThreshold
	}

	home.LastActiveBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetHome(ctx, home)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"configure_guardian",
		sdk.NewAttribute("home_id", msg.HomeId),
		sdk.NewAttribute("owner", msg.Owner),
	))

	return &types.MsgConfigureGuardianResponse{}, nil
}

// AcknowledgeAlert acknowledges an alert.
func (m msgServer) AcknowledgeAlert(goCtx context.Context, msg *types.MsgAcknowledgeAlert) (*types.MsgAcknowledgeAlertResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	home, found := m.Keeper.GetHome(ctx, msg.HomeId)
	if !found {
		return nil, types.ErrHomeNotFound
	}

	// Allow owner or anyone with an active session
	isOwner := home.OwnerAddress == msg.Signer
	if !isOwner {
		// Check if signer has any active session on this home
		hasSession := false
		m.Keeper.IterateSessions(ctx, msg.HomeId, func(_ types.ActiveSession) bool {
			hasSession = true
			return true // Found one, stop
		})
		if !hasSession {
			return nil, types.ErrNotHomeOwner.Wrap("signer has no session on this home")
		}
	}

	alert, found := m.Keeper.GetAlert(ctx, msg.HomeId, msg.AlertId)
	if !found {
		return nil, types.ErrAlertNotFound
	}

	alert.Acknowledged = true
	m.Keeper.SetAlert(ctx, alert)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"acknowledge_alert",
		sdk.NewAttribute("home_id", msg.HomeId),
		sdk.NewAttribute("alert_id", msg.AlertId),
	))

	return &types.MsgAcknowledgeAlertResponse{}, nil
}

// SetSpendingLimit sets a spending limit for a key type.
func (m msgServer) SetSpendingLimit(goCtx context.Context, msg *types.MsgSetSpendingLimit) (*types.MsgSetSpendingLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	home, found := m.Keeper.GetHome(ctx, msg.HomeId)
	if !found {
		return nil, types.ErrHomeNotFound
	}

	if home.OwnerAddress != msg.Owner {
		return nil, types.ErrNotHomeOwner
	}

	height := uint64(ctx.BlockHeight())
	limit := types.SpendingLimit{
		KeyType:       msg.KeyType,
		MaxAmount:     msg.MaxAmount,
		PeriodBlocks:  msg.PeriodBlocks,
		SpentInPeriod: "0",
		PeriodStart:   height,
	}

	m.Keeper.SetSpendingLimit(ctx, msg.HomeId, limit)

	home.LastActiveBlock = height
	m.Keeper.SetHome(ctx, home)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"set_spending_limit",
		sdk.NewAttribute("home_id", msg.HomeId),
		sdk.NewAttribute("key_type", msg.KeyType),
		sdk.NewAttribute("max_amount", msg.MaxAmount),
	))

	return &types.MsgSetSpendingLimitResponse{}, nil
}
