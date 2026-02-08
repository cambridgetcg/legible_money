package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/grpc"
)

// RegisterCodec registers the module's types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgOpenDispute{}, "disputes/OpenDispute", nil)
	cdc.RegisterConcrete(&MsgSubmitEvidence{}, "disputes/SubmitEvidence", nil)
	cdc.RegisterConcrete(&MsgVoteDispute{}, "disputes/VoteDispute", nil)
	cdc.RegisterConcrete(&MsgEscalateDispute{}, "disputes/EscalateDispute", nil)
}

// RegisterInterfaces registers the module's interface types with the interface registry.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgOpenDispute{},
		&MsgSubmitEvidence{},
		&MsgVoteDispute{},
		&MsgEscalateDispute{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}
