package types

import "context"

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

type QueryResearchResponse struct {
	Research Research     `json:"research"`
	Reviews  []PeerReview `json:"reviews"`
}

type QuerySubmissionsRequest struct {
	Status string `json:"status,omitempty"`
	Domain string `json:"domain,omitempty"`
}

type QuerySubmissionsResponse struct {
	Submissions []Research `json:"submissions"`
}

type QueryBountyRequest struct {
	BountyId string `json:"bounty_id"`
}

type QueryBountyResponse struct {
	Bounty Bounty `json:"bounty"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params          Params `json:"params"`
	TreasuryBalance string `json:"treasury_balance"`
}
