package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// BeginBlockerAdversarial is the adversarial verification BeginBlocker orchestrator.
// When AdversarialVerificationEnabled is true globally, this replaces the legacy
// verification flow. When per-domain overrides exist, both flows run concurrently:
// legacy rounds are advanced for non-adversarial domains, and adversarial evaluation
// runs for adversarial domains.
func (k Keeper) BeginBlockerAdversarial(ctx sdk.Context) error {
	height := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)
	ep := k.GetExtendedParams(ctx)

	hasPerDomainOverrides := len(ep.AdversarialDomainOverrides) > 0

	if !params.AdversarialVerificationEnabled && !hasPerDomainOverrides {
		// Pure legacy mode — use traditional AdvanceRoundPhases
		return k.AdvanceRoundPhases(ctx, height)
	}

	// Always advance legacy verification rounds (handles non-adversarial domains
	// and any rounds created before per-domain overrides were set)
	if err := k.AdvanceRoundPhases(ctx, height); err != nil {
		k.Logger(ctx).Error("failed to advance legacy round phases", "error", err)
	}

	// Adversarial evaluation pipeline (runs for adversarial-mode domains)
	// 1. Evaluate pending claims (structural scoring)
	k.EvaluatePendingClaims(ctx, height)

	// 2. Resolve deterministic challenges (Path A)
	k.ResolvePendingDeterministicChallenges(ctx, height)

	// 3. Advance challenge round phases (Path B: commit→reveal→aggregation→complete)
	//    Only processes rounds with Type == "challenge"
	if err := k.AdvanceChallengeRoundPhases(ctx, height); err != nil {
		k.Logger(ctx).Error("failed to advance challenge round phases", "error", err)
	}

	// 4. Process challenge window expiry → transition provisional to verified
	k.ProcessChallengeWindowExpiry(ctx, height)

	// 5. At epoch boundaries: confidence growth + maturity advancement
	if height%types.ConfidenceGrowthEpoch == 0 {
		k.ApplyConfidenceGrowth(ctx, height)
		k.AdvanceMaturityLevels(ctx, height)
	}

	return nil
}

// AdvanceChallengeRoundPhases advances only challenge-type verification rounds.
// Legacy verification rounds are not processed in adversarial mode.
func (k Keeper) AdvanceChallengeRoundPhases(ctx sdk.Context, currentHeight uint64) error {
	params := k.GetParams(ctx)
	activeRounds := k.GetActiveRounds(ctx)

	for _, round := range activeRounds {
		// Only process challenge rounds in adversarial mode
		if round.Type != types.RoundTypeChallenge {
			continue
		}

		expectedPhase := GetRoundPhase(round, currentHeight, params)
		if expectedPhase == round.Phase {
			continue
		}

		switch expectedPhase {
		case "reveal":
			if err := k.transitionToReveal(ctx, round); err != nil {
				k.Logger(ctx).Error("failed to transition challenge round to reveal",
					"round_id", round.Id, "error", err)
			}
		case "aggregation":
			if err := k.transitionToAggregation(ctx, round); err != nil {
				k.Logger(ctx).Error("failed to transition challenge round to aggregation",
					"round_id", round.Id, "error", err)
			}
		case "expired":
			if err := k.handleExpiredRound(ctx, round); err != nil {
				k.Logger(ctx).Error("failed to handle expired challenge round",
					"round_id", round.Id, "error", err)
			}
		}
	}

	return nil
}

// AdvanceRoundPhases is called by BeginBlocker to advance verification rounds
// through their lifecycle phases based on the current block height.
// In legacy mode, processes all rounds. In adversarial mode, only non-challenge rounds.
//
// Phase transitions:
//
//	commit  → reveal       (at CommitDeadlineHeight)
//	reveal  → aggregation  (at RevealDeadlineHeight)
//	aggregation → complete (at RevealDeadlineHeight + AggregationPhaseDuration)
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
		// Reveal deadline passed — aggregate whatever reveals we have.
		// Missed-reveal slashing is handled by calculateRewardsAndSlashes
		// in confidence.go (2% proportional to stake), so no separate slash here.
		if uint32(len(round.Reveals)) == 0 {
			result := types.VerificationResult{
				Verdict:          "insufficient",
				Confidence:       0,
				ParticipantCount: 0,
			}
			return k.CompleteRound(ctx, round, result)
		}

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

	// P0-2: Determine if this is a contested round for Guardian tier tracking
	contested := round.Type == types.RoundTypeChallenge || round.Type == types.RoundTypeContradiction

	// Apply rewards and slashes
	if err := k.ApplyRewardsAndSlashes(ctx, result, contested); err != nil {
		k.Logger(ctx).Error("failed to apply rewards/slashes",
			"round_id", round.Id, "error", err)
	}

	return k.CompleteRound(ctx, round, result)
}
