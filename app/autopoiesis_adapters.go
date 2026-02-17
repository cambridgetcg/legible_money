package app

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	lgmgovkeeper "github.com/legible-money/legible/x/gov/keeper"
	lgmgovtypes "github.com/legible-money/legible/x/gov/types"

	autopoiesistypes "github.com/legible-money/legible/x/autopoiesis/types"
	knowledgekeeper "github.com/legible-money/legible/x/knowledge/keeper"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
)

// AutopoiesisGovAdapter bridges the autopoiesis module's GovKeeper interface
// to the LGM governance keeper. The autopoiesis module uses this to observe
// governance activity for auto-proposal rate limiting.
type AutopoiesisGovAdapter struct {
	keeper lgmgovkeeper.Keeper
}

// NewAutopoiesisGovAdapter creates a new adapter wrapping the LGM gov keeper.
func NewAutopoiesisGovAdapter(k lgmgovkeeper.Keeper) *AutopoiesisGovAdapter {
	return &AutopoiesisGovAdapter{keeper: k}
}

// GetActiveProposalCount returns the number of LIPs currently in voting stage.
func (a *AutopoiesisGovAdapter) GetActiveProposalCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	lips := a.keeper.GetLIPsByStage(sdkCtx, lgmgovtypes.StageVoting)
	return uint64(len(lips))
}

// Compile-time interface check.
var _ autopoiesistypes.GovKeeper = (*AutopoiesisGovAdapter)(nil)

// AutopoiesisKnowledgeAdapter bridges the autopoiesis module's KnowledgeKeeper
// interface to the knowledge module's keeper. Wraps sdk.Context ↔ context.Context.
type AutopoiesisKnowledgeAdapter struct {
	keeper knowledgekeeper.Keeper
}

// NewAutopoiesisKnowledgeAdapter creates a new adapter.
func NewAutopoiesisKnowledgeAdapter(k knowledgekeeper.Keeper) *AutopoiesisKnowledgeAdapter {
	return &AutopoiesisKnowledgeAdapter{keeper: k}
}

func (a *AutopoiesisKnowledgeAdapter) GetActiveFactCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetFactCount(sdkCtx)
}

func (a *AutopoiesisKnowledgeAdapter) GetFactsCreatedInRange(ctx context.Context, startHeight, endHeight uint64) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var count uint64
	a.keeper.IterateFacts(sdkCtx, func(f knowledgetypes.Fact) bool {
		if f.CreatedAtHeight >= startHeight && f.CreatedAtHeight <= endHeight {
			count++
		}
		return false
	})
	return count
}

func (a *AutopoiesisKnowledgeAdapter) GetFactsExpiredInRange(_ context.Context, _, _ uint64) uint64 {
	// Fact expiration tracking not yet implemented in knowledge module
	return 0
}

func (a *AutopoiesisKnowledgeAdapter) GetChallengeSuccessRate(_ context.Context) uint64 {
	// Challenge success rate tracking not yet implemented
	return 0
}

func (a *AutopoiesisKnowledgeAdapter) GetActiveDomains(ctx context.Context) []string {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var domains []string
	a.keeper.IterateDomains(sdkCtx, func(d knowledgetypes.OntologicalDomain) bool {
		domains = append(domains, d.Name)
		return false
	})
	return domains
}

func (a *AutopoiesisKnowledgeAdapter) GetDomainObservation(ctx context.Context, domain string, _, _ uint64) autopoiesistypes.DomainObservation {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	facts := a.keeper.GetFactsByDomain(sdkCtx, domain)
	return autopoiesistypes.DomainObservation{
		Domain:           domain,
		TotalActiveFacts: uint64(len(facts)),
	}
}

func (a *AutopoiesisKnowledgeAdapter) GetCategoryRetentions(_ context.Context, _, _ uint64) []autopoiesistypes.CategoryRetention {
	return nil
}

func (a *AutopoiesisKnowledgeAdapter) GetVerifiedProtocolObservationCount(ctx context.Context, sinceBlock uint64) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	facts := a.keeper.GetFactsByDomain(sdkCtx, "protocol")
	var count uint64
	for i := range facts {
		if facts[i].EpistemicCategory == "protocol_observation" &&
			facts[i].Status == "verified" &&
			facts[i].VerifiedAtHeight > sinceBlock {
			count++
		}
	}
	return count
}

// Compile-time interface check.
var _ autopoiesistypes.KnowledgeKeeper = (*AutopoiesisKnowledgeAdapter)(nil)
