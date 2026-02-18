package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ktypes "github.com/legible-money/legible/x/knowledge/types"
	"github.com/legible-money/legible/x/ontology/types"
)

// ---------- OntologyKeeper query support methods ----------
// These methods satisfy the extended OntologyKeeper interface defined in
// x/knowledge/types/expected_keepers.go, converting ontology types to
// knowledge proto types for the knowledge module's query server.

// stratumLevelToID maps stratum levels to their canonical string IDs
// used in StratumDefinition. Matches the stratum names from types.go.
var stratumLevelToID = map[uint32]string{
	0: "axiomatic",
	1: "formal",
	2: "protocol",
	3: "computational",
	4: "empirical",
	5: "historical",
	6: "testimonial",
}

// GetStratumDefinition returns a StratumDefinition for a given stratum level.
func (k Keeper) GetStratumDefinition(ctx sdk.Context, level uint32) (*ktypes.StratumDefinition, bool) {
	props, found := k.GetStratum(ctx, types.Stratum(level))
	if !found {
		return nil, false
	}
	return stratumPropsToDefinition(props), true
}

// GetAllStratumDefinitions returns all strata as StratumDefinitions.
func (k Keeper) GetAllStratumDefinitions(ctx sdk.Context) []*ktypes.StratumDefinition {
	allProps := k.GetAllStrata(ctx)
	result := make([]*ktypes.StratumDefinition, 0, len(allProps))
	for _, props := range allProps {
		result = append(result, stratumPropsToDefinition(props))
	}
	return result
}

// GetOntologyDomains returns all registered domains as OntologicalDomains.
func (k Keeper) GetOntologyDomains(ctx sdk.Context) []*ktypes.OntologicalDomain {
	allDomains := k.GetAllDomains(ctx)
	result := make([]*ktypes.OntologicalDomain, 0, len(allDomains))
	for _, d := range allDomains {
		result = append(result, domainToOntological(d))
	}
	return result
}

// GetOntologyDomainsByStratum returns domains belonging to a specific stratum level.
func (k Keeper) GetOntologyDomainsByStratum(ctx sdk.Context, level uint32) []*ktypes.OntologicalDomain {
	domains := k.GetDomainsByStratum(ctx, types.Stratum(level))
	result := make([]*ktypes.OntologicalDomain, 0, len(domains))
	for _, d := range domains {
		result = append(result, domainToOntological(d))
	}
	return result
}

// GetDomainProposalById returns a domain proposal by its ID.
func (k Keeper) GetDomainProposalById(ctx sdk.Context, id string) (*ktypes.DomainProposal, bool) {
	prop, found := k.GetProposal(ctx, id)
	if !found {
		return nil, false
	}
	return ontologyProposalToKnowledge(prop), true
}

// GetAllDomainProposals returns all domain proposals.
func (k Keeper) GetAllDomainProposals(ctx sdk.Context) []*ktypes.DomainProposal {
	var result []*ktypes.DomainProposal
	k.IterateProposals(ctx, func(p types.DomainProposal) bool {
		result = append(result, ontologyProposalToKnowledge(p))
		return false
	})
	return result
}

// ---------- Conversion helpers ----------

// stratumPropsToDefinition converts ontology StratumProperties to knowledge StratumDefinition.
func stratumPropsToDefinition(props types.StratumProperties) *ktypes.StratumDefinition {
	id := stratumLevelToID[props.Stratum]
	if id == "" {
		id = fmt.Sprintf("stratum_%d", props.Stratum)
	}

	return &ktypes.StratumDefinition{
		Id:                id,
		Name:              props.Name,
		Level:             props.Stratum,
		ConfidenceCeiling: uint32(props.MaxConfidence), // ontology uses uint64, knowledge uses uint32
		Description:       props.Description,
	}
}

// domainToOntological converts ontology Domain to knowledge OntologicalDomain.
func domainToOntological(d types.Domain) *ktypes.OntologicalDomain {
	return &ktypes.OntologicalDomain{
		Name:            d.Name,
		Description:     d.Description,
		Proposer:        d.ProposedBy,
		Status:          d.Status,
		CreatedAtHeight: d.CreatedAt,
	}
}

// ontologyProposalToKnowledge converts ontology DomainProposal to knowledge DomainProposal.
func ontologyProposalToKnowledge(p types.DomainProposal) *ktypes.DomainProposal {
	kp := &ktypes.DomainProposal{
		Id:               p.Id,
		Proposer:         p.Proposer,
		Stake:            p.Stake,
		Status:           ontologyStatusToKnowledge(p.Status),
		ProposedAt:       p.CreatedAt,
		VotingEndsAt:     p.ExpiresAt,
		EndorsementStake: fmt.Sprintf("%d", p.VotesFor),
		ChallengeStake:   fmt.Sprintf("%d", p.VotesAgainst),
	}

	if p.Domain != nil {
		kp.ProposedDomain = domainToOntological(*p.Domain)
	}

	return kp
}

// ontologyStatusToKnowledge converts ontology proposal status string to knowledge DomainProposalStatus.
func ontologyStatusToKnowledge(status string) ktypes.DomainProposalStatus {
	switch status {
	case "active":
		return ktypes.DomainProposalStatus_DOMAIN_PROPOSAL_STATUS_PENDING
	case "passed":
		return ktypes.DomainProposalStatus_DOMAIN_PROPOSAL_STATUS_APPROVED
	case "rejected":
		return ktypes.DomainProposalStatus_DOMAIN_PROPOSAL_STATUS_REJECTED
	case "expired":
		return ktypes.DomainProposalStatus_DOMAIN_PROPOSAL_STATUS_EXPIRED
	default:
		return ktypes.DomainProposalStatus_DOMAIN_PROPOSAL_STATUS_UNSPECIFIED
	}
}
