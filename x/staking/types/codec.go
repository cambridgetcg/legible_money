package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/grpc"
)

// RegisterCodec registers concrete types on codec
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterValidator{}, "legible_staking/RegisterValidator", nil)
	cdc.RegisterConcrete(&MsgUpdateValidatorStake{}, "legible_staking/UpdateValidatorStake", nil)
	cdc.RegisterConcrete(&MsgDelegate{}, "legible_staking/Delegate", nil)
	cdc.RegisterConcrete(&MsgUndelegate{}, "legible_staking/Undelegate", nil)
	cdc.RegisterConcrete(&MsgRedelegate{}, "legible_staking/Redelegate", nil)
}

// RegisterInterfaces registers the interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterValidator{},
	)
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

func init() {
	RegisterCodec(amino)
	amino.Seal()
}

// Note: RegisterMsgServer and Msg_ServiceDesc are now defined in tx_grpc.pb.go (proto-generated).

// ======================== Query Service Descriptor ========================

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.staking.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Validator", Handler: _Query_Validator_Handler},
		{MethodName: "Validators", Handler: _Query_Validators_Handler},
		{MethodName: "Delegation", Handler: _Query_Delegation_Handler},
		{MethodName: "DelegationsForValidator", Handler: _Query_DelegationsForValidator_Handler},
		{MethodName: "Unbondings", Handler: _Query_Unbondings_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/staking/v1/query.proto",
}

func _Query_Validator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryValidatorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Validator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.staking.v1.Query/Validator"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Validator(ctx, req.(*QueryValidatorRequest))
	})
}

func _Query_Validators_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryValidatorsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Validators(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.staking.v1.Query/Validators"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Validators(ctx, req.(*QueryValidatorsRequest))
	})
}

func _Query_Delegation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDelegationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Delegation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.staking.v1.Query/Delegation"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Delegation(ctx, req.(*QueryDelegationRequest))
	})
}

func _Query_DelegationsForValidator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDelegationsForValidatorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).DelegationsForValidator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.staking.v1.Query/DelegationsForValidator"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).DelegationsForValidator(ctx, req.(*QueryDelegationsForValidatorRequest))
	})
}

func _Query_Unbondings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryUnbondingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Unbondings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.staking.v1.Query/Unbondings"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Unbondings(ctx, req.(*QueryUnbondingsRequest))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.staking.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}
