package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgTransfer{}, "tokens/Transfer", nil)
	cdc.RegisterConcrete(&MsgApproveAllowance{}, "tokens/ApproveAllowance", nil)
	cdc.RegisterConcrete(&MsgBurn{}, "tokens/Burn", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "tokens/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
