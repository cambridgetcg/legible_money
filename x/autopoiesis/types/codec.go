package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the module's types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, "autopoiesis/UpdateParams", nil)
	cdc.RegisterConcrete(&MsgActivateAutopoiesis{}, "autopoiesis/Activate", nil)
	cdc.RegisterConcrete(&MsgOverrideMultiplier{}, "autopoiesis/OverrideMultiplier", nil)
	cdc.RegisterConcrete(&MsgFreezeMultiplier{}, "autopoiesis/FreezeMultiplier", nil)
}

// RegisterInterfaces registers the module's interface types with the interface registry.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgActivateAutopoiesis{},
		&MsgOverrideMultiplier{},
		&MsgFreezeMultiplier{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
