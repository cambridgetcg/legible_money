package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"google.golang.org/grpc"
)

// RegisterCodec registers the governance module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitLIP{}, "lgm_gov/SubmitLIP", nil)
	cdc.RegisterConcrete(&MsgStakeLIP{}, "lgm_gov/StakeLIP", nil)
	cdc.RegisterConcrete(&MsgCastVote{}, "lgm_gov/CastVote", nil)
	cdc.RegisterConcrete(&MsgAdvanceLIPStage{}, "lgm_gov/AdvanceLIPStage", nil)
	cdc.RegisterConcrete(&MsgWithdrawLIP{}, "lgm_gov/WithdrawLIP", nil)
	cdc.RegisterConcrete(&MsgSubmitDisbursement{}, "lgm_gov/SubmitDisbursement", nil)
	cdc.RegisterConcrete(&MsgExecuteDisbursement{}, "lgm_gov/ExecuteDisbursement", nil)
	cdc.RegisterConcrete(&MsgCancelDisbursement{}, "lgm_gov/CancelDisbursement", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "lgm_gov/UpdateParams", nil)
}

// RegisterInterfaces registers the governance module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

// --- gRPC service descriptors for msg server ---

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.lgm_gov.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "SubmitLIP", Handler: _Msg_SubmitLIP_Handler},
		{MethodName: "StakeLIP", Handler: _Msg_StakeLIP_Handler},
		{MethodName: "CastVote", Handler: _Msg_CastVote_Handler},
		{MethodName: "AdvanceLIPStage", Handler: _Msg_AdvanceLIPStage_Handler},
		{MethodName: "WithdrawLIP", Handler: _Msg_WithdrawLIP_Handler},
		{MethodName: "SubmitDisbursement", Handler: _Msg_SubmitDisbursement_Handler},
		{MethodName: "ExecuteDisbursement", Handler: _Msg_ExecuteDisbursement_Handler},
		{MethodName: "CancelDisbursement", Handler: _Msg_CancelDisbursement_Handler},
		{MethodName: "UpdateParams", Handler: _Msg_UpdateParams_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Msg_SubmitLIP_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitLIP)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SubmitLIP(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Msg/SubmitLIP"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SubmitLIP(ctx, req.(*MsgSubmitLIP))
	})
}

func _Msg_StakeLIP_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgStakeLIP)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).StakeLIP(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Msg/StakeLIP"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).StakeLIP(ctx, req.(*MsgStakeLIP))
	})
}

func _Msg_CastVote_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCastVote)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CastVote(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Msg/CastVote"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CastVote(ctx, req.(*MsgCastVote))
	})
}

func _Msg_AdvanceLIPStage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAdvanceLIPStage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AdvanceLIPStage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Msg/AdvanceLIPStage"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AdvanceLIPStage(ctx, req.(*MsgAdvanceLIPStage))
	})
}

func _Msg_WithdrawLIP_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgWithdrawLIP)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).WithdrawLIP(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Msg/WithdrawLIP"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).WithdrawLIP(ctx, req.(*MsgWithdrawLIP))
	})
}

func _Msg_SubmitDisbursement_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitDisbursement)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SubmitDisbursement(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Msg/SubmitDisbursement"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SubmitDisbursement(ctx, req.(*MsgSubmitDisbursement))
	})
}

func _Msg_ExecuteDisbursement_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgExecuteDisbursement)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ExecuteDisbursement(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Msg/ExecuteDisbursement"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ExecuteDisbursement(ctx, req.(*MsgExecuteDisbursement))
	})
}

func _Msg_CancelDisbursement_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCancelDisbursement)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CancelDisbursement(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Msg/CancelDisbursement"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CancelDisbursement(ctx, req.(*MsgCancelDisbursement))
	})
}

func _Msg_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Msg/UpdateParams"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	})
}

// RegisterMsgServer registers the governance msg server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// --- gRPC service descriptors for query server ---

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.lgm_gov.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "LIP", Handler: _Query_LIP_Handler},
		{MethodName: "LIPs", Handler: _Query_LIPs_Handler},
		{MethodName: "Disbursement", Handler: _Query_Disbursement_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Query_LIP_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryLIPRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).LIP(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Query/LIP"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).LIP(ctx, req.(*QueryLIPRequest))
	})
}

func _Query_LIPs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryLIPsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).LIPs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Query/LIPs"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).LIPs(ctx, req.(*QueryLIPsRequest))
	})
}

func _Query_Disbursement_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDisbursementRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Disbursement(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Query/Disbursement"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Disbursement(ctx, req.(*QueryDisbursementRequest))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.lgm_gov.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

// RegisterQueryServer registers the governance query server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
