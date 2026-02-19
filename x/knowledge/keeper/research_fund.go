package keeper

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// IsEligibleProposer checks if an address has enough verified facts to propose.
// Requires ≥ ResearchMinVerifiedFacts verified/active facts submitted by this address.
func (k Keeper) IsEligibleProposer(ctx sdk.Context, addr string) bool {
	count := k.countVerifiedFactsBySubmitter(ctx, addr)
	return count >= types.ResearchMinVerifiedFacts
}

// IsEligibleVoter checks if an address is eligible to vote on research proposals.
// Eligible if: ≥1 verified fact OR ≥10 verification round participations.
func (k Keeper) IsEligibleVoter(ctx sdk.Context, addr string) bool {
	factCount := k.countVerifiedFactsBySubmitter(ctx, addr)
	if factCount >= types.ResearchVoterMinFacts {
		return true
	}
	// Check validator participation as alternative path
	record, found := k.GetValidatorParticipation(ctx, addr)
	if found && record.RoundsParticipated >= types.ResearchVoterMinRounds {
		return true
	}
	return false
}

// GetVotingWeight returns the voting weight for an address.
// Uses verified fact count as a proxy for reputation.
// TODO(production): Use LegibleAccount.ReputationScore for richer weighting.
func (k Keeper) GetVotingWeight(ctx sdk.Context, addr string) uint64 {
	count := k.countVerifiedFactsBySubmitter(ctx, addr)
	if count == 0 {
		// Fallback: if voter qualified via participation, give weight 1
		return 1
	}
	return uint64(count)
}

// GetResearchFundBalance returns the current balance of the research_fund module account.
func (k Keeper) GetResearchFundBalance(ctx sdk.Context) sdk.Coin {
	moduleAddr := authtypes.NewModuleAddress("research_fund")
	return k.bankKeeper.GetBalance(ctx, moduleAddr, "ulgm")
}

// TallyResearchProposal tallies votes and determines if a proposal passes.
// Returns (pass, quorumMet).
func (k Keeper) TallyResearchProposal(ctx sdk.Context, proposal *types.ResearchProposal) (pass bool, quorumMet bool) {
	eligibleCount := k.GetEligibleVoterCount(ctx)
	if eligibleCount == 0 {
		return false, false
	}

	votes := k.GetVotesByProposal(ctx, proposal.ProposalId)
	voterCount := uint64(len(votes))

	// Quorum: voterCount * 1_000_000 >= eligibleCount * QuorumBps
	// Cross-multiply to avoid division
	if voterCount*1_000_000 < eligibleCount*uint64(types.ResearchQuorumBps) {
		return false, false
	}

	totalWeight := proposal.VotesFor + proposal.VotesAgainst
	if totalWeight == 0 {
		return false, true
	}

	// Pass threshold: VotesFor * 1_000_000 >= totalWeight * PassThresholdBps
	pass = proposal.VotesFor*1_000_000 >= totalWeight*uint64(types.ResearchPassThresholdBps)
	return pass, true
}

// GetEligibleVoterCount returns the number of unique addresses with ≥1 verified fact.
func (k Keeper) GetEligibleVoterCount(ctx sdk.Context) uint64 {
	seen := make(map[string]bool)
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if f.Submitter != "" && (f.Status == types.FactStatusVerified || f.Status == types.FactStatusActive) {
			seen[f.Submitter] = true
		}
		return false
	})
	return uint64(len(seen))
}

// countVerifiedFactsBySubmitter counts verified/active facts for a given submitter.
func (k Keeper) countVerifiedFactsBySubmitter(ctx sdk.Context, addr string) int {
	count := 0
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if f.Submitter == addr && (f.Status == types.FactStatusVerified || f.Status == types.FactStatusActive) {
			count++
		}
		return false
	})
	return count
}
