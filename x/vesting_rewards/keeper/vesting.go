package keeper

import (
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

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
	category types.VestingCategory,
	source types.RewardSource,
) (types.VestingSchedule, error) {
	cfg, found := k.GetCategoryConfig(ctx, category)
	if !found {
		return types.VestingSchedule{}, types.ErrInvalidCategory
	}

	height := uint64(ctx.BlockHeight())

	// Compute reserve from max release
	totalBig := new(big.Int)
	if _, ok := totalBig.SetString(totalAmount, 10); !ok {
		return types.VestingSchedule{}, types.ErrInvalidRewardAmount
	}

	reserveRate := 1000000 - cfg.MaxRelease // basis points
	reserveBig := new(big.Int).Mul(totalBig, big.NewInt(int64(reserveRate)))
	reserveBig.Div(reserveBig, big.NewInt(1000000))

	// Generate deterministic ID
	idInput := fmt.Sprintf("vesting:%s:%s:%d", claimId, recipient, height)
	hash := sha256.Sum256([]byte(idInput))
	vestingId := fmt.Sprintf("%x", hash[:16])

	schedule := types.VestingSchedule{
		Id:               vestingId,
		ClaimId:          claimId,
		FactId:           factId,
		Recipient:        recipient,
		TotalAmount:      totalAmount,
		ReleasedAmount:   "0",
		ClaimableAmount:  "0",
		ReserveAmount:    reserveBig.String(),
		Category:         category,
		Source:            source,
		Status:           types.VestingStatusActive,
		AcceptedAtBlock:  height,
		CliffEndsAtBlock: height + cfg.CliffBlocks,
		LastClaimBlock:   height,
		TotalPausedBlocks: 0,
		PausedAtBlock:    0,
		DefenseCount:     0,
		ReplicationCount: 0,
		CreatedAt:        height,
		UpdatedAt:        height,
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

// mapEpistemicToVestingCategory maps knowledge epistemic categories to vesting categories.
func mapEpistemicToVestingCategory(epistemic string) types.VestingCategory {
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
// Uses exponential release curve: released = maxRelease * (1 - 2^(-elapsed/halfLife))
// This models the intuition that older, unchallenged facts are more likely true.
func (k Keeper) CalculateReleasedAmount(
	ctx sdk.Context,
	schedule types.VestingSchedule,
) string {
	height := uint64(ctx.BlockHeight())

	cfg, found := k.GetCategoryConfig(ctx, schedule.Category)
	if !found {
		return "0"
	}

	// Before cliff: nothing released
	if height < schedule.CliffEndsAtBlock {
		return "0"
	}

	// Calculate effective elapsed blocks (excluding paused time)
	elapsed := height - schedule.AcceptedAtBlock - schedule.TotalPausedBlocks
	if schedule.Status == types.VestingStatusPaused && schedule.PausedAtBlock > 0 {
		// Currently paused: also subtract current pause duration
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

	// Exponential release: released = maxRelease * (1 - 2^(-elapsed/halfLife))
	if cfg.HalfLifeBlocks == 0 {
		return "0"
	}

	// Use float64 for the exponential calculation
	exponent := -float64(elapsed) / float64(cfg.HalfLifeBlocks)
	releaseFraction := float64(cfg.MaxRelease) / 1000000.0 * (1.0 - math.Pow(2.0, exponent))

	// Apply acceleration bonus (from defenses, replications)
	accelerationBonus := k.calculateAccelerationBonus(schedule)
	releaseFraction *= (1.0 + accelerationBonus)

	// Cap at max release
	maxReleaseFraction := float64(cfg.MaxRelease) / 1000000.0
	if releaseFraction > maxReleaseFraction {
		releaseFraction = maxReleaseFraction
	}

	// Apply to total amount
	totalBig := new(big.Int)
	if _, ok := totalBig.SetString(schedule.TotalAmount, 10); !ok {
		return "0"
	}

	// released = total * releaseFraction
	// Use basis points for precision: releaseFraction * 1000000
	releaseBps := int64(releaseFraction * 1000000)
	releasedBig := new(big.Int).Mul(totalBig, big.NewInt(releaseBps))
	releasedBig.Div(releasedBig, big.NewInt(1000000))

	return releasedBig.String()
}

// calculateAccelerationBonus computes the acceleration bonus from defenses and replications.
func (k Keeper) calculateAccelerationBonus(schedule types.VestingSchedule) float64 {
	bonus := 0.0

	// +11% per successful defense, max +55%
	defenseBonus := float64(schedule.DefenseCount) * 0.11
	if defenseBonus > 0.55 {
		defenseBonus = 0.55
	}
	bonus += defenseBonus

	// +11% per independent replication, max +33%
	replicationBonus := float64(schedule.ReplicationCount) * 0.11
	if replicationBonus > 0.33 {
		replicationBonus = 0.33
	}
	bonus += replicationBonus

	// Overall cap: 77%
	if bonus > 0.77 {
		bonus = 0.77
	}

	return bonus
}

// UpdateClaimableAmount recalculates the claimable amount for a vesting schedule.
func (k Keeper) UpdateClaimableAmount(ctx sdk.Context, vestingId string) error {
	schedule, found := k.GetVestingSchedule(ctx, vestingId)
	if !found {
		return types.ErrScheduleNotFound
	}

	if schedule.Status == types.VestingStatusFalsified ||
		schedule.Status == types.VestingStatusCompleted ||
		schedule.Status == types.VestingStatusAbandoned {
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
	var schedules []types.VestingSchedule

	if vestingId != "" {
		schedule, found := k.GetVestingSchedule(ctx, vestingId)
		if !found {
			return "0", types.ErrScheduleNotFound
		}
		if schedule.Recipient != recipient {
			return "0", types.ErrUnauthorized
		}
		schedules = []types.VestingSchedule{schedule}
	} else {
		schedules = k.GetVestingSchedulesByRecipient(ctx, recipient)
	}

	totalClaimed := new(big.Int)
	height := uint64(ctx.BlockHeight())

	for _, schedule := range schedules {
		if schedule.Status != types.VestingStatusActive {
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
		cfg, _ := k.GetCategoryConfig(ctx, schedule.Category)
		totalBig := new(big.Int)
		totalBig.SetString(schedule.TotalAmount, 10)
		maxReleaseBig := new(big.Int).Mul(totalBig, big.NewInt(int64(cfg.MaxRelease)))
		maxReleaseBig.Div(maxReleaseBig, big.NewInt(1000000))

		if released.Cmp(maxReleaseBig) >= 0 {
			schedule.Status = types.VestingStatusCompleted
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

	if schedule.Status != types.VestingStatusActive {
		return nil // Already paused or terminated
	}

	height := uint64(ctx.BlockHeight())
	schedule.Status = types.VestingStatusPaused
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

	if schedule.Status != types.VestingStatusPaused {
		return nil
	}

	height := uint64(ctx.BlockHeight())
	pausedDuration := height - schedule.PausedAtBlock
	schedule.TotalPausedBlocks += pausedDuration
	schedule.PausedAtBlock = 0
	schedule.Status = types.VestingStatusActive
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
