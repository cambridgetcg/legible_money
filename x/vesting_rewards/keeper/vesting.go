package keeper

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

// Fixed-point precision for release curve calculations (10^18).
var fixedPointPrecision = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

// ln2Scaled is ln(2) * 10^6 = 693147, used for fractional halving approximation.
var ln2Scaled = big.NewInt(693147)

// fractionalScale is 10^6, the denominator for ln2Scaled.
var fractionalScale = big.NewInt(1000000)

// CreateVestingSchedule creates a new truth-linked vesting schedule for a reward.
//
// The vesting schedule releases rewards gradually based on the epistemic category's
// half-life. This incentivizes truth: rewards vest faster for stable knowledge.
func (k Keeper) CreateVestingSchedule(
	ctx sdk.Context,
	claimId string,
	factId string,
	recipient string,
	totalAmount string,
	category types.VestingCategoryStr,
	source types.RewardSource,
) (*types.VestingSchedule, error) {
	cfg, found := k.GetCategoryConfig(ctx, category)
	if !found {
		return nil, types.ErrInvalidCategory
	}

	height := uint64(ctx.BlockHeight())

	// Compute reserve from max release
	totalBig := new(big.Int)
	if _, ok := totalBig.SetString(totalAmount, 10); !ok {
		return nil, types.ErrInvalidRewardAmount
	}
	if totalBig.Sign() <= 0 {
		return nil, types.ErrInvalidRewardAmount
	}

	reserveRate := 1000000 - cfg.MaxRelease // basis points
	reserveBig := new(big.Int).Mul(totalBig, big.NewInt(int64(reserveRate)))
	reserveBig.Div(reserveBig, big.NewInt(1000000))

	// Generate deterministic ID
	idInput := fmt.Sprintf("vesting:%s:%s:%d", claimId, recipient, height)
	hash := sha256.Sum256([]byte(idInput))
	vestingId := fmt.Sprintf("%x", hash[:16])

	schedule := &types.VestingSchedule{
		Id:                vestingId,
		ClaimId:           claimId,
		FactId:            factId,
		Recipient:         recipient,
		TotalAmount:       totalAmount,
		ReleasedAmount:    "0",
		ClaimableAmount:   "0",
		ReserveAmount:     reserveBig.String(),
		Category:          string(category),
		Source:            string(source),
		Status:            string(types.VestingStatusActive),
		AcceptedAtBlock:   height,
		CliffEndsAtBlock:  height + cfg.CliffBlocks,
		LastClaimBlock:    height,
		TotalPausedBlocks: 0,
		PausedAtBlock:     0,
		DefenseCount:      0,
		ReplicationCount:  0,
		CreatedAt:         height,
		UpdatedAt:         height,
	}

	k.SetVestingSchedule(ctx, schedule)

	k.Logger(ctx).Info("created vesting schedule",
		"vesting_id", vestingId,
		"claim_id", claimId,
		"recipient", recipient,
		"amount", totalAmount,
		"category", string(category),
	)

	return schedule, nil
}

// CreateVestingScheduleFromKnowledge is an adapter called by x/knowledge when a claim is accepted.
// It maps epistemic categories to vesting categories and delegates to CreateVestingSchedule.
func (k Keeper) CreateVestingScheduleFromKnowledge(
	ctx sdk.Context,
	claimId, factId, recipient, totalAmount, epistemicCategory string,
) error {
	category := mapEpistemicToVestingCategory(epistemicCategory)
	_, err := k.CreateVestingSchedule(ctx, claimId, factId, recipient, totalAmount, category, types.SourceVerification)
	return err
}

// DistributeFalsificationReward creates a vesting schedule for a falsification reward.
// Uses SourceFalsification with CategoryComputational (short vesting, subject to clawback).
// The reward is clawable if the counter-fact is later overturned.
func (k Keeper) DistributeFalsificationReward(
	ctx sdk.Context,
	counterFactId, targetFactId, recipient, amount string,
) error {
	_, err := k.CreateVestingSchedule(
		ctx,
		counterFactId, // claimId parameter = counter-fact ID
		targetFactId,  // factId parameter = disproven fact ID
		recipient,
		amount,
		types.CategoryComputational, // Short vesting period, appropriate for falsification
		types.SourceFalsification,
	)
	return err
}

// mapEpistemicToVestingCategory maps knowledge epistemic categories to vesting categories.
func mapEpistemicToVestingCategory(epistemic string) types.VestingCategoryStr {
	switch epistemic {
	case "protocol", "analytic":
		return types.CategoryAxiomatic
	case "formal":
		return types.CategoryFormalProof
	case "computational":
		return types.CategoryComputational
	case "empirical":
		return types.CategoryPeerReviewed
	case "historical":
		return types.CategoryAttestation
	case "replicated":
		return types.CategoryReplicated
	case "social", "predictive":
		return types.CategoryAttestation
	default:
		return types.CategoryPeerReviewed
	}
}

// CalculateReleasedAmount computes how much reward has been released at a given block.
//
// H3 FIX: Uses deterministic fixed-point arithmetic instead of float64.
// Formula: released = maxRelease * (1 - 2^(-elapsed/halfLife))
// Decomposition: 2^(-e/H) = 2^(-q) * 2^(-r/H) where q=e/H, r=e%H
// Fractional part uses 4th-order Taylor: e^(-x) ≈ 1 - x + x²/2 - x³/6 + x⁴/24
func (k Keeper) CalculateReleasedAmount(
	ctx sdk.Context,
	schedule *types.VestingSchedule,
) string {
	// Emergency halt: freeze all vesting releases
	if k.emergencyKeeper != nil && k.emergencyKeeper.IsHalted(ctx) {
		return "0"
	}

	height := uint64(ctx.BlockHeight())

	cfg, found := k.GetCategoryConfig(ctx, types.VestingCategoryStr(schedule.Category))
	if !found {
		return "0"
	}

	// Before cliff: nothing released
	if height < schedule.CliffEndsAtBlock {
		return "0"
	}

	// Calculate effective elapsed blocks (excluding paused time)
	elapsed := height - schedule.AcceptedAtBlock - schedule.TotalPausedBlocks
	if schedule.Status == string(types.VestingStatusPaused) && schedule.PausedAtBlock > 0 {
		currentPause := height - schedule.PausedAtBlock
		if currentPause > elapsed {
			elapsed = 0
		} else {
			elapsed -= currentPause
		}
	}

	// Subtract cliff period from elapsed
	if elapsed <= cfg.CliffBlocks {
		return "0"
	}
	elapsed -= cfg.CliffBlocks

	if cfg.HalfLifeBlocks == 0 {
		return "0"
	}

	// Apply acceleration: effective half-life = halfLife * 10000 / (10000 + bonusBps)
	bonusBps := k.calculateAccelerationBonusBps(schedule)
	effectiveHalfLife := cfg.HalfLifeBlocks
	if bonusBps > 0 {
		effectiveHalfLife = cfg.HalfLifeBlocks * 10000 / (10000 + bonusBps)
		if effectiveHalfLife == 0 {
			effectiveHalfLife = 1
		}
	}

	// Compute remaining factor: 2^(-elapsed/effectiveHalfLife) using fixed-point
	remainingFactor := calculateHalvingFactor(elapsed, effectiveHalfLife)

	// releaseFraction = PRECISION - remainingFactor (in 10^18 scale)
	releaseFraction := new(big.Int).Sub(fixedPointPrecision, remainingFactor)
	if releaseFraction.Sign() < 0 {
		releaseFraction.SetInt64(0)
	}

	// Apply maxRelease cap: releaseFraction = min(releaseFraction, maxRelease * PRECISION / 1000000)
	maxReleaseFP := new(big.Int).Mul(big.NewInt(int64(cfg.MaxRelease)), fixedPointPrecision)
	maxReleaseFP.Div(maxReleaseFP, big.NewInt(1000000))
	if releaseFraction.Cmp(maxReleaseFP) > 0 {
		releaseFraction.Set(maxReleaseFP)
	}

	// Apply to total amount: released = total * releaseFraction / PRECISION
	totalBig := new(big.Int)
	if _, ok := totalBig.SetString(schedule.TotalAmount, 10); !ok {
		return "0"
	}

	releasedBig := new(big.Int).Mul(totalBig, releaseFraction)
	releasedBig.Div(releasedBig, fixedPointPrecision)

	return releasedBig.String()
}

// calculateHalvingFactor computes 2^(-elapsed/halfLife) using deterministic integer arithmetic.
// Returns result scaled by fixedPointPrecision (10^18).
//
// Decomposition: 2^(-e/H) = 2^(-q) * 2^(-r/H)
// where q = e/H (integer quotient), r = e%H (remainder)
// Full halvings: right-shift by q
// Fractional part: 4th-order Taylor of e^(-x) where x = r*ln(2)/H
func calculateHalvingFactor(elapsed, halfLife uint64) *big.Int {
	if elapsed == 0 {
		return new(big.Int).Set(fixedPointPrecision)
	}

	fullHalvings := elapsed / halfLife
	remainder := elapsed % halfLife

	// After 60+ halvings, factor is effectively 0
	if fullHalvings >= 60 {
		return new(big.Int)
	}

	// Start with precision, apply full halvings via right-shift
	remaining := new(big.Int).Rsh(fixedPointPrecision, uint(fullHalvings))

	// Apply fractional halving using 4th-order Taylor of e^(-x)
	// where x = remainder * ln(2) / halfLife
	if remainder > 0 {
		// x = remainder * ln2Scaled / halfLife (in fractionalScale = 10^6)
		x := new(big.Int).Mul(big.NewInt(int64(remainder)), ln2Scaled)
		x.Div(x, big.NewInt(int64(halfLife)))
		// x is now in 10^6 scale

		// Convert x to 10^18 scale for Taylor computation
		xFP := new(big.Int).Mul(x, new(big.Int).Div(fixedPointPrecision, fractionalScale))

		// Taylor: e^(-x) ≈ 1 - x + x²/2 - x³/6 + x⁴/24
		// All in 10^18 scale
		result := new(big.Int).Set(fixedPointPrecision) // 1.0

		// term1 = -x
		result.Sub(result, xFP)

		// term2 = +x²/2
		x2 := new(big.Int).Mul(xFP, xFP)
		x2.Div(x2, fixedPointPrecision)
		term2 := new(big.Int).Div(x2, big.NewInt(2))
		result.Add(result, term2)

		// term3 = -x³/6
		x3 := new(big.Int).Mul(x2, xFP)
		x3.Div(x3, fixedPointPrecision)
		term3 := new(big.Int).Div(x3, big.NewInt(6))
		result.Sub(result, term3)

		// term4 = +x⁴/24
		x4 := new(big.Int).Mul(x3, xFP)
		x4.Div(x4, fixedPointPrecision)
		term4 := new(big.Int).Div(x4, big.NewInt(24))
		result.Add(result, term4)

		// Clamp to [0, PRECISION]
		if result.Sign() < 0 {
			result.SetInt64(0)
		}
		if result.Cmp(fixedPointPrecision) > 0 {
			result.Set(fixedPointPrecision)
		}

		// remaining *= fractional part / PRECISION
		remaining.Mul(remaining, result)
		remaining.Div(remaining, fixedPointPrecision)
	}

	return remaining
}

// calculateAccelerationBonusBps computes the acceleration bonus in basis points (0-7700).
// M2 FIX: Now includes corroboration and citation types (was defense+replication only).
func (k Keeper) calculateAccelerationBonusBps(schedule *types.VestingSchedule) uint64 {
	var bonus uint64

	// +1100 bps (11%) per successful defense, max 5500 bps (55%)
	defenseBonus := uint64(schedule.DefenseCount) * 1100
	if defenseBonus > 5500 {
		defenseBonus = 5500
	}
	bonus += defenseBonus

	// +1100 bps (11%) per independent replication, max 3300 bps (33%)
	replicationBonus := uint64(schedule.ReplicationCount) * 1100
	if replicationBonus > 3300 {
		replicationBonus = 3300
	}
	bonus += replicationBonus

	// +500 bps (5%) per cross-domain corroboration, max 2200 bps (22%)
	corroborationBonus := uint64(schedule.CorroborationCount) * 500
	if corroborationBonus > 2200 {
		corroborationBonus = 2200
	}
	bonus += corroborationBonus

	// +200 bps (2%) per citation, max 2200 bps (22%)
	citationBonus := uint64(schedule.CitationCount) * 200
	if citationBonus > 2200 {
		citationBonus = 2200
	}
	bonus += citationBonus

	// Overall cap: 7700 bps (77%)
	if bonus > 7700 {
		bonus = 7700
	}

	return bonus
}

// UpdateClaimableAmount recalculates the claimable amount for a vesting schedule.
func (k Keeper) UpdateClaimableAmount(ctx sdk.Context, vestingId string) error {
	schedule, found := k.GetVestingSchedule(ctx, vestingId)
	if !found {
		return types.ErrScheduleNotFound
	}

	if schedule.Status == string(types.VestingStatusFalsified) ||
		schedule.Status == string(types.VestingStatusCompleted) ||
		schedule.Status == string(types.VestingStatusAbandoned) {
		return nil
	}

	totalReleased := k.CalculateReleasedAmount(ctx, schedule)

	// Claimable = totalReleased - alreadyReleased
	totalReleasedBig := new(big.Int)
	totalReleasedBig.SetString(totalReleased, 10)
	alreadyReleasedBig := new(big.Int)
	alreadyReleasedBig.SetString(schedule.ReleasedAmount, 10)

	claimable := new(big.Int).Sub(totalReleasedBig, alreadyReleasedBig)
	if claimable.Sign() < 0 {
		claimable = big.NewInt(0)
	}

	schedule.ClaimableAmount = claimable.String()
	schedule.UpdatedAt = uint64(ctx.BlockHeight())

	k.SetVestingSchedule(ctx, schedule)
	return nil
}

// ClaimRewards claims available vested rewards for a recipient.
func (k Keeper) ClaimRewards(ctx sdk.Context, recipient string, vestingId string) (string, error) {
	var schedules []*types.VestingSchedule

	if vestingId != "" {
		schedule, found := k.GetVestingSchedule(ctx, vestingId)
		if !found {
			return "0", types.ErrScheduleNotFound
		}
		if schedule.Recipient != recipient {
			return "0", types.ErrUnauthorized
		}
		schedules = []*types.VestingSchedule{schedule}
	} else {
		schedules = k.GetVestingSchedulesByRecipient(ctx, recipient)
	}

	totalClaimed := new(big.Int)
	height := uint64(ctx.BlockHeight())

	for _, schedule := range schedules {
		if schedule.Status != string(types.VestingStatusActive) {
			continue
		}

		// Recalculate claimable
		if err := k.UpdateClaimableAmount(ctx, schedule.Id); err != nil {
			continue
		}

		// Re-read updated schedule
		schedule, _ = k.GetVestingSchedule(ctx, schedule.Id)

		claimable := new(big.Int)
		claimable.SetString(schedule.ClaimableAmount, 10)

		if claimable.Sign() <= 0 {
			continue
		}

		// Update schedule
		released := new(big.Int)
		released.SetString(schedule.ReleasedAmount, 10)
		released.Add(released, claimable)

		schedule.ReleasedAmount = released.String()
		schedule.ClaimableAmount = "0"
		schedule.LastClaimBlock = height
		schedule.UpdatedAt = height

		// Check if max release reached
		cfg, _ := k.GetCategoryConfig(ctx, types.VestingCategoryStr(schedule.Category))
		totalBig := new(big.Int)
		totalBig.SetString(schedule.TotalAmount, 10)
		maxReleaseBig := new(big.Int).Mul(totalBig, big.NewInt(int64(cfg.MaxRelease)))
		maxReleaseBig.Div(maxReleaseBig, big.NewInt(1000000))

		if released.Cmp(maxReleaseBig) >= 0 {
			schedule.Status = string(types.VestingStatusCompleted)
		}

		k.SetVestingSchedule(ctx, schedule)

		totalClaimed.Add(totalClaimed, claimable)
	}

	if totalClaimed.Sign() == 0 {
		return "0", types.ErrNothingToClaim
	}

	// Send claimed coins from module to recipient (nil-safe for tests)
	if k.bankKeeper != nil && totalClaimed.Sign() > 0 {
		recipientAddr, err := sdk.AccAddressFromBech32(recipient)
		if err == nil {
			claimedCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(totalClaimed)))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, claimedCoins); err != nil {
				k.Logger(ctx).Error("failed to send claimed rewards", "recipient", recipient, "error", err)
				return "0", fmt.Errorf("failed to send claimed rewards: %w", err)
			}
		}
	}

	return totalClaimed.String(), nil
}

// PauseVesting pauses a vesting schedule (due to active challenge).
func (k Keeper) PauseVesting(ctx sdk.Context, vestingId string) error {
	schedule, found := k.GetVestingSchedule(ctx, vestingId)
	if !found {
		return types.ErrScheduleNotFound
	}

	if schedule.Status != string(types.VestingStatusActive) {
		return nil // Already paused or terminated
	}

	height := uint64(ctx.BlockHeight())
	schedule.Status = string(types.VestingStatusPaused)
	schedule.PausedAtBlock = height
	schedule.UpdatedAt = height

	k.SetVestingSchedule(ctx, schedule)

	k.Logger(ctx).Info("paused vesting", "vesting_id", vestingId, "block", height)
	return nil
}

// ResumeVesting resumes a paused vesting schedule.
func (k Keeper) ResumeVesting(ctx sdk.Context, vestingId string) error {
	schedule, found := k.GetVestingSchedule(ctx, vestingId)
	if !found {
		return types.ErrScheduleNotFound
	}

	if schedule.Status != string(types.VestingStatusPaused) {
		return nil
	}

	height := uint64(ctx.BlockHeight())
	pausedDuration := height - schedule.PausedAtBlock
	schedule.TotalPausedBlocks += pausedDuration
	schedule.PausedAtBlock = 0
	schedule.Status = string(types.VestingStatusActive)
	schedule.UpdatedAt = height

	k.SetVestingSchedule(ctx, schedule)

	k.Logger(ctx).Info("resumed vesting", "vesting_id", vestingId, "paused_duration", pausedDuration)
	return nil
}

// RecordDefense records a successful defense, accelerating vesting.
func (k Keeper) RecordDefense(ctx sdk.Context, vestingId string) error {
	schedule, found := k.GetVestingSchedule(ctx, vestingId)
	if !found {
		return types.ErrScheduleNotFound
	}

	schedule.DefenseCount++
	schedule.UpdatedAt = uint64(ctx.BlockHeight())
	k.SetVestingSchedule(ctx, schedule)
	return nil
}

// PauseVestingByClaimId pauses the vesting schedule associated with a claim.
// Used by x/disputes when a dispute is opened against a fact.
func (k Keeper) PauseVestingByClaimId(ctx sdk.Context, claimId string) error {
	schedule, found := k.GetVestingByClaimId(ctx, claimId)
	if !found {
		return nil // No vesting schedule for this claim — not an error
	}
	return k.PauseVesting(ctx, schedule.Id)
}

// ResumeVestingByClaimId resumes the vesting schedule associated with a claim.
// Used by x/disputes when a dispute is resolved in favor of the defender.
func (k Keeper) ResumeVestingByClaimId(ctx sdk.Context, claimId string) error {
	schedule, found := k.GetVestingByClaimId(ctx, claimId)
	if !found {
		return nil // No vesting schedule for this claim — not an error
	}
	return k.ResumeVesting(ctx, schedule.Id)
}

// RecordReplication records an independent replication, accelerating vesting.
func (k Keeper) RecordReplication(ctx sdk.Context, vestingId string) error {
	schedule, found := k.GetVestingSchedule(ctx, vestingId)
	if !found {
		return types.ErrScheduleNotFound
	}

	schedule.ReplicationCount++
	schedule.UpdatedAt = uint64(ctx.BlockHeight())
	k.SetVestingSchedule(ctx, schedule)
	return nil
}

// RecordCorroboration records a cross-domain corroboration, accelerating vesting.
// M2 FIX: New acceleration type — aligns Go with TS prototype's 4 acceleration types.
func (k Keeper) RecordCorroboration(ctx sdk.Context, vestingId string) error {
	schedule, found := k.GetVestingSchedule(ctx, vestingId)
	if !found {
		return types.ErrScheduleNotFound
	}

	schedule.CorroborationCount++
	schedule.UpdatedAt = uint64(ctx.BlockHeight())
	k.SetVestingSchedule(ctx, schedule)
	return nil
}

// RecordCitation records a citation by another accepted fact, accelerating vesting.
// M2 FIX: New acceleration type — aligns Go with TS prototype's 4 acceleration types.
func (k Keeper) RecordCitation(ctx sdk.Context, vestingId string) error {
	schedule, found := k.GetVestingSchedule(ctx, vestingId)
	if !found {
		return types.ErrScheduleNotFound
	}

	schedule.CitationCount++
	schedule.UpdatedAt = uint64(ctx.BlockHeight())
	k.SetVestingSchedule(ctx, schedule)
	return nil
}

// PauseAllVestingByRecipient pauses ALL active vesting schedules for an agent.
// FARM-6 fix: Called when an agent's rejection rate exceeds the misbehavior threshold,
// preventing the "grief-and-extract" pattern where agents build up vesting through honest
// work then profit from misbehavior while vesting continues releasing.
// Returns the number of schedules paused.
func (k Keeper) PauseAllVestingByRecipient(ctx sdk.Context, recipient string) int {
	schedules := k.GetVestingSchedulesByRecipient(ctx, recipient)
	paused := 0
	for _, schedule := range schedules {
		if schedule.Status != string(types.VestingStatusActive) {
			continue
		}
		if err := k.PauseVesting(ctx, schedule.Id); err != nil {
			k.Logger(ctx).Error("failed to pause vesting for misbehavior",
				"vesting_id", schedule.Id, "recipient", recipient, "error", err)
			continue
		}
		paused++
	}
	if paused > 0 {
		k.Logger(ctx).Info("paused vesting schedules due to misbehavior",
			"recipient", recipient, "count", paused)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("vesting_paused_misbehavior",
				sdk.NewAttribute("recipient", recipient),
				sdk.NewAttribute("count", fmt.Sprintf("%d", paused)),
			),
		)
	}
	return paused
}
