package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"google.golang.org/grpc"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgClaimVestedRewards{}, "vesting_rewards/ClaimVestedRewards", nil)
	cdc.RegisterConcrete(&MsgFalsifyClaim{}, "vesting_rewards/FalsifyClaim", nil)
	cdc.RegisterConcrete(&MsgCreateVesting{}, "vesting_rewards/CreateVesting", nil)
	cdc.RegisterConcrete(&MsgPauseVesting{}, "vesting_rewards/PauseVesting", nil)
	cdc.RegisterConcrete(&MsgResumeVesting{}, "vesting_rewards/ResumeVesting", nil)
	cdc.RegisterConcrete(&MsgAccelerateVesting{}, "vesting_rewards/AccelerateVesting", nil)
	cdc.RegisterConcrete(&MsgCompleteVesting{}, "vesting_rewards/CompleteVesting", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}

// --- Msg gRPC Service Descriptor ---

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.vesting_rewards.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "ClaimVestedRewards", Handler: _Msg_ClaimVestedRewards_Handler},
		{MethodName: "FalsifyClaim", Handler: _Msg_FalsifyClaim_Handler},
		{MethodName: "CreateVesting", Handler: _Msg_CreateVesting_Handler},
		{MethodName: "PauseVesting", Handler: _Msg_PauseVesting_Handler},
		{MethodName: "ResumeVesting", Handler: _Msg_ResumeVesting_Handler},
		{MethodName: "AccelerateVesting", Handler: _Msg_AccelerateVesting_Handler},
		{MethodName: "CompleteVesting", Handler: _Msg_CompleteVesting_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/vesting_rewards/v1/tx.proto",
}

func _Msg_ClaimVestedRewards_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgClaimVestedRewards)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ClaimVestedRewards(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Msg/ClaimVestedRewards"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ClaimVestedRewards(ctx, req.(*MsgClaimVestedRewards))
	})
}

func _Msg_FalsifyClaim_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgFalsifyClaim)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).FalsifyClaim(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Msg/FalsifyClaim"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).FalsifyClaim(ctx, req.(*MsgFalsifyClaim))
	})
}

func _Msg_CreateVesting_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCreateVesting)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CreateVesting(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Msg/CreateVesting"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CreateVesting(ctx, req.(*MsgCreateVesting))
	})
}

func _Msg_PauseVesting_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgPauseVesting)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).PauseVesting(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Msg/PauseVesting"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).PauseVesting(ctx, req.(*MsgPauseVesting))
	})
}

func _Msg_ResumeVesting_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgResumeVesting)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ResumeVesting(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Msg/ResumeVesting"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ResumeVesting(ctx, req.(*MsgResumeVesting))
	})
}

func _Msg_AccelerateVesting_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAccelerateVesting)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AccelerateVesting(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Msg/AccelerateVesting"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AccelerateVesting(ctx, req.(*MsgAccelerateVesting))
	})
}

func _Msg_CompleteVesting_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCompleteVesting)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CompleteVesting(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Msg/CompleteVesting"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CompleteVesting(ctx, req.(*MsgCompleteVesting))
	})
}

// --- Query gRPC Service Descriptor ---

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.vesting_rewards.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "VestingSchedule", Handler: _Query_VestingSchedule_Handler},
		{MethodName: "VestingSchedulesByRecipient", Handler: _Query_VestingSchedulesByRecipient_Handler},
		{MethodName: "ActiveVestingSchedules", Handler: _Query_ActiveVestingSchedules_Handler},
		{MethodName: "BlockRewardDistribution", Handler: _Query_BlockRewardDistribution_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/vesting_rewards/v1/query.proto",
}

func _Query_VestingSchedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryVestingScheduleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).VestingSchedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Query/VestingSchedule"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).VestingSchedule(ctx, req.(*QueryVestingScheduleRequest))
	})
}

func _Query_VestingSchedulesByRecipient_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryVestingSchedulesByRecipientRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).VestingSchedulesByRecipient(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Query/VestingSchedulesByRecipient"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).VestingSchedulesByRecipient(ctx, req.(*QueryVestingSchedulesByRecipientRequest))
	})
}

func _Query_ActiveVestingSchedules_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryActiveVestingSchedulesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ActiveVestingSchedules(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Query/ActiveVestingSchedules"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ActiveVestingSchedules(ctx, req.(*QueryActiveVestingSchedulesRequest))
	})
}

func _Query_BlockRewardDistribution_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryBlockRewardDistributionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).BlockRewardDistribution(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Query/BlockRewardDistribution"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).BlockRewardDistribution(ctx, req.(*QueryBlockRewardDistributionRequest))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.vesting_rewards.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}
