package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// RegisterCodec registers the bvm module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgDeployContract{}, "bvm/DeployContract", nil)
	cdc.RegisterConcrete(&MsgCallContract{}, "bvm/CallContract", nil)
	cdc.RegisterConcrete(&MsgScheduleContract{}, "bvm/ScheduleContract", nil)
	cdc.RegisterConcrete(&MsgCancelSchedule{}, "bvm/CancelSchedule", nil)
	cdc.RegisterConcrete(&MsgUpdateContractState{}, "bvm/UpdateContractState", nil)
}

// RegisterInterfaces registers the bvm module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDeployContract{},
		&MsgCallContract{},
		&MsgScheduleExecution{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// --- Query service descriptor (no query.proto generated yet) ---

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.bvm.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Contract", Handler: _Query_Contract_Handler},
		{MethodName: "ContractsByCreator", Handler: _Query_ContractsByCreator_Handler},
		{MethodName: "Code", Handler: _Query_Code_Handler},
		{MethodName: "ContractState", Handler: _Query_ContractState_Handler},
		{MethodName: "Schedule", Handler: _Query_Schedule_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Query_Contract_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryContractRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Contract(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.bvm.v1.Query/Contract"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Contract(ctx, req.(*QueryContractRequest))
	})
}

func _Query_ContractsByCreator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryContractsByCreatorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ContractsByCreator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.bvm.v1.Query/ContractsByCreator"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ContractsByCreator(ctx, req.(*QueryContractsByCreatorRequest))
	})
}

func _Query_Code_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryCodeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Code(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.bvm.v1.Query/Code"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Code(ctx, req.(*QueryCodeRequest))
	})
}

func _Query_ContractState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryContractStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ContractState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.bvm.v1.Query/ContractState"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ContractState(ctx, req.(*QueryContractStateRequest))
	})
}

func _Query_Schedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryScheduleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Schedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.bvm.v1.Query/Schedule"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Schedule(ctx, req.(*QueryScheduleRequest))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.bvm.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

// RegisterQueryServer registers the bvm query server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
