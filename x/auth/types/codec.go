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
	cdc.RegisterConcrete(&MsgUpdateParams{}, "legible_auth/UpdateParams", nil)
}

// RegisterInterfaces registers the legible_auth module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// RegisterExtendedMsgServer registers the extended msg server that includes
// both proto-generated handlers (via Msg_ServiceDesc) and hand-written
// handlers for message types not yet in the proto service definition.
func RegisterExtendedMsgServer(s grpc.ServiceRegistrar, srv ExtendedMsgServer) {
	// Register the proto-generated service (RotateKey, CreateSession, RevokeSession, UpdateParams)
	RegisterMsgServer(s, srv)

	// Register extended handlers for non-proto message types
	s.RegisterService(&_ExtendedMsg_serviceDesc, srv)
}

// gRPC service descriptor for extended (non-proto) message handlers
var _ExtendedMsg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.auth.v1.ExtendedMsg",
	HandlerType: (*ExtendedMsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "RegisterAccount", Handler: _Msg_RegisterAccount_Handler},
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
		return srv.(ExtendedMsgServer).RegisterAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.ExtendedMsg/RegisterAccount"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).RegisterAccount(ctx, req.(*MsgRegisterAccount))
	})
}

func _Msg_FreezeAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgFreezeAccount)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).FreezeAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.ExtendedMsg/FreezeAccount"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).FreezeAccount(ctx, req.(*MsgFreezeAccount))
	})
}

func _Msg_UnfreezeAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUnfreezeAccount)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).UnfreezeAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.ExtendedMsg/UnfreezeAccount"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).UnfreezeAccount(ctx, req.(*MsgUnfreezeAccount))
	})
}

func _Msg_SetRecoveryConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSetRecoveryConfig)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).SetRecoveryConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.ExtendedMsg/SetRecoveryConfig"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).SetRecoveryConfig(ctx, req.(*MsgSetRecoveryConfig))
	})
}

func _Msg_InitiateRecovery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgInitiateRecovery)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).InitiateRecovery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.ExtendedMsg/InitiateRecovery"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).InitiateRecovery(ctx, req.(*MsgInitiateRecovery))
	})
}

func _Msg_SubmitRecoveryShard_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitRecoveryShard)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).SubmitRecoveryShard(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.ExtendedMsg/SubmitRecoveryShard"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).SubmitRecoveryShard(ctx, req.(*MsgSubmitRecoveryShard))
	})
}

func _Msg_ChallengeRecovery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgChallengeRecovery)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).ChallengeRecovery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.ExtendedMsg/ChallengeRecovery"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).ChallengeRecovery(ctx, req.(*MsgChallengeRecovery))
	})
}

func _Msg_ExecuteRecovery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgExecuteRecovery)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).ExecuteRecovery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.auth.v1.ExtendedMsg/ExecuteRecovery"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).ExecuteRecovery(ctx, req.(*MsgExecuteRecovery))
	})
}
