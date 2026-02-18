package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the channels module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgOpenChannel{}, "channels/OpenChannel", nil)
	cdc.RegisterConcrete(&MsgDepositChannel{}, "channels/DepositChannel", nil)
	cdc.RegisterConcrete(&MsgUpdateState{}, "channels/UpdateState", nil)
	cdc.RegisterConcrete(&MsgDisputeChannel{}, "channels/DisputeChannel", nil)
	cdc.RegisterConcrete(&MsgSettleChannel{}, "channels/SettleChannel", nil)
	cdc.RegisterConcrete(&MsgCloseChannel{}, "channels/CloseChannel", nil)
	cdc.RegisterConcrete(&MsgClaimExpired{}, "channels/ClaimExpired", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "channels/UpdateParams", nil)
}

// RegisterInterfaces registers the channels module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgOpenChannel{},
		&MsgUpdateChannel{},
		&MsgCloseChannel{},
		&MsgSettleChannel{},
		&MsgDisputeChannel{},
		&MsgDepositChannel{},
		&MsgUpdateState{},
		&MsgClaimExpired{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// All Msg service handlers are now provided by the proto-generated tx_grpc.pb.go file.
// Query service descriptor, handlers, and RegisterQueryServer are now provided
// by the proto-generated query_grpc.pb.go file.
