package types

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// -----------------------------------------------------------------------
// Query request/response types
// -----------------------------------------------------------------------

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

// QueryProviderRequest is the request for the Provider query.
type QueryProviderRequest struct {
	Address string `json:"address"`
}

func (r *QueryProviderRequest) Reset()        {}
func (r *QueryProviderRequest) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryProviderRequest) ProtoMessage()  {}

// QueryProviderResponse is the response for the Provider query.
type QueryProviderResponse struct {
	Provider Provider `json:"provider"`
}

func (r *QueryProviderResponse) Reset()        {}
func (r *QueryProviderResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryProviderResponse) ProtoMessage()  {}

// QueryQuoteRequest is the request for the Quote query.
type QueryQuoteRequest struct {
	QuoteId uint64 `json:"quote_id"`
}

func (r *QueryQuoteRequest) Reset()        {}
func (r *QueryQuoteRequest) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryQuoteRequest) ProtoMessage()  {}

// QueryQuoteResponse is the response for the Quote query.
type QueryQuoteResponse struct {
	Quote APIQuote `json:"quote"`
}

func (r *QueryQuoteResponse) Reset()        {}
func (r *QueryQuoteResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryQuoteResponse) ProtoMessage()  {}

// -----------------------------------------------------------------------
// QueryServer interface
// -----------------------------------------------------------------------

// QueryServer defines the billing module query service.
type QueryServer interface {
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	Provider(context.Context, *QueryProviderRequest) (*QueryProviderResponse, error)
	Quote(context.Context, *QueryQuoteRequest) (*QueryQuoteResponse, error)
}

// -----------------------------------------------------------------------
// gRPC query service descriptor
// -----------------------------------------------------------------------

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.billing.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

func _Query_Provider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Provider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.billing.v1.Query/Provider"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Provider(ctx, req.(*QueryProviderRequest))
	})
}

func _Query_Quote_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryQuoteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Quote(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.billing.v1.Query/Quote"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Quote(ctx, req.(*QueryQuoteRequest))
	})
}

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.billing.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Params", Handler: _Query_Params_Handler},
		{MethodName: "Provider", Handler: _Query_Provider_Handler},
		{MethodName: "Quote", Handler: _Query_Quote_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/billing/v1/query.proto",
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
