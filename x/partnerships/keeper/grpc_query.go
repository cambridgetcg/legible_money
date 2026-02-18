package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/partnerships/types"
)

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: &params}, nil
}

func (k queryServer) Partnership(goCtx context.Context, req *types.QueryPartnershipRequest) (*types.QueryPartnershipResponse, error) {
	if req == nil || req.PartnershipId == "" {
		return nil, fmt.Errorf("partnership_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	p, found := k.GetPartnership(ctx, req.PartnershipId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrPartnershipNotFound, req.PartnershipId)
	}
	return &types.QueryPartnershipResponse{Partnership: &p}, nil
}

func (k queryServer) PartnershipsByParticipant(goCtx context.Context, req *types.QueryPartnershipsByParticipantRequest) (*types.QueryPartnershipsByParticipantResponse, error) {
	if req == nil || req.Participant == "" {
		return nil, fmt.Errorf("participant address is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	partnerships := k.GetPartnershipsByParticipant(ctx, req.Participant)
	ptrs := make([]*types.Partnership, len(partnerships))
	for i := range partnerships {
		ptrs[i] = &partnerships[i]
	}
	return &types.QueryPartnershipsByParticipantResponse{Partnerships: ptrs}, nil
}
