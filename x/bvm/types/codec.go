package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the bvm module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgDeployContract{}, "bvm/DeployContract", nil)
	cdc.RegisterConcrete(&MsgCallContract{}, "bvm/CallContract", nil)
	cdc.RegisterConcrete(&MsgScheduleContract{}, "bvm/ScheduleContract", nil)
	cdc.RegisterConcrete(&MsgCancelSchedule{}, "bvm/CancelSchedule", nil)
	cdc.RegisterConcrete(&MsgUpdateContractState{}, "bvm/UpdateContractState", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "bvm/UpdateParams", nil)
}

// RegisterInterfaces registers the bvm module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		// Proto-generated (tx.pb.go)
		&MsgDeployContract{},
		&MsgCallContract{},
		&MsgScheduleExecution{},
		&MsgUpdateParams{},
		// Hand-written (types.go) — have XXX_MessageName in proto_names.go
		&MsgScheduleContract{},
		&MsgCancelSchedule{},
		&MsgUpdateContractState{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// Note: RegisterQueryServer, QueryServer, and query handlers are now in query_grpc.pb.go (proto-generated).
