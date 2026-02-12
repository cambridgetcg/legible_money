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
) (*types.RewardRouting, error) {
	params := k.GetParams(ctx)

	amountBig := new(big.Int)
	if _, ok := amountBig.SetString(amount, 10); !ok {
		return nil, types.ErrInvalidRewardAmount
	}

	if amountBig.Sign() <= 0 {
		return nil, types.ErrInvalidRewardAmount
	}

	// Calculate research share: amount * rateBps / 1000000
	researchShare := new(big.Int).Mul(amountBig, big.NewInt(int64(params.ResearchShareBps)))
	researchShare.Div(researchShare, big.NewInt(1000000))

	// Remaining goes to recipient
	recipientAmount := new(big.Int).Sub(amountBig, researchShare)

	routing := &types.RewardRouting{
		Source:          string(source),
		OriginalAmount:  amount,
		ResearchShare:   researchShare.String(),
		RecipientAmount: recipientAmount.String(),
		Recipient:       recipient,
		FactId:          factId,
		BlockNumber:     uint64(ctx.BlockHeight()),
	}

	return routing, nil
}

// applyDecay computes: amount * (decayBps/1000000)^epochs using integer exponentiation by squaring.
// decayBps is on a 1,000,000 scale (900000 = 0.9).
func applyDecay(amount *big.Int, decayBps uint64, epochs uint64) *big.Int {
	if epochs == 0 {
		return new(big.Int).Set(amount)
	}

	denom := big.NewInt(1000000)
	base := big.NewInt(int64(decayBps))
	exp := epochs

	// Exponentiation by squaring for decay^epochs in fixed-point (denom scale)
	result := new(big.Int).Set(denom) // start at 1.0
	for exp > 0 {
		if exp%2 == 1 {
			result.Mul(result, base)
			result.Div(result, denom)
		}
		base.Mul(base, base)
		base.Div(base, denom)
		exp /= 2
	}

	// amount * result / denom
	out := new(big.Int).Mul(amount, result)
	out.Div(out, denom)
	return out
}

// calculateFiniteFundReward computes the block reward under finite fund mode.
// 3-factor formula: R = max(initialReward * timeDecay * fundHealth, floorReward)
// Hard stop: R = 0 when fund is exhausted.
func (k Keeper) calculateFiniteFundReward(ctx sdk.Context, params *types.Params) *big.Int {
	height := uint64(ctx.BlockHeight())

	// Parse initial per-block reward (BlockRewardAmount)
	initialReward := new(big.Int)
	if _, ok := initialReward.SetString(params.BlockRewardAmount, 10); !ok || initialReward.Sign() <= 0 {
		return new(big.Int)
	}

	// Calculate epoch number
	epoch := height / params.BlocksPerEpoch

	// Time decay: initialReward * decayFactor^epoch
	reward := applyDecay(initialReward, params.DecayFactorBps, epoch)

	// Fund health factor
	currentBalance := k.GetFundBalance(ctx)
	if currentBalance.Sign() <= 0 {
		return new(big.Int) // Hard stop: fund exhausted
	}

	initialFund := new(big.Int)
	if _, ok := initialFund.SetString(params.InitialFundBalance, 10); !ok || initialFund.Sign() <= 0 {
		return new(big.Int)
	}

	// Health floor = initialFund * thresholdBps / 1000000
	healthFloor := new(big.Int).Mul(initialFund, big.NewInt(int64(params.FundHealthThresholdBps)))
	healthFloor.Div(healthFloor, big.NewInt(1000000))

	if healthFloor.Sign() > 0 && currentBalance.Cmp(healthFloor) < 0 {
		// Linear taper: reward *= currentBalance / healthFloor
		reward.Mul(reward, currentBalance)
		reward.Div(reward, healthFloor)
	}

	// Apply floor reward (only while fund has balance)
	floorReward := new(big.Int)
	if params.FloorReward != "" {
		floorReward.SetString(params.FloorReward, 10)
	}
	if floorReward.Sign() > 0 && reward.Cmp(floorReward) < 0 {
		reward.Set(floorReward)
	}

	// Cap reward to available fund balance
	if reward.Cmp(currentBalance) > 0 {
		reward.Set(currentBalance)
	}

	return reward
}

// GetEpochBlockRewardPool estimates the total block rewards for a given epoch (in ulgm).
// Used by the knowledge module to cap falsification rewards at FalsificationPoolShareBps.
// Approximation: per-block reward at epoch start × blocks_per_epoch.
func (k Keeper) GetEpochBlockRewardPool(ctx sdk.Context, epoch uint64) uint64 {
	params := k.GetParams(ctx)

	initialReward := new(big.Int)
	if _, ok := initialReward.SetString(params.BlockRewardAmount, 10); !ok || initialReward.Sign() <= 0 {
		return 0
	}

	// Per-block reward after decay for this epoch
	perBlock := applyDecay(initialReward, params.DecayFactorBps, epoch)
	if perBlock.Sign() <= 0 {
		return 0
	}

	// Total epoch pool = perBlock × blocksPerEpoch
	pool := new(big.Int).Mul(perBlock, big.NewInt(int64(params.BlocksPerEpoch)))
	if !pool.IsUint64() {
		return ^uint64(0) // saturate
	}
	return pool.Uint64()
}

// DistributeBlockReward distributes the block production reward from the finite fund.
//
// The fund is pre-minted at genesis and depletes over time with exponential decay.
// The reward is scaled by validator participation:
//
//	reward = baseReward * min(1, activeValidators / targetValidators)
func (k Keeper) DistributeBlockReward(
	ctx sdk.Context,
	producer string,
	activeValidatorCount uint32,
	hasTransactions bool,
) (*types.BlockRewardDistribution, error) {
	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())

	// Empty block check: 0% reward for empty blocks (PoT alignment)
	if !hasTransactions && params.EmptyBlockRewardRate == 0 {
		return &types.BlockRewardDistribution{
			BlockHeight:    height,
			ProducerReward: "0",
			ResearchShare:  "0",
			TotalMinted:    "0",
			ValidatorCount: activeValidatorCount,
		}, nil
	}

	// Calculate reward from finite fund (decay + health taper + floor + cap)
	effectiveReward := k.calculateFiniteFundReward(ctx, params)

	if effectiveReward == nil || effectiveReward.Sign() <= 0 {
		dist := &types.BlockRewardDistribution{
			BlockHeight:    height,
			ProducerReward: "0",
			ResearchShare:  "0",
			TotalMinted:    "0",
			ValidatorCount: activeValidatorCount,
		}
		k.SetBlockRewardDistribution(ctx, dist)
		return dist, nil
	}

	// Scale by validator participation: min(1, active/target)
	if params.MinValidatorsForFullReward > 0 && activeValidatorCount < params.MinValidatorsForFullReward {
		effectiveReward.Mul(effectiveReward, big.NewInt(int64(activeValidatorCount)))
		effectiveReward.Div(effectiveReward, big.NewInt(int64(params.MinValidatorsForFullReward)))
	}

	// Apply empty block rate if applicable
	if !hasTransactions && params.EmptyBlockRewardRate > 0 {
		effectiveReward.Mul(effectiveReward, big.NewInt(int64(params.EmptyBlockRewardRate)))
		effectiveReward.Div(effectiveReward, big.NewInt(10000))
	}

	if effectiveReward.Sign() <= 0 {
		dist := &types.BlockRewardDistribution{
			BlockHeight:    height,
			ProducerReward: "0",
			ResearchShare:  "0",
			TotalMinted:    "0",
			ValidatorCount: activeValidatorCount,
		}
		k.SetBlockRewardDistribution(ctx, dist)
		return dist, nil
	}

	// Route through research share
	routing, err := k.RouteReward(ctx, types.SourceBlockProduction, effectiveReward.String(), producer, "")
	if err != nil {
		dist := &types.BlockRewardDistribution{
			BlockHeight:    height,
			ProducerReward: "0",
			ResearchShare:  "0",
			TotalMinted:    "0",
			ValidatorCount: activeValidatorCount,
		}
		k.SetBlockRewardDistribution(ctx, dist)
		return dist, nil
	}

	dist := &types.BlockRewardDistribution{
		BlockHeight:    height,
		ProducerReward: routing.RecipientAmount,
		ResearchShare:  routing.ResearchShare,
		TotalMinted:    routing.OriginalAmount,
		ValidatorCount: activeValidatorCount,
	}

	// Distribute coins via bank keeper (nil-safe for tests/devnet)
	if k.bankKeeper != nil {
		totalRewardBig := new(big.Int)
		totalRewardBig.SetString(routing.OriginalAmount, 10)

		if totalRewardBig.Sign() > 0 {
			// Deduct from fund (no minting — fund was pre-minted at genesis)
			if !k.DeductFromFund(ctx, totalRewardBig) {
				// Fund exhausted — zero reward
				dist.ProducerReward = "0"
				dist.ResearchShare = "0"
				dist.TotalMinted = "0"
				k.SetBlockRewardDistribution(ctx, dist)
				return dist, nil
			}

			// Send producer share (78%) to block producer
			producerBig := new(big.Int)
			producerBig.SetString(routing.RecipientAmount, 10)
			if producerBig.Sign() > 0 {
				producerAddr, addrErr := sdk.AccAddressFromBech32(producer)
				if addrErr == nil {
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

	// Record fund balance after distribution
	dist.FundBalanceAfter = k.GetFundBalance(ctx).String()

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
) (*types.ClawbackRecord, error) {
	schedule, found := k.GetVestingByClaimId(ctx, claimId)
	if !found {
		return nil, types.ErrScheduleNotFound
	}

	if schedule.Status == string(types.VestingStatusFalsified) {
		return nil, types.ErrAlreadyFalsified
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

	record := &types.ClawbackRecord{
		Id:               recordId,
		VestingId:        schedule.Id,
		ReleasedClawback: releasedClawback.String(),
		UnvestedForfeited: unvested.String(),
		ReserveForfeited: reserveBig.String(),
		ChallengerReward: challengerReward.String(),
		BlockHeight:      height,
	}

	// Update vesting schedule status
	schedule.Status = string(types.VestingStatusFalsified)
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
