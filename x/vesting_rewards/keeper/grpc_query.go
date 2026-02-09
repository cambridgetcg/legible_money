package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns a query server implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// VestingSchedule returns a single vesting schedule by ID.
func (q queryServer) VestingSchedule(goCtx context.Context, req *types.QueryVestingScheduleRequest) (*types.QueryVestingScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	schedule, found := q.Keeper.GetVestingSchedule(ctx, req.VestingId)
	if !found {
		return nil, types.ErrScheduleNotFound
	}

	return &types.QueryVestingScheduleResponse{Schedule: schedule}, nil
}

// VestingSchedulesByRecipient returns all vesting schedules for a recipient.
func (q queryServer) VestingSchedulesByRecipient(goCtx context.Context, req *types.QueryVestingSchedulesByRecipientRequest) (*types.QueryVestingSchedulesByRecipientResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	schedules := q.Keeper.GetVestingSchedulesByRecipient(ctx, req.Recipient)

	return &types.QueryVestingSchedulesByRecipientResponse{Schedules: schedules}, nil
}

// ActiveVestingSchedules returns all active/paused vesting schedules.
func (q queryServer) ActiveVestingSchedules(goCtx context.Context, req *types.QueryActiveVestingSchedulesRequest) (*types.QueryActiveVestingSchedulesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	all := q.Keeper.GetAllActiveVestingSchedules(ctx)

	total := uint32(len(all))
	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 100
	}

	start := req.Offset
	if start >= total {
		return &types.QueryActiveVestingSchedulesResponse{Total: total}, nil
	}

	end := start + limit
	if end > total {
		end = total
	}

	return &types.QueryActiveVestingSchedulesResponse{
		Schedules: all[start:end],
		Total:     total,
	}, nil
}

// BlockRewardDistribution returns the reward distribution for a specific block.
func (q queryServer) BlockRewardDistribution(goCtx context.Context, req *types.QueryBlockRewardDistributionRequest) (*types.QueryBlockRewardDistributionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dist, found := q.Keeper.GetBlockRewardDistribution(ctx, req.BlockHeight)

	return &types.QueryBlockRewardDistributionResponse{
		Distribution: dist,
		Found:        found,
	}, nil
}

// Params returns the module parameters and category configurations.
func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{
		Params:          q.Keeper.GetParams(ctx),
		CategoryConfigs: q.Keeper.GetAllCategoryConfigs(ctx),
	}, nil
}
