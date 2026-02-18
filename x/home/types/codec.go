package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the home module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateHome{}, "home/CreateHome", nil)
	cdc.RegisterConcrete(&MsgUpdateHome{}, "home/UpdateHome", nil)
	cdc.RegisterConcrete(&MsgUpdateMemoryCID{}, "home/UpdateMemoryCID", nil)
	cdc.RegisterConcrete(&MsgStartSession{}, "home/StartSession", nil)
	cdc.RegisterConcrete(&MsgEndSession{}, "home/EndSession", nil)
	cdc.RegisterConcrete(&MsgRegisterKey{}, "home/RegisterKey", nil)
	cdc.RegisterConcrete(&MsgRevokeKey{}, "home/RevokeKey", nil)
	cdc.RegisterConcrete(&MsgConfigureGuardian{}, "home/ConfigureGuardian", nil)
	cdc.RegisterConcrete(&MsgAcknowledgeAlert{}, "home/AcknowledgeAlert", nil)
	cdc.RegisterConcrete(&MsgSetSpendingLimit{}, "home/SetSpendingLimit", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "home/UpdateParams", nil)
}

// RegisterInterfaces registers the home module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateHome{},
		&MsgUpdateHome{},
		&MsgUpdateMemoryCID{},
		&MsgStartSession{},
		&MsgEndSession{},
		&MsgRegisterKey{},
		&MsgRevokeKey{},
		&MsgConfigureGuardian{},
		&MsgAcknowledgeAlert{},
		&MsgSetSpendingLimit{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// NOTE: Query service descriptor, handlers, and RegisterQueryServer
// are now proto-generated in query_grpc.pb.go.
