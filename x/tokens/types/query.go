package types

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// -----------------------------------------------------------------------
// Query request/response types
// -----------------------------------------------------------------------

// QueryAccountRequest is the request for the Account query.
type QueryAccountRequest struct {
	Address string `json:"address"`
}

func (r *QueryAccountRequest) Reset()        {}
func (r *QueryAccountRequest) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryAccountRequest) ProtoMessage()  {}

// QueryAccountResponse is the response for the Account query.
// ActiveBalance is the computed spendable balance (excludes expired-but-unswept periods).
// Account.TotalBalance is the raw stored total (includes expired-but-unswept periods
// until they are swept to the RAI pool).
type QueryAccountResponse struct {
	Account       ExpirableAccount `json:"account"`
	ActiveBalance string           `json:"active_balance"` // computed: sum of non-expired, non-swept periods
}

func (r *QueryAccountResponse) Reset()        {}
func (r *QueryAccountResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryAccountResponse) ProtoMessage()  {}

// QueryBalanceRequest is the request for the Balance query.
type QueryBalanceRequest struct {
	Address string `json:"address"`
}

func (r *QueryBalanceRequest) Reset()        {}
func (r *QueryBalanceRequest) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryBalanceRequest) ProtoMessage()  {}

// QueryBalanceResponse is the response for the Balance query.
type QueryBalanceResponse struct {
	ActiveBalance  string          `json:"active_balance"`
	PeriodBalances []PeriodBalance `json:"period_balances"`
}

func (r *QueryBalanceResponse) Reset()        {}
func (r *QueryBalanceResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryBalanceResponse) ProtoMessage()  {}

// QueryAllowanceRequest is the request for the Allowance query.
type QueryAllowanceRequest struct {
	Owner   string `json:"owner"`
	Spender string `json:"spender"`
}

func (r *QueryAllowanceRequest) Reset()        {}
func (r *QueryAllowanceRequest) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryAllowanceRequest) ProtoMessage()  {}

// QueryAllowanceResponse is the response for the Allowance query.
type QueryAllowanceResponse struct {
	Allowance string `json:"allowance"`
}

func (r *QueryAllowanceResponse) Reset()        {}
func (r *QueryAllowanceResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryAllowanceResponse) ProtoMessage()  {}

// QueryRAIRequest is the request for the RAI query.
type QueryRAIRequest struct{}

func (r *QueryRAIRequest) Reset()        {}
func (r *QueryRAIRequest) String() string { return "QueryRAIRequest" }
func (r *QueryRAIRequest) ProtoMessage()  {}

// QueryRAIResponse is the response for the RAI query.
type QueryRAIResponse struct {
	RAI RAIHoldingAccount `json:"rai"`
}

func (r *QueryRAIResponse) Reset()        {}
func (r *QueryRAIResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryRAIResponse) ProtoMessage()  {}

// QueryParamsRequest is the request for the Params query.
type QueryParamsRequest struct{}

func (r *QueryParamsRequest) Reset()        {}
func (r *QueryParamsRequest) String() string { return "QueryParamsRequest" }
func (r *QueryParamsRequest) ProtoMessage()  {}

// QueryParamsResponse is the response for the Params query.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (r *QueryParamsResponse) Reset()        {}
func (r *QueryParamsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryParamsResponse) ProtoMessage()  {}

// -----------------------------------------------------------------------
// QueryServer interface
// -----------------------------------------------------------------------

// QueryServer defines the tokens module query service.
type QueryServer interface {
	Account(context.Context, *QueryAccountRequest) (*QueryAccountResponse, error)
	Balance(context.Context, *QueryBalanceRequest) (*QueryBalanceResponse, error)
	Allowance(context.Context, *QueryAllowanceRequest) (*QueryAllowanceResponse, error)
	RAI(context.Context, *QueryRAIRequest) (*QueryRAIResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// -----------------------------------------------------------------------
// gRPC query service descriptor
// -----------------------------------------------------------------------

func _Query_Account_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Account(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tokens.v1.Query/Account"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Account(ctx, req.(*QueryAccountRequest))
	})
}

func _Query_Balance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryBalanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Balance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tokens.v1.Query/Balance"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Balance(ctx, req.(*QueryBalanceRequest))
	})
}

func _Query_Allowance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAllowanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Allowance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tokens.v1.Query/Allowance"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Allowance(ctx, req.(*QueryAllowanceRequest))
	})
}

func _Query_RAI_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryRAIRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).RAI(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tokens.v1.Query/RAI"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).RAI(ctx, req.(*QueryRAIRequest))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tokens.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.tokens.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Account", Handler: _Query_Account_Handler},
		{MethodName: "Balance", Handler: _Query_Balance_Handler},
		{MethodName: "Allowance", Handler: _Query_Allowance_Handler},
		{MethodName: "RAI", Handler: _Query_RAI_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/tokens/v1/query.proto",
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
