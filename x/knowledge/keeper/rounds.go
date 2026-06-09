package keeper

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// CreateVerificationRound creates a new verification round for a claim.
// Called after a claim is submitted via MsgSubmitClaim.
func (k Keeper) CreateVerificationRound(ctx sdk.Context, claim *types.Claim) (*types.VerificationRound, error) {
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
		Commitments:          make(map[string]*types.VerificationCommitment),
		Reveals:              make(map[string]*types.VerificationReveal),
	}

	k.SetVerificationRound(ctx, &round)

	k.Logger(ctx).Info("verification round created",
		"round_id", roundID,
		"claim_id", claim.Id,
		"start_height", height,
		"commit_deadline", round.CommitDeadlineHeight,
		"reveal_deadline", round.RevealDeadlineHeight,
	)

	return &round, nil
}

// GetActiveRoundsForHeight returns active rounds relevant at a given block height.
// This includes rounds in commit, reveal, or aggregation phase.
func (k Keeper) GetActiveRoundsForHeight(ctx sdk.Context, height uint64) []*types.VerificationRound {
	var rounds []*types.VerificationRound
	k.IterateVerificationRounds(ctx, func(r *types.VerificationRound) bool {
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
func (k Keeper) GetRoundsInPhase(ctx sdk.Context, phase string) []*types.VerificationRound {
	var rounds []*types.VerificationRound
	k.IterateVerificationRounds(ctx, func(r *types.VerificationRound) bool {
		if r.Phase == phase {
			rounds = append(rounds, r)
		}
		return false
	})
	return rounds
}

// GetRoundPhase computes what phase a round should be in at a given height.
// This is a pure function that doesn't modify state.
func GetRoundPhase(round *types.VerificationRound, height uint64, params *types.Params) string {
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
	commitment *types.VerificationCommitment,
) error {
	round, found := k.GetVerificationRound(ctx, roundID)
	if !found {
		return fmt.Errorf("%w: round %s", types.ErrRoundNotFound, roundID)
	}

	if round.Phase != "commit" {
		return fmt.Errorf("%w: round is in %s phase", types.ErrWrongPhase, round.Phase)
	}

	if existing, exists := round.Commitments[commitment.Validator]; exists {
		// Check for equivocation: same validator, different commitment hash
		if existing.CommitmentHash != commitment.CommitmentHash {
			// EQUIVOCATION DETECTED — most severe PoT violation
			k.handleEquivocation(ctx, roundID, commitment.Validator, "commitment",
				existing.CommitmentHash, commitment.CommitmentHash)
			return fmt.Errorf("%w: %s submitted conflicting commitment in round %s",
				types.ErrEquivocation, commitment.Validator, roundID)
		}
		// Same hash = idempotent duplicate, not equivocation
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
func (k Keeper) GetCommitmentsForRound(ctx sdk.Context, roundID string) ([]*types.VerificationCommitment, error) {
	round, found := k.GetVerificationRound(ctx, roundID)
	if !found {
		return nil, fmt.Errorf("%w: round %s", types.ErrRoundNotFound, roundID)
	}

	commitments := make([]*types.VerificationCommitment, 0, len(round.Commitments))
	for _, k := range sortedMapKeys(round.Commitments) {
		commitments = append(commitments, round.Commitments[k])
	}
	return commitments, nil
}

// StoreRevealInRound adds a reveal to a round and persists.
func (k Keeper) StoreRevealInRound(
	ctx sdk.Context,
	roundID string,
	reveal *types.VerificationReveal,
) error {
	round, found := k.GetVerificationRound(ctx, roundID)
	if !found {
		return fmt.Errorf("%w: round %s", types.ErrRoundNotFound, roundID)
	}

	if round.Phase != "reveal" {
		return fmt.Errorf("%w: round is in %s phase", types.ErrWrongPhase, round.Phase)
	}

	// Verify the validator committed
	commitment, hasCommitment := round.Commitments[reveal.Validator]
	if !hasCommitment {
		return fmt.Errorf("%w: %s", types.ErrNoCommitment, reveal.Validator)
	}

	// P0-2 FIX: Verify the reveal data produces the correct commitment hash.
	// Without this check, a malicious proposer could tamper with reveal data
	// (e.g., change verdict from "accept" to "reject") in the VEX injection tx.
	// VerifyVoteExtension checks this for the original extension, but cannot
	// protect against proposer tampering of the injection tx content.
	if !types.VerifyCommitmentHash(commitment.CommitmentHash, roundID, reveal.Verdict, reveal.Confidence, reveal.Salt) {
		return fmt.Errorf("reveal does not match commitment hash for %s in round %s", reveal.Validator, roundID)
	}

	if existing, exists := round.Reveals[reveal.Validator]; exists {
		// Check for equivocation: same validator, different reveal verdict
		if existing.Verdict != reveal.Verdict || existing.Confidence != reveal.Confidence {
			k.handleEquivocation(ctx, roundID, reveal.Validator, "reveal",
				fmt.Sprintf("%s:%d", existing.Verdict, existing.Confidence),
				fmt.Sprintf("%s:%d", reveal.Verdict, reveal.Confidence))
			return fmt.Errorf("%w: %s submitted conflicting reveal in round %s",
				types.ErrEquivocation, reveal.Validator, roundID)
		}
		// Same verdict+confidence = idempotent duplicate
		return fmt.Errorf("%w: %s", types.ErrDuplicateReveal, reveal.Validator)
	}

	round.Reveals[reveal.Validator] = reveal
	k.SetVerificationRound(ctx, round)
	return nil
}

// GetRevealsForRound returns all reveals in a round.
func (k Keeper) GetRevealsForRound(ctx sdk.Context, roundID string) ([]*types.VerificationReveal, error) {
	round, found := k.GetVerificationRound(ctx, roundID)
	if !found {
		return nil, fmt.Errorf("%w: round %s", types.ErrRoundNotFound, roundID)
	}

	reveals := make([]*types.VerificationReveal, 0, len(round.Reveals))
	for _, k := range sortedMapKeys(round.Reveals) {
		reveals = append(reveals, round.Reveals[k])
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
	round *types.VerificationRound,
	result *types.VerificationResult,
) error {
	// Idempotency guard: if this round was already completed (e.g., during state
	// replay or migration), skip all side effects to prevent duplicate fact
	// creation and double stake returns. (PP-CONSENSUS-2 remediation)
	if round.Phase == "complete" {
		k.Logger(ctx).Debug("skipping already-completed round",
			"round_id", round.Id, "claim_id", round.ClaimId)
		return nil
	}

	round.Phase = "complete"
	round.FinalVerdict = result.Verdict
	round.FinalConfidence = result.Confidence
	round.CompletedAtHeight = uint64(ctx.BlockHeight())

	// Handle contradiction rounds — update the counter-fact instead of a claim
	if round.Type == types.RoundTypeContradiction {
		k.completeContradictionRound(ctx, round, result)
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
		return nil
	}

	// Update claim status and handle stake settlement
	claim, claimFound := k.GetClaim(ctx, round.ClaimId)
	if claimFound {
		params := k.GetParams(ctx)

		// AH-5: Reverification round completion — update existing fact, don't create new
		if round.Type == types.RoundTypeReverification && claim.ReverificationFactId != "" {
			switch result.Verdict {
			case "accepted":
				claim.Status = "accepted"
				k.SetClaim(ctx, claim)

				if err := k.ReverifyFact(ctx, claim.ReverificationFactId, result.Confidence, uint64(ctx.BlockHeight())); err != nil {
					k.Logger(ctx).Error("reverification fact update failed",
						"fact_id", claim.ReverificationFactId, "error", err)
				}

			case "rejected":
				claim.Status = "rejected"
				k.SetClaim(ctx, claim)

				// Fact confidence was already decayed; rejection means confidence
				// stays at current (low) level — the fact may expire naturally.
				k.Logger(ctx).Info("reverification rejected, fact confidence unchanged",
					"fact_id", claim.ReverificationFactId)

			default:
				claim.Status = result.Verdict
				k.SetClaim(ctx, claim)
			}

			// Skip standard fact creation / stake settlement — reverification claims have no stake
			goto roundFinalize
		}

		switch result.Verdict {
		case "accepted":
			claim.Status = "accepted"
			k.SetClaim(ctx, claim)
			k.UpdateSubmitterCalibration(ctx, claim.Submitter, true)

			// Create fact from accepted claim
			height := uint64(ctx.BlockHeight())
			factID := GenerateFactID(claim.Id, round.Id)
			// CONF-002: Cannot mark a fact verified with zero participants.
			// Zero-participant finalization produces phantom confidence signals.
			factStatus := types.FactStatusVerified
			factIsProvisional := false
			if result.ParticipantCount == 0 {
				factStatus = types.FactStatusProvisional
				factIsProvisional = true
			}

			fact := types.Fact{
				Id:                   factID,
				ClaimText:            claim.ClaimText,
				Domain:               claim.Domain,
				EpistemicCategory:    claim.EpistemicCategory,
				Confidence:           result.Confidence,
				Status:               factStatus,
				IsProvisional:        factIsProvisional,
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
				OriginalStake:        claim.StakeAmount,
				PartnershipId:        claim.PartnershipId, // B12-4: propagate partnership attribution
			}

			// Transfer evidence summaries from claim to fact (evidence provenance survives)
			if k.evidenceMgmtKeeper != nil {
				ep := k.GetExtendedParams(ctx)
				evidences := k.evidenceMgmtKeeper.GetEvidenceByClaimId(ctx, claim.Id)
				summaries := make([]*types.FactEvidenceSummary, 0, len(evidences))
				for _, ev := range evidences {
					if ev.Status == "rejected" || ev.Status == "expired" {
						continue
					}
					typeStrength, ok := ep.GetEvidenceTypeStrength(ev.EvidenceType)
					if !ok {
						typeStrength = 400_000
					}
					summaries = append(summaries, &types.FactEvidenceSummary{
						EvidenceId:   ev.Id,
						EvidenceType: ev.EvidenceType,
						Reliability:  ev.Reliability,
						TypeStrength: typeStrength,
					})
				}
				fact.EvidenceSummaries = summaries
				fact.EvidenceBonus = result.EvidenceBonus
			}

			k.SetFact(ctx, &fact)

			// B12-4: Record partnership fact creation and credit both partners
			k.RecordPartnershipFactCreation(ctx, &fact)
			k.CreditPartnershipReputation(ctx, &fact, true)

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
				k.SetFactReference(ctx, &types.FactReference{
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
					k.DistributeCitationRewards(ctx, factID, totalReward, claim.Submitter)
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
			k.UpdateSubmitterCalibration(ctx, claim.Submitter, false)

			// FARM-6 fix: Check agent's rejection rate and pause all vesting if misbehaving.
			// This prevents the "grief-and-extract" pattern where agents earn vesting through
			// honest work then profit during misbehavior.
			k.checkMisbehaviorVestingPause(ctx, claim.Submitter)

			// Path A: Automatic counter-fact creation on rejection.
			// Rejected claims produce a "rejection" counter-fact, recording the
			// negative knowledge that "this claim was evaluated and found false."
			k.createRejectionCounterFact(ctx, claim, round, result)

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
			k.SetFact(ctx, &fact)

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

		case "inconclusive":
			claim.Status = "inconclusive"
			k.SetClaim(ctx, claim)

			// Return stake on inconclusive (below 55% agreement) — not the submitter's fault.
			// Without this, the submitter's stake would be permanently locked in the module
			// account with no recovery mechanism. (OpenClaw C-1 fix)
			k.returnClaimStake(ctx, claim)
		}
	}

roundFinalize:
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

	// FARM-1: Track verifier conformity for rubber-stamp detection.
	// FARM-8: Track validator participation for passive staking detection.
	if result.Verdict != "insufficient" && result.Verdict != "inconclusive" {
		// Track assignment for all selected validators (FARM-8)
		for _, validatorAddr := range round.SelectedValidators {
			k.IncrementRoundsAssigned(ctx, validatorAddr)
		}
		// Track participation (FARM-8) and conformity (FARM-1) for revealing validators
		for _, validatorAddr := range sortedMapKeys(round.Reveals) {
			k.IncrementRoundsParticipated(ctx, validatorAddr)
			reveal := round.Reveals[validatorAddr]
			votedWithOutcome := (result.Verdict == "accepted" && reveal.Verdict == "accept") ||
				(result.Verdict == "rejected" && reveal.Verdict == "reject")
			k.UpdateVerifierConformity(ctx, validatorAddr, votedWithOutcome)
		}
	}

	// Notify home module of verification results for epistemic stats.
	// Each validator who revealed gets a RecordVerification call with whether
	// their verdict matched consensus.
	if k.homeKeeper != nil && result.Verdict != "insufficient" {
		domain := ""
		if claimFound {
			domain = claim.Domain
		}
		for _, validatorAddr := range sortedMapKeys(round.Reveals) {
			reveal := round.Reveals[validatorAddr]
			matchedConsensus := reveal.Verdict == result.Verdict
			k.homeKeeper.RecordVerification(ctx, reveal.Validator, domain, matchedConsensus)
		}
	}

	return nil
}

// returnClaimStake returns the full staked amount from the knowledge module back to the claim submitter.
func (k Keeper) returnClaimStake(ctx sdk.Context, claim *types.Claim) {
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
func (k Keeper) slashAndReturnClaimStake(ctx sdk.Context, claim *types.Claim, slashBps uint64) {
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

// handleEquivocation slashes the validator's full stake and records equivocation evidence.
// This is the most severe PoT violation — a validator attempted to submit conflicting
// commitments or reveals in the same verification round.
//
// The slash is attempted BEFORE recording evidence so that the on-chain audit trail
// accurately reflects the actual slashed amount. If the slash fails, evidence is still
// recorded with SlashedAmount="0" to preserve the audit trail, and the error is
// propagated so callers can handle appropriately. (PP-CONSENSUS-3 remediation)
func (k Keeper) handleEquivocation(
	ctx sdk.Context,
	roundID string,
	validatorAddr string,
	equivocationType string,
	originalValue string,
	conflictingValue string,
) {
	height := uint64(ctx.BlockHeight())

	// Get validator's effective stake for 100% slash
	stake := k.stakingKeeper.GetEffectiveStake(ctx, validatorAddr)
	requestedSlash := fmt.Sprintf("%d", stake)

	// Attempt slash FIRST — determine actual slashed amount before recording evidence
	actualSlashed := "0"
	slashFailed := false
	if stake > 0 {
		reason := fmt.Sprintf("PoT %s equivocation in round %s", equivocationType, roundID)
		if err := k.stakingKeeper.SlashValidator(ctx, validatorAddr, requestedSlash, reason); err != nil {
			k.Logger(ctx).Error("CRITICAL: failed to slash equivocating validator — economic security degraded",
				"validator", validatorAddr,
				"round_id", roundID,
				"requested_slash", requestedSlash,
				"error", err,
			)
			slashFailed = true
		} else {
			actualSlashed = requestedSlash
		}
	}

	// Record equivocation evidence with accurate slashed amount
	evidence := types.EquivocationEvidence{
		RoundId:          roundID,
		Validator:        validatorAddr,
		Type:             equivocationType,
		OriginalValue:    originalValue,
		ConflictingValue: conflictingValue,
		DetectedAtHeight: height,
		SlashedAmount:    actualSlashed,
	}
	k.SetEquivocationEvidence(ctx, evidence)

	// Emit event for indexers and monitoring
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("pot_equivocation_detected",
			sdk.NewAttribute("round_id", roundID),
			sdk.NewAttribute("validator", validatorAddr),
			sdk.NewAttribute("type", equivocationType),
			sdk.NewAttribute("original_value", originalValue),
			sdk.NewAttribute("conflicting_value", conflictingValue),
			sdk.NewAttribute("slashed_amount", actualSlashed),
			sdk.NewAttribute("slash_failed", fmt.Sprintf("%t", slashFailed)),
			sdk.NewAttribute("height", fmt.Sprintf("%d", height)),
		),
	)

	if slashFailed {
		k.Logger(ctx).Error("EQUIVOCATION DETECTED — slash FAILED, validator retains stake",
			"validator", validatorAddr,
			"round_id", roundID,
			"type", equivocationType,
			"requested_slash", requestedSlash,
		)
	} else {
		k.Logger(ctx).Error("EQUIVOCATION DETECTED — validator slashed 100%",
			"validator", validatorAddr,
			"round_id", roundID,
			"type", equivocationType,
			"slashed", actualSlashed,
		)
	}
}

// completeContradictionRound handles the completion of a contradiction verification round.
// Updates the counter-fact status and, if accepted, marks the target fact as contested/disproven.
func (k Keeper) completeContradictionRound(
	ctx sdk.Context,
	round *types.VerificationRound,
	result *types.VerificationResult,
) {
	// round.ClaimId is the counter-fact ID for contradiction rounds
	cf, found := k.GetCounterFact(ctx, round.ClaimId)
	if !found {
		k.Logger(ctx).Error("contradiction round completed but counter-fact not found",
			"counter_fact_id", round.ClaimId, "round_id", round.Id)
		return
	}

	height := uint64(ctx.BlockHeight())

	// Collect verifiers (sorted for deterministic state)
	cf.Verifiers = sortedMapKeys(round.Reveals)

	switch result.Verdict {
	case "accepted":
		// Contradiction accepted by verifiers — enter challenge window (R-2).
		// Counter-fact transitions to "provisional" instead of directly to "verified".
		// Stake return + target fact update + falsification reward are deferred
		// until the challenge window closes (ProcessCounterFactChallengeExpiry).
		cf.Status = types.CounterFactStatusProvisional
		cf.Confidence = result.Confidence
		// VerifiedAtHeight set later when window closes

		// Open challenge window
		windowEnd := height + types.CounterFactChallengeWindowBlocks
		k.SetCounterFactChallengeWindow(ctx, cf.Id, windowEnd)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("contradiction_provisional",
				sdk.NewAttribute("counter_fact_id", cf.Id),
				sdk.NewAttribute("target_fact_id", cf.TargetFactId),
				sdk.NewAttribute("confidence", fmt.Sprintf("%d", cf.Confidence)),
				sdk.NewAttribute("challenge_window_end", fmt.Sprintf("%d", windowEnd)),
			),
		)

	case "rejected":
		// Contradiction rejected: the target fact stands, counter-fact overturned.
		cf.Status = types.CounterFactStatusOverturned
		cf.Confidence = 0

		// Slash FailedContradictionSlashBps (50%) of locked stake, return remainder.
		if cf.Submitter != "" && cf.Submitter != "system" && cf.StakeAmount != "" && k.bankKeeper != nil {
			lockedStake := new(big.Int)
			if _, ok := lockedStake.SetString(cf.StakeAmount, 10); ok && lockedStake.Sign() > 0 {
				// Calculate slash amount: lockedStake * FailedContradictionSlashBps / 1_000_000
				slashAmount := new(big.Int).Mul(lockedStake, big.NewInt(types.FailedContradictionSlashBps))
				slashAmount.Div(slashAmount, big.NewInt(1_000_000))
				remainder := new(big.Int).Sub(lockedStake, slashAmount)

				// Burn the slashed portion
				if slashAmount.Sign() > 0 {
					burnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(slashAmount)))
					if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
						k.Logger(ctx).Error("failed to burn contradiction slash",
							"counter_fact_id", cf.Id, "amount", slashAmount.String(), "error", err)
					}
				}

				// Return remainder to submitter
				if remainder.Sign() > 0 {
					submitterAddr, err := sdk.AccAddressFromBech32(cf.Submitter)
					if err == nil {
						returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(remainder)))
						if err := k.bankKeeper.SendCoinsFromModuleToAccount(
							ctx, types.ModuleName, submitterAddr, returnCoins,
						); err != nil {
							k.Logger(ctx).Error("failed to return contradiction stake remainder",
								"counter_fact_id", cf.Id, "amount", remainder.String(), "error", err)
						}
					}
				}

				k.Logger(ctx).Info("failed contradiction — stake slashed",
					"counter_fact_id", cf.Id,
					"submitter", cf.Submitter,
					"slashed", slashAmount.String(),
					"returned", remainder.String(),
				)
			}
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("contradiction_rejected",
				sdk.NewAttribute("counter_fact_id", cf.Id),
				sdk.NewAttribute("target_fact_id", cf.TargetFactId),
			),
		)

	default:
		// Contested or insufficient — leave as pending/provisional
		cf.Status = types.CounterFactStatusContested
		cf.Confidence = result.Confidence
	}

	k.SetCounterFact(ctx, cf)
}

// distributeFalsificationReward pays the contradiction submitter for successfully
// disproving a fact. The reward is proportional to the target fact's original stake,
// scaled by the counter-fact's confidence, and dynamically capped based on the
// target fact's age and citation impact (FARM-15).
//
// Reward formula:
//
//	rewardBase = targetFact.OriginalStake * cf.Confidence / 1_000_000
//	dynamicCap = cfStake * (baseMultiplier + ageBonus + citationBonus) / 1_000_000
//	reward     = min(rewardBase, dynamicCap)
//
// Where ageBonus scales linearly from 0 to FalsificationAgeMaxBonusBps over
// FalsificationAgeScaleBlocks, and citationBonus scales linearly from 0 to
// FalsificationCiteMaxBonusBps over FalsificationCiteScaleCount citations.
//
// The reward is minted from the knowledge module account. The submitter's locked
// stake is returned separately in completeContradictionRound (accepted case).
func (k Keeper) distributeFalsificationReward(ctx sdk.Context, cf *types.CounterFact, targetFact *types.Fact) {
	if k.bankKeeper == nil || cf.Submitter == "" || cf.Submitter == "system" {
		return
	}

	submitterAddr, err := sdk.AccAddressFromBech32(cf.Submitter)
	if err != nil {
		k.Logger(ctx).Error("invalid falsification reward recipient", "submitter", cf.Submitter, "error", err)
		return
	}

	// Determine base reward from target fact's original stake scaled by confidence
	var rewardAmount *big.Int

	targetStake := new(big.Int)
	if targetFact.OriginalStake != "" {
		targetStake.SetString(targetFact.OriginalStake, 10)
	}

	if targetStake.Sign() > 0 {
		// rewardBase = targetStake * confidence / 1_000_000
		rewardAmount = new(big.Int).Mul(targetStake, new(big.Int).SetUint64(cf.Confidence))
		rewardAmount.Div(rewardAmount, big.NewInt(1_000_000))
	} else {
		// Fallback for facts without OriginalStake (pre-migration facts):
		// reward = confidence ulgm (1 LGM max) — preserves backward compatibility
		rewardAmount = new(big.Int).SetUint64(cf.Confidence)
	}

	// FARM-15: Dynamic reward cap based on target fact's age and citation impact.
	// Replaces the flat FalsificationRewardCapBps (3x) with:
	//   dynamicCapBps = baseMultiplier + ageBonus + citationBonus
	// where ageBonus and citationBonus ramp linearly from 0 to their max values.
	if cf.StakeAmount != "" {
		cfStake := new(big.Int)
		if _, ok := cfStake.SetString(cf.StakeAmount, 10); ok && cfStake.Sign() > 0 {
			ep := k.GetExtendedParams(ctx)

			// Age bonus: linear ramp from 0 to FalsificationAgeMaxBonusBps
			currentHeight := uint64(ctx.BlockHeight())
			ageBlocks := uint64(0)
			if currentHeight > targetFact.CreatedAtHeight {
				ageBlocks = currentHeight - targetFact.CreatedAtHeight
			}
			ageBonusBps := ep.FalsificationAgeMaxBonusBps
			if ageBlocks < ep.FalsificationAgeScaleBlocks {
				ageBonusBps = safeMulDiv(ep.FalsificationAgeMaxBonusBps, ageBlocks, ep.FalsificationAgeScaleBlocks)
			}

			// Citation bonus: linear ramp from 0 to FalsificationCiteMaxBonusBps
			citeBonusBps := ep.FalsificationCiteMaxBonusBps
			if targetFact.CitationCount < ep.FalsificationCiteScaleCount {
				citeBonusBps = safeMulDiv(ep.FalsificationCiteMaxBonusBps, targetFact.CitationCount, ep.FalsificationCiteScaleCount)
			}

			// Total multiplier: base + age + citation
			dynamicCapBps := ep.FalsificationBaseMultiplierBps + ageBonusBps + citeBonusBps

			// rewardCap = cfStake * dynamicCapBps / 1_000_000
			rewardCap := new(big.Int).Mul(cfStake, new(big.Int).SetUint64(dynamicCapBps))
			rewardCap.Div(rewardCap, big.NewInt(1_000_000))
			if rewardAmount.Cmp(rewardCap) > 0 {
				rewardAmount.Set(rewardCap)
			}
		}
	}

	// Apply FalsificationPoolShareBps epoch cap with carry-forward (R-3)
	height := uint64(ctx.BlockHeight())
	epoch := height / 240_000
	epochPaid := k.GetFalsificationEpochPaid(ctx, epoch)
	epochCap := k.getFalsificationEpochCap(ctx)
	if epochCap > 0 {
		remaining := new(big.Int).Sub(big.NewInt(int64(epochCap)), big.NewInt(int64(epochPaid)))
		if remaining.Sign() <= 0 {
			// Carry forward the full reward to next epoch
			k.AddFalsificationCarryForward(ctx, epoch+1, rewardAmount.Uint64())
			k.Logger(ctx).Info("falsification epoch cap reached, reward carried forward",
				"counter_fact_id", cf.Id, "epoch", epoch,
				"deferred_amount", rewardAmount.String(),
				"next_epoch", epoch+1)
			return
		}
		if rewardAmount.Cmp(remaining) > 0 {
			// Pay what we can, carry forward the rest
			deferred := new(big.Int).Sub(rewardAmount, remaining)
			k.AddFalsificationCarryForward(ctx, epoch+1, deferred.Uint64())
			rewardAmount.Set(remaining)
			k.Logger(ctx).Info("falsification reward partially deferred",
				"counter_fact_id", cf.Id, "epoch", epoch,
				"paid", remaining.String(),
				"deferred", deferred.String())
		}
	}

	if rewardAmount.Sign() <= 0 {
		return
	}

	// Apply unified research tax (3.33%) — falsification rewards are new value flowing to individuals
	researchBps := k.GetResearchTaxBps(ctx)
	if researchBps > 0 && k.bankKeeper != nil {
		taxAmount := new(big.Int).Mul(rewardAmount, new(big.Int).SetUint64(researchBps))
		taxAmount.Div(taxAmount, big.NewInt(1_000_000))
		if taxAmount.Sign() > 0 {
			taxCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(taxAmount)))
			if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "research_fund", taxCoins); err != nil {
				k.Logger(ctx).Error("falsification research tax transfer failed",
					"counter_fact_id", cf.Id, "tax", taxAmount.String(), "error", err)
			} else {
				rewardAmount.Sub(rewardAmount, taxAmount)
			}
		}
	}

	if rewardAmount.Sign() <= 0 {
		return
	}

	// Route through vesting system (clawback-protected) if available, else direct bank transfer
	if k.vestingRewardsKeeper != nil {
		if err := k.vestingRewardsKeeper.DistributeFalsificationReward(
			ctx, cf.Id, cf.TargetFactId, cf.Submitter, rewardAmount.String(),
		); err != nil {
			k.Logger(ctx).Error("failed to distribute falsification reward via vesting",
				"counter_fact_id", cf.Id, "submitter", cf.Submitter, "amount", rewardAmount.String(), "error", err)
			return
		}
	} else {
		// Fallback: direct bank transfer (no clawback protection), routed through partnership if active
		rewardCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(rewardAmount)))
		if err := k.RouteReward(ctx, submitterAddr, rewardCoins, "falsification"); err != nil {
			k.Logger(ctx).Error("failed to distribute falsification reward",
				"counter_fact_id", cf.Id, "submitter", cf.Submitter, "amount", rewardAmount.String(), "error", err)
			return
		}
	}

	// Track epoch payout for pool cap
	k.SetFalsificationEpochPaid(ctx, epoch, epochPaid+rewardAmount.Uint64())

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("falsification_reward_distributed",
			sdk.NewAttribute("counter_fact_id", cf.Id),
			sdk.NewAttribute("target_fact_id", cf.TargetFactId),
			sdk.NewAttribute("recipient", cf.Submitter),
			sdk.NewAttribute("amount", rewardAmount.String()),
			sdk.NewAttribute("target_stake", targetStake.String()),
			sdk.NewAttribute("cf_confidence", fmt.Sprintf("%d", cf.Confidence)),
			sdk.NewAttribute("cf_stake", cf.StakeAmount),
			sdk.NewAttribute("reward_cap_bps", "dynamic"), // FARM-15: age×citation weighted
			sdk.NewAttribute("epoch", fmt.Sprintf("%d", epoch)),
			sdk.NewAttribute("epoch_paid_before", fmt.Sprintf("%d", epochPaid)),
		),
	)
}

// getFalsificationEpochCap returns the max ulgm payable for falsification rewards in an epoch.
// Calculated as FalsificationPoolShareBps (15%) of the epoch's total block reward pool,
// plus any carry-forward from prior epochs where the cap was hit.
// Returns 0 if the vesting rewards keeper is unavailable (cap disabled).
func (k Keeper) getFalsificationEpochCap(ctx sdk.Context) uint64 {
	if k.vestingRewardsKeeper == nil {
		return 0
	}
	height := uint64(ctx.BlockHeight())
	epoch := height / 240_000
	epochPool := k.vestingRewardsKeeper.GetEpochBlockRewardPool(ctx, epoch)
	if epochPool == 0 {
		return 0
	}
	// baseCap = epochPool * FalsificationPoolShareBps / 1_000_000
	baseCap := epochPool * types.FalsificationPoolShareBps / 1_000_000
	// Add carry-forward from prior epoch cap overflow
	carryForward := k.GetFalsificationCarryForward(ctx, epoch)
	return baseCap + carryForward
}

// SystemCounterFactStakeShareBps is the fraction of the rejected claim's slash amount
// allocated as the system-generated counter-fact's economic backing (R-4).
// 10% of the slashed amount gives auto-counter-facts meaningful weight in
// bidirectional confidence calculations.
const SystemCounterFactStakeShareBps = 100_000 // 10%

// createRejectionCounterFact creates a counter-fact from a rejected claim (Path A).
// This records negative knowledge: "claim X was evaluated and found false by validators."
// Per R-4, a nominal stake is allocated from the slashed claim amount to give
// the auto-counter-fact economic backing in the bidirectional confidence system.
func (k Keeper) createRejectionCounterFact(
	ctx sdk.Context,
	claim *types.Claim,
	round *types.VerificationRound,
	result *types.VerificationResult,
) {
	height := uint64(ctx.BlockHeight())
	cfID := GenerateCounterFactID("", claim.Id, "system", height)

	// Collect verifier addresses from reveals (sorted for deterministic state)
	verifiers := sortedMapKeys(round.Reveals)

	// R-4: Derive counter-fact stake from the rejected claim's slash amount.
	// The slash amount sits in the module account after slashAndReturnClaimStake.
	// We reserve SystemCounterFactStakeShareBps (10%) of the slash as backing.
	// Use default slash BPS (220k = 22%) since we don't have params in this context.
	var stakeAmount string
	if claim.StakeAmount != "" {
		claimStake := new(big.Int)
		if _, ok := claimStake.SetString(claim.StakeAmount, 10); ok && claimStake.Sign() > 0 {
			slashAmount := new(big.Int).Mul(claimStake, big.NewInt(220_000))
			slashAmount.Div(slashAmount, big.NewInt(1_000_000))
			cfStake := new(big.Int).Mul(slashAmount, big.NewInt(SystemCounterFactStakeShareBps))
			cfStake.Div(cfStake, big.NewInt(1_000_000))
			if cfStake.Sign() > 0 {
				stakeAmount = cfStake.String()
			}
		}
	}

	cf := types.CounterFact{
		Id:                cfID,
		TargetClaimId:     claim.Id,
		NegationType:      types.NegationTypeRejection,
		Confidence:        1_000_000 - result.Confidence, // Inverse of claim confidence
		Justification:     fmt.Sprintf("claim rejected by %d verifiers with confidence %d", result.ParticipantCount, result.Confidence),
		Submitter:         "system", // Auto-generated by protocol
		Verifiers:         verifiers,
		Domain:            claim.Domain,
		LogicZone:         claim.LogicZone,
		EpistemicCategory: claim.EpistemicCategory,
		Status:            types.CounterFactStatusVerified, // Already verified by round
		CreatedAtHeight:   height,
		VerifiedAtHeight:  height,
		RoundId:           round.Id,
		StakeAmount:       stakeAmount,
	}
	k.SetCounterFact(ctx, &cf)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("rejection_counter_fact_created",
			sdk.NewAttribute("counter_fact_id", cfID),
			sdk.NewAttribute("claim_id", claim.Id),
			sdk.NewAttribute("confidence", fmt.Sprintf("%d", cf.Confidence)),
			sdk.NewAttribute("stake", stakeAmount),
		),
	)
}

// CreateSupersessionCounterFact creates a counter-fact when new evidence supersedes
// an existing fact (Path D). Called by external modules (e.g., research, capture_challenge)
// when a fact is superseded by newer knowledge.
func (k Keeper) CreateSupersessionCounterFact(
	ctx sdk.Context,
	targetFactId string,
	newFactId string,
	submitter string,
	justification string,
	confidence uint64,
) (string, error) {
	fact, found := k.GetFact(ctx, targetFactId)
	if !found {
		return "", fmt.Errorf("%w: %s", types.ErrFactNotFound, targetFactId)
	}

	height := uint64(ctx.BlockHeight())
	cfID := GenerateCounterFactID(targetFactId, "", submitter, height)

	cf := types.CounterFact{
		Id:                cfID,
		TargetFactId:      targetFactId,
		NegationType:      types.NegationTypeSupersession,
		Confidence:        confidence,
		Justification:     justification,
		Submitter:         submitter,
		Domain:            fact.Domain,
		LogicZone:         fact.LogicZone,
		EpistemicCategory: fact.EpistemicCategory,
		Status:            types.CounterFactStatusVerified,
		CreatedAtHeight:   height,
		VerifiedAtHeight:  height,
	}
	k.SetCounterFact(ctx, &cf)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("supersession_counter_fact_created",
			sdk.NewAttribute("counter_fact_id", cfID),
			sdk.NewAttribute("target_fact_id", targetFactId),
			sdk.NewAttribute("new_fact_id", newFactId),
			sdk.NewAttribute("confidence", fmt.Sprintf("%d", confidence)),
		),
	)

	return cfID, nil
}

// ---------- Counter-Fact Challenge Window (R-2) ----------

// OpenCounterFactChallenge creates a challenge against a provisional counter-fact.
// Enables symmetry: negative knowledge must survive adversarial scrutiny.
func (k Keeper) OpenCounterFactChallenge(
	ctx sdk.Context,
	challenger, counterFactId, stake, reason string,
	evidenceIds []string,
) (types.CounterFactChallenge, error) {
	cf, found := k.GetCounterFact(ctx, counterFactId)
	if !found {
		return types.CounterFactChallenge{}, fmt.Errorf("counter-fact not found: %s", counterFactId)
	}
	if cf.Status != types.CounterFactStatusProvisional {
		return types.CounterFactChallenge{}, fmt.Errorf("counter-fact %s is not provisional (status: %s)", counterFactId, cf.Status)
	}

	// Check challenge window
	windowEnd, hasWindow := k.GetCounterFactChallengeWindow(ctx, counterFactId)
	height := uint64(ctx.BlockHeight())
	if hasWindow && height >= windowEnd {
		return types.CounterFactChallenge{}, fmt.Errorf("challenge window closed at block %d", windowEnd)
	}

	// Cannot challenge your own counter-fact
	if challenger == cf.Submitter {
		return types.CounterFactChallenge{}, fmt.Errorf("cannot challenge your own counter-fact")
	}

	// Validate stake
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return types.CounterFactChallenge{}, fmt.Errorf("invalid stake amount")
	}
	minStake := big.NewInt(int64(types.CounterFactChallengeMinStake))
	if stakeAmount.Cmp(minStake) < 0 {
		return types.CounterFactChallenge{}, fmt.Errorf("stake %s below minimum %d", stake, types.CounterFactChallengeMinStake)
	}

	// Lock challenger stake
	challengerAddr, err := sdk.AccAddressFromBech32(challenger)
	if err != nil {
		return types.CounterFactChallenge{}, fmt.Errorf("invalid challenger address: %w", err)
	}
	if k.bankKeeper != nil {
		stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount))
		if err := k.bankKeeper.SendCoinsFromAccountToModule(
			ctx, challengerAddr, types.ModuleName, sdk.NewCoins(stakeCoin),
		); err != nil {
			return types.CounterFactChallenge{}, fmt.Errorf("insufficient stake: %w", err)
		}
	}

	challengeID := fmt.Sprintf("cfc-%x", sha256Hash([]byte(counterFactId + challenger + fmt.Sprintf("%d", height)))[:16])

	challenge := types.CounterFactChallenge{
		Id:              challengeID,
		CounterFactId:   counterFactId,
		Challenger:      challenger,
		Stake:           stake,
		Reason:          reason,
		EvidenceIds:     evidenceIds,
		Status:          types.CounterFactChallengeStatusOpen,
		CreatedAtHeight: height,
	}
	k.SetCounterFactChallenge(ctx, challenge)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("counter_fact_challenge_opened",
			sdk.NewAttribute("challenge_id", challengeID),
			sdk.NewAttribute("counter_fact_id", counterFactId),
			sdk.NewAttribute("challenger", challenger),
			sdk.NewAttribute("stake", stake),
		),
	)

	return challenge, nil
}

// ProcessCounterFactChallengeExpiry handles counter-facts whose challenge window has closed.
// If unchallenged → finalize as verified (stake return + target update + reward).
// Called in BeginBlocker.
func (k Keeper) ProcessCounterFactChallengeExpiry(ctx sdk.Context, currentHeight uint64) {
	k.IterateCounterFacts(ctx, func(cf *types.CounterFact) bool {
		if cf.Status != types.CounterFactStatusProvisional {
			return false
		}

		windowEnd, hasWindow := k.GetCounterFactChallengeWindow(ctx, cf.Id)
		if !hasWindow || currentHeight < windowEnd {
			return false
		}

		// Check for open challenges
		hasOpenChallenge := false
		k.IterateCounterFactChallenges(ctx, func(c types.CounterFactChallenge) bool {
			if c.CounterFactId == cf.Id && c.Status == types.CounterFactChallengeStatusOpen {
				hasOpenChallenge = true
				return true
			}
			return false
		})
		if hasOpenChallenge {
			return false
		}

		// Window closed, no open challenges → finalize
		k.FinalizeCounterFact(ctx, cf, currentHeight)
		return false
	})
}

// FinalizeCounterFact transitions a provisional counter-fact to verified and
// executes all deferred effects: stake return, target fact update, falsification reward.
func (k Keeper) FinalizeCounterFact(ctx sdk.Context, cf *types.CounterFact, height uint64) {
	cf.Status = types.CounterFactStatusVerified
	cf.VerifiedAtHeight = height

	// Return locked contradiction stake to submitter
	if cf.Submitter != "" && cf.Submitter != "system" && cf.StakeAmount != "" && k.bankKeeper != nil {
		lockedStake := new(big.Int)
		if _, ok := lockedStake.SetString(cf.StakeAmount, 10); ok && lockedStake.Sign() > 0 {
			submitterAddr, err := sdk.AccAddressFromBech32(cf.Submitter)
			if err == nil {
				returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(lockedStake)))
				if err := k.bankKeeper.SendCoinsFromModuleToAccount(
					ctx, types.ModuleName, submitterAddr, returnCoins,
				); err != nil {
					k.Logger(ctx).Error("failed to return contradiction stake",
						"counter_fact_id", cf.Id, "amount", lockedStake.String(), "error", err)
				} else {
					k.Logger(ctx).Info("returned contradiction stake",
						"counter_fact_id", cf.Id,
						"submitter", cf.Submitter,
						"amount", lockedStake.String(),
					)
				}
			}
		}
	}

	// Update target fact status
	if cf.TargetFactId != "" {
		targetFact, targetFound := k.GetFact(ctx, cf.TargetFactId)
		if targetFound {
			bidir := k.GetBidirectionalConfidence(ctx, cf.TargetFactId)
			if bidir.Status == "disproven" {
				targetFact.Status = types.FactStatusDisproven
			} else {
				targetFact.Status = types.FactStatusContested
			}
			k.SetFact(ctx, targetFact)

			// Falsification reward
			k.distributeFalsificationReward(ctx, cf, targetFact)
		}
	}

	k.SetCounterFact(ctx, cf)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("counter_fact_finalized",
			sdk.NewAttribute("counter_fact_id", cf.Id),
			sdk.NewAttribute("target_fact_id", cf.TargetFactId),
			sdk.NewAttribute("confidence", fmt.Sprintf("%d", cf.Confidence)),
		),
	)
}

// ResolveCounterFactChallenge resolves an open counter-fact challenge.
// If the challenge succeeds: counter-fact is overturned, challenger bond returned.
// If the challenge fails: counter-fact survives, challenger slashed.
func (k Keeper) ResolveCounterFactChallenge(
	ctx sdk.Context,
	challengeId string,
	outcome string, // "upheld" (counter-fact survives) or "reverted" (counter-fact wrong)
) error {
	challenge, found := k.GetCounterFactChallenge(ctx, challengeId)
	if !found {
		return fmt.Errorf("counter-fact challenge not found: %s", challengeId)
	}
	if challenge.Status != types.CounterFactChallengeStatusOpen {
		return fmt.Errorf("challenge %s not open (status: %s)", challengeId, challenge.Status)
	}

	height := uint64(ctx.BlockHeight())
	challenge.ResolvedAtHeight = height

	cf, found := k.GetCounterFact(ctx, challenge.CounterFactId)
	if !found {
		return fmt.Errorf("counter-fact %s not found", challenge.CounterFactId)
	}

	switch outcome {
	case types.CounterFactChallengeStatusUpheld:
		// Counter-fact survives — slash challenger
		challenge.Status = types.CounterFactChallengeStatusUpheld
		k.slashCounterFactChallengerBond(ctx, challenge)

		// Boost counter-fact confidence by 11% (mirrors fact challenge survival)
		boost := uint64(110_000)
		cf.Confidence += boost
		if cf.Confidence > 1_000_000 {
			cf.Confidence = 1_000_000
		}
		k.SetCounterFact(ctx, cf)

	case types.CounterFactChallengeStatusReverted:
		// Counter-fact overturned — return challenger bond, revert counter-fact
		challenge.Status = types.CounterFactChallengeStatusReverted
		k.returnCounterFactChallengerBond(ctx, challenge)

		cf.Status = types.CounterFactStatusOverturned
		cf.Confidence = 0
		k.SetCounterFact(ctx, cf)

		// Slash contradiction submitter's remaining stake (if locked in module)
		if cf.Submitter != "" && cf.Submitter != "system" && cf.StakeAmount != "" && k.bankKeeper != nil {
			lockedStake := new(big.Int)
			if _, ok := lockedStake.SetString(cf.StakeAmount, 10); ok && lockedStake.Sign() > 0 {
				slashAmount := new(big.Int).Mul(lockedStake, big.NewInt(types.FailedContradictionSlashBps))
				slashAmount.Div(slashAmount, big.NewInt(1_000_000))
				remainder := new(big.Int).Sub(lockedStake, slashAmount)

				if slashAmount.Sign() > 0 {
					burnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(slashAmount)))
					_ = k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins)
				}
				if remainder.Sign() > 0 {
					submitterAddr, err := sdk.AccAddressFromBech32(cf.Submitter)
					if err == nil {
						returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(remainder)))
						_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, submitterAddr, returnCoins)
					}
				}
			}
		}

		// Restore target fact status
		if cf.TargetFactId != "" {
			targetFact, targetFound := k.GetFact(ctx, cf.TargetFactId)
			if targetFound && (targetFact.Status == types.FactStatusContested || targetFact.Status == types.FactStatusDisproven) {
				targetFact.Status = types.FactStatusActive
				k.SetFact(ctx, targetFact)
			}
		}

	default:
		return fmt.Errorf("unknown outcome: %s", outcome)
	}

	k.SetCounterFactChallenge(ctx, challenge)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("counter_fact_challenge_resolved",
			sdk.NewAttribute("challenge_id", challengeId),
			sdk.NewAttribute("counter_fact_id", challenge.CounterFactId),
			sdk.NewAttribute("outcome", outcome),
		),
	)

	return nil
}

// slashCounterFactChallengerBond slashes and returns bond for a failed counter-fact challenge.
func (k Keeper) slashCounterFactChallengerBond(ctx sdk.Context, challenge types.CounterFactChallenge) {
	if k.bankKeeper == nil || challenge.Stake == "" {
		return
	}
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(challenge.Stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return
	}

	slashAmount := new(big.Int).Mul(stakeAmount, big.NewInt(types.CounterFactChallengeSlashBps))
	slashAmount.Div(slashAmount, big.NewInt(1_000_000))
	returnAmount := new(big.Int).Sub(stakeAmount, slashAmount)

	challengerAddr, err := sdk.AccAddressFromBech32(challenge.Challenger)
	if err != nil {
		return
	}

	if returnAmount.Sign() > 0 {
		returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(returnAmount)))
		_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, returnCoins)
	}
	if slashAmount.Sign() > 0 {
		burnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(slashAmount)))
		_ = k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins)
	}
}

// returnCounterFactChallengerBond returns the full bond to a successful counter-fact challenger.
func (k Keeper) returnCounterFactChallengerBond(ctx sdk.Context, challenge types.CounterFactChallenge) {
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
	_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, returnCoins)
}

// sha256Hash is a thin wrapper for deterministic ID generation.
func sha256Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// checkMisbehaviorVestingPause checks an agent's rejection rate and pauses all their
// active vesting schedules if it exceeds MisbehaviorRejectionThreshold (30%).
// FARM-6 remediation: prevents "grief-and-extract" where agents accumulate vesting
// from honest work then misbehave while continuing to receive vesting releases.
func (k Keeper) checkMisbehaviorVestingPause(ctx sdk.Context, submitter string) {
	if k.vestingRewardsKeeper == nil {
		return
	}

	ep := k.GetExtendedParams(ctx)
	record, found := k.GetCalibrationRecord(ctx, submitter)
	if !found || record.TotalClaims < uint64(ep.MisbehaviorMinSamples) {
		return
	}

	// Calculate rejection rate: rejectedClaims * 1,000,000 / totalClaims
	rejectionRate := (record.RejectedClaims * 1_000_000) / record.TotalClaims
	if rejectionRate > ep.MisbehaviorRejectionThreshold {
		paused := k.vestingRewardsKeeper.PauseAllVestingByRecipient(ctx, submitter)
		if paused > 0 {
			k.Logger(ctx).Warn("FARM-6: agent misbehavior detected — vesting paused",
				"submitter", submitter,
				"rejection_rate", rejectionRate,
				"threshold", ep.MisbehaviorRejectionThreshold,
				"schedules_paused", paused,
			)
		}
	}
}

// CleanupExpiredRounds removes rounds that completed more than retentionBlocks ago.
func (k Keeper) CleanupExpiredRounds(ctx sdk.Context, retentionBlocks uint64) {
	currentHeight := uint64(ctx.BlockHeight())
	if currentHeight < retentionBlocks {
		return
	}
	cutoff := currentHeight - retentionBlocks

	var toDelete []string
	k.IterateVerificationRounds(ctx, func(r *types.VerificationRound) bool {
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
