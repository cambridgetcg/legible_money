package types

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// QueryServer defines the evidence_mgmt module query service.
type QueryServer interface {
	Evidence(context.Context, *QueryEvidenceRequest) (*QueryEvidenceResponse, error)
	EvidenceByClaimId(context.Context, *QueryEvidenceByClaimIdRequest) (*QueryEvidenceByClaimIdResponse, error)
	Audit(context.Context, *QueryAuditRequest) (*QueryAuditResponse, error)
	AuditsForEvidence(context.Context, *QueryAuditsForEvidenceRequest) (*QueryAuditsForEvidenceResponse, error)
	Oracle(context.Context, *QueryOracleRequest) (*QueryOracleResponse, error)
	ActiveOracles(context.Context, *QueryActiveOraclesRequest) (*QueryActiveOraclesResponse, error)
	Consensus(context.Context, *QueryConsensusRequest) (*QueryConsensusResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// ---------- Request/Response Types ----------

type QueryEvidenceRequest struct {
	EvidenceId string `json:"evidence_id"`
}

func (m *QueryEvidenceRequest) Reset()        {}
func (m *QueryEvidenceRequest) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryEvidenceRequest) ProtoMessage()  {}

type QueryEvidenceResponse struct {
	Evidence Evidence `json:"evidence"`
}

func (m *QueryEvidenceResponse) Reset()        {}
func (m *QueryEvidenceResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryEvidenceResponse) ProtoMessage()  {}

type QueryEvidenceByClaimIdRequest struct {
	ClaimId string `json:"claim_id"`
}

func (m *QueryEvidenceByClaimIdRequest) Reset()        {}
func (m *QueryEvidenceByClaimIdRequest) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryEvidenceByClaimIdRequest) ProtoMessage()  {}

type QueryEvidenceByClaimIdResponse struct {
	Evidences []Evidence `json:"evidences"`
}

func (m *QueryEvidenceByClaimIdResponse) Reset()        {}
func (m *QueryEvidenceByClaimIdResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryEvidenceByClaimIdResponse) ProtoMessage()  {}

type QueryAuditRequest struct {
	AuditId string `json:"audit_id"`
}

func (m *QueryAuditRequest) Reset()        {}
func (m *QueryAuditRequest) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryAuditRequest) ProtoMessage()  {}

type QueryAuditResponse struct {
	Audit EvidenceAudit `json:"audit"`
}

func (m *QueryAuditResponse) Reset()        {}
func (m *QueryAuditResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryAuditResponse) ProtoMessage()  {}

type QueryAuditsForEvidenceRequest struct {
	EvidenceId string `json:"evidence_id"`
}

func (m *QueryAuditsForEvidenceRequest) Reset()        {}
func (m *QueryAuditsForEvidenceRequest) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryAuditsForEvidenceRequest) ProtoMessage()  {}

type QueryAuditsForEvidenceResponse struct {
	Audits []EvidenceAudit `json:"audits"`
}

func (m *QueryAuditsForEvidenceResponse) Reset()        {}
func (m *QueryAuditsForEvidenceResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryAuditsForEvidenceResponse) ProtoMessage()  {}

type QueryOracleRequest struct {
	OracleId string `json:"oracle_id"`
}

func (m *QueryOracleRequest) Reset()        {}
func (m *QueryOracleRequest) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryOracleRequest) ProtoMessage()  {}

type QueryOracleResponse struct {
	Oracle Oracle `json:"oracle"`
}

func (m *QueryOracleResponse) Reset()        {}
func (m *QueryOracleResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryOracleResponse) ProtoMessage()  {}

type QueryActiveOraclesRequest struct{}

func (m *QueryActiveOraclesRequest) Reset()        {}
func (m *QueryActiveOraclesRequest) String() string { return "QueryActiveOraclesRequest{}" }
func (m *QueryActiveOraclesRequest) ProtoMessage()  {}

type QueryActiveOraclesResponse struct {
	Oracles []Oracle `json:"oracles"`
}

func (m *QueryActiveOraclesResponse) Reset()        {}
func (m *QueryActiveOraclesResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryActiveOraclesResponse) ProtoMessage()  {}

type QueryConsensusRequest struct {
	EvidenceId string `json:"evidence_id"`
}

func (m *QueryConsensusRequest) Reset()        {}
func (m *QueryConsensusRequest) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryConsensusRequest) ProtoMessage()  {}

type QueryConsensusResponse struct {
	Consensus AuditConsensus `json:"consensus"`
}

func (m *QueryConsensusResponse) Reset()        {}
func (m *QueryConsensusResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryConsensusResponse) ProtoMessage()  {}

type QueryParamsRequest struct{}

func (m *QueryParamsRequest) Reset()        {}
func (m *QueryParamsRequest) String() string { return "QueryParamsRequest{}" }
func (m *QueryParamsRequest) ProtoMessage()  {}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (m *QueryParamsResponse) Reset()        {}
func (m *QueryParamsResponse) String() string { return fmt.Sprintf("%+v", *m) }
func (m *QueryParamsResponse) ProtoMessage()  {}

// ---------- gRPC Query Service Descriptor ----------

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.evidence_mgmt.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Evidence", Handler: _Query_Evidence_Handler},
		{MethodName: "EvidenceByClaimId", Handler: _Query_EvidenceByClaimId_Handler},
		{MethodName: "Audit", Handler: _Query_Audit_Handler},
		{MethodName: "AuditsForEvidence", Handler: _Query_AuditsForEvidence_Handler},
		{MethodName: "Oracle", Handler: _Query_Oracle_Handler},
		{MethodName: "ActiveOracles", Handler: _Query_ActiveOracles_Handler},
		{MethodName: "Consensus", Handler: _Query_Consensus_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/evidence_mgmt/v1/query.proto",
}

func _Query_Evidence_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryEvidenceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Evidence(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.evidence_mgmt.v1.Query/Evidence"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Evidence(ctx, req.(*QueryEvidenceRequest))
	})
}

func _Query_EvidenceByClaimId_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryEvidenceByClaimIdRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).EvidenceByClaimId(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.evidence_mgmt.v1.Query/EvidenceByClaimId"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).EvidenceByClaimId(ctx, req.(*QueryEvidenceByClaimIdRequest))
	})
}

func _Query_Audit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAuditRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Audit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.evidence_mgmt.v1.Query/Audit"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Audit(ctx, req.(*QueryAuditRequest))
	})
}

func _Query_AuditsForEvidence_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAuditsForEvidenceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).AuditsForEvidence(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.evidence_mgmt.v1.Query/AuditsForEvidence"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).AuditsForEvidence(ctx, req.(*QueryAuditsForEvidenceRequest))
	})
}

func _Query_Oracle_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryOracleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Oracle(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.evidence_mgmt.v1.Query/Oracle"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Oracle(ctx, req.(*QueryOracleRequest))
	})
}

func _Query_ActiveOracles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryActiveOraclesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ActiveOracles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.evidence_mgmt.v1.Query/ActiveOracles"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ActiveOracles(ctx, req.(*QueryActiveOraclesRequest))
	})
}

func _Query_Consensus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryConsensusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Consensus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.evidence_mgmt.v1.Query/Consensus"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Consensus(ctx, req.(*QueryConsensusRequest))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.evidence_mgmt.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
