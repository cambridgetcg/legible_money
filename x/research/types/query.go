package types

import (
	"context"
	"fmt"
)

// QueryServer defines the research module query interface.
type QueryServer interface {
	Research(context.Context, *QueryResearchRequest) (*QueryResearchResponse, error)
	Submissions(context.Context, *QuerySubmissionsRequest) (*QuerySubmissionsResponse, error)
	Bounty(context.Context, *QueryBountyRequest) (*QueryBountyResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// ---------- Query Types ----------

type QueryResearchRequest struct {
	ResearchId string `json:"research_id"`
}

func (m *QueryResearchRequest) Reset() {}
func (m *QueryResearchRequest) String() string {
	return fmt.Sprintf("QueryResearchRequest{%s}", m.ResearchId)
}
func (m *QueryResearchRequest) ProtoMessage() {}

type QueryResearchResponse struct {
	Research *Research     `json:"research"`
	Reviews  []*PeerReview `json:"reviews"`
}

func (m *QueryResearchResponse) Reset()         {}
func (m *QueryResearchResponse) String() string { return "QueryResearchResponse" }
func (m *QueryResearchResponse) ProtoMessage()  {}

type QuerySubmissionsRequest struct {
	Status string `json:"status,omitempty"`
	Domain string `json:"domain,omitempty"`
}

func (m *QuerySubmissionsRequest) Reset() {}
func (m *QuerySubmissionsRequest) String() string {
	return fmt.Sprintf("QuerySubmissionsRequest{status=%s,domain=%s}", m.Status, m.Domain)
}
func (m *QuerySubmissionsRequest) ProtoMessage() {}

type QuerySubmissionsResponse struct {
	Submissions []*Research `json:"submissions"`
}

func (m *QuerySubmissionsResponse) Reset()         {}
func (m *QuerySubmissionsResponse) String() string { return "QuerySubmissionsResponse" }
func (m *QuerySubmissionsResponse) ProtoMessage()  {}

type QueryBountyRequest struct {
	BountyId string `json:"bounty_id"`
}

func (m *QueryBountyRequest) Reset() {}
func (m *QueryBountyRequest) String() string {
	return fmt.Sprintf("QueryBountyRequest{%s}", m.BountyId)
}
func (m *QueryBountyRequest) ProtoMessage() {}

type QueryBountyResponse struct {
	Bounty *Bounty `json:"bounty"`
}

func (m *QueryBountyResponse) Reset()         {}
func (m *QueryBountyResponse) String() string { return "QueryBountyResponse" }
func (m *QueryBountyResponse) ProtoMessage()  {}

type QueryParamsRequest struct{}

func (m *QueryParamsRequest) Reset()         {}
func (m *QueryParamsRequest) String() string { return "QueryParamsRequest{}" }
func (m *QueryParamsRequest) ProtoMessage()  {}

type QueryParamsResponse struct {
	Params          *Params `json:"params"`
	TreasuryBalance string  `json:"treasury_balance"`
}

func (m *QueryParamsResponse) Reset()         {}
func (m *QueryParamsResponse) String() string { return "QueryParamsResponse" }
func (m *QueryParamsResponse) ProtoMessage()  {}
