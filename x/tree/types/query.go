package types

import (
	"context"

	"google.golang.org/grpc"
)

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

// ---------- gRPC Query Service Descriptor ----------

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.tree.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Project", Handler: _Query_Project_Handler},
		{MethodName: "ProjectsByFounder", Handler: _Query_ProjectsByFounder_Handler},
		{MethodName: "Task", Handler: _Query_Task_Handler},
		{MethodName: "TasksByProject", Handler: _Query_TasksByProject_Handler},
		{MethodName: "Service", Handler: _Query_Service_Handler},
		{MethodName: "Seed", Handler: _Query_Seed_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/tree/v1/query.proto",
}

func _Query_Project_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryProjectRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Project(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Query/Project"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Project(ctx, req.(*QueryProjectRequest))
	})
}

func _Query_ProjectsByFounder_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryProjectsByFounderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ProjectsByFounder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Query/ProjectsByFounder"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ProjectsByFounder(ctx, req.(*QueryProjectsByFounderRequest))
	})
}

func _Query_Task_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTaskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Task(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Query/Task"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Task(ctx, req.(*QueryTaskRequest))
	})
}

func _Query_TasksByProject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTasksByProjectRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).TasksByProject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Query/TasksByProject"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).TasksByProject(ctx, req.(*QueryTasksByProjectRequest))
	})
}

func _Query_Service_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryServiceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Service(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Query/Service"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Service(ctx, req.(*QueryServiceRequest))
	})
}

func _Query_Seed_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuerySeedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Seed(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Query/Seed"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Seed(ctx, req.(*QuerySeedRequest))
	})
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
