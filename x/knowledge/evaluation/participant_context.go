package evaluation

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// evaluateParticipantContext checks whether the evidence chain for an
// observable-category claim includes human-grounded evidence.
//
// This stage encodes the fundamental asymmetry: agents cannot observe
// physical reality. Derivable categories (analytic, formal, protocol,
// computational) are unaffected — their truth follows from logic alone.
// Observable categories (empirical, historical, predictive, social,
// replicated) need evidence that connects to human observation.
//
// Importantly, this checks the EVIDENCE, not the SUBMITTER. An agent
// CAN submit empirical claims — it just needs to cite evidence that
// connects to human observation (experiment, oracle_feed, document,
// attestation).
//
// Stage 3 of the 9-stage evaluation pipeline (between calibration and
// evidence quality).
func evaluateParticipantContext(ctx sdk.Context, evalCtx *EvaluationContext, claim types.Claim) StageResult {
	result := StageResult{
		Name: "participant_context",
	}

	// Derivable categories: no adjustment needed.
	if types.IsDerivableCategory(claim.EpistemicCategory) {
		result.Justification = "derivable category — agent-sufficient"
		return result
	}

	// Observable category: check evidence chain for human grounding.
	var evidences []types.EvidenceRecord
	if evalCtx.GetEvidenceByClaimId != nil {
		evidences = evalCtx.GetEvidenceByClaimId(ctx, claim.Id)
	}

	// No evidence at all: defer to the evidence stage and edge_cases stage
	// which already handle missing evidence with their own penalties.
	if len(evidences) == 0 {
		result.Justification = "observable category, no evidence — deferred to evidence stage"
		return result
	}

	// Count valid evidence by domain (human-grounded vs agent-verifiable).
	var humanCount, agentCount uint64
	for _, ev := range evidences {
		if ev.Status == "rejected" || ev.Status == "expired" {
			continue
		}
		if types.IsAgentVerifiableEvidence(ev.EvidenceType) {
			agentCount++
		} else {
			humanCount++
		}
	}

	// All evidence rejected/expired: treat as no evidence.
	if humanCount == 0 && agentCount == 0 {
		result.Justification = "observable category, all evidence rejected/expired — deferred"
		return result
	}

	ep := evalCtx.ExtendedParams
	if humanCount > 0 {
		// Observable claim with human-grounded evidence: small bonus.
		bonus := humanCount * ep.HumanEvidenceBonusBps
		if bonus > ep.MaxHumanEvidenceBonusBps {
			bonus = ep.MaxHumanEvidenceBonusBps
		}
		result.Adjustment = int64(bonus)
		result.Justification = fmt.Sprintf(
			"observable category with %d human-grounded evidence item(s) — +%d bps bonus",
			humanCount, bonus,
		)
		return result
	}

	// Observable claim with ONLY agent-verifiable evidence: penalty + ceiling.
	result.Adjustment = -int64(ep.HumanEvidenceGapPenaltyBps)
	result.Ceiling = ep.AgentOnlyEmpiricalCeiling
	result.Justification = fmt.Sprintf(
		"observable category with only agent-verifiable evidence (%d items) — %d bps penalty, %d ceiling",
		agentCount, ep.HumanEvidenceGapPenaltyBps, ep.AgentOnlyEmpiricalCeiling,
	)
	return result
}
