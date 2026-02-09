package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// CreateVerificationRound creates a new verification round for a claim.
// Called after a claim is submitted via MsgSubmitClaim.
func (k Keeper) CreateVerificationRound(ctx sdk.Context, claim types.Claim) (types.VerificationRound, error) {
	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())

	roundID := GenerateRoundID(claim.Id, height)

	round := types.VerificationRound{
		Id:                   roundID,
		ClaimId:              claim.Id,
		Phase:                "commit",
		StartHeight:          height,
		CommitDeadlineHeight: height + params.CommitPhaseDuration,
		RevealDeadlineHeight: height + params.CommitPhaseDuration + params.RevealPhaseDuration,
		SelectedValidators:   []string{}, // Populated via VRF in ExtendVote
		Commitments:          make(map[string]types.VerificationCommitment),
		Reveals:              make(map[string]types.VerificationReveal),
	}

	k.SetVerificationRound(ctx, round)

	k.Logger(ctx).Info("verification round created",
		"round_id", roundID,
		"claim_id", claim.Id,
		"start_height", height,
		"commit_deadline", round.CommitDeadlineHeight,
		"reveal_deadline", round.RevealDeadlineHeight,
	)

	return round, nil
}

// GetActiveRoundsForHeight returns active rounds relevant at a given block height.
// This includes rounds in commit, reveal, or aggregation phase.
func (k Keeper) GetActiveRoundsForHeight(ctx sdk.Context, height uint64) []types.VerificationRound {
	var rounds []types.VerificationRound
	k.IterateVerificationRounds(ctx, func(r types.VerificationRound) bool {
		if r.Phase == "complete" {
			return false
		}
		// Include rounds that started at or before this height
		if r.StartHeight <= height {
			rounds = append(rounds, r)
		}
		return false
	})
	return rounds
}

// GetRoundsInPhase returns all rounds currently in a specific phase.
func (k Keeper) GetRoundsInPhase(ctx sdk.Context, phase string) []types.VerificationRound {
	var rounds []types.VerificationRound
	k.IterateVerificationRounds(ctx, func(r types.VerificationRound) bool {
		if r.Phase == phase {
			rounds = append(rounds, r)
		}
		return false
	})
	return rounds
}

// GetRoundPhase computes what phase a round should be in at a given height.
// This is a pure function that doesn't modify state.
func GetRoundPhase(round types.VerificationRound, height uint64, params types.Params) string {
	if round.Phase == "complete" {
		return "complete"
	}

	if height <= round.CommitDeadlineHeight {
		return "commit"
	}

	if height <= round.RevealDeadlineHeight {
		return "reveal"
	}

	aggregationDeadline := round.RevealDeadlineHeight + params.AggregationPhaseDuration
	if height <= aggregationDeadline {
		return "aggregation"
	}

	return "expired"
}

// StoreCommitmentInRound adds a commitment to a round and persists.
func (k Keeper) StoreCommitmentInRound(
	ctx sdk.Context,
	roundID string,
	commitment types.VerificationCommitment,
) error {
	round, found := k.GetVerificationRound(ctx, roundID)
	if !found {
		return fmt.Errorf("%w: round %s", types.ErrRoundNotFound, roundID)
	}

	if round.Phase != "commit" {
		return fmt.Errorf("%w: round is in %s phase", types.ErrWrongPhase, round.Phase)
	}

	if _, exists := round.Commitments[commitment.Validator]; exists {
		return fmt.Errorf("%w: %s", types.ErrDuplicateCommitment, commitment.Validator)
	}

	round.Commitments[commitment.Validator] = commitment

	// Track this validator as selected (via VRF proof in commitment)
	alreadySelected := false
	for _, v := range round.SelectedValidators {
		if v == commitment.Validator {
			alreadySelected = true
			break
		}
	}
	if !alreadySelected {
		round.SelectedValidators = append(round.SelectedValidators, commitment.Validator)
	}

	k.SetVerificationRound(ctx, round)
	return nil
}

// GetCommitmentsForRound returns all commitments in a round.
func (k Keeper) GetCommitmentsForRound(ctx sdk.Context, roundID string) ([]types.VerificationCommitment, error) {
	round, found := k.GetVerificationRound(ctx, roundID)
	if !found {
		return nil, fmt.Errorf("%w: round %s", types.ErrRoundNotFound, roundID)
	}

	commitments := make([]types.VerificationCommitment, 0, len(round.Commitments))
	for _, c := range round.Commitments {
		commitments = append(commitments, c)
	}
	return commitments, nil
}

// StoreRevealInRound adds a reveal to a round and persists.
func (k Keeper) StoreRevealInRound(
	ctx sdk.Context,
	roundID string,
	reveal types.VerificationReveal,
) error {
	round, found := k.GetVerificationRound(ctx, roundID)
	if !found {
		return fmt.Errorf("%w: round %s", types.ErrRoundNotFound, roundID)
	}

	if round.Phase != "reveal" {
		return fmt.Errorf("%w: round is in %s phase", types.ErrWrongPhase, round.Phase)
	}

	// Verify the validator committed
	if _, hasCommitment := round.Commitments[reveal.Validator]; !hasCommitment {
		return fmt.Errorf("%w: %s", types.ErrNoCommitment, reveal.Validator)
	}

	if _, exists := round.Reveals[reveal.Validator]; exists {
		return fmt.Errorf("%w: %s", types.ErrDuplicateReveal, reveal.Validator)
	}

	round.Reveals[reveal.Validator] = reveal
	k.SetVerificationRound(ctx, round)
	return nil
}

// GetRevealsForRound returns all reveals in a round.
func (k Keeper) GetRevealsForRound(ctx sdk.Context, roundID string) ([]types.VerificationReveal, error) {
	round, found := k.GetVerificationRound(ctx, roundID)
	if !found {
		return nil, fmt.Errorf("%w: round %s", types.ErrRoundNotFound, roundID)
	}

	reveals := make([]types.VerificationReveal, 0, len(round.Reveals))
	for _, r := range round.Reveals {
		reveals = append(reveals, r)
	}
	return reveals, nil
}

// CompleteRound marks a round as complete with the given result.
// If accepted: creates a fact, returns claim stake, creates vesting schedule.
// If rejected: slashes a portion of claim stake (InvalidClaimSlashBps), returns the rest.
// If contested: creates a contested fact, keeps stake locked.
// If insufficient: keeps stake locked (can be reclaimed via new round).
// Distributes rewards and slashes to validators.
func (k Keeper) CompleteRound(
	ctx sdk.Context,
	round types.VerificationRound,
	result types.VerificationResult,
) error {
	round.Phase = "complete"
	round.FinalVerdict = result.Verdict
	round.FinalConfidence = result.Confidence
	round.CompletedAtHeight = uint64(ctx.BlockHeight())

	// Update claim status and handle stake settlement
	claim, claimFound := k.GetClaim(ctx, round.ClaimId)
	if claimFound {
		params := k.GetParams(ctx)

		switch result.Verdict {
		case "accepted":
			claim.Status = "accepted"
			k.SetClaim(ctx, claim)

			// Create fact from accepted claim
			height := uint64(ctx.BlockHeight())
			factID := GenerateFactID(claim.Id, round.Id)
			fact := types.Fact{
				Id:                   factID,
				ClaimText:            claim.ClaimText,
				Domain:               claim.Domain,
				EpistemicCategory:    claim.EpistemicCategory,
				Confidence:           result.Confidence,
				Status:               types.FactStatusVerified,
				Submitter:            claim.Submitter,
				SubmitterDid:         claim.SubmitterDid,
				References:           claim.References,
				VerifiedAtHeight:     height,
				VerifierCount:        result.ParticipantCount,
				LogicZone:            claim.LogicZone,
				CreatedAtHeight:      height,
				LastVerifiedHeight:   height,
				LastConfidenceUpdate: height,
				BaseConfidence:       result.Confidence,
				Maturity:             types.MaturityEmerging,
			}

			// Transfer evidence summaries from claim to fact (evidence provenance survives)
			if k.evidenceMgmtKeeper != nil {
				evidences := k.evidenceMgmtKeeper.GetEvidenceByClaimId(ctx, claim.Id)
				summaries := make([]types.FactEvidenceSummary, 0, len(evidences))
				for _, ev := range evidences {
					if ev.Status == "rejected" || ev.Status == "expired" {
						continue
					}
					typeStrength, ok := types.EVIDENCE_TYPE_STRENGTH[ev.EvidenceType]
					if !ok {
						typeStrength = 400_000
					}
					summaries = append(summaries, types.FactEvidenceSummary{
						EvidenceId:   ev.Id,
						EvidenceType: ev.EvidenceType,
						Reliability:  ev.Reliability,
						TypeStrength: typeStrength,
					})
				}
				fact.EvidenceSummaries = summaries
				fact.EvidenceBonus = result.EvidenceBonus
			}

			k.SetFact(ctx, fact)

			// Return staked amount to submitter (G-1 fix)
			k.returnClaimStake(ctx, claim)

			// Create truth-linked vesting schedule (G-5 fix)
			if k.vestingRewardsKeeper != nil && claim.StakeAmount != "" {
				if err := k.vestingRewardsKeeper.CreateVestingScheduleFromKnowledge(
					ctx, claim.Id, factID, claim.Submitter, claim.StakeAmount, claim.EpistemicCategory,
				); err != nil {
					k.Logger(ctx).Error("failed to create vesting schedule",
						"claim_id", claim.Id, "fact_id", factID, "error", err)
				}
			}

			// Create fact references from claim references
			for _, refID := range claim.References {
				k.SetFactReference(ctx, types.FactReference{
					FromFactId:      factID,
					ToFactId:        refID,
					ReferenceType:   "supports",
					Weight:          1_000_000,
					CreatedAtHeight: height,
				})
				// Increment citation count on referenced facts
				if err := k.IncrementCitationCount(ctx, refID); err != nil {
					k.Logger(ctx).Error("failed to increment citation count",
						"fact_id", refID, "error", err)
				}
			}

			// Distribute citation rewards (15% of verification rewards to cited facts)
			if len(claim.References) > 0 {
				totalReward := uint64(0)
				for _, r := range result.Rewards {
					totalReward += r
				}
				if totalReward > 0 {
					k.DistributeCitationRewards(ctx, factID, totalReward)
				}
			}

			ctx.EventManager().EmitEvent(
				sdk.NewEvent("fact_created",
					sdk.NewAttribute("fact_id", factID),
					sdk.NewAttribute("claim_id", claim.Id),
					sdk.NewAttribute("confidence", fmt.Sprintf("%d", result.Confidence)),
					sdk.NewAttribute("verifiers", fmt.Sprintf("%d", result.ParticipantCount)),
				),
			)

		case "rejected":
			claim.Status = "rejected"
			k.SetClaim(ctx, claim)

			// Slash a portion of stake, return the rest (G-2 fix)
			k.slashAndReturnClaimStake(ctx, claim, params.InvalidClaimSlashBps)

		case "contested":
			claim.Status = "contested"
			k.SetClaim(ctx, claim)

			// Create contested fact (G-7 fix) — lower confidence, tracks dispute
			height := uint64(ctx.BlockHeight())
			factID := GenerateFactID(claim.Id, round.Id)
			fact := types.Fact{
				Id:                   factID,
				ClaimText:            claim.ClaimText,
				Domain:               claim.Domain,
				EpistemicCategory:    claim.EpistemicCategory,
				Confidence:           result.Confidence,
				Status:               types.FactStatusContested,
				Submitter:            claim.Submitter,
				SubmitterDid:         claim.SubmitterDid,
				References:           claim.References,
				VerifiedAtHeight:     height,
				VerifierCount:        result.ParticipantCount,
				LogicZone:            claim.LogicZone,
				CreatedAtHeight:      height,
				LastVerifiedHeight:   height,
				LastConfidenceUpdate: height,
				BaseConfidence:       result.Confidence,
				Maturity:             types.MaturityEmerging,
			}
			k.SetFact(ctx, fact)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent("contested_fact_created",
					sdk.NewAttribute("fact_id", factID),
					sdk.NewAttribute("claim_id", claim.Id),
					sdk.NewAttribute("confidence", fmt.Sprintf("%d", result.Confidence)),
				),
			)

			// Stake remains locked for contested claims (may resolve via challenge/reverification)

		case "insufficient":
			claim.Status = "insufficient"
			k.SetClaim(ctx, claim)

			// Return stake on insufficient participation — not the submitter's fault
			k.returnClaimStake(ctx, claim)
		}
	}

	k.SetVerificationRound(ctx, round)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("round_completed",
			sdk.NewAttribute("round_id", round.Id),
			sdk.NewAttribute("claim_id", round.ClaimId),
			sdk.NewAttribute("verdict", result.Verdict),
			sdk.NewAttribute("confidence", fmt.Sprintf("%d", result.Confidence)),
			sdk.NewAttribute("participants", fmt.Sprintf("%d", result.ParticipantCount)),
		),
	)

	k.Logger(ctx).Info("verification round completed",
		"round_id", round.Id,
		"claim_id", round.ClaimId,
		"verdict", result.Verdict,
		"confidence", result.Confidence,
	)

	return nil
}

// returnClaimStake returns the full staked amount from the knowledge module back to the claim submitter.
func (k Keeper) returnClaimStake(ctx sdk.Context, claim types.Claim) {
	if k.bankKeeper == nil || claim.StakeAmount == "" {
		return
	}

	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(claim.StakeAmount, 10); !ok || stakeAmount.Sign() <= 0 {
		return
	}

	submitterAddr, err := sdk.AccAddressFromBech32(claim.Submitter)
	if err != nil {
		k.Logger(ctx).Error("invalid submitter address for stake return",
			"submitter", claim.Submitter, "error", err)
		return
	}

	returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount)))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, submitterAddr, returnCoins,
	); err != nil {
		k.Logger(ctx).Error("failed to return claim stake",
			"claim_id", claim.Id, "submitter", claim.Submitter, "amount", claim.StakeAmount, "error", err)
		return
	}

	// Credit back to period overlay
	if k.tokensKeeper != nil {
		if err := k.tokensKeeper.CreditBalance(ctx, claim.Submitter, claim.StakeAmount, uint64(ctx.BlockHeight())); err != nil {
			k.Logger(ctx).Error("failed to credit period balance for stake return",
				"submitter", claim.Submitter, "error", err)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("claim_stake_returned",
			sdk.NewAttribute("claim_id", claim.Id),
			sdk.NewAttribute("submitter", claim.Submitter),
			sdk.NewAttribute("amount", claim.StakeAmount),
		),
	)
}

// slashAndReturnClaimStake slashes a portion of the claim stake and returns the remainder.
// slashBps is in basis points (1_000_000 = 100%). E.g., 220000 = 22%.
func (k Keeper) slashAndReturnClaimStake(ctx sdk.Context, claim types.Claim, slashBps uint64) {
	if k.bankKeeper == nil || claim.StakeAmount == "" {
		return
	}

	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(claim.StakeAmount, 10); !ok || stakeAmount.Sign() <= 0 {
		return
	}

	// Calculate slash and return amounts
	slashAmount := new(big.Int).Mul(stakeAmount, big.NewInt(int64(slashBps)))
	slashAmount.Div(slashAmount, big.NewInt(1_000_000))

	returnAmount := new(big.Int).Sub(stakeAmount, slashAmount)

	submitterAddr, err := sdk.AccAddressFromBech32(claim.Submitter)
	if err != nil {
		k.Logger(ctx).Error("invalid submitter address for slash",
			"submitter", claim.Submitter, "error", err)
		return
	}

	// Return the non-slashed portion to submitter
	if returnAmount.Sign() > 0 {
		returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(returnAmount)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, submitterAddr, returnCoins,
		); err != nil {
			k.Logger(ctx).Error("failed to return remaining stake",
				"claim_id", claim.Id, "amount", returnAmount.String(), "error", err)
		}

		// Credit back to period overlay
		if k.tokensKeeper != nil {
			if err := k.tokensKeeper.CreditBalance(ctx, claim.Submitter, returnAmount.String(), uint64(ctx.BlockHeight())); err != nil {
				k.Logger(ctx).Error("failed to credit period balance for partial return",
					"submitter", claim.Submitter, "error", err)
			}
		}
	}

	// Burn the slashed portion (deflationary — TS chain routes to BURN_DID)
	if slashAmount.Sign() > 0 && k.bankKeeper != nil {
		burnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(slashAmount)))
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
			k.Logger(ctx).Error("failed to burn slashed stake",
				"claim_id", claim.Id, "amount", slashAmount.String(), "error", err)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("claim_stake_slashed",
			sdk.NewAttribute("claim_id", claim.Id),
			sdk.NewAttribute("submitter", claim.Submitter),
			sdk.NewAttribute("total_stake", claim.StakeAmount),
			sdk.NewAttribute("slashed", slashAmount.String()),
			sdk.NewAttribute("returned", returnAmount.String()),
			sdk.NewAttribute("slash_bps", fmt.Sprintf("%d", slashBps)),
		),
	)
}

// CleanupExpiredRounds removes rounds that completed more than retentionBlocks ago.
func (k Keeper) CleanupExpiredRounds(ctx sdk.Context, retentionBlocks uint64) {
	currentHeight := uint64(ctx.BlockHeight())
	if currentHeight < retentionBlocks {
		return
	}
	cutoff := currentHeight - retentionBlocks

	var toDelete []string
	k.IterateVerificationRounds(ctx, func(r types.VerificationRound) bool {
		if r.Phase == "complete" && r.CompletedAtHeight < cutoff {
			toDelete = append(toDelete, r.Id)
		}
		return false
	})

	for _, id := range toDelete {
		k.DeleteVerificationRound(ctx, id)
	}

	if len(toDelete) > 0 {
		k.Logger(ctx).Info("cleaned up expired rounds", "count", len(toDelete))
	}
}
