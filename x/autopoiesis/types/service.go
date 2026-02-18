package types

import (
	"context"

	"google.golang.org/grpc"
)

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.autopoiesis.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "UpdateParams", Handler: _Msg_UpdateParams_Handler},
		{MethodName: "Activate", Handler: _Msg_Activate_Handler},
		{MethodName: "OverrideMultiplier", Handler: _Msg_OverrideMultiplier_Handler},
		{MethodName: "FreezeMultiplier", Handler: _Msg_FreezeMultiplier_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/autopoiesis/v1/tx.proto",
}

func _Msg_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.autopoiesis.v1.Msg/UpdateParams"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	})
}

func _Msg_Activate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgActivateAutopoiesis)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).Activate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.autopoiesis.v1.Msg/Activate"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).Activate(ctx, req.(*MsgActivateAutopoiesis))
	})
}

func _Msg_OverrideMultiplier_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgOverrideMultiplier)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).OverrideMultiplier(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.autopoiesis.v1.Msg/OverrideMultiplier"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).OverrideMultiplier(ctx, req.(*MsgOverrideMultiplier))
	})
}

func _Msg_FreezeMultiplier_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgFreezeMultiplier)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).FreezeMultiplier(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.autopoiesis.v1.Msg/FreezeMultiplier"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).FreezeMultiplier(ctx, req.(*MsgFreezeMultiplier))
	})
}
