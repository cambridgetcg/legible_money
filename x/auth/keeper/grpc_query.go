package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/auth/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

// NewQueryServerImpl returns an implementation of the QueryServer interface
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

// Account returns a Legible account by bech32 address.
func (qs queryServer) Account(goCtx context.Context, req *types.QueryAccountRequest) (*types.QueryAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	account, found := qs.GetAccount(ctx, req.Address)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	return &types.QueryAccountResponse{Account: account}, nil
}

// AccountByDID returns a Legible account by DID.
func (qs queryServer) AccountByDID(goCtx context.Context, req *types.QueryAccountByDIDRequest) (*types.QueryAccountByDIDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	account, found := qs.GetAccountByDID(ctx, req.Did)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	return &types.QueryAccountByDIDResponse{Account: account}, nil
}

// SessionKeys returns all session keys for an owner.
func (qs queryServer) SessionKeys(goCtx context.Context, req *types.QuerySessionKeysRequest) (*types.QuerySessionKeysResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sessions := qs.GetSessionKeysForOwner(ctx, req.Owner)
	if sessions == nil {
		sessions = []*types.SessionKey{}
	}

	return &types.QuerySessionKeysResponse{SessionKeys: sessions}, nil
}

// Params returns the module parameters.
func (qs queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := qs.GetParams(ctx)

	return &types.QueryParamsResponse{Params: &params}, nil
}
