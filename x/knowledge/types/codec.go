package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/grpc"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitClaim{}, "knowledge/SubmitClaim", nil)
	cdc.RegisterConcrete(&MsgSubmitCommitment{}, "knowledge/SubmitCommitment", nil)
	cdc.RegisterConcrete(&MsgSubmitReveal{}, "knowledge/SubmitReveal", nil)
	cdc.RegisterConcrete(&MsgChallengeFact{}, "knowledge/ChallengeFact", nil)
	cdc.RegisterConcrete(&MsgAddFact{}, "knowledge/AddFact", nil)
	cdc.RegisterConcrete(&MsgProposeDomain{}, "knowledge/ProposeDomain", nil)
	cdc.RegisterConcrete(&MsgEndorseDomainProposal{}, "knowledge/EndorseDomainProposal", nil)
	cdc.RegisterConcrete(&MsgChallengeDomainProposal{}, "knowledge/ChallengeDomainProposal", nil)
	cdc.RegisterConcrete(&MsgRegisterStratum{}, "knowledge/RegisterStratum", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "knowledge/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgProposeDomain{},
		&MsgEndorseDomainProposal{},
		&MsgChallengeDomainProposal{},
		&MsgRegisterStratum{},
		&MsgUpdateParams{},
		&MsgSubmitClaim{},
		&MsgSubmitCommitment{},
		&MsgSubmitReveal{},
		&MsgChallengeFact{},
		&MsgAddFact{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
// This will be auto-generated from proto; hand-written stub for Phase 1.
// _Msg_serviceDesc is defined in types.go with full method descriptors.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}
