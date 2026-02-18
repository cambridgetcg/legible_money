package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	cdc.RegisterConcrete(&MsgUpdateParams{}, "channels/UpdateParams", nil)
}

// RegisterInterfaces registers the channels module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgOpenChannel{},
		&MsgUpdateChannel{},
		&MsgCloseChannel{},
		&MsgSettleChannel{},
		&MsgDisputeChannel{},
		&MsgUpdateParams{},
	)
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
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
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

// _SupplementalMsg_serviceDesc is the gRPC service descriptor for supplemental
// message handlers not in the protobuf Msg service definition.
var _SupplementalMsg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.channels.v1.SupplementalMsg",
	HandlerType: (*SupplementalMsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "DepositChannel", Handler: _Msg_DepositChannel_Handler},
		{MethodName: "UpdateState", Handler: _Msg_UpdateState_Handler},
		{MethodName: "ClaimExpired", Handler: _Msg_ClaimExpired_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/channels/v1/tx.proto",
}

// RegisterFullMsgServer registers the full msg server that includes both
// proto-generated handlers and supplemental handlers.
func RegisterFullMsgServer(s grpc.ServiceRegistrar, srv FullMsgServer) {
	RegisterMsgServer(s, srv)
	s.RegisterService(&_SupplementalMsg_serviceDesc, srv)
}

// Query service descriptor, handlers, and RegisterQueryServer are now provided
// by the proto-generated query_grpc.pb.go file.
