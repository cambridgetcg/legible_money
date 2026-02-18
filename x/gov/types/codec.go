package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	cdc.RegisterConcrete(&MsgAttachUpgradePlan{}, "lgm_gov/AttachUpgradePlan", nil)
}

// RegisterInterfaces registers the governance module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitLIP{},
		&MsgStakeLIP{},
		&MsgAmendLIP{},
		&MsgAdvanceLIPStage{},
		&MsgWithdrawLIP{},
		&MsgCastVote{},
		&MsgLockVote{},
		&MsgUnlockVote{},
		&MsgSwitchVote{},
		&MsgCommitReview{},
		&MsgRevealReview{},
		&MsgFinalizeReview{},
		&MsgSubmitDisbursement{},
		&MsgStakeDisbursement{},
		&MsgAdvanceDisbursement{},
		&MsgExecuteDisbursement{},
		&MsgCancelDisbursement{},
		&MsgRegisterOperator{},
		&MsgAddAgent{},
		&MsgRemoveAgent{},
		&MsgSlashOperator{},
		&MsgCreateDeployment{},
		&MsgAdvanceDeployment{},
		&MsgApproveDeployment{},
		&MsgRollbackDeployment{},
		// Hand-written (types.go) — have XXX_MessageName in proto_names.go
		&MsgUpdateParams{},
		&MsgAttachUpgradePlan{},
	)
}

// --- gRPC service descriptors for query server ---
// (No query_grpc.pb.go exists, so these remain hand-written.)

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
