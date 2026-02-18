package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/compute_pool/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

// NewQueryServerImpl returns a QueryServer implementation for the compute_pool module.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper, UnimplementedQueryServer: types.UnimplementedQueryServer{}}
}

var _ types.QueryServer = queryServer{}

// Params returns the current module parameters.
func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// Provider returns a compute provider by address.
func (k queryServer) Provider(goCtx context.Context, req *types.QueryProviderRequest) (*types.QueryProviderResponse, error) {
	if req == nil || req.Address == "" {
		return nil, fmt.Errorf("address is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	provider, found := k.GetProvider(ctx, req.Address)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProviderNotFound, req.Address)
	}
	return &types.QueryProviderResponse{Provider: provider}, nil
}

// ActiveProviders returns all active providers, optionally filtered by service type.
func (k queryServer) ActiveProviders(goCtx context.Context, req *types.QueryActiveProvidersRequest) (*types.QueryActiveProvidersResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	providers := k.GetActiveProvidersByType(ctx, req.ServiceType)
	return &types.QueryActiveProvidersResponse{Providers: providers}, nil
}

// CreditBalance returns a validator's compute credit balance.
func (k queryServer) CreditBalance(goCtx context.Context, req *types.QueryCreditBalanceRequest) (*types.QueryCreditBalanceResponse, error) {
	if req == nil || req.ValidatorAddr == "" {
		return nil, fmt.Errorf("validator_addr is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	credit, found := k.GetCredit(ctx, req.ValidatorAddr)
	if !found {
		// Return empty credit rather than error
		credit = &types.ComputeCredit{
			ValidatorAddr: req.ValidatorAddr,
		}
	}
	return &types.QueryCreditBalanceResponse{Credit: credit}, nil
}

// RedemptionTicket returns a redemption ticket by ID.
func (k queryServer) RedemptionTicket(goCtx context.Context, req *types.QueryRedemptionTicketRequest) (*types.QueryRedemptionTicketResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	ticket, found := k.GetTicket(ctx, req.TicketId)
	if !found {
		return nil, fmt.Errorf("%w: %d", types.ErrTicketNotFound, req.TicketId)
	}
	return &types.QueryRedemptionTicketResponse{Ticket: ticket}, nil
}

// PoolStats returns current pool balance and epoch statistics.
func (k queryServer) PoolStats(goCtx context.Context, req *types.QueryPoolStatsRequest) (*types.QueryPoolStatsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	height := uint64(ctx.BlockHeight())
	epoch := height / types.BlocksPerEpoch

	poolBalance := k.GetPoolBalance(ctx)
	stats := k.GetOrCreateEpochStats(ctx, epoch)

	adjustmentFactor := k.GetAdjustmentFactor(ctx)

	return &types.QueryPoolStatsResponse{
		PoolBalance:         poolBalance.String(),
		CurrentEpoch:        stats,
		AdjustmentFactorBps: adjustmentFactor,
	}, nil
}
