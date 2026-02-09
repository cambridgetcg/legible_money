package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"

	"google.golang.org/grpc"
)

// RegisterCodec registers the module's types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgOpenDispute{}, "disputes/OpenDispute", nil)
	cdc.RegisterConcrete(&MsgCommitEvidence{}, "disputes/CommitEvidence", nil)
	cdc.RegisterConcrete(&MsgSubmitEvidence{}, "disputes/SubmitEvidence", nil)
	cdc.RegisterConcrete(&MsgVoteDispute{}, "disputes/VoteDispute", nil)
	cdc.RegisterConcrete(&MsgEscalateDispute{}, "disputes/EscalateDispute", nil)
	cdc.RegisterConcrete(&MsgSettleDispute{}, "disputes/SettleDispute", nil)
	cdc.RegisterConcrete(&MsgTimeoutDispute{}, "disputes/TimeoutDispute", nil)
}

// RegisterInterfaces registers the module's interface types with the interface registry.
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

// RegisterQueryServer is defined in query.go (re-export not needed).

// ---------- gRPC Msg Service Descriptor ----------

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.disputes.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "OpenDispute", Handler: _Msg_OpenDispute_Handler},
		{MethodName: "CommitEvidence", Handler: _Msg_CommitEvidence_Handler},
		{MethodName: "SubmitEvidence", Handler: _Msg_SubmitEvidence_Handler},
		{MethodName: "VoteDispute", Handler: _Msg_VoteDispute_Handler},
		{MethodName: "EscalateDispute", Handler: _Msg_EscalateDispute_Handler},
		{MethodName: "SettleDispute", Handler: _Msg_SettleDispute_Handler},
		{MethodName: "TimeoutDispute", Handler: _Msg_TimeoutDispute_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/disputes/v1/tx.proto",
}

func _Msg_OpenDispute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgOpenDispute)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).OpenDispute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Msg/OpenDispute"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).OpenDispute(ctx, req.(*MsgOpenDispute))
	})
}

func _Msg_CommitEvidence_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCommitEvidence)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CommitEvidence(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Msg/CommitEvidence"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CommitEvidence(ctx, req.(*MsgCommitEvidence))
	})
}

func _Msg_SubmitEvidence_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitEvidence)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SubmitEvidence(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Msg/SubmitEvidence"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SubmitEvidence(ctx, req.(*MsgSubmitEvidence))
	})
}

func _Msg_VoteDispute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgVoteDispute)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).VoteDispute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Msg/VoteDispute"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).VoteDispute(ctx, req.(*MsgVoteDispute))
	})
}

func _Msg_EscalateDispute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgEscalateDispute)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).EscalateDispute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Msg/EscalateDispute"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).EscalateDispute(ctx, req.(*MsgEscalateDispute))
	})
}

func _Msg_SettleDispute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSettleDispute)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SettleDispute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Msg/SettleDispute"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SettleDispute(ctx, req.(*MsgSettleDispute))
	})
}

func _Msg_TimeoutDispute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgTimeoutDispute)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).TimeoutDispute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.disputes.v1.Msg/TimeoutDispute"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).TimeoutDispute(ctx, req.(*MsgTimeoutDispute))
	})
}
