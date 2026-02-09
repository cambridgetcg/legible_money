package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/bvm/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns a QueryServer implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Contract returns a specific deployed contract.
func (q queryServer) Contract(goCtx context.Context, req *types.QueryContractRequest) (*types.QueryContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	contract, found := q.Keeper.GetContract(ctx, req.Address)
	if !found {
		return nil, types.ErrContractNotFound
	}
	return &types.QueryContractResponse{Contract: &contract}, nil
}

// ContractsByCreator returns all contracts for a creator with pagination.
func (q queryServer) ContractsByCreator(goCtx context.Context, req *types.QueryContractsByCreatorRequest) (*types.QueryContractsByCreatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	contracts := q.Keeper.GetContractsByCreator(ctx, req.Creator)
	if contracts == nil {
		contracts = []types.DeployedContract{}
	}

	total := uint64(len(contracts))
	offset := uint32(0)
	limit := uint32(100)
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	if uint32(len(contracts)) > offset {
		contracts = contracts[offset:]
	} else {
		contracts = []types.DeployedContract{}
	}
	if uint32(len(contracts)) > limit {
		contracts = contracts[:limit]
	}

	return &types.QueryContractsByCreatorResponse{
		Contracts: contracts,
		Total:     total,
	}, nil
}

// Code returns a specific contract code by hash.
func (q queryServer) Code(goCtx context.Context, req *types.QueryCodeRequest) (*types.QueryCodeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	code, found := q.Keeper.GetCode(ctx, req.CodeHash)
	if !found {
		return nil, types.ErrCodeNotFound
	}
	return &types.QueryCodeResponse{Code: &code}, nil
}

// ContractState returns a value from a contract's state.
func (q queryServer) ContractState(goCtx context.Context, req *types.QueryContractStateRequest) (*types.QueryContractStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	value, found := q.Keeper.GetContractState(ctx, req.ContractAddress, req.Key)
	return &types.QueryContractStateResponse{
		Value: value,
		Found: found,
	}, nil
}

// Schedule returns a specific schedule by ID.
func (q queryServer) Schedule(goCtx context.Context, req *types.QueryScheduleRequest) (*types.QueryScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	schedule, found := q.Keeper.GetSchedule(ctx, req.ScheduleId)
	if !found {
		return nil, types.ErrScheduleNotFound
	}
	return &types.QueryScheduleResponse{Schedule: &schedule}, nil
}

// Params returns the module parameters.
func (q queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
