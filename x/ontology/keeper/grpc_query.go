package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/ontology/types"
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

// Stratum returns the properties of a single stratum.
func (q queryServer) Stratum(goCtx context.Context, req *types.QueryStratumRequest) (*types.QueryStratumResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !req.Stratum.IsValid() {
		return nil, fmt.Errorf("%w: %d", types.ErrInvalidStratum, req.Stratum)
	}

	props, found := q.GetStratum(ctx, req.Stratum)
	if !found {
		return nil, fmt.Errorf("%w: stratum %d not registered", types.ErrInvalidStratum, req.Stratum)
	}

	return &types.QueryStratumResponse{Properties: props}, nil
}

// AllStrata returns all registered strata.
func (q queryServer) AllStrata(goCtx context.Context, _ *types.QueryAllStrataRequest) (*types.QueryAllStrataResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	strata := q.GetAllStrata(ctx)
	return &types.QueryAllStrataResponse{Strata: strata}, nil
}

// Domain returns a single domain by name.
func (q queryServer) Domain(goCtx context.Context, req *types.QueryDomainRequest) (*types.QueryDomainResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: domain name is required", types.ErrDomainNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	domain, found := q.GetDomain(ctx, req.Name)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDomainNotFound, req.Name)
	}

	return &types.QueryDomainResponse{Domain: domain}, nil
}

// DomainsByStratum returns all domains belonging to a specific stratum.
func (q queryServer) DomainsByStratum(goCtx context.Context, req *types.QueryDomainsByStratumRequest) (*types.QueryDomainsByStratumResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !req.Stratum.IsValid() {
		return nil, fmt.Errorf("%w: %d", types.ErrInvalidStratum, req.Stratum)
	}

	domains := q.GetDomainsByStratum(ctx, req.Stratum)
	return &types.QueryDomainsByStratumResponse{Domains: domains}, nil
}

// AllDomains returns all registered domains.
func (q queryServer) AllDomains(goCtx context.Context, _ *types.QueryAllDomainsRequest) (*types.QueryAllDomainsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	domains := q.GetAllDomains(ctx)
	return &types.QueryAllDomainsResponse{Domains: domains}, nil
}

// Proposal returns a single domain proposal by ID.
func (q queryServer) Proposal(goCtx context.Context, req *types.QueryProposalRequest) (*types.QueryProposalResponse, error) {
	if req.ProposalId == "" {
		return nil, fmt.Errorf("%w: proposal_id is required", types.ErrProposalNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	proposal, found := q.GetProposal(ctx, req.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProposalNotFound, req.ProposalId)
	}

	return &types.QueryProposalResponse{Proposal: proposal}, nil
}

// ConfidenceCeiling returns the maximum confidence and decay rate for a domain.
func (q queryServer) ConfidenceCeiling(goCtx context.Context, req *types.QueryConfidenceCeilingRequest) (*types.QueryConfidenceCeilingResponse, error) {
	if req.DomainName == "" {
		return nil, fmt.Errorf("%w: domain_name is required", types.ErrDomainNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	maxConf, decayRate, err := q.GetConfidenceCeiling(ctx, req.DomainName)
	if err != nil {
		return nil, err
	}

	stratum, err := q.GetStratumForDomain(ctx, req.DomainName)
	if err != nil {
		return nil, err
	}

	return &types.QueryConfidenceCeilingResponse{
		MaxConfidence: maxConf,
		DecayRate:     decayRate,
		DomainName:    req.DomainName,
		StratumName:   stratum.Name,
	}, nil
}
