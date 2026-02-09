package types

import "context"

// QueryServer defines the home module query interface.
type QueryServer interface {
	Home(context.Context, *QueryHomeRequest) (*QueryHomeResponse, error)
	Homes(context.Context, *QueryHomesRequest) (*QueryHomesResponse, error)
	Sessions(context.Context, *QuerySessionsRequest) (*QuerySessionsResponse, error)
	Alerts(context.Context, *QueryAlertsRequest) (*QueryAlertsResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// ---------- Query Types ----------

type QueryHomeRequest struct {
	DID string `json:"did"`
}

type QueryHomeResponse struct {
	Home  *AgentHome `json:"home"`
	Found bool       `json:"found"`
}

type QueryHomesRequest struct {
	Owner  string `json:"owner,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  uint32 `json:"limit,omitempty"`
	Offset uint32 `json:"offset,omitempty"`
}

type QueryHomesResponse struct {
	Homes []AgentHome `json:"homes"`
	Total uint64      `json:"total"`
}

type QuerySessionsRequest struct {
	HomeId string `json:"home_id"`
}

type QuerySessionsResponse struct {
	Sessions []ActiveSession `json:"sessions"`
}

type QueryAlertsRequest struct {
	HomeId       string `json:"home_id"`
	Acknowledged *bool  `json:"acknowledged,omitempty"`
}

type QueryAlertsResponse struct {
	Alerts []Alert `json:"alerts"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}
