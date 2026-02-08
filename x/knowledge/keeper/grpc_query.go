package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Params returns the module parameters.
func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// Fact returns a single fact by ID.
func (q queryServer) Fact(goCtx context.Context, req *types.QueryFactRequest) (*types.QueryFactResponse, error) {
	if req.FactId == "" {
		return nil, fmt.Errorf("%w: fact_id is required", types.ErrFactNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	fact, found := q.GetFact(ctx, req.FactId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrFactNotFound, req.FactId)
	}

	return &types.QueryFactResponse{Fact: fact}, nil
}

// Facts returns a paginated list of all facts.
func (q queryServer) Facts(goCtx context.Context, req *types.QueryFactsRequest) (*types.QueryFactsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 100
	}
	offset := req.Offset

	var facts []types.Fact
	var total uint64
	var skipped uint32

	q.IterateFacts(ctx, func(f types.Fact) bool {
		total++
		if skipped < offset {
			skipped++
			return false
		}
		if uint32(len(facts)) >= limit {
			return false // Continue counting total but stop collecting
		}
		facts = append(facts, f)
		return false
	})

	return &types.QueryFactsResponse{
		Facts: facts,
		Total: total,
	}, nil
}

// FactsByDomain returns all facts in a specific domain.
func (q queryServer) FactsByDomain(goCtx context.Context, req *types.QueryFactsByDomainRequest) (*types.QueryFactsByDomainResponse, error) {
	if req.Domain == "" {
		return nil, fmt.Errorf("%w: domain is required", types.ErrDomainNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	facts := q.GetFactsByDomain(ctx, req.Domain)

	return &types.QueryFactsByDomainResponse{Facts: facts}, nil
}

// Claim returns a single claim by ID.
func (q queryServer) Claim(goCtx context.Context, req *types.QueryClaimRequest) (*types.QueryClaimResponse, error) {
	if req.ClaimId == "" {
		return nil, fmt.Errorf("%w: claim_id is required", types.ErrClaimNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	claim, found := q.GetClaim(ctx, req.ClaimId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrClaimNotFound, req.ClaimId)
	}

	return &types.QueryClaimResponse{Claim: claim}, nil
}

// VerificationRound returns a single round by ID.
func (q queryServer) VerificationRound(goCtx context.Context, req *types.QueryVerificationRoundRequest) (*types.QueryVerificationRoundResponse, error) {
	if req.RoundId == "" {
		return nil, fmt.Errorf("%w: round_id is required", types.ErrRoundNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	round, found := q.GetVerificationRound(ctx, req.RoundId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrRoundNotFound, req.RoundId)
	}

	return &types.QueryVerificationRoundResponse{Round: round}, nil
}

// ActiveRounds returns all non-complete verification rounds.
func (q queryServer) ActiveRounds(goCtx context.Context, _ *types.QueryActiveRoundsRequest) (*types.QueryActiveRoundsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	rounds := q.GetActiveRounds(ctx)
	return &types.QueryActiveRoundsResponse{Rounds: rounds}, nil
}

// Domain returns a single domain by name.
func (q queryServer) Domain(goCtx context.Context, req *types.QueryDomainRequest) (*types.QueryDomainResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: domain name is required", types.ErrDomainNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	domain, found := q.GetDomainFromState(ctx, req.Name)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDomainNotFound, req.Name)
	}

	return &types.QueryDomainResponse{Domain: domain}, nil
}
