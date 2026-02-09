package types

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc"
)

// --- Enums ---

// EmergencyStatus represents the current state of the emergency system.
type EmergencyStatus string

const (
	StatusNormal       EmergencyStatus = "normal"
	StatusHaltVoting   EmergencyStatus = "halt_voting"
	StatusHalted       EmergencyStatus = "halted"
	StatusRevertVoting EmergencyStatus = "revert_voting"
	StatusReverting    EmergencyStatus = "reverting"
	StatusResumeVoting EmergencyStatus = "resume_voting"
)

// CeremonyType represents the type of emergency ceremony.
type CeremonyType string

const (
	CeremonyHalt   CeremonyType = "halt"
	CeremonyRevert CeremonyType = "revert"
	CeremonyResume CeremonyType = "resume"
)

// CeremonyPhase represents the current phase of a ceremony.
type CeremonyPhase string

const (
	PhasePrevote   CeremonyPhase = "prevote"
	PhasePrecommit CeremonyPhase = "precommit"
	PhaseFinalized CeremonyPhase = "finalized"
	PhaseFailed    CeremonyPhase = "failed"
)

// AuditAction represents an emergency audit log action.
type AuditAction string

const (
	AuditHaltProposed      AuditAction = "halt_proposed"
	AuditHaltPrevote       AuditAction = "halt_prevote"
	AuditHaltPrecommit     AuditAction = "halt_precommit"
	AuditHaltFinalized     AuditAction = "halt_finalized"
	AuditHaltFailed        AuditAction = "halt_failed"
	AuditHaltExecuted      AuditAction = "halt_executed"
	AuditRevertProposed    AuditAction = "revert_proposed"
	AuditRevertPrevote     AuditAction = "revert_prevote"
	AuditRevertPrecommit   AuditAction = "revert_precommit"
	AuditRevertFinalized   AuditAction = "revert_finalized"
	AuditRevertFailed      AuditAction = "revert_failed"
	AuditRevertExecuted    AuditAction = "revert_executed"
	AuditResumeProposed    AuditAction = "resume_proposed"
	AuditResumePrevote     AuditAction = "resume_prevote"
	AuditResumePrecommit   AuditAction = "resume_precommit"
	AuditResumeFinalized   AuditAction = "resume_finalized"
	AuditResumeFailed      AuditAction = "resume_failed"
	AuditResumeExecuted    AuditAction = "resume_executed"
)

// --- Proposal Types ---

// EmergencyHaltProposal is a proposal to halt the chain.
type EmergencyHaltProposal struct {
	Id                  string `json:"id"`
	Proposer            string `json:"proposer"`
	Reason              string `json:"reason"`
	EvidenceHash        string `json:"evidence_hash"`
	ProposedAtBlock     uint64 `json:"proposed_at_block"`
	SuggestedRevertBlock uint64 `json:"suggested_revert_block,omitempty"`
}

// EmergencyRevertProposal is a proposal to revert chain state.
type EmergencyRevertProposal struct {
	Id                string `json:"id"`
	Proposer          string `json:"proposer"`
	TargetBlockNumber uint64 `json:"target_block_number"`
	TargetBlockHash   string `json:"target_block_hash"`
	TargetStateRoot   string `json:"target_state_root"`
	Justification     string `json:"justification"`
	HaltCeremonyId    string `json:"halt_ceremony_id"`
}

// EmergencyResumeProposal is a proposal to resume chain operations.
type EmergencyResumeProposal struct {
	Id               string `json:"id"`
	Proposer         string `json:"proposer"`
	ResumeFromBlock  uint64 `json:"resume_from_block"`
	ResumeFromHash   string `json:"resume_from_hash"`
	ResumeStateRoot  string `json:"resume_state_root"`
	HaltCeremonyId   string `json:"halt_ceremony_id"`
	RevertCeremonyId string `json:"revert_ceremony_id,omitempty"`
}

// --- Vote Types ---

// EmergencyVote represents a Guardian's prevote on a ceremony.
type EmergencyVote struct {
	Voter    string `json:"voter"`
	Approve  bool   `json:"approve"`
	Signature string `json:"signature"`
}

// EmergencyPrecommit represents a Guardian's precommit on a ceremony.
type EmergencyPrecommit struct {
	Voter     string `json:"voter"`
	Signature string `json:"signature"`
}

// --- Ceremony ---

// EmergencyCeremony represents an active or completed emergency ceremony.
type EmergencyCeremony struct {
	Id                string            `json:"id"`
	Type              CeremonyType      `json:"type"`
	Phase             CeremonyPhase     `json:"phase"`
	ProposalData      json.RawMessage   `json:"proposal_data"`
	StartBlock        uint64            `json:"start_block"`
	PrevoteDeadline   uint64            `json:"prevote_deadline"`
	PrecommitDeadline uint64            `json:"precommit_deadline"`
	TimeoutDeadline   uint64            `json:"timeout_deadline"`
	Prevotes          map[string]EmergencyVote      `json:"prevotes"`
	Precommits        map[string]EmergencyPrecommit `json:"precommits"`
	YesPrevoteStake   string            `json:"yes_prevote_stake"`
	NoPrevoteStake    string            `json:"no_prevote_stake"`
	PrecommitStake    string            `json:"precommit_stake"`
	FailureReason     string            `json:"failure_reason,omitempty"`
}

func (c *EmergencyCeremony) Reset()         { *c = EmergencyCeremony{} }
func (c *EmergencyCeremony) String() string { return proto.CompactTextString(c) }
func (c *EmergencyCeremony) ProtoMessage()  {}

// --- Audit ---

// EmergencyAuditEntry records an emergency action for transparency.
type EmergencyAuditEntry struct {
	Timestamp   int64       `json:"timestamp"`
	BlockNumber uint64      `json:"block_number"`
	Action      AuditAction `json:"action"`
	Actor       string      `json:"actor"`
	CeremonyId  string      `json:"ceremony_id"`
	Details     string      `json:"details"`
}

// --- Params ---

// Params defines emergency module parameters.
type Params struct {
	// Quorum thresholds in basis points (1000000 = 100%)
	HaltQuorum   uint64 `json:"halt_quorum"`
	RevertQuorum uint64 `json:"revert_quorum"`
	ResumeQuorum uint64 `json:"resume_quorum"`

	// Ceremony timing (in blocks)
	HaltPrevoteBlocks    uint64 `json:"halt_prevote_blocks"`
	HaltPrecommitBlocks  uint64 `json:"halt_precommit_blocks"`
	HaltTimeoutBlocks    uint64 `json:"halt_timeout_blocks"`
	RevertPrevoteBlocks  uint64 `json:"revert_prevote_blocks"`
	RevertPrecommitBlocks uint64 `json:"revert_precommit_blocks"`
	RevertTimeoutBlocks  uint64 `json:"revert_timeout_blocks"`
	ResumePrevoteBlocks  uint64 `json:"resume_prevote_blocks"`
	ResumePrecommitBlocks uint64 `json:"resume_precommit_blocks"`
	ResumeTimeoutBlocks  uint64 `json:"resume_timeout_blocks"`

	// Anti-abuse limits
	MaxProposalsPerEpoch            uint64 `json:"max_proposals_per_epoch"`
	MaxProposalsPerGuardianPerEpoch uint64 `json:"max_proposals_per_guardian_per_epoch"`
	CooldownBlocks                  uint64 `json:"cooldown_blocks"`
	MinGuardianStake                string `json:"min_guardian_stake"`
	MinDistinctVoters               uint64 `json:"min_distinct_voters"`

	// Revert constraints
	MaxRevertDepth uint64 `json:"max_revert_depth"`

	// Epoch length (blocks)
	EpochBlocks uint64 `json:"epoch_blocks"`

	// Genesis emergency council (H-5)
	GenesisCouncil      []string `json:"genesis_council"`
	CouncilExpiryBlock  uint64   `json:"council_expiry_block"`
	CouncilVirtualStake string   `json:"council_virtual_stake"`
}

func (p *Params) Reset()         { *p = Params{} }
func (p *Params) String() string { return proto.CompactTextString(p) }
func (p *Params) ProtoMessage()  {}

// DefaultParams returns the default emergency module parameters.
func DefaultParams() Params {
	return Params{
		HaltQuorum:                      750000,  // 75%
		RevertQuorum:                    800000,  // 80%
		ResumeQuorum:                    800000,  // 80%
		HaltPrevoteBlocks:               11,
		HaltPrecommitBlocks:             11,
		HaltTimeoutBlocks:               44,
		RevertPrevoteBlocks:             22,
		RevertPrecommitBlocks:           22,
		RevertTimeoutBlocks:             111,
		ResumePrevoteBlocks:             22,
		ResumePrecommitBlocks:           22,
		ResumeTimeoutBlocks:             111,
		MaxProposalsPerEpoch:            3,
		MaxProposalsPerGuardianPerEpoch: 1,
		CooldownBlocks:                  111,
		MinGuardianStake:                "111111000000", // 111,111 LGM in ulgm
		MinDistinctVoters:               4,
		MaxRevertDepth:                  111111,
		EpochBlocks:                     34272, // ~1 day at 2521ms blocks
		GenesisCouncil:                  []string{},
		CouncilExpiryBlock:              0,
		CouncilVirtualStake:             "11111000000", // 11,111 LGM in ulgm
	}
}

// Validate performs basic param validation.
func (p Params) Validate() error {
	if p.HaltQuorum == 0 || p.HaltQuorum > 1000000 {
		return fmt.Errorf("halt quorum must be between 1 and 1000000")
	}
	if p.RevertQuorum == 0 || p.RevertQuorum > 1000000 {
		return fmt.Errorf("revert quorum must be between 1 and 1000000")
	}
	if p.ResumeQuorum == 0 || p.ResumeQuorum > 1000000 {
		return fmt.Errorf("resume quorum must be between 1 and 1000000")
	}
	if p.HaltTimeoutBlocks == 0 {
		return fmt.Errorf("halt timeout blocks must be positive")
	}
	if p.MinDistinctVoters == 0 {
		return fmt.Errorf("min distinct voters must be positive")
	}
	if p.MaxRevertDepth == 0 {
		return fmt.Errorf("max revert depth must be positive")
	}
	return nil
}

// --- Genesis ---

// GenesisState defines the emergency module's genesis state.
type GenesisState struct {
	Params     Params              `json:"params"`
	Status     EmergencyStatus     `json:"status"`
	Ceremonies []EmergencyCeremony `json:"ceremonies"`
	AuditLog   []EmergencyAuditEntry `json:"audit_log"`
}

func (gs *GenesisState) Reset()         { *gs = GenesisState{} }
func (gs *GenesisState) String() string { return proto.CompactTextString(gs) }
func (gs *GenesisState) ProtoMessage()  {}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		Status:     StatusNormal,
		Ceremonies: []EmergencyCeremony{},
		AuditLog:   []EmergencyAuditEntry{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

// --- Message Types ---

// MsgProposeHalt proposes an emergency chain halt.
type MsgProposeHalt struct {
	Proposer            string `json:"proposer"`
	Reason              string `json:"reason"`
	EvidenceHash        string `json:"evidence_hash"`
	SuggestedRevertBlock uint64 `json:"suggested_revert_block,omitempty"`
}

func (m *MsgProposeHalt) Reset()         { *m = MsgProposeHalt{} }
func (m *MsgProposeHalt) String() string { return proto.CompactTextString(m) }
func (m *MsgProposeHalt) ProtoMessage()  {}

func (m *MsgProposeHalt) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if m.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

func (m *MsgProposeHalt) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Proposer)
	return []sdk.AccAddress{addr}
}

// MsgProposeHaltResponse is the response.
type MsgProposeHaltResponse struct {
	CeremonyId string `json:"ceremony_id"`
}

func (m *MsgProposeHaltResponse) Reset()         { *m = MsgProposeHaltResponse{} }
func (m *MsgProposeHaltResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgProposeHaltResponse) ProtoMessage()  {}

// MsgVoteHalt votes on an active halt ceremony.
type MsgVoteHalt struct {
	Voter       string `json:"voter"`
	CeremonyId  string `json:"ceremony_id"`
	Approve     bool   `json:"approve"`
	IsPrecommit bool   `json:"is_precommit"`
	Signature   string `json:"signature"`
}

func (m *MsgVoteHalt) Reset()         { *m = MsgVoteHalt{} }
func (m *MsgVoteHalt) String() string { return proto.CompactTextString(m) }
func (m *MsgVoteHalt) ProtoMessage()  {}

func (m *MsgVoteHalt) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if m.CeremonyId == "" {
		return fmt.Errorf("ceremony_id is required")
	}
	return nil
}

func (m *MsgVoteHalt) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Voter)
	return []sdk.AccAddress{addr}
}

// MsgVoteHaltResponse is the response.
type MsgVoteHaltResponse struct {
	Phase     CeremonyPhase `json:"phase"`
	Finalized bool          `json:"finalized"`
}

func (m *MsgVoteHaltResponse) Reset()         { *m = MsgVoteHaltResponse{} }
func (m *MsgVoteHaltResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgVoteHaltResponse) ProtoMessage()  {}

// MsgProposeRevert proposes a state revert.
type MsgProposeRevert struct {
	Proposer          string `json:"proposer"`
	TargetBlockNumber uint64 `json:"target_block_number"`
	TargetBlockHash   string `json:"target_block_hash"`
	TargetStateRoot   string `json:"target_state_root"`
	Justification     string `json:"justification"`
	HaltCeremonyId    string `json:"halt_ceremony_id"`
}

func (m *MsgProposeRevert) Reset()         { *m = MsgProposeRevert{} }
func (m *MsgProposeRevert) String() string { return proto.CompactTextString(m) }
func (m *MsgProposeRevert) ProtoMessage()  {}

func (m *MsgProposeRevert) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if m.TargetBlockNumber == 0 {
		return fmt.Errorf("target block number must be positive")
	}
	if m.HaltCeremonyId == "" {
		return fmt.Errorf("halt_ceremony_id is required")
	}
	return nil
}

func (m *MsgProposeRevert) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Proposer)
	return []sdk.AccAddress{addr}
}

// MsgProposeRevertResponse is the response.
type MsgProposeRevertResponse struct {
	CeremonyId string `json:"ceremony_id"`
}

func (m *MsgProposeRevertResponse) Reset()         { *m = MsgProposeRevertResponse{} }
func (m *MsgProposeRevertResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgProposeRevertResponse) ProtoMessage()  {}

// MsgVoteRevert votes on an active revert ceremony.
type MsgVoteRevert struct {
	Voter       string `json:"voter"`
	CeremonyId  string `json:"ceremony_id"`
	Approve     bool   `json:"approve"`
	IsPrecommit bool   `json:"is_precommit"`
	Signature   string `json:"signature"`
}

func (m *MsgVoteRevert) Reset()         { *m = MsgVoteRevert{} }
func (m *MsgVoteRevert) String() string { return proto.CompactTextString(m) }
func (m *MsgVoteRevert) ProtoMessage()  {}

func (m *MsgVoteRevert) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if m.CeremonyId == "" {
		return fmt.Errorf("ceremony_id is required")
	}
	return nil
}

func (m *MsgVoteRevert) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Voter)
	return []sdk.AccAddress{addr}
}

// MsgVoteRevertResponse is the response.
type MsgVoteRevertResponse struct {
	Phase     CeremonyPhase `json:"phase"`
	Finalized bool          `json:"finalized"`
}

func (m *MsgVoteRevertResponse) Reset()         { *m = MsgVoteRevertResponse{} }
func (m *MsgVoteRevertResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgVoteRevertResponse) ProtoMessage()  {}

// MsgProposeResume proposes resuming chain operations.
type MsgProposeResume struct {
	Proposer         string `json:"proposer"`
	ResumeFromBlock  uint64 `json:"resume_from_block"`
	ResumeFromHash   string `json:"resume_from_hash"`
	ResumeStateRoot  string `json:"resume_state_root"`
	HaltCeremonyId   string `json:"halt_ceremony_id"`
	RevertCeremonyId string `json:"revert_ceremony_id,omitempty"`
}

func (m *MsgProposeResume) Reset()         { *m = MsgProposeResume{} }
func (m *MsgProposeResume) String() string { return proto.CompactTextString(m) }
func (m *MsgProposeResume) ProtoMessage()  {}

func (m *MsgProposeResume) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if m.HaltCeremonyId == "" {
		return fmt.Errorf("halt_ceremony_id is required")
	}
	return nil
}

func (m *MsgProposeResume) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Proposer)
	return []sdk.AccAddress{addr}
}

// MsgProposeResumeResponse is the response.
type MsgProposeResumeResponse struct {
	CeremonyId string `json:"ceremony_id"`
}

func (m *MsgProposeResumeResponse) Reset()         { *m = MsgProposeResumeResponse{} }
func (m *MsgProposeResumeResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgProposeResumeResponse) ProtoMessage()  {}

// MsgVoteResume votes on an active resume ceremony.
type MsgVoteResume struct {
	Voter       string `json:"voter"`
	CeremonyId  string `json:"ceremony_id"`
	Approve     bool   `json:"approve"`
	IsPrecommit bool   `json:"is_precommit"`
	Signature   string `json:"signature"`
}

func (m *MsgVoteResume) Reset()         { *m = MsgVoteResume{} }
func (m *MsgVoteResume) String() string { return proto.CompactTextString(m) }
func (m *MsgVoteResume) ProtoMessage()  {}

func (m *MsgVoteResume) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if m.CeremonyId == "" {
		return fmt.Errorf("ceremony_id is required")
	}
	return nil
}

func (m *MsgVoteResume) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Voter)
	return []sdk.AccAddress{addr}
}

// MsgVoteResumeResponse is the response.
type MsgVoteResumeResponse struct {
	Phase     CeremonyPhase `json:"phase"`
	Finalized bool          `json:"finalized"`
}

func (m *MsgVoteResumeResponse) Reset()         { *m = MsgVoteResumeResponse{} }
func (m *MsgVoteResumeResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgVoteResumeResponse) ProtoMessage()  {}

// --- Service Interfaces ---

// MsgServer defines the emergency module's message server interface.
type MsgServer interface {
	ProposeHalt(context.Context, *MsgProposeHalt) (*MsgProposeHaltResponse, error)
	VoteHalt(context.Context, *MsgVoteHalt) (*MsgVoteHaltResponse, error)
	ProposeRevert(context.Context, *MsgProposeRevert) (*MsgProposeRevertResponse, error)
	VoteRevert(context.Context, *MsgVoteRevert) (*MsgVoteRevertResponse, error)
	ProposeResume(context.Context, *MsgProposeResume) (*MsgProposeResumeResponse, error)
	VoteResume(context.Context, *MsgVoteResume) (*MsgVoteResumeResponse, error)
}

// QueryServer defines the emergency module's query server interface.
type QueryServer interface {
	Status(context.Context, *QueryStatusRequest) (*QueryStatusResponse, error)
	ActiveCeremony(context.Context, *QueryActiveCeremonyRequest) (*QueryActiveCeremonyResponse, error)
	CompletedCeremonies(context.Context, *QueryCompletedCeremoniesRequest) (*QueryCompletedCeremoniesResponse, error)
	AuditLog(context.Context, *QueryAuditLogRequest) (*QueryAuditLogResponse, error)
}

// --- Query Types ---

type QueryStatusRequest struct{}
type QueryStatusResponse struct {
	Status                EmergencyStatus `json:"status"`
	IsHalted              bool            `json:"is_halted"`
	ActiveHaltCeremonyId  string          `json:"active_halt_ceremony_id"`
}

type QueryActiveCeremonyRequest struct{}
type QueryActiveCeremonyResponse struct {
	Ceremony *EmergencyCeremony `json:"ceremony,omitempty"`
	Found    bool               `json:"found"`
}

type QueryCompletedCeremoniesRequest struct {
	Limit  uint32 `json:"limit"`
	Offset uint32 `json:"offset"`
}
type QueryCompletedCeremoniesResponse struct {
	Ceremonies []EmergencyCeremony `json:"ceremonies"`
	Total      uint64              `json:"total"`
}

type QueryAuditLogRequest struct {
	Limit  uint32 `json:"limit"`
	Offset uint32 `json:"offset"`
}
type QueryAuditLogResponse struct {
	Entries []EmergencyAuditEntry `json:"entries"`
	Total   uint64                `json:"total"`
}

// --- IsEmergencyMsg ---

// IsEmergencyMsg returns true if the message is an emergency module message.
func IsEmergencyMsg(msg sdk.Msg) bool {
	switch msg.(type) {
	case *MsgProposeHalt, *MsgVoteHalt,
		*MsgProposeRevert, *MsgVoteRevert,
		*MsgProposeResume, *MsgVoteResume:
		return true
	default:
		return false
	}
}

// --- Codec Registration ---

// RegisterCodec registers the emergency module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgProposeHalt{}, "emergency/ProposeHalt", nil)
	cdc.RegisterConcrete(&MsgVoteHalt{}, "emergency/VoteHalt", nil)
	cdc.RegisterConcrete(&MsgProposeRevert{}, "emergency/ProposeRevert", nil)
	cdc.RegisterConcrete(&MsgVoteRevert{}, "emergency/VoteRevert", nil)
	cdc.RegisterConcrete(&MsgProposeResume{}, "emergency/ProposeResume", nil)
	cdc.RegisterConcrete(&MsgVoteResume{}, "emergency/VoteResume", nil)
}

// RegisterInterfaces registers the emergency module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

// gRPC service descriptors for msg server
var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.emergency.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "ProposeHalt", Handler: _Msg_ProposeHalt_Handler},
		{MethodName: "VoteHalt", Handler: _Msg_VoteHalt_Handler},
		{MethodName: "ProposeRevert", Handler: _Msg_ProposeRevert_Handler},
		{MethodName: "VoteRevert", Handler: _Msg_VoteRevert_Handler},
		{MethodName: "ProposeResume", Handler: _Msg_ProposeResume_Handler},
		{MethodName: "VoteResume", Handler: _Msg_VoteResume_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Msg_ProposeHalt_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgProposeHalt)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ProposeHalt(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Msg/ProposeHalt"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ProposeHalt(ctx, req.(*MsgProposeHalt))
	})
}

func _Msg_VoteHalt_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgVoteHalt)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).VoteHalt(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Msg/VoteHalt"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).VoteHalt(ctx, req.(*MsgVoteHalt))
	})
}

func _Msg_ProposeRevert_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgProposeRevert)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ProposeRevert(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Msg/ProposeRevert"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ProposeRevert(ctx, req.(*MsgProposeRevert))
	})
}

func _Msg_VoteRevert_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgVoteRevert)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).VoteRevert(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Msg/VoteRevert"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).VoteRevert(ctx, req.(*MsgVoteRevert))
	})
}

func _Msg_ProposeResume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgProposeResume)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ProposeResume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Msg/ProposeResume"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ProposeResume(ctx, req.(*MsgProposeResume))
	})
}

func _Msg_VoteResume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgVoteResume)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).VoteResume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Msg/VoteResume"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).VoteResume(ctx, req.(*MsgVoteResume))
	})
}

// RegisterMsgServer registers the emergency msg server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// gRPC service descriptors for query server
var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.emergency.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Status", Handler: _Query_Status_Handler},
		{MethodName: "ActiveCeremony", Handler: _Query_ActiveCeremony_Handler},
		{MethodName: "CompletedCeremonies", Handler: _Query_CompletedCeremonies_Handler},
		{MethodName: "AuditLog", Handler: _Query_AuditLog_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Query_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Query/Status"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Status(ctx, req.(*QueryStatusRequest))
	})
}

func _Query_ActiveCeremony_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryActiveCeremonyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ActiveCeremony(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Query/ActiveCeremony"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ActiveCeremony(ctx, req.(*QueryActiveCeremonyRequest))
	})
}

func _Query_CompletedCeremonies_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryCompletedCeremoniesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).CompletedCeremonies(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Query/CompletedCeremonies"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).CompletedCeremonies(ctx, req.(*QueryCompletedCeremoniesRequest))
	})
}

func _Query_AuditLog_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAuditLogRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AuditLog(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Query/AuditLog"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AuditLog(ctx, req.(*QueryAuditLogRequest))
	})
}

// RegisterQueryServer registers the emergency query server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
