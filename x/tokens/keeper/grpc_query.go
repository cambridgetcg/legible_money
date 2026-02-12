package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tokens/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

func (k queryServer) Account(goCtx context.Context, req *types.QueryAccountRequest) (*types.QueryAccountResponse, error) {
	if req == nil || req.Address == "" {
		return nil, fmt.Errorf("address is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	acct, found := k.GetExpirableAccount(ctx, req.Address)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrAccountNotFound, req.Address)
	}
	// Recompute ExpiresAt from current params (stored values become stale
	// if PeriodsUntilExpiry or PeriodBlocks change via governance).
	params := k.GetParams(ctx)
	k.refreshExpiresAt(&acct, params)
	// Compute active balance (excludes expired-but-unswept periods).
	activeBalance := k.GetBalance(ctx, req.Address)
	return &types.QueryAccountResponse{Account: acct, ActiveBalance: activeBalance}, nil
}

func (k queryServer) Balance(goCtx context.Context, req *types.QueryBalanceRequest) (*types.QueryBalanceResponse, error) {
	if req == nil || req.Address == "" {
		return nil, fmt.Errorf("address is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	activeBalance := k.GetBalance(ctx, req.Address)
	periodBalancePtrs := k.GetPeriodBalances(ctx, req.Address)
	if periodBalancePtrs == nil {
		periodBalancePtrs = []*types.PeriodBalance{}
	}
	// Recompute ExpiresAt from current params so clients see accurate values.
	params := k.GetParams(ctx)
	periodBalances := make([]types.PeriodBalance, len(periodBalancePtrs))
	for i, pb := range periodBalancePtrs {
		pb.ExpiresAt = k.PeriodToExpirationBlock(pb.Period, params)
		periodBalances[i] = *pb
	}
	return &types.QueryBalanceResponse{
		ActiveBalance:  activeBalance,
		PeriodBalances: periodBalances,
	}, nil
}

func (k queryServer) Allowance(goCtx context.Context, req *types.QueryAllowanceRequest) (*types.QueryAllowanceResponse, error) {
	if req == nil || req.Owner == "" || req.Spender == "" {
		return nil, fmt.Errorf("owner and spender are required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	allowance := k.GetAllowance(ctx, req.Owner, req.Spender)
	return &types.QueryAllowanceResponse{Allowance: allowance}, nil
}

func (k queryServer) RAI(goCtx context.Context, req *types.QueryRAIRequest) (*types.QueryRAIResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	rai := k.GetRAIAccount(ctx)
	return &types.QueryRAIResponse{RAI: rai}, nil
}

func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
