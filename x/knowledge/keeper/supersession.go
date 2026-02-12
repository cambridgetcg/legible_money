package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// SupersedeFact marks an existing fact as superseded by a newer fact.
// The old fact's status changes to "superseded" and an "extends" reference is created.
//
// From TS state/knowledge.ts: supersession is the mechanism for knowledge evolution.
// A fact is superseded when a more accurate/complete version is verified.
func (k Keeper) SupersedeFact(ctx sdk.Context, oldFactID, newFactID string) error {
	oldFact, found := k.GetFact(ctx, oldFactID)
	if !found {
		return fmt.Errorf("%w: old fact %s", types.ErrFactNotFound, oldFactID)
	}

	newFact, found := k.GetFact(ctx, newFactID)
	if !found {
		return fmt.Errorf("%w: new fact %s", types.ErrFactNotFound, newFactID)
	}

	// Only active/verified facts can be superseded
	if oldFact.Status != types.FactStatusActive &&
		oldFact.Status != types.FactStatusVerified &&
		oldFact.Status != types.FactStatusContested {
		return fmt.Errorf("fact %s is in %s state, cannot supersede", oldFactID, oldFact.Status)
	}

	// Must be in the same domain
	if oldFact.Domain != newFact.Domain {
		return fmt.Errorf("facts must be in the same domain: %s vs %s", oldFact.Domain, newFact.Domain)
	}

	height := uint64(ctx.BlockHeight())

	// Mark old fact as superseded
	oldFact.Status = types.FactStatusSuperseded
	oldFact.SupersededBy = newFactID
	oldFact.SupersededAtHeight = height
	k.SetFact(ctx, oldFact)

	// Create "extends" reference from new fact to old fact
	k.SetFactReference(ctx, types.FactReference{
		FromFactId:      newFactID,
		ToFactId:        oldFactID,
		ReferenceType:   "extends",
		Weight:          1_000_000, // full weight
		CreatedAtHeight: height,
	})

	// Transfer citation count to new fact (knowledge continuity)
	if oldFact.CitationCount > 0 {
		newFact.CitationCount += oldFact.CitationCount
		k.SetFact(ctx, newFact)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("fact_superseded",
			sdk.NewAttribute("old_fact_id", oldFactID),
			sdk.NewAttribute("new_fact_id", newFactID),
			sdk.NewAttribute("domain", oldFact.Domain),
			sdk.NewAttribute("height", fmt.Sprintf("%d", height)),
		),
	)

	// Propagate confidence change through the citation graph
	// Facts that cited the old fact should now consider the new fact's confidence
	k.PropagateConfidenceChange(ctx, oldFactID, 0) // old fact effectively has 0 confidence now

	k.Logger(ctx).Info("fact superseded",
		"old_fact", oldFactID,
		"new_fact", newFactID,
		"domain", oldFact.Domain,
	)

	return nil
}

// GetSupersessionChain returns the chain of supersessions for a fact.
// Follows SupersededBy links until reaching a non-superseded fact.
func (k Keeper) GetSupersessionChain(ctx sdk.Context, factID string) []types.Fact {
	var chain []types.Fact
	visited := make(map[string]bool)
	currentID := factID

	for {
		if visited[currentID] {
			break // prevent infinite loops
		}
		visited[currentID] = true

		fact, found := k.GetFact(ctx, currentID)
		if !found {
			break
		}
		chain = append(chain, fact)

		if fact.SupersededBy == "" {
			break
		}
		currentID = fact.SupersededBy
	}

	return chain
}

// GetLatestVersion returns the most current version of a fact by following
// the supersession chain to its end.
func (k Keeper) GetLatestVersion(ctx sdk.Context, factID string) (types.Fact, bool) {
	chain := k.GetSupersessionChain(ctx, factID)
	if len(chain) == 0 {
		return types.Fact{}, false
	}
	return chain[len(chain)-1], true
}

// TriggerReverificationScan scans for facts needing reverification and
// automatically creates reverification rounds for them (AH-5).
// Called periodically from EndBlocker (amortized, max per block).
//
// For each decayed fact that needs reverification:
//  1. Skip if there's already an active round for this fact's claim
//  2. Create a synthetic claim for the fact's original content
//  3. Create a reverification round (Type = "reverification")
//
// Reverification rounds follow the same commit-reveal flow as verification,
// but on completion apply the category-specific ReverificationBonus if accepted.
func (k Keeper) TriggerReverificationScan(ctx sdk.Context, maxPerBlock int) int {
	facts := k.GetFactsNeedingReverification(ctx, maxPerBlock)
	triggered := 0
	maxCreated := types.MaxReverificationsPerBlock

	for _, fact := range facts {
		if triggered >= maxCreated {
			break
		}

		// Guard: skip if a round already exists for this fact's original claim
		if fact.OriginalClaimId != "" && k.HasActiveRoundForClaim(ctx, fact.OriginalClaimId) {
			continue
		}

		// Create or find the claim for reverification
		claimId := k.ensureReverificationClaim(ctx, fact)
		if claimId == "" {
			continue
		}

		// Guard: skip if a round already exists for this reverification claim
		if k.HasActiveRoundForClaim(ctx, claimId) {
			continue
		}

		// Create the reverification round
		claim, found := k.GetClaim(ctx, claimId)
		if !found {
			continue
		}

		round, err := k.CreateVerificationRound(ctx, claim)
		if err != nil {
			k.Logger(ctx).Error("failed to create reverification round",
				"fact_id", fact.Id, "error", err)
			continue
		}

		// Mark as reverification type
		round.Type = types.RoundTypeReverification
		k.SetVerificationRound(ctx, round)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("reverification_triggered",
				sdk.NewAttribute("fact_id", fact.Id),
				sdk.NewAttribute("round_id", round.Id),
				sdk.NewAttribute("domain", fact.Domain),
				sdk.NewAttribute("confidence", fmt.Sprintf("%d", fact.Confidence)),
				sdk.NewAttribute("category", fact.EpistemicCategory),
			),
		)

		k.Logger(ctx).Info("auto-reverification round created",
			"fact_id", fact.Id,
			"round_id", round.Id,
			"decayed_confidence", fact.Confidence,
		)

		triggered++
	}

	return triggered
}

// ensureReverificationClaim creates or retrieves a claim for reverification of a fact.
// If the fact's original claim still exists, reuse it. Otherwise, create a synthetic one.
func (k Keeper) ensureReverificationClaim(ctx sdk.Context, fact types.Fact) string {
	// Try original claim first
	if fact.OriginalClaimId != "" {
		if _, found := k.GetClaim(ctx, fact.OriginalClaimId); found {
			return fact.OriginalClaimId
		}
	}

	// Create synthetic reverification claim
	height := uint64(ctx.BlockHeight())
	claimId := fmt.Sprintf("reverify-%s-%d", fact.Id, height)

	claim := types.Claim{
		Id:                   claimId,
		ClaimText:            fact.ClaimText,
		Domain:               fact.Domain,
		EpistemicCategory:    fact.EpistemicCategory,
		Submitter:            fact.SubmitterDid,
		SubmitterDid:         fact.SubmitterDid,
		Status:               "pending",
		LogicZone:            fact.LogicZone,
		SubmittedAtHeight:    height,
		References:           fact.References,
		ReverificationFactId: fact.Id,
	}
	k.SetClaim(ctx, claim)

	return claimId
}

// MasterConfidence computes the final confidence for a fact using the master formula:
// C_final = min(C_max, C_base × M_temporal × M_adversarial)
//
// Where:
//   - C_max is the stratum/category ceiling
//   - C_base is the verified base confidence
//   - M_temporal is the time decay modifier (0.0 to 1.0)
//   - M_adversarial is the adversarial resilience modifier (challenges survived)
//
// All values in basis points (0-1_000_000).
func (k Keeper) MasterConfidence(ctx sdk.Context, factID string) uint64 {
	fact, found := k.GetFact(ctx, factID)
	if !found {
		return 0
	}

	currentHeight := uint64(ctx.BlockHeight())

	// C_base: original verified confidence
	cBase := fact.BaseConfidence
	if cBase == 0 {
		cBase = fact.Confidence
	}

	// M_temporal: time decay modifier
	var blocksSince uint64
	lastVerified := fact.LastVerifiedHeight
	if lastVerified == 0 {
		lastVerified = fact.VerifiedAtHeight
	}
	if currentHeight > lastVerified {
		blocksSince = currentHeight - lastVerified
	}

	decayed := CalculateDecayedConfidence(cBase, blocksSince, fact.EpistemicCategory)
	var mTemporal uint64
	if cBase > 0 {
		mTemporal = (decayed * 1_000_000) / cBase
	} else {
		mTemporal = 1_000_000
	}

	// M_adversarial: challenge survival modifier
	// Each survived challenge boosts resilience; each unsurived one reduces it.
	// Simple model: survived_ratio where VerifierCount tracks positive outcomes
	mAdversarial := uint64(1_000_000) // default 1.0
	if fact.ChallengeCount > 0 {
		// If the fact has been challenged but is still active, it survived
		// Bonus: 2% per survived challenge (up to 20%)
		bonus := uint64(fact.ChallengeCount) * 20_000
		if bonus > 200_000 {
			bonus = 200_000
		}
		mAdversarial = 1_000_000 + bonus
		if mAdversarial > 1_200_000 {
			mAdversarial = 1_200_000
		}
	}

	// C_final = C_base × M_temporal × M_adversarial / 1e12
	cFinal := (cBase * mTemporal / 1_000_000) * mAdversarial / 1_000_000

	// Apply ceiling: min(C_max, C_final)
	// Category ceiling
	if profile, ok := types.CATEGORY_DECAY[fact.EpistemicCategory]; ok {
		if cFinal > profile.MaxConfidence {
			cFinal = profile.MaxConfidence
		}
	}

	// Stratum ceiling
	cFinal = k.applyStratumCeiling(ctx, fact.Domain, cFinal)

	return cFinal
}
