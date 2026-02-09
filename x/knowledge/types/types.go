package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// Proto message interface stubs
type ProtoMessage interface {
	Reset()
	String() string
	ProtoMessage()
}

// Fact status constants (9-state lifecycle from TS reference)
const (
	FactStatusPending    = "pending"
	FactStatusVerified   = "verified"
	FactStatusActive     = "active"
	FactStatusContested  = "contested"
	FactStatusChallenged = "challenged"
	FactStatusSuperseded = "superseded"
	FactStatusExpired    = "expired"
	FactStatusDisproven  = "disproven"
	FactStatusRevoked    = "revoked"
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
	HalfLifeBlocks      uint64 // 0 = eternal (analytic, formal, protocol)
	MaxConfidence        uint64 // stratum ceiling (basis points, 0-1_000_000)
	MinConfidence        uint64 // floor after decay
	ReverificationBonus  uint64 // bonus on successful re-verification
	RequiresReverification bool
}

// CATEGORY_DECAY maps epistemic category to decay profile.
// Values from TS core/knowledge/types.ts EPISTEMIC_PROFILES.
// Block time: 2521ms → BLOCKS_PER_DAY = 34272
var CATEGORY_DECAY = map[string]CategoryDecayProfile{
	"analytic":      {HalfLifeBlocks: 0, MaxConfidence: 1_000_000, MinConfidence: 0, ReverificationBonus: 0, RequiresReverification: false},
	"formal":        {HalfLifeBlocks: 0, MaxConfidence: 1_000_000, MinConfidence: 0, ReverificationBonus: 0, RequiresReverification: false},
	"protocol":      {HalfLifeBlocks: 0, MaxConfidence: 1_000_000, MinConfidence: 0, ReverificationBonus: 0, RequiresReverification: false},
	"empirical":     {HalfLifeBlocks: 100_000, MaxConfidence: 990_000, MinConfidence: 300_000, ReverificationBonus: 50_000, RequiresReverification: true},
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
	CitationShareBPS          = 150_000 // 15% of verification rewards to citation pool
	CitationDecayPerLevel     = 500_000 // 50% decay per ancestor level
	CitationProtocolFloorBPS  = 20_000  // 2% protocol floor
	CrossDomainBonusRate      = 200_000 // 20% cross-domain bonus
	MaxCrossDomainBonus       = 2_000_000 // 2.0x max

	// Fundamentality thresholds
	FundamentalityNotable     = 250_000 // 0.25
	FundamentalityFoundational = 500_000 // 0.50
	FundamentalityFundamental  = 750_000 // 0.75
	FundamentalityAxiomatic    = 900_000 // 0.90

	// Fundamentality reward multipliers
	FundamentalityBonusNotable     = 1_100_000 // 1.1x
	FundamentalityBonusFoundational = 1_250_000 // 1.25x
	FundamentalityBonusFundamental  = 1_500_000 // 1.5x
	FundamentalityBonusAxiomatic    = 2_000_000 // 2.0x
)

// Confidence propagation constants
const (
	PropagationThreshold  = 10_000  // 1% change triggers propagation (basis points)
	MaxPropagationDepth   = 10      // max BFS hops
	CyclePenaltyFactor    = 400_000 // 0.4× penalty for cycle members
)

// Reverification constants
const (
	ReverificationThreshold = 600_000  // 60% confidence floor
	StalenessMultiplier     = 2        // age > 2× halfLife → needs reverification
	MinConfidenceThreshold  = 300_000  // 30% — below this, fact expires
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
	MaturityEstablishedMinConf         = 800_000 // 0.8 confidence
	MaturityEstablishedMinAge          = 34_272  // ~1 day in blocks
	MaturityEstablishedMinVerifications = 3
	MaturityCanonicalMinConf           = 900_000 // 0.9 confidence
	MaturityCanonicalMinAge            = 342_720 // ~10 days in blocks
	MaturityCanonicalMinVerifications  = 10
)

// Fact represents a verified knowledge claim
type Fact struct {
	Id                   string   `json:"id"`
	ClaimText            string   `json:"claim_text"`
	Domain               string   `json:"domain"`
	EpistemicCategory    string   `json:"epistemic_category"` // protocol, analytic, formal, empirical, historical, computational, social, predictive, replicated
	Confidence           uint64   `json:"confidence"`         // 0-1000000 (6 decimal precision)
	Status               string   `json:"status"`             // pending, verified, active, contested, challenged, superseded, expired, disproven, revoked
	Submitter            string   `json:"submitter"`          // bech32 address
	SubmitterDid         string   `json:"submitter_did"`
	References           []string `json:"references"`      // fact IDs
	VerifiedAtHeight     uint64   `json:"verified_at_height"`
	VerifierCount        uint32   `json:"verifier_count"`
	LogicZone            string   `json:"logic_zone"`
	CreatedAtHeight      uint64   `json:"created_at_height"`
	LastVerifiedHeight   uint64   `json:"last_verified_height"`
	LastConfidenceUpdate uint64   `json:"last_confidence_update,omitempty"` // block of last decay calculation
	BaseConfidence       uint64   `json:"base_confidence,omitempty"`       // confidence before decay applied

	// Expiration fields
	ExpiresAtBlock uint64 `json:"expires_at_block,omitempty"` // explicit expiration (0 = no explicit expiry)

	// Citation economics fields
	CitationCount         uint64 `json:"citation_count,omitempty"`
	LastCitedBlock        uint64 `json:"last_cited_block,omitempty"`
	TotalCitationRewards  uint64 `json:"total_citation_rewards,omitempty"` // cumulative ulgm earned

	// Supersession fields
	SupersededBy       string `json:"superseded_by,omitempty"`        // fact ID that supersedes this one
	SupersededAtHeight uint64 `json:"superseded_at_height,omitempty"`

	// Maturity and stratum fields
	Maturity       string `json:"maturity,omitempty"`        // emerging, established, canonical
	StratumCeiling uint64 `json:"stratum_ceiling,omitempty"` // inherited ceiling from stratum (0 = no ceiling)
	ChallengeCount uint32 `json:"challenge_count,omitempty"` // number of challenges received

	// Evidence fields (transferred from claim on acceptance)
	EvidenceSummaries []FactEvidenceSummary `json:"evidence_summaries,omitempty"` // evidence that supported this fact
	EvidenceBonus     uint64                `json:"evidence_bonus,omitempty"`     // bonus applied during verification (basis points)
}

// FactEvidenceSummary is a snapshot of evidence that supported a fact at verification time.
// Stored on the Fact so evidence provenance is preserved after the claim is consumed.
type FactEvidenceSummary struct {
	EvidenceId   string `json:"evidence_id"`
	EvidenceType string `json:"evidence_type"`
	Reliability  uint64 `json:"reliability"`   // basis points at time of verification
	TypeStrength uint64 `json:"type_strength"` // intrinsic type strength (basis points)
}

func (f *Fact) Reset()        {}
func (f *Fact) String() string { return fmt.Sprintf("%+v", *f) }
func (f *Fact) ProtoMessage()  {}

// ClaimEvidence is supporting evidence for a knowledge claim.
type ClaimEvidence struct {
	EvidenceType string            `json:"evidence_type"`
	URI          string            `json:"uri"`
	ContentHash  string            `json:"content_hash"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Claim represents a knowledge claim pending verification
type Claim struct {
	Id                string          `json:"id"`
	ClaimText         string          `json:"claim_text"`
	ContentHash       string          `json:"content_hash"`
	Domain            string          `json:"domain"`
	EpistemicCategory string          `json:"epistemic_category"`
	Submitter         string          `json:"submitter"`     // bech32 address
	SubmitterDid      string          `json:"submitter_did"`
	StakeAmount       string          `json:"stake_amount"` // bigint as string
	Status            string          `json:"status"`       // pending, in_verification, accepted, rejected, challenged, expired
	SubmittedAtHeight uint64          `json:"submitted_at_height"`
	References        []string        `json:"references"` // fact IDs
	LogicZone         string          `json:"logic_zone"`
	RoundId           string          `json:"round_id"`
	Evidence          []ClaimEvidence `json:"evidence,omitempty"` // supporting evidence
}

func (c *Claim) Reset()        {}
func (c *Claim) String() string { return fmt.Sprintf("%+v", *c) }
func (c *Claim) ProtoMessage()  {}

// VerificationRound represents a commit-reveal verification round
type VerificationRound struct {
	Id                    string                            `json:"id"`
	ClaimId               string                            `json:"claim_id"`
	Phase                 string                            `json:"phase"` // commit, reveal, aggregation, complete
	StartHeight           uint64                            `json:"start_height"`
	CommitDeadlineHeight  uint64                            `json:"commit_deadline_height"`
	RevealDeadlineHeight  uint64                            `json:"reveal_deadline_height"`
	SelectedValidators    []string                          `json:"selected_validators"` // addresses
	Commitments           map[string]VerificationCommitment `json:"commitments"`         // validator→commitment
	Reveals               map[string]VerificationReveal     `json:"reveals"`             // validator→reveal
	FinalVerdict          string                            `json:"final_verdict"`       // accepted, rejected, inconclusive
	FinalConfidence       uint64                            `json:"final_confidence"`
	CompletedAtHeight     uint64                            `json:"completed_at_height"`
}

func (vr *VerificationRound) Reset()        {}
func (vr *VerificationRound) String() string { return fmt.Sprintf("%+v", *vr) }
func (vr *VerificationRound) ProtoMessage()  {}

// VerificationCommitment represents a validator's commitment in commit phase
type VerificationCommitment struct {
	Validator         string `json:"validator"`
	CommitmentHash    string `json:"commitment_hash"`
	VrfOutput         string `json:"vrf_output"`          // hex-encoded VRF output (for selection verification)
	VrfProof          string `json:"vrf_proof"`            // hex-encoded VRF proof (96 bytes)
	SubmittedAtHeight uint64 `json:"submitted_at_height"`
}

func (vc *VerificationCommitment) Reset()        {}
func (vc *VerificationCommitment) String() string { return fmt.Sprintf("%+v", *vc) }
func (vc *VerificationCommitment) ProtoMessage()  {}

// VerificationReveal represents a validator's reveal in reveal phase
type VerificationReveal struct {
	Validator         string `json:"validator"`
	Verdict           string `json:"verdict"` // accept, reject, abstain
	Confidence        uint64 `json:"confidence"`
	Justification     string `json:"justification"`
	VrfProof          string `json:"vrf_proof"`
	Salt              string `json:"salt,omitempty"` // salt used in commitment hash, stored for auditability
	SubmittedAtHeight uint64 `json:"submitted_at_height"`
}

func (vr *VerificationReveal) Reset()        {}
func (vr *VerificationReveal) String() string { return fmt.Sprintf("%+v", *vr) }
func (vr *VerificationReveal) ProtoMessage()  {}

// FactReference represents a relationship between facts
type FactReference struct {
	FromFactId      string `json:"from_fact_id"`
	ToFactId        string `json:"to_fact_id"`
	ReferenceType   string `json:"reference_type"` // supports, contradicts, extends, qualifies
	Weight          uint64 `json:"weight"`
	CreatedAtHeight uint64 `json:"created_at_height,omitempty"` // block when reference was created
}

func (fr *FactReference) Reset()        {}
func (fr *FactReference) String() string { return fmt.Sprintf("%+v", *fr) }
func (fr *FactReference) ProtoMessage()  {}

// ValidatorInfo represents a validator eligible for verification rounds.
type ValidatorInfo struct {
	Address         string `json:"address"`          // bech32 operator address
	Stake           uint64 `json:"stake"`            // effective stake in ulgm
	Tier            uint32 `json:"tier"`             // validator tier (0-3)
	PublicKey       []byte `json:"public_key"`       // Ed25519 public key bytes
	Reputation      uint64 `json:"reputation"`       // 0-1000000
}

func (vi *ValidatorInfo) Reset()        {}
func (vi *ValidatorInfo) String() string { return fmt.Sprintf("%+v", *vi) }
func (vi *ValidatorInfo) ProtoMessage()  {}

// VerificationResult represents the aggregated outcome of a verification round.
type VerificationResult struct {
	Verdict           string            `json:"verdict"`          // accepted, rejected, contested, insufficient
	Confidence        uint64            `json:"confidence"`       // 0-1000000 (6 decimal precision)
	AgreementRatio    uint64            `json:"agreement_ratio"`  // 0-1000000
	TotalWeight       uint64            `json:"total_weight"`     // total stake weight of participants
	Rewards           map[string]uint64 `json:"rewards"`          // validator → reward amount
	Slashes           map[string]uint64 `json:"slashes"`          // validator → slash amount
	ParticipantCount  uint32            `json:"participant_count"`
	CompletedAtHeight uint64            `json:"completed_at_height,omitempty"` // block height when aggregation completed
	QuorumReached     bool              `json:"quorum_reached"`                // whether quorum threshold was met
	EvidenceBonus     uint64            `json:"evidence_bonus,omitempty"`      // additive evidence bonus applied (basis points)
}

func (vr *VerificationResult) Reset()        {}
func (vr *VerificationResult) String() string { return fmt.Sprintf("%+v", *vr) }
func (vr *VerificationResult) ProtoMessage()  {}

// OntologicalDomain represents a knowledge domain
type OntologicalDomain struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	Proposer        string `json:"proposer"` // bech32 address
	Status          string `json:"status"`   // proposed, active, deprecated
	Endorsements    uint32 `json:"endorsements"`
	Challenges      uint32 `json:"challenges"`
	CreatedAtHeight uint64 `json:"created_at_height"`
}

func (od *OntologicalDomain) Reset()        {}
func (od *OntologicalDomain) String() string { return fmt.Sprintf("%+v", *od) }
func (od *OntologicalDomain) ProtoMessage()  {}

// GenesisState represents the knowledge module genesis state
type GenesisState struct {
	Params         Params                `json:"params"`
	Domains        []OntologicalDomain   `json:"domains"`
	Facts          []Fact                `json:"facts"`
	Claims         []Claim               `json:"claims"`
	FactReferences []FactReference       `json:"fact_references"`
}

func (gs *GenesisState) Reset()        {}
func (gs *GenesisState) String() string { return fmt.Sprintf("%+v", *gs) }
func (gs *GenesisState) ProtoMessage()  {}

// Params defines the knowledge module parameters
type Params struct {
	OntologyVersion                 uint64 `json:"ontology_version"`
	CrossStratumDiscount            uint64 `json:"cross_stratum_discount"`
	MinDomainProposalStake          string `json:"min_domain_proposal_stake"`
	DomainVotingPeriod              uint64 `json:"domain_voting_period"`
	MinEndorsements                 uint32 `json:"min_endorsements"`
	AllowPartialCoverage            bool   `json:"allow_partial_coverage"`
	PartialCoveragePenalty          uint64 `json:"partial_coverage_penalty"`
	LowerStratumVetoWeight          uint64 `json:"lower_stratum_veto_weight"`
	CrossStratumAgreementThreshold  uint64 `json:"cross_stratum_agreement_threshold"`
	MinClaimTextLength              uint32 `json:"min_claim_text_length"`           // default 20
	MaxClaimRewardMultiplier        uint64 `json:"max_claim_reward_multiplier"`     // default 2 (times stake)
	CommitPhaseDuration             uint64 `json:"commit_phase_duration"`           // blocks, default 4
	RevealPhaseDuration             uint64 `json:"reveal_phase_duration"`           // blocks, default 4
	AggregationPhaseDuration        uint64 `json:"aggregation_phase_duration"`      // blocks, default 3
	MinVerifiers                    uint32 `json:"min_verifiers"`                   // default 3
	AcceptanceThreshold             uint64 `json:"acceptance_threshold"`            // 0-1000000, default 770000 (77%)
	QuorumThreshold                 uint64 `json:"quorum_threshold"`               // 0-1000000, default 660000 (66%)
	MaxValidatorsPerRound           uint32 `json:"max_validators_per_round"`       // default 22
	InvalidClaimSlashBps            uint64 `json:"invalid_claim_slash_bps"`        // slash on rejection (220000 = 22%)
	VerificationRewardAmount        uint64 `json:"verification_reward_amount"`     // ulgm per correct verification (default 100000 = 0.1 LGM)
	ResearchTaxRate                 uint64 `json:"research_tax_rate"`              // basis points taxed from rewards to research fund (default 220000 = 22%)
}

func (p *Params) Reset()        {}
func (p *Params) String() string { return fmt.Sprintf("%+v", *p) }
func (p *Params) ProtoMessage()  {}

// MustMarshal marshals params to bytes, panics on error
func (p Params) MustMarshal() []byte {
	// Placeholder - in real implementation would use amino/proto encoding
	return []byte{}
}

// MustUnmarshal unmarshals params from bytes, panics on error
func (p *Params) MustUnmarshal(bz []byte) {
	// Placeholder - in real implementation would use amino/proto encoding
}

// Message types

// MsgProposeDomain proposes a new ontological domain
type MsgProposeDomain struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Proposer    string `json:"proposer"` // bech32 address
	Stake       string `json:"stake"`    // bigint as string
}

func (msg *MsgProposeDomain) Reset()        {}
func (msg *MsgProposeDomain) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgProposeDomain) ProtoMessage()  {}

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

type MsgProposeDomainResponse struct {
	DomainName string `json:"domain_name"`
}

func (msg *MsgProposeDomainResponse) Reset()        {}
func (msg *MsgProposeDomainResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgProposeDomainResponse) ProtoMessage()  {}

// MsgEndorseDomainProposal endorses a domain proposal
type MsgEndorseDomainProposal struct {
	DomainName string `json:"domain_name"`
	Endorser   string `json:"endorser"` // bech32 address
}

func (msg *MsgEndorseDomainProposal) Reset()        {}
func (msg *MsgEndorseDomainProposal) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgEndorseDomainProposal) ProtoMessage()  {}

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

type MsgEndorseDomainProposalResponse struct{}

func (msg *MsgEndorseDomainProposalResponse) Reset()        {}
func (msg *MsgEndorseDomainProposalResponse) String() string { return "" }
func (msg *MsgEndorseDomainProposalResponse) ProtoMessage()  {}

// MsgChallengeDomainProposal challenges a domain proposal
type MsgChallengeDomainProposal struct {
	DomainName string `json:"domain_name"`
	Challenger string `json:"challenger"` // bech32 address
	Reason     string `json:"reason"`
}

func (msg *MsgChallengeDomainProposal) Reset()        {}
func (msg *MsgChallengeDomainProposal) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgChallengeDomainProposal) ProtoMessage()  {}

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

type MsgChallengeDomainProposalResponse struct{}

func (msg *MsgChallengeDomainProposalResponse) Reset()        {}
func (msg *MsgChallengeDomainProposalResponse) String() string { return "" }
func (msg *MsgChallengeDomainProposalResponse) ProtoMessage()  {}

// MsgRegisterStratum registers an ontological stratum
type MsgRegisterStratum struct {
	StratumName string `json:"stratum_name"`
	Description string `json:"description"`
	Registrar   string `json:"registrar"` // bech32 address
}

func (msg *MsgRegisterStratum) Reset()        {}
func (msg *MsgRegisterStratum) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRegisterStratum) ProtoMessage()  {}

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

type MsgRegisterStratumResponse struct{}

func (msg *MsgRegisterStratumResponse) Reset()        {}
func (msg *MsgRegisterStratumResponse) String() string { return "" }
func (msg *MsgRegisterStratumResponse) ProtoMessage()  {}

// MsgUpdateParams updates module parameters
type MsgUpdateParams struct {
	Authority string `json:"authority"` // bech32 address (gov module)
	Params    Params `json:"params"`
}

func (msg *MsgUpdateParams) Reset()        {}
func (msg *MsgUpdateParams) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgUpdateParams) ProtoMessage()  {}

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

type MsgUpdateParamsResponse struct{}

func (msg *MsgUpdateParamsResponse) Reset()        {}
func (msg *MsgUpdateParamsResponse) String() string { return "" }
func (msg *MsgUpdateParamsResponse) ProtoMessage()  {}

// MsgSubmitClaim submits a knowledge claim for verification
type MsgSubmitClaim struct {
	ClaimText         string          `json:"claim_text"`
	Domain            string          `json:"domain"`
	EpistemicCategory string          `json:"epistemic_category"`
	Submitter         string          `json:"submitter"`      // bech32 address
	SubmitterDid      string          `json:"submitter_did"`
	StakeAmount       string          `json:"stake_amount"`   // bigint as string
	References        []string        `json:"references"`     // fact IDs
	LogicZone         string          `json:"logic_zone"`
	Evidence          []ClaimEvidence `json:"evidence,omitempty"` // supporting evidence
}

func (msg *MsgSubmitClaim) Reset()        {}
func (msg *MsgSubmitClaim) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitClaim) ProtoMessage()  {}

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

type MsgSubmitClaimResponse struct {
	ClaimId string `json:"claim_id"`
	RoundId string `json:"round_id"`
}

func (msg *MsgSubmitClaimResponse) Reset()        {}
func (msg *MsgSubmitClaimResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitClaimResponse) ProtoMessage()  {}

// MsgSubmitCommitment submits a commitment in the commit phase
type MsgSubmitCommitment struct {
	RoundId        string `json:"round_id"`
	Validator      string `json:"validator"`        // bech32 address
	CommitmentHash string `json:"commitment_hash"`
}

func (msg *MsgSubmitCommitment) Reset()        {}
func (msg *MsgSubmitCommitment) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitCommitment) ProtoMessage()  {}

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

type MsgSubmitCommitmentResponse struct{}

func (msg *MsgSubmitCommitmentResponse) Reset()        {}
func (msg *MsgSubmitCommitmentResponse) String() string { return "" }
func (msg *MsgSubmitCommitmentResponse) ProtoMessage()  {}

// MsgSubmitReveal submits a reveal in the reveal phase
type MsgSubmitReveal struct {
	RoundId       string `json:"round_id"`
	Validator     string `json:"validator"` // bech32 address
	Verdict       string `json:"verdict"`   // accept, reject, abstain
	Confidence    uint64 `json:"confidence"`
	Justification string `json:"justification"`
	VrfProof      string `json:"vrf_proof"`
	Salt          string `json:"salt"` // for commitment verification
}

func (msg *MsgSubmitReveal) Reset()        {}
func (msg *MsgSubmitReveal) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitReveal) ProtoMessage()  {}

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

type MsgSubmitRevealResponse struct{}

func (msg *MsgSubmitRevealResponse) Reset()        {}
func (msg *MsgSubmitRevealResponse) String() string { return "" }
func (msg *MsgSubmitRevealResponse) ProtoMessage()  {}

// MsgChallengeFact challenges an existing fact
type MsgChallengeFact struct {
	FactId     string `json:"fact_id"`
	Challenger string `json:"challenger"` // bech32 address
	Reason     string `json:"reason"`
	Stake      string `json:"stake"` // bigint as string
}

func (msg *MsgChallengeFact) Reset()        {}
func (msg *MsgChallengeFact) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgChallengeFact) ProtoMessage()  {}

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

type MsgChallengeFactResponse struct {
	RoundId string `json:"round_id"`
}

func (msg *MsgChallengeFactResponse) Reset()        {}
func (msg *MsgChallengeFactResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgChallengeFactResponse) ProtoMessage()  {}

// MsgAddFact manually adds a fact (gov-only, for genesis/migration)
type MsgAddFact struct {
	Authority         string   `json:"authority"` // bech32 address (gov module)
	ClaimText         string   `json:"claim_text"`
	Domain            string   `json:"domain"`
	EpistemicCategory string   `json:"epistemic_category"`
	Confidence        uint64   `json:"confidence"`
	References        []string `json:"references"`
	LogicZone         string   `json:"logic_zone"`
}

func (msg *MsgAddFact) Reset()        {}
func (msg *MsgAddFact) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgAddFact) ProtoMessage()  {}

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

type MsgAddFactResponse struct {
	FactId string `json:"fact_id"`
}

func (msg *MsgAddFactResponse) Reset()        {}
func (msg *MsgAddFactResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgAddFactResponse) ProtoMessage()  {}

// MsgServer is the service interface for the knowledge module
type MsgServer interface {
	ProposeDomain(context.Context, *MsgProposeDomain) (*MsgProposeDomainResponse, error)
	EndorseDomainProposal(context.Context, *MsgEndorseDomainProposal) (*MsgEndorseDomainProposalResponse, error)
	ChallengeDomainProposal(context.Context, *MsgChallengeDomainProposal) (*MsgChallengeDomainProposalResponse, error)
	RegisterStratum(context.Context, *MsgRegisterStratum) (*MsgRegisterStratumResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
	SubmitClaim(context.Context, *MsgSubmitClaim) (*MsgSubmitClaimResponse, error)
	SubmitCommitment(context.Context, *MsgSubmitCommitment) (*MsgSubmitCommitmentResponse, error)
	SubmitReveal(context.Context, *MsgSubmitReveal) (*MsgSubmitRevealResponse, error)
	ChallengeFact(context.Context, *MsgChallengeFact) (*MsgChallengeFactResponse, error)
	AddFact(context.Context, *MsgAddFact) (*MsgAddFactResponse, error)
}

// Placeholder service descriptor for codec registration
var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.knowledge.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ProposeDomain",
			Handler:    nil,
		},
		{
			MethodName: "EndorseDomainProposal",
			Handler:    nil,
		},
		{
			MethodName: "ChallengeDomainProposal",
			Handler:    nil,
		},
		{
			MethodName: "RegisterStratum",
			Handler:    nil,
		},
		{
			MethodName: "UpdateParams",
			Handler:    nil,
		},
		{
			MethodName: "SubmitClaim",
			Handler:    nil,
		},
		{
			MethodName: "SubmitCommitment",
			Handler:    nil,
		},
		{
			MethodName: "SubmitReveal",
			Handler:    nil,
		},
		{
			MethodName: "ChallengeFact",
			Handler:    nil,
		},
		{
			MethodName: "AddFact",
			Handler:    nil,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/knowledge/v1/tx.proto",
}

// Note: RegisterCodec and RegisterInterfaces are in codec.go
