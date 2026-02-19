package keeper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
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

	var facts []*types.Fact
	var total uint64
	var skipped uint32

	q.IterateFacts(ctx, func(f *types.Fact) bool {
		total++
		if skipped < offset {
			skipped++
			return false
		}
		if uint32(len(facts)) >= limit {
			return false // Continue counting total but stop collecting
		}
		fc := f // already a pointer from iterator
		facts = append(facts, fc)
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
	rawFacts := q.GetFactsByDomain(ctx, req.Domain)
	facts := make([]*types.Fact, len(rawFacts))
	for i := range rawFacts {
		facts[i] = rawFacts[i]
	}

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
	if req.DomainId == "" {
		return nil, fmt.Errorf("%w: domain name is required", types.ErrDomainNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	domain, found := q.GetDomainFromState(ctx, req.DomainId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDomainNotFound, req.DomainId)
	}

	return &types.QueryDomainResponse{Domain: domain}, nil
}

// ---------- Negative Knowledge Queries ----------

// CounterFact returns a single counter-fact by ID.
func (q queryServer) CounterFact(goCtx context.Context, req *types.QueryCounterFactRequest) (*types.QueryCounterFactResponse, error) {
	if req.CounterFactId == "" {
		return nil, fmt.Errorf("%w: counter_fact_id is required", types.ErrCounterFactNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	cf, found := q.GetCounterFact(ctx, req.CounterFactId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrCounterFactNotFound, req.CounterFactId)
	}

	return &types.QueryCounterFactResponse{
		Id:           cf.Id,
		TargetFactId: cf.TargetFactId,
		NegationType: cf.NegationType,
		Evidence:     cf.Justification,
		Submitter:    cf.Submitter,
		Status:       cf.Status,
		Confidence:   cf.Confidence,
	}, nil
}

// CounterFactsByFact returns all counter-facts targeting a specific fact.
func (q queryServer) CounterFactsByFact(goCtx context.Context, req *types.QueryCounterFactsByFactRequest) (*types.QueryCounterFactsByFactResponse, error) {
	if req.FactId == "" {
		return nil, fmt.Errorf("%w: fact_id is required", types.ErrFactNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	rawCFs := q.GetCounterFactsByFact(ctx, req.FactId)
	counterFacts := make([]*types.CounterFact, len(rawCFs))
	for i := range rawCFs {
		counterFacts[i] = rawCFs[i]
	}

	return &types.QueryCounterFactsByFactResponse{CounterFacts: counterFacts}, nil
}

// BidirectionalConfidence returns the dual-track epistemic assessment for a fact.
func (q queryServer) BidirectionalConfidence(goCtx context.Context, req *types.QueryBidirectionalConfidenceRequest) (*types.QueryBidirectionalConfidenceResponse, error) {
	if req.FactId == "" {
		return nil, fmt.Errorf("%w: fact_id is required", types.ErrFactNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify fact exists
	_, found := q.GetFact(ctx, req.FactId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrFactNotFound, req.FactId)
	}

	bidir := q.GetBidirectionalConfidence(ctx, req.FactId)
	return &types.QueryBidirectionalConfidenceResponse{BidirectionalConfidence: bidir}, nil
}

// CounterFactsByDomain returns counter-facts in a specific domain.
func (q queryServer) CounterFactsByDomain(goCtx context.Context, req *types.QueryCounterFactsByDomainRequest) (*types.QueryCounterFactsByDomainResponse, error) {
	if req.Domain == "" {
		return nil, fmt.Errorf("%w: domain is required", types.ErrDomainNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	rawCFs := q.GetCounterFactsByDomain(ctx, req.Domain)
	counterFacts := make([]*types.CounterFact, len(rawCFs))
	for i := range rawCFs {
		counterFacts[i] = rawCFs[i]
	}

	return &types.QueryCounterFactsByDomainResponse{CounterFacts: counterFacts}, nil
}

// ---------- Ontology Queries ----------

// Domains returns domains with optional filters (stratum, parent, pattern).
func (q queryServer) Domains(goCtx context.Context, req *types.QueryDomainsRequest) (*types.QueryDomainsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 100
	}
	offset := req.Offset

	// Collect candidates: either from ontology keeper (stratum filter) or local domains.
	var candidates []*types.OntologicalDomain

	if req.Stratum != "" && q.ontologyKeeper != nil {
		// Parse stratum as level (numeric) or name.
		level, err := parseStratumID(req.Stratum)
		if err != nil {
			return nil, fmt.Errorf("invalid stratum filter: %w", err)
		}
		candidates = q.ontologyKeeper.GetOntologyDomainsByStratum(ctx, level)
	} else if req.Stratum != "" {
		// No ontology keeper — return empty.
		return &types.QueryDomainsResponse{}, nil
	} else {
		// No stratum filter — iterate local knowledge domains.
		q.IterateDomains(ctx, func(d *types.OntologicalDomain) bool {
			candidates = append(candidates, d)
			return false
		})
	}

	// Apply parent_domain and pattern filters.
	var filtered []*types.OntologicalDomain
	for _, d := range candidates {
		if req.ParentDomain != "" && !isChildOf(d.Name, req.ParentDomain) {
			continue
		}
		if req.Pattern != "" && !matchPattern(d.Name, req.Pattern) {
			continue
		}
		filtered = append(filtered, d)
	}

	total := uint64(len(filtered))

	// Apply pagination.
	if offset >= total {
		return &types.QueryDomainsResponse{Total: total}, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := filtered[offset:end]

	return &types.QueryDomainsResponse{
		Domains: page,
		Total:   total,
	}, nil
}

// DomainsByStratum returns all domains in a specific stratum.
func (q queryServer) DomainsByStratum(goCtx context.Context, req *types.QueryDomainsByStratumRequest) (*types.QueryDomainsByStratumResponse, error) {
	if req.StratumId == "" {
		return nil, fmt.Errorf("%w: stratum_id is required", types.ErrStratumNotFound)
	}

	if q.ontologyKeeper == nil {
		return &types.QueryDomainsByStratumResponse{}, nil
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	level, err := parseStratumID(req.StratumId)
	if err != nil {
		return nil, fmt.Errorf("invalid stratum_id: %w", err)
	}

	domains := q.ontologyKeeper.GetOntologyDomainsByStratum(ctx, level)
	return &types.QueryDomainsByStratumResponse{Domains: domains}, nil
}

// DomainChildren returns child domains of a parent domain.
// Uses dot-separated naming convention (e.g., "physics.mechanics" is child of "physics").
func (q queryServer) DomainChildren(goCtx context.Context, req *types.QueryDomainChildrenRequest) (*types.QueryDomainChildrenResponse, error) {
	if req.DomainId == "" {
		return nil, fmt.Errorf("%w: domain_id is required", types.ErrDomainNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	prefix := req.DomainId + "."

	var children []*types.OntologicalDomain

	// Check ontology keeper first for comprehensive domain list.
	if q.ontologyKeeper != nil {
		allDomains := q.ontologyKeeper.GetOntologyDomains(ctx)
		for _, d := range allDomains {
			if req.IncludeDescendants {
				if strings.HasPrefix(d.Name, prefix) {
					children = append(children, d)
				}
			} else {
				if isDirectChild(d.Name, req.DomainId) {
					children = append(children, d)
				}
			}
		}
	} else {
		// Fallback: iterate local knowledge domains.
		q.IterateDomains(ctx, func(d *types.OntologicalDomain) bool {
			if req.IncludeDescendants {
				if strings.HasPrefix(d.Name, prefix) {
					children = append(children, d)
				}
			} else {
				if isDirectChild(d.Name, req.DomainId) {
					children = append(children, d)
				}
			}
			return false
		})
	}

	return &types.QueryDomainChildrenResponse{Children: children}, nil
}

// DomainPath returns the path from root to a domain.
// For "physics.mechanics.quantum", returns [physics, physics.mechanics, physics.mechanics.quantum].
func (q queryServer) DomainPath(goCtx context.Context, req *types.QueryDomainPathRequest) (*types.QueryDomainPathResponse, error) {
	if req.DomainId == "" {
		return nil, fmt.Errorf("%w: domain_id is required", types.ErrDomainNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	parts := strings.Split(req.DomainId, ".")

	var path []*types.OntologicalDomain
	for i := 1; i <= len(parts); i++ {
		name := strings.Join(parts[:i], ".")
		// Try local knowledge domains first.
		domain, found := q.GetDomainFromState(ctx, name)
		if found {
			path = append(path, domain)
			continue
		}
		// Placeholder for unregistered path segments.
		path = append(path, &types.OntologicalDomain{
			Name:   name,
			Status: "unregistered",
		})
	}

	return &types.QueryDomainPathResponse{Path: path}, nil
}

// Stratum returns a single stratum definition by ID.
func (q queryServer) Stratum(goCtx context.Context, req *types.QueryStratumRequest) (*types.QueryStratumResponse, error) {
	if req.StratumId == "" {
		return nil, fmt.Errorf("%w: stratum_id is required", types.ErrStratumNotFound)
	}

	if q.ontologyKeeper == nil {
		return nil, fmt.Errorf("%w: ontology module not available", types.ErrStratumNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	level, err := parseStratumID(req.StratumId)
	if err != nil {
		return nil, fmt.Errorf("invalid stratum_id: %w", err)
	}

	def, found := q.ontologyKeeper.GetStratumDefinition(ctx, level)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrStratumNotFound, req.StratumId)
	}

	return &types.QueryStratumResponse{Stratum: def}, nil
}

// Strata returns all strata, with optional level range filter.
func (q queryServer) Strata(goCtx context.Context, req *types.QueryStrataRequest) (*types.QueryStrataResponse, error) {
	if q.ontologyKeeper == nil {
		return &types.QueryStrataResponse{}, nil
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	allDefs := q.ontologyKeeper.GetAllStratumDefinitions(ctx)

	// Apply level range filter if specified.
	if req.MinLevel > 0 || req.MaxLevel > 0 {
		maxLevel := req.MaxLevel
		if maxLevel == 0 {
			maxLevel = 6 // MaxStratum = Testimonial
		}
		var filtered []*types.StratumDefinition
		for _, def := range allDefs {
			if def.Level >= req.MinLevel && def.Level <= maxLevel {
				filtered = append(filtered, def)
			}
		}
		return &types.QueryStrataResponse{Strata: filtered}, nil
	}

	return &types.QueryStrataResponse{Strata: allDefs}, nil
}

// StratumRegistry returns the full stratum registry.
func (q queryServer) StratumRegistry(goCtx context.Context, _ *types.QueryStratumRegistryRequest) (*types.QueryStratumRegistryResponse, error) {
	if q.ontologyKeeper == nil {
		return &types.QueryStratumRegistryResponse{
			Registry: &types.StratumRegistry{SchemaVersion: 1},
		}, nil
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	allDefs := q.ontologyKeeper.GetAllStratumDefinitions(ctx)

	registry := &types.StratumRegistry{
		SchemaVersion: 1,
		Strata:        allDefs,
		LastModified:  uint64(ctx.BlockHeight()),
	}

	return &types.QueryStratumRegistryResponse{Registry: registry}, nil
}

// ValidateDependency checks if a stratum dependency is valid.
// A dependency is valid if both strata exist and the source stratum level
// is >= the target stratum level (higher strata can depend on lower/more fundamental ones).
func (q queryServer) ValidateDependency(goCtx context.Context, req *types.QueryValidateDependencyRequest) (*types.QueryValidateDependencyResponse, error) {
	if req.FromStratum == "" || req.ToStratum == "" {
		return &types.QueryValidateDependencyResponse{
			Valid:  false,
			Reason: "both from_stratum and to_stratum are required",
		}, nil
	}

	if q.ontologyKeeper == nil {
		return &types.QueryValidateDependencyResponse{
			Valid:  false,
			Reason: "ontology module not available",
		}, nil
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	fromLevel, err := parseStratumID(req.FromStratum)
	if err != nil {
		return &types.QueryValidateDependencyResponse{
			Valid:  false,
			Reason: fmt.Sprintf("invalid from_stratum: %v", err),
		}, nil
	}

	toLevel, err := parseStratumID(req.ToStratum)
	if err != nil {
		return &types.QueryValidateDependencyResponse{
			Valid:  false,
			Reason: fmt.Sprintf("invalid to_stratum: %v", err),
		}, nil
	}

	// Verify both strata exist.
	_, fromFound := q.ontologyKeeper.GetStratumDefinition(ctx, fromLevel)
	if !fromFound {
		return &types.QueryValidateDependencyResponse{
			Valid:  false,
			Reason: fmt.Sprintf("source stratum %q not found", req.FromStratum),
		}, nil
	}

	_, toFound := q.ontologyKeeper.GetStratumDefinition(ctx, toLevel)
	if !toFound {
		return &types.QueryValidateDependencyResponse{
			Valid:  false,
			Reason: fmt.Sprintf("target stratum %q not found", req.ToStratum),
		}, nil
	}

	// Self-dependency is not valid.
	if fromLevel == toLevel {
		return &types.QueryValidateDependencyResponse{
			Valid:  false,
			Reason: "a stratum cannot depend on itself",
		}, nil
	}

	// Higher strata (higher level number) can depend on lower (more fundamental) strata.
	if fromLevel < toLevel {
		return &types.QueryValidateDependencyResponse{
			Valid:  false,
			Reason: fmt.Sprintf("stratum at level %d cannot depend on higher stratum at level %d", fromLevel, toLevel),
		}, nil
	}

	return &types.QueryValidateDependencyResponse{
		Valid:  true,
		Reason: fmt.Sprintf("valid: stratum level %d can depend on level %d", fromLevel, toLevel),
	}, nil
}

// CalculateConfidence calculates effective confidence for a domain,
// applying stratum ceiling and dependency discounts.
func (q queryServer) CalculateConfidence(goCtx context.Context, req *types.QueryCalculateConfidenceRequest) (*types.QueryCalculateConfidenceResponse, error) {
	if req.DomainId == "" {
		return nil, fmt.Errorf("%w: domain_id is required", types.ErrDomainNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	result := &types.OntologicalConfidenceResult{
		BaseConfidence:      req.BaseConfidence,
		EffectiveConfidence: req.BaseConfidence,
		StratumCeiling:      1000000, // Default: no ceiling
		InheritedCeiling:    1000000,
		StratumDiscount:     1000000, // 1.0 = no discount
	}

	// Apply stratum ceiling if ontology keeper is available.
	if q.ontologyKeeper != nil {
		maxConf, _, err := q.ontologyKeeper.GetConfidenceCeiling(ctx, req.DomainId)
		if err == nil {
			result.StratumCeiling = uint32(maxConf)
			if result.EffectiveConfidence > uint32(maxConf) {
				result.EffectiveConfidence = uint32(maxConf)
			}
		}
	}

	// Apply dependency discounts.
	// Each dependency reduces effective confidence by (1 - strength/1000000).
	for _, dep := range req.Dependencies {
		detail := &types.DependencyDetail{
			Stratum:  dep.PrimaryStratum,
			Type:     dep.DependencyType,
			Strength: dep.DependencyStrength,
		}
		result.Dependencies = append(result.Dependencies, detail)

		if dep.DependencyStrength > 0 && dep.DependencyStrength < 1000000 {
			// Discount: effective = effective * (1 - (1 - strength/1M) * 0.1)
			// Simplified: lose up to 10% per weak dependency.
			discount := uint32(1000000) - (1000000-dep.DependencyStrength)/10
			result.StratumDiscount = uint32(uint64(result.StratumDiscount) * uint64(discount) / 1000000)
			result.EffectiveConfidence = uint32(uint64(result.EffectiveConfidence) * uint64(discount) / 1000000)
		}
	}

	return &types.QueryCalculateConfidenceResponse{Result: result}, nil
}

// DomainProposal returns a single domain proposal by ID.
func (q queryServer) DomainProposal(goCtx context.Context, req *types.QueryDomainProposalRequest) (*types.QueryDomainProposalResponse, error) {
	if req.ProposalId == "" {
		return nil, fmt.Errorf("%w: proposal_id is required", types.ErrProposalNotFound)
	}

	if q.ontologyKeeper == nil {
		return nil, fmt.Errorf("%w: ontology module not available", types.ErrProposalNotFound)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	proposal, found := q.ontologyKeeper.GetDomainProposalById(ctx, req.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrProposalNotFound, req.ProposalId)
	}

	// Endorsements/challenges are not stored separately in ontology module.
	// Return empty slices — future work to track individual endorsement/challenge records.
	return &types.QueryDomainProposalResponse{
		Proposal: proposal,
	}, nil
}

// DomainProposals returns a paginated list of domain proposals with optional status filter.
func (q queryServer) DomainProposals(goCtx context.Context, req *types.QueryDomainProposalsRequest) (*types.QueryDomainProposalsResponse, error) {
	if q.ontologyKeeper == nil {
		return &types.QueryDomainProposalsResponse{}, nil
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	allProposals := q.ontologyKeeper.GetAllDomainProposals(ctx)

	// Apply status filter.
	var filtered []*types.DomainProposal
	for _, p := range allProposals {
		if req.Status != types.DomainProposalStatus_DOMAIN_PROPOSAL_STATUS_UNSPECIFIED && p.Status != req.Status {
			continue
		}
		filtered = append(filtered, p)
	}

	total := uint64(len(filtered))

	// Apply pagination.
	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 100
	}
	offset := req.Offset
	if offset >= total {
		return &types.QueryDomainProposalsResponse{Total: total}, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := filtered[offset:end]

	return &types.QueryDomainProposalsResponse{
		Proposals: page,
		Total:     total,
	}, nil
}

// ---------- Helpers ----------

// parseStratumID parses a stratum identifier as either a numeric level or a name.
func parseStratumID(id string) (uint32, error) {
	// Try numeric first.
	if level, err := strconv.ParseUint(id, 10, 32); err == nil {
		return uint32(level), nil
	}

	// Try name lookup.
	nameToLevel := map[string]uint32{
		"axiomatic":     0,
		"formal":        1,
		"protocol":      2,
		"computational": 3,
		"empirical":     4,
		"historical":    5,
		"testimonial":   6,
	}
	if level, ok := nameToLevel[strings.ToLower(id)]; ok {
		return level, nil
	}

	return 0, fmt.Errorf("unknown stratum: %s", id)
}

// matchPattern checks if a domain name matches a glob-like pattern.
// Supports trailing wildcard: "physics.*" matches "physics.mechanics", "physics.optics".
func matchPattern(name, pattern string) bool {
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(name, prefix+".") || name == prefix
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(name, prefix)
	}
	return name == pattern
}

// isChildOf checks if name is a child of parent using dot notation.
// "physics.mechanics" is a child of "physics".
func isChildOf(name, parent string) bool {
	return strings.HasPrefix(name, parent+".")
}

// isDirectChild checks if name is a direct child (one level below parent).
// "physics.mechanics" is a direct child of "physics".
// "physics.mechanics.quantum" is NOT a direct child of "physics".
func isDirectChild(name, parent string) bool {
	prefix := parent + "."
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	rest := strings.TrimPrefix(name, prefix)
	return !strings.Contains(rest, ".")
}

// ResearchFund returns the research fund balance and statistics.
func (q queryServer) ResearchFund(goCtx context.Context, _ *types.QueryResearchFundRequest) (*types.QueryResearchFundResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	balance := q.GetResearchFundBalance(ctx)
	stats := q.GetResearchFundStats(ctx)

	// Count active proposals
	var activeCount uint32
	q.IterateResearchProposals(ctx, func(p *types.ResearchProposal) bool {
		if p.Status == types.ResearchProposalStatusVoting {
			activeCount++
		}
		return false
	})

	return &types.QueryResearchFundResponse{
		Balance:         balance.Amount.String(),
		ActiveProposals: activeCount,
		TotalDisbursed:  strconv.FormatUint(stats.TotalDisbursed, 10),
		TotalCollected:  strconv.FormatUint(stats.TotalCollected, 10),
	}, nil
}
