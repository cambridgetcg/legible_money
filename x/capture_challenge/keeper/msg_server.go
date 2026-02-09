package keeper

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/big"

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
		Status:            types.ChallengeEvidencePeriod,
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

	if challenge.Status != types.ChallengeEvidencePeriod {
		return nil, types.ErrInvalidChallengeStatus
	}

	if currentBlock > challenge.EvidenceDeadline {
		return nil, types.ErrDeadlinePassed
	}

	challenge.Evidence = append(challenge.Evidence, msg.Evidence)
	k.SetChallenge(ctx, challenge)

	return &types.MsgAddEvidenceResponse{EvidenceCount: len(challenge.Evidence)}, nil
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

	if challenge.Status != types.ChallengeUnderReview && challenge.Status != types.ChallengeEvidencePeriod {
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

	switch msg.Outcome {
	case "upheld":
		// Challenger gets 7.7x stake + 777 LGM bonus
		stakeInt := new(big.Int)
		stakeInt.SetString(challenge.ChallengeStake, 10)
		// 7.7x = stake * 77 / 10
		reward := new(big.Int).Mul(stakeInt, big.NewInt(77))
		reward.Div(reward, big.NewInt(10))
		bonus := new(big.Int)
		bonus.SetString(types.CaptureExposureBonus, 10)
		reward.Add(reward, bonus)
		challengerReward = reward.String()
		resolution.ChallengerReward = challengerReward

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
		challenge.DomainPausedUntil = currentBlock + types.DomainPauseDurationBlocks

		pool.ChallengesUpheld++
		challenge.Status = types.ChallengeUpheld

	case "rejected":
		// Challenger loses 77% of stake to bounty pool
		stakeInt := new(big.Int)
		stakeInt.SetString(challenge.ChallengeStake, 10)
		penalty := new(big.Int).Mul(stakeInt, big.NewInt(int64(types.FailedChallengePenaltyBPS)))
		penalty.Div(penalty, big.NewInt(1_000_000))

		poolBal := new(big.Int)
		poolBal.SetString(pool.Balance, 10)
		poolBal.Add(poolBal, penalty)
		pool.Balance = poolBal.String()

		pool.ChallengesRejected++
		challenge.Status = types.ChallengeRejected

	case "partial":
		// Partial: reduced reward, no domain pause
		stakeInt := new(big.Int)
		stakeInt.SetString(challenge.ChallengeStake, 10)
		// 3.85x (half of 7.7x)
		reward := new(big.Int).Mul(stakeInt, big.NewInt(385))
		reward.Div(reward, big.NewInt(100))
		challengerReward = reward.String()
		resolution.ChallengerReward = challengerReward

		challenge.Status = types.ChallengePartial
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
