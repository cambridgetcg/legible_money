package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/evaluation"
	"github.com/legible-money/legible/x/knowledge/types"
)

// OpenChallenge validates and creates a challenge against a provisional/evaluated claim.
func (k Keeper) OpenChallenge(
	ctx sdk.Context,
	challenger, claimId, factId, stake, reason string,
	evidenceIds []string,
	counterClaim string,
) (types.ProvisionalChallenge, error) {
	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())

	// Verify claim exists and is in challengeable state
	claim, found := k.GetClaim(ctx, claimId)
	if !found {
		return types.ProvisionalChallenge{}, fmt.Errorf("%w: claim %s", types.ErrClaimNotFound, claimId)
	}
	if claim.Status != types.ClaimStatusProvisional && claim.Status != types.ClaimStatusEvaluated {
		return types.ProvisionalChallenge{}, fmt.Errorf("%w: claim status is %s", types.ErrNotProvisional, claim.Status)
	}

	// Check challenge window is still open
	if height >= claim.ChallengeWindowEnd {
		return types.ProvisionalChallenge{}, fmt.Errorf("%w: window closed at block %d", types.ErrChallengeWindowClosed, claim.ChallengeWindowEnd)
	}

	// Cannot self-challenge
	if challenger == claim.Submitter {
		return types.ProvisionalChallenge{}, types.ErrSelfChallenge
	}

	// Check max concurrent challenges
	ep := k.GetExtendedParams(ctx)
	if claim.ActiveChallengeCount >= ep.MaxConcurrentChallenges {
		return types.ProvisionalChallenge{}, types.ErrMaxChallengesReached
	}

	// Check exponential cooldown
	if lastHeight, found := k.GetChallengerCooldown(ctx, challenger, claimId); found {
		// Count previous attempts
		attemptNumber := uint32(0)
		k.IterateProvisionalChallenges(ctx, func(c types.ProvisionalChallenge) bool {
			if c.ClaimId == claimId && c.Challenger == challenger {
				attemptNumber++
			}
			return false
		})
		cooldown := ep.BaseCooldownBlocks
		for i := uint32(0); i < attemptNumber; i++ {
			cooldown *= 2
			if cooldown > ep.MaxCooldownBlocks {
				cooldown = ep.MaxCooldownBlocks
				break
			}
		}
		if height < lastHeight+cooldown {
			return types.ProvisionalChallenge{}, fmt.Errorf("%w: must wait until block %d",
				types.ErrChallengeCooldown, lastHeight+cooldown)
		}
	}

	// Validate stake amount
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return types.ProvisionalChallenge{}, fmt.Errorf("%w: invalid stake amount", types.ErrInsufficientStake)
	}
	minStake := new(big.Int)
	minChallengeStake := params.MinChallengeStake
	if minChallengeStake == "" {
		minChallengeStake = fmt.Sprintf("%d", ep.MinClaimStakeUlgm)
	}
	if _, ok := minStake.SetString(minChallengeStake, 10); !ok {
		minStake.SetInt64(int64(ep.MinClaimStakeUlgm))
	}
	if stakeAmount.Cmp(minStake) < 0 {
		return types.ProvisionalChallenge{}, fmt.Errorf("%w: need at least %s, got %s",
			types.ErrInsufficientStake, params.MinChallengeStake, stake)
	}

	// FARM-9: Proportional stake requirement — challenger must stake ≥50% of target claim's stake.
	// This prevents asymmetric challenge spam where low-stake challengers target high-stake facts.
	if claim.StakeAmount != "" {
		claimStake := new(big.Int)
		if _, ok := claimStake.SetString(claim.StakeAmount, 10); ok && claimStake.Sign() > 0 {
			minPropStake := new(big.Int).Mul(claimStake, big.NewInt(int64(ep.ChallengeStakeRatioMinBps)))
			minPropStake.Div(minPropStake, big.NewInt(1_000_000))
			if stakeAmount.Cmp(minPropStake) < 0 {
				return types.ProvisionalChallenge{}, fmt.Errorf("%w: must stake at least 50%% of claim stake (%s), got %s",
					types.ErrInsufficientStake, minPropStake.String(), stake)
			}
		}
	}

	// Check evidence requirement
	if params.ChallengeEvidenceRequired && len(evidenceIds) == 0 {
		return types.ProvisionalChallenge{}, types.ErrEvidenceRequired
	}

	// Lock challenger stake
	challengerAddr, err := sdk.AccAddressFromBech32(challenger)
	if err != nil {
		return types.ProvisionalChallenge{}, fmt.Errorf("invalid challenger address: %w", err)
	}
	stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, challengerAddr, types.ModuleName, sdk.NewCoins(stakeCoin),
	); err != nil {
		return types.ProvisionalChallenge{}, fmt.Errorf("%w: %v", types.ErrInsufficientStake, err)
	}

	// Determine resolution path
	resolutionPath := "arbiter"
	if types.IsDeterministicCategory(claim.EpistemicCategory) {
		resolutionPath = "deterministic"
	}

	challengeID := GenerateChallengeID(claimId, challenger, height)

	challenge := types.ProvisionalChallenge{
		Id:              challengeID,
		ClaimId:         claimId,
		FactId:          factId,
		Challenger:      challenger,
		Stake:           stake,
		Reason:          reason,
		EvidenceIds:     evidenceIds,
		CounterClaim:    counterClaim,
		Status:          types.ChallengeStatusOpen,
		ResolutionPath:  resolutionPath,
		CreatedAtHeight: height,
	}

	k.SetProvisionalChallenge(ctx, challenge)
	k.SetChallengerCooldown(ctx, challenger, claimId, height)

	// Update claim challenge count
	claim.ActiveChallengeCount++
	k.SetClaim(ctx, claim)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("provisional_challenge_opened",
			sdk.NewAttribute("challenge_id", challengeID),
			sdk.NewAttribute("claim_id", claimId),
			sdk.NewAttribute("challenger", challenger),
			sdk.NewAttribute("stake", stake),
			sdk.NewAttribute("path", resolutionPath),
		),
	)

	return challenge, nil
}

// ResolvePendingDeterministicChallenges resolves all open deterministic (Path A) challenges.
// Called in BeginBlocker.
func (k Keeper) ResolvePendingDeterministicChallenges(ctx sdk.Context, currentHeight uint64) {
	k.IterateProvisionalChallenges(ctx, func(challenge types.ProvisionalChallenge) bool {
		if challenge.Status != types.ChallengeStatusOpen || challenge.ResolutionPath != "deterministic" {
			return false
		}

		outcome, newScore, err := k.ResolveDeterministicChallenge(ctx, challenge)
		if err != nil {
			k.Logger(ctx).Error("failed to resolve deterministic challenge",
				"challenge_id", challenge.Id, "error", err)
			return false
		}

		if err := k.ApplyChallengeOutcome(ctx, challenge, outcome, newScore, currentHeight); err != nil {
			k.Logger(ctx).Error("failed to apply challenge outcome",
				"challenge_id", challenge.Id, "error", err)
		}
		return false
	})
}

// ResolveDeterministicChallenge re-runs the evaluation engine with counter-evidence
// factored in. For deterministic categories (protocol, computational, formal, analytic).
//
// The challenge's evidence IDs are injected as additional contradicting references
// on the claim before re-evaluation. The evaluation engine's fact graph stage
// will discover these contradictions and adjust the score accordingly.
func (k Keeper) ResolveDeterministicChallenge(
	ctx sdk.Context,
	challenge types.ProvisionalChallenge,
) (string, uint64, error) {
	claim, found := k.GetClaim(ctx, challenge.ClaimId)
	if !found {
		return "", 0, fmt.Errorf("%w: %s", types.ErrClaimNotFound, challenge.ClaimId)
	}

	// Build a modified claim that includes counter-evidence as additional references.
	// The evaluation engine's fact graph stage (stage 3) queries GetContradictingFacts
	// and GetSupportingFacts via the EvaluationContext, so counter-evidence that
	// contradicts the claim's references will naturally reduce the score.
	challengedClaim := &types.Claim{
		Id:                claim.Id,
		ClaimText:         claim.ClaimText,
		ContentHash:       claim.ContentHash,
		Domain:            claim.Domain,
		EpistemicCategory: claim.EpistemicCategory,
		Submitter:         claim.Submitter,
		SubmitterDid:      claim.SubmitterDid,
		StakeAmount:       claim.StakeAmount,
		Status:            claim.Status,
		SubmittedAtHeight: claim.SubmittedAtHeight,
		LogicZone:         claim.LogicZone,
		RoundId:           claim.RoundId,
		Evidence:          claim.Evidence,
		StructuralScore:   claim.StructuralScore,
		References: append(
			append([]string{}, claim.References...),
			challenge.EvidenceIds...,
		),
		ChallengeWindowStart: claim.ChallengeWindowStart,
		ChallengeWindowEnd:   claim.ChallengeWindowEnd,
		ProvisionalFactId:    claim.ProvisionalFactId,
		ActiveChallengeCount: claim.ActiveChallengeCount,
		ReverificationFactId: claim.ReverificationFactId,
	}

	// Use the same evaluation pipeline as provisional acceptance
	evalCtx := k.BuildEvaluationContext(ctx)
	currentHeight := uint64(ctx.BlockHeight())

	syntheticRound := types.VerificationRound{
		Id:          "challenge:" + challenge.Id,
		ClaimId:     claim.Id,
		Phase:       "aggregation",
		StartHeight: currentHeight,
	}

	evalResult, err := evaluation.EvaluateClaim(ctx, evalCtx, &syntheticRound, challengedClaim)
	if err != nil {
		return "", 0, fmt.Errorf("re-evaluation failed for challenge %s: %w", challenge.Id, err)
	}

	newScore := evalResult.Confidence
	originalScore := claim.StructuralScore

	// Determine outcome based on score change
	ep := k.GetExtendedParams(ctx)
	if newScore < ep.RejectThreshold {
		return types.ChallengeOutcomeOverturned, newScore, nil
	}

	// Significant reduction (>20%) = weakened
	if originalScore > 0 && newScore < originalScore {
		reductionBps := ((originalScore - newScore) * 1_000_000) / originalScore
		if reductionBps > 200_000 {
			return types.ChallengeOutcomeWeakened, newScore, nil
		}
	}

	// No significant change = challenge fails, fact upheld
	return types.ChallengeOutcomeUpheld, newScore, nil
}

// ApplyChallengeOutcome applies the result of a challenge resolution.
func (k Keeper) ApplyChallengeOutcome(
	ctx sdk.Context,
	challenge types.ProvisionalChallenge,
	outcome string,
	newScore uint64,
	currentHeight uint64,
) error {
	params := k.GetParams(ctx)

	challenge.Status = types.ChallengeStatusResolved
	challenge.Outcome = outcome
	challenge.ResolvedAtHeight = currentHeight

	claim, claimFound := k.GetClaim(ctx, challenge.ClaimId)
	if !claimFound {
		return fmt.Errorf("%w: %s", types.ErrClaimNotFound, challenge.ClaimId)
	}

	switch outcome {
	case types.ChallengeOutcomeUpheld:
		// Fact survives — slash challenger, boost fact confidence
		failedSlashBps := params.FailedChallengeSlashBps
		if failedSlashBps == 0 {
			failedSlashBps = k.GetExtendedParams(ctx).ChallengeStakeRatioMinBps
		}
		k.slashChallengerBond(ctx, challenge, failedSlashBps)

		if claim.ProvisionalFactId != "" {
			if fact, found := k.GetFact(ctx, claim.ProvisionalFactId); found {
				// Boost confidence by 11%
				boost := uint64(110_000) // 11%
				newConf := fact.Confidence + boost
				ep := k.GetExtendedParams(ctx)
				categoryCeiling := getCategoryCeilingFromParams(fact.EpistemicCategory, ep)
				if newConf > categoryCeiling {
					newConf = categoryCeiling
				}
				fact.Confidence = newConf
				fact.SurvivedChallenges++
				k.SetFact(ctx, fact)
			}
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("challenge_upheld",
				sdk.NewAttribute("challenge_id", challenge.Id),
				sdk.NewAttribute("claim_id", challenge.ClaimId),
			),
		)

	case types.ChallengeOutcomeOverturned:
		// Challenge succeeds — revoke fact, slash submitter, reward challenger
		if claim.ProvisionalFactId != "" {
			if fact, found := k.GetFact(ctx, claim.ProvisionalFactId); found {
				fact.Status = types.FactStatusRevoked
				fact.IsProvisional = false
				k.SetFact(ctx, fact)

				// Cascade confidence reduction to dependent facts
				k.cascadeConfidenceReduction(ctx, fact.Id, 220_000) // 22% reduction
			}
		}

		// Slash submitter stake and reward challenger
		k.slashSubmitterAndRewardChallenger(ctx, claim, challenge, params)

		claim.Status = types.ClaimStatusRejected
		k.SetClaim(ctx, claim)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("challenge_overturned",
				sdk.NewAttribute("challenge_id", challenge.Id),
				sdk.NewAttribute("claim_id", challenge.ClaimId),
			),
		)

	case types.ChallengeOutcomeWeakened:
		// Partial success — reduce confidence, return challenger bond
		if claim.ProvisionalFactId != "" {
			if fact, found := k.GetFact(ctx, claim.ProvisionalFactId); found {
				fact.Confidence = newScore
				k.SetFact(ctx, fact)
			}
		}
		claim.StructuralScore = newScore
		k.SetClaim(ctx, claim)
		k.returnChallengerBond(ctx, challenge)

	case types.ChallengeOutcomeInconclusive:
		// No clear winner — return challenger bond
		k.returnChallengerBond(ctx, challenge)

	case types.ChallengeOutcomeUndecidability:
		// Gödel zones — cap confidence at zone ceiling, return bond
		if claim.ProvisionalFactId != "" {
			if fact, found := k.GetFact(ctx, claim.ProvisionalFactId); found {
				zoneCeiling := uint64(1_000_000)
				if k.ontologyKeeper != nil {
					zoneCeiling = k.ontologyKeeper.GetZoneConfidenceCeiling(ctx, fact.LogicZone)
				}
				if fact.Confidence > zoneCeiling {
					fact.Confidence = zoneCeiling
				}
				fact.Status = types.FactStatusContested
				k.SetFact(ctx, fact)
			}
		}
		k.returnChallengerBond(ctx, challenge)
	}

	// Decrement active challenge count
	if claimFound {
		if claim.ActiveChallengeCount > 0 {
			claim.ActiveChallengeCount--
		}
		k.SetClaim(ctx, claim)
	}

	k.SetProvisionalChallenge(ctx, challenge)
	return nil
}

// ---------- Stake Operations ----------

// slashChallengerBond slashes a portion of the challenger's bond and returns the rest.
func (k Keeper) slashChallengerBond(ctx sdk.Context, challenge types.ProvisionalChallenge, slashBps uint64) {
	if k.bankKeeper == nil || challenge.Stake == "" {
		return
	}

	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(challenge.Stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return
	}

	slashAmount := new(big.Int).Mul(stakeAmount, big.NewInt(int64(slashBps)))
	slashAmount.Div(slashAmount, big.NewInt(1_000_000))
	returnAmount := new(big.Int).Sub(stakeAmount, slashAmount)

	challengerAddr, err := sdk.AccAddressFromBech32(challenge.Challenger)
	if err != nil {
		return
	}

	if returnAmount.Sign() > 0 {
		returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(returnAmount)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, challengerAddr, returnCoins,
		); err != nil {
			k.Logger(ctx).Error("failed to return challenger bond",
				"challenge_id", challenge.Id, "error", err)
		}
	}

	if slashAmount.Sign() > 0 {
		burnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(slashAmount)))
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
			k.Logger(ctx).Error("failed to burn slashed challenger bond",
				"challenge_id", challenge.Id, "error", err)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("challenger_bond_slashed",
			sdk.NewAttribute("challenge_id", challenge.Id),
			sdk.NewAttribute("challenger", challenge.Challenger),
			sdk.NewAttribute("slashed", slashAmount.String()),
			sdk.NewAttribute("returned", returnAmount.String()),
		),
	)
}

// returnChallengerBond returns the full bond to the challenger.
func (k Keeper) returnChallengerBond(ctx sdk.Context, challenge types.ProvisionalChallenge) {
	if k.bankKeeper == nil || challenge.Stake == "" {
		return
	}

	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(challenge.Stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return
	}

	challengerAddr, err := sdk.AccAddressFromBech32(challenge.Challenger)
	if err != nil {
		return
	}

	returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount)))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, challengerAddr, returnCoins,
	); err != nil {
		k.Logger(ctx).Error("failed to return challenger bond",
			"challenge_id", challenge.Id, "error", err)
	}
}

// slashSubmitterAndRewardChallenger slashes the submitter's stake and pays a portion to the challenger.
func (k Keeper) slashSubmitterAndRewardChallenger(
	ctx sdk.Context,
	claim *types.Claim,
	challenge types.ProvisionalChallenge,
	params *types.Params,
) {
	if k.bankKeeper == nil || claim.StakeAmount == "" {
		return
	}

	claimStake := new(big.Int)
	if _, ok := claimStake.SetString(claim.StakeAmount, 10); !ok || claimStake.Sign() <= 0 {
		return
	}

	// Calculate slash amount (22% of claim stake)
	slashAmount := new(big.Int).Mul(claimStake, big.NewInt(int64(params.InvalidClaimSlashBps)))
	slashAmount.Div(slashAmount, big.NewInt(1_000_000))

	// Challenger reward = 30% of slashed amount
	rewardBps := params.SuccessfulChallengeRewardBps
	if rewardBps == 0 {
		rewardBps = 300_000 // default 30%
	}
	rewardAmount := new(big.Int).Mul(slashAmount, big.NewInt(int64(rewardBps)))
	rewardAmount.Div(rewardAmount, big.NewInt(1_000_000))

	// FARM-9: Cap reward at 100% of challenger's own stake to prevent asymmetric payoffs.
	ep := k.GetExtendedParams(ctx)
	challengerStake := new(big.Int)
	if _, ok := challengerStake.SetString(challenge.Stake, 10); ok && challengerStake.Sign() > 0 {
		maxReward := new(big.Int).Mul(challengerStake, big.NewInt(int64(ep.ChallengeRewardCapBps)))
		maxReward.Div(maxReward, big.NewInt(1_000_000))
		if rewardAmount.Cmp(maxReward) > 0 {
			rewardAmount.Set(maxReward)
		}
	}

	burnAmount := new(big.Int).Sub(slashAmount, rewardAmount)
	returnToSubmitter := new(big.Int).Sub(claimStake, slashAmount)

	// Return non-slashed portion to submitter
	submitterAddr, err := sdk.AccAddressFromBech32(claim.Submitter)
	if err == nil && returnToSubmitter.Sign() > 0 {
		returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(returnToSubmitter)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, submitterAddr, returnCoins,
		); err != nil {
			k.Logger(ctx).Error("failed to return remaining stake to submitter",
				"claim_id", claim.Id, "error", err)
		}
	}

	// Pay reward to challenger
	challengerAddr, err := sdk.AccAddressFromBech32(challenge.Challenger)
	if err == nil && rewardAmount.Sign() > 0 {
		rewardCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(rewardAmount)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, challengerAddr, rewardCoins,
		); err != nil {
			k.Logger(ctx).Error("failed to pay challenger reward",
				"challenge_id", challenge.Id, "error", err)
		}
	}

	// Return challenger's own bond in full
	k.returnChallengerBond(ctx, challenge)

	// Burn the remaining slashed portion
	if burnAmount.Sign() > 0 {
		burnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(burnAmount)))
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
			k.Logger(ctx).Error("failed to burn slashed stake",
				"claim_id", claim.Id, "error", err)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("submitter_slashed_challenger_rewarded",
			sdk.NewAttribute("claim_id", claim.Id),
			sdk.NewAttribute("challenge_id", challenge.Id),
			sdk.NewAttribute("slashed", slashAmount.String()),
			sdk.NewAttribute("reward", rewardAmount.String()),
			sdk.NewAttribute("burned", burnAmount.String()),
		),
	)
}

// cascadeConfidenceReduction reduces confidence of facts that depend on a revoked fact.
func (k Keeper) cascadeConfidenceReduction(ctx sdk.Context, revokedFactId string, reductionBps uint64) {
	// Find all facts that reference the revoked fact
	incoming := k.GetIncomingReferences(ctx, revokedFactId)
	for _, ref := range incoming {
		if ref.ReferenceType != "supports" {
			continue
		}
		dependentFact, found := k.GetFact(ctx, ref.FromFactId)
		if !found {
			continue
		}
		reduction := (dependentFact.Confidence * reductionBps) / 1_000_000
		if reduction > dependentFact.Confidence {
			dependentFact.Confidence = 0
		} else {
			dependentFact.Confidence -= reduction
		}
		k.SetFact(ctx, dependentFact)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("confidence_cascade",
				sdk.NewAttribute("fact_id", dependentFact.Id),
				sdk.NewAttribute("caused_by", revokedFactId),
				sdk.NewAttribute("reduction_bps", fmt.Sprintf("%d", reductionBps)),
			),
		)
	}
}

// getCategoryCeilingFromParams returns the max confidence for an epistemic category.
func getCategoryCeilingFromParams(category string, ep types.ExtendedParams) uint64 {
	if profile, ok := ep.GetCategoryDecayProfile(category); ok {
		return profile.MaxConfidence
	}
	return 1_000_000
}
