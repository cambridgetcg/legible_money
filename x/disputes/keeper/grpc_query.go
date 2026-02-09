package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/disputes/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Dispute returns a single dispute by ID.
func (k queryServer) Dispute(goCtx context.Context, req *types.QueryDisputeRequest) (*types.QueryDisputeResponse, error) {
	if req == nil || req.DisputeId == "" {
		return nil, fmt.Errorf("invalid request: dispute_id is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	dispute, found := k.GetDispute(ctx, req.DisputeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeNotFound, req.DisputeId)
	}

	return &types.QueryDisputeResponse{Dispute: dispute}, nil
}

// DisputesByFact returns all disputes for a given fact.
func (k queryServer) DisputesByFact(goCtx context.Context, req *types.QueryDisputesByFactRequest) (*types.QueryDisputesByFactResponse, error) {
	if req == nil || req.FactId == "" {
		return nil, fmt.Errorf("invalid request: fact_id is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	disputes := k.GetDisputesByFact(ctx, req.FactId)

	return &types.QueryDisputesByFactResponse{Disputes: disputes}, nil
}

// ActiveDisputes returns all non-resolved, non-expired disputes.
func (k queryServer) ActiveDisputes(goCtx context.Context, req *types.QueryActiveDisputesRequest) (*types.QueryActiveDisputesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	disputes := k.GetActiveDisputes(ctx)

	return &types.QueryActiveDisputesResponse{Disputes: disputes}, nil
}

// Evidence returns all evidence for a given dispute.
func (k queryServer) Evidence(goCtx context.Context, req *types.QueryEvidenceRequest) (*types.QueryEvidenceResponse, error) {
	if req == nil || req.DisputeId == "" {
		return nil, fmt.Errorf("invalid request: dispute_id is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	evidence := k.GetEvidenceByDispute(ctx, req.DisputeId)

	return &types.QueryEvidenceResponse{Evidence: evidence}, nil
}

// Votes returns all votes for a given dispute.
func (k queryServer) Votes(goCtx context.Context, req *types.QueryVotesRequest) (*types.QueryVotesResponse, error) {
	if req == nil || req.DisputeId == "" {
		return nil, fmt.Errorf("invalid request: dispute_id is required")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	votes := k.GetVotes(ctx, req.DisputeId)

	return &types.QueryVotesResponse{Votes: votes}, nil
}

// Params returns the module parameters.
func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	return &types.QueryParamsResponse{Params: params}, nil
}
