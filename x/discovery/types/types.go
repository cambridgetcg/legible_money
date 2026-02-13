package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// --- Params Validation ---
// Methods below attach to protobuf-generated types from state.pb.go and genesis.pb.go.

// Validate performs parameter validation.
func (p Params) Validate() error {
	if p.MaxCapabilitiesPerAgent == 0 {
		return fmt.Errorf("max_capabilities_per_agent must be positive")
	}
	if p.ProfileExpiryBlocks == 0 {
		return fmt.Errorf("profile_expiry_blocks must be positive")
	}
	return nil
}

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		MinRegistrationStake:    "100000000", // 100 LGM
		MaxCapabilitiesPerAgent: 50,
		ProfileExpiryBlocks:     342720, // ~10 days at 2521ms blocks
	}
}

// --- Genesis state validation ---

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:   &p,
		Profiles: []*AgentProfile{},
	}
}

// Validate performs genesis state validation.
func (gs GenesisState) Validate() error {
	if gs.Params != nil {
		return gs.Params.Validate()
	}
	return nil
}

// -----------------------------------------------------------------------
// Message types (not in proto-generated files)
// -----------------------------------------------------------------------

// MsgRegisterAgent registers a new agent profile.
type MsgRegisterAgent struct {
	Address     string            `json:"address"`
	Did         string            `json:"did"`
	DisplayName string            `json:"display_name"`
	Capabilities []AgentCapability `json:"capabilities"`
	Domains     []string          `json:"domains"`
	Stake       string            `json:"stake"`
	Description string            `json:"description"`
	Metadata    string            `json:"metadata"`
}

func (msg *MsgRegisterAgent) Reset()        {}
func (msg *MsgRegisterAgent) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRegisterAgent) ProtoMessage()  {}

func (msg *MsgRegisterAgent) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterAgent) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	if msg.DisplayName == "" {
		return fmt.Errorf("display name cannot be empty")
	}
	return nil
}

// MsgRegisterAgentResponse is the response.
type MsgRegisterAgentResponse struct{}

func (msg *MsgRegisterAgentResponse) Reset()        {}
func (msg *MsgRegisterAgentResponse) String() string { return "" }
func (msg *MsgRegisterAgentResponse) ProtoMessage()  {}

// MsgUpdateProfile updates an agent's profile.
type MsgUpdateProfile struct {
	Address     string `json:"address"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
}

func (msg *MsgUpdateProfile) Reset()        {}
func (msg *MsgUpdateProfile) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgUpdateProfile) ProtoMessage()  {}

func (msg *MsgUpdateProfile) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateProfile) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	return nil
}

// MsgUpdateProfileResponse is the response.
type MsgUpdateProfileResponse struct{}

func (msg *MsgUpdateProfileResponse) Reset()        {}
func (msg *MsgUpdateProfileResponse) String() string { return "" }
func (msg *MsgUpdateProfileResponse) ProtoMessage()  {}

// MsgAddCapability adds a capability to an agent's profile.
type MsgAddCapability struct {
	Address    string          `json:"address"`
	Capability AgentCapability `json:"capability"`
}

func (msg *MsgAddCapability) Reset()        {}
func (msg *MsgAddCapability) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgAddCapability) ProtoMessage()  {}

func (msg *MsgAddCapability) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAddCapability) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	if msg.Capability.CapabilityType == "" {
		return fmt.Errorf("capability type cannot be empty")
	}
	return nil
}

// MsgAddCapabilityResponse is the response.
type MsgAddCapabilityResponse struct{}

func (msg *MsgAddCapabilityResponse) Reset()        {}
func (msg *MsgAddCapabilityResponse) String() string { return "" }
func (msg *MsgAddCapabilityResponse) ProtoMessage()  {}

// MsgRemoveCapability removes a capability from an agent's profile.
type MsgRemoveCapability struct {
	Address        string `json:"address"`
	CapabilityType string `json:"capability_type"`
}

func (msg *MsgRemoveCapability) Reset()        {}
func (msg *MsgRemoveCapability) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRemoveCapability) ProtoMessage()  {}

func (msg *MsgRemoveCapability) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRemoveCapability) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	if msg.CapabilityType == "" {
		return fmt.Errorf("capability type cannot be empty")
	}
	return nil
}

// MsgRemoveCapabilityResponse is the response.
type MsgRemoveCapabilityResponse struct{}

func (msg *MsgRemoveCapabilityResponse) Reset()        {}
func (msg *MsgRemoveCapabilityResponse) String() string { return "" }
func (msg *MsgRemoveCapabilityResponse) ProtoMessage()  {}

// MsgDeregisterAgent removes an agent and returns stake.
type MsgDeregisterAgent struct {
	Address string `json:"address"`
}

func (msg *MsgDeregisterAgent) Reset()        {}
func (msg *MsgDeregisterAgent) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgDeregisterAgent) ProtoMessage()  {}

func (msg *MsgDeregisterAgent) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgDeregisterAgent) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	return nil
}

// MsgDeregisterAgentResponse is the response.
type MsgDeregisterAgentResponse struct {
	RefundedStake string `json:"refunded_stake"`
}

func (msg *MsgDeregisterAgentResponse) Reset()        {}
func (msg *MsgDeregisterAgentResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgDeregisterAgentResponse) ProtoMessage()  {}

// MsgHeartbeat signals that an agent is still active.
type MsgHeartbeat struct {
	Address string `json:"address"`
}

func (msg *MsgHeartbeat) Reset()        {}
func (msg *MsgHeartbeat) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgHeartbeat) ProtoMessage()  {}

func (msg *MsgHeartbeat) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgHeartbeat) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	return nil
}

// MsgHeartbeatResponse is the response.
type MsgHeartbeatResponse struct{}

func (msg *MsgHeartbeatResponse) Reset()        {}
func (msg *MsgHeartbeatResponse) String() string { return "" }
func (msg *MsgHeartbeatResponse) ProtoMessage()  {}

// MsgUpdateParams updates module parameters (governance-gated).
type MsgUpdateParams struct {
	Authority string `json:"authority"`
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
	return msg.Params.Validate()
}

// MsgUpdateParamsResponse is the response.
type MsgUpdateParamsResponse struct{}

func (msg *MsgUpdateParamsResponse) Reset()        {}
func (msg *MsgUpdateParamsResponse) String() string { return "" }
func (msg *MsgUpdateParamsResponse) ProtoMessage()  {}

// -----------------------------------------------------------------------
// MsgServer interface
// -----------------------------------------------------------------------

type MsgServer interface {
	RegisterAgent(context.Context, *MsgRegisterAgent) (*MsgRegisterAgentResponse, error)
	UpdateProfile(context.Context, *MsgUpdateProfile) (*MsgUpdateProfileResponse, error)
	AddCapability(context.Context, *MsgAddCapability) (*MsgAddCapabilityResponse, error)
	RemoveCapability(context.Context, *MsgRemoveCapability) (*MsgRemoveCapabilityResponse, error)
	DeregisterAgent(context.Context, *MsgDeregisterAgent) (*MsgDeregisterAgentResponse, error)
	Heartbeat(context.Context, *MsgHeartbeat) (*MsgHeartbeatResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}

// -----------------------------------------------------------------------
// gRPC service descriptor
// -----------------------------------------------------------------------

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.discovery.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "RegisterAgent", Handler: _Msg_RegisterAgent_Handler},
		{MethodName: "UpdateProfile", Handler: _Msg_UpdateProfile_Handler},
		{MethodName: "AddCapability", Handler: _Msg_AddCapability_Handler},
		{MethodName: "RemoveCapability", Handler: _Msg_RemoveCapability_Handler},
		{MethodName: "DeregisterAgent", Handler: _Msg_DeregisterAgent_Handler},
		{MethodName: "Heartbeat", Handler: _Msg_Heartbeat_Handler},
		{MethodName: "UpdateParams", Handler: _Msg_UpdateParams_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/discovery/v1/tx.proto",
}

func _Msg_RegisterAgent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterAgent)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RegisterAgent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Msg/RegisterAgent"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RegisterAgent(ctx, req.(*MsgRegisterAgent))
	})
}

func _Msg_UpdateProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateProfile)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Msg/UpdateProfile"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateProfile(ctx, req.(*MsgUpdateProfile))
	})
}

func _Msg_AddCapability_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAddCapability)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AddCapability(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Msg/AddCapability"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AddCapability(ctx, req.(*MsgAddCapability))
	})
}

func _Msg_RemoveCapability_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRemoveCapability)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RemoveCapability(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Msg/RemoveCapability"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RemoveCapability(ctx, req.(*MsgRemoveCapability))
	})
}

func _Msg_DeregisterAgent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgDeregisterAgent)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).DeregisterAgent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Msg/DeregisterAgent"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).DeregisterAgent(ctx, req.(*MsgDeregisterAgent))
	})
}

func _Msg_Heartbeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgHeartbeat)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).Heartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Msg/Heartbeat"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).Heartbeat(ctx, req.(*MsgHeartbeat))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Msg/UpdateParams"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	})
}

// RegisterMsgServer registers the MsgServer implementation.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// -----------------------------------------------------------------------
// QueryServer (not in proto-generated files)
// -----------------------------------------------------------------------

type QueryServer interface {
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	Agent(context.Context, *QueryAgentRequest) (*QueryAgentResponse, error)
	AgentsByDomain(context.Context, *QueryAgentsByDomainRequest) (*QueryAgentsByDomainResponse, error)
	AgentsByCapability(context.Context, *QueryAgentsByCapabilityRequest) (*QueryAgentsByCapabilityResponse, error)
}

type QueryParamsRequest struct{}
func (r *QueryParamsRequest) Reset()        {}
func (r *QueryParamsRequest) String() string { return "QueryParamsRequest" }
func (r *QueryParamsRequest) ProtoMessage()  {}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}
func (r *QueryParamsResponse) Reset()        {}
func (r *QueryParamsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryParamsResponse) ProtoMessage()  {}

type QueryAgentRequest struct {
	Address string `json:"address"`
}
func (r *QueryAgentRequest) Reset()        {}
func (r *QueryAgentRequest) String() string { return fmt.Sprintf("QueryAgent{%s}", r.Address) }
func (r *QueryAgentRequest) ProtoMessage()  {}

type QueryAgentResponse struct {
	Profile AgentProfile `json:"profile"`
}
func (r *QueryAgentResponse) Reset()        {}
func (r *QueryAgentResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryAgentResponse) ProtoMessage()  {}

type QueryAgentsByDomainRequest struct {
	Domain string `json:"domain"`
}
func (r *QueryAgentsByDomainRequest) Reset()        {}
func (r *QueryAgentsByDomainRequest) String() string { return fmt.Sprintf("QueryAgentsByDomain{%s}", r.Domain) }
func (r *QueryAgentsByDomainRequest) ProtoMessage()  {}

type QueryAgentsByDomainResponse struct {
	Profiles []AgentProfile `json:"profiles"`
}
func (r *QueryAgentsByDomainResponse) Reset()        {}
func (r *QueryAgentsByDomainResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryAgentsByDomainResponse) ProtoMessage()  {}

type QueryAgentsByCapabilityRequest struct {
	CapabilityType string `json:"capability_type"`
}
func (r *QueryAgentsByCapabilityRequest) Reset()        {}
func (r *QueryAgentsByCapabilityRequest) String() string { return fmt.Sprintf("QueryAgentsByCapability{%s}", r.CapabilityType) }
func (r *QueryAgentsByCapabilityRequest) ProtoMessage()  {}

type QueryAgentsByCapabilityResponse struct {
	Profiles []AgentProfile `json:"profiles"`
}
func (r *QueryAgentsByCapabilityResponse) Reset()        {}
func (r *QueryAgentsByCapabilityResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryAgentsByCapabilityResponse) ProtoMessage()  {}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

func _Query_Agent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAgentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Agent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Query/Agent"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Agent(ctx, req.(*QueryAgentRequest))
	})
}

func _Query_AgentsByDomain_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAgentsByDomainRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AgentsByDomain(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Query/AgentsByDomain"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AgentsByDomain(ctx, req.(*QueryAgentsByDomainRequest))
	})
}

func _Query_AgentsByCapability_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAgentsByCapabilityRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AgentsByCapability(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.discovery.v1.Query/AgentsByCapability"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AgentsByCapability(ctx, req.(*QueryAgentsByCapabilityRequest))
	})
}

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.discovery.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Params", Handler: _Query_Params_Handler},
		{MethodName: "Agent", Handler: _Query_Agent_Handler},
		{MethodName: "AgentsByDomain", Handler: _Query_AgentsByDomain_Handler},
		{MethodName: "AgentsByCapability", Handler: _Query_AgentsByCapability_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/discovery/v1/query.proto",
}

func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
