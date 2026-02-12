package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
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
	// Proto-generated types handle their own registration.
	// Genesis and keeper use json.Marshal — no proto codec needed.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
