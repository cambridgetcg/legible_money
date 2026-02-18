package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/channels/types"
)

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

// NewQueryServerImpl returns a QueryServer implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Channel returns a specific payment channel.
func (q queryServer) Channel(goCtx context.Context, req *types.QueryChannelRequest) (*types.QueryChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ch, found := q.Keeper.GetChannel(ctx, req.ChannelId)
	if !found {
		return nil, types.ErrChannelNotFound
	}
	return &types.QueryChannelResponse{Channel: ch}, nil
}

// ChannelsByPayer returns all channels for a payer.
func (q queryServer) ChannelsByPayer(goCtx context.Context, req *types.QueryChannelsByPayerRequest) (*types.QueryChannelsByPayerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	channels := q.Keeper.GetChannelsByPayer(ctx, req.Payer)

	// Apply pagination
	total := uint64(len(channels))
	offset := uint32(0)
	limit := uint32(100)
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	if uint32(len(channels)) > offset {
		channels = channels[offset:]
	} else {
		channels = nil
	}
	if uint32(len(channels)) > limit {
		channels = channels[:limit]
	}

	return &types.QueryChannelsByPayerResponse{
		Channels: channels,
		Total:    total,
	}, nil
}

// ChannelsByProvider returns all channels for a provider.
func (q queryServer) ChannelsByProvider(goCtx context.Context, req *types.QueryChannelsByProviderRequest) (*types.QueryChannelsByProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	channels := q.Keeper.GetChannelsByProvider(ctx, req.Provider)

	// Apply pagination
	total := uint64(len(channels))
	offset := uint32(0)
	limit := uint32(100)
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	if uint32(len(channels)) > offset {
		channels = channels[offset:]
	} else {
		channels = nil
	}
	if uint32(len(channels)) > limit {
		channels = channels[:limit]
	}

	return &types.QueryChannelsByProviderResponse{
		Channels: channels,
		Total:    total,
	}, nil
}

// ActiveDispute returns the active dispute for a channel.
func (q queryServer) ActiveDispute(goCtx context.Context, req *types.QueryActiveDisputeRequest) (*types.QueryActiveDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	dispute, found := q.Keeper.GetDispute(ctx, req.ChannelId)
	if !found {
		return &types.QueryActiveDisputeResponse{Found: false}, nil
	}
	return &types.QueryActiveDisputeResponse{
		Dispute: dispute,
		Found:   true,
	}, nil
}

// Params returns the module parameters.
func (q queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
