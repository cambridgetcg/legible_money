package evaluation

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// axiomConfidenceThreshold is the confidence level above which a fact is
// considered axiomatic (foundational).
const axiomConfidenceThreshold = 900_000

// substantiveEvidenceThreshold is the minimum evidence type strength that
// counts as "substantive" for the hard evidence gate. Attestation alone
// (500,000) is insufficient — agents can attest to hallucinated facts.
// Requires on_chain, computation, experiment, oracle_feed, or document.
const substantiveEvidenceThreshold = 500_000

// evaluateEdgeCases checks for conditions that force a verdict override.
//
// Stage 6 of the evaluation pipeline. Checked last, can override everything.
//
// Edge cases:
//  1. Self-referential claim (content hash matches a referenced fact ID)
//  2. Axiom contradiction (referenced fact is disproven but was axiomatic)
//  3. No evidence AND no references for evidence-requiring categories
//  4. Empty claim text
func evaluateEdgeCases(ctx sdk.Context, evalCtx *EvaluationContext, claim types.Claim) StageResult {
	result := StageResult{
		Name: "edge_cases",
	}

	// 1. Empty claim text → reject
	if len(claim.ClaimText) == 0 {
		result.ForceVerdict = "reject"
		result.Justification = "empty claim text"
		return result
	}

	// 2. Self-referential claim: content hash matches a referenced fact ID
	if claim.ContentHash != "" {
		for _, refId := range claim.References {
			if refId == claim.ContentHash {
				result.ForceVerdict = "abstain"
				result.Justification = "self-referential claim: content hash matches referenced fact ID"
				return result
			}
		}
	}

	// 3. Axiom contradiction: referenced fact is disproven but was high-confidence
	if evalCtx.GetFact != nil {
		for _, refId := range claim.References {
			fact, found := evalCtx.GetFact(ctx, refId)
			if !found {
				continue
			}
			if (fact.Status == types.FactStatusDisproven || fact.Status == types.FactStatusRevoked) &&
				fact.BaseConfidence >= axiomConfidenceThreshold {
				result.ForceVerdict = "reject"
				result.Justification = "axiom contradiction: referenced disproven fact was axiomatic (base confidence >= 900,000)"
				return result
			}
		}
	}

	// 4. Hard evidence gate: evidence-requiring categories MUST have at least
	// one substantive evidence piece (strength > attestation). References
	// provide a bonus in Stage 3 but do NOT substitute for evidence.
	// This prevents hallucinated claims that cite real facts about the right
	// topic but make false assertions about what those facts mean.
	if evidenceRequiredCategories[claim.EpistemicCategory] {
		hasSubstantiveEvidence := false
		if evalCtx.GetEvidenceByClaimId != nil {
			evidences := evalCtx.GetEvidenceByClaimId(ctx, claim.Id)
			for _, ev := range evidences {
				if ev.Status == "rejected" || ev.Status == "expired" {
					continue
				}
				strength, ok := evalCtx.ExtendedParams.GetEvidenceTypeStrength(ev.EvidenceType)
				if !ok {
					continue
				}
				if strength > substantiveEvidenceThreshold {
					hasSubstantiveEvidence = true
					break
				}
			}
		}
		if !hasSubstantiveEvidence {
			result.ForceVerdict = "abstain"
			result.Justification = "evidence-requiring category lacks substantive evidence (attestation alone insufficient)"
			return result
		}
	}

	result.Justification = "no edge cases triggered"
	return result
}
