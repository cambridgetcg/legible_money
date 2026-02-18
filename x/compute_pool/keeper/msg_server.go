package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/compute_pool/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns a MsgServer implementation for the compute_pool module.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// RegisterComputeProvider registers a new compute infrastructure provider.
func (k msgServer) RegisterComputeProvider(goCtx context.Context, msg *types.MsgRegisterComputeProvider) (*types.MsgRegisterComputeProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.RegisterProvider(ctx, msg); err != nil {
		return nil, err
	}

	k.Logger(ctx).Info("registered compute provider",
		"address", msg.Sender,
		"service_type", msg.ServiceType,
		"price_per_cu", msg.PricePerCu,
	)

	return &types.MsgRegisterComputeProviderResponse{}, nil
}

// UpdateProvider updates a provider's configuration.
func (k msgServer) UpdateProvider(goCtx context.Context, msg *types.MsgUpdateProvider) (*types.MsgUpdateProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.UpdateProviderState(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgUpdateProviderResponse{}, nil
}

// DeregisterProvider begins provider unbonding.
func (k msgServer) DeregisterProvider(goCtx context.Context, msg *types.MsgDeregisterProvider) (*types.MsgDeregisterProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.DeregisterProviderState(ctx, msg.Sender); err != nil {
		return nil, err
	}

	k.Logger(ctx).Info("provider unbonding started", "address", msg.Sender)

	return &types.MsgDeregisterProviderResponse{}, nil
}

// ProviderHeartbeat records a provider liveness proof.
func (k msgServer) ProviderHeartbeat(goCtx context.Context, msg *types.MsgProviderHeartbeat) (*types.MsgProviderHeartbeatResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.ProcessHeartbeat(ctx, msg.Sender); err != nil {
		return nil, err
	}

	return &types.MsgProviderHeartbeatResponse{}, nil
}

// RedeemCredits spends compute credits against a provider.
func (k msgServer) RedeemCredits(goCtx context.Context, msg *types.MsgRedeemCredits) (*types.MsgRedeemCreditsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ticket, err := k.CreateRedemptionTicket(ctx, msg.Sender, msg.Provider, msg.Cu)
	if err != nil {
		return nil, err
	}

	k.Logger(ctx).Info("compute credits redeemed",
		"ticket_id", ticket.Id,
		"validator", msg.Sender,
		"provider", msg.Provider,
		"cu_amount", msg.Cu,
	)

	return &types.MsgRedeemCreditsResponse{TicketId: fmt.Sprintf("%d", ticket.Id)}, nil
}

// ConfirmUsage confirms that compute was delivered for a redemption ticket.
func (k msgServer) ConfirmUsage(goCtx context.Context, msg *types.MsgConfirmUsage) (*types.MsgConfirmUsageResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ticketId, err := strconv.ParseUint(msg.TicketId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket_id: %w", err)
	}

	if err := k.ConfirmRedemption(ctx, msg.Sender, ticketId, msg.ActualCu); err != nil {
		return nil, err
	}

	k.Logger(ctx).Info("compute usage confirmed",
		"ticket_id", msg.TicketId,
		"actual_cu", msg.ActualCu,
	)

	return &types.MsgConfirmUsageResponse{}, nil
}

// DisputeUsage disputes a redemption ticket (compute not delivered).
func (k msgServer) DisputeUsage(goCtx context.Context, msg *types.MsgDisputeUsage) (*types.MsgDisputeUsageResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ticketId, err := strconv.ParseUint(msg.TicketId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket_id: %w", err)
	}

	if err := k.DisputeRedemption(ctx, msg.Sender, ticketId, msg.Reason); err != nil {
		return nil, err
	}

	k.Logger(ctx).Warn("compute usage disputed",
		"ticket_id", msg.TicketId,
		"reason", msg.Reason,
	)

	return &types.MsgDisputeUsageResponse{}, nil
}

// UpdateComputeParams updates module parameters (governance-gated).
func (k msgServer) UpdateComputeParams(goCtx context.Context, msg *types.MsgUpdateComputeParams) (*types.MsgUpdateComputeParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("%w: expected %s, got %s", types.ErrUnauthorized, k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.Params != nil {
		k.SetParams(ctx, msg.Params)
	}

	k.Logger(ctx).Info("compute pool params updated",
		"authority", msg.Authority,
	)

	return &types.MsgUpdateComputeParamsResponse{}, nil
}
