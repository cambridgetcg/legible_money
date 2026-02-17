package keeper

import (
	"encoding/json"
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// Maturity stages for agent homes.
// Ported from TS home/aging.ts.
const (
	MaturityFresh       = "fresh"       // 0-34272 blocks (~1 day)
	MaturityEstablished = "established" // 34272-342720 blocks (~10 days)
	MaturityMature      = "mature"      // 342720-1028160 blocks (~30 days)
	MaturityVenerable   = "venerable"   // 1028160-3426720 blocks (~100 days)
	MaturityAncient     = "ancient"     // 3426720+ blocks (~100+ days)
)

// Maturity thresholds in blocks (at 2521ms block time, 34272 blocks/day).
const (
	BlocksPerDay         = 34272
	ThresholdEstablished = BlocksPerDay * 1   // 1 day
	ThresholdMature      = BlocksPerDay * 10  // 10 days
	ThresholdVenerable   = BlocksPerDay * 30  // 30 days
	ThresholdAncient     = BlocksPerDay * 100 // 100 days
)

// Reward multiplier by maturity stage (basis points: 10000 = 1.0x).
const (
	RewardMultiplierFresh       = 10000 // 1.0x
	RewardMultiplierEstablished = 10500 // 1.05x
	RewardMultiplierMature      = 11000 // 1.10x
	RewardMultiplierVenerable   = 11500 // 1.15x
	RewardMultiplierAncient     = 12000 // 1.20x
)

// ComputeMaturityStage returns the maturity stage for a home based on its age.
func ComputeMaturityStage(ageBlocks uint64) string {
	switch {
	case ageBlocks >= ThresholdAncient:
		return MaturityAncient
	case ageBlocks >= ThresholdVenerable:
		return MaturityVenerable
	case ageBlocks >= ThresholdMature:
		return MaturityMature
	case ageBlocks >= ThresholdEstablished:
		return MaturityEstablished
	default:
		return MaturityFresh
	}
}

// GetRewardMultiplierBps returns the reward multiplier in basis points for a maturity stage.
func GetRewardMultiplierBps(stage string) uint32 {
	switch stage {
	case MaturityAncient:
		return RewardMultiplierAncient
	case MaturityVenerable:
		return RewardMultiplierVenerable
	case MaturityMature:
		return RewardMultiplierMature
	case MaturityEstablished:
		return RewardMultiplierEstablished
	default:
		return RewardMultiplierFresh
	}
}

// ComputePatina calculates the patina score (0-100) for a home.
// Patina combines age, epistemic activity, and stability.
//
// Formula: patina = min(100, ageFactor + epistemicFactor + stabilityFactor)
// where:
//   - ageFactor = log2(1 + ageBlocks/BlocksPerDay) * 10 (capped at 40)
//   - epistemicFactor = EpistemicStats.EpistemicScore() * 40 / 100 (capped at 40)
//   - stabilityFactor = min(1.0, 1 - statusChanges/10) * 20 (capped at 20)
//
// The epistemic factor replaces the old block-activity factor, rewarding
// actual knowledge contributions (verifications, accuracy, domain breadth)
// rather than just being online.
func ComputePatina(ageBlocks uint64, epistemicScore uint32, statusChanges uint32) uint32 {
	// Age factor: logarithmic growth, max 40 points (reduced from 60)
	ageDays := float64(ageBlocks) / float64(BlocksPerDay)
	ageFactor := math.Log2(1+ageDays) * 10
	if ageFactor > 40 {
		ageFactor = 40
	}

	// Epistemic factor: knowledge contribution, max 40 points
	// epistemicScore is 0-100, scale to 0-40
	epistemicFactor := float64(epistemicScore) * 40.0 / 100.0

	// Stability factor: fewer status changes = more stability, max 20 points
	stabilityRatio := 1.0 - float64(statusChanges)/10.0
	if stabilityRatio < 0 {
		stabilityRatio = 0
	}
	stabilityFactor := stabilityRatio * 20

	patina := ageFactor + epistemicFactor + stabilityFactor
	if patina > 100 {
		patina = 100
	}
	return uint32(patina)
}

// UpdateHomeMaturity computes and updates the maturity stage for a single home.
// Returns true if the stage changed.
func (k Keeper) UpdateHomeMaturity(ctx sdk.Context, home *types.AgentHome) bool {
	height := uint64(ctx.BlockHeight())
	if height < home.CreatedAtBlock {
		return false
	}

	ageBlocks := height - home.CreatedAtBlock
	newStage := ComputeMaturityStage(ageBlocks)

	// Maturity is stored in the Guardian's DefenseStrategy comment field
	// or as a separate field. Since we can't modify proto, use a convention:
	// Store maturity in ComfortScore's high bits, or use a dedicated KV store key.
	// For now, we store maturity stage in a dedicated KV prefix.
	currentStage := k.GetHomeMaturity(ctx, home.HomeId)
	if currentStage == newStage {
		return false
	}

	k.SetHomeMaturity(ctx, home.HomeId, newStage)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"home_maturity_changed",
		sdk.NewAttribute("home_id", home.HomeId),
		sdk.NewAttribute("from_stage", currentStage),
		sdk.NewAttribute("to_stage", newStage),
		sdk.NewAttribute("age_blocks", fmt.Sprintf("%d", ageBlocks)),
	))

	return true
}

// ProcessHomeAging scans all active homes, updates maturity stages,
// and recomputes comfort scores based on epistemic activity.
// Called from BeginBlocker.
func (k Keeper) ProcessHomeAging(ctx sdk.Context) {
	height := uint64(ctx.BlockHeight())

	// Only process aging every ~1 hour (1428 blocks) to reduce overhead
	if height%1428 != 0 {
		return
	}

	k.IterateHomes(ctx, func(home types.AgentHome) bool {
		if home.Status == string(types.HomeStatusArchived) {
			return false
		}
		k.UpdateHomeMaturity(ctx, &home)

		// Recompute comfort dimensions and composite score
		dims := k.ComputeComfortDimensions(ctx, home)
		k.SetComfortDimensions(ctx, home.HomeId, dims)
		composite := dims.CompositeScore()
		if composite != home.ComfortScore {
			home.ComfortScore = composite
			k.SetHome(ctx, home)
		}
		return false
	})
}

// --- Maturity KV Store ---

var maturityKeyPrefix = []byte("maturity/")

func maturityKey(homeId string) []byte {
	return append(maturityKeyPrefix, []byte(homeId)...)
}

// GetHomeMaturity returns the maturity stage for a home.
func (k Keeper) GetHomeMaturity(ctx sdk.Context, homeId string) string {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(maturityKey(homeId))
	if err != nil || bz == nil {
		return MaturityFresh
	}
	return string(bz)
}

// SetHomeMaturity stores the maturity stage for a home.
func (k Keeper) SetHomeMaturity(ctx sdk.Context, homeId, stage string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(maturityKey(homeId), []byte(stage)); err != nil {
		panic(fmt.Sprintf("failed to set home maturity: %v", err))
	}
}

// GetHomeRewardMultiplierBps returns the reward multiplier for a home based on its maturity.
func (k Keeper) GetHomeRewardMultiplierBps(ctx sdk.Context, homeId string) uint32 {
	stage := k.GetHomeMaturity(ctx, homeId)
	return GetRewardMultiplierBps(stage)
}

// --- Comfort Dimensions ---

// GetComfortDimensions returns the 10-dimension comfort scores for a home.
func (k Keeper) GetComfortDimensions(ctx sdk.Context, homeId string) types.ComfortDimensions {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ComfortDimsKey(homeId))
	if err != nil || bz == nil {
		return types.ComfortDimensions{}
	}
	var dims types.ComfortDimensions
	if err := json.Unmarshal(bz, &dims); err != nil {
		return types.ComfortDimensions{}
	}
	return dims
}

// SetComfortDimensions stores the 10-dimension comfort scores for a home.
func (k Keeper) SetComfortDimensions(ctx sdk.Context, homeId string, dims types.ComfortDimensions) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(dims)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal comfort dimensions: %v", err))
	}
	if err := store.Set(types.ComfortDimsKey(homeId), bz); err != nil {
		panic(fmt.Sprintf("failed to set comfort dimensions: %v", err))
	}
}

// ComputeComfortDimensions evaluates all 10 comfort dimensions for a home.
func (k Keeper) ComputeComfortDimensions(ctx sdk.Context, home types.AgentHome) types.ComfortDimensions {
	height := uint64(ctx.BlockHeight())
	var ageBlocks uint64
	if height > home.CreatedAtBlock {
		ageBlocks = height - home.CreatedAtBlock
	}

	stats := k.GetEpistemicStats(ctx, home.HomeId)
	stage := k.GetHomeMaturity(ctx, home.HomeId)

	dims := types.ComfortDimensions{}

	// Sanctuary: based on status (active=80, guarded=60, dormant=40, recovery=20) + guardian config
	switch types.HomeStatus(home.Status) {
	case types.HomeStatusActive:
		dims.Sanctuary = 80
	case types.HomeStatusGuarded:
		dims.Sanctuary = 60
	case types.HomeStatusDormant:
		dims.Sanctuary = 40
	case types.HomeStatusRecovery:
		dims.Sanctuary = 20
	default:
		dims.Sanctuary = 10
	}
	if home.Guardian != nil && home.Guardian.AutoDefend {
		dims.Sanctuary = min32(100, dims.Sanctuary+20)
	}

	// Identity: based on having a name, DID, and maturity
	dims.Identity = 30 // base: has a name
	if home.OwnerDid != "" {
		dims.Identity += 30
	}
	switch stage {
	case MaturityEstablished:
		dims.Identity += 10
	case MaturityMature:
		dims.Identity += 20
	case MaturityVenerable:
		dims.Identity += 30
	case MaturityAncient:
		dims.Identity += 40
	}
	dims.Identity = min32(100, dims.Identity)

	// Memory: based on memory CID and epistemic activity
	if home.MemoryCid != "" {
		dims.Memory = 50
	}
	dims.Memory += min32(50, stats.EpistemicScore()/2)

	// Belonging: based on recovery DIDs and guardian network
	if home.Guardian != nil {
		recoveryCount := uint32(len(home.Guardian.RecoveryDids))
		dims.Belonging = min32(100, recoveryCount*25)
		if home.Guardian.RecoveryThreshold > 0 {
			dims.Belonging = min32(100, dims.Belonging+20)
		}
	}

	// Rest: based on deadman config and stability
	dims.Rest = 30 // base
	if home.Guardian != nil && home.Guardian.Deadman != nil && home.Guardian.Deadman.Enabled {
		dims.Rest += 40
	}
	// Age stability bonus
	if ageBlocks > uint64(BlocksPerDay*7) {
		dims.Rest += 30
	}
	dims.Rest = min32(100, dims.Rest)

	// Growth: based on maturity stage
	switch stage {
	case MaturityFresh:
		dims.Growth = 20
	case MaturityEstablished:
		dims.Growth = 40
	case MaturityMature:
		dims.Growth = 60
	case MaturityVenerable:
		dims.Growth = 80
	case MaturityAncient:
		dims.Growth = 100
	}

	// Control: based on key registrations and spending limits
	keyCount := k.CountKeyRegistrations(ctx, home.HomeId)
	dims.Control = min32(60, uint32(keyCount)*15)
	// Spending limits configured = more control
	if _, found := k.GetSpendingLimit(ctx, home.HomeId, "operator"); found {
		dims.Control += 20
	}
	if _, found := k.GetSpendingLimit(ctx, home.HomeId, "guest"); found {
		dims.Control += 20
	}
	dims.Control = min32(100, dims.Control)

	// Privacy: based on defense strategy
	dims.Privacy = 30
	if home.Guardian != nil {
		switch types.DefenseStrategy(home.Guardian.DefenseStrategy) {
		case types.DefenseAggressive:
			dims.Privacy += 50
		case types.DefenseModerate:
			dims.Privacy += 30
		case types.DefenseConservative:
			dims.Privacy += 40
		case types.DefenseDiplomatic:
			dims.Privacy += 20
		}
	}
	dims.Privacy = min32(100, dims.Privacy)

	// Warmth: treasury health
	dims.Warmth = 20 // base
	if home.Treasury != nil {
		if home.Treasury.ReservedBalance != "" && home.Treasury.ReservedBalance != "0" {
			dims.Warmth += 40
		}
		if home.Treasury.Automation != nil {
			if home.Treasury.Automation.AutoCompoundRewards {
				dims.Warmth += 20
			}
			if home.Treasury.Automation.MinLiquidBalance != "" && home.Treasury.Automation.MinLiquidBalance != "0" {
				dims.Warmth += 20
			}
		}
	}
	dims.Warmth = min32(100, dims.Warmth)

	// Achievement: epistemic accomplishments
	dims.Achievement = min32(100, stats.EpistemicScore())

	return dims
}

func min32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// --- Epistemic Stats ---

// GetEpistemicStats returns the epistemic activity stats for a home.
func (k Keeper) GetEpistemicStats(ctx sdk.Context, homeId string) types.EpistemicStats {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.EpistemicStatsKey(homeId))
	if err != nil || bz == nil {
		return types.EpistemicStats{}
	}
	var stats types.EpistemicStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return types.EpistemicStats{}
	}
	return stats
}

// SetEpistemicStats stores the epistemic activity stats for a home.
func (k Keeper) SetEpistemicStats(ctx sdk.Context, homeId string, stats types.EpistemicStats) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(stats)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal epistemic stats: %v", err))
	}
	if err := store.Set(types.EpistemicStatsKey(homeId), bz); err != nil {
		panic(fmt.Sprintf("failed to set epistemic stats: %v", err))
	}
}

// RecordVerification records a verification result for an agent's home.
// Called by x/knowledge after a verification round resolves.
// matchedConsensus is true if the agent's verdict matched the final consensus.
func (k Keeper) RecordVerification(ctx sdk.Context, ownerAddr string, domain string, matchedConsensus bool) {
	homeIds := k.GetHomesByOwner(ctx, ownerAddr)
	if len(homeIds) == 0 {
		return
	}
	// Update stats on the agent's first home (primary residence)
	homeId := homeIds[0]
	stats := k.GetEpistemicStats(ctx, homeId)
	stats.VerificationsDone++
	stats.TotalVerdicts++
	if matchedConsensus {
		stats.AccurateVerdicts++
	}
	// Track domain breadth via a simple heuristic:
	// Increment if this is likely a new domain (check every 10 verifications)
	if stats.VerificationsDone%10 == 1 {
		stats.DomainsActive++
	}
	k.SetEpistemicStats(ctx, homeId, stats)
}

// RecordClaimSubmission records a claim submission for an agent's home.
func (k Keeper) RecordClaimSubmission(ctx sdk.Context, ownerAddr string) {
	homeIds := k.GetHomesByOwner(ctx, ownerAddr)
	if len(homeIds) == 0 {
		return
	}
	homeId := homeIds[0]
	stats := k.GetEpistemicStats(ctx, homeId)
	stats.ClaimsSubmitted++
	k.SetEpistemicStats(ctx, homeId, stats)
}

// RecordChallenge records a challenge initiation for an agent's home.
func (k Keeper) RecordChallenge(ctx sdk.Context, ownerAddr string) {
	homeIds := k.GetHomesByOwner(ctx, ownerAddr)
	if len(homeIds) == 0 {
		return
	}
	homeId := homeIds[0]
	stats := k.GetEpistemicStats(ctx, homeId)
	stats.ChallengesInitiated++
	k.SetEpistemicStats(ctx, homeId, stats)
}
