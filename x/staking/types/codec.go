package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

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
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

func init() {
	RegisterCodec(amino)
	amino.Seal()
}

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
// This will be auto-generated from proto; hand-written stub for Phase 1.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_MsgService_serviceDesc, srv)
}

var _MsgService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.staking.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "RegisterValidator", Handler: nil},
		{MethodName: "UpdateValidatorStake", Handler: nil},
		{MethodName: "Delegate", Handler: nil},
		{MethodName: "Undelegate", Handler: nil},
		{MethodName: "Redelegate", Handler: nil},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/staking/v1/tx.proto",
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.staking.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Validator", Handler: nil},
		{MethodName: "Validators", Handler: nil},
		{MethodName: "Delegation", Handler: nil},
		{MethodName: "DelegationsForValidator", Handler: nil},
		{MethodName: "Unbondings", Handler: nil},
		{MethodName: "Params", Handler: nil},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/staking/v1/query.proto",
}
