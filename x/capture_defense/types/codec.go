package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the capture defense module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRequestQualification{}, "capture_defense/RequestQualification", nil)
	cdc.RegisterConcrete(&MsgEndorseQualification{}, "capture_defense/EndorseQualification", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "capture_defense/UpdateParams", nil)
}

// RegisterInterfaces registers the module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRequestQualification{},
		&MsgEndorseQualification{},
		&MsgUpdateParams{},
	)
}
