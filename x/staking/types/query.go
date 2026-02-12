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

func (m *QueryValidatorRequest) Reset()         { *m = QueryValidatorRequest{} }
func (m *QueryValidatorRequest) String() string { return "QueryValidatorRequest" }
func (m *QueryValidatorRequest) ProtoMessage()  {}

type QueryValidatorResponse struct {
	Validator LegibleValidator `json:"validator"`
}

func (m *QueryValidatorResponse) Reset()         { *m = QueryValidatorResponse{} }
func (m *QueryValidatorResponse) String() string { return "QueryValidatorResponse" }
func (m *QueryValidatorResponse) ProtoMessage()  {}

type QueryValidatorsRequest struct {
	ActiveOnly bool   `json:"active_only,omitempty"`
	Tier       int32  `json:"tier,omitempty"` // -1 means all tiers
	Limit      uint32 `json:"limit,omitempty"`
	Offset     uint32 `json:"offset,omitempty"`
}

func (m *QueryValidatorsRequest) Reset()         { *m = QueryValidatorsRequest{} }
func (m *QueryValidatorsRequest) String() string { return "QueryValidatorsRequest" }
func (m *QueryValidatorsRequest) ProtoMessage()  {}

type QueryValidatorsResponse struct {
	Validators []LegibleValidator `json:"validators"`
	Total      uint32             `json:"total"`
}

func (m *QueryValidatorsResponse) Reset()         { *m = QueryValidatorsResponse{} }
func (m *QueryValidatorsResponse) String() string { return "QueryValidatorsResponse" }
func (m *QueryValidatorsResponse) ProtoMessage()  {}

type QueryDelegationRequest struct {
	DelegatorAddress string `json:"delegator_address"`
	ValidatorAddress string `json:"validator_address"`
}

func (m *QueryDelegationRequest) Reset()         { *m = QueryDelegationRequest{} }
func (m *QueryDelegationRequest) String() string { return "QueryDelegationRequest" }
func (m *QueryDelegationRequest) ProtoMessage()  {}

type QueryDelegationResponse struct {
	Delegation Delegation `json:"delegation"`
}

func (m *QueryDelegationResponse) Reset()         { *m = QueryDelegationResponse{} }
func (m *QueryDelegationResponse) String() string { return "QueryDelegationResponse" }
func (m *QueryDelegationResponse) ProtoMessage()  {}

type QueryDelegationsForValidatorRequest struct {
	ValidatorAddress string `json:"validator_address"`
}

func (m *QueryDelegationsForValidatorRequest) Reset()         { *m = QueryDelegationsForValidatorRequest{} }
func (m *QueryDelegationsForValidatorRequest) String() string { return "QueryDelegationsForValidatorRequest" }
func (m *QueryDelegationsForValidatorRequest) ProtoMessage()  {}

type QueryDelegationsForValidatorResponse struct {
	Delegations []Delegation `json:"delegations"`
}

func (m *QueryDelegationsForValidatorResponse) Reset()         { *m = QueryDelegationsForValidatorResponse{} }
func (m *QueryDelegationsForValidatorResponse) String() string { return "QueryDelegationsForValidatorResponse" }
func (m *QueryDelegationsForValidatorResponse) ProtoMessage()  {}

type QueryUnbondingsRequest struct {
	DelegatorAddress string `json:"delegator_address"`
}

func (m *QueryUnbondingsRequest) Reset()         { *m = QueryUnbondingsRequest{} }
func (m *QueryUnbondingsRequest) String() string { return "QueryUnbondingsRequest" }
func (m *QueryUnbondingsRequest) ProtoMessage()  {}

type QueryUnbondingsResponse struct {
	Unbondings []UnbondingEntry `json:"unbondings"`
}

func (m *QueryUnbondingsResponse) Reset()         { *m = QueryUnbondingsResponse{} }
func (m *QueryUnbondingsResponse) String() string { return "QueryUnbondingsResponse" }
func (m *QueryUnbondingsResponse) ProtoMessage()  {}

type QueryParamsRequest struct{}

func (m *QueryParamsRequest) Reset()         { *m = QueryParamsRequest{} }
func (m *QueryParamsRequest) String() string { return "QueryParamsRequest" }
func (m *QueryParamsRequest) ProtoMessage()  {}

type QueryParamsResponse struct {
	Params      Params       `json:"params"`
	TierConfigs []TierConfig `json:"tier_configs"`
}

func (m *QueryParamsResponse) Reset()         { *m = QueryParamsResponse{} }
func (m *QueryParamsResponse) String() string { return "QueryParamsResponse" }
func (m *QueryParamsResponse) ProtoMessage()  {}
