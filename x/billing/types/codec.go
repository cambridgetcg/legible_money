package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterProvider{}, "billing/RegisterProvider", nil)
	cdc.RegisterConcrete(&MsgDeregisterProvider{}, "billing/DeregisterProvider", nil)
	cdc.RegisterConcrete(&MsgRequestQuote{}, "billing/RequestQuote", nil)
	cdc.RegisterConcrete(&MsgExecutePayment{}, "billing/ExecutePayment", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "billing/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
