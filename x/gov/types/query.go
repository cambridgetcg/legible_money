package types

import "context"

// QueryServer defines the governance module's query server interface.
type QueryServer interface {
	LIP(context.Context, *QueryLIPRequest) (*QueryLIPResponse, error)
	LIPs(context.Context, *QueryLIPsRequest) (*QueryLIPsResponse, error)
	Disbursement(context.Context, *QueryDisbursementRequest) (*QueryDisbursementResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// --- Query Request/Response Types ---

type QueryLIPRequest struct {
	LipId string `json:"lip_id"`
}

type QueryLIPResponse struct {
	Lip   LIP             `json:"lip"`
	Votes []QuadraticVote `json:"votes"`
}

type QueryLIPsRequest struct {
	Stage  string `json:"stage,omitempty"`
	Limit  uint32 `json:"limit"`
	Offset uint32 `json:"offset"`
}

type QueryLIPsResponse struct {
	Lips  []LIP  `json:"lips"`
	Total uint64 `json:"total"`
}

type QueryDisbursementRequest struct {
	DisbursementId string `json:"disbursement_id"`
}

type QueryDisbursementResponse struct {
	Disbursement DisbursementProposal `json:"disbursement"`
	Votes        []QuadraticVote      `json:"votes"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}

// --- proto.Message stubs for query types ---

func (q QueryLIPRequest) Reset()         {}
func (q QueryLIPRequest) String() string { return "QueryLIPRequest{}" }
func (q QueryLIPRequest) ProtoMessage()  {}

func (q QueryLIPResponse) Reset()         {}
func (q QueryLIPResponse) String() string { return "QueryLIPResponse{}" }
func (q QueryLIPResponse) ProtoMessage()  {}

func (q QueryLIPsRequest) Reset()         {}
func (q QueryLIPsRequest) String() string { return "QueryLIPsRequest{}" }
func (q QueryLIPsRequest) ProtoMessage()  {}

func (q QueryLIPsResponse) Reset()         {}
func (q QueryLIPsResponse) String() string { return "QueryLIPsResponse{}" }
func (q QueryLIPsResponse) ProtoMessage()  {}

func (q QueryDisbursementRequest) Reset()         {}
func (q QueryDisbursementRequest) String() string { return "QueryDisbursementRequest{}" }
func (q QueryDisbursementRequest) ProtoMessage()  {}

func (q QueryDisbursementResponse) Reset()         {}
func (q QueryDisbursementResponse) String() string { return "QueryDisbursementResponse{}" }
func (q QueryDisbursementResponse) ProtoMessage()  {}

func (q QueryParamsRequest) Reset()         {}
func (q QueryParamsRequest) String() string { return "QueryParamsRequest{}" }
func (q QueryParamsRequest) ProtoMessage()  {}

func (q QueryParamsResponse) Reset()         {}
func (q QueryParamsResponse) String() string { return "QueryParamsResponse{}" }
func (q QueryParamsResponse) ProtoMessage()  {}
