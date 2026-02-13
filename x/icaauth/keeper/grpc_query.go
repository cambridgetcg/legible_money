package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/icaauth/types"
)

type queryServer struct {
	k Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{k: keeper}
}

// InterchainAccounts returns all ICA accounts for a DID.
func (qs *queryServer) InterchainAccounts(goCtx context.Context, req *types.QueryInterchainAccountsRequest) (*types.QueryInterchainAccountsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	accounts := qs.k.GetRemoteAccountsForDID(ctx, req.Did)
	if accounts == nil {
		accounts = []*types.RemoteAccount{}
	}
	return &types.QueryInterchainAccountsResponse{Accounts: accounts}, nil
}

// Params returns the module parameters.
func (qs *queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := qs.k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
