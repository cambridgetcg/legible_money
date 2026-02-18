package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the claiming_pot module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreatePot{}, "claiming_pot/CreatePot", nil)
	cdc.RegisterConcrete(&MsgFundPot{}, "claiming_pot/FundPot", nil)
	cdc.RegisterConcrete(&MsgClaimFromPot{}, "claiming_pot/ClaimFromPot", nil)
	cdc.RegisterConcrete(&MsgClosePot{}, "claiming_pot/ClosePot", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "claiming_pot/UpdateParams", nil)
}

// RegisterInterfaces registers the claiming_pot module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreatePot{},
		&MsgFundPot{},
		&MsgClaimFromPot{},
		&MsgClosePot{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// Query service descriptor, handlers, and RegisterQueryServer are now provided
// by the proto-generated query_grpc.pb.go file.
