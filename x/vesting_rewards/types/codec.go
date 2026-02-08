package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/grpc"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgClaimVestedRewards{}, "vesting_rewards/ClaimVestedRewards", nil)
	cdc.RegisterConcrete(&MsgFalsifyClaim{}, "vesting_rewards/FalsifyClaim", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgClaimVestedRewards{},
		&MsgFalsifyClaim{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
// _Msg_serviceDesc is defined below with method descriptors.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.vesting_rewards.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ClaimVestedRewards",
			Handler:    nil,
		},
		{
			MethodName: "FalsifyClaim",
			Handler:    nil,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/vesting_rewards/v1/tx.proto",
}
