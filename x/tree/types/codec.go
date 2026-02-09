package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"google.golang.org/grpc"
)

// RegisterCodec registers the tree module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	// Projects
	cdc.RegisterConcrete(&MsgCreateProject{}, "tree/CreateProject", nil)
	cdc.RegisterConcrete(&MsgProposeProject{}, "tree/ProposeProject", nil)
	cdc.RegisterConcrete(&MsgStartDevelopment{}, "tree/StartDevelopment", nil)
	cdc.RegisterConcrete(&MsgCompleteProject{}, "tree/CompleteProject", nil)
	cdc.RegisterConcrete(&MsgPauseProject{}, "tree/PauseProject", nil)
	cdc.RegisterConcrete(&MsgResumeProject{}, "tree/ResumeProject", nil)
	cdc.RegisterConcrete(&MsgAbandonProject{}, "tree/AbandonProject", nil)
	cdc.RegisterConcrete(&MsgSpawnChildProject{}, "tree/SpawnChildProject", nil)
	// Tasks
	cdc.RegisterConcrete(&MsgAddTask{}, "tree/AddTask", nil)
	cdc.RegisterConcrete(&MsgAssignTask{}, "tree/AssignTask", nil)
	cdc.RegisterConcrete(&MsgStartWork{}, "tree/StartWork", nil)
	cdc.RegisterConcrete(&MsgSubmitDeliverable{}, "tree/SubmitDeliverable", nil)
	cdc.RegisterConcrete(&MsgApproveDeliverable{}, "tree/ApproveDeliverable", nil)
	cdc.RegisterConcrete(&MsgRejectDeliverable{}, "tree/RejectDeliverable", nil)
	cdc.RegisterConcrete(&MsgReopenTask{}, "tree/ReopenTask", nil)
	// Applications
	cdc.RegisterConcrete(&MsgApplyToProject{}, "tree/ApplyToProject", nil)
	cdc.RegisterConcrete(&MsgReviewApplication{}, "tree/ReviewApplication", nil)
	cdc.RegisterConcrete(&MsgSetAvailability{}, "tree/SetAvailability", nil)
	// Services
	cdc.RegisterConcrete(&MsgDeployService{}, "tree/DeployService", nil)
	cdc.RegisterConcrete(&MsgPauseService{}, "tree/PauseService", nil)
	cdc.RegisterConcrete(&MsgResumeService{}, "tree/ResumeService", nil)
	cdc.RegisterConcrete(&MsgRetireService{}, "tree/RetireService", nil)
}

// RegisterInterfaces registers the tree module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// ======================== Msg Service Descriptor ========================

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.tree.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		// Projects
		{MethodName: "CreateProject", Handler: _Msg_CreateProject_Handler},
		{MethodName: "ProposeProject", Handler: _Msg_ProposeProject_Handler},
		{MethodName: "StartDevelopment", Handler: _Msg_StartDevelopment_Handler},
		{MethodName: "CompleteProject", Handler: _Msg_CompleteProject_Handler},
		{MethodName: "PauseProject", Handler: _Msg_PauseProject_Handler},
		{MethodName: "ResumeProject", Handler: _Msg_ResumeProject_Handler},
		{MethodName: "AbandonProject", Handler: _Msg_AbandonProject_Handler},
		{MethodName: "SpawnChildProject", Handler: _Msg_SpawnChildProject_Handler},
		// Tasks
		{MethodName: "AddTask", Handler: _Msg_AddTask_Handler},
		{MethodName: "AssignTask", Handler: _Msg_AssignTask_Handler},
		{MethodName: "StartWork", Handler: _Msg_StartWork_Handler},
		{MethodName: "SubmitDeliverable", Handler: _Msg_SubmitDeliverable_Handler},
		{MethodName: "ApproveDeliverable", Handler: _Msg_ApproveDeliverable_Handler},
		{MethodName: "RejectDeliverable", Handler: _Msg_RejectDeliverable_Handler},
		{MethodName: "ReopenTask", Handler: _Msg_ReopenTask_Handler},
		// Applications
		{MethodName: "ApplyToProject", Handler: _Msg_ApplyToProject_Handler},
		{MethodName: "ReviewApplication", Handler: _Msg_ReviewApplication_Handler},
		{MethodName: "SetAvailability", Handler: _Msg_SetAvailability_Handler},
		// Services
		{MethodName: "DeployService", Handler: _Msg_DeployService_Handler},
		{MethodName: "PauseService", Handler: _Msg_PauseService_Handler},
		{MethodName: "ResumeService", Handler: _Msg_ResumeService_Handler},
		{MethodName: "RetireService", Handler: _Msg_RetireService_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

// --- Project handlers ---

func _Msg_CreateProject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCreateProject)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CreateProject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/CreateProject"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CreateProject(ctx, req.(*MsgCreateProject))
	})
}

func _Msg_ProposeProject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgProposeProject)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ProposeProject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/ProposeProject"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ProposeProject(ctx, req.(*MsgProposeProject))
	})
}

func _Msg_StartDevelopment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgStartDevelopment)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).StartDevelopment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/StartDevelopment"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).StartDevelopment(ctx, req.(*MsgStartDevelopment))
	})
}

func _Msg_CompleteProject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCompleteProject)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CompleteProject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/CompleteProject"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CompleteProject(ctx, req.(*MsgCompleteProject))
	})
}

func _Msg_PauseProject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgPauseProject)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).PauseProject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/PauseProject"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).PauseProject(ctx, req.(*MsgPauseProject))
	})
}

func _Msg_ResumeProject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgResumeProject)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ResumeProject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/ResumeProject"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ResumeProject(ctx, req.(*MsgResumeProject))
	})
}

func _Msg_AbandonProject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAbandonProject)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AbandonProject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/AbandonProject"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AbandonProject(ctx, req.(*MsgAbandonProject))
	})
}

func _Msg_SpawnChildProject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSpawnChildProject)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SpawnChildProject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/SpawnChildProject"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SpawnChildProject(ctx, req.(*MsgSpawnChildProject))
	})
}

// --- Task handlers ---

func _Msg_AddTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAddTask)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AddTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/AddTask"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AddTask(ctx, req.(*MsgAddTask))
	})
}

func _Msg_AssignTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAssignTask)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AssignTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/AssignTask"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AssignTask(ctx, req.(*MsgAssignTask))
	})
}

func _Msg_StartWork_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgStartWork)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).StartWork(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/StartWork"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).StartWork(ctx, req.(*MsgStartWork))
	})
}

func _Msg_SubmitDeliverable_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitDeliverable)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SubmitDeliverable(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/SubmitDeliverable"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SubmitDeliverable(ctx, req.(*MsgSubmitDeliverable))
	})
}

func _Msg_ApproveDeliverable_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgApproveDeliverable)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ApproveDeliverable(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/ApproveDeliverable"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ApproveDeliverable(ctx, req.(*MsgApproveDeliverable))
	})
}

func _Msg_RejectDeliverable_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRejectDeliverable)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RejectDeliverable(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/RejectDeliverable"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RejectDeliverable(ctx, req.(*MsgRejectDeliverable))
	})
}

func _Msg_ReopenTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgReopenTask)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ReopenTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/ReopenTask"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ReopenTask(ctx, req.(*MsgReopenTask))
	})
}

// --- Application handlers ---

func _Msg_ApplyToProject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgApplyToProject)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ApplyToProject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/ApplyToProject"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ApplyToProject(ctx, req.(*MsgApplyToProject))
	})
}

func _Msg_ReviewApplication_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgReviewApplication)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ReviewApplication(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/ReviewApplication"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ReviewApplication(ctx, req.(*MsgReviewApplication))
	})
}

func _Msg_SetAvailability_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSetAvailability)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SetAvailability(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/SetAvailability"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SetAvailability(ctx, req.(*MsgSetAvailability))
	})
}

// --- Service handlers ---

func _Msg_DeployService_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgDeployService)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).DeployService(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/DeployService"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).DeployService(ctx, req.(*MsgDeployService))
	})
}

func _Msg_PauseService_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgPauseService)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).PauseService(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/PauseService"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).PauseService(ctx, req.(*MsgPauseService))
	})
}

func _Msg_ResumeService_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgResumeService)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ResumeService(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/ResumeService"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ResumeService(ctx, req.(*MsgResumeService))
	})
}

func _Msg_RetireService_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRetireService)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RetireService(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.tree.v1.Msg/RetireService"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RetireService(ctx, req.(*MsgRetireService))
	})
}

// RegisterMsgServer registers the tree msg server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// ======================== Query Service Descriptor ========================

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
	Streams: []grpc.StreamDesc{},
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

// RegisterQueryServer registers the tree query server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
