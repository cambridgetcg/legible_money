package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/alignment/types"
)

// Verify Keeper satisfies the QueryServer interface at compile time.
var _ types.QueryServer = queryServer{}

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

// NewQueryServerImpl returns a QueryServer implementation for the alignment module.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

// AlignmentState returns the current alignment module state and params.
func (q queryServer) AlignmentState(
	goCtx context.Context,
	_ *types.QueryAlignmentStateRequest,
) (*types.QueryAlignmentStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryAlignmentStateResponse{
		State:  q.Keeper.GetState(ctx),
		Params: q.Keeper.GetParams(ctx),
	}, nil
}

// DimensionScores returns the dimension scores, observation, and corrections
// for a given epoch. If epoch is 0, returns the latest (current) epoch.
func (q queryServer) DimensionScores(
	goCtx context.Context,
	req *types.QueryDimensionScoresRequest,
) (*types.QueryDimensionScoresResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	epoch := req.Epoch
	if epoch == 0 {
		state := q.Keeper.GetState(ctx)
		epoch = state.CurrentEpoch
	}

	resp := &types.QueryDimensionScoresResponse{
		Scores:      q.Keeper.GetScores(ctx, epoch),
		Observation: q.Keeper.GetObservation(ctx, epoch),
		Corrections: q.Keeper.GetCorrections(ctx, epoch),
	}

	return resp, nil
}

// Params returns the current alignment module parameters.
func (q queryServer) Params(
	goCtx context.Context,
	_ *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{
		Params: q.Keeper.GetParams(ctx),
	}, nil
}
