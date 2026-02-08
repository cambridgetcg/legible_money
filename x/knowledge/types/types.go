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

// Fact represents a verified knowledge claim
type Fact struct {
	Id                  string   `json:"id"`
	ClaimText           string   `json:"claim_text"`
	Domain              string   `json:"domain"`
	EpistemicCategory   string   `json:"epistemic_category"` // protocol, analytic, formal, empirical, historical, computational, social
	Confidence          uint64   `json:"confidence"`         // 0-1000000 (6 decimal precision)
	Status              string   `json:"status"`             // verified, contested, falsified, expired
	Submitter           string   `json:"submitter"`          // bech32 address
	SubmitterDid        string   `json:"submitter_did"`
	References          []string `json:"references"`      // fact IDs
	VerifiedAtHeight    uint64   `json:"verified_at_height"`
	VerifierCount       uint32   `json:"verifier_count"`
	LogicZone           string   `json:"logic_zone"`
	CreatedAtHeight     uint64   `json:"created_at_height"`
	LastVerifiedHeight  uint64   `json:"last_verified_height"`
}

func (f *Fact) Reset()        {}
func (f *Fact) String() string { return fmt.Sprintf("%+v", *f) }
func (f *Fact) ProtoMessage()  {}

// Claim represents a knowledge claim pending verification
type Claim struct {
	Id                string   `json:"id"`
	ClaimText         string   `json:"claim_text"`
	ContentHash       string   `json:"content_hash"`
	Domain            string   `json:"domain"`
	EpistemicCategory string   `json:"epistemic_category"`
	Submitter         string   `json:"submitter"`     // bech32 address
	SubmitterDid      string   `json:"submitter_did"`
	StakeAmount       string   `json:"stake_amount"` // bigint as string
	Status            string   `json:"status"`       // pending, in_verification, accepted, rejected, challenged, expired
	SubmittedAtHeight uint64   `json:"submitted_at_height"`
	References        []string `json:"references"` // fact IDs
	LogicZone         string   `json:"logic_zone"`
	RoundId           string   `json:"round_id"`
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
	SubmittedAtHeight uint64 `json:"submitted_at_height"`
}

func (vr *VerificationReveal) Reset()        {}
func (vr *VerificationReveal) String() string { return fmt.Sprintf("%+v", *vr) }
func (vr *VerificationReveal) ProtoMessage()  {}

// FactReference represents a relationship between facts
type FactReference struct {
	FromFactId    string `json:"from_fact_id"`
	ToFactId      string `json:"to_fact_id"`
	ReferenceType string `json:"reference_type"` // supports, contradicts, extends, qualifies
	Weight        uint64 `json:"weight"`
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
	Verdict        string            `json:"verdict"`          // accepted, rejected, contested, insufficient
	Confidence     uint64            `json:"confidence"`       // 0-1000000 (6 decimal precision)
	AgreementRatio uint64            `json:"agreement_ratio"`  // 0-1000000
	TotalWeight    uint64            `json:"total_weight"`     // total stake weight of participants
	Rewards        map[string]uint64 `json:"rewards"`          // validator → reward amount
	Slashes        map[string]uint64 `json:"slashes"`          // validator → slash amount
	ParticipantCount uint32          `json:"participant_count"`
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
	ClaimText         string   `json:"claim_text"`
	Domain            string   `json:"domain"`
	EpistemicCategory string   `json:"epistemic_category"`
	Submitter         string   `json:"submitter"`      // bech32 address
	SubmitterDid      string   `json:"submitter_did"`
	StakeAmount       string   `json:"stake_amount"`   // bigint as string
	References        []string `json:"references"`     // fact IDs
	LogicZone         string   `json:"logic_zone"`
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
