package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/ibcratelimit/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns a QueryServer implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Params returns the module parameters.
func (q queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// ChannelFlow returns the flow record for a specific IBC channel.
func (q queryServer) ChannelFlow(goCtx context.Context, req *types.QueryChannelFlowRequest) (*types.QueryChannelFlowResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.ChannelID == "" {
		return nil, types.ErrFlowNotFound.Wrap("channel_id is required")
	}

	flow, found := q.Keeper.GetFlow(ctx, req.ChannelID)
	if !found {
		return &types.QueryChannelFlowResponse{Found: false}, nil
	}
	return &types.QueryChannelFlowResponse{
		Flow:  &flow,
		Found: true,
	}, nil
}
