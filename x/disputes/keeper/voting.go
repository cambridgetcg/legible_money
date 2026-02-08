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
	if dispute.Status != types.DisputeStatusVoting {
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
func (k Keeper) TallyVotes(ctx sdk.Context, dispute types.Dispute) (types.DisputeOutcome, error) {
	params := k.GetParams(ctx)
	tierConfig, err := k.GetTierConfig(params, dispute.Tier)
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
func (k Keeper) ResolveDispute(ctx sdk.Context, dispute types.Dispute, outcome types.DisputeOutcome) error {
	if dispute.Status == types.DisputeStatusResolved {
		return fmt.Errorf("%w: %s", types.ErrDisputeAlreadyResolved, dispute.Id)
	}

	params := k.GetParams(ctx)
	dispute.Status = types.DisputeStatusResolved
	dispute.Outcome = outcome

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
		// Update fact status
		k.updateFactStatus(ctx, dispute.FactId, "contested")

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

	// Distribute arbiter rewards (split evenly)
	if arbiterReward.Sign() > 0 && len(dispute.Arbiters) > 0 && outcome != types.OutcomeInconclusive {
		perArbiter := new(big.Int).Div(arbiterReward, big.NewInt(int64(len(dispute.Arbiters))))
		if perArbiter.Sign() > 0 {
			for _, arbiter := range dispute.Arbiters {
				k.distributeToAddress(ctx, arbiter, perArbiter)
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

	return nil
}

// EscalateDispute moves a dispute to the next escalation tier.
func (k Keeper) EscalateDispute(ctx sdk.Context, disputeId, requester, additionalBond string) error {
	dispute, found := k.GetDispute(ctx, disputeId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrDisputeNotFound, disputeId)
	}

	if dispute.Status == types.DisputeStatusResolved || dispute.Status == types.DisputeStatusExpired {
		return fmt.Errorf("%w: %s", types.ErrDisputeAlreadyResolved, disputeId)
	}

	// Check max tier
	if dispute.Tier >= types.TierFinal {
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
	nextTier := types.EscalationTier(uint32(dispute.Tier) + 1)
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

	dispute.Tier = nextTier
	dispute.Status = types.DisputeStatusEvidence
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

// updateFactStatus updates a fact's status via the knowledge keeper.
func (k Keeper) updateFactStatus(ctx sdk.Context, factId, newStatus string) {
	fact, found := k.knowledgeKeeper.GetFact(ctx, factId)
	if !found {
		return
	}
	fact.Status = newStatus
	k.knowledgeKeeper.SetFact(ctx, fact)
}
