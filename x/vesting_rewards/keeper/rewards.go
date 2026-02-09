package keeper

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

const (
	// ResearchFundModuleName is the module account name for the research fund.
	ResearchFundModuleName = "research_fund"
)

// RouteReward routes a reward through the research share deduction.
// 22% goes to the research fund, 78% goes to the recipient.
//
// All rewards in the protocol flow through this router to ensure
// consistent funding for research verification.
func (k Keeper) RouteReward(
	ctx sdk.Context,
	source types.RewardSource,
	amount string,
	recipient string,
	factId string,
) (types.RewardRouting, error) {
	params := k.GetParams(ctx)

	amountBig := new(big.Int)
	if _, ok := amountBig.SetString(amount, 10); !ok {
		return types.RewardRouting{}, types.ErrInvalidRewardAmount
	}

	if amountBig.Sign() <= 0 {
		return types.RewardRouting{}, types.ErrInvalidRewardAmount
	}

	// Calculate research share: amount * rateBps / 10000
	researchShare := new(big.Int).Mul(amountBig, big.NewInt(int64(params.ResearchShareBps)))
	researchShare.Div(researchShare, big.NewInt(10000))

	// Remaining goes to recipient
	recipientAmount := new(big.Int).Sub(amountBig, researchShare)

	routing := types.RewardRouting{
		Source:          source,
		OriginalAmount:  amount,
		ResearchShare:   researchShare.String(),
		RecipientAmount: recipientAmount.String(),
		Recipient:       recipient,
		FactId:          factId,
		BlockNumber:     uint64(ctx.BlockHeight()),
	}

	return routing, nil
}

// DistributeBlockReward distributes the block production reward.
//
// The reward is scaled by validator participation:
//   reward = baseReward * min(1, activeValidators / targetValidators)
//
// This prevents outsized rewards when validator count is low.
func (k Keeper) DistributeBlockReward(
	ctx sdk.Context,
	producer string,
	activeValidatorCount uint32,
	hasTransactions bool,
) (types.BlockRewardDistribution, error) {
	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())

	// Empty block check: 0% reward for empty blocks (PoT alignment)
	if !hasTransactions && params.EmptyBlockRewardRate == 0 {
		return types.BlockRewardDistribution{
			BlockHeight:    height,
			ProducerReward: "0",
			ResearchShare:  "0",
			TotalMinted:    "0",
			ValidatorCount: activeValidatorCount,
		}, nil
	}

	baseReward := new(big.Int)
	if _, ok := baseReward.SetString(params.BlockRewardAmount, 10); !ok {
		return types.BlockRewardDistribution{}, types.ErrInvalidRewardAmount
	}

	// Scale by validator participation: min(1, active/target)
	if params.MinValidatorsForFullReward > 0 && activeValidatorCount < params.MinValidatorsForFullReward {
		baseReward.Mul(baseReward, big.NewInt(int64(activeValidatorCount)))
		baseReward.Div(baseReward, big.NewInt(int64(params.MinValidatorsForFullReward)))
	}

	// Apply empty block rate if applicable
	if !hasTransactions && params.EmptyBlockRewardRate > 0 {
		baseReward.Mul(baseReward, big.NewInt(int64(params.EmptyBlockRewardRate)))
		baseReward.Div(baseReward, big.NewInt(10000))
	}

	// Route through research share
	routing, err := k.RouteReward(ctx, types.SourceBlockProduction, baseReward.String(), producer, "")
	if err != nil {
		// If amount is 0, just record empty distribution
		dist := types.BlockRewardDistribution{
			BlockHeight:    height,
			ProducerReward: "0",
			ResearchShare:  "0",
			TotalMinted:    "0",
			ValidatorCount: activeValidatorCount,
		}
		k.SetBlockRewardDistribution(ctx, dist)
		return dist, nil
	}

	dist := types.BlockRewardDistribution{
		BlockHeight:    height,
		ProducerReward: routing.RecipientAmount,
		ResearchShare:  routing.ResearchShare,
		TotalMinted:    routing.OriginalAmount,
		ValidatorCount: activeValidatorCount,
	}

	// Mint and distribute coins via bank keeper (nil-safe for tests/devnet)
	if k.bankKeeper != nil {
		totalMintedBig := new(big.Int)
		totalMintedBig.SetString(routing.OriginalAmount, 10)

		if totalMintedBig.Sign() > 0 {
			mintCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(totalMintedBig)))

			// Mint to vesting_rewards module account
			if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
				k.Logger(ctx).Error("failed to mint block reward", "error", err)
				k.SetBlockRewardDistribution(ctx, dist)
				return dist, err
			}

			// Send producer share (78%) to block producer
			producerBig := new(big.Int)
			producerBig.SetString(routing.RecipientAmount, 10)
			if producerBig.Sign() > 0 {
				producerAddr, err := sdk.AccAddressFromBech32(producer)
				if err == nil {
					producerCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(producerBig)))
					if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, producerAddr, producerCoins); err != nil {
						k.Logger(ctx).Error("failed to send producer reward", "error", err)
					}
				}
			}

			// Send research share (22%) to research fund
			researchBig := new(big.Int)
			researchBig.SetString(routing.ResearchShare, 10)
			if researchBig.Sign() > 0 {
				researchCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(researchBig)))
				if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, ResearchFundModuleName, researchCoins); err != nil {
					k.Logger(ctx).Error("failed to send research share", "error", err)
				}
			}
		}
	}

	k.SetBlockRewardDistribution(ctx, dist)

	k.Logger(ctx).Debug("distributed block reward",
		"block", height,
		"producer", producer,
		"reward", routing.RecipientAmount,
		"research", routing.ResearchShare,
	)

	return dist, nil
}

// FalsifyClaim handles clawback when a claim is proven false.
//
// Clawback logic:
//   - 33% of already-released rewards are clawed back
//   - All unvested amount is forfeited
//   - Reserve goes to challenger as bonus
func (k Keeper) FalsifyClaim(
	ctx sdk.Context,
	claimId string,
	challenger string,
) (types.ClawbackRecord, error) {
	schedule, found := k.GetVestingByClaimId(ctx, claimId)
	if !found {
		return types.ClawbackRecord{}, types.ErrScheduleNotFound
	}

	if schedule.Status == types.VestingStatusFalsified {
		return types.ClawbackRecord{}, types.ErrAlreadyFalsified
	}

	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())

	// Calculate clawback amounts
	releasedBig := new(big.Int)
	releasedBig.SetString(schedule.ReleasedAmount, 10)

	// Released clawback: 33% of already released
	releasedClawback := new(big.Int).Mul(releasedBig, big.NewInt(int64(params.ReleasedClawbackRate)))
	releasedClawback.Div(releasedClawback, big.NewInt(10000))

	// Unvested = total - released - reserve
	totalBig := new(big.Int)
	totalBig.SetString(schedule.TotalAmount, 10)
	reserveBig := new(big.Int)
	reserveBig.SetString(schedule.ReserveAmount, 10)

	unvested := new(big.Int).Sub(totalBig, releasedBig)
	unvested.Sub(unvested, reserveBig)
	if unvested.Sign() < 0 {
		unvested = big.NewInt(0)
	}

	// Challenger reward = released clawback + unvested + reserve
	challengerReward := new(big.Int).Add(releasedClawback, unvested)
	challengerReward.Add(challengerReward, reserveBig)

	// Generate clawback record ID
	idInput := fmt.Sprintf("clawback:%s:%d", claimId, height)
	hash := sha256.Sum256([]byte(idInput))
	recordId := fmt.Sprintf("%x", hash[:16])

	record := types.ClawbackRecord{
		Id:               recordId,
		VestingId:        schedule.Id,
		ReleasedClawback: releasedClawback.String(),
		UnvestedForfeited: unvested.String(),
		ReserveForfeited: reserveBig.String(),
		ChallengerReward: challengerReward.String(),
		BlockHeight:      height,
	}

	// Update vesting schedule status
	schedule.Status = types.VestingStatusFalsified
	schedule.UpdatedAt = height
	k.SetVestingSchedule(ctx, schedule)

	// Store clawback record
	k.SetClawbackRecord(ctx, record)

	k.Logger(ctx).Info("falsified claim, clawback executed",
		"claim_id", claimId,
		"vesting_id", schedule.Id,
		"challenger", challenger,
		"challenger_reward", challengerReward.String(),
	)

	return record, nil
}
