package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/schedule/types"
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

func (q queryServer) Process(goCtx context.Context, req *types.QueryProcessRequest) (*types.QueryProcessResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	process, found := q.GetProcess(ctx, req.ProcessId)
	if !found {
		return nil, types.ErrProcessNotFound
	}
	return &types.QueryProcessResponse{Process: process}, nil
}

func (q queryServer) ProcessesByCreator(goCtx context.Context, req *types.QueryProcessesByCreatorRequest) (*types.QueryProcessesByCreatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	processes := q.GetProcessesByCreator(ctx, req.Creator)
	return &types.QueryProcessesByCreatorResponse{Processes: processes}, nil
}
