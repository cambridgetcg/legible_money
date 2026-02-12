package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/discovery/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryParamsResponse{Params: q.GetParams(ctx)}, nil
}

func (q queryServer) Agent(goCtx context.Context, req *types.QueryAgentRequest) (*types.QueryAgentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	profile, found := q.GetProfile(ctx, req.Address)
	if !found {
		return nil, types.ErrAgentNotFound
	}
	return &types.QueryAgentResponse{Profile: profile}, nil
}

func (q queryServer) AgentsByDomain(goCtx context.Context, req *types.QueryAgentsByDomainRequest) (*types.QueryAgentsByDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	profiles := q.GetProfilesByDomain(ctx, req.Domain)
	return &types.QueryAgentsByDomainResponse{Profiles: profiles}, nil
}

func (q queryServer) AgentsByCapability(goCtx context.Context, req *types.QueryAgentsByCapabilityRequest) (*types.QueryAgentsByCapabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	profiles := q.GetProfilesByCapability(ctx, req.CapabilityType)
	return &types.QueryAgentsByCapabilityResponse{Profiles: profiles}, nil
}
