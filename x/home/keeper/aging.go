package keeper

import (
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
	BlocksPerDay          = 34272
	ThresholdEstablished  = BlocksPerDay * 1   // 1 day
	ThresholdMature       = BlocksPerDay * 10  // 10 days
	ThresholdVenerable    = BlocksPerDay * 30  // 30 days
	ThresholdAncient      = BlocksPerDay * 100 // 100 days
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
// Patina combines age, activity consistency, and stability.
// Formula: patina = min(100, ageFactor * activityFactor * stabilityFactor)
// where:
//   - ageFactor = log2(1 + ageBlocks/BlocksPerDay) * 10 (capped at 60)
//   - activityFactor = min(1.0, activeBlocks / totalBlocks) * 20 (capped at 20)
//   - stabilityFactor = min(1.0, 1 - statusChanges/10) * 20 (capped at 20)
func ComputePatina(ageBlocks, activeRatioBlocks, totalBlocks uint64, statusChanges uint32) uint32 {
	// Age factor: logarithmic growth, max 60 points
	ageDays := float64(ageBlocks) / float64(BlocksPerDay)
	ageFactor := math.Log2(1+ageDays) * 10
	if ageFactor > 60 {
		ageFactor = 60
	}

	// Activity factor: ratio of active blocks to total blocks, max 20 points
	var activityFactor float64
	if totalBlocks > 0 {
		activityFactor = float64(activeRatioBlocks) / float64(totalBlocks) * 20
	}
	if activityFactor > 20 {
		activityFactor = 20
	}

	// Stability factor: fewer status changes = more stability, max 20 points
	stabilityRatio := 1.0 - float64(statusChanges)/10.0
	if stabilityRatio < 0 {
		stabilityRatio = 0
	}
	stabilityFactor := stabilityRatio * 20

	patina := ageFactor + activityFactor + stabilityFactor
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

// ProcessHomeAging scans all active homes and updates maturity stages.
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
