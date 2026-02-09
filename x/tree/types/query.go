package types

import "context"

// QueryServer defines the tree module query interface.
type QueryServer interface {
	Project(context.Context, *QueryProjectRequest) (*QueryProjectResponse, error)
	ProjectsByFounder(context.Context, *QueryProjectsByFounderRequest) (*QueryProjectsByFounderResponse, error)
	Task(context.Context, *QueryTaskRequest) (*QueryTaskResponse, error)
	TasksByProject(context.Context, *QueryTasksByProjectRequest) (*QueryTasksByProjectResponse, error)
	Service(context.Context, *QueryServiceRequest) (*QueryServiceResponse, error)
	Seed(context.Context, *QuerySeedRequest) (*QuerySeedResponse, error)
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
}

// --- Query Request/Response Types ---

type QueryProjectRequest struct {
	ProjectId string `json:"project_id"`
}

type QueryProjectResponse struct {
	Project *ProductProject `json:"project"`
}

func (m *QueryProjectResponse) Reset()         { *m = QueryProjectResponse{} }
func (m *QueryProjectResponse) String() string { return "QueryProjectResponse" }
func (m *QueryProjectResponse) ProtoMessage()  {}

type QueryProjectsByFounderRequest struct {
	Founder string `json:"founder"`
	Limit   uint32 `json:"limit,omitempty"`
	Offset  uint32 `json:"offset,omitempty"`
}

type QueryProjectsByFounderResponse struct {
	Projects []ProductProject `json:"projects"`
	Total    uint64           `json:"total"`
}

func (m *QueryProjectsByFounderResponse) Reset()         { *m = QueryProjectsByFounderResponse{} }
func (m *QueryProjectsByFounderResponse) String() string { return "QueryProjectsByFounderResponse" }
func (m *QueryProjectsByFounderResponse) ProtoMessage()  {}

type QueryTaskRequest struct {
	TaskId string `json:"task_id"`
}

type QueryTaskResponse struct {
	Task *ProjectTask `json:"task"`
}

func (m *QueryTaskResponse) Reset()         { *m = QueryTaskResponse{} }
func (m *QueryTaskResponse) String() string { return "QueryTaskResponse" }
func (m *QueryTaskResponse) ProtoMessage()  {}

type QueryTasksByProjectRequest struct {
	ProjectId string `json:"project_id"`
	Limit     uint32 `json:"limit,omitempty"`
	Offset    uint32 `json:"offset,omitempty"`
}

type QueryTasksByProjectResponse struct {
	Tasks []ProjectTask `json:"tasks"`
	Total uint64        `json:"total"`
}

func (m *QueryTasksByProjectResponse) Reset()         { *m = QueryTasksByProjectResponse{} }
func (m *QueryTasksByProjectResponse) String() string { return "QueryTasksByProjectResponse" }
func (m *QueryTasksByProjectResponse) ProtoMessage()  {}

type QueryServiceRequest struct {
	ServiceId string `json:"service_id"`
}

type QueryServiceResponse struct {
	Service *ServiceLeaf `json:"service"`
}

func (m *QueryServiceResponse) Reset()         { *m = QueryServiceResponse{} }
func (m *QueryServiceResponse) String() string { return "QueryServiceResponse" }
func (m *QueryServiceResponse) ProtoMessage()  {}

type QuerySeedRequest struct {
	SeedId string `json:"seed_id"`
}

type QuerySeedResponse struct {
	Seed *OpportunitySeed `json:"seed"`
}

func (m *QuerySeedResponse) Reset()         { *m = QuerySeedResponse{} }
func (m *QuerySeedResponse) String() string { return "QuerySeedResponse" }
func (m *QuerySeedResponse) ProtoMessage()  {}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (m *QueryParamsResponse) Reset()         { *m = QueryParamsResponse{} }
func (m *QueryParamsResponse) String() string { return "QueryParamsResponse" }
func (m *QueryParamsResponse) ProtoMessage()  {}
