package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterCodec registers the module's message types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterAgent{}, "discovery/RegisterAgent", nil)
	cdc.RegisterConcrete(&MsgUpdateProfile{}, "discovery/UpdateProfile", nil)
	cdc.RegisterConcrete(&MsgAddCapability{}, "discovery/AddCapability", nil)
	cdc.RegisterConcrete(&MsgRemoveCapability{}, "discovery/RemoveCapability", nil)
	cdc.RegisterConcrete(&MsgDeregisterAgent{}, "discovery/DeregisterAgent", nil)
	cdc.RegisterConcrete(&MsgHeartbeat{}, "discovery/Heartbeat", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "discovery/UpdateParams", nil)
}

// RegisterInterfaces registers the module's interface types.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
