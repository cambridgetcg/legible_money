package types

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// QueryServer is the query service interface for the knowledge module.
type QueryServer interface {
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	Fact(context.Context, *QueryFactRequest) (*QueryFactResponse, error)
	Facts(context.Context, *QueryFactsRequest) (*QueryFactsResponse, error)
	FactsByDomain(context.Context, *QueryFactsByDomainRequest) (*QueryFactsByDomainResponse, error)
	Claim(context.Context, *QueryClaimRequest) (*QueryClaimResponse, error)
	VerificationRound(context.Context, *QueryVerificationRoundRequest) (*QueryVerificationRoundResponse, error)
	ActiveRounds(context.Context, *QueryActiveRoundsRequest) (*QueryActiveRoundsResponse, error)
	Domain(context.Context, *QueryDomainRequest) (*QueryDomainResponse, error)
}

// ---- Request/Response Types ----

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

type QueryFactRequest struct {
	FactId string `json:"fact_id"`
}

func (r *QueryFactRequest) Reset()        {}
func (r *QueryFactRequest) String() string { return fmt.Sprintf("QueryFactRequest{%s}", r.FactId) }
func (r *QueryFactRequest) ProtoMessage()  {}

type QueryFactResponse struct {
	Fact Fact `json:"fact"`
}

func (r *QueryFactResponse) Reset()        {}
func (r *QueryFactResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryFactResponse) ProtoMessage()  {}

type QueryFactsRequest struct {
	Limit  uint32 `json:"limit"`
	Offset uint32 `json:"offset"`
}

func (r *QueryFactsRequest) Reset()        {}
func (r *QueryFactsRequest) String() string { return "QueryFactsRequest" }
func (r *QueryFactsRequest) ProtoMessage()  {}

type QueryFactsResponse struct {
	Facts []Fact `json:"facts"`
	Total uint64 `json:"total"`
}

func (r *QueryFactsResponse) Reset()        {}
func (r *QueryFactsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryFactsResponse) ProtoMessage()  {}

type QueryFactsByDomainRequest struct {
	Domain string `json:"domain"`
}

func (r *QueryFactsByDomainRequest) Reset()        {}
func (r *QueryFactsByDomainRequest) String() string { return fmt.Sprintf("QueryFactsByDomain{%s}", r.Domain) }
func (r *QueryFactsByDomainRequest) ProtoMessage()  {}

type QueryFactsByDomainResponse struct {
	Facts []Fact `json:"facts"`
}

func (r *QueryFactsByDomainResponse) Reset()        {}
func (r *QueryFactsByDomainResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryFactsByDomainResponse) ProtoMessage()  {}

type QueryClaimRequest struct {
	ClaimId string `json:"claim_id"`
}

func (r *QueryClaimRequest) Reset()        {}
func (r *QueryClaimRequest) String() string { return fmt.Sprintf("QueryClaimRequest{%s}", r.ClaimId) }
func (r *QueryClaimRequest) ProtoMessage()  {}

type QueryClaimResponse struct {
	Claim Claim `json:"claim"`
}

func (r *QueryClaimResponse) Reset()        {}
func (r *QueryClaimResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryClaimResponse) ProtoMessage()  {}

type QueryVerificationRoundRequest struct {
	RoundId string `json:"round_id"`
}

func (r *QueryVerificationRoundRequest) Reset()        {}
func (r *QueryVerificationRoundRequest) String() string { return fmt.Sprintf("QueryRound{%s}", r.RoundId) }
func (r *QueryVerificationRoundRequest) ProtoMessage()  {}

type QueryVerificationRoundResponse struct {
	Round VerificationRound `json:"round"`
}

func (r *QueryVerificationRoundResponse) Reset()        {}
func (r *QueryVerificationRoundResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryVerificationRoundResponse) ProtoMessage()  {}

type QueryActiveRoundsRequest struct{}

func (r *QueryActiveRoundsRequest) Reset()        {}
func (r *QueryActiveRoundsRequest) String() string { return "QueryActiveRoundsRequest" }
func (r *QueryActiveRoundsRequest) ProtoMessage()  {}

type QueryActiveRoundsResponse struct {
	Rounds []VerificationRound `json:"rounds"`
}

func (r *QueryActiveRoundsResponse) Reset()        {}
func (r *QueryActiveRoundsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryActiveRoundsResponse) ProtoMessage()  {}

type QueryDomainRequest struct {
	Name string `json:"name"`
}

func (r *QueryDomainRequest) Reset()        {}
func (r *QueryDomainRequest) String() string { return fmt.Sprintf("QueryDomain{%s}", r.Name) }
func (r *QueryDomainRequest) ProtoMessage()  {}

type QueryDomainResponse struct {
	Domain OntologicalDomain `json:"domain"`
}

func (r *QueryDomainResponse) Reset()        {}
func (r *QueryDomainResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryDomainResponse) ProtoMessage()  {}

// ---- Service Descriptor ----

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.knowledge.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Params", Handler: nil},
		{MethodName: "Fact", Handler: nil},
		{MethodName: "Facts", Handler: nil},
		{MethodName: "FactsByDomain", Handler: nil},
		{MethodName: "Claim", Handler: nil},
		{MethodName: "VerificationRound", Handler: nil},
		{MethodName: "ActiveRounds", Handler: nil},
		{MethodName: "Domain", Handler: nil},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/knowledge/v1/query.proto",
}

// RegisterQueryServer registers the QueryServer implementation.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
