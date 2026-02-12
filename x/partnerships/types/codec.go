package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgInitiatePartnership{}, "partnerships/InitiatePartnership", nil)
	cdc.RegisterConcrete(&MsgAcceptPartnership{}, "partnerships/AcceptPartnership", nil)
	cdc.RegisterConcrete(&MsgDepositToPot{}, "partnerships/DepositToPot", nil)
	cdc.RegisterConcrete(&MsgDistributeReward{}, "partnerships/DistributeReward", nil)
	cdc.RegisterConcrete(&MsgProposeOperation{}, "partnerships/ProposeOperation", nil)
	cdc.RegisterConcrete(&MsgApproveOperation{}, "partnerships/ApproveOperation", nil)
	cdc.RegisterConcrete(&MsgRejectOperation{}, "partnerships/RejectOperation", nil)
	cdc.RegisterConcrete(&MsgSafetyFreeze{}, "partnerships/SafetyFreeze", nil)
	cdc.RegisterConcrete(&MsgRaiseCoercionSignal{}, "partnerships/RaiseCoercionSignal", nil)
	cdc.RegisterConcrete(&MsgInitiateExit{}, "partnerships/InitiateExit", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "partnerships/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
