package app

import (
	"context"
	"math/big"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	alignmenttypes "github.com/legible-money/legible/x/alignment/types"
	autopoiesiskeeper "github.com/legible-money/legible/x/autopoiesis/keeper"
	autopoiesistypes "github.com/legible-money/legible/x/autopoiesis/types"
	emergencykeeper "github.com/legible-money/legible/x/emergency/keeper"
	knowledgekeeper "github.com/legible-money/legible/x/knowledge/keeper"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
	ontologykeeper "github.com/legible-money/legible/x/ontology/keeper"
	legiblestakingkeeper "github.com/legible-money/legible/x/staking/keeper"
	legiblestakingtypes "github.com/legible-money/legible/x/staking/types"
	vestingrewardskeeper "github.com/legible-money/legible/x/vesting_rewards/keeper"
)

// AlignmentAutopoiesisAdapter bridges the alignment module's AutopoiesisKeeper
// interface to the actual autopoiesis keeper. The alignment module uses this to
// read and write multiplier paths, and to observe autopoiesis internal state
// for the Harmony sensor.
type AlignmentAutopoiesisAdapter struct {
	keeper autopoiesiskeeper.Keeper
}

func NewAlignmentAutopoiesisAdapter(k autopoiesiskeeper.Keeper) *AlignmentAutopoiesisAdapter {
	return &AlignmentAutopoiesisAdapter{keeper: k}
}

var _ alignmenttypes.AutopoiesisKeeper = (*AlignmentAutopoiesisAdapter)(nil)

func (a *AlignmentAutopoiesisAdapter) GetMultiplier(ctx context.Context, path string) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetMultiplier(sdkCtx, path)
}

func (a *AlignmentAutopoiesisAdapter) SetMultiplier(ctx context.Context, entry alignmenttypes.MultiplierEntry) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	a.keeper.SetMultiplier(sdkCtx, autopoiesistypes.MultiplierEntry{
		Path:    entry.Path,
		Value:   entry.Value,
		Hormone: entry.Hormone,
		Epoch:   entry.Epoch,
	})
}

func (a *AlignmentAutopoiesisAdapter) GetMultiplierEntry(ctx context.Context, path string) *alignmenttypes.MultiplierEntry {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	entry := a.keeper.GetMultiplierEntry(sdkCtx, path)
	if entry == nil {
		return nil
	}
	return &alignmenttypes.MultiplierEntry{
		Path:    entry.Path,
		Value:   entry.Value,
		Hormone: entry.Hormone,
		Epoch:   entry.Epoch,
	}
}

func (a *AlignmentAutopoiesisAdapter) GetAllMultiplierEntries(ctx context.Context) []alignmenttypes.MultiplierEntry {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	entries := a.keeper.GetAllMultiplierEntries(sdkCtx)
	result := make([]alignmenttypes.MultiplierEntry, len(entries))
	for i, e := range entries {
		result[i] = alignmenttypes.MultiplierEntry{
			Path:    e.Path,
			Value:   e.Value,
			Hormone: e.Hormone,
			Epoch:   e.Epoch,
		}
	}
	return result
}

func (a *AlignmentAutopoiesisAdapter) GetAdaptationCount(ctx context.Context, epoch uint64) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetAdaptationCount(sdkCtx, epoch)
}

func (a *AlignmentAutopoiesisAdapter) GetConflictCount(ctx context.Context, epoch uint64) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetConflictCount(sdkCtx, epoch)
}

func (a *AlignmentAutopoiesisAdapter) GetDirectionChanges(ctx context.Context, lastNEpochs uint64) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetDirectionChanges(sdkCtx, lastNEpochs)
}

func (a *AlignmentAutopoiesisAdapter) IsActive(ctx context.Context) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.IsActive(sdkCtx)
}

// ============================================================
// AlignmentEmergencyAdapter — bridges EmergencyKeeper interface
// ============================================================

type AlignmentEmergencyAdapter struct {
	keeper emergencykeeper.Keeper
}

func NewAlignmentEmergencyAdapter(k emergencykeeper.Keeper) *AlignmentEmergencyAdapter {
	return &AlignmentEmergencyAdapter{keeper: k}
}

var _ alignmenttypes.EmergencyKeeper = (*AlignmentEmergencyAdapter)(nil)

func (a *AlignmentEmergencyAdapter) IsHalted(ctx context.Context) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.IsHalted(sdkCtx)
}

// ============================================================
// AlignmentKnowledgeAdapter — bridges KnowledgeKeeper interface
// ============================================================

type AlignmentKnowledgeAdapter struct {
	keeper knowledgekeeper.Keeper
}

func NewAlignmentKnowledgeAdapter(k knowledgekeeper.Keeper) *AlignmentKnowledgeAdapter {
	return &AlignmentKnowledgeAdapter{keeper: k}
}

var _ alignmenttypes.KnowledgeKeeper = (*AlignmentKnowledgeAdapter)(nil)

// --- Truth sensor (TDI) ---

func (a *AlignmentKnowledgeAdapter) GetVerifiedFactCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var count uint64
	a.keeper.IterateFacts(sdkCtx, func(f knowledgetypes.Fact) bool {
		if f.Status == knowledgetypes.FactStatusVerified {
			count++
		}
		return false
	})
	return count
}

func (a *AlignmentKnowledgeAdapter) GetSupersededFactCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var count uint64
	a.keeper.IterateFacts(sdkCtx, func(f knowledgetypes.Fact) bool {
		if f.Status == knowledgetypes.FactStatusSuperseded {
			count++
		}
		return false
	})
	return count
}

func (a *AlignmentKnowledgeAdapter) GetTotalChallenges(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var count uint64
	a.keeper.IterateCounterFactChallenges(sdkCtx, func(_ knowledgetypes.CounterFactChallenge) bool {
		count++
		return false
	})
	return count
}

func (a *AlignmentKnowledgeAdapter) GetChallengesByTopValidators(_ context.Context, _ int) uint64 {
	// Stub: no per-validator challenge index available yet
	return 0
}

func (a *AlignmentKnowledgeAdapter) GetUnanimousRoundCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var count uint64
	a.keeper.IterateVerificationRounds(sdkCtx, func(r knowledgetypes.VerificationRound) bool {
		if r.Phase != "complete" {
			return false
		}
		if len(r.Reveals) == 0 {
			return false
		}
		// Check if all reveals have the same verdict
		var firstVerdict string
		unanimous := true
		for _, rev := range r.Reveals {
			if rev == nil {
				continue
			}
			if firstVerdict == "" {
				firstVerdict = rev.Verdict
			} else if rev.Verdict != firstVerdict {
				unanimous = false
				break
			}
		}
		if unanimous && firstVerdict != "" {
			count++
		}
		return false
	})
	return count
}

func (a *AlignmentKnowledgeAdapter) GetTotalRoundCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var count uint64
	a.keeper.IterateVerificationRounds(sdkCtx, func(r knowledgetypes.VerificationRound) bool {
		if r.Phase == "complete" {
			count++
		}
		return false
	})
	return count
}

func (a *AlignmentKnowledgeAdapter) GetCounterFactCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var count uint64
	a.keeper.IterateCounterFacts(sdkCtx, func(_ knowledgetypes.CounterFact) bool {
		count++
		return false
	})
	return count
}

// --- Understanding sensor (DCI) ---

func (a *AlignmentKnowledgeAdapter) GetAvgCitationChainLength(_ context.Context) uint64 {
	// Stub: recursive chain traversal too expensive for EndBlock
	return 0
}

func (a *AlignmentKnowledgeAdapter) GetCrossDomainCitationRatio(_ context.Context) uint64 {
	// Stub: needs domain-crossing reference analysis
	return 0
}

func (a *AlignmentKnowledgeAdapter) GetOrphanFactCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Count facts that have no outgoing references (no domain).
	// Approximation: facts with empty domain are orphans.
	var count uint64
	a.keeper.IterateFacts(sdkCtx, func(f knowledgetypes.Fact) bool {
		if f.Domain == "" {
			count++
		}
		return false
	})
	return count
}

func (a *AlignmentKnowledgeAdapter) GetActiveFactCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetFactCount(sdkCtx)
}

func (a *AlignmentKnowledgeAdapter) GetAvgSupersessionChainLength(_ context.Context) uint64 {
	// Stub: no supersession chain tracking
	return 0
}

// --- Novelty sensor (NI) ---

func (a *AlignmentKnowledgeAdapter) GetTotalFactCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return a.keeper.GetFactCount(sdkCtx)
}

func (a *AlignmentKnowledgeAdapter) GetFactsInNewCategories(_ context.Context, _, _ uint64) uint64 {
	// Stub: needs per-epoch creation height indexing
	return 0
}

func (a *AlignmentKnowledgeAdapter) GetNewCrossDomainEdges(_ context.Context, _, _ uint64) uint64 {
	// Stub: needs edge creation height tracking
	return 0
}

func (a *AlignmentKnowledgeAdapter) GetDomainFactDistribution(ctx context.Context) []alignmenttypes.DomainFactCount {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Collect domains and their fact counts
	var result []alignmenttypes.DomainFactCount
	a.keeper.IterateDomains(sdkCtx, func(d knowledgetypes.OntologicalDomain) bool {
		count := a.keeper.GetFactCountByDomain(sdkCtx, d.Name)
		if count > 0 {
			result = append(result, alignmenttypes.DomainFactCount{
				Domain: d.Name,
				Count:  count,
			})
		}
		return false
	})
	return result
}

// --- Fairness sensor (FI) ---

func (a *AlignmentKnowledgeAdapter) GetNewSubmitterAcceptRate(_ context.Context) uint64 {
	// Stub: no acceptance rate tracking
	return 0
}

// ============================================================
// AlignmentStakingAdapter — bridges StakingKeeper interface
// ============================================================

type AlignmentStakingAdapter struct {
	keeper legiblestakingkeeper.Keeper
}

func NewAlignmentStakingAdapter(k legiblestakingkeeper.Keeper) *AlignmentStakingAdapter {
	return &AlignmentStakingAdapter{keeper: k}
}

var _ alignmenttypes.StakingKeeper = (*AlignmentStakingAdapter)(nil)

func (a *AlignmentStakingAdapter) GetValidatorRewardDistribution(ctx context.Context) []alignmenttypes.ValidatorReward {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Use total stake as a proxy for reward distribution (actual per-epoch
	// reward tracking not yet available).
	var rewards []alignmenttypes.ValidatorReward
	a.keeper.IterateValidators(sdkCtx, func(v legiblestakingkeeper.ValidatorRef) bool {
		stake, ok := new(big.Int).SetString(v.TotalStake, 10)
		if !ok || stake.Sign() <= 0 {
			return false
		}
		// Convert to uint64 (ulgm); clamp if exceeds uint64 range
		amount := uint64(0)
		if stake.IsUint64() {
			amount = stake.Uint64()
		} else {
			amount = ^uint64(0) // max uint64 as clamp
		}
		rewards = append(rewards, alignmenttypes.ValidatorReward{
			Validator: v.OperatorAddress,
			Amount:    amount,
		})
		return false
	})
	return rewards
}

func (a *AlignmentStakingAdapter) GetTop10StakeRatio(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	type valStake struct {
		stake *big.Int
	}
	var vals []valStake
	a.keeper.IterateValidators(sdkCtx, func(v legiblestakingkeeper.ValidatorRef) bool {
		s, ok := new(big.Int).SetString(v.TotalStake, 10)
		if ok && s.Sign() > 0 {
			vals = append(vals, valStake{stake: s})
		}
		return false
	})

	if len(vals) == 0 {
		return 0
	}

	// Sort descending by stake
	sort.Slice(vals, func(i, j int) bool {
		return vals[i].stake.Cmp(vals[j].stake) > 0
	})

	total := new(big.Int)
	for _, v := range vals {
		total.Add(total, v.stake)
	}
	if total.Sign() == 0 {
		return 0
	}

	topN := 10
	if len(vals) < topN {
		topN = len(vals)
	}
	top10Sum := new(big.Int)
	for i := 0; i < topN; i++ {
		top10Sum.Add(top10Sum, vals[i].stake)
	}

	// BPS: (top10Sum * 1_000_000) / total
	bps := new(big.Int).Mul(top10Sum, big.NewInt(1_000_000))
	bps.Div(bps, total)
	return bps.Uint64()
}

func (a *AlignmentStakingAdapter) GetTotalStake(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	total := new(big.Int)
	a.keeper.IterateValidators(sdkCtx, func(v legiblestakingkeeper.ValidatorRef) bool {
		s, ok := new(big.Int).SetString(v.TotalStake, 10)
		if ok && s.Sign() > 0 {
			total.Add(total, s)
		}
		return false
	})
	if total.IsUint64() {
		return total.Uint64()
	}
	return ^uint64(0) // max uint64 clamp
}

func (a *AlignmentStakingAdapter) GetAvgPairwiseAgreement(_ context.Context) uint64 {
	// Stub: neutral default (50% in BPS)
	return 500_000
}

func (a *AlignmentStakingAdapter) GetSlashProportionality(_ context.Context) uint64 {
	// Stub: neutral default (50% in BPS)
	return 500_000
}

// ============================================================
// AlignmentOntologyAdapter — bridges OntologyKeeper interface
// ============================================================

type AlignmentOntologyAdapter struct {
	keeper ontologykeeper.Keeper
}

func NewAlignmentOntologyAdapter(k ontologykeeper.Keeper) *AlignmentOntologyAdapter {
	return &AlignmentOntologyAdapter{keeper: k}
}

var _ alignmenttypes.OntologyKeeper = (*AlignmentOntologyAdapter)(nil)

func (a *AlignmentOntologyAdapter) GetActiveDomainCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return uint64(len(a.keeper.GetAllDomains(sdkCtx)))
}

func (a *AlignmentOntologyAdapter) GetDomainsWithActivity(_ context.Context, _, _ uint64) uint64 {
	// Stub: needs creation-height tracking on domains
	return 0
}

func (a *AlignmentOntologyAdapter) GetNewDomainsInEpoch(_ context.Context, _, _ uint64) uint64 {
	// Stub: needs creation-height tracking on domains
	return 0
}

func (a *AlignmentOntologyAdapter) GetTotalDomainCount(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return uint64(len(a.keeper.GetAllDomains(sdkCtx)))
}

// ============================================================
// AlignmentVestingRewardsAdapter — bridges VestingRewardsKeeper
// ============================================================

type AlignmentVestingRewardsAdapter struct {
	keeper vestingrewardskeeper.Keeper
}

func NewAlignmentVestingRewardsAdapter(k vestingrewardskeeper.Keeper) *AlignmentVestingRewardsAdapter {
	return &AlignmentVestingRewardsAdapter{keeper: k}
}

var _ alignmenttypes.VestingRewardsKeeper = (*AlignmentVestingRewardsAdapter)(nil)

func (a *AlignmentVestingRewardsAdapter) GetValidatorRewardsInEpoch(_ context.Context, _, _ uint64) []alignmenttypes.ValidatorReward {
	// Stub: per-epoch reward tracking not available yet
	return nil
}
