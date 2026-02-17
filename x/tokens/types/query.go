package types

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// QueryParamsRequest is the request for the Params query.
type QueryParamsRequest struct{}

func (r *QueryParamsRequest) Reset()         {}
func (r *QueryParamsRequest) String() string { return "QueryParamsRequest" }
func (r *QueryParamsRequest) ProtoMessage()  {}

// QueryParamsResponse is the response for the Params query.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (r *QueryParamsResponse) Reset()         {}
func (r *QueryParamsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryParamsResponse) ProtoMessage()  {}

// QueryServer defines the tokens module query service.
type QueryServer interface {
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
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
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/tokens/v1/query.proto",
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
