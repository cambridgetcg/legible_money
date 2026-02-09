package types

import "context"

// QueryServer defines the staking module query interface.
type QueryServer interface {
	Validator(context.Context, *QueryValidatorRequest) (*QueryValidatorResponse, error)
	Validators(context.Context, *QueryValidatorsRequest) (*QueryValidatorsResponse, error)
	Delegation(context.Context, *QueryDelegationRequest) (*QueryDelegationResponse, error)
	DelegationsForValidator(context.Context, *QueryDelegationsForValidatorRequest) (*QueryDelegationsForValidatorResponse, error)
	Unbondings(context.Context, *QueryUnbondingsRequest) (*QueryUnbondingsResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// ---------- Query Types ----------

type QueryValidatorRequest struct {
	OperatorAddress string `json:"operator_address"`
}

type QueryValidatorResponse struct {
	Validator LegibleValidator `json:"validator"`
}

type QueryValidatorsRequest struct {
	ActiveOnly bool   `json:"active_only,omitempty"`
	Tier       int32  `json:"tier,omitempty"` // -1 means all tiers
	Limit      uint32 `json:"limit,omitempty"`
	Offset     uint32 `json:"offset,omitempty"`
}

type QueryValidatorsResponse struct {
	Validators []LegibleValidator `json:"validators"`
	Total      uint32             `json:"total"`
}

type QueryDelegationRequest struct {
	DelegatorAddress string `json:"delegator_address"`
	ValidatorAddress string `json:"validator_address"`
}

type QueryDelegationResponse struct {
	Delegation Delegation `json:"delegation"`
}

type QueryDelegationsForValidatorRequest struct {
	ValidatorAddress string `json:"validator_address"`
}

type QueryDelegationsForValidatorResponse struct {
	Delegations []Delegation `json:"delegations"`
}

type QueryUnbondingsRequest struct {
	DelegatorAddress string `json:"delegator_address"`
}

type QueryUnbondingsResponse struct {
	Unbondings []UnbondingEntry `json:"unbondings"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params      Params       `json:"params"`
	TierConfigs []TierConfig `json:"tier_configs"`
}
