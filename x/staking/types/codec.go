package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/grpc"
)

// RegisterCodec registers concrete types on codec
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterValidator{}, "legible_staking/RegisterValidator", nil)
	cdc.RegisterConcrete(&MsgUpdateValidatorStake{}, "legible_staking/UpdateValidatorStake", nil)
}

// RegisterInterfaces registers the interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterValidator{},
		&MsgUpdateValidatorStake{},
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

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
// This will be auto-generated from proto; hand-written stub for Phase 1.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_MsgService_serviceDesc, srv)
}

var _MsgService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.staking.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams:     []grpc.StreamDesc{},
	Metadata:    "legible/staking/v1/tx.proto",
}
