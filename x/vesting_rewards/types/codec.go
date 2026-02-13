package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateVesting{}, "vesting_rewards/CreateVesting", nil)
	cdc.RegisterConcrete(&MsgClaimVesting{}, "vesting_rewards/ClaimVesting", nil)
	cdc.RegisterConcrete(&MsgFalsifyVesting{}, "vesting_rewards/FalsifyVesting", nil)
	cdc.RegisterConcrete(&MsgPauseVesting{}, "vesting_rewards/PauseVesting", nil)
	cdc.RegisterConcrete(&MsgResumeVesting{}, "vesting_rewards/ResumeVesting", nil)
	cdc.RegisterConcrete(&MsgAccelerateVesting{}, "vesting_rewards/AccelerateVesting", nil)
	cdc.RegisterConcrete(&MsgCompleteVesting{}, "vesting_rewards/CompleteVesting", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "vesting_rewards/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateVesting{},
		&MsgClaimVesting{},
		&MsgPauseVesting{},
		&MsgResumeVesting{},
		&MsgAccelerateVesting{},
		&MsgFalsifyVesting{},
		&MsgCompleteVesting{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
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
