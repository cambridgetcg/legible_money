package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/discovery/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// RegisterAgent registers a new agent profile.
func (k msgServer) RegisterAgent(goCtx context.Context, msg *types.MsgRegisterAgent) (*types.MsgRegisterAgentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	// Check if already registered
	if _, found := k.GetProfile(ctx, msg.Address); found {
		return nil, types.ErrAgentExists
	}

	// Validate capability count
	if uint32(len(msg.Capabilities)) > params.MaxCapabilitiesPerAgent {
		return nil, types.ErrMaxCapabilities
	}

	// Validate and deduct stake
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(msg.Stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid stake amount: %s", msg.Stake)
	}

	minStake := new(big.Int)
	minStake.SetString(params.MinRegistrationStake, 10)
	if stakeAmount.Cmp(minStake) < 0 {
		return nil, types.ErrInsufficientStake
	}

	addr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, sdk.NewCoins(stakeCoin)); err != nil {
		return nil, fmt.Errorf("insufficient funds for stake: %w", err)
	}

	currentHeight := uint64(ctx.BlockHeight())

	profile := types.AgentProfile{
		Address:           msg.Address,
		Did:               msg.Did,
		DisplayName:       msg.DisplayName,
		Capabilities:      msg.Capabilities,
		Domains:           msg.Domains,
		Status:            "active",
		ReputationScore:   500000, // start at 0.5
		RegisteredAtBlock: currentHeight,
		LastActiveBlock:   currentHeight,
		Stake:             msg.Stake,
		Description:       msg.Description,
		Metadata:          msg.Metadata,
	}

	k.SetProfile(ctx, profile)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent("agent_registered",
			sdk.NewAttribute("address", msg.Address),
			sdk.NewAttribute("display_name", msg.DisplayName),
			sdk.NewAttribute("domains", fmt.Sprintf("%v", msg.Domains)),
		),
	})

	k.Logger(ctx).Info("agent registered",
		"address", msg.Address,
		"display_name", msg.DisplayName,
	)

	return &types.MsgRegisterAgentResponse{}, nil
}

// UpdateProfile updates an agent's profile metadata.
func (k msgServer) UpdateProfile(goCtx context.Context, msg *types.MsgUpdateProfile) (*types.MsgUpdateProfileResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	profile, found := k.GetProfile(ctx, msg.Address)
	if !found {
		return nil, types.ErrAgentNotFound
	}

	if msg.DisplayName != "" {
		profile.DisplayName = msg.DisplayName
	}
	if msg.Description != "" {
		profile.Description = msg.Description
	}
	if msg.Metadata != "" {
		profile.Metadata = msg.Metadata
	}

	k.SetProfile(ctx, profile)

	return &types.MsgUpdateProfileResponse{}, nil
}

// AddCapability adds a capability to an agent's profile.
func (k msgServer) AddCapability(goCtx context.Context, msg *types.MsgAddCapability) (*types.MsgAddCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	profile, found := k.GetProfile(ctx, msg.Address)
	if !found {
		return nil, types.ErrAgentNotFound
	}
	if profile.Status != "active" {
		return nil, types.ErrAgentInactive
	}

	// Check for duplicates
	for _, cap := range profile.Capabilities {
		if cap.CapabilityType == msg.Capability.CapabilityType {
			return nil, types.ErrCapabilityExists
		}
	}

	// Check limit
	if uint32(len(profile.Capabilities)+1) > params.MaxCapabilitiesPerAgent {
		return nil, types.ErrMaxCapabilities
	}

	// Remove old indexes before mutation
	k.DeleteProfile(ctx, profile)

	profile.Capabilities = append(profile.Capabilities, msg.Capability)
	k.SetProfile(ctx, profile)

	return &types.MsgAddCapabilityResponse{}, nil
}

// RemoveCapability removes a capability from an agent's profile.
func (k msgServer) RemoveCapability(goCtx context.Context, msg *types.MsgRemoveCapability) (*types.MsgRemoveCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	profile, found := k.GetProfile(ctx, msg.Address)
	if !found {
		return nil, types.ErrAgentNotFound
	}

	// Find and remove capability
	foundCap := false
	newCaps := make([]*types.AgentCapability, 0, len(profile.Capabilities))
	for _, cap := range profile.Capabilities {
		if cap.CapabilityType == msg.CapabilityType {
			foundCap = true
			continue
		}
		newCaps = append(newCaps, cap)
	}
	if !foundCap {
		return nil, types.ErrCapabilityNotFound
	}

	// Remove old indexes before mutation
	k.DeleteProfile(ctx, profile)

	profile.Capabilities = newCaps
	k.SetProfile(ctx, profile)

	return &types.MsgRemoveCapabilityResponse{}, nil
}

// DeregisterAgent removes an agent and returns stake.
func (k msgServer) DeregisterAgent(goCtx context.Context, msg *types.MsgDeregisterAgent) (*types.MsgDeregisterAgentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	profile, found := k.GetProfile(ctx, msg.Address)
	if !found {
		return nil, types.ErrAgentNotFound
	}

	// Refund stake
	refundedStr := "0"
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(profile.Stake, 10); ok && stakeAmount.Sign() > 0 {
		addr, err := sdk.AccAddressFromBech32(profile.Address)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %w", err)
		}
		refundCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(refundCoin)); err != nil {
			return nil, fmt.Errorf("failed to refund stake: %w", err)
		}
		refundedStr = stakeAmount.String()
	}

	k.DeleteProfile(ctx, profile)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent("agent_deregistered",
			sdk.NewAttribute("address", msg.Address),
			sdk.NewAttribute("refunded", refundedStr),
		),
	})

	return &types.MsgDeregisterAgentResponse{RefundedStake: refundedStr}, nil
}

// Heartbeat signals that an agent is still active.
func (k msgServer) Heartbeat(goCtx context.Context, msg *types.MsgHeartbeat) (*types.MsgHeartbeatResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	profile, found := k.GetProfile(ctx, msg.Address)
	if !found {
		return nil, types.ErrAgentNotFound
	}

	currentHeight := uint64(ctx.BlockHeight())
	profile.LastActiveBlock = currentHeight

	// Reactivate if expired
	if profile.Status == "expired" {
		profile.Status = "active"
	}

	k.SetProfile(ctx, profile)

	return &types.MsgHeartbeatResponse{}, nil
}

// UpdateParams updates module parameters.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != k.GetAuthority() {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	k.SetParams(ctx, *msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
