package types

import (
	"context"
	"fmt"

	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc"
)

// Query request/response types

// QueryAccountRequest is the request for the Account query.
type QueryAccountRequest struct {
	Address string `json:"address"`
}

// QueryAccountResponse is the response for the Account query.
type QueryAccountResponse struct {
	Account *LegibleAccount `json:"account"`
}

// QueryAccountByDIDRequest is the request for the AccountByDID query.
type QueryAccountByDIDRequest struct {
	Did string `json:"did"`
}

// QueryAccountByDIDResponse is the response for the AccountByDID query.
type QueryAccountByDIDResponse struct {
	Account *LegibleAccount `json:"account"`
}

// QuerySessionKeysRequest is the request for the SessionKeys query.
type QuerySessionKeysRequest struct {
	Owner string `json:"owner"`
}

// QuerySessionKeysResponse is the response for the SessionKeys query.
type QuerySessionKeysResponse struct {
	SessionKeys []SessionKey `json:"session_keys"`
}

// QueryParamsRequest is the request for the Params query.
type QueryParamsRequest struct{}

// QueryParamsResponse is the response for the Params query.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

// proto.Message stubs
func (m QueryAccountRequest) Reset()         {}
func (m QueryAccountRequest) String() string { return fmt.Sprintf("QueryAccountRequest{%s}", m.Address) }
func (m QueryAccountRequest) ProtoMessage()  {}

func (m QueryAccountResponse) Reset()         {}
func (m QueryAccountResponse) String() string { return "QueryAccountResponse{}" }
func (m QueryAccountResponse) ProtoMessage()  {}

func (m QueryAccountByDIDRequest) Reset()         {}
func (m QueryAccountByDIDRequest) String() string { return fmt.Sprintf("QueryAccountByDIDRequest{%s}", m.Did) }
func (m QueryAccountByDIDRequest) ProtoMessage()  {}

func (m QueryAccountByDIDResponse) Reset()         {}
func (m QueryAccountByDIDResponse) String() string { return "QueryAccountByDIDResponse{}" }
func (m QueryAccountByDIDResponse) ProtoMessage()  {}

func (m QuerySessionKeysRequest) Reset()         {}
func (m QuerySessionKeysRequest) String() string { return fmt.Sprintf("QuerySessionKeysRequest{%s}", m.Owner) }
func (m QuerySessionKeysRequest) ProtoMessage()  {}

func (m QuerySessionKeysResponse) Reset()         {}
func (m QuerySessionKeysResponse) String() string { return "QuerySessionKeysResponse{}" }
func (m QuerySessionKeysResponse) ProtoMessage()  {}

func (m QueryParamsRequest) Reset()         {}
func (m QueryParamsRequest) String() string { return "QueryParamsRequest{}" }
func (m QueryParamsRequest) ProtoMessage()  {}

func (m QueryParamsResponse) Reset()         {}
func (m QueryParamsResponse) String() string { return "QueryParamsResponse{}" }
func (m QueryParamsResponse) ProtoMessage()  {}

var _ proto.Message = (*QueryAccountRequest)(nil)
var _ proto.Message = (*QueryAccountResponse)(nil)
var _ proto.Message = (*QueryAccountByDIDRequest)(nil)
var _ proto.Message = (*QueryAccountByDIDResponse)(nil)
var _ proto.Message = (*QuerySessionKeysRequest)(nil)
var _ proto.Message = (*QuerySessionKeysResponse)(nil)
var _ proto.Message = (*QueryParamsRequest)(nil)
var _ proto.Message = (*QueryParamsResponse)(nil)

// QueryServer defines the query server interface for the legible_auth module.
type QueryServer interface {
	Account(context.Context, *QueryAccountRequest) (*QueryAccountResponse, error)
	AccountByDID(context.Context, *QueryAccountByDIDRequest) (*QueryAccountByDIDResponse, error)
	SessionKeys(context.Context, *QuerySessionKeysRequest) (*QuerySessionKeysResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// RegisterQueryServer registers the legible_auth query server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}

// gRPC service descriptor for query server
var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.auth.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Account", Handler: _Query_Account_Handler},
		{MethodName: "AccountByDID", Handler: _Query_AccountByDID_Handler},
		{MethodName: "SessionKeys", Handler: _Query_SessionKeys_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Query_Account_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Account(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Query/Account"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Account(ctx, req.(*QueryAccountRequest))
	})
}

func _Query_AccountByDID_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAccountByDIDRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AccountByDID(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Query/AccountByDID"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AccountByDID(ctx, req.(*QueryAccountByDIDRequest))
	})
}

func _Query_SessionKeys_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuerySessionKeysRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).SessionKeys(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Query/SessionKeys"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).SessionKeys(ctx, req.(*QuerySessionKeysRequest))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}
