package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/staking/types"
)

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

// NewQueryServerImpl returns a query server implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Validator returns a single validator by operator address.
func (q queryServer) Validator(goCtx context.Context, req *types.QueryValidatorRequest) (*types.QueryValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	validator, found := q.Keeper.GetValidator(ctx, req.OperatorAddress)
	if !found {
		return nil, types.ErrValidatorNotFound
	}

	return &types.QueryValidatorResponse{Validator: validator}, nil
}

// Validators returns validators with optional filtering.
func (q queryServer) Validators(goCtx context.Context, req *types.QueryValidatorsRequest) (*types.QueryValidatorsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 100
	}

	var all []*types.LegibleValidator
	q.Keeper.IterateValidators(ctx, func(v *types.LegibleValidator) bool {
		if req.ActiveOnly && !v.IsActive {
			return false
		}
		if req.Tier >= 0 && types.ValidatorTierString(types.ValidatorTier(req.Tier)) != v.Tier {
			return false
		}
		all = append(all, v)
		return false
	})

	total := uint32(len(all))

	// Apply pagination
	start := req.Offset
	if start >= total {
		return &types.QueryValidatorsResponse{Total: total}, nil
	}

	end := start + limit
	if end > total {
		end = total
	}

	return &types.QueryValidatorsResponse{
		Validators: all[start:end],
		Total:      total,
	}, nil
}

// Delegation returns a single delegation.
func (q queryServer) Delegation(goCtx context.Context, req *types.QueryDelegationRequest) (*types.QueryDelegationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	delegation, found := q.Keeper.GetDelegation(ctx, req.DelegatorAddress, req.ValidatorAddress)
	if !found {
		return nil, types.ErrDelegationNotFound
	}

	return &types.QueryDelegationResponse{Delegation: delegation}, nil
}

// DelegationsForValidator returns all delegations to a specific validator.
func (q queryServer) DelegationsForValidator(goCtx context.Context, req *types.QueryDelegationsForValidatorRequest) (*types.QueryDelegationsForValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	delegations := q.Keeper.GetDelegationsForValidator(ctx, req.ValidatorAddress)

	return &types.QueryDelegationsForValidatorResponse{Delegations: delegations}, nil
}

// Unbondings returns unbonding entries for a delegator.
func (q queryServer) Unbondings(goCtx context.Context, req *types.QueryUnbondingsRequest) (*types.QueryUnbondingsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	unbondings := q.Keeper.GetUnbondingsForDelegator(ctx, req.DelegatorAddress)

	return &types.QueryUnbondingsResponse{Unbondings: unbondings}, nil
}

// Params returns the module parameters and tier configurations.
func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := q.Keeper.GetParams(ctx) // GetParams backward compat fills TierConfigs

	return &types.QueryParamsResponse{
		Params:      params,
		TierConfigs: params.TierConfigs, // sourced from params (canonical location)
	}, nil
}
