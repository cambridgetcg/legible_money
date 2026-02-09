package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"

	"google.golang.org/grpc"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitEvidence{}, "evidence_mgmt/SubmitEvidence", nil)
	cdc.RegisterConcrete(&MsgAuditEvidence{}, "evidence_mgmt/AuditEvidence", nil)
	cdc.RegisterConcrete(&MsgRegisterOracle{}, "evidence_mgmt/RegisterOracle", nil)
	cdc.RegisterConcrete(&MsgSubmitOracleData{}, "evidence_mgmt/SubmitOracleData", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "evidence_mgmt/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}
