package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the module's message types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgProposeDomain{}, "ontology/ProposeDomain", nil)
	cdc.RegisterConcrete(&MsgVoteDomainProposal{}, "ontology/VoteDomainProposal", nil)
	cdc.RegisterConcrete(&MsgUpdateDomain{}, "ontology/UpdateDomain", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "ontology/UpdateParams", nil)
}

// RegisterInterfaces registers the module's interface types.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgProposeDomain{},
		&MsgVoteDomainProposal{},
		&MsgUpdateDomain{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
