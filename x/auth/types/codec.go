package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the legible_auth module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterAccount{}, "legible_auth/RegisterAccount", nil)
	cdc.RegisterConcrete(&MsgRotateKey{}, "legible_auth/RotateKey", nil)
	cdc.RegisterConcrete(&MsgCreateSession{}, "legible_auth/CreateSession", nil)
	cdc.RegisterConcrete(&MsgRevokeSession{}, "legible_auth/RevokeSession", nil)
	cdc.RegisterConcrete(&MsgFreezeAccount{}, "legible_auth/FreezeAccount", nil)
	cdc.RegisterConcrete(&MsgUnfreezeAccount{}, "legible_auth/UnfreezeAccount", nil)
	cdc.RegisterConcrete(&MsgSetRecoveryConfig{}, "legible_auth/SetRecoveryConfig", nil)
	cdc.RegisterConcrete(&MsgInitiateRecovery{}, "legible_auth/InitiateRecovery", nil)
	cdc.RegisterConcrete(&MsgSubmitRecoveryShard{}, "legible_auth/SubmitRecoveryShard", nil)
	cdc.RegisterConcrete(&MsgChallengeRecovery{}, "legible_auth/ChallengeRecovery", nil)
	cdc.RegisterConcrete(&MsgExecuteRecovery{}, "legible_auth/ExecuteRecovery", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "legible_auth/UpdateParams", nil)
}

// RegisterInterfaces registers the legible_auth module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRotateKey{},
		&MsgCreateSession{},
		&MsgRevokeSession{},
		&MsgUpdateParams{},
		&MsgRegisterAccount{},
		&MsgFreezeAccount{},
		&MsgUnfreezeAccount{},
		&MsgSetRecoveryConfig{},
		&MsgInitiateRecovery{},
		&MsgSubmitRecoveryShard{},
		&MsgChallengeRecovery{},
		&MsgExecuteRecovery{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

