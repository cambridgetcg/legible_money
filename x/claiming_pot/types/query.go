package types

import "context"

// QueryServer defines the claiming_pot module query interface.
type QueryServer interface {
	Pot(context.Context, *QueryPotRequest) (*QueryPotResponse, error)
	Pots(context.Context, *QueryPotsRequest) (*QueryPotsResponse, error)
	PotsByCreator(context.Context, *QueryPotsByCreatorRequest) (*QueryPotsByCreatorResponse, error)
	Eligibility(context.Context, *QueryEligibilityRequest) (*QueryEligibilityResponse, error)
	ClaimRecords(context.Context, *QueryClaimRecordsRequest) (*QueryClaimRecordsResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// ---------- Query Types ----------

type QueryPotRequest struct {
	PotId string `json:"pot_id"`
}

type QueryPotResponse struct {
	Pot *ClaimingPot `json:"pot"`
}

type QueryPotsRequest struct {
	Status string `json:"status,omitempty"`
	Limit  uint32 `json:"limit,omitempty"`
	Offset uint32 `json:"offset,omitempty"`
}

type QueryPotsResponse struct {
	Pots  []ClaimingPot `json:"pots"`
	Total uint64        `json:"total"`
}

type QueryPotsByCreatorRequest struct {
	Creator string `json:"creator"`
	Limit   uint32 `json:"limit,omitempty"`
	Offset  uint32 `json:"offset,omitempty"`
}

type QueryPotsByCreatorResponse struct {
	Pots  []ClaimingPot `json:"pots"`
	Total uint64        `json:"total"`
}

type QueryEligibilityRequest struct {
	PotId   string `json:"pot_id"`
	Claimer string `json:"claimer"`
}

type QueryEligibilityResponse struct {
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason"`
}

type QueryClaimRecordsRequest struct {
	PotId  string `json:"pot_id"`
	Limit  uint32 `json:"limit,omitempty"`
	Offset uint32 `json:"offset,omitempty"`
}

type QueryClaimRecordsResponse struct {
	Records []ClaimRecord `json:"records"`
	Total   uint64        `json:"total"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}
