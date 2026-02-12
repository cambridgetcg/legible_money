package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"google.golang.org/grpc"
)

// RegisterCodec registers the channels module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgOpenChannel{}, "channels/OpenChannel", nil)
	cdc.RegisterConcrete(&MsgDepositChannel{}, "channels/DepositChannel", nil)
	cdc.RegisterConcrete(&MsgUpdateState{}, "channels/UpdateState", nil)
	cdc.RegisterConcrete(&MsgDisputeChannel{}, "channels/DisputeChannel", nil)
	cdc.RegisterConcrete(&MsgSettleChannel{}, "channels/SettleChannel", nil)
	cdc.RegisterConcrete(&MsgCloseChannel{}, "channels/CloseChannel", nil)
	cdc.RegisterConcrete(&MsgClaimExpired{}, "channels/ClaimExpired", nil)
}

// RegisterInterfaces registers the channels module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// --- Supplemental Msg handlers for methods not in protobuf service definition ---

// SupplementalMsgServer defines additional message handlers beyond the protobuf Msg service.
type SupplementalMsgServer interface {
	DepositChannel(context.Context, *MsgDepositChannel) (*MsgDepositChannelResponse, error)
	UpdateState(context.Context, *MsgUpdateState) (*MsgUpdateStateResponse, error)
	ClaimExpired(context.Context, *MsgClaimExpired) (*MsgClaimExpiredResponse, error)
}

// FullMsgServer combines the protobuf-generated MsgServer with supplemental methods.
type FullMsgServer interface {
	MsgServer
	SupplementalMsgServer
}

func _Msg_DepositChannel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgDepositChannel)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SupplementalMsgServer).DepositChannel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.channels.v1.Msg/DepositChannel"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SupplementalMsgServer).DepositChannel(ctx, req.(*MsgDepositChannel))
	})
}

func _Msg_UpdateState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateState)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SupplementalMsgServer).UpdateState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.channels.v1.Msg/UpdateState"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SupplementalMsgServer).UpdateState(ctx, req.(*MsgUpdateState))
	})
}

func _Msg_ClaimExpired_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgClaimExpired)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SupplementalMsgServer).ClaimExpired(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.channels.v1.Msg/ClaimExpired"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SupplementalMsgServer).ClaimExpired(ctx, req.(*MsgClaimExpired))
	})
}

// --- Query service descriptor (no protobuf query service was generated) ---

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.channels.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Channel", Handler: _Query_Channel_Handler},
		{MethodName: "ChannelsByPayer", Handler: _Query_ChannelsByPayer_Handler},
		{MethodName: "ChannelsByProvider", Handler: _Query_ChannelsByProvider_Handler},
		{MethodName: "ActiveDispute", Handler: _Query_ActiveDispute_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Query_Channel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryChannelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Channel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.channels.v1.Query/Channel"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Channel(ctx, req.(*QueryChannelRequest))
	})
}

func _Query_ChannelsByPayer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryChannelsByPayerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ChannelsByPayer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.channels.v1.Query/ChannelsByPayer"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ChannelsByPayer(ctx, req.(*QueryChannelsByPayerRequest))
	})
}

func _Query_ChannelsByProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryChannelsByProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ChannelsByProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.channels.v1.Query/ChannelsByProvider"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ChannelsByProvider(ctx, req.(*QueryChannelsByProviderRequest))
	})
}

func _Query_ActiveDispute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryActiveDisputeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ActiveDispute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.channels.v1.Query/ActiveDispute"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ActiveDispute(ctx, req.(*QueryActiveDisputeRequest))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.channels.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

// RegisterQueryServer registers the channels query server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
