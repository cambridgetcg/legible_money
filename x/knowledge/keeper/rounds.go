package keeper

import (
	"fmt"

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
// If accepted, creates a fact from the claim. Distributes rewards and slashes.
func (k Keeper) CompleteRound(
	ctx sdk.Context,
	round types.VerificationRound,
	result types.VerificationResult,
) error {
	round.Phase = "complete"
	round.FinalVerdict = result.Verdict
	round.FinalConfidence = result.Confidence
	round.CompletedAtHeight = uint64(ctx.BlockHeight())

	// Update claim status
	claim, claimFound := k.GetClaim(ctx, round.ClaimId)
	if claimFound {
		switch result.Verdict {
		case "accepted":
			claim.Status = "accepted"
			k.SetClaim(ctx, claim)

			// Create fact from accepted claim
			factID := GenerateFactID(claim.Id, round.Id)
			fact := types.Fact{
				Id:                 factID,
				ClaimText:          claim.ClaimText,
				Domain:             claim.Domain,
				EpistemicCategory:  claim.EpistemicCategory,
				Confidence:         result.Confidence,
				Status:             "verified",
				Submitter:          claim.Submitter,
				SubmitterDid:       claim.SubmitterDid,
				References:         claim.References,
				VerifiedAtHeight:   uint64(ctx.BlockHeight()),
				VerifierCount:      result.ParticipantCount,
				LogicZone:          claim.LogicZone,
				CreatedAtHeight:    uint64(ctx.BlockHeight()),
				LastVerifiedHeight: uint64(ctx.BlockHeight()),
			}
			k.SetFact(ctx, fact)

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

		case "contested":
			claim.Status = "contested"
			k.SetClaim(ctx, claim)

		case "insufficient":
			claim.Status = "insufficient"
			k.SetClaim(ctx, claim)
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
