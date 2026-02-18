package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the ibcratelimit module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, "ibcratelimit/UpdateParams", nil)
}

// RegisterInterfaces registers the ibcratelimit module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// Msg service descriptor, handlers, and RegisterMsgServer are now provided
// by the proto-generated tx_grpc.pb.go file.

// Query service descriptor, handlers, and RegisterQueryServer are now provided
// by the proto-generated query_grpc.pb.go file.
