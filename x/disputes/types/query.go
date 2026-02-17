package types

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// QueryServer defines the disputes module query service.
type QueryServer interface {
	Dispute(context.Context, *QueryDisputeRequest) (*QueryDisputeResponse, error)
	DisputesByFact(context.Context, *QueryDisputesByFactRequest) (*QueryDisputesByFactResponse, error)
	ActiveDisputes(context.Context, *QueryActiveDisputesRequest) (*QueryActiveDisputesResponse, error)
	Evidence(context.Context, *QueryEvidenceRequest) (*QueryEvidenceResponse, error)
	Votes(context.Context, *QueryVotesRequest) (*QueryVotesResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// ---------- Request/Response Types ----------

type QueryDisputeRequest struct {
	DisputeId string `json:"dispute_id"`
}

func (m *QueryDisputeRequest) Reset()         {}
func (m *QueryDisputeRequest) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryDisputeRequest) ProtoMessage()  {}

type QueryDisputeResponse struct {
	Dispute Dispute `json:"dispute"`
}

func (m *QueryDisputeResponse) Reset()         {}
func (m *QueryDisputeResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryDisputeResponse) ProtoMessage()  {}

type QueryDisputesByFactRequest struct {
	FactId string `json:"fact_id"`
}

func (m *QueryDisputesByFactRequest) Reset()         {}
func (m *QueryDisputesByFactRequest) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryDisputesByFactRequest) ProtoMessage()  {}

type QueryDisputesByFactResponse struct {
	Disputes []Dispute `json:"disputes"`
}

func (m *QueryDisputesByFactResponse) Reset()         {}
func (m *QueryDisputesByFactResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryDisputesByFactResponse) ProtoMessage()  {}

type QueryActiveDisputesRequest struct{}

func (m *QueryActiveDisputesRequest) Reset()         {}
func (m *QueryActiveDisputesRequest) String() string { return "QueryActiveDisputesRequest{}" }
func (m *QueryActiveDisputesRequest) ProtoMessage()  {}

type QueryActiveDisputesResponse struct {
	Disputes []Dispute `json:"disputes"`
}

func (m *QueryActiveDisputesResponse) Reset()         {}
func (m *QueryActiveDisputesResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryActiveDisputesResponse) ProtoMessage()  {}

type QueryEvidenceRequest struct {
	DisputeId string `json:"dispute_id"`
}

func (m *QueryEvidenceRequest) Reset()         {}
func (m *QueryEvidenceRequest) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryEvidenceRequest) ProtoMessage()  {}

type QueryEvidenceResponse struct {
	Evidence []DisputeEvidence `json:"evidence"`
}

func (m *QueryEvidenceResponse) Reset()         {}
func (m *QueryEvidenceResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryEvidenceResponse) ProtoMessage()  {}

type QueryVotesRequest struct {
	DisputeId string `json:"dispute_id"`
}

func (m *QueryVotesRequest) Reset()         {}
func (m *QueryVotesRequest) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryVotesRequest) ProtoMessage()  {}

type QueryVotesResponse struct {
	Votes []DisputeVote `json:"votes"`
}

func (m *QueryVotesResponse) Reset()         {}
func (m *QueryVotesResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryVotesResponse) ProtoMessage()  {}

type QueryParamsRequest struct{}

func (m *QueryParamsRequest) Reset()         {}
func (m *QueryParamsRequest) String() string { return "QueryParamsRequest{}" }
func (m *QueryParamsRequest) ProtoMessage()  {}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (m *QueryParamsResponse) Reset()         {}
func (m *QueryParamsResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryParamsResponse) ProtoMessage()  {}

// ---------- gRPC Query Service Descriptor ----------

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.disputes.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Dispute", Handler: _Query_Dispute_Handler},
		{MethodName: "DisputesByFact", Handler: _Query_DisputesByFact_Handler},
		{MethodName: "ActiveDisputes", Handler: _Query_ActiveDisputes_Handler},
		{MethodName: "Evidence", Handler: _Query_Evidence_Handler},
		{MethodName: "Votes", Handler: _Query_Votes_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/disputes/v1/query.proto",
}

func _Query_Dispute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDisputeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Dispute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Query/Dispute"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Dispute(ctx, req.(*QueryDisputeRequest))
	})
}

func _Query_DisputesByFact_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDisputesByFactRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).DisputesByFact(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Query/DisputesByFact"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).DisputesByFact(ctx, req.(*QueryDisputesByFactRequest))
	})
}

func _Query_ActiveDisputes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryActiveDisputesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ActiveDisputes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Query/ActiveDisputes"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ActiveDisputes(ctx, req.(*QueryActiveDisputesRequest))
	})
}

func _Query_Evidence_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryEvidenceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Evidence(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Query/Evidence"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Evidence(ctx, req.(*QueryEvidenceRequest))
	})
}

func _Query_Votes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryVotesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Votes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Query/Votes"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Votes(ctx, req.(*QueryVotesRequest))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
