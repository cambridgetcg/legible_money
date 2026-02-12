package evaluation

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

const (
	// noEvidencePenalty is applied to evidence-requiring categories with no
	// evidence AND no references.
	noEvidencePenalty = 200_000

	// defaultTypeStrength is used for unknown evidence types (conservative).
	defaultTypeStrength = 400_000

	// baseReliability is assigned to unaudited evidence (reliability == 0).
	baseReliability = 100_000
)

// evaluateEvidence scores the evidence quality for a claim.
// Reuses the formula from x/knowledge/keeper/confidence.go:computeEvidenceBonus.
//
// Stage 2 of the evaluation pipeline.
func evaluateEvidence(ctx sdk.Context, evalCtx *EvaluationContext, claim types.Claim) StageResult {
	result := StageResult{
		Name: "evidence_quality",
	}

	// Get evidence records (nil-safe: no evidence keeper = no evidence)
	var evidences []types.EvidenceRecord
	if evalCtx.GetEvidenceByClaimId != nil {
		evidences = evalCtx.GetEvidenceByClaimId(ctx, claim.Id)
	}

	if len(evidences) == 0 {
		// No evidence: apply penalty for evidence-requiring categories.
		// The hard evidence gate in Stage 6 will force abstain, but we
		// still apply the penalty here so the confidence score reflects
		// the weakness even for non-evidence-requiring categories.
		if evidenceRequiredCategories[claim.EpistemicCategory] {
			result.Adjustment = -int64(noEvidencePenalty)
			result.Justification = "no evidence for evidence-requiring category"
		} else {
			result.Justification = "no evidence submitted"
		}
		return result
	}

	var totalScore uint64
	var validCount uint64

	for _, ev := range evidences {
		if ev.Status == "rejected" || ev.Status == "expired" {
			continue
		}

		typeStrength, ok := types.EVIDENCE_TYPE_STRENGTH[ev.EvidenceType]
		if !ok {
			typeStrength = defaultTypeStrength
		}

		reliability := ev.Reliability
		if reliability == 0 {
			reliability = baseReliability
		}

		score := (typeStrength * reliability) / 1_000_000
		totalScore += score
		validCount++
	}

	if validCount == 0 {
		result.Justification = "all evidence rejected or expired"
		return result
	}

	bonus := totalScore / validCount
	if bonus > types.MaxEvidenceBonus {
		bonus = types.MaxEvidenceBonus
	}

	result.Adjustment = int64(bonus)
	result.Justification = "evidence bonus from valid evidence"

	return result
}
