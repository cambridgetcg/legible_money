package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// --- ValidateBasic methods for protobuf message types ---
// Struct definitions for RemoteAccount, InterchainAccountRecord, Params, and
// GenesisState are provided by the proto-generated .pb.go files (state.pb.go
// and genesis.pb.go).  Only methods and non-proto types are kept here.

// Validate validates the module parameters.
func (p Params) Validate() error {
	if p.MaxRemoteAccountsPerDid == 0 {
		return fmt.Errorf("max_remote_accounts_per_did must be positive")
	}
	if p.RegistrationCooldown == 0 {
		return fmt.Errorf("registration_cooldown must be positive")
	}
	return nil
}

// --- Messages ---

// MsgRegisterInterchainAccount requests registration of an ICA on a remote chain.
// Must be signed by the account's operational key (not a session key).
type MsgRegisterInterchainAccount struct {
	Owner        string   `json:"owner"` // bech32 address on Legible
	ConnectionId string   `json:"connection_id"`
	Version      string   `json:"version"`      // ICA version string
	AllowedMsgs  []string `json:"allowed_msgs"` // proto URL allowlist
}

func (msg MsgRegisterInterchainAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{sender}
}

func (msg MsgRegisterInterchainAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.ConnectionId == "" {
		return fmt.Errorf("connection_id cannot be empty")
	}
	return nil
}

func (msg MsgRegisterInterchainAccount) Reset() {}
func (msg MsgRegisterInterchainAccount) String() string {
	return fmt.Sprintf("MsgRegisterICA{%s}", msg.Owner)
}
func (msg MsgRegisterInterchainAccount) ProtoMessage() {}

// MsgRegisterInterchainAccountResponse is the response type.
type MsgRegisterInterchainAccountResponse struct {
	ChannelId string `json:"channel_id"`
	PortId    string `json:"port_id"`
}

func (msg MsgRegisterInterchainAccountResponse) Reset()         {}
func (msg MsgRegisterInterchainAccountResponse) String() string { return "MsgRegisterICAResponse{}" }
func (msg MsgRegisterInterchainAccountResponse) ProtoMessage()  {}

// MsgSendICATransaction sends a transaction to a remote chain via ICA.
type MsgSendICATransaction struct {
	Owner        string `json:"owner"` // bech32 address on Legible
	ConnectionId string `json:"connection_id"`
	MsgBytes     []byte `json:"msg_bytes"`     // serialized CosmosTx
	TimeoutNanos uint64 `json:"timeout_nanos"` // relative timeout in nanoseconds
}

func (msg MsgSendICATransaction) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{sender}
}

func (msg MsgSendICATransaction) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.ConnectionId == "" {
		return fmt.Errorf("connection_id cannot be empty")
	}
	if len(msg.MsgBytes) == 0 {
		return fmt.Errorf("msg_bytes cannot be empty")
	}
	return nil
}

func (msg MsgSendICATransaction) Reset()         {}
func (msg MsgSendICATransaction) String() string { return fmt.Sprintf("MsgSendICATx{%s}", msg.Owner) }
func (msg MsgSendICATransaction) ProtoMessage()  {}

// MsgSendICATransactionResponse is the response type.
type MsgSendICATransactionResponse struct {
	Sequence uint64 `json:"sequence"`
}

func (msg MsgSendICATransactionResponse) Reset()         {}
func (msg MsgSendICATransactionResponse) String() string { return "MsgSendICATxResponse{}" }
func (msg MsgSendICATransactionResponse) ProtoMessage()  {}

// --- Query types ---

// QueryServer defines the gRPC query server for the icaauth module.
type QueryServer interface {
	InterchainAccounts(ctx context.Context, req *QueryInterchainAccountsRequest) (*QueryInterchainAccountsResponse, error)
	Params(ctx context.Context, req *QueryParamsRequest) (*QueryParamsResponse, error)
}

// QueryInterchainAccountsRequest queries all ICA accounts for a DID.
type QueryInterchainAccountsRequest struct {
	Did string `json:"did"`
}

func (r *QueryInterchainAccountsRequest) Reset()         {}
func (r *QueryInterchainAccountsRequest) String() string { return fmt.Sprintf("QueryICA{%s}", r.Did) }
func (r *QueryInterchainAccountsRequest) ProtoMessage()  {}

// QueryInterchainAccountsResponse returns the ICA accounts for a DID.
type QueryInterchainAccountsResponse struct {
	Accounts []*RemoteAccount `json:"accounts"`
}

func (r *QueryInterchainAccountsResponse) Reset() {}
func (r *QueryInterchainAccountsResponse) String() string {
	return fmt.Sprintf("QueryICAResp{%d}", len(r.Accounts))
}
func (r *QueryInterchainAccountsResponse) ProtoMessage() {}

// QueryParamsRequest queries module parameters.
type QueryParamsRequest struct{}

func (r *QueryParamsRequest) Reset()         {}
func (r *QueryParamsRequest) String() string { return "QueryParamsRequest{}" }
func (r *QueryParamsRequest) ProtoMessage()  {}

// QueryParamsResponse returns module parameters.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (r *QueryParamsResponse) Reset()         {}
func (r *QueryParamsResponse) String() string { return "QueryParamsResponse{}" }
func (r *QueryParamsResponse) ProtoMessage()  {}

// --- MsgServer interface ---

// MsgServer defines the gRPC msg server for the icaauth module.
type MsgServer interface {
	RegisterInterchainAccount(context.Context, *MsgRegisterInterchainAccount) (*MsgRegisterInterchainAccountResponse, error)
	SendICATransaction(context.Context, *MsgSendICATransaction) (*MsgSendICATransactionResponse, error)
}

// --- gRPC service descriptors and handlers ---

func _Msg_RegisterInterchainAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterInterchainAccount)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RegisterInterchainAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.icaauth.v1.Msg/RegisterInterchainAccount"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RegisterInterchainAccount(ctx, req.(*MsgRegisterInterchainAccount))
	})
}

func _Msg_SendICATransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSendICATransaction)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SendICATransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.icaauth.v1.Msg/SendICATransaction"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SendICATransaction(ctx, req.(*MsgSendICATransaction))
	})
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.icaauth.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "RegisterInterchainAccount", Handler: _Msg_RegisterInterchainAccount_Handler},
		{MethodName: "SendICATransaction", Handler: _Msg_SendICATransaction_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/icaauth/v1/tx.proto",
}

// RegisterMsgServer registers the MsgServer implementation.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// --- Query gRPC handlers ---

func _Query_InterchainAccounts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryInterchainAccountsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).InterchainAccounts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.icaauth.v1.Query/InterchainAccounts"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).InterchainAccounts(ctx, req.(*QueryInterchainAccountsRequest))
	})
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.icaauth.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.icaauth.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "InterchainAccounts", Handler: _Query_InterchainAccounts_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/icaauth/v1/query.proto",
}

// RegisterQueryServer registers the QueryServer implementation.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
