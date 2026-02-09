package types

import "context"

// --- Query Request/Response Types ---

type QueryContractRequest struct {
	Address string `json:"address"`
}

type QueryContractResponse struct {
	Contract *DeployedContract `json:"contract"`
}

type QueryContractsByCreatorRequest struct {
	Creator string `json:"creator"`
	Offset  uint32 `json:"offset,omitempty"`
	Limit   uint32 `json:"limit,omitempty"`
}

type QueryContractsByCreatorResponse struct {
	Contracts []DeployedContract `json:"contracts"`
	Total     uint64             `json:"total"`
}

type QueryCodeRequest struct {
	CodeHash string `json:"code_hash"`
}

type QueryCodeResponse struct {
	Code *ContractCode `json:"code"`
}

type QueryContractStateRequest struct {
	ContractAddress string `json:"contract_address"`
	Key             string `json:"key"`
}

type QueryContractStateResponse struct {
	Value string `json:"value"`
	Found bool   `json:"found"`
}

type QueryScheduleRequest struct {
	ScheduleId string `json:"schedule_id"`
}

type QueryScheduleResponse struct {
	Schedule *ContractSchedule `json:"schedule"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}

// --- QueryServer Interface ---

// QueryServer defines the bvm module's query server interface.
type QueryServer interface {
	Contract(context.Context, *QueryContractRequest) (*QueryContractResponse, error)
	ContractsByCreator(context.Context, *QueryContractsByCreatorRequest) (*QueryContractsByCreatorResponse, error)
	Code(context.Context, *QueryCodeRequest) (*QueryCodeResponse, error)
	ContractState(context.Context, *QueryContractStateRequest) (*QueryContractStateResponse, error)
	Schedule(context.Context, *QueryScheduleRequest) (*QueryScheduleResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}
