package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// Stratum represents a level in the knowledge ontology.
// Lower strata are more formally verifiable.
type Stratum uint32

const (
	StratumAxiomatic     Stratum = 0 // Mathematical axioms, tautologies (complete, decidable)
	StratumFormal        Stratum = 1 // Formal proofs (complete in restricted domains)
	StratumProtocol      Stratum = 2 // Blockchain-verifiable (deterministic)
	StratumComputational Stratum = 3 // Computation results (reproducible)
	StratumEmpirical     Stratum = 4 // Scientific observations (falsifiable)
	StratumHistorical    Stratum = 5 // Historical records (evidence-based)
	StratumTestimonial   Stratum = 6 // Human attestations (trust-weighted)

	MaxStratum = StratumTestimonial
)

// IsValid returns whether the stratum is a valid, recognized level.
func (s Stratum) IsValid() bool {
	return s <= MaxStratum
}

// StratumProperties defines the logical properties of each stratum.
type StratumProperties struct {
	Stratum          Stratum `json:"stratum"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	Complete         bool    `json:"complete"`          // Goedel: is the system complete?
	Decidable        bool    `json:"decidable"`         // is there an algorithm to decide all statements?
	GoedelApplies    bool    `json:"goedel_applies"`    // does Goedel's incompleteness apply?
	ConsistencyProof string  `json:"consistency_proof"` // "internal", "assumed", "external"
	MaxConfidence    uint64  `json:"max_confidence"`    // max confidence ceiling (basis points, 1000000 = 1.0)
	DecayRate        uint64  `json:"decay_rate"`        // confidence decay rate per day (basis points)
}

func (sp *StratumProperties) Reset()         {}
func (sp *StratumProperties) String() string { return fmt.Sprintf("%+v", *sp) }
func (sp *StratumProperties) ProtoMessage()  {}

// Domain represents a knowledge domain within a stratum.
type Domain struct {
	Name        string  `json:"name"`         // unique identifier
	DisplayName string  `json:"display_name"` // human-readable name
	Description string  `json:"description"`  // domain description
	Stratum     Stratum `json:"stratum"`      // which stratum this domain belongs to
	Status      string  `json:"status"`       // "active", "proposed", "deprecated"
	CreatedAt   uint64  `json:"created_at"`   // block height
	UpdatedAt   uint64  `json:"updated_at"`   // block height
	ClaimCount  uint64  `json:"claim_count"`  // total claims in domain
	FactCount   uint64  `json:"fact_count"`   // total verified facts in domain
	ProposedBy  string  `json:"proposed_by"`  // bech32 address of proposer
}

func (d *Domain) Reset()         {}
func (d *Domain) String() string { return fmt.Sprintf("%+v", *d) }
func (d *Domain) ProtoMessage()  {}

// DomainProposal for adding/modifying domains.
type DomainProposal struct {
	Id           string `json:"id"`            // proposal ID
	Domain       Domain `json:"domain"`        // proposed domain
	Proposer     string `json:"proposer"`      // bech32 address
	ProposalType string `json:"proposal_type"` // "add", "deprecate", "modify"
	Stake        string `json:"stake"`         // stake locked for proposal
	VotesFor     uint32 `json:"votes_for"`     // endorsement count
	VotesAgainst uint32 `json:"votes_against"` // rejection count
	Voters       []string `json:"voters"`      // addresses that have voted
	Status       string `json:"status"`        // "active", "passed", "rejected", "expired"
	CreatedAt    uint64 `json:"created_at"`    // block height
	ExpiresAt    uint64 `json:"expires_at"`    // block height
}

func (dp *DomainProposal) Reset()         {}
func (dp *DomainProposal) String() string { return fmt.Sprintf("%+v", *dp) }
func (dp *DomainProposal) ProtoMessage()  {}

// HasVoted returns whether the given address has already voted on this proposal.
func (dp *DomainProposal) HasVoted(addr string) bool {
	for _, v := range dp.Voters {
		if v == addr {
			return true
		}
	}
	return false
}

// CrossStratumLink represents a relationship between domains in different strata.
type CrossStratumLink struct {
	SourceDomain string `json:"source_domain"`
	TargetDomain string `json:"target_domain"`
	LinkType     string `json:"link_type"` // "depends_on", "generalizes", "restricts"
	Discount     uint64 `json:"discount"`  // confidence discount for cross-stratum references (basis points)
}

func (csl *CrossStratumLink) Reset()         {}
func (csl *CrossStratumLink) String() string { return fmt.Sprintf("%+v", *csl) }
func (csl *CrossStratumLink) ProtoMessage()  {}

// Params for the ontology module.
type Params struct {
	MinProposalStake     string `json:"min_proposal_stake"`      // minimum stake for domain proposal
	ProposalVotingPeriod uint64 `json:"proposal_voting_period"`  // blocks
	MinEndorsements      uint32 `json:"min_endorsements"`        // minimum votes for domain approval
	CrossStratumDiscount uint64 `json:"cross_stratum_discount"`  // default discount for cross-stratum refs (basis points)
	MaxDomainsPerStratum uint32 `json:"max_domains_per_stratum"` // maximum domains per stratum
	AllowNewStrata       bool   `json:"allow_new_strata"`        // whether governance can add new strata
}

func (p *Params) Reset()         {}
func (p *Params) String() string { return fmt.Sprintf("%+v", *p) }
func (p *Params) ProtoMessage()  {}

// Validate performs parameter validation.
func (p Params) Validate() error {
	if p.CrossStratumDiscount > 1000000 {
		return fmt.Errorf("cross stratum discount cannot exceed 1000000 basis points")
	}
	if p.MinEndorsements == 0 {
		return fmt.Errorf("min endorsements must be greater than 0")
	}
	if p.ProposalVotingPeriod == 0 {
		return fmt.Errorf("proposal voting period must be greater than 0")
	}
	return nil
}

// ---------- Message Types ----------

// MsgProposeDomain proposes a new domain within a stratum.
type MsgProposeDomain struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Description string  `json:"description"`
	Stratum     Stratum `json:"stratum"`
	Proposer    string  `json:"proposer"` // bech32 address
	Stake       string  `json:"stake"`    // bigint as string
}

func (msg *MsgProposeDomain) Reset()         {}
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
	if msg.DisplayName == "" {
		return fmt.Errorf("display name cannot be empty")
	}
	if !msg.Stratum.IsValid() {
		return fmt.Errorf("invalid stratum: %d", msg.Stratum)
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
	ProposalId string `json:"proposal_id"`
}

func (msg *MsgProposeDomainResponse) Reset()         {}
func (msg *MsgProposeDomainResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgProposeDomainResponse) ProtoMessage()  {}

// MsgVoteDomainProposal votes on a domain proposal.
type MsgVoteDomainProposal struct {
	ProposalId string `json:"proposal_id"`
	Voter      string `json:"voter"`   // bech32 address
	Approve    bool   `json:"approve"` // true = for, false = against
}

func (msg *MsgVoteDomainProposal) Reset()         {}
func (msg *MsgVoteDomainProposal) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgVoteDomainProposal) ProtoMessage()  {}

func (msg *MsgVoteDomainProposal) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Voter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgVoteDomainProposal) ValidateBasic() error {
	if msg.ProposalId == "" {
		return fmt.Errorf("proposal ID cannot be empty")
	}
	if msg.Voter == "" {
		return fmt.Errorf("voter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	return nil
}

type MsgVoteDomainProposalResponse struct{}

func (msg *MsgVoteDomainProposalResponse) Reset()         {}
func (msg *MsgVoteDomainProposalResponse) String() string { return "" }
func (msg *MsgVoteDomainProposalResponse) ProtoMessage()  {}

// MsgUpdateDomain updates domain metadata (authority only).
type MsgUpdateDomain struct {
	Authority   string `json:"authority"` // bech32 address (gov module)
	DomainName  string `json:"domain_name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"` // "active", "deprecated"
}

func (msg *MsgUpdateDomain) Reset()         {}
func (msg *MsgUpdateDomain) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgUpdateDomain) ProtoMessage()  {}

func (msg *MsgUpdateDomain) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateDomain) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.DomainName == "" {
		return fmt.Errorf("domain name cannot be empty")
	}
	if msg.Status != "" && msg.Status != "active" && msg.Status != "deprecated" {
		return fmt.Errorf("invalid status: must be 'active' or 'deprecated'")
	}
	return nil
}

type MsgUpdateDomainResponse struct{}

func (msg *MsgUpdateDomainResponse) Reset()         {}
func (msg *MsgUpdateDomainResponse) String() string { return "" }
func (msg *MsgUpdateDomainResponse) ProtoMessage()  {}

// MsgUpdateParams updates module parameters (authority only).
type MsgUpdateParams struct {
	Authority string `json:"authority"` // bech32 address (gov module)
	Params    Params `json:"params"`
}

func (msg *MsgUpdateParams) Reset()         {}
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

func (msg *MsgUpdateParamsResponse) Reset()         {}
func (msg *MsgUpdateParamsResponse) String() string { return "" }
func (msg *MsgUpdateParamsResponse) ProtoMessage()  {}

// ---------- MsgServer Interface ----------

// MsgServer is the service interface for the ontology module.
type MsgServer interface {
	ProposeDomain(context.Context, *MsgProposeDomain) (*MsgProposeDomainResponse, error)
	VoteDomainProposal(context.Context, *MsgVoteDomainProposal) (*MsgVoteDomainProposalResponse, error)
	UpdateDomain(context.Context, *MsgUpdateDomain) (*MsgUpdateDomainResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}

// Service descriptor for MsgServer registration.
var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.ontology.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "ProposeDomain", Handler: nil},
		{MethodName: "VoteDomainProposal", Handler: nil},
		{MethodName: "UpdateDomain", Handler: nil},
		{MethodName: "UpdateParams", Handler: nil},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/ontology/v1/tx.proto",
}

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// ---------- QueryServer Interface ----------

// QueryServer is the query service interface for the ontology module.
type QueryServer interface {
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	Stratum(context.Context, *QueryStratumRequest) (*QueryStratumResponse, error)
	AllStrata(context.Context, *QueryAllStrataRequest) (*QueryAllStrataResponse, error)
	Domain(context.Context, *QueryDomainRequest) (*QueryDomainResponse, error)
	DomainsByStratum(context.Context, *QueryDomainsByStratumRequest) (*QueryDomainsByStratumResponse, error)
	AllDomains(context.Context, *QueryAllDomainsRequest) (*QueryAllDomainsResponse, error)
	Proposal(context.Context, *QueryProposalRequest) (*QueryProposalResponse, error)
	ConfidenceCeiling(context.Context, *QueryConfidenceCeilingRequest) (*QueryConfidenceCeilingResponse, error)
}

// Query request/response types.

type QueryParamsRequest struct{}

func (r *QueryParamsRequest) Reset()         {}
func (r *QueryParamsRequest) String() string { return "QueryParamsRequest" }
func (r *QueryParamsRequest) ProtoMessage()  {}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (r *QueryParamsResponse) Reset()         {}
func (r *QueryParamsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryParamsResponse) ProtoMessage()  {}

type QueryStratumRequest struct {
	Stratum Stratum `json:"stratum"`
}

func (r *QueryStratumRequest) Reset()         {}
func (r *QueryStratumRequest) String() string { return fmt.Sprintf("QueryStratum{%d}", r.Stratum) }
func (r *QueryStratumRequest) ProtoMessage()  {}

type QueryStratumResponse struct {
	Properties StratumProperties `json:"properties"`
}

func (r *QueryStratumResponse) Reset()         {}
func (r *QueryStratumResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryStratumResponse) ProtoMessage()  {}

type QueryAllStrataRequest struct{}

func (r *QueryAllStrataRequest) Reset()         {}
func (r *QueryAllStrataRequest) String() string { return "QueryAllStrataRequest" }
func (r *QueryAllStrataRequest) ProtoMessage()  {}

type QueryAllStrataResponse struct {
	Strata []StratumProperties `json:"strata"`
}

func (r *QueryAllStrataResponse) Reset()         {}
func (r *QueryAllStrataResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryAllStrataResponse) ProtoMessage()  {}

type QueryDomainRequest struct {
	Name string `json:"name"`
}

func (r *QueryDomainRequest) Reset()         {}
func (r *QueryDomainRequest) String() string { return fmt.Sprintf("QueryDomain{%s}", r.Name) }
func (r *QueryDomainRequest) ProtoMessage()  {}

type QueryDomainResponse struct {
	Domain Domain `json:"domain"`
}

func (r *QueryDomainResponse) Reset()         {}
func (r *QueryDomainResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryDomainResponse) ProtoMessage()  {}

type QueryDomainsByStratumRequest struct {
	Stratum Stratum `json:"stratum"`
}

func (r *QueryDomainsByStratumRequest) Reset()         {}
func (r *QueryDomainsByStratumRequest) String() string { return fmt.Sprintf("QueryDomainsByStratum{%d}", r.Stratum) }
func (r *QueryDomainsByStratumRequest) ProtoMessage()  {}

type QueryDomainsByStratumResponse struct {
	Domains []Domain `json:"domains"`
}

func (r *QueryDomainsByStratumResponse) Reset()         {}
func (r *QueryDomainsByStratumResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryDomainsByStratumResponse) ProtoMessage()  {}

type QueryAllDomainsRequest struct{}

func (r *QueryAllDomainsRequest) Reset()         {}
func (r *QueryAllDomainsRequest) String() string { return "QueryAllDomainsRequest" }
func (r *QueryAllDomainsRequest) ProtoMessage()  {}

type QueryAllDomainsResponse struct {
	Domains []Domain `json:"domains"`
}

func (r *QueryAllDomainsResponse) Reset()         {}
func (r *QueryAllDomainsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryAllDomainsResponse) ProtoMessage()  {}

type QueryProposalRequest struct {
	ProposalId string `json:"proposal_id"`
}

func (r *QueryProposalRequest) Reset()         {}
func (r *QueryProposalRequest) String() string { return fmt.Sprintf("QueryProposal{%s}", r.ProposalId) }
func (r *QueryProposalRequest) ProtoMessage()  {}

type QueryProposalResponse struct {
	Proposal DomainProposal `json:"proposal"`
}

func (r *QueryProposalResponse) Reset()         {}
func (r *QueryProposalResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryProposalResponse) ProtoMessage()  {}

type QueryConfidenceCeilingRequest struct {
	DomainName string `json:"domain_name"`
}

func (r *QueryConfidenceCeilingRequest) Reset()         {}
func (r *QueryConfidenceCeilingRequest) String() string { return fmt.Sprintf("QueryConfidenceCeiling{%s}", r.DomainName) }
func (r *QueryConfidenceCeilingRequest) ProtoMessage()  {}

type QueryConfidenceCeilingResponse struct {
	MaxConfidence uint64 `json:"max_confidence"`
	DecayRate     uint64 `json:"decay_rate"`
	DomainName    string `json:"domain_name"`
	StratumName   string `json:"stratum_name"`
}

func (r *QueryConfidenceCeilingResponse) Reset()         {}
func (r *QueryConfidenceCeilingResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryConfidenceCeilingResponse) ProtoMessage()  {}

// Service descriptor for QueryServer registration.
var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.ontology.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Params", Handler: nil},
		{MethodName: "Stratum", Handler: nil},
		{MethodName: "AllStrata", Handler: nil},
		{MethodName: "Domain", Handler: nil},
		{MethodName: "DomainsByStratum", Handler: nil},
		{MethodName: "AllDomains", Handler: nil},
		{MethodName: "Proposal", Handler: nil},
		{MethodName: "ConfidenceCeiling", Handler: nil},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/ontology/v1/query.proto",
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
