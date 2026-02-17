package evaluation

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// EvaluationResult is the output of the claim evaluation pipeline.
// All validators produce identical results for the same inputs (deterministic).
type EvaluationResult struct {
	Verdict       string        `json:"verdict"`       // "accept", "reject", "abstain"
	Confidence    uint64        `json:"confidence"`    // 0-1,000,000 basis points
	Justification string        `json:"justification"` // human-readable reason
	Stages        []StageResult `json:"stages"`        // per-stage audit trail
}

// StageResult records the output of one evaluation stage for auditability.
type StageResult struct {
	Name          string `json:"name"`
	Adjustment    int64  `json:"adjustment"`    // signed delta to confidence
	Ceiling       uint64 `json:"ceiling"`       // ceiling applied (0 = none)
	ForceVerdict  string `json:"force_verdict"` // "reject", "abstain", or "" (no override)
	Justification string `json:"justification"`
}

// EvaluationContext provides access to on-chain state for claim evaluation.
// All functions are wired from keeper methods, making the engine testable
// with mock implementations.
type EvaluationContext struct {
	GetClaim                  func(ctx sdk.Context, claimId string) (types.Claim, bool)
	GetFact                   func(ctx sdk.Context, factId string) (types.Fact, bool)
	GetSupportingFacts        func(ctx sdk.Context, factId string) []types.FactReference
	GetContradictingFacts     func(ctx sdk.Context, factId string) []types.FactReference
	GetEvidenceByClaimId      func(ctx sdk.Context, claimId string) []types.EvidenceRecord
	GetZoneConfidenceCeiling  func(ctx sdk.Context, zone string) uint64
	RequiresIncompletenessAck func(ctx sdk.Context, zone string) bool
	GetConfidenceCeiling      func(ctx sdk.Context, domain string) (maxConfidence uint64, decayRate uint64, err error)
	GetSubmitterCalibration   func(ctx sdk.Context, submitter string) (types.CalibrationRecord, bool)

	// Negative knowledge lookups (Stage 8)
	GetCounterFactsByFact      func(ctx sdk.Context, factId string) []types.CounterFact
	GetBidirectionalConfidence func(ctx sdk.Context, factId string) types.BidirectionalConfidence

	// Novelty assessment lookups (Stage 5.5)
	GetFactCountByDomain func(ctx sdk.Context, domain string) uint64
	GetDomainCount       func(ctx sdk.Context) uint64
}

// OntologyToKnowledgeScale converts the ontology module's basis point scale
// to the knowledge module's scale. Both now use 0-1,000,000, so the factor is 1.
const OntologyToKnowledgeScale = 1

// Verdict thresholds aligned with existing knowledge module constants.
const (
	AcceptThreshold = types.ReverificationThreshold // 600,000 (60%)
	RejectThreshold = types.MinConfidenceThreshold  // 300,000 (30%)
)

// Categories that require evidence or references to be credible.
var evidenceRequiredCategories = map[string]bool{
	"empirical":  true,
	"historical": true,
	"predictive": true,
	"social":     true,
	"replicated": true,
}
