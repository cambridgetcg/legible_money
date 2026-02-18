package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tree/types"
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

// Project returns a specific project.
func (q queryServer) Project(goCtx context.Context, req *types.QueryProjectRequest) (*types.QueryProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	project, found := q.Keeper.GetProject(ctx, req.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	return &types.QueryProjectResponse{Project: &project}, nil
}

// ProjectsByFounder returns all projects for a founder with pagination.
func (q queryServer) ProjectsByFounder(goCtx context.Context, req *types.QueryProjectsByFounderRequest) (*types.QueryProjectsByFounderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	projects := q.Keeper.GetProjectsByFounder(ctx, req.Founder)

	total := uint64(len(projects))
	offset := uint32(0)
	limit := uint32(100)
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	if uint32(len(projects)) > offset {
		projects = projects[offset:]
	} else {
		projects = nil
	}
	if uint32(len(projects)) > limit {
		projects = projects[:limit]
	}

	// Convert []ProductProject to []*ProductProject for proto
	ptrs := make([]*types.ProductProject, len(projects))
	for i := range projects {
		ptrs[i] = &projects[i]
	}

	return &types.QueryProjectsByFounderResponse{
		Projects: ptrs,
		Total:    total,
	}, nil
}

// Task returns a specific task.
func (q queryServer) Task(goCtx context.Context, req *types.QueryTaskRequest) (*types.QueryTaskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	task, found := q.Keeper.GetTask(ctx, req.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}
	return &types.QueryTaskResponse{Task: &task}, nil
}

// TasksByProject returns all tasks for a project with pagination.
func (q queryServer) TasksByProject(goCtx context.Context, req *types.QueryTasksByProjectRequest) (*types.QueryTasksByProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	tasks := q.Keeper.GetTasksByProject(ctx, req.ProjectId)

	total := uint64(len(tasks))
	offset := uint32(0)
	limit := uint32(100)
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	if uint32(len(tasks)) > offset {
		tasks = tasks[offset:]
	} else {
		tasks = nil
	}
	if uint32(len(tasks)) > limit {
		tasks = tasks[:limit]
	}

	// Convert []ProjectTask to []*ProjectTask for proto
	ptrs := make([]*types.ProjectTask, len(tasks))
	for i := range tasks {
		ptrs[i] = &tasks[i]
	}

	return &types.QueryTasksByProjectResponse{
		Tasks: ptrs,
		Total: total,
	}, nil
}

// Service returns a specific service.
func (q queryServer) Service(goCtx context.Context, req *types.QueryServiceRequest) (*types.QueryServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	service, found := q.Keeper.GetService(ctx, req.ServiceId)
	if !found {
		return nil, types.ErrServiceNotFound
	}
	return &types.QueryServiceResponse{Service: &service}, nil
}

// Seed returns a specific opportunity seed.
func (q queryServer) Seed(goCtx context.Context, req *types.QuerySeedRequest) (*types.QuerySeedResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	seed, found := q.Keeper.GetSeed(ctx, req.SeedId)
	if !found {
		return nil, types.ErrSeedNotFound
	}
	return &types.QuerySeedResponse{Seed: &seed}, nil
}

// Params returns the module parameters.
func (q queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: &params}, nil
}
