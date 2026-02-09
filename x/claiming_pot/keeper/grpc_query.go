package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/claiming_pot/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns a query server implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Pot returns a single claiming pot by ID.
func (q queryServer) Pot(goCtx context.Context, req *types.QueryPotRequest) (*types.QueryPotResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pot, found := q.Keeper.GetPot(ctx, req.PotId)
	if !found {
		return nil, types.ErrPotNotFound
	}

	return &types.QueryPotResponse{Pot: &pot}, nil
}

// Pots returns pots with optional status filter and pagination.
func (q queryServer) Pots(goCtx context.Context, req *types.QueryPotsRequest) (*types.QueryPotsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var all []types.ClaimingPot
	if req.Status != "" {
		all = q.Keeper.GetPotsByStatus(ctx, types.PotStatus(req.Status))
	} else {
		q.Keeper.IteratePots(ctx, func(pot types.ClaimingPot) bool {
			all = append(all, pot)
			return false
		})
	}

	total := uint64(len(all))
	limit := uint32(100)
	if req.Limit > 0 && req.Limit < 100 {
		limit = req.Limit
	}

	start := req.Offset
	if uint64(start) >= total {
		return &types.QueryPotsResponse{Total: total}, nil
	}

	end := uint64(start) + uint64(limit)
	if end > total {
		end = total
	}

	return &types.QueryPotsResponse{
		Pots:  all[start:end],
		Total: total,
	}, nil
}

// PotsByCreator returns all pots created by a specific address.
func (q queryServer) PotsByCreator(goCtx context.Context, req *types.QueryPotsByCreatorRequest) (*types.QueryPotsByCreatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	all := q.Keeper.GetPotsByCreator(ctx, req.Creator)

	total := uint64(len(all))
	limit := uint32(100)
	if req.Limit > 0 && req.Limit < 100 {
		limit = req.Limit
	}

	start := req.Offset
	if uint64(start) >= total {
		return &types.QueryPotsByCreatorResponse{Total: total}, nil
	}

	end := uint64(start) + uint64(limit)
	if end > total {
		end = total
	}

	return &types.QueryPotsByCreatorResponse{
		Pots:  all[start:end],
		Total: total,
	}, nil
}

// Eligibility checks if an address is eligible to claim from a pot.
func (q queryServer) Eligibility(goCtx context.Context, req *types.QueryEligibilityRequest) (*types.QueryEligibilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pot, found := q.Keeper.GetPot(ctx, req.PotId)
	if !found {
		return nil, types.ErrPotNotFound
	}

	eligible, reason := q.Keeper.EvaluateEligibility(ctx, pot, req.Claimer)

	return &types.QueryEligibilityResponse{
		Eligible: eligible,
		Reason:   reason,
	}, nil
}

// ClaimRecords returns all claim records for a pot.
func (q queryServer) ClaimRecords(goCtx context.Context, req *types.QueryClaimRecordsRequest) (*types.QueryClaimRecordsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	all := q.Keeper.GetAllClaimRecords(ctx, req.PotId)

	total := uint64(len(all))
	limit := uint32(100)
	if req.Limit > 0 && req.Limit < 100 {
		limit = req.Limit
	}

	start := req.Offset
	if uint64(start) >= total {
		return &types.QueryClaimRecordsResponse{Total: total}, nil
	}

	end := uint64(start) + uint64(limit)
	if end > total {
		end = total
	}

	return &types.QueryClaimRecordsResponse{
		Records: all[start:end],
		Total:   total,
	}, nil
}

// Params returns the module parameters.
func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{
		Params: q.Keeper.GetParams(ctx),
	}, nil
}
