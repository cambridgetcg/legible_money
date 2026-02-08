package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// AdvanceRoundPhases is called by BeginBlocker to advance verification rounds
// through their lifecycle phases based on the current block height.
//
// Phase transitions:
//   commit  → reveal       (at CommitDeadlineHeight)
//   reveal  → aggregation  (at RevealDeadlineHeight)
//   aggregation → complete (at RevealDeadlineHeight + AggregationPhaseDuration)
func (k Keeper) AdvanceRoundPhases(ctx sdk.Context, currentHeight uint64) error {
	params := k.GetParams(ctx)
	activeRounds := k.GetActiveRounds(ctx)

	for _, round := range activeRounds {
		expectedPhase := GetRoundPhase(round, currentHeight, params)

		if expectedPhase == round.Phase {
			continue
		}

		switch expectedPhase {
		case "reveal":
			if err := k.transitionToReveal(ctx, round); err != nil {
				k.Logger(ctx).Error("failed to transition to reveal",
					"round_id", round.Id, "error", err)
			}

		case "aggregation":
			if err := k.transitionToAggregation(ctx, round); err != nil {
				k.Logger(ctx).Error("failed to transition to aggregation",
					"round_id", round.Id, "error", err)
			}

		case "expired":
			if err := k.handleExpiredRound(ctx, round); err != nil {
				k.Logger(ctx).Error("failed to handle expired round",
					"round_id", round.Id, "error", err)
			}
		}
	}

	return nil
}

// transitionToReveal marks a round as entering the reveal phase.
// Validators who committed can now reveal their verdicts.
func (k Keeper) transitionToReveal(ctx sdk.Context, round types.VerificationRound) error {
	if round.Phase != "commit" {
		return fmt.Errorf("cannot transition to reveal from %s phase", round.Phase)
	}

	round.Phase = "reveal"
	k.SetVerificationRound(ctx, round)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("round_phase_changed",
			sdk.NewAttribute("round_id", round.Id),
			sdk.NewAttribute("new_phase", "reveal"),
			sdk.NewAttribute("commitments", fmt.Sprintf("%d", len(round.Commitments))),
		),
	)

	k.Logger(ctx).Info("round transitioned to reveal phase",
		"round_id", round.Id,
		"commitments", len(round.Commitments),
	)

	return nil
}

// transitionToAggregation marks a round as entering the aggregation phase.
// No more reveals accepted — the system aggregates results.
func (k Keeper) transitionToAggregation(ctx sdk.Context, round types.VerificationRound) error {
	if round.Phase != "reveal" {
		return fmt.Errorf("cannot transition to aggregation from %s phase", round.Phase)
	}

	params := k.GetParams(ctx)

	// Check if we have enough reveals for quorum
	if uint32(len(round.Reveals)) < params.MinVerifiers {
		// Insufficient participation — mark as insufficient
		result := types.VerificationResult{
			Verdict:          "insufficient",
			Confidence:       0,
			ParticipantCount: uint32(len(round.Reveals)),
		}

		return k.CompleteRound(ctx, round, result)
	}

	round.Phase = "aggregation"
	k.SetVerificationRound(ctx, round)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("round_phase_changed",
			sdk.NewAttribute("round_id", round.Id),
			sdk.NewAttribute("new_phase", "aggregation"),
			sdk.NewAttribute("reveals", fmt.Sprintf("%d", len(round.Reveals))),
		),
	)

	k.Logger(ctx).Info("round transitioned to aggregation phase",
		"round_id", round.Id,
		"reveals", len(round.Reveals),
	)

	return nil
}

// handleExpiredRound handles a round that has passed all deadlines.
// If still in aggregation, perform final aggregation. Otherwise mark as expired.
func (k Keeper) handleExpiredRound(ctx sdk.Context, round types.VerificationRound) error {
	switch round.Phase {
	case "commit":
		// No commitments received or deadline expired during commit
		result := types.VerificationResult{
			Verdict:          "insufficient",
			Confidence:       0,
			ParticipantCount: 0,
		}
		return k.CompleteRound(ctx, round, result)

	case "reveal":
		// Some commits but reveals haven't been collected
		// Slash committed-but-not-revealed validators
		k.slashMissedReveals(ctx, round)

		if uint32(len(round.Reveals)) == 0 {
			result := types.VerificationResult{
				Verdict:          "insufficient",
				Confidence:       0,
				ParticipantCount: 0,
			}
			return k.CompleteRound(ctx, round, result)
		}

		// Aggregate whatever reveals we have
		return k.performAggregation(ctx, round)

	case "aggregation":
		// Aggregation deadline passed — perform final aggregation
		return k.performAggregation(ctx, round)

	default:
		// Already complete or unknown phase
		return nil
	}
}

// performAggregation runs the stake-weighted confidence aggregation engine
// and completes the round. Delegates to AggregateVerificationResult in confidence.go.
func (k Keeper) performAggregation(ctx sdk.Context, round types.VerificationRound) error {
	result, err := k.AggregateVerificationResult(ctx, round)
	if err != nil {
		return fmt.Errorf("aggregation failed for round %s: %w", round.Id, err)
	}

	// Apply rewards and slashes
	if err := k.ApplyRewardsAndSlashes(ctx, result); err != nil {
		k.Logger(ctx).Error("failed to apply rewards/slashes",
			"round_id", round.Id, "error", err)
	}

	return k.CompleteRound(ctx, round, result)
}

// slashMissedReveals slashes validators who committed but failed to reveal.
func (k Keeper) slashMissedReveals(ctx sdk.Context, round types.VerificationRound) {
	for validator := range round.Commitments {
		if _, revealed := round.Reveals[validator]; !revealed {
			// Slash for missed reveal
			if err := k.SlashMissedVerification(ctx, validator, "1000000"); err != nil {
				k.Logger(ctx).Error("failed to slash missed reveal",
					"validator", validator, "error", err)
			}

			ctx.EventManager().EmitEvent(
				sdk.NewEvent("missed_reveal_slashed",
					sdk.NewAttribute("round_id", round.Id),
					sdk.NewAttribute("validator", validator),
				),
			)
		}
	}
}
