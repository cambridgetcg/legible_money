package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/gov/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// LIP returns a single LIP with its votes.
func (q queryServer) LIP(goCtx context.Context, req *types.QueryLIPRequest) (*types.QueryLIPResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	lip, found := q.GetLIP(ctx, req.LipId)
	if !found {
		return nil, types.ErrLIPNotFound
	}

	votes := q.GetVotesForProposal(ctx, req.LipId)

	return &types.QueryLIPResponse{
		Lip:   lip,
		Votes: votes,
	}, nil
}

// LIPs returns LIPs optionally filtered by stage.
func (q queryServer) LIPs(goCtx context.Context, req *types.QueryLIPsRequest) (*types.QueryLIPsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 100
	}

	var allLIPs []types.LIP
	if req.Stage != "" {
		allLIPs = q.GetLIPsByStage(ctx, types.HandwrittenLIPStage(req.Stage))
	} else {
		allLIPs = q.GetAllLIPs(ctx)
	}

	total := uint64(len(allLIPs))

	// Apply offset and limit
	start := uint32(0)
	if req.Offset < uint32(len(allLIPs)) {
		start = req.Offset
	} else {
		start = uint32(len(allLIPs))
	}
	end := start + limit
	if end > uint32(len(allLIPs)) {
		end = uint32(len(allLIPs))
	}

	return &types.QueryLIPsResponse{
		Lips:  allLIPs[start:end],
		Total: total,
	}, nil
}

// Disbursement returns a single disbursement with its votes.
func (q queryServer) Disbursement(goCtx context.Context, req *types.QueryDisbursementRequest) (*types.QueryDisbursementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	disb, found := q.GetDisbursement(ctx, req.DisbursementId)
	if !found {
		return nil, types.ErrDisbursementNotFound
	}

	votes := q.GetVotesForProposal(ctx, req.DisbursementId)

	return &types.QueryDisbursementResponse{
		Disbursement: disb,
		Votes:        votes,
	}, nil
}

// Params returns the governance module parameters.
func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryParamsResponse{
		Params: q.GetParams(ctx),
	}, nil
}
