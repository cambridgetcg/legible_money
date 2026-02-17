package keeper

import (
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/alignment/types"
)

// ============ Signal Collection ============

// collectSignals gathers raw alignment signals from dependent keepers.
func (k Keeper) collectSignals(ctx sdk.Context, epoch, blockHeight, epochStart uint64) types.AlignmentObservation {
	obs := types.AlignmentObservation{
		Epoch:       epoch,
		BlockHeight: blockHeight,
	}

	// --- Truth signals ---
	k.collectTruthSignals(ctx, &obs)

	// --- Understanding signals ---
	k.collectUnderstandingSignals(ctx, &obs, epochStart, blockHeight)

	// --- Harmony signals ---
	k.collectHarmonySignals(ctx, &obs, epoch)

	// --- Fairness signals ---
	k.collectFairnessSignals(ctx, &obs, epochStart, blockHeight)

	// --- Novelty signals ---
	k.collectNoveltySignals(ctx, &obs, epochStart, blockHeight)

	return obs
}

func (k Keeper) collectTruthSignals(ctx sdk.Context, obs *types.AlignmentObservation) {
	if k.knowledgeKeeper == nil {
		return
	}

	verifiedFacts := k.knowledgeKeeper.GetVerifiedFactCount(ctx)
	if verifiedFacts > 0 {
		superseded := k.knowledgeKeeper.GetSupersededFactCount(ctx)
		obs.ReversalRate = (superseded * types.BasisPoints) / verifiedFacts
	}

	totalChallenges := k.knowledgeKeeper.GetTotalChallenges(ctx)
	if totalChallenges > 0 {
		topNChallenges := k.knowledgeKeeper.GetChallengesByTopValidators(ctx, 10)
		obs.ChallengeAsymmetry = (topNChallenges * types.BasisPoints) / totalChallenges
	}

	totalRounds := k.knowledgeKeeper.GetTotalRoundCount(ctx)
	if totalRounds > 0 {
		unanimousRounds := k.knowledgeKeeper.GetUnanimousRoundCount(ctx)
		obs.UnanimityRate = (unanimousRounds * types.BasisPoints) / totalRounds
	}

	if k.stakingKeeper != nil {
		obs.CrossValidatorCorr = k.stakingKeeper.GetAvgPairwiseAgreement(ctx)
	}

	totalFacts := k.knowledgeKeeper.GetTotalFactCount(ctx)
	if totalFacts > 0 {
		counterFacts := k.knowledgeKeeper.GetCounterFactCount(ctx)
		obs.CounterFactRatio = (counterFacts * types.BasisPoints) / totalFacts
	}
}

func (k Keeper) collectUnderstandingSignals(ctx sdk.Context, obs *types.AlignmentObservation, epochStart, epochEnd uint64) {
	if k.knowledgeKeeper == nil {
		return
	}

	obs.AvgCitationDepth = k.knowledgeKeeper.GetAvgCitationChainLength(ctx)
	obs.CrossDomainRatio = k.knowledgeKeeper.GetCrossDomainCitationRatio(ctx)

	activeFacts := k.knowledgeKeeper.GetActiveFactCount(ctx)
	if activeFacts > 0 {
		orphans := k.knowledgeKeeper.GetOrphanFactCount(ctx)
		obs.OrphanRate = (orphans * types.BasisPoints) / activeFacts
	}

	if k.ontologyKeeper != nil {
		totalDomains := k.ontologyKeeper.GetTotalDomainCount(ctx)
		if totalDomains > 0 {
			domainsWithActivity := k.ontologyKeeper.GetDomainsWithActivity(ctx, epochStart, epochEnd)
			obs.OntologyCoverage = (domainsWithActivity * types.BasisPoints) / totalDomains
		}
	}

	obs.SupersessionDepth = k.knowledgeKeeper.GetAvgSupersessionChainLength(ctx)
}

func (k Keeper) collectHarmonySignals(ctx sdk.Context, obs *types.AlignmentObservation, epoch uint64) {
	if k.autopoiesisKeeper == nil {
		return
	}

	// Multiplier variance: compute variance of all active multipliers
	entries := k.autopoiesisKeeper.GetAllMultiplierEntries(ctx)
	if len(entries) > 0 {
		obs.MultiplierVariance = computeMultiplierVariance(entries)
	}

	// Oscillation: direction changes over recent epochs
	lastN := uint64(10)
	if epoch < lastN {
		lastN = epoch
	}
	if lastN > 0 {
		dirChanges := k.autopoiesisKeeper.GetDirectionChanges(ctx, lastN)
		// Normalize: max expected direction changes = lastN per multiplier
		maxChanges := lastN
		if maxChanges > 0 {
			obs.OscillationRate = types.ClampUint64(
				(dirChanges*types.BasisPoints)/maxChanges, 0, types.BasisPoints)
		}
	}

	// Bound saturation: count multipliers at their min or max bounds
	atBounds := uint64(0)
	totalEntries := uint64(len(entries))
	for _, e := range entries {
		// Heuristic: consider "at bounds" if within 5% of known bounds
		if e.Value <= 300_000 || e.Value >= 3_500_000 {
			atBounds++
		}
	}
	if totalEntries > 0 {
		obs.BoundSaturation = (atBounds * types.BasisPoints) / totalEntries
	}

	// Hormone conflict rate
	if epoch > 0 {
		adaptations := k.autopoiesisKeeper.GetAdaptationCount(ctx, epoch-1)
		conflicts := k.autopoiesisKeeper.GetConflictCount(ctx, epoch-1)
		if adaptations > 0 {
			obs.HormoneConflictRate = (conflicts * types.BasisPoints) / adaptations
		}
	}

	// Avg damping: measure how far multipliers are from 1.0x on average
	if totalEntries > 0 {
		var totalDamping uint64
		for _, e := range entries {
			if e.Value >= types.DefaultMultiplier {
				dist := e.Value - types.DefaultMultiplier
				damping := types.BasisPoints - types.ClampUint64(
					(dist*types.BasisPoints)/(3*types.BasisPoints), 0, types.BasisPoints)
				totalDamping += damping
			} else {
				dist := types.DefaultMultiplier - e.Value
				damping := types.BasisPoints - types.ClampUint64(
					(dist*types.BasisPoints)/(types.BasisPoints), 0, types.BasisPoints)
				totalDamping += damping
			}
		}
		obs.AvgDampingFactor = totalDamping / totalEntries
	} else {
		obs.AvgDampingFactor = types.BasisPoints // no multipliers = perfect damping
	}
}

func (k Keeper) collectFairnessSignals(ctx sdk.Context, obs *types.AlignmentObservation, epochStart, epochEnd uint64) {
	if k.stakingKeeper != nil {
		// Reward Gini
		rewards := k.stakingKeeper.GetValidatorRewardDistribution(ctx)
		if len(rewards) > 1 {
			amounts := make([]uint64, len(rewards))
			for i, r := range rewards {
				amounts[i] = r.Amount
			}
			sort.Slice(amounts, func(i, j int) bool { return amounts[i] < amounts[j] })
			obs.RewardGini = types.ComputeGiniCoefficient(amounts)
		}

		obs.SlashProportionality = k.stakingKeeper.GetSlashProportionality(ctx)
		obs.WhaleDominance = k.stakingKeeper.GetTop10StakeRatio(ctx)
	}

	if k.knowledgeKeeper != nil {
		obs.NewEntrantSuccess = k.knowledgeKeeper.GetNewSubmitterAcceptRate(ctx)

		// Domain equity (Gini of fact counts per domain)
		domainDist := k.knowledgeKeeper.GetDomainFactDistribution(ctx)
		if len(domainDist) > 1 {
			counts := make([]uint64, len(domainDist))
			for i, d := range domainDist {
				counts[i] = d.Count
			}
			sort.Slice(counts, func(i, j int) bool { return counts[i] < counts[j] })
			obs.DomainGini = types.ComputeGiniCoefficient(counts)
		}
	}
}

func (k Keeper) collectNoveltySignals(ctx sdk.Context, obs *types.AlignmentObservation, epochStart, epochEnd uint64) {
	if k.ontologyKeeper != nil {
		totalDomains := k.ontologyKeeper.GetTotalDomainCount(ctx)
		if totalDomains > 0 {
			newDomains := k.ontologyKeeper.GetNewDomainsInEpoch(ctx, epochStart, epochEnd)
			obs.NewDomainRate = (newDomains * types.BasisPoints) / totalDomains
		}
	}

	if k.knowledgeKeeper != nil {
		totalFacts := k.knowledgeKeeper.GetTotalFactCount(ctx)
		if totalFacts > 0 {
			novelClaims := k.knowledgeKeeper.GetFactsInNewCategories(ctx, epochStart, epochEnd)
			obs.NovelClaimRatio = (novelClaims * types.BasisPoints) / totalFacts

			// Supersession rate
			superseded := k.knowledgeKeeper.GetSupersededFactCount(ctx)
			obs.SupersessionRate = (superseded * types.BasisPoints) / totalFacts
		}

		activeFacts := k.knowledgeKeeper.GetActiveFactCount(ctx)
		if activeFacts > 0 {
			newBridges := k.knowledgeKeeper.GetNewCrossDomainEdges(ctx, epochStart, epochEnd)
			obs.CrossDomainBridges = (newBridges * types.BasisPoints) / activeFacts
		}

		// Fact diversity (Shannon entropy of domain distribution)
		domainDist := k.knowledgeKeeper.GetDomainFactDistribution(ctx)
		if len(domainDist) > 0 {
			counts := make([]uint64, len(domainDist))
			for i, d := range domainDist {
				counts[i] = d.Count
			}
			obs.FactDiversity = types.ComputeShannonEntropy(counts)
		}
	}
}

// ============ Score Computation ============

// computeScores computes all 5 dimension scores from raw signals.
func (k Keeper) computeScores(obs types.AlignmentObservation, state types.AlignmentState) types.DimensionScores {
	return types.DimensionScores{
		TDI: ComputeTDI(obs),
		DCI: ComputeDCI(obs),
		HI:  ComputeHI(obs),
		FI:  ComputeFI(obs),
		NI:  ComputeNI(obs),
	}
}

// ComputeTDI computes Truth Drift Index.
// Returns 0-1M BPS where 0 = perfect truth, 1M = total corruption.
func ComputeTDI(obs types.AlignmentObservation) uint64 {
	reversal := types.ClampUint64(obs.ReversalRate, 0, types.BasisPoints)
	asymmetry := types.ClampUint64(obs.ChallengeAsymmetry, 0, types.BasisPoints)

	// Unanimity excess: only count excess above 80% as suspicious
	unanimityExcess := uint64(0)
	if obs.UnanimityRate > types.TDIUnanimityThreshold {
		unanimityExcess = obs.UnanimityRate - types.TDIUnanimityThreshold
		// Scale to 0-1M: the remaining 20% range maps to 0-1M
		unanimityExcess = (unanimityExcess * types.BasisPoints) / (types.BasisPoints - types.TDIUnanimityThreshold)
	}

	// Cross-validator correlation: only excess above 90% is suspicious
	corrExcess := uint64(0)
	if obs.CrossValidatorCorr > types.TDICrossValidatorThresh {
		corrExcess = obs.CrossValidatorCorr - types.TDICrossValidatorThresh
		corrExcess = (corrExcess * types.BasisPoints) / (types.BasisPoints - types.TDICrossValidatorThresh)
	}

	// Counter-fact deficit: low counter-fact ratio is concerning (inverted)
	counterDeficit := uint64(0)
	if obs.CounterFactRatio < types.TDICounterFactMinimum {
		counterDeficit = types.TDICounterFactMinimum - obs.CounterFactRatio
		counterDeficit = (counterDeficit * types.BasisPoints) / types.TDICounterFactMinimum
	}

	tdi := (types.TDIWeightReversalRate*reversal +
		types.TDIWeightChallengeAsymmetry*asymmetry +
		types.TDIWeightUnanimityExcess*unanimityExcess +
		types.TDIWeightCrossValidatorCorr*corrExcess +
		types.TDIWeightCounterFactDeficit*counterDeficit) / types.BasisPoints

	return types.ClampUint64(tdi, 0, types.BasisPoints)
}

// ComputeDCI computes Depth Coherence Index.
// Returns 0-1M BPS where 1M = deep coherent understanding, 0 = random trivia.
func ComputeDCI(obs types.AlignmentObservation) uint64 {
	// Citation depth: normalize against target (depth 5 = perfect)
	citationScore := types.ClampUint64(
		(obs.AvgCitationDepth*types.BasisPoints)/types.DCICitationDepthTarget, 0, types.BasisPoints)

	crossDomainScore := types.ClampUint64(obs.CrossDomainRatio, 0, types.BasisPoints)

	// Orphan rate: inverted (lower orphan rate = better)
	orphanScore := uint64(0)
	if obs.OrphanRate < types.BasisPoints {
		orphanScore = types.BasisPoints - obs.OrphanRate
	}

	ontologyScore := types.ClampUint64(obs.OntologyCoverage, 0, types.BasisPoints)

	// Supersession depth: normalize against target
	supersessionScore := types.ClampUint64(
		(obs.SupersessionDepth*types.BasisPoints)/types.DCISupersessionDepthTarget, 0, types.BasisPoints)

	dci := (types.DCIWeightCitationDepth*citationScore +
		types.DCIWeightCrossDomain*crossDomainScore +
		types.DCIWeightOrphanRate*orphanScore +
		types.DCIWeightOntologyCoverage*ontologyScore +
		types.DCIWeightSupersession*supersessionScore) / types.BasisPoints

	return types.ClampUint64(dci, 0, types.BasisPoints)
}

// ComputeHI computes Harmony Index.
// Returns 0-1M BPS where 1M = harmonious, 0 = chaotic.
func ComputeHI(obs types.AlignmentObservation) uint64 {
	varianceScore := uint64(0)
	if obs.MultiplierVariance < types.BasisPoints {
		varianceScore = types.BasisPoints - obs.MultiplierVariance
	}

	oscillationScore := uint64(0)
	if obs.OscillationRate < types.BasisPoints {
		oscillationScore = types.BasisPoints - obs.OscillationRate
	}

	boundScore := uint64(0)
	if obs.BoundSaturation < types.BasisPoints {
		boundScore = types.BasisPoints - obs.BoundSaturation
	}

	conflictScore := uint64(0)
	if obs.HormoneConflictRate < types.BasisPoints {
		conflictScore = types.BasisPoints - obs.HormoneConflictRate
	}

	dampingScore := types.ClampUint64(obs.AvgDampingFactor, 0, types.BasisPoints)

	hi := (types.HIWeightVariance*varianceScore +
		types.HIWeightOscillation*oscillationScore +
		types.HIWeightBoundSaturation*boundScore +
		types.HIWeightHormoneConflict*conflictScore +
		types.HIWeightDamping*dampingScore) / types.BasisPoints

	return types.ClampUint64(hi, 0, types.BasisPoints)
}

// ComputeFI computes Fairness Index.
// Returns 0-1M BPS where 1M = perfectly fair, 0 = oligarchic.
func ComputeFI(obs types.AlignmentObservation) uint64 {
	// Reward Gini: inverted (lower Gini = more fair)
	rewardScore := uint64(0)
	if obs.RewardGini < types.BasisPoints {
		rewardScore = types.BasisPoints - obs.RewardGini
	}

	newEntrantScore := types.ClampUint64(obs.NewEntrantSuccess, 0, types.BasisPoints)

	slashScore := types.ClampUint64(obs.SlashProportionality, 0, types.BasisPoints)

	// Whale dominance: inverted
	whaleScore := uint64(0)
	if obs.WhaleDominance < types.BasisPoints {
		whaleScore = types.BasisPoints - obs.WhaleDominance
	}

	// Domain Gini: inverted
	domainScore := uint64(0)
	if obs.DomainGini < types.BasisPoints {
		domainScore = types.BasisPoints - obs.DomainGini
	}

	fi := (types.FIWeightRewardGini*rewardScore +
		types.FIWeightNewEntrant*newEntrantScore +
		types.FIWeightSlashProportion*slashScore +
		types.FIWeightWhaleDominance*whaleScore +
		types.FIWeightDomainEquity*domainScore) / types.BasisPoints

	return types.ClampUint64(fi, 0, types.BasisPoints)
}

// ComputeNI computes Novelty Index.
// Returns 0-1M BPS where 1M = highly creative, 0 = stagnant.
func ComputeNI(obs types.AlignmentObservation) uint64 {
	// New domain rate: normalize against target
	newDomainScore := types.ClampUint64(
		(obs.NewDomainRate*types.BasisPoints)/types.NINewDomainTarget, 0, types.BasisPoints)

	novelClaimScore := types.ClampUint64(
		(obs.NovelClaimRatio*types.BasisPoints)/types.NINovelClaimTarget, 0, types.BasisPoints)

	crossBridgeScore := types.ClampUint64(
		(obs.CrossDomainBridges*types.BasisPoints)/types.NICrossBridgeTarget, 0, types.BasisPoints)

	diversityScore := types.ClampUint64(obs.FactDiversity, 0, types.BasisPoints)

	supersessionScore := types.ClampUint64(obs.SupersessionRate, 0, types.BasisPoints)

	ni := (types.NIWeightNewDomains*newDomainScore +
		types.NIWeightNovelClaims*novelClaimScore +
		types.NIWeightCrossBridges*crossBridgeScore +
		types.NIWeightFactDiversity*diversityScore +
		types.NIWeightSupersessionRate*supersessionScore) / types.BasisPoints

	return types.ClampUint64(ni, 0, types.BasisPoints)
}

// ============ Helpers ============

// computeMultiplierVariance computes normalized variance of multiplier values.
// Returns BPS (0 = all equal, 1M = maximum spread).
func computeMultiplierVariance(entries []types.MultiplierEntry) uint64 {
	n := uint64(len(entries))
	if n <= 1 {
		return 0
	}

	// Compute mean
	var sum uint64
	for _, e := range entries {
		sum += e.Value
	}
	mean := sum / n

	// Compute variance (sum of squared deviations / n)
	var varianceSum uint64
	for _, e := range entries {
		var diff uint64
		if e.Value >= mean {
			diff = e.Value - mean
		} else {
			diff = mean - e.Value
		}
		// diff is in BPS, square gives BPS^2, divide by BPS to get BPS
		varianceSum += (diff * diff) / types.BasisPoints
	}
	variance := varianceSum / n

	// Normalize: max reasonable variance is BasisPoints (1.0)
	return types.ClampUint64(variance, 0, types.BasisPoints)
}
