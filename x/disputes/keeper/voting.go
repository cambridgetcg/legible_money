package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/disputes/types"
)

// CastVote records an arbiter's vote on a dispute.
func (k Keeper) CastVote(ctx sdk.Context, disputeId, arbiter, vote, rationale string) error {
	dispute, found := k.GetDispute(ctx, disputeId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrDisputeNotFound, disputeId)
	}

	// Check dispute is in voting phase
	if dispute.Status != string(types.DisputeStatusVoting) {
		return fmt.Errorf("%w: dispute is in %s phase", types.ErrNotInVotingPhase, dispute.Status)
	}

	// Check voting deadline not passed
	currentHeight := uint64(ctx.BlockHeight())
	if currentHeight > dispute.VotingDeadline {
		return fmt.Errorf("%w: deadline was at height %d", types.ErrVotingDeadlinePassed, dispute.VotingDeadline)
	}

	// Check arbiter is assigned to this dispute
	isArbiter := false
	for _, a := range dispute.Arbiters {
		if a == arbiter {
			isArbiter = true
			break
		}
	}
	if !isArbiter {
		return fmt.Errorf("%w: %s", types.ErrNotArbiter, arbiter)
	}

	// Check arbiter hasn't already voted
	if _, exists := k.GetVote(ctx, disputeId, arbiter); exists {
		return fmt.Errorf("%w: %s on dispute %s", types.ErrAlreadyVoted, arbiter, disputeId)
	}

	// Get arbiter's effective stake
	stake := k.stakingKeeper.GetEffectiveStake(ctx, arbiter)

	disputeVote := types.DisputeVote{
		DisputeId: disputeId,
		Arbiter:   arbiter,
		Vote:      vote,
		Stake:     fmt.Sprintf("%d", stake),
		Rationale: rationale,
		VotedAt:   currentHeight,
	}
	k.SetVote(ctx, disputeVote)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dispute_vote_cast",
			sdk.NewAttribute("dispute_id", disputeId),
			sdk.NewAttribute("arbiter", arbiter),
			sdk.NewAttribute("vote", vote),
			sdk.NewAttribute("stake", fmt.Sprintf("%d", stake)),
		),
	)

	k.Logger(ctx).Info("vote cast on dispute",
		"dispute_id", disputeId,
		"arbiter", arbiter,
		"vote", vote,
	)

	return nil
}

// TallyVotes computes the outcome of a dispute from the cast votes.
// Uses stake-weighted voting with quorum and majority thresholds.
func (k Keeper) TallyVotes(ctx sdk.Context, dispute types.Dispute) (string, error) {
	params := k.GetParams(ctx)
	tierConfig, err := k.GetTierConfig(params, types.EscalationTier(dispute.Tier))
	if err != nil {
		return types.OutcomeInconclusive, err
	}

	votes := k.GetVotes(ctx, dispute.Id)
	if len(votes) == 0 {
		return types.OutcomeInconclusive, nil
	}

	// Calculate stake-weighted totals
	upholdWeight := new(big.Int)
	overturnWeight := new(big.Int)
	abstainWeight := new(big.Int)
	totalWeight := new(big.Int)

	for _, v := range votes {
		stake := new(big.Int)
		if _, ok := stake.SetString(v.Stake, 10); !ok {
			continue
		}
		totalWeight.Add(totalWeight, stake)

		switch v.Vote {
		case "uphold":
			upholdWeight.Add(upholdWeight, stake)
		case "overturn":
			overturnWeight.Add(overturnWeight, stake)
		case "abstain":
			abstainWeight.Add(abstainWeight, stake)
		}
	}

	if totalWeight.Sign() == 0 {
		return types.OutcomeInconclusive, nil
	}

	// Check quorum: (votes cast / total arbiter count) >= quorum threshold
	// We approximate by checking if enough arbiters voted
	arbiterCount := len(dispute.Arbiters)
	if arbiterCount == 0 {
		return types.OutcomeInconclusive, nil
	}

	// Quorum check: participation rate in basis points
	participationBps := uint64(len(votes)) * 1000000 / uint64(arbiterCount)
	if participationBps < tierConfig.QuorumThreshold {
		k.Logger(ctx).Info("dispute quorum not met",
			"dispute_id", dispute.Id,
			"participation_bps", participationBps,
			"quorum_threshold", tierConfig.QuorumThreshold,
		)
		return types.OutcomeInconclusive, nil
	}

	// Calculate majority: which side has more stake-weighted votes?
	// Exclude abstentions from majority calculation
	votingWeight := new(big.Int).Add(upholdWeight, overturnWeight)
	if votingWeight.Sign() == 0 {
		return types.OutcomeInconclusive, nil
	}

	// upholdRatio = upholdWeight * 1000000 / votingWeight (basis points)
	upholdRatio := new(big.Int).Mul(upholdWeight, big.NewInt(1000000))
	upholdRatio.Div(upholdRatio, votingWeight)

	overturnRatio := new(big.Int).Mul(overturnWeight, big.NewInt(1000000))
	overturnRatio.Div(overturnRatio, votingWeight)

	majorityThreshold := big.NewInt(int64(tierConfig.MajorityThreshold))

	if upholdRatio.Cmp(majorityThreshold) >= 0 {
		return types.OutcomeUpheld, nil
	}
	if overturnRatio.Cmp(majorityThreshold) >= 0 {
		return types.OutcomeOverturned, nil
	}

	// No clear majority
	return types.OutcomeInconclusive, nil
}

// ResolveDispute finalizes a dispute with the given outcome and distributes rewards/slashes.
func (k Keeper) ResolveDispute(ctx sdk.Context, dispute types.Dispute, outcome string) error {
	if dispute.Status == string(types.DisputeStatusResolved) {
		return fmt.Errorf("%w: %s", types.ErrDisputeAlreadyResolved, dispute.Id)
	}

	params := k.GetParams(ctx)
	dispute.Status = string(types.DisputeStatusResolved)
	dispute.Outcome = string(outcome)

	// Calculate slashing and reward amounts
	bondAmount := new(big.Int)
	if _, ok := bondAmount.SetString(dispute.Bond, 10); !ok {
		// If bond can't be parsed, just mark resolved without distributions
		k.SetDispute(ctx, dispute)
		return nil
	}

	// slash = bond * slashRate / 1000000
	slashAmount := new(big.Int).Mul(bondAmount, big.NewInt(int64(params.SlashRateLoser)))
	slashAmount.Div(slashAmount, big.NewInt(1000000))

	// winnerReward = slash * rewardRate / 1000000
	winnerReward := new(big.Int).Mul(slashAmount, big.NewInt(int64(params.RewardRateWinner)))
	winnerReward.Div(winnerReward, big.NewInt(1000000))

	// arbiterReward = slash * arbiterRate / 1000000
	arbiterReward := new(big.Int).Mul(slashAmount, big.NewInt(int64(params.ArbiterRewardRate)))
	arbiterReward.Div(arbiterReward, big.NewInt(1000000))

	switch outcome {
	case types.OutcomeOverturned:
		// Challenger wins: return bond + reward from slash
		k.distributeToAddress(ctx, dispute.Challenger, bondAmount)
		if winnerReward.Sign() > 0 {
			k.distributeToAddress(ctx, dispute.Challenger, winnerReward)
		}

	case types.OutcomeUpheld:
		// Defender wins: challenger loses slashed portion of bond, gets remainder back
		remainder := new(big.Int).Sub(bondAmount, slashAmount)
		if remainder.Sign() > 0 {
			k.distributeToAddress(ctx, dispute.Challenger, remainder)
		}
		// Burn the slashed amount minus rewards
		burnAmount := new(big.Int).Sub(slashAmount, winnerReward)
		burnAmount.Sub(burnAmount, arbiterReward)
		if burnAmount.Sign() > 0 {
			k.burnFromModule(ctx, burnAmount)
		}
		// Reward the defender from slash proceeds
		if winnerReward.Sign() > 0 {
			k.distributeToAddress(ctx, dispute.Defender, winnerReward)
		}

	case types.OutcomeInconclusive:
		// No clear outcome: return bond to challenger
		k.distributeToAddress(ctx, dispute.Challenger, bondAmount)
	}

	// Distribute arbiter rewards to correct voters, slash wrong voters
	if len(dispute.Arbiters) > 0 && outcome != types.OutcomeInconclusive {
		votes := k.GetVotes(ctx, dispute.Id)
		voteMap := make(map[string]string, len(votes))
		for _, v := range votes {
			voteMap[v.Arbiter] = v.Vote
		}

		// Determine winning vote alignment
		var winningVote string
		if outcome == types.OutcomeUpheld {
			winningVote = "uphold"
		} else {
			winningVote = "overturn"
		}

		// Count correct voters for reward splitting
		correctCount := 0
		for _, arbiter := range dispute.Arbiters {
			if voteMap[arbiter] == winningVote {
				correctCount++
			}
		}

		for _, arbiter := range dispute.Arbiters {
			vote, voted := voteMap[arbiter]
			if !voted {
				// Non-voter: slash 10% of effective stake for dereliction
				// (TS: 10% of minArbiterStake — aligned to 10% rate)
				arbiterStake := k.stakingKeeper.GetEffectiveStake(ctx, arbiter)
				nonVoterSlash := new(big.Int).SetUint64(arbiterStake * 10 / 100)
				if nonVoterSlash.Sign() > 0 {
					k.burnFromModule(ctx, nonVoterSlash) // burn the slash amount
					ctx.EventManager().EmitEvent(sdk.NewEvent("arbiter_slashed",
						sdk.NewAttribute("dispute_id", dispute.Id),
						sdk.NewAttribute("arbiter", arbiter),
						sdk.NewAttribute("reason", "non_voter"),
						sdk.NewAttribute("amount", nonVoterSlash.String()),
					))
				}
				continue
			}

			if vote == winningVote {
				// Correct voter: gets share of arbiter reward pool
				if arbiterReward.Sign() > 0 && correctCount > 0 {
					share := new(big.Int).Div(arbiterReward, big.NewInt(int64(correctCount)))
					if share.Sign() > 0 {
						k.distributeToAddress(ctx, arbiter, share)
					}
				}
			} else if vote != "abstain" {
				// Wrong voter: slash 11% of effective stake (TS: wrongArbitrationSlash = 0.11)
				arbiterStake := k.stakingKeeper.GetEffectiveStake(ctx, arbiter)
				wrongSlash := new(big.Int).SetUint64(arbiterStake * 11 / 100)
				if wrongSlash.Sign() > 0 {
					k.burnFromModule(ctx, wrongSlash)
					ctx.EventManager().EmitEvent(sdk.NewEvent("arbiter_slashed",
						sdk.NewAttribute("dispute_id", dispute.Id),
						sdk.NewAttribute("arbiter", arbiter),
						sdk.NewAttribute("reason", "wrong_vote"),
						sdk.NewAttribute("amount", wrongSlash.String()),
					))
				}
			}
			// Abstainers: no reward, no slash
		}
	}

	// Propagate outcome to fact confidence and status
	k.resolveFactOutcome(ctx, dispute.FactId, outcome, types.EscalationTier(dispute.Tier))

	// Vesting integration: resume on upheld, clawback on overturned
	if k.vestingKeeper != nil && dispute.ClaimId != "" {
		switch outcome {
		case types.OutcomeUpheld:
			// Fact survived — resume vesting
			if err := k.vestingKeeper.ResumeVestingByClaimId(ctx, dispute.ClaimId); err != nil {
				k.Logger(ctx).Error("failed to resume vesting", "claim_id", dispute.ClaimId, "error", err)
			}
		case types.OutcomeOverturned:
			// Fact overturned — trigger clawback (33% released + 100% unvested)
			record, err := k.vestingKeeper.FalsifyClaim(ctx, dispute.ClaimId, dispute.Challenger)
			if err != nil {
				k.Logger(ctx).Error("failed to execute clawback", "claim_id", dispute.ClaimId, "error", err)
			} else {
				ctx.EventManager().EmitEvent(sdk.NewEvent("vesting_clawback",
					sdk.NewAttribute("dispute_id", dispute.Id),
					sdk.NewAttribute("claim_id", dispute.ClaimId),
					sdk.NewAttribute("clawback_id", record.Id),
					sdk.NewAttribute("released_clawback", record.ReleasedClawback),
					sdk.NewAttribute("unvested_forfeited", record.UnvestedForfeited),
					sdk.NewAttribute("challenger_reward", record.ChallengerReward),
				))
			}
		case types.OutcomeInconclusive:
			// Uncertainty — resume vesting (benefit of the doubt)
			if err := k.vestingKeeper.ResumeVestingByClaimId(ctx, dispute.ClaimId); err != nil {
				k.Logger(ctx).Error("failed to resume vesting", "claim_id", dispute.ClaimId, "error", err)
			}
		}
	}

	k.SetDispute(ctx, dispute)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dispute_resolved",
			sdk.NewAttribute("dispute_id", dispute.Id),
			sdk.NewAttribute("fact_id", dispute.FactId),
			sdk.NewAttribute("outcome", string(outcome)),
			sdk.NewAttribute("tier", fmt.Sprintf("%d", dispute.Tier)),
		),
	)

	k.Logger(ctx).Info("dispute resolved",
		"dispute_id", dispute.Id,
		"outcome", outcome,
	)

	// Post-resolution processing: cross-module notifications and cascade effects
	k.processPostResolution(ctx, dispute, outcome)

	return nil
}

// EscalateDispute moves a dispute to the next escalation tier.
func (k Keeper) EscalateDispute(ctx sdk.Context, disputeId, requester, additionalBond string) error {
	dispute, found := k.GetDispute(ctx, disputeId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrDisputeNotFound, disputeId)
	}

	if dispute.Status == string(types.DisputeStatusResolved) || dispute.Status == string(types.DisputeStatusExpired) {
		return fmt.Errorf("%w: %s", types.ErrDisputeAlreadyResolved, disputeId)
	}

	// Check max tier
	if dispute.Tier >= uint32(types.TierFinal) {
		return fmt.Errorf("%w", types.ErrMaxEscalation)
	}

	// Check escalation delay
	params := k.GetParams(ctx)
	currentHeight := uint64(ctx.BlockHeight())
	if currentHeight < dispute.CreatedAt+params.EscalationDelay {
		return fmt.Errorf("%w: must wait until height %d",
			types.ErrEscalationTooEarly, dispute.CreatedAt+params.EscalationDelay)
	}

	// Get next tier config
	nextTier := types.EscalationTier(dispute.Tier + 1)
	tierConfig, err := k.GetTierConfig(params, nextTier)
	if err != nil {
		return err
	}

	// Validate additional bond
	additionalAmount := new(big.Int)
	if _, ok := additionalAmount.SetString(additionalBond, 10); !ok || additionalAmount.Sign() <= 0 {
		return fmt.Errorf("%w: invalid additional bond", types.ErrInvalidBond)
	}

	// Lock additional bond from requester
	requesterAddr, err := sdk.AccAddressFromBech32(requester)
	if err != nil {
		return fmt.Errorf("invalid requester address: %w", err)
	}
	bondCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(additionalAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, requesterAddr, types.ModuleName, sdk.NewCoins(bondCoin),
	); err != nil {
		return fmt.Errorf("%w: %v", types.ErrInsufficientBond, err)
	}

	// Update bond total
	existingBond := new(big.Int)
	existingBond.SetString(dispute.Bond, 10)
	newBond := new(big.Int).Add(existingBond, additionalAmount)

	// Select new arbiters for the escalated tier
	newArbiters, err := k.SelectArbiters(ctx, tierConfig.ArbiterCount, dispute.Challenger, dispute.Defender)
	if err != nil {
		return err
	}

	// Recalculate deadlines from current height
	evidenceDeadline := currentHeight + tierConfig.EvidencePeriod
	votingDeadline := evidenceDeadline + tierConfig.VotingPeriod

	dispute.Tier = uint32(nextTier)
	dispute.Status = string(types.DisputeStatusEvidence)
	dispute.Bond = newBond.String()
	dispute.Arbiters = newArbiters
	dispute.EvidenceDeadline = evidenceDeadline
	dispute.VotingDeadline = votingDeadline

	k.SetDispute(ctx, dispute)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dispute_escalated",
			sdk.NewAttribute("dispute_id", dispute.Id),
			sdk.NewAttribute("new_tier", fmt.Sprintf("%d", nextTier)),
			sdk.NewAttribute("requester", requester),
			sdk.NewAttribute("additional_bond", additionalBond),
			sdk.NewAttribute("new_arbiter_count", fmt.Sprintf("%d", len(newArbiters))),
		),
	)

	k.Logger(ctx).Info("dispute escalated",
		"dispute_id", dispute.Id,
		"new_tier", nextTier,
		"requester", requester,
	)

	return nil
}

// ---------- Confidence Adjustment ----------

// Confidence adjustment constants (from TS CONFIDENCE_ADJUSTMENT_CONFIG).
// All values in basis points (1_000_000 = 100%).
const (
	// ConfidenceReductionOverturned: confidence multiplied by (1 - 0.55) = 0.45 when challenger wins.
	ConfidenceReductionBps uint64 = 550_000

	// ConfidenceBoostUpheld: confidence multiplied by (1 + 0.11) = 1.11 when defender wins.
	ConfidenceBoostBps uint64 = 110_000

	// RetractionThreshold: below this confidence, fact is revoked (10%).
	RetractionThresholdBps uint64 = 100_000

	// MaxConfidence: confidence cannot exceed this (100%).
	MaxConfidenceBps uint64 = 1_000_000
)

// resolveFactOutcome adjusts a fact's confidence and status based on dispute outcome.
// Delegates to the knowledge module's ResolveFactDispute method, which owns the
// invariants for fact state transitions (confidence adjustment, retraction, events).
func (k Keeper) resolveFactOutcome(ctx sdk.Context, factId string, outcome string, tier types.EscalationTier) {
	// Compute tier-scaled reduction for overturned outcomes
	reductionBps := ConfidenceReductionBps
	if outcome == types.OutcomeOverturned {
		tierMultiplier := uint64(tier)
		if tierMultiplier > 1 {
			reductionBps = ConfidenceReductionBps + (tierMultiplier-1)*100_000
			if reductionBps > 900_000 {
				reductionBps = 900_000
			}
		}
	}

	_, err := k.knowledgeKeeper.ResolveFactDispute(ctx, factId, string(outcome), reductionBps)
	if err != nil {
		k.Logger(ctx).Error("failed to resolve fact dispute outcome",
			"fact_id", factId, "outcome", outcome, "error", err)
	}
}

// ---------- Post-Resolution Processing ----------

// processPostResolution handles cross-module notifications and cascading effects
// after a dispute has been resolved. This is the extension point for:
// 1. Confidence cascade to dependent facts (overturned → reduce referencing facts)
// 2. Structured event emission for other modules (evidence_mgmt, reputation, etc.)
func (k Keeper) processPostResolution(ctx sdk.Context, dispute types.Dispute, outcome string) {
	// 1. Cascade confidence reduction to facts referencing a retracted fact
	if outcome == types.OutcomeOverturned {
		k.cascadeConfidenceReduction(ctx, dispute.FactId)
	}

	// 2. Emit structured notification event for cross-module consumption
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dispute_resolution_notification",
			sdk.NewAttribute("dispute_id", dispute.Id),
			sdk.NewAttribute("fact_id", dispute.FactId),
			sdk.NewAttribute("claim_id", dispute.ClaimId),
			sdk.NewAttribute("outcome", string(outcome)),
			sdk.NewAttribute("tier", fmt.Sprintf("%d", dispute.Tier)),
			sdk.NewAttribute("challenger", dispute.Challenger),
			sdk.NewAttribute("defender", dispute.Defender),
		),
	)
}

// cascadeConfidenceReduction reduces confidence of facts that reference a retracted fact.
// Delegates to the knowledge module's CascadeFactConfidence method.
func (k Keeper) cascadeConfidenceReduction(ctx sdk.Context, retractedFactId string) {
	const cascadeReductionBps uint64 = 220_000 // 22%

	_, err := k.knowledgeKeeper.CascadeFactConfidence(ctx, retractedFactId, cascadeReductionBps)
	if err != nil {
		k.Logger(ctx).Error("failed to cascade confidence reduction",
			"retracted_fact", retractedFactId, "error", err)
	}
}

// ---------- Internal helpers ----------

// distributeToAddress sends coins from the module account to a recipient.
func (k Keeper) distributeToAddress(ctx sdk.Context, recipient string, amount *big.Int) {
	if amount.Sign() <= 0 {
		return
	}
	addr, err := sdk.AccAddressFromBech32(recipient)
	if err != nil {
		k.Logger(ctx).Error("invalid recipient address in distribution", "address", recipient, "error", err)
		return
	}
	coin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amount))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(coin)); err != nil {
		k.Logger(ctx).Error("failed to distribute funds", "recipient", recipient, "amount", amount.String(), "error", err)
	}
}

// burnFromModule burns coins held by the module.
func (k Keeper) burnFromModule(ctx sdk.Context, amount *big.Int) {
	if amount.Sign() <= 0 {
		return
	}
	coin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amount))
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		k.Logger(ctx).Error("failed to burn coins", "amount", amount.String(), "error", err)
	}
}

