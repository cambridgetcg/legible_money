package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Fact status constants (10-state lifecycle from TS reference + adversarial verification)
const (
	FactStatusPending     = "pending"
	FactStatusProvisional = "provisional" // survives structural eval, in challenge window
	FactStatusVerified    = "verified"
	FactStatusActive      = "active"
	FactStatusContested   = "contested"
	FactStatusChallenged  = "challenged"
	FactStatusSuperseded  = "superseded"
	FactStatusExpired     = "expired"
	FactStatusDisproven   = "disproven"
	FactStatusRevoked     = "revoked"
	FactStatusAtRisk      = "at_risk"
	FactStatusPruned      = "pruned"
)

// Claim status constants (adversarial verification lifecycle)
const (
	ClaimStatusPending           = "pending"
	ClaimStatusPendingEvaluation = "pending_evaluation" // submitted, awaiting structural eval
	ClaimStatusEvaluated         = "evaluated"          // scored 300k-500k, in challenge window, no provisional fact
	ClaimStatusProvisional       = "provisional"        // scored ≥500k, provisional fact exists, in challenge window
	ClaimStatusInVerification    = "in_verification"    // legacy: in commit-reveal round
	ClaimStatusAccepted          = "accepted"
	ClaimStatusRejected          = "rejected"
	ClaimStatusChallenged        = "challenged"
	ClaimStatusExpired           = "expired"
	ClaimStatusInsufficient      = "insufficient"
	ClaimStatusContested         = "contested"
)

// Fact maturity levels
const (
	MaturityEmerging    = "emerging"
	MaturityEstablished = "established"
	MaturityCanonical   = "canonical"
)

// CategoryDecayProfile defines per-category confidence decay parameters.
// Half-life in blocks (0 = eternal, no decay). From TS EPISTEMIC_PROFILES.
type CategoryDecayProfile struct {
	HalfLifeBlocks         uint64 // 0 = eternal (analytic, formal, protocol)
	MaxConfidence          uint64 // stratum ceiling (basis points, 0-1_000_000)
	MinConfidence          uint64 // floor after decay
	ReverificationBonus    uint64 // bonus on successful re-verification
	RequiresReverification bool
}

// CATEGORY_DECAY maps epistemic category to decay profile.
// Values from TS core/knowledge/types.ts EPISTEMIC_PROFILES.
// Block time: 2521ms → BLOCKS_PER_DAY = 34272
var CATEGORY_DECAY = map[string]CategoryDecayProfile{
	"analytic":      {HalfLifeBlocks: 0, MaxConfidence: 1_000_000, MinConfidence: 0, ReverificationBonus: 0, RequiresReverification: false},
	"formal":        {HalfLifeBlocks: 0, MaxConfidence: 1_000_000, MinConfidence: 0, ReverificationBonus: 0, RequiresReverification: false},
	"protocol":      {HalfLifeBlocks: 0, MaxConfidence: 1_000_000, MinConfidence: 0, ReverificationBonus: 0, RequiresReverification: false},
	"empirical":     {HalfLifeBlocks: 1_000_000, MaxConfidence: 990_000, MinConfidence: 330_000, ReverificationBonus: 110_000, RequiresReverification: true},
	"historical":    {HalfLifeBlocks: 500_000, MaxConfidence: 950_000, MinConfidence: 200_000, ReverificationBonus: 30_000, RequiresReverification: true},
	"predictive":    {HalfLifeBlocks: 20_000, MaxConfidence: 900_000, MinConfidence: 100_000, ReverificationBonus: 80_000, RequiresReverification: true},
	"social":        {HalfLifeBlocks: 150_000, MaxConfidence: 850_000, MinConfidence: 200_000, ReverificationBonus: 40_000, RequiresReverification: true},
	"replicated":    {HalfLifeBlocks: 1_000_000, MaxConfidence: 995_000, MinConfidence: 300_000, ReverificationBonus: 20_000, RequiresReverification: true},
	"computational": {HalfLifeBlocks: 0, MaxConfidence: 1_000_000, MinConfidence: 0, ReverificationBonus: 0, RequiresReverification: false},
}

// Stratum confidence ceilings (from TS ontology/types.ts STRATUM_CONFIDENCE_CEILING)
var STRATUM_CONFIDENCE_CEILING = map[string]uint64{
	"fundamental":   1_000_000,
	"physical":      990_000,
	"chemical":      980_000,
	"biological":    950_000,
	"cognitive":     900_000,
	"social":        850_000,
	"technological": 1_000_000,
}

// Citation economics constants (from TS core/knowledge/types.ts)
const (
	CitationShareBPS         = 150_000   // 15% of verification rewards to citation pool
	CitationDecayPerLevel    = 500_000   // 50% decay per ancestor level
	CitationProtocolFloorBPS = 20_000    // 2% protocol floor
	SelfCitationDiscountBPS  = 500_000   // 50% discount when citing own facts (FARM-3 fix)
	CrossDomainBonusRate     = 200_000   // 20% cross-domain bonus
	MaxCrossDomainBonus      = 2_000_000 // 2.0x max

	// Fundamentality thresholds
	FundamentalityNotable      = 250_000 // 0.25
	FundamentalityFoundational = 500_000 // 0.50
	FundamentalityFundamental  = 750_000 // 0.75
	FundamentalityAxiomatic    = 900_000 // 0.90

	// Fundamentality reward multipliers
	FundamentalityBonusNotable      = 1_100_000 // 1.1x
	FundamentalityBonusFoundational = 1_250_000 // 1.25x
	FundamentalityBonusFundamental  = 1_500_000 // 1.5x
	FundamentalityBonusAxiomatic    = 2_000_000 // 2.0x
)

// Confidence propagation constants
const (
	PropagationThreshold = 10_000  // 1% change triggers propagation (basis points)
	MaxPropagationDepth  = 10      // max BFS hops
	CyclePenaltyFactor   = 400_000 // 0.4× penalty for cycle members
)

// Reverification constants
const (
	ReverificationThreshold = 600_000 // 60% confidence floor
	StalenessMultiplier     = 2       // age > 2× halfLife → needs reverification
	MinConfidenceThreshold  = 300_000 // 30% — below this, fact expires
)

// Minimum claim stake to prevent zero-slash spam.
// At 22% (220k bps) slash rate, stakes below 5 ulgm produce a 0 slash due to integer
// truncation. This floor ensures every rejected claim incurs a meaningful penalty.
// (OpenClaw C-3 fix)
const MinClaimStakeUlgm = 1000 // 0.001 LGM — floor below which claims are rejected

// Adversarial verification constants
const (
	BlocksPerDay = 34_272 // 2521ms block time → 34,272 blocks/day

	// Structural evaluation thresholds
	ProvisionalThresholdDefault = 500_000 // 50% — min score to create provisional fact
	RejectThresholdDefault      = 300_000 // 30% — below this, claim is rejected

	// Challenge window durations (in blocks)
	BaseChallengeDurationDefault = BlocksPerDay // ~1 day
	EternalChallengeMultiplier   = 3            // 3 days for eternal categories
	SlowDecayChallengeMultiplier = 2            // 2 days for slow-decay categories
	FastDecayChallengeMultiplier = 1            // 1 day for fast-decay categories

	// Challenge economics
	MinChallengeStakeDefault            = 11_000_000 // 11 LGM in ulgm
	FailedChallengeSlashBpsDefault      = 220_000    // 22% slash on failed challenge
	SuccessfulChallengeRewardBpsDefault = 300_000    // 30% of submitter's slashed stake to challenger
	MaxConcurrentChallenges             = 3          // max 3 concurrent challenges per claim

	// Confidence growth
	ConfidenceGrowthPerEpochDefault = 11_000  // 1.1% per epoch
	MaxSurvivalConfidenceDefault    = 770_000 // 77% cap for unchallenged facts
	SurvivedChallengeConfidenceCap  = 880_000 // 88% cap after surviving challenge
	ConfidenceGrowthEpoch           = 1_111   // growth applied every 1,111 blocks (~47 min)

	// Challenge cooldown (anti-spam)
	BaseCooldownBlocks = 17_136  // ~0.5 day
	MaxCooldownBlocks  = 171_360 // ~5 days

	// Verification round types
	RoundTypeVerification   = "verification"   // legacy commit-reveal
	RoundTypeChallenge      = "challenge"      // arbiter round for Path B challenge
	RoundTypeReverification = "reverification" // AH-5: auto-triggered by confidence decay

	// MaxReverificationsPerBlock limits how many reverification rounds can be created
	// per EndBlocker call to prevent block processing spikes.
	MaxReverificationsPerBlock = 5
)

// ChallengeWindowProfile maps epistemic category to challenge window multiplier.
// Eternal categories: analytic, formal, protocol, computational (3-day window)
// Slow decay: historical, replicated (2-day window)
// Fast decay: empirical, social, predictive (1-day window)
var ChallengeWindowProfile = map[string]uint64{
	"analytic":      EternalChallengeMultiplier,
	"formal":        EternalChallengeMultiplier,
	"protocol":      EternalChallengeMultiplier,
	"computational": EternalChallengeMultiplier,
	"historical":    SlowDecayChallengeMultiplier,
	"replicated":    SlowDecayChallengeMultiplier,
	"empirical":     FastDecayChallengeMultiplier,
	"social":        FastDecayChallengeMultiplier,
	"predictive":    FastDecayChallengeMultiplier,
}

// IsDeterministicCategory returns true for categories where challenge resolution
// is fully deterministic (Path A). These categories can be verified by computation.
func IsDeterministicCategory(category string) bool {
	switch category {
	case "protocol", "computational", "formal", "analytic":
		return true
	}
	return false
}

// ============ Epistemic Domains ============

// EpistemicDomain classifies whether a category's truth can be derived
// purely from computation/logic (agent-sufficient) or requires observation
// of the physical world (reality-dependent, needing human involvement).
type EpistemicDomain string

const (
	EpistemicDomainDerivable  EpistemicDomain = "derivable"  // agent-sufficient: truth from logic/computation
	EpistemicDomainObservable EpistemicDomain = "observable" // reality-dependent: truth requires observation
)

// CategoryEpistemicDomain maps each epistemic category to its domain.
// Derivable: categories where truth follows from axioms, definitions, or computation.
// Observable: categories where truth depends on physical reality.
var CategoryEpistemicDomain = map[string]EpistemicDomain{
	"analytic":      EpistemicDomainDerivable,
	"formal":        EpistemicDomainDerivable,
	"protocol":      EpistemicDomainDerivable,
	"computational": EpistemicDomainDerivable,
	"empirical":     EpistemicDomainObservable,
	"historical":    EpistemicDomainObservable,
	"predictive":    EpistemicDomainObservable,
	"social":        EpistemicDomainObservable,
	"replicated":    EpistemicDomainObservable,
}

// IsDerivableCategory returns true for categories where truth is derivable
// from logic, axioms, or computation alone — no physical observation needed.
func IsDerivableCategory(category string) bool {
	domain, ok := CategoryEpistemicDomain[category]
	return ok && domain == EpistemicDomainDerivable
}

// IsAgentVerifiableEvidence returns true for evidence types that an AI agent
// can verify without human involvement: on-chain data and deterministic computation.
func IsAgentVerifiableEvidence(evidenceType string) bool {
	switch evidenceType {
	case "on_chain", "computation":
		return true
	}
	return false
}

// RequiresHumanInvolvement returns true for evidence types that connect to
// physical observation — experiment results, oracle feeds, documents, attestations.
// These evidence types carry implicit human grounding.
func RequiresHumanInvolvement(evidenceType string) bool {
	return !IsAgentVerifiableEvidence(evidenceType)
}

// Participant context constants (governance defaults)
const (
	// HumanEvidenceGapPenaltyBpsDefault is the confidence penalty applied when an
	// observable-category claim has ONLY agent-verifiable evidence (no human grounding).
	HumanEvidenceGapPenaltyBpsDefault = 100_000 // 10%

	// AgentOnlyEmpiricalCeilingDefault is the confidence ceiling for observable-category
	// claims backed solely by agent-verifiable evidence.
	AgentOnlyEmpiricalCeilingDefault = 700_000 // 70%

	// HumanEvidenceBonusBpsDefault is the small bonus for observable claims that DO
	// include human-sourced evidence, rewarding proper grounding. Per human evidence item.
	HumanEvidenceBonusBpsDefault = 10_000 // 1% per item

	// MaxHumanEvidenceBonusBpsDefault caps the total human evidence bonus.
	MaxHumanEvidenceBonusBpsDefault = 30_000 // 3% max
)

// Evidence confidence bonus constants.
// Evidence bonus is ADDITIVE to the verification confidence, capped at MaxEvidenceBonus.
const (
	MaxEvidenceBonus = 150_000 // 15% maximum additive evidence bonus (basis points)
)

// EVIDENCE_TYPE_STRENGTH maps evidence types to intrinsic strength values.
// Values are in basis points (0-1,000,000), translated from TS EVIDENCE_TYPE_STRENGTH.
var EVIDENCE_TYPE_STRENGTH = map[string]uint64{
	"on_chain":    1_000_000, // 1.0 — highest: immutable on-chain data
	"computation": 950_000,   // 0.95 — deterministic computational verification
	"experiment":  850_000,   // 0.85 — experimental/peer-reviewed
	"oracle_feed": 700_000,   // 0.70 — external oracle data
	"document":    600_000,   // 0.60 — archived documents
	"attestation": 500_000,   // 0.50 — human/agent attestation
}

// Maturity criteria
const (
	MaturityEstablishedMinConf          = 800_000 // 0.8 confidence
	MaturityEstablishedMinAge           = 34_272  // ~1 day in blocks
	MaturityEstablishedMinVerifications = 3
	MaturityCanonicalMinConf            = 900_000 // 0.9 confidence
	MaturityCanonicalMinAge             = 342_720 // ~10 days in blocks
	MaturityCanonicalMinVerifications   = 10
)

// EquivocationEvidence records proof that a validator submitted conflicting
// commitments or reveals in the same verification round.
// This is the most serious PoT protocol violation — 100% stake slash.
type EquivocationEvidence struct {
	RoundId          string `json:"round_id"`
	Validator        string `json:"validator"`
	Type             string `json:"type"`              // "commitment" or "reveal"
	OriginalValue    string `json:"original_value"`    // first commitment hash or reveal verdict
	ConflictingValue string `json:"conflicting_value"` // equivocating commitment hash or reveal verdict
	DetectedAtHeight uint64 `json:"detected_at_height"`
	SlashedAmount    string `json:"slashed_amount"` // amount slashed (string for bigint)
}

// MustMarshal marshals params to bytes, panics on error
func (p Params) MustMarshal() []byte {
	// Placeholder - in real implementation would use amino/proto encoding
	return []byte{}
}

// MustUnmarshal unmarshals params from bytes, panics on error
func (p *Params) MustUnmarshal(bz []byte) {
	// Placeholder - in real implementation would use amino/proto encoding
}

// --- ValidateBasic and GetSigners for proto-generated Msg types ---

func (msg *MsgProposeDomain) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg *MsgProposeDomain) ValidateBasic() error {
	if msg.Name == "" {
		return fmt.Errorf("domain name cannot be empty")
	}
	if msg.Proposer == "" {
		return fmt.Errorf("proposer cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	return nil
}

func (msg *MsgEndorseDomainProposal) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Endorser)
	return []sdk.AccAddress{addr}
}

func (msg *MsgEndorseDomainProposal) ValidateBasic() error {
	if msg.DomainName == "" {
		return fmt.Errorf("domain name cannot be empty")
	}
	if msg.Endorser == "" {
		return fmt.Errorf("endorser cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Endorser); err != nil {
		return fmt.Errorf("invalid endorser address: %w", err)
	}
	return nil
}

func (msg *MsgChallengeDomainProposal) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Challenger)
	return []sdk.AccAddress{addr}
}

func (msg *MsgChallengeDomainProposal) ValidateBasic() error {
	if msg.DomainName == "" {
		return fmt.Errorf("domain name cannot be empty")
	}
	if msg.Challenger == "" {
		return fmt.Errorf("challenger cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	return nil
}

func (msg *MsgRegisterStratum) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Registrar)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterStratum) ValidateBasic() error {
	if msg.StratumName == "" {
		return fmt.Errorf("stratum name cannot be empty")
	}
	if msg.Registrar == "" {
		return fmt.Errorf("registrar cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Registrar); err != nil {
		return fmt.Errorf("invalid registrar address: %w", err)
	}
	return nil
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return nil
}

func (msg *MsgSubmitClaim) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitClaim) ValidateBasic() error {
	if msg.ClaimText == "" {
		return fmt.Errorf("claim text cannot be empty")
	}
	if msg.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if msg.Submitter == "" {
		return fmt.Errorf("submitter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	return nil
}

func (msg *MsgSubmitCommitment) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Validator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitCommitment) ValidateBasic() error {
	if msg.RoundId == "" {
		return fmt.Errorf("round ID cannot be empty")
	}
	if msg.Validator == "" {
		return fmt.Errorf("validator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Validator); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if msg.CommitmentHash == "" {
		return fmt.Errorf("commitment hash cannot be empty")
	}
	return nil
}

func (msg *MsgSubmitReveal) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Validator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitReveal) ValidateBasic() error {
	if msg.RoundId == "" {
		return fmt.Errorf("round ID cannot be empty")
	}
	if msg.Validator == "" {
		return fmt.Errorf("validator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Validator); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if msg.Verdict == "" {
		return fmt.Errorf("verdict cannot be empty")
	}
	if msg.Verdict != "accept" && msg.Verdict != "reject" && msg.Verdict != "abstain" {
		return fmt.Errorf("verdict must be accept, reject, or abstain")
	}
	return nil
}

func (msg *MsgChallengeFact) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Challenger)
	return []sdk.AccAddress{addr}
}

func (msg *MsgChallengeFact) ValidateBasic() error {
	if msg.FactId == "" {
		return fmt.Errorf("fact ID cannot be empty")
	}
	if msg.Challenger == "" {
		return fmt.Errorf("challenger cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	return nil
}

// MsgChallengeProvisionalFact challenges a provisionally accepted fact during its challenge window.
// This is the core adversarial verification message — any agent can stake counter-evidence.
// Not in proto — hand-written message type for adversarial verification.
type MsgChallengeProvisionalFact struct {
	Challenger   string   `json:"challenger"`    // bech32
	ClaimId      string   `json:"claim_id"`      // target claim
	FactId       string   `json:"fact_id"`       // provisional fact (if exists)
	Stake        string   `json:"stake"`         // challenge bond
	Reason       string   `json:"reason"`        // human-readable
	EvidenceIds  []string `json:"evidence_ids"`  // counter-evidence from evidence_mgmt
	CounterClaim string   `json:"counter_claim"` // optional: what challenger believes instead
}

func (msg *MsgChallengeProvisionalFact) Reset()         {}
func (msg *MsgChallengeProvisionalFact) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgChallengeProvisionalFact) ProtoMessage()  {}

func (msg *MsgChallengeProvisionalFact) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Challenger)
	return []sdk.AccAddress{addr}
}

func (msg *MsgChallengeProvisionalFact) ValidateBasic() error {
	if msg.Challenger == "" {
		return fmt.Errorf("challenger cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if msg.ClaimId == "" {
		return fmt.Errorf("claim ID cannot be empty")
	}
	if msg.Stake == "" {
		return fmt.Errorf("stake cannot be empty")
	}
	if msg.Reason == "" {
		return fmt.Errorf("reason cannot be empty")
	}
	return nil
}

type MsgChallengeProvisionalFactResponse struct {
	ChallengeId string `json:"challenge_id"`
	DisputeId   string `json:"dispute_id,omitempty"` // set for Path B (arbiter resolution)
}

func (msg *MsgChallengeProvisionalFactResponse) Reset()         {}
func (msg *MsgChallengeProvisionalFactResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgChallengeProvisionalFactResponse) ProtoMessage()  {}

func (msg *MsgAddFact) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAddFact) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.ClaimText == "" {
		return fmt.Errorf("claim text cannot be empty")
	}
	return nil
}

func (msg *MsgSubmitContradiction) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitContradiction) ValidateBasic() error {
	if msg.TargetFactId == "" {
		return fmt.Errorf("target fact ID cannot be empty")
	}
	if msg.CounterClaim == "" {
		return fmt.Errorf("counter-claim text cannot be empty")
	}
	if msg.Submitter == "" {
		return fmt.Errorf("submitter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if msg.Stake == "" {
		return fmt.Errorf("stake cannot be empty")
	}
	if len(msg.EvidenceIds) == 0 {
		return fmt.Errorf("at least one evidence ID required for contradiction")
	}
	return nil
}

// ProvisionalChallenge tracks a challenge against a provisional fact.
// Not in proto — hand-written type for adversarial verification.
type ProvisionalChallenge struct {
	Id               string   `json:"id"`
	ClaimId          string   `json:"claim_id"`
	FactId           string   `json:"fact_id"`
	Challenger       string   `json:"challenger"` // bech32
	Stake            string   `json:"stake"`      // challenge bond
	Reason           string   `json:"reason"`
	EvidenceIds      []string `json:"evidence_ids"` // counter-evidence
	CounterClaim     string   `json:"counter_claim"`
	Status           string   `json:"status"`          // open, resolved_upheld, resolved_overturned, resolved_weakened, resolved_inconclusive
	ResolutionPath   string   `json:"resolution_path"` // "deterministic" (Path A) or "arbiter" (Path B)
	DisputeId        string   `json:"dispute_id"`      // for Path B arbiter resolution
	CreatedAtHeight  uint64   `json:"created_at_height"`
	ResolvedAtHeight uint64   `json:"resolved_at_height,omitempty"`
	Outcome          string   `json:"outcome,omitempty"` // upheld, overturned, weakened, inconclusive, undecidability
	AttemptNumber    uint32   `json:"attempt_number"`    // for exponential cooldown
}

// Challenge outcome constants
const (
	ChallengeOutcomeUpheld         = "upheld"         // claim survives challenge
	ChallengeOutcomeOverturned     = "overturned"     // claim falls
	ChallengeOutcomeWeakened       = "weakened"       // partial success
	ChallengeOutcomeInconclusive   = "inconclusive"   // no clear winner
	ChallengeOutcomeUndecidability = "undecidability" // Gödel zones — provably unprovable
)

// Challenge status constants
const (
	ChallengeStatusOpen     = "open"
	ChallengeStatusResolved = "resolved"
)

// CalibrationRecord tracks a submitter's historical claim acceptance rate.
// Used by the evaluation pipeline (Stage 1.5) to penalize hallucination-prone
// agents and reward reliable submitters.
// Not in proto — hand-written type for evaluation pipeline.
type CalibrationRecord struct {
	Submitter        string `json:"submitter"`          // bech32 address
	TotalClaims      uint64 `json:"total_claims"`       // total finalized claims
	AcceptedClaims   uint64 `json:"accepted_claims"`    // claims accepted
	RejectedClaims   uint64 `json:"rejected_claims"`    // claims rejected
	LastUpdateHeight uint64 `json:"last_update_height"` // block of last update
}

// Calibration constants
const (
	// MinCalibrationSamples is the minimum number of finalized claims before
	// calibration adjustments are applied. Below this, the submitter is "unrated".
	MinCalibrationSamples = 5

	// CalibrationNeutralRate is the acceptance rate at which no adjustment is
	// applied (70% acceptance = neutral). In basis points (0-1,000,000).
	CalibrationNeutralRate = 700_000

	// MaxCalibrationBonus is the maximum confidence bonus for high-acceptance submitters.
	MaxCalibrationBonus = 50_000 // +5%

	// MaxCalibrationPenalty is the maximum confidence penalty for low-acceptance submitters.
	MaxCalibrationPenalty = 150_000 // -15%

	// CalibrationTrivialThreshold is the acceptance rate above which claims are considered
	// "suspiciously trivial." Agents with >95% acceptance likely submit obvious truths
	// to farm vesting rewards. FARM-2 remediation.
	CalibrationTrivialThreshold = 950_000 // 95%

	// MaxCalibrationTrivialPenalty is the maximum penalty for suspiciously high acceptance.
	// Applied when acceptance rate exceeds CalibrationTrivialThreshold.
	MaxCalibrationTrivialPenalty = 100_000 // -10%

	// MisbehaviorRejectionThreshold is the rejection rate above which ALL vesting
	// schedules for an agent are paused. Prevents the "grief-and-extract" pattern
	// (FARM-6) where agents build vesting through honest work then misbehave.
	// Value: 300,000 = 30% rejection rate.
	MisbehaviorRejectionThreshold = 300_000

	// MisbehaviorMinSamples is the minimum number of finalized claims before
	// misbehavior vesting pause is evaluated. Prevents false positives on new agents.
	MisbehaviorMinSamples = 10
)

// AH-4: Verifier method diversity constants.
// These detect and penalize suspiciously uniform verification results,
// which may indicate coordinated AI-only verification without genuine diversity.
const (
	// MinDiversityParticipants is the minimum number of reveals before
	// uniformity detection is applied. Below this, sample size is too small.
	MinDiversityParticipants = 3

	// UniformityPenaltyBps is the confidence penalty (basis points) applied when
	// ALL reveals have identical confidence values, suggesting homogeneous methods.
	UniformityPenaltyBps uint64 = 100_000 // 10% penalty

	// SingleMethodPenaltyBps is the additional penalty when all reveals declare
	// the same non-empty verification method (or all omit it).
	SingleMethodPenaltyBps uint64 = 50_000 // 5% additional penalty

	// DiversityBonusBps is a small bonus when 3+ distinct methods are declared,
	// rewarding genuine methodological diversity.
	DiversityBonusBps uint64 = 30_000 // 3% bonus
)

// ============ Negative Knowledge System ============

// Counter-fact negation types
const (
	NegationTypeRejection     = "rejection"     // Claim evaluated and rejected by validators
	NegationTypeContradiction = "contradiction" // Explicit counter-claim with evidence
	NegationTypeFalsification = "falsification" // Research outcome = "refutes"
	NegationTypeSupersession  = "supersession"  // Newer evidence overturns prior acceptance
)

// Counter-fact lifecycle statuses
const (
	CounterFactStatusPending     = "pending"     // Awaiting verification
	CounterFactStatusProvisional = "provisional" // In challenge window
	CounterFactStatusVerified    = "verified"    // Confirmed false
	CounterFactStatusContested   = "contested"   // Mixed verification
	CounterFactStatusOverturned  = "overturned"  // Counter-fact itself was disproved
)

// ComputeBidirectionalStatus derives the epistemic status from net confidence.
func ComputeBidirectionalStatus(net int64) string {
	if net > 600_000 {
		return "established"
	}
	if net < -600_000 {
		return "disproven"
	}
	if net > -200_000 && net < 200_000 {
		return "uncertain"
	}
	return "contested"
}

// Negative knowledge constants
const (
	// MaxContradictionsPerAgentPerEpoch limits contradiction spam.
	MaxContradictionsPerAgentPerEpoch = 3

	// MinContradictionStakeRatio is the minimum stake for contradiction as a
	// fraction of the target fact's original stake (basis points).
	MinContradictionStakeRatio = 500_000 // 50%

	// FailedContradictionSlashBps is the slash applied when a submitted
	// contradiction is rejected (the target fact was actually true).
	// Set to 33% (reduced from 50% per R-1) to reduce asymmetry with
	// the 22% claim rejection slash while still deterring frivolous contradictions.
	FailedContradictionSlashBps = 330_000 // 33%

	// FalsificationRewardCapBps caps the reward for disproving a fact
	// at 3× the original claim's stake.
	FalsificationRewardCapBps = 3_000_000

	// FalsificationPoolShareBps is the max share of epoch block reward pool
	// that can go to falsification rewards.
	FalsificationPoolShareBps = 150_000 // 15%

	// CounterFactHighConfidence is the threshold above which a counter-fact
	// significantly penalizes new claims referencing the target fact.
	CounterFactHighConfidence = 700_000

	// CounterFactMediumConfidence threshold for moderate penalty.
	CounterFactMediumConfidence = 400_000

	// Evaluation stage 8 penalties for counter-facts
	CounterFactHighPenalty   = 80_000
	CounterFactMediumPenalty = 40_000
	CounterFactLowPenalty    = 10_000
	MaxCounterFactPenalty    = 400_000

	// EstablishedFactChallengePenalty when contradicting a high-confidence fact.
	EstablishedFactChallengePenalty = 100_000

	// WeakFactChallengeBoost when contradicting a low-confidence fact.
	WeakFactChallengeBoost = 50_000

	// AlreadyDisprovenPenalty when submitting a claim already known to be false.
	AlreadyDisprovenPenalty = 200_000

	// RoundTypeContradiction is the round type for contradiction verification.
	RoundTypeContradiction = "contradiction"

	// Counter-fact challenge window constants (R-2)
	// Counter-facts must survive a challenge window before being finalized as "verified".
	// This creates symmetry: both positive and negative knowledge face adversarial scrutiny.
	CounterFactChallengeWindowBlocks = 34_272  // ~1 day (matches BLOCKS_PER_DAY)
	CounterFactChallengeMinStake     = 50_000  // Minimum stake to challenge a counter-fact (50k ulgm)
	CounterFactChallengeSlashBps     = 220_000 // 22% slash on failed counter-fact challenge (same as claim challenges)

	// Counter-fact challenge statuses
	CounterFactChallengeStatusOpen     = "open"
	CounterFactChallengeStatusUpheld   = "upheld"   // Counter-fact survives challenge
	CounterFactChallengeStatusReverted = "reverted" // Counter-fact overturned by challenge

	// Stale-contested resolution constants (R-5)
	StaleContestedDecayEpochs  = uint64(10)     // Epochs without activity before decay starts
	StaleContestedDecayRateBps = uint64(50_000) // 5% confidence decay per epoch when stale

	// ========================================================================
	// Novelty & Insight Metrics
	// ========================================================================

	// NoveltyMaxBonusBps is the max confidence bonus from novelty assessment (10%).
	NoveltyMaxBonusBps = uint64(100_000)

	// MinEvidenceForNoveltyBonus requires minimum evidence quality to earn novelty bonus.
	// Claims with weak evidence get no novelty credit (prevents rewarding hallucinated "novel" claims).
	MinEvidenceForNoveltyBonus = uint64(500_000) // 50% evidence quality floor

	// VerifiedBridgeThreshold is the confidence above which a cross-domain reference
	// flips from penalty (AH-3) to bonus — the referenced fact must be well-verified.
	VerifiedBridgeThreshold = uint64(700_000) // 70%

	// CrossDomainBridgeBonusBps is the bonus per verified cross-domain reference.
	CrossDomainBridgeBonusBps = uint64(15_000) // 1.5%

	// MaxBridgeBonusBps caps total bridge bonus across all cross-domain refs.
	MaxBridgeBonusBps = uint64(75_000) // 7.5%

	// PropheticConfidenceCap is the max confidence a prophetic fact can grow to.
	PropheticConfidenceCap = uint64(920_000) // 92%

	// PropheticBaseThreshold is the max acceptance confidence for a claim to be
	// eligible for prophetic recognition. Claims barely accepted (thin margin
	// above the 600k accept threshold) are candidates.
	PropheticBaseThreshold = uint64(650_000) // 65% — within 5% of acceptance threshold

	// PropheticCitationThreshold is how many citations a barely-accepted fact
	// needs to earn "prophetic" status (ahead of its time, later vindicated).
	PropheticCitationThreshold = uint64(10)

	// NoveltyImmunityThreshold is the novelty score above which a fact gets
	// extended grace period before pruning (protecting novel but underutilized knowledge).
	NoveltyImmunityThreshold = uint64(500_000) // 50%

	// NoveltyGraceMultiplier extends grace period for novel facts.
	NoveltyGraceMultiplier = uint64(3) // 3× normal grace period

	// NoveltyKMSWeight is the KMS weight allocated to novelty score (15%).
	// When added, the other weights are proportionally reduced.
	NoveltyKMSWeight = uint64(150_000)

	// NoveltyDecayHalfLifeMultiplier gives novel facts slower decay (2× half-life).
	NoveltyDecayHalfLifeMultiplier = uint64(2)
)

// CounterFactChallenge represents a challenge against a verified counter-fact (R-2).
// Enables symmetry: negative knowledge must survive adversarial scrutiny just like
// positive knowledge does via the survive-to-accept model.
// Not in proto — hand-written type for counter-fact challenge system.
type CounterFactChallenge struct {
	Id               string   `json:"id"`
	CounterFactId    string   `json:"counter_fact_id"` // Counter-fact being challenged
	Challenger       string   `json:"challenger"`      // bech32 address
	Stake            string   `json:"stake"`           // Locked challenge stake (ulgm)
	Reason           string   `json:"reason"`          // Why the counter-fact is wrong
	EvidenceIds      []string `json:"evidence_ids"`    // Supporting evidence
	Status           string   `json:"status"`          // open, upheld, reverted
	CreatedAtHeight  uint64   `json:"created_at_height"`
	ResolvedAtHeight uint64   `json:"resolved_at_height,omitempty"`
}

// ExtendedParams holds governance-tunable constants that are not in the
// proto-generated Params struct. Stored as JSON under ExtendedParamsKey.
// These were previously hardcoded constants in this file.
type ExtendedParams struct {
	// Citation economics
	CitationShareBps         uint64 `json:"citation_share_bps"`          // share of verification rewards to citation pool (default 150000 = 15%)
	CitationDecayPerLevel    uint64 `json:"citation_decay_per_level"`    // decay per ancestor level (default 500000 = 50%)
	CitationProtocolFloorBps uint64 `json:"citation_protocol_floor_bps"` // protocol floor (default 20000 = 2%)
	CrossDomainBonusRate     uint64 `json:"cross_domain_bonus_rate"`     // cross-domain bonus (default 200000 = 20%)
	MaxCrossDomainBonus      uint64 `json:"max_cross_domain_bonus"`      // max bonus multiplier (default 2000000 = 2.0x)

	// Confidence propagation
	PropagationThreshold uint64 `json:"propagation_threshold"` // min change to trigger (default 10000 = 1%)
	MaxPropagationDepth  uint32 `json:"max_propagation_depth"` // max BFS hops (default 10)
	CyclePenaltyFactor   uint64 `json:"cycle_penalty_factor"`  // penalty for cycle members (default 400000 = 0.4x)

	// Reverification
	ReverificationThreshold uint64 `json:"reverification_threshold"` // confidence floor (default 600000 = 60%)
	StalenessMultiplier     uint32 `json:"staleness_multiplier"`     // age multiplier (default 2)
	MinConfidenceThreshold  uint64 `json:"min_confidence_threshold"` // below this, fact expires (default 300000 = 30%)

	// Challenge tuning
	RejectThreshold             uint64 `json:"reject_threshold"`               // below this, claim rejected (default 300000 = 30%)
	MaxConcurrentChallenges     uint32 `json:"max_concurrent_challenges"`      // per claim (default 3)
	SurvivedChallengeConfCap    uint64 `json:"survived_challenge_conf_cap"`    // cap after surviving (default 880000 = 88%)
	ConfidenceGrowthEpochBlocks uint64 `json:"confidence_growth_epoch_blocks"` // growth interval (default 1111 blocks)
	BaseCooldownBlocks          uint64 `json:"base_cooldown_blocks"`           // anti-spam cooldown (default 17136 = ~0.5 day)
	MaxCooldownBlocks           uint64 `json:"max_cooldown_blocks"`            // max exponential cooldown (default 171360 = ~5 days)
	MaxReverificationsPerBlock  uint32 `json:"max_reverifications_per_block"`  // EndBlocker cap (default 5)

	// Maturity criteria
	MaturityEstablishedMinConf  uint64 `json:"maturity_established_min_conf"`  // default 800000 = 80%
	MaturityEstablishedMinAge   uint64 `json:"maturity_established_min_age"`   // default 34272 = ~1 day
	MaturityEstablishedMinVerif uint32 `json:"maturity_established_min_verif"` // default 3
	MaturityCanonicalMinConf    uint64 `json:"maturity_canonical_min_conf"`    // default 900000 = 90%
	MaturityCanonicalMinAge     uint64 `json:"maturity_canonical_min_age"`     // default 342720 = ~10 days
	MaturityCanonicalMinVerif   uint32 `json:"maturity_canonical_min_verif"`   // default 10

	// Calibration
	MinCalibrationSamples  uint32 `json:"min_calibration_samples"`  // default 5
	CalibrationNeutralRate uint64 `json:"calibration_neutral_rate"` // default 700000 = 70%
	MaxCalibrationBonus    uint64 `json:"max_calibration_bonus"`    // default 50000 = 5%
	MaxCalibrationPenalty  uint64 `json:"max_calibration_penalty"`  // default 150000 = 15%

	// Diversity detection
	MinDiversityParticipants uint32 `json:"min_diversity_participants"` // default 3
	UniformityPenaltyBps     uint64 `json:"uniformity_penalty_bps"`     // default 100000 = 10%
	SingleMethodPenaltyBps   uint64 `json:"single_method_penalty_bps"`  // default 50000 = 5%
	DiversityBonusBps        uint64 `json:"diversity_bonus_bps"`        // default 30000 = 3%

	// Negative knowledge
	MaxContradictionsPerAgentPerEpoch uint32 `json:"max_contradictions_per_agent_per_epoch"` // default 3
	MinContradictionStakeRatio        uint64 `json:"min_contradiction_stake_ratio"`          // default 500000 = 50%
	FailedContradictionSlashBps       uint64 `json:"failed_contradiction_slash_bps"`         // default 330000 = 33%
	FalsificationRewardCapBps         uint64 `json:"falsification_reward_cap_bps"`           // default 3000000 = 3x
	FalsificationPoolShareBps         uint64 `json:"falsification_pool_share_bps"`           // default 150000 = 15%
	CounterFactChallengeWindowBlocks  uint64 `json:"counter_fact_challenge_window_blocks"`   // default 34272 = ~1 day
	CounterFactChallengeMinStake      uint64 `json:"counter_fact_challenge_min_stake"`       // default 50000 ulgm
	CounterFactChallengeSlashBps      uint64 `json:"counter_fact_challenge_slash_bps"`       // default 220000 = 22%

	// Evidence
	MaxEvidenceBonus uint64 `json:"max_evidence_bonus"` // default 150000 = 15%

	// Epistemic domain (participant context)
	HumanEvidenceGapPenaltyBps uint64 `json:"human_evidence_gap_penalty_bps"` // penalty for observable claims with only agent evidence (default 100000 = 10%)
	AgentOnlyEmpiricalCeiling  uint64 `json:"agent_only_empirical_ceiling"`   // confidence ceiling in that case (default 700000 = 70%)
	HumanEvidenceBonusBps      uint64 `json:"human_evidence_bonus_bps"`       // bonus per human evidence item (default 10000 = 1%)
	MaxHumanEvidenceBonusBps   uint64 `json:"max_human_evidence_bonus_bps"`   // max total bonus (default 30000 = 3%)

	// Per-domain adversarial mode overrides.
	// Keys are domain names; values are true (adversarial) or false (legacy).
	// When set, this overrides the global AdversarialVerificationEnabled for that domain.
	AdversarialDomainOverrides map[string]bool `json:"adversarial_domain_overrides,omitempty"`

	// Knowledge pruning
	PruningEnabled                  bool   `json:"pruning_enabled"`                   // kill switch (default false)
	PruningThresholdBps             uint64 `json:"pruning_threshold_bps"`             // KMS below this → at_risk (default 100000 = 10%)
	GracePeriodBlocks               uint64 `json:"grace_period_blocks"`               // default grace before pruning (default 1028160 = ~30d)
	GraceMultiplier                 uint64 `json:"grace_multiplier"`                  // grace = max(GracePeriodBlocks, HalfLifeBlocks*GraceMultiplier) (default 2)
	ArchiveRetentionBlocks          uint64 `json:"archive_retention_blocks"`          // time in "pruned" before hard delete (default 3427200 = ~100d)
	MaxPruningPerBlock              uint32 `json:"max_pruning_per_block"`             // amortization cap (default 20)
	ContributionWeight              uint64 `json:"contribution_weight"`               // KMS weight (default 340000 = 34%)
	UtilizationWeight               uint64 `json:"utilization_weight"`                // KMS weight (default 255000 = 25.5%)
	TrajectoryWeight                uint64 `json:"trajectory_weight"`                 // KMS weight (default 170000 = 17%)
	MaturityWeight                  uint64 `json:"maturity_weight"`                   // KMS weight (default 85000 = 8.5%)
	NoveltyWeight                   uint64 `json:"novelty_weight"`                    // KMS weight (default 150000 = 15%)
	FundamentalityImmunityThreshold uint64 `json:"fundamentality_immunity_threshold"` // citation count for immunity (default 50)
	MinPatronageStake               uint64 `json:"min_patronage_stake"`               // min patronage stake in ulgm (default 1000000 = 1 LGM)

	// Novelty & insight metrics
	NoveltyMaxBonusBps             uint64 `json:"novelty_max_bonus_bps"`              // max bonus from novelty stage (default 100000 = 10%)
	MinEvidenceForNoveltyBonusBps  uint64 `json:"min_evidence_for_novelty_bonus_bps"` // evidence quality floor for novelty (default 500000 = 50%)
	VerifiedBridgeThresholdBps     uint64 `json:"verified_bridge_threshold_bps"`      // cross-domain ref flips to bonus above this (default 700000 = 70%)
	CrossDomainBridgeBonusBps      uint64 `json:"cross_domain_bridge_bonus_bps"`      // bonus per verified bridge (default 15000 = 1.5%)
	MaxBridgeBonusBps              uint64 `json:"max_bridge_bonus_bps"`               // cap on total bridge bonus (default 75000 = 7.5%)
	PropheticConfidenceCapBps      uint64 `json:"prophetic_confidence_cap_bps"`       // max confidence for prophetic facts (default 920000 = 92%)
	PropheticBaseThresholdBps      uint64 `json:"prophetic_base_threshold_bps"`       // max acceptance conf for prophetic eligibility (default 650000 = 65%)
	PropheticCitationThreshold     uint64 `json:"prophetic_citation_threshold"`       // citations needed for prophetic status (default 10)
	NoveltyImmunityThresholdBps    uint64 `json:"novelty_immunity_threshold_bps"`     // novelty score for extended grace (default 500000 = 50%)
	NoveltyGraceMultiplier         uint64 `json:"novelty_grace_multiplier"`           // grace period multiplier for novel facts (default 3)
	NoveltyDecayHalfLifeMultiplier uint64 `json:"novelty_decay_half_life_multiplier"` // half-life extension for novel facts (default 2)
}

// DefaultExtendedParams returns the default extended parameters.
// Values match the hardcoded constants defined above.
func DefaultExtendedParams() ExtendedParams {
	return ExtendedParams{
		CitationShareBps:         CitationShareBPS,
		CitationDecayPerLevel:    CitationDecayPerLevel,
		CitationProtocolFloorBps: CitationProtocolFloorBPS,
		CrossDomainBonusRate:     CrossDomainBonusRate,
		MaxCrossDomainBonus:      MaxCrossDomainBonus,

		PropagationThreshold: PropagationThreshold,
		MaxPropagationDepth:  MaxPropagationDepth,
		CyclePenaltyFactor:   CyclePenaltyFactor,

		ReverificationThreshold: ReverificationThreshold,
		StalenessMultiplier:     StalenessMultiplier,
		MinConfidenceThreshold:  MinConfidenceThreshold,

		RejectThreshold:             RejectThresholdDefault,
		MaxConcurrentChallenges:     MaxConcurrentChallenges,
		SurvivedChallengeConfCap:    SurvivedChallengeConfidenceCap,
		ConfidenceGrowthEpochBlocks: ConfidenceGrowthEpoch,
		BaseCooldownBlocks:          BaseCooldownBlocks,
		MaxCooldownBlocks:           MaxCooldownBlocks,
		MaxReverificationsPerBlock:  MaxReverificationsPerBlock,

		MaturityEstablishedMinConf:  MaturityEstablishedMinConf,
		MaturityEstablishedMinAge:   MaturityEstablishedMinAge,
		MaturityEstablishedMinVerif: MaturityEstablishedMinVerifications,
		MaturityCanonicalMinConf:    MaturityCanonicalMinConf,
		MaturityCanonicalMinAge:     MaturityCanonicalMinAge,
		MaturityCanonicalMinVerif:   MaturityCanonicalMinVerifications,

		MinCalibrationSamples:  MinCalibrationSamples,
		CalibrationNeutralRate: CalibrationNeutralRate,
		MaxCalibrationBonus:    MaxCalibrationBonus,
		MaxCalibrationPenalty:  MaxCalibrationPenalty,

		MinDiversityParticipants: MinDiversityParticipants,
		UniformityPenaltyBps:     uint64(UniformityPenaltyBps),
		SingleMethodPenaltyBps:   uint64(SingleMethodPenaltyBps),
		DiversityBonusBps:        uint64(DiversityBonusBps),

		MaxContradictionsPerAgentPerEpoch: MaxContradictionsPerAgentPerEpoch,
		MinContradictionStakeRatio:        MinContradictionStakeRatio,
		FailedContradictionSlashBps:       FailedContradictionSlashBps,
		FalsificationRewardCapBps:         FalsificationRewardCapBps,
		FalsificationPoolShareBps:         FalsificationPoolShareBps,
		CounterFactChallengeWindowBlocks:  CounterFactChallengeWindowBlocks,
		CounterFactChallengeMinStake:      CounterFactChallengeMinStake,
		CounterFactChallengeSlashBps:      CounterFactChallengeSlashBps,

		MaxEvidenceBonus: MaxEvidenceBonus,

		// Epistemic domain defaults
		HumanEvidenceGapPenaltyBps: HumanEvidenceGapPenaltyBpsDefault,
		AgentOnlyEmpiricalCeiling:  AgentOnlyEmpiricalCeilingDefault,
		HumanEvidenceBonusBps:      HumanEvidenceBonusBpsDefault,
		MaxHumanEvidenceBonusBps:   MaxHumanEvidenceBonusBpsDefault,

		// Knowledge pruning defaults
		// KMS weights sum to 1,000,000: contribution(34%) + utilization(25.5%) + trajectory(17%) + maturity(8.5%) + novelty(15%)
		PruningEnabled:                  false,
		PruningThresholdBps:             100_000,   // 10%
		GracePeriodBlocks:               1_028_160, // ~30 days at 2521ms blocks
		GraceMultiplier:                 2,
		ArchiveRetentionBlocks:          3_427_200, // ~100 days
		MaxPruningPerBlock:              20,
		ContributionWeight:              340_000, // 34% (was 40%, reduced to make room for novelty)
		UtilizationWeight:               255_000, // 25.5% (was 30%)
		TrajectoryWeight:                170_000, // 17% (was 20%)
		MaturityWeight:                  85_000,  // 8.5% (was 10%)
		NoveltyWeight:                   150_000, // 15% (new)
		FundamentalityImmunityThreshold: 50,
		MinPatronageStake:               1_000_000, // 1 LGM

		// Novelty & insight metrics defaults
		NoveltyMaxBonusBps:             NoveltyMaxBonusBps,
		MinEvidenceForNoveltyBonusBps:  MinEvidenceForNoveltyBonus,
		VerifiedBridgeThresholdBps:     VerifiedBridgeThreshold,
		CrossDomainBridgeBonusBps:      CrossDomainBridgeBonusBps,
		MaxBridgeBonusBps:              MaxBridgeBonusBps,
		PropheticConfidenceCapBps:      PropheticConfidenceCap,
		PropheticBaseThresholdBps:      PropheticBaseThreshold,
		PropheticCitationThreshold:     PropheticCitationThreshold,
		NoveltyImmunityThresholdBps:    NoveltyImmunityThreshold,
		NoveltyGraceMultiplier:         NoveltyGraceMultiplier,
		NoveltyDecayHalfLifeMultiplier: NoveltyDecayHalfLifeMultiplier,
	}
}

// ValidateExtendedParams performs basic validation on extended parameters.
func (ep ExtendedParams) ValidateExtendedParams() error {
	if ep.CitationShareBps > 1_000_000 {
		return fmt.Errorf("citation_share_bps must be <= 1000000")
	}
	if ep.CitationDecayPerLevel > 1_000_000 {
		return fmt.Errorf("citation_decay_per_level must be <= 1000000")
	}
	if ep.MaxPropagationDepth == 0 || ep.MaxPropagationDepth > 100 {
		return fmt.Errorf("max_propagation_depth must be 1-100")
	}
	if ep.ReverificationThreshold > 1_000_000 {
		return fmt.Errorf("reverification_threshold must be <= 1000000")
	}
	if ep.MinConfidenceThreshold > ep.ReverificationThreshold {
		return fmt.Errorf("min_confidence_threshold must be <= reverification_threshold")
	}
	if ep.RejectThreshold > 1_000_000 {
		return fmt.Errorf("reject_threshold must be <= 1000000")
	}
	if ep.MaxConcurrentChallenges == 0 {
		return fmt.Errorf("max_concurrent_challenges must be positive")
	}
	if ep.SurvivedChallengeConfCap > 1_000_000 {
		return fmt.Errorf("survived_challenge_conf_cap must be <= 1000000")
	}
	if ep.MaturityEstablishedMinConf > ep.MaturityCanonicalMinConf {
		return fmt.Errorf("established min confidence must be <= canonical min confidence")
	}
	if ep.FailedContradictionSlashBps > 1_000_000 {
		return fmt.Errorf("failed_contradiction_slash_bps must be <= 1000000")
	}
	if ep.MaxEvidenceBonus > 1_000_000 {
		return fmt.Errorf("max_evidence_bonus must be <= 1000000")
	}
	// Epistemic domain validation
	if ep.HumanEvidenceGapPenaltyBps > 1_000_000 {
		return fmt.Errorf("human_evidence_gap_penalty_bps must be <= 1000000")
	}
	if ep.AgentOnlyEmpiricalCeiling > 1_000_000 {
		return fmt.Errorf("agent_only_empirical_ceiling must be <= 1000000")
	}
	if ep.MaxHumanEvidenceBonusBps > 1_000_000 {
		return fmt.Errorf("max_human_evidence_bonus_bps must be <= 1000000")
	}
	// Pruning params validation
	if ep.PruningThresholdBps > 1_000_000 {
		return fmt.Errorf("pruning_threshold_bps must be <= 1000000")
	}
	if ep.ContributionWeight+ep.UtilizationWeight+ep.TrajectoryWeight+ep.MaturityWeight+ep.NoveltyWeight != 1_000_000 && ep.ContributionWeight > 0 {
		return fmt.Errorf("KMS weights must sum to 1000000")
	}
	if ep.MaxPruningPerBlock > 100 {
		return fmt.Errorf("max_pruning_per_block must be <= 100")
	}
	// Novelty params validation
	if ep.NoveltyMaxBonusBps > 1_000_000 {
		return fmt.Errorf("novelty_max_bonus_bps must be <= 1000000")
	}
	if ep.VerifiedBridgeThresholdBps > 1_000_000 {
		return fmt.Errorf("verified_bridge_threshold_bps must be <= 1000000")
	}
	if ep.PropheticConfidenceCapBps > 1_000_000 {
		return fmt.Errorf("prophetic_confidence_cap_bps must be <= 1000000")
	}
	if ep.PropheticBaseThresholdBps > 1_000_000 {
		return fmt.Errorf("prophetic_base_threshold_bps must be <= 1000000")
	}
	if ep.NoveltyDecayHalfLifeMultiplier == 0 {
		return fmt.Errorf("novelty_decay_half_life_multiplier must be > 0")
	}
	return nil
}

// MsgUpdateExtendedParams updates the extended (non-proto) module parameters.
// Governance-gated: only the gov module authority can call this.
type MsgUpdateExtendedParams struct {
	Authority      string         `json:"authority"`
	ExtendedParams ExtendedParams `json:"extended_params"`
}

func (msg *MsgUpdateExtendedParams) Reset()         {}
func (msg *MsgUpdateExtendedParams) String() string { return "MsgUpdateExtendedParams" }
func (msg *MsgUpdateExtendedParams) ProtoMessage()  {}

func (msg *MsgUpdateExtendedParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateExtendedParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return msg.ExtendedParams.ValidateExtendedParams()
}

// MsgUpdateExtendedParamsResponse is the response for MsgUpdateExtendedParams.
type MsgUpdateExtendedParamsResponse struct{}

func (msg *MsgUpdateExtendedParamsResponse) Reset()         {}
func (msg *MsgUpdateExtendedParamsResponse) String() string { return "" }
func (msg *MsgUpdateExtendedParamsResponse) ProtoMessage()  {}

// PatronageRecord tracks a patron's stake in keeping a fact alive.
type PatronageRecord struct {
	FactId    string `json:"fact_id"`
	Patron    string `json:"patron"`     // bech32 address
	Amount    string `json:"amount"`     // ulgm staked
	CreatedAt uint64 `json:"created_at"` // block height
	ExpiresAt uint64 `json:"expires_at"` // lock period end
	Settled   bool   `json:"settled"`
}

func (msg *MsgPatronizeFact) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Patron)
	return []sdk.AccAddress{addr}
}

func (msg *MsgPatronizeFact) ValidateBasic() error {
	if msg.Patron == "" {
		return fmt.Errorf("patron cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Patron); err != nil {
		return fmt.Errorf("invalid patron address: %w", err)
	}
	if msg.FactId == "" {
		return fmt.Errorf("fact_id cannot be empty")
	}
	if msg.Amount == "" {
		return fmt.Errorf("amount cannot be empty")
	}
	return nil
}

// Note: RegisterCodec and RegisterInterfaces are in codec.go
