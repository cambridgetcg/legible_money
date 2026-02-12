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

// HasVoted returns whether the given address has already voted on this proposal.
func (dp *DomainProposal) HasVoted(addr string) bool {
	for _, v := range dp.Voters {
		if v == addr {
			return true
		}
	}
	return false
}

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

// -----------------------------------------------------------------------
// Godelian Logic Zone types
// -----------------------------------------------------------------------

// LogicZone identifies a logic system with known completeness properties.
type LogicZone string

const (
	ZonePropositional LogicZone = "propositional"
	ZonePresburger    LogicZone = "presburger"
	ZonePeano         LogicZone = "peano"
	ZoneSetTheory     LogicZone = "set_theory"
	ZoneEmpirical     LogicZone = "empirical"
)

// LogicZoneProperties defines the formal properties of a logic zone.
type LogicZoneProperties struct {
	Zone           LogicZone `json:"zone"`
	Complete       bool      `json:"complete"`           // is the system complete?
	Decidable      bool      `json:"decidable"`          // is there a decision procedure?
	GoedelApplies  bool      `json:"goedel_applies"`     // does Godel's incompleteness apply?
	MaxConfidenceBps uint64  `json:"max_confidence_bps"` // confidence ceiling (basis points 0-1000000)
	Description    string    `json:"description"`
}

func (lz *LogicZoneProperties) Reset()        {}
func (lz *LogicZoneProperties) String() string { return fmt.Sprintf("%+v", *lz) }
func (lz *LogicZoneProperties) ProtoMessage()  {}

// IncompletenessAcknowledgment records that a fact in an incomplete zone
// has been properly flagged with its epistemic limitations.
type IncompletenessAcknowledgment struct {
	FactId           string    `json:"fact_id"`
	Zone             LogicZone `json:"zone"`
	Reason           string    `json:"reason"` // why this claim is in an incomplete zone
	AcknowledgedAt   uint64    `json:"acknowledged_at"` // block height
	AcknowledgedBy   string    `json:"acknowledged_by"` // bech32 address
}

func (ia *IncompletenessAcknowledgment) Reset()        {}
func (ia *IncompletenessAcknowledgment) String() string { return fmt.Sprintf("%+v", *ia) }
func (ia *IncompletenessAcknowledgment) ProtoMessage()  {}

// DefaultLogicZones returns the pre-defined logic zone properties.
func DefaultLogicZones() []LogicZoneProperties {
	return []LogicZoneProperties{
		{
			Zone:             ZonePropositional,
			Complete:         true,
			Decidable:        true,
			GoedelApplies:    false,
			MaxConfidenceBps: 1000000, // 100% confidence possible
			Description:      "Propositional logic: complete, decidable, truth-table decidable",
		},
		{
			Zone:             ZonePresburger,
			Complete:         true,
			Decidable:        true,
			GoedelApplies:    false,
			MaxConfidenceBps: 1000000,
			Description:      "Presburger arithmetic: complete, decidable (no multiplication)",
		},
		{
			Zone:             ZonePeano,
			Complete:         false,
			Decidable:        false,
			GoedelApplies:    true,
			MaxConfidenceBps: 850000, // 85% max -- incompleteness acknowledged
			Description:      "Peano arithmetic: incomplete (Godel), undecidable, contains true but unprovable statements",
		},
		{
			Zone:             ZoneSetTheory,
			Complete:         false,
			Decidable:        false,
			GoedelApplies:    true,
			MaxConfidenceBps: 800000, // 80% max
			Description:      "Set theory (ZFC): incomplete, undecidable, CH-independent",
		},
		{
			Zone:             ZoneEmpirical,
			Complete:         false,
			Decidable:        false,
			GoedelApplies:    false, // not formal system
			MaxConfidenceBps: 700000, // 70% max -- inherently probabilistic
			Description:      "Empirical claims: inherently probabilistic, falsifiable but not provable",
		},
	}
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

// ---------- Logic Zone Messages ----------

// MsgRegisterLogicZone registers a new logic zone (authority only).
type MsgRegisterLogicZone struct {
	Authority      string             `json:"authority"`
	ZoneProperties LogicZoneProperties `json:"zone_properties"`
}

func (msg *MsgRegisterLogicZone) Reset()         {}
func (msg *MsgRegisterLogicZone) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRegisterLogicZone) ProtoMessage()  {}

func (msg *MsgRegisterLogicZone) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterLogicZone) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.ZoneProperties.Zone == "" {
		return fmt.Errorf("zone name cannot be empty")
	}
	if msg.ZoneProperties.MaxConfidenceBps > 1000000 {
		return fmt.Errorf("max confidence cannot exceed 1000000 basis points")
	}
	return nil
}

type MsgRegisterLogicZoneResponse struct{}

func (msg *MsgRegisterLogicZoneResponse) Reset()         {}
func (msg *MsgRegisterLogicZoneResponse) String() string { return "" }
func (msg *MsgRegisterLogicZoneResponse) ProtoMessage()  {}

// MsgAcknowledgeIncompleteness records that a fact in an incomplete zone has been acknowledged.
type MsgAcknowledgeIncompleteness struct {
	Submitter string    `json:"submitter"` // bech32 address
	FactId    string    `json:"fact_id"`
	Zone      LogicZone `json:"zone"`
	Reason    string    `json:"reason"`
}

func (msg *MsgAcknowledgeIncompleteness) Reset()         {}
func (msg *MsgAcknowledgeIncompleteness) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgAcknowledgeIncompleteness) ProtoMessage()  {}

func (msg *MsgAcknowledgeIncompleteness) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAcknowledgeIncompleteness) ValidateBasic() error {
	if msg.Submitter == "" {
		return fmt.Errorf("submitter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if msg.FactId == "" {
		return fmt.Errorf("fact ID cannot be empty")
	}
	if msg.Zone == "" {
		return fmt.Errorf("zone cannot be empty")
	}
	if msg.Reason == "" {
		return fmt.Errorf("reason cannot be empty")
	}
	return nil
}

type MsgAcknowledgeIncompletenessResponse struct{}

func (msg *MsgAcknowledgeIncompletenessResponse) Reset()         {}
func (msg *MsgAcknowledgeIncompletenessResponse) String() string { return "" }
func (msg *MsgAcknowledgeIncompletenessResponse) ProtoMessage()  {}

// ---------- MsgServer Interface ----------

// MsgServer is the service interface for the ontology module.
type MsgServer interface {
	ProposeDomain(context.Context, *MsgProposeDomain) (*MsgProposeDomainResponse, error)
	VoteDomainProposal(context.Context, *MsgVoteDomainProposal) (*MsgVoteDomainProposalResponse, error)
	UpdateDomain(context.Context, *MsgUpdateDomain) (*MsgUpdateDomainResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
	RegisterLogicZone(context.Context, *MsgRegisterLogicZone) (*MsgRegisterLogicZoneResponse, error)
	AcknowledgeIncompleteness(context.Context, *MsgAcknowledgeIncompleteness) (*MsgAcknowledgeIncompletenessResponse, error)
}


func _Msg_ProposeDomain_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgProposeDomain)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ProposeDomain(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Msg/ProposeDomain"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ProposeDomain(ctx, req.(*MsgProposeDomain))
	})
}

func _Msg_VoteDomainProposal_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgVoteDomainProposal)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).VoteDomainProposal(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Msg/VoteDomainProposal"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).VoteDomainProposal(ctx, req.(*MsgVoteDomainProposal))
	})
}

func _Msg_UpdateDomain_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateDomain)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateDomain(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Msg/UpdateDomain"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateDomain(ctx, req.(*MsgUpdateDomain))
	})
}

func _Msg_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Msg/UpdateParams"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	})
}

func _Msg_RegisterLogicZone_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterLogicZone)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RegisterLogicZone(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Msg/RegisterLogicZone"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RegisterLogicZone(ctx, req.(*MsgRegisterLogicZone))
	})
}

func _Msg_AcknowledgeIncompleteness_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAcknowledgeIncompleteness)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AcknowledgeIncompleteness(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Msg/AcknowledgeIncompleteness"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AcknowledgeIncompleteness(ctx, req.(*MsgAcknowledgeIncompleteness))
	})
}

// Service descriptor for MsgServer registration.
var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.ontology.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "ProposeDomain", Handler: _Msg_ProposeDomain_Handler},
		{MethodName: "VoteDomainProposal", Handler: _Msg_VoteDomainProposal_Handler},
		{MethodName: "UpdateDomain", Handler: _Msg_UpdateDomain_Handler},
		{MethodName: "UpdateParams", Handler: _Msg_UpdateParams_Handler},
		{MethodName: "RegisterLogicZone", Handler: _Msg_RegisterLogicZone_Handler},
		{MethodName: "AcknowledgeIncompleteness", Handler: _Msg_AcknowledgeIncompleteness_Handler},
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
	LogicZone(context.Context, *QueryLogicZoneRequest) (*QueryLogicZoneResponse, error)
	AllLogicZones(context.Context, *QueryAllLogicZonesRequest) (*QueryAllLogicZonesResponse, error)
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

// Logic zone query types.

type QueryLogicZoneRequest struct {
	Zone LogicZone `json:"zone"`
}

func (r *QueryLogicZoneRequest) Reset()         {}
func (r *QueryLogicZoneRequest) String() string { return fmt.Sprintf("QueryLogicZone{%s}", r.Zone) }
func (r *QueryLogicZoneRequest) ProtoMessage()  {}

type QueryLogicZoneResponse struct {
	Properties LogicZoneProperties `json:"properties"`
}

func (r *QueryLogicZoneResponse) Reset()         {}
func (r *QueryLogicZoneResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryLogicZoneResponse) ProtoMessage()  {}

type QueryAllLogicZonesRequest struct{}

func (r *QueryAllLogicZonesRequest) Reset()         {}
func (r *QueryAllLogicZonesRequest) String() string { return "QueryAllLogicZonesRequest" }
func (r *QueryAllLogicZonesRequest) ProtoMessage()  {}

type QueryAllLogicZonesResponse struct {
	Zones []LogicZoneProperties `json:"zones"`
}

func (r *QueryAllLogicZonesResponse) Reset()         {}
func (r *QueryAllLogicZonesResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryAllLogicZonesResponse) ProtoMessage()  {}


func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

func _Query_Stratum_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryStratumRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Stratum(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Query/Stratum"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Stratum(ctx, req.(*QueryStratumRequest))
	})
}

func _Query_AllStrata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAllStrataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AllStrata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Query/AllStrata"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AllStrata(ctx, req.(*QueryAllStrataRequest))
	})
}

func _Query_Domain_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDomainRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Domain(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Query/Domain"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Domain(ctx, req.(*QueryDomainRequest))
	})
}

func _Query_DomainsByStratum_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDomainsByStratumRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).DomainsByStratum(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Query/DomainsByStratum"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).DomainsByStratum(ctx, req.(*QueryDomainsByStratumRequest))
	})
}

func _Query_AllDomains_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAllDomainsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AllDomains(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Query/AllDomains"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AllDomains(ctx, req.(*QueryAllDomainsRequest))
	})
}

func _Query_Proposal_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryProposalRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Proposal(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Query/Proposal"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Proposal(ctx, req.(*QueryProposalRequest))
	})
}

func _Query_ConfidenceCeiling_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryConfidenceCeilingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ConfidenceCeiling(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Query/ConfidenceCeiling"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ConfidenceCeiling(ctx, req.(*QueryConfidenceCeilingRequest))
	})
}

func _Query_LogicZone_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryLogicZoneRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).LogicZone(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Query/LogicZone"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).LogicZone(ctx, req.(*QueryLogicZoneRequest))
	})
}

func _Query_AllLogicZones_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAllLogicZonesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AllLogicZones(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.ontology.v1.Query/AllLogicZones"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AllLogicZones(ctx, req.(*QueryAllLogicZonesRequest))
	})
}

// Service descriptor for QueryServer registration.
var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.ontology.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Params", Handler: _Query_Params_Handler},
		{MethodName: "Stratum", Handler: _Query_Stratum_Handler},
		{MethodName: "AllStrata", Handler: _Query_AllStrata_Handler},
		{MethodName: "Domain", Handler: _Query_Domain_Handler},
		{MethodName: "DomainsByStratum", Handler: _Query_DomainsByStratum_Handler},
		{MethodName: "AllDomains", Handler: _Query_AllDomains_Handler},
		{MethodName: "Proposal", Handler: _Query_Proposal_Handler},
		{MethodName: "ConfidenceCeiling", Handler: _Query_ConfidenceCeiling_Handler},
		{MethodName: "LogicZone", Handler: _Query_LogicZone_Handler},
		{MethodName: "AllLogicZones", Handler: _Query_AllLogicZones_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/ontology/v1/query.proto",
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
