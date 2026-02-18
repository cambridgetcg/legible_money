package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the module's message types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateSchedule{}, "schedule/CreateSchedule", nil)
	cdc.RegisterConcrete(&MsgPauseSchedule{}, "schedule/PauseSchedule", nil)
	cdc.RegisterConcrete(&MsgResumeSchedule{}, "schedule/ResumeSchedule", nil)
	cdc.RegisterConcrete(&MsgCancelSchedule{}, "schedule/CancelSchedule", nil)
	cdc.RegisterConcrete(&MsgAddFee{}, "schedule/AddFee", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "schedule/UpdateParams", nil)
}

// RegisterInterfaces registers the module's interface types.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateSchedule{},
		&MsgPauseSchedule{},
		&MsgResumeSchedule{},
		&MsgCancelSchedule{},
		&MsgAddFee{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
