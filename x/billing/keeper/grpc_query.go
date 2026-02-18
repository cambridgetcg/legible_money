package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/billing/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = &queryServer{}

func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

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

func (k queryServer) Quote(goCtx context.Context, req *types.QueryQuoteRequest) (*types.QueryQuoteResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	quote, found := k.GetQuote(ctx, req.QuoteId)
	if !found {
		return nil, fmt.Errorf("%w: %d", types.ErrQuoteNotFound, req.QuoteId)
	}
	return &types.QueryQuoteResponse{Quote: quote}, nil
}
