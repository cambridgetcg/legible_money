package types

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// --- EmergencyCeremony helpers ---
// These provide map-like access over the proto slice-based Prevotes/Precommits fields.

// GetPrevote returns the prevote for a given voter, and whether it exists.
func (c *EmergencyCeremony) GetPrevote(voter string) (*EmergencyVote, bool) {
	for _, entry := range c.Prevotes {
		if entry != nil && entry.Key == voter {
			return entry.Value, true
		}
	}
	return nil, false
}

// SetPrevote sets or replaces a prevote for a given voter.
func (c *EmergencyCeremony) SetPrevote(voter string, vote EmergencyVote) {
	for _, entry := range c.Prevotes {
		if entry != nil && entry.Key == voter {
			entry.Value = &vote
			return
		}
	}
	c.Prevotes = append(c.Prevotes, &PrevoteEntry{Key: voter, Value: &vote})
}

// GetPrecommit returns the precommit for a given voter, and whether it exists.
func (c *EmergencyCeremony) GetPrecommit(voter string) (*EmergencyPrecommit, bool) {
	for _, entry := range c.Precommits {
		if entry != nil && entry.Key == voter {
			return entry.Value, true
		}
	}
	return nil, false
}

// SetPrecommit sets or replaces a precommit for a given voter.
func (c *EmergencyCeremony) SetPrecommit(voter string, precommit EmergencyPrecommit) {
	for _, entry := range c.Precommits {
		if entry != nil && entry.Key == voter {
			entry.Value = &precommit
			return
		}
	}
	c.Precommits = append(c.Precommits, &PrecommitEntry{Key: voter, Value: &precommit})
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
	if p.MaxHaltDurationBlocks == 0 {
		return fmt.Errorf("max halt duration blocks must be positive (prevents permanent chain freeze)")
	}
	return nil
}

// --- Message Validation ---

// ValidateBasic validates MsgProposeHalt fields.
func (m *MsgProposeHalt) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if m.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

// GetSigners returns the expected signers for MsgProposeHalt.
func (m *MsgProposeHalt) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Proposer)
	return []sdk.AccAddress{addr}
}

// ValidateBasic validates MsgVoteHalt fields.
func (m *MsgVoteHalt) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if m.ProposalId == "" {
		return fmt.Errorf("proposal_id is required")
	}
	return nil
}

// GetSigners returns the expected signers for MsgVoteHalt.
func (m *MsgVoteHalt) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Voter)
	return []sdk.AccAddress{addr}
}

// ValidateBasic validates MsgProposeRevert fields.
func (m *MsgProposeRevert) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if m.RevertToHeight == 0 {
		return fmt.Errorf("revert_to_height must be positive")
	}
	return nil
}

// GetSigners returns the expected signers for MsgProposeRevert.
func (m *MsgProposeRevert) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Proposer)
	return []sdk.AccAddress{addr}
}

// ValidateBasic validates MsgVoteRevert fields.
func (m *MsgVoteRevert) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if m.ProposalId == "" {
		return fmt.Errorf("proposal_id is required")
	}
	return nil
}

// GetSigners returns the expected signers for MsgVoteRevert.
func (m *MsgVoteRevert) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Voter)
	return []sdk.AccAddress{addr}
}

// ValidateBasic validates MsgProposeResume fields.
func (m *MsgProposeResume) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	return nil
}

// GetSigners returns the expected signers for MsgProposeResume.
func (m *MsgProposeResume) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Proposer)
	return []sdk.AccAddress{addr}
}

// ValidateBasic validates MsgVoteResume fields.
func (m *MsgVoteResume) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if m.ProposalId == "" {
		return fmt.Errorf("proposal_id is required")
	}
	return nil
}

// GetSigners returns the expected signers for MsgVoteResume.
func (m *MsgVoteResume) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Voter)
	return []sdk.AccAddress{addr}
}

// --- MsgUpdateParams (hand-written, not in proto) ---

// MsgUpdateParams updates module parameters (governance-gated).
type MsgUpdateParams struct {
	Authority string `json:"authority"` // gov module address
	Params    Params `json:"params"`
}

func (msg *MsgUpdateParams) Reset()        {}
func (msg *MsgUpdateParams) String() string { return "MsgUpdateParams" }
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
	return msg.Params.Validate()
}

// MsgUpdateParamsResponse is the response for MsgUpdateParams.
type MsgUpdateParamsResponse struct{}

func (msg *MsgUpdateParamsResponse) Reset()        {}
func (msg *MsgUpdateParamsResponse) String() string { return "" }
func (msg *MsgUpdateParamsResponse) ProtoMessage()  {}

// --- Query Server Interface ---
// (No query_grpc.pb.go exists, so hand-written QueryServer is needed.)

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
	cdc.RegisterConcrete(&MsgUpdateParams{}, "emergency/UpdateParams", nil)
}

// RegisterInterfaces registers the emergency module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgProposeHalt{},
		&MsgVoteHalt{},
		&MsgProposeRevert{},
		&MsgVoteRevert{},
		&MsgProposeResume{},
		&MsgVoteResume{},
		&MsgUpdateParams{},
	)
}

// gRPC service descriptors for query server
// (No query_grpc.pb.go exists, so keep hand-written query service registration.)
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

// --- MsgUpdateParams gRPC handler (hand-written extension to proto-generated Msg service) ---

// MsgServerUpdateParams extends the proto-generated MsgServer with UpdateParams.
// Implementations that support governance param updates should implement this interface.
type MsgServerUpdateParams interface {
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}

func _Msg_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServerUpdateParams).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.emergency.v1.Msg/UpdateParams"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServerUpdateParams).UpdateParams(ctx, req.(*MsgUpdateParams))
	})
}

func init() {
	Msg_ServiceDesc.Methods = append(Msg_ServiceDesc.Methods, grpc.MethodDesc{
		MethodName: "UpdateParams",
		Handler:    _Msg_UpdateParams_Handler,
	})
}
