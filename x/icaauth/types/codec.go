package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the module's message types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterInterchainAccount{}, "icaauth/RegisterInterchainAccount", nil)
	cdc.RegisterConcrete(&MsgSendICATransaction{}, "icaauth/SendICATransaction", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "icaauth/UpdateParams", nil)
}

// RegisterInterfaces registers the module's interface types.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterInterchainAccount{},
		&MsgSendICATransaction{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
