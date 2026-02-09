package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"

	"google.golang.org/grpc"
)

// RegisterCodec registers the legible_auth module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterAccount{}, "legible_auth/RegisterAccount", nil)
	cdc.RegisterConcrete(&MsgRotateKey{}, "legible_auth/RotateKey", nil)
	cdc.RegisterConcrete(&MsgCreateSession{}, "legible_auth/CreateSession", nil)
	cdc.RegisterConcrete(&MsgRevokeSession{}, "legible_auth/RevokeSession", nil)
	cdc.RegisterConcrete(&MsgFreezeAccount{}, "legible_auth/FreezeAccount", nil)
	cdc.RegisterConcrete(&MsgUnfreezeAccount{}, "legible_auth/UnfreezeAccount", nil)
	cdc.RegisterConcrete(&MsgSetRecoveryConfig{}, "legible_auth/SetRecoveryConfig", nil)
	cdc.RegisterConcrete(&MsgInitiateRecovery{}, "legible_auth/InitiateRecovery", nil)
	cdc.RegisterConcrete(&MsgSubmitRecoveryShard{}, "legible_auth/SubmitRecoveryShard", nil)
	cdc.RegisterConcrete(&MsgChallengeRecovery{}, "legible_auth/ChallengeRecovery", nil)
	cdc.RegisterConcrete(&MsgExecuteRecovery{}, "legible_auth/ExecuteRecovery", nil)
}

// RegisterInterfaces registers the legible_auth module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// RegisterMsgServer registers the legible_auth msg server with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// gRPC service descriptor for msg server
var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.auth.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "RegisterAccount", Handler: _Msg_RegisterAccount_Handler},
		{MethodName: "RotateKey", Handler: _Msg_RotateKey_Handler},
		{MethodName: "CreateSession", Handler: _Msg_CreateSession_Handler},
		{MethodName: "RevokeSession", Handler: _Msg_RevokeSession_Handler},
		{MethodName: "FreezeAccount", Handler: _Msg_FreezeAccount_Handler},
		{MethodName: "UnfreezeAccount", Handler: _Msg_UnfreezeAccount_Handler},
		{MethodName: "SetRecoveryConfig", Handler: _Msg_SetRecoveryConfig_Handler},
		{MethodName: "InitiateRecovery", Handler: _Msg_InitiateRecovery_Handler},
		{MethodName: "SubmitRecoveryShard", Handler: _Msg_SubmitRecoveryShard_Handler},
		{MethodName: "ChallengeRecovery", Handler: _Msg_ChallengeRecovery_Handler},
		{MethodName: "ExecuteRecovery", Handler: _Msg_ExecuteRecovery_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/auth/v1/tx.proto",
}

func _Msg_RegisterAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterAccount)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RegisterAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/RegisterAccount"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RegisterAccount(ctx, req.(*MsgRegisterAccount))
	})
}

func _Msg_RotateKey_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRotateKey)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RotateKey(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/RotateKey"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RotateKey(ctx, req.(*MsgRotateKey))
	})
}

func _Msg_CreateSession_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCreateSession)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CreateSession(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/CreateSession"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CreateSession(ctx, req.(*MsgCreateSession))
	})
}

func _Msg_RevokeSession_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRevokeSession)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RevokeSession(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/RevokeSession"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RevokeSession(ctx, req.(*MsgRevokeSession))
	})
}

func _Msg_FreezeAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgFreezeAccount)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).FreezeAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/FreezeAccount"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).FreezeAccount(ctx, req.(*MsgFreezeAccount))
	})
}

func _Msg_UnfreezeAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUnfreezeAccount)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UnfreezeAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/UnfreezeAccount"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UnfreezeAccount(ctx, req.(*MsgUnfreezeAccount))
	})
}

func _Msg_SetRecoveryConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSetRecoveryConfig)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SetRecoveryConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/SetRecoveryConfig"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SetRecoveryConfig(ctx, req.(*MsgSetRecoveryConfig))
	})
}

func _Msg_InitiateRecovery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgInitiateRecovery)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).InitiateRecovery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/InitiateRecovery"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).InitiateRecovery(ctx, req.(*MsgInitiateRecovery))
	})
}

func _Msg_SubmitRecoveryShard_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitRecoveryShard)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SubmitRecoveryShard(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/SubmitRecoveryShard"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SubmitRecoveryShard(ctx, req.(*MsgSubmitRecoveryShard))
	})
}

func _Msg_ChallengeRecovery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgChallengeRecovery)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ChallengeRecovery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/ChallengeRecovery"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ChallengeRecovery(ctx, req.(*MsgChallengeRecovery))
	})
}

func _Msg_ExecuteRecovery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgExecuteRecovery)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ExecuteRecovery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.Msg/ExecuteRecovery"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ExecuteRecovery(ctx, req.(*MsgExecuteRecovery))
	})
}
