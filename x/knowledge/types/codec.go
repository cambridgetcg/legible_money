package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitClaim{}, "knowledge/SubmitClaim", nil)
	cdc.RegisterConcrete(&MsgSubmitCommitment{}, "knowledge/SubmitCommitment", nil)
	cdc.RegisterConcrete(&MsgSubmitReveal{}, "knowledge/SubmitReveal", nil)
	cdc.RegisterConcrete(&MsgChallengeFact{}, "knowledge/ChallengeFact", nil)
	cdc.RegisterConcrete(&MsgAddFact{}, "knowledge/AddFact", nil)
	cdc.RegisterConcrete(&MsgSubmitContradiction{}, "knowledge/SubmitContradiction", nil)
	cdc.RegisterConcrete(&MsgPatronizeFact{}, "knowledge/PatronizeFact", nil)
	cdc.RegisterConcrete(&MsgProposeDomain{}, "knowledge/ProposeDomain", nil)
	cdc.RegisterConcrete(&MsgEndorseDomainProposal{}, "knowledge/EndorseDomainProposal", nil)
	cdc.RegisterConcrete(&MsgChallengeDomainProposal{}, "knowledge/ChallengeDomainProposal", nil)
	cdc.RegisterConcrete(&MsgRegisterStratum{}, "knowledge/RegisterStratum", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "knowledge/UpdateParams", nil)
	cdc.RegisterConcrete(&MsgUpdateExtendedParams{}, "knowledge/UpdateExtendedParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		// PoT lifecycle (claim.pb.go)
		&MsgSubmitClaim{},
		&MsgSubmitCommitment{},
		&MsgSubmitReveal{},
		&MsgChallengeFact{},
		&MsgAddFact{},
		// Negative knowledge (counter_fact.pb.go)
		&MsgSubmitContradiction{},
		// Knowledge pruning (claim.pb.go)
		&MsgPatronizeFact{},
		// Domain management (tx.pb.go)
		&MsgProposeDomain{},
		&MsgEndorseDomainProposal{},
		&MsgChallengeDomainProposal{},
		&MsgRegisterStratum{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
