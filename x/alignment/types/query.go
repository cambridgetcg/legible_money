package types

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ============ Query Request/Response Types ============

// QueryAlignmentStateRequest is the request for the AlignmentState query.
type QueryAlignmentStateRequest struct{}

func (m *QueryAlignmentStateRequest) ProtoMessage()             {}
func (m *QueryAlignmentStateRequest) Reset()                     {}
func (m *QueryAlignmentStateRequest) String() string             { return "" }

// QueryAlignmentStateResponse is the response for the AlignmentState query.
type QueryAlignmentStateResponse struct {
	State  AlignmentState `json:"state"`
	Params Params         `json:"params"`
}

func (m *QueryAlignmentStateResponse) ProtoMessage()             {}
func (m *QueryAlignmentStateResponse) Reset()                     {}
func (m *QueryAlignmentStateResponse) String() string             { return "" }

// QueryDimensionScoresRequest is the request for the DimensionScores query.
type QueryDimensionScoresRequest struct {
	Epoch uint64 `json:"epoch"`
}

func (m *QueryDimensionScoresRequest) ProtoMessage()             {}
func (m *QueryDimensionScoresRequest) Reset()                     {}
func (m *QueryDimensionScoresRequest) String() string             { return "" }

// QueryDimensionScoresResponse is the response for the DimensionScores query.
type QueryDimensionScoresResponse struct {
	Scores      *DimensionScores      `json:"scores,omitempty"`
	Observation *AlignmentObservation `json:"observation,omitempty"`
	Corrections []CorrectionRecord    `json:"corrections,omitempty"`
}

func (m *QueryDimensionScoresResponse) ProtoMessage()             {}
func (m *QueryDimensionScoresResponse) Reset()                     {}
func (m *QueryDimensionScoresResponse) String() string             { return "" }

// QueryParamsRequest is the request for the Params query.
type QueryParamsRequest struct{}

func (m *QueryParamsRequest) ProtoMessage()             {}
func (m *QueryParamsRequest) Reset()                     {}
func (m *QueryParamsRequest) String() string             { return "" }

// QueryParamsResponse is the response for the Params query.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (m *QueryParamsResponse) ProtoMessage()             {}
func (m *QueryParamsResponse) Reset()                     {}
func (m *QueryParamsResponse) String() string             { return "" }

// ============ QueryServer Interface ============

// QueryServer defines the alignment module's query handler interface.
type QueryServer interface {
	AlignmentState(context.Context, *QueryAlignmentStateRequest) (*QueryAlignmentStateResponse, error)
	DimensionScores(context.Context, *QueryDimensionScoresRequest) (*QueryDimensionScoresResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer provides default unimplemented methods for forward compatibility.
type UnimplementedQueryServer struct{}

func (UnimplementedQueryServer) AlignmentState(context.Context, *QueryAlignmentStateRequest) (*QueryAlignmentStateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method AlignmentState not implemented")
}
func (UnimplementedQueryServer) DimensionScores(context.Context, *QueryDimensionScoresRequest) (*QueryDimensionScoresResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method DimensionScores not implemented")
}
func (UnimplementedQueryServer) Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Params not implemented")
}
func (UnimplementedQueryServer) mustEmbedUnimplementedQueryServer() {}
func (UnimplementedQueryServer) testEmbeddedByValue()               {}

// ============ gRPC Registration ============

const (
	Query_AlignmentState_FullMethodName  = "/legible.alignment.v1.Query/AlignmentState"
	Query_DimensionScores_FullMethodName = "/legible.alignment.v1.Query/DimensionScores"
	Query_Params_FullMethodName          = "/legible.alignment.v1.Query/Params"
)

func _Query_AlignmentState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAlignmentStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AlignmentState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_AlignmentState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AlignmentState(ctx, req.(*QueryAlignmentStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_DimensionScores_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDimensionScoresRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).DimensionScores(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_DimensionScores_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).DimensionScores(ctx, req.(*QueryDimensionScoresRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Params_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for the alignment Query service.
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "legible.alignment.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AlignmentState",
			Handler:    _Query_AlignmentState_Handler,
		},
		{
			MethodName: "DimensionScores",
			Handler:    _Query_DimensionScores_Handler,
		},
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/alignment/v1/query.proto",
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC service registrar.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Query_ServiceDesc, srv)
}
