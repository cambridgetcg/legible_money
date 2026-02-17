package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// RegisterCodec registers the home module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateHome{}, "home/CreateHome", nil)
	cdc.RegisterConcrete(&MsgUpdateHome{}, "home/UpdateHome", nil)
	cdc.RegisterConcrete(&MsgUpdateMemoryCID{}, "home/UpdateMemoryCID", nil)
	cdc.RegisterConcrete(&MsgStartSession{}, "home/StartSession", nil)
	cdc.RegisterConcrete(&MsgEndSession{}, "home/EndSession", nil)
	cdc.RegisterConcrete(&MsgRegisterKey{}, "home/RegisterKey", nil)
	cdc.RegisterConcrete(&MsgRevokeKey{}, "home/RevokeKey", nil)
	cdc.RegisterConcrete(&MsgConfigureGuardian{}, "home/ConfigureGuardian", nil)
	cdc.RegisterConcrete(&MsgAcknowledgeAlert{}, "home/AcknowledgeAlert", nil)
	cdc.RegisterConcrete(&MsgSetSpendingLimit{}, "home/SetSpendingLimit", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "home/UpdateParams", nil)
}

// RegisterInterfaces registers the home module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateHome{},
		&MsgUpdateHome{},
		&MsgUpdateMemoryCID{},
		&MsgStartSession{},
		&MsgEndSession{},
		&MsgRegisterKey{},
		&MsgRevokeKey{},
		&MsgConfigureGuardian{},
		&MsgAcknowledgeAlert{},
		&MsgSetSpendingLimit{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// --- Query service descriptor ---

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.home.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Home", Handler: _Query_Home_Handler},
		{MethodName: "Homes", Handler: _Query_Homes_Handler},
		{MethodName: "Sessions", Handler: _Query_Sessions_Handler},
		{MethodName: "Alerts", Handler: _Query_Alerts_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Query_Home_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryHomeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Home(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.home.v1.Query/Home"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Home(ctx, req.(*QueryHomeRequest))
	})
}

func _Query_Homes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryHomesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Homes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.home.v1.Query/Homes"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Homes(ctx, req.(*QueryHomesRequest))
	})
}

func _Query_Sessions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuerySessionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Sessions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.home.v1.Query/Sessions"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Sessions(ctx, req.(*QuerySessionsRequest))
	})
}

func _Query_Alerts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryAlertsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Alerts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.home.v1.Query/Alerts"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Alerts(ctx, req.(*QueryAlertsRequest))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.home.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

// RegisterQueryServer registers the home query server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
