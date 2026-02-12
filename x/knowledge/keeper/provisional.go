package keeper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/evaluation"
	"github.com/legible-money/legible/x/knowledge/types"
)

// EvaluatePendingClaims runs the deterministic structural evaluation engine
// on all claims with status "pending_evaluation". Called in BeginBlocker.
func (k Keeper) EvaluatePendingClaims(ctx sdk.Context, currentHeight uint64) {
	var pendingClaims []types.Claim
	k.IterateClaims(ctx, func(c types.Claim) bool {
		if c.Status == types.ClaimStatusPendingEvaluation {
			pendingClaims = append(pendingClaims, c)
		}
		return false
	})

	params := k.GetParams(ctx)
	for _, claim := range pendingClaims {
		if err := k.EvaluateAndProvisionClaim(ctx, claim, params, currentHeight); err != nil {
			k.Logger(ctx).Error("failed to evaluate claim",
				"claim_id", claim.Id, "error", err)
		}
	}
}

// EvaluateAndProvisionClaim runs the 6-stage structural evaluation pipeline on a claim
// and transitions it to the appropriate status:
//   - score < RejectThreshold (300k): REJECTED — slash 22%, return rest
//   - 300k ≤ score < ProvisionalThreshold (500k): EVALUATED — challenge window, no provisional fact
//   - score ≥ ProvisionalThreshold (500k): PROVISIONAL — create provisional fact, challenge window
func (k Keeper) EvaluateAndProvisionClaim(ctx sdk.Context, claim types.Claim, params types.Params, currentHeight uint64) error {
	// Run the deterministic 6-stage evaluation engine using the same API
	// as vote_extensions.go — all validators produce identical results.
	evalCtx := k.BuildEvaluationContext()

	// Create a synthetic round for the evaluation engine (required by the API
	// but not used for adversarial verification — the round serves only as context)
	syntheticRound := types.VerificationRound{
		Id:          "adversarial:" + claim.Id,
		ClaimId:     claim.Id,
		Phase:       "aggregation",
		StartHeight: currentHeight,
	}

	evalResult, err := evaluation.EvaluateClaim(ctx, evalCtx, syntheticRound, claim)
	if err != nil {
		return fmt.Errorf("evaluation failed for claim %s: %w", claim.Id, err)
	}

	score := evalResult.Confidence
	claim.StructuralScore = score

	rejectThreshold := uint64(types.RejectThresholdDefault)
	provisionalThreshold := params.ProvisionalThreshold
	if provisionalThreshold == 0 {
		provisionalThreshold = types.ProvisionalThresholdDefault
	}

	if score < rejectThreshold {
		// REJECTED — structural score too low
		claim.Status = types.ClaimStatusRejected
		k.SetClaim(ctx, claim)
		k.UpdateSubmitterCalibration(ctx, claim.Submitter, false)
		k.slashAndReturnClaimStake(ctx, claim, params.InvalidClaimSlashBps)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("claim_rejected_structural",
				sdk.NewAttribute("claim_id", claim.Id),
				sdk.NewAttribute("score", fmt.Sprintf("%d", score)),
			),
		)
		return nil
	}

	// Calculate challenge window duration based on epistemic category
	windowDuration := k.calculateChallengeWindow(claim.EpistemicCategory, claim.LogicZone, score, params)
	claim.ChallengeWindowStart = currentHeight
	claim.ChallengeWindowEnd = currentHeight + windowDuration

	if score >= provisionalThreshold {
		// PROVISIONAL — create provisional fact
		factID := GenerateFactID(claim.Id, fmt.Sprintf("provisional:%d", currentHeight))
		fact := types.Fact{
			Id:                   factID,
			ClaimText:            claim.ClaimText,
			Domain:               claim.Domain,
			EpistemicCategory:    claim.EpistemicCategory,
			Confidence:           score,
			Status:               types.FactStatusProvisional,
			Submitter:            claim.Submitter,
			SubmitterDid:         claim.SubmitterDid,
			References:           claim.References,
			VerifiedAtHeight:     0, // not yet verified
			LogicZone:            claim.LogicZone,
			CreatedAtHeight:      currentHeight,
			LastConfidenceUpdate: currentHeight,
			BaseConfidence:       score,
			Maturity:             types.MaturityEmerging,
			IsProvisional:        true,
			ChallengeWindowEnd:   claim.ChallengeWindowEnd,
			OriginalClaimId:      claim.Id,
		}
		k.SetFact(ctx, fact)

		claim.Status = types.ClaimStatusProvisional
		claim.ProvisionalFactId = factID
		k.SetClaim(ctx, claim)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("claim_provisional",
				sdk.NewAttribute("claim_id", claim.Id),
				sdk.NewAttribute("fact_id", factID),
				sdk.NewAttribute("score", fmt.Sprintf("%d", score)),
				sdk.NewAttribute("window_end", fmt.Sprintf("%d", claim.ChallengeWindowEnd)),
			),
		)
	} else {
		// EVALUATED — in challenge window but no provisional fact
		claim.Status = types.ClaimStatusEvaluated
		k.SetClaim(ctx, claim)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("claim_evaluated",
				sdk.NewAttribute("claim_id", claim.Id),
				sdk.NewAttribute("score", fmt.Sprintf("%d", score)),
				sdk.NewAttribute("window_end", fmt.Sprintf("%d", claim.ChallengeWindowEnd)),
			),
		)
	}

	return nil
}

// ProcessChallengeWindowExpiry checks claims whose challenge window has closed
// and transitions them to verified (or final rejection). Called in BeginBlocker.
func (k Keeper) ProcessChallengeWindowExpiry(ctx sdk.Context, currentHeight uint64) {
	k.IterateClaims(ctx, func(claim types.Claim) bool {
		// Only process claims in challenge window
		if claim.Status != types.ClaimStatusProvisional && claim.Status != types.ClaimStatusEvaluated {
			return false
		}
		// Skip if window hasn't closed yet
		if currentHeight < claim.ChallengeWindowEnd {
			return false
		}
		// Skip if there are active challenges
		if claim.ActiveChallengeCount > 0 {
			return false
		}

		if err := k.TransitionToVerified(ctx, claim, currentHeight); err != nil {
			k.Logger(ctx).Error("failed to transition claim to verified",
				"claim_id", claim.Id, "error", err)
		}
		return false
	})
}

// TransitionToVerified transitions a claim from provisional/evaluated to verified
// after its challenge window closes without successful challenges.
func (k Keeper) TransitionToVerified(ctx sdk.Context, claim types.Claim, currentHeight uint64) error {
	if claim.Status == types.ClaimStatusProvisional {
		// Upgrade provisional fact to verified
		fact, found := k.GetFact(ctx, claim.ProvisionalFactId)
		if found {
			fact.Status = types.FactStatusVerified
			fact.IsProvisional = false
			fact.VerifiedAtHeight = currentHeight
			fact.LastVerifiedHeight = currentHeight
			k.SetFact(ctx, fact)

			// Create fact references from claim references
			for _, refID := range claim.References {
				k.SetFactReference(ctx, types.FactReference{
					FromFactId:      fact.Id,
					ToFactId:        refID,
					ReferenceType:   "supports",
					Weight:          1_000_000,
					CreatedAtHeight: currentHeight,
				})
				if err := k.IncrementCitationCount(ctx, refID); err != nil {
					k.Logger(ctx).Error("failed to increment citation count",
						"fact_id", refID, "error", err)
				}
			}

			ctx.EventManager().EmitEvent(
				sdk.NewEvent("fact_verified_survival",
					sdk.NewAttribute("fact_id", fact.Id),
					sdk.NewAttribute("claim_id", claim.Id),
					sdk.NewAttribute("survived_challenges", fmt.Sprintf("%d", fact.SurvivedChallenges)),
				),
			)
		}
	} else if claim.Status == types.ClaimStatusEvaluated {
		// Score was 300k-500k, no challenge → still accepted but at lower confidence
		factID := GenerateFactID(claim.Id, fmt.Sprintf("verified:%d", currentHeight))
		fact := types.Fact{
			Id:                   factID,
			ClaimText:            claim.ClaimText,
			Domain:               claim.Domain,
			EpistemicCategory:    claim.EpistemicCategory,
			Confidence:           claim.StructuralScore,
			Status:               types.FactStatusVerified,
			Submitter:            claim.Submitter,
			SubmitterDid:         claim.SubmitterDid,
			References:           claim.References,
			VerifiedAtHeight:     currentHeight,
			LogicZone:            claim.LogicZone,
			CreatedAtHeight:      currentHeight,
			LastVerifiedHeight:   currentHeight,
			LastConfidenceUpdate: currentHeight,
			BaseConfidence:       claim.StructuralScore,
			Maturity:             types.MaturityEmerging,
			OriginalClaimId:      claim.Id,
		}
		k.SetFact(ctx, fact)
		claim.ProvisionalFactId = factID

		for _, refID := range claim.References {
			k.SetFactReference(ctx, types.FactReference{
				FromFactId:      factID,
				ToFactId:        refID,
				ReferenceType:   "supports",
				Weight:          1_000_000,
				CreatedAtHeight: currentHeight,
			})
		}
	}

	// Mark claim as accepted and return stake
	claim.Status = types.ClaimStatusAccepted
	k.SetClaim(ctx, claim)
	k.UpdateSubmitterCalibration(ctx, claim.Submitter, true)
	k.returnClaimStake(ctx, claim)

	// Create truth-linked vesting schedule
	if k.vestingRewardsKeeper != nil && claim.StakeAmount != "" && claim.ProvisionalFactId != "" {
		if err := k.vestingRewardsKeeper.CreateVestingScheduleFromKnowledge(
			ctx, claim.Id, claim.ProvisionalFactId, claim.Submitter, claim.StakeAmount, claim.EpistemicCategory,
		); err != nil {
			k.Logger(ctx).Error("failed to create vesting schedule",
				"claim_id", claim.Id, "error", err)
		}
	}

	return nil
}

// calculateChallengeWindow determines the challenge window duration based on epistemic category.
func (k Keeper) calculateChallengeWindow(category, logicZone string, score uint64, params types.Params) uint64 {
	baseDuration := params.BaseChallengeDuration
	if baseDuration == 0 {
		baseDuration = types.BaseChallengeDurationDefault
	}

	multiplier, ok := types.ChallengeWindowProfile[category]
	if !ok {
		multiplier = types.FastDecayChallengeMultiplier
	}

	duration := baseDuration * multiplier

	// Special cases from the plan
	// Propositional tautologies (score at zone ceiling): halved window
	if logicZone == "propositional" && score >= 950_000 {
		duration /= 2
	}
	// Protocol axioms: quarter window, only structural contradiction challenges
	if category == "protocol" && logicZone == "propositional" {
		duration /= 4
	}

	return duration
}

// BuildEvaluationContext creates an EvaluationContext wired from keeper methods.
// This is the keeper-side equivalent of NewEvaluationContextFromKeeper in app/.
// Defined here so the BeginBlocker can call it without importing app/.
func (k Keeper) BuildEvaluationContext() *evaluation.EvaluationContext {
	evalCtx := &evaluation.EvaluationContext{
		GetClaim:                k.GetClaim,
		GetFact:                 k.GetFact,
		GetSupportingFacts:      k.GetSupportingFacts,
		GetContradictingFacts:   k.GetContradictingFacts,
		GetSubmitterCalibration: k.GetCalibrationRecord,
	}

	if k.evidenceMgmtKeeper != nil {
		evalCtx.GetEvidenceByClaimId = k.evidenceMgmtKeeper.GetEvidenceByClaimId
	}

	if k.ontologyKeeper != nil {
		evalCtx.GetZoneConfidenceCeiling = k.ontologyKeeper.GetZoneConfidenceCeiling
		evalCtx.RequiresIncompletenessAck = k.ontologyKeeper.RequiresIncompletenessAck
		evalCtx.GetConfidenceCeiling = k.ontologyKeeper.GetConfidenceCeiling
	}

	// Wire negative knowledge lookups (Stage 8: counter-fact analysis)
	evalCtx.GetCounterFactsByFact = k.GetCounterFactsByFact
	evalCtx.GetBidirectionalConfidence = k.GetBidirectionalConfidence

	return evalCtx
}

// ---------- Provisional Challenge State Operations ----------

// SetProvisionalChallenge stores a provisional challenge.
func (k Keeper) SetProvisionalChallenge(ctx sdk.Context, challenge types.ProvisionalChallenge) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := provisionalChallengeKey(challenge.Id)
	bz, err := json.Marshal(challenge)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal provisional challenge: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set provisional challenge: %v", err))
	}
}

// GetProvisionalChallenge retrieves a provisional challenge by ID.
func (k Keeper) GetProvisionalChallenge(ctx sdk.Context, challengeID string) (types.ProvisionalChallenge, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := provisionalChallengeKey(challengeID)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return types.ProvisionalChallenge{}, false
	}
	var challenge types.ProvisionalChallenge
	if err := json.Unmarshal(bz, &challenge); err != nil {
		return types.ProvisionalChallenge{}, false
	}
	return challenge, true
}

// IterateProvisionalChallenges iterates over all provisional challenges.
func (k Keeper) IterateProvisionalChallenges(ctx sdk.Context, cb func(types.ProvisionalChallenge) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ProvisionalChallengeKeyPrefix, prefixEndBytes(types.ProvisionalChallengeKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var challenge types.ProvisionalChallenge
		if err := json.Unmarshal(iter.Value(), &challenge); err != nil {
			continue
		}
		if cb(challenge) {
			break
		}
	}
}

// GetActiveChallengesForClaim returns all open challenges for a claim.
func (k Keeper) GetActiveChallengesForClaim(ctx sdk.Context, claimId string) []types.ProvisionalChallenge {
	var challenges []types.ProvisionalChallenge
	k.IterateProvisionalChallenges(ctx, func(c types.ProvisionalChallenge) bool {
		if c.ClaimId == claimId && c.Status == types.ChallengeStatusOpen {
			challenges = append(challenges, c)
		}
		return false
	})
	return challenges
}

// SetChallengerCooldown records the last challenge height for a challenger against a claim.
func (k Keeper) SetChallengerCooldown(ctx sdk.Context, challenger, claimId string, height uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := challengerCooldownKey(challenger, claimId)
	bz := []byte(fmt.Sprintf("%d", height))
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set challenger cooldown: %v", err))
	}
}

// GetChallengerCooldown retrieves the last challenge height for a challenger against a claim.
func (k Keeper) GetChallengerCooldown(ctx sdk.Context, challenger, claimId string) (uint64, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := challengerCooldownKey(challenger, claimId)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return 0, false
	}
	var height uint64
	_, err = fmt.Sscanf(string(bz), "%d", &height)
	return height, err == nil
}

// GenerateChallengeID generates a deterministic challenge ID.
func GenerateChallengeID(claimId, challenger string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.challenge.id.v1:"))
	h.Write([]byte(claimId))
	h.Write([]byte(":"))
	h.Write([]byte(challenger))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}

// ---------- Key Construction ----------

func provisionalChallengeKey(id string) []byte {
	return append(types.ProvisionalChallengeKeyPrefix, []byte(id)...)
}

func challengerCooldownKey(challenger, claimId string) []byte {
	return append(types.ChallengerCooldownKeyPrefix, []byte(challenger+":"+claimId)...)
}
