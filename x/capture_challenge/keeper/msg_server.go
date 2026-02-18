package keeper

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_challenge/types"
)

var _ types.MsgServer = Keeper{}

// SubmitChallenge submits a new capture challenge.
func (k Keeper) SubmitChallenge(goCtx context.Context, msg *types.MsgSubmitChallenge) (*types.MsgSubmitChallengeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentBlock := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)

	// Validate stake >= min
	stakeInt := new(big.Int)
	stakeInt.SetString(msg.Stake, 10)
	minStake := new(big.Int)
	minStake.SetString(types.MinChallengeStake, 10)
	if stakeInt.Cmp(minStake) < 0 {
		return nil, types.ErrInsufficientChallengeStake
	}

	// Lock challenger's stake in the module account
	challengerAddr, err := sdk.AccAddressFromBech32(msg.Challenger)
	if err != nil {
		return nil, fmt.Errorf("invalid challenger address: %w", err)
	}
	stakeCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeInt)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, challengerAddr, types.ModuleName, stakeCoins); err != nil {
		return nil, fmt.Errorf("failed to lock challenge stake: %w", err)
	}

	// Validate evidence count
	if len(msg.Evidence) < types.MinEvidencePieces {
		return nil, types.ErrInsufficientEvidence
	}

	// Generate challenge ID
	idInput := fmt.Sprintf("%s:%s:%d", msg.Challenger, msg.Domain, currentBlock)
	idHash := sha256.Sum256([]byte(idInput))
	challengeId := fmt.Sprintf("challenge_%x", idHash[:8])

	challenge := types.CaptureChallenge{
		Id:                challengeId,
		Domain:            msg.Domain,
		Challenger:        msg.Challenger,
		ChallengeStake:    msg.Stake,
		Evidence:          msg.Evidence,
		AccusedValidators: msg.AccusedValidators,
		SubmittedAtBlock:  currentBlock,
		EvidenceDeadline:  currentBlock + params.EvidencePeriodBlocks,
		ReviewDeadline:    currentBlock + params.EvidencePeriodBlocks + params.ReviewPeriodBlocks,
		Status:            string(types.ChallengeEvidencePeriod),
	}

	k.SetChallenge(ctx, challenge)

	// Update bounty pool
	pool := k.GetOrCreateBountyPool(ctx, msg.Domain, currentBlock)
	pool.ChallengesSubmitted++
	pool.LastUpdatedBlock = currentBlock
	k.SetBountyPool(ctx, pool)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"capture_challenge_submitted",
			sdk.NewAttribute("challenge_id", challengeId),
			sdk.NewAttribute("domain", msg.Domain),
			sdk.NewAttribute("challenger", msg.Challenger),
			sdk.NewAttribute("stake", msg.Stake),
		),
	)

	return &types.MsgSubmitChallengeResponse{ChallengeId: challengeId}, nil
}

// AddEvidence adds evidence to an active challenge.
func (k Keeper) AddEvidence(goCtx context.Context, msg *types.MsgAddEvidence) (*types.MsgAddEvidenceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentBlock := uint64(ctx.BlockHeight())

	challenge, found := k.GetChallenge(ctx, msg.ChallengeId)
	if !found {
		return nil, types.ErrChallengeNotFound
	}

	if challenge.Challenger != msg.Challenger {
		return nil, fmt.Errorf("only the challenger can add evidence")
	}

	if challenge.Status != string(types.ChallengeEvidencePeriod) {
		return nil, types.ErrInvalidChallengeStatus
	}

	if currentBlock > challenge.EvidenceDeadline {
		return nil, types.ErrDeadlinePassed
	}

	if len(challenge.Evidence) >= types.MaxEvidencePieces {
		return nil, types.ErrMaxEvidenceExceeded
	}

	challenge.Evidence = append(challenge.Evidence, msg.Evidence)
	k.SetChallenge(ctx, challenge)

	return &types.MsgAddEvidenceResponse{EvidenceCount: int32(len(challenge.Evidence))}, nil
}

// ResolveChallenge resolves a capture challenge with an outcome.
func (k Keeper) ResolveChallenge(goCtx context.Context, msg *types.MsgResolveChallenge) (*types.MsgResolveChallengeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentBlock := uint64(ctx.BlockHeight())

	// Only authority can resolve
	if msg.Resolver != k.authority {
		return nil, types.ErrUnauthorizedResolver
	}

	challenge, found := k.GetChallenge(ctx, msg.ChallengeId)
	if !found {
		return nil, types.ErrChallengeNotFound
	}

	if challenge.Status != string(types.ChallengeUnderReview) && challenge.Status != string(types.ChallengeEvidencePeriod) {
		return nil, types.ErrInvalidChallengeStatus
	}

	pool := k.GetOrCreateBountyPool(ctx, challenge.Domain, currentBlock)
	challengerReward := "0"

	resolution := types.CaptureResolution{
		Outcome:         msg.Outcome,
		ResolvedAtBlock: currentBlock,
		ResolvedBy:      "governance",
		Justification:   msg.Justification,
	}

	challengerAddr, _ := sdk.AccAddressFromBech32(challenge.Challenger)

	switch msg.Outcome {
	case "upheld":
		// Challenger gets 7.7x stake + 777 LGM bonus
		stakeInt := new(big.Int)
		stakeInt.SetString(challenge.ChallengeStake, 10)
		// 7.7x = stake * ChallengeStakeMultiplierBPS / 1_000_000
		reward := new(big.Int).Mul(stakeInt, big.NewInt(int64(types.ChallengeStakeMultiplierBPS)))
		reward.Div(reward, big.NewInt(1_000_000))
		bonus := new(big.Int)
		bonus.SetString(types.CaptureExposureBonus, 10)
		reward.Add(reward, bonus)
		challengerReward = reward.String()
		resolution.ChallengerReward = challengerReward

		// Pay challenger: original stake + reward
		totalPayout := new(big.Int).Add(stakeInt, reward)
		payoutCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(totalPayout)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, payoutCoins); err != nil {
			return nil, fmt.Errorf("failed to pay challenger reward: %w", err)
		}

		// Slash accused validators
		slashes := make(map[string]string)
		for _, validatorAddr := range challenge.AccusedValidators {
			if k.stakingKeeper != nil {
				stake := k.stakingKeeper.GetEffectiveStake(ctx, validatorAddr)
				slashAmount := (stake * types.ValidatorSlashOnUpheldBPS) / 1_000_000
				if slashAmount > 0 {
					slashes[validatorAddr] = fmt.Sprintf("%d", slashAmount)
					_ = k.stakingKeeper.SlashValidator(ctx, validatorAddr, fmt.Sprintf("%d", slashAmount), "capture_challenge_upheld")
				}
			}
		}
		resolution.ValidatorSlashes = slashes

		// Pause domain
		challenge.DomainPausedUntil = currentBlock + types.DefaultDomainPauseDurationBlocks

		pool.ChallengesUpheld++
		challenge.Status = string(types.ChallengeUpheld)

	case "rejected":
		// Challenger loses 77% of stake to bounty pool
		stakeInt := new(big.Int)
		stakeInt.SetString(challenge.ChallengeStake, 10)
		penalty := new(big.Int).Mul(stakeInt, big.NewInt(int64(types.FailedChallengePenaltyBPS)))
		penalty.Div(penalty, big.NewInt(1_000_000))

		// Penalty stays in module (tracked by pool balance)
		poolBal := new(big.Int)
		poolBal.SetString(pool.Balance, 10)
		poolBal.Add(poolBal, penalty)
		pool.Balance = poolBal.String()

		// Return remainder (23%) to challenger
		remainder := new(big.Int).Sub(stakeInt, penalty)
		if remainder.Sign() > 0 {
			returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(remainder)))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, returnCoins); err != nil {
				return nil, fmt.Errorf("failed to return stake remainder: %w", err)
			}
		}

		pool.ChallengesRejected++
		challenge.Status = string(types.ChallengeRejected)

	case "partial":
		// Partial: reduced reward (half of full multiplier), no domain pause
		stakeInt := new(big.Int)
		stakeInt.SetString(challenge.ChallengeStake, 10)
		// Half of ChallengeStakeMultiplierBPS
		halfMultiplierBPS := int64(types.ChallengeStakeMultiplierBPS / 2)
		reward := new(big.Int).Mul(stakeInt, big.NewInt(halfMultiplierBPS))
		reward.Div(reward, big.NewInt(1_000_000))
		challengerReward = reward.String()
		resolution.ChallengerReward = challengerReward

		// Pay challenger: original stake + partial reward
		totalPayout := new(big.Int).Add(stakeInt, reward)
		payoutCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(totalPayout)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, payoutCoins); err != nil {
			return nil, fmt.Errorf("failed to pay partial challenger reward: %w", err)
		}

		challenge.Status = string(types.ChallengePartial)
	}

	challenge.Resolution = &resolution
	k.SetChallenge(ctx, challenge)
	pool.LastUpdatedBlock = currentBlock
	k.SetBountyPool(ctx, pool)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"capture_challenge_resolved",
			sdk.NewAttribute("challenge_id", msg.ChallengeId),
			sdk.NewAttribute("outcome", msg.Outcome),
			sdk.NewAttribute("challenger_reward", challengerReward),
		),
	)

	return &types.MsgResolveChallengeResponse{ChallengerReward: challengerReward}, nil
}


// UpdateParams handles MsgUpdateParams -- governance-gated parameter update.
func (k Keeper) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, *msg.Params)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"capture_challenge_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)
	return &types.MsgUpdateParamsResponse{}, nil
}
