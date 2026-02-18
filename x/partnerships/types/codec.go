package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	// Cold-start
	cdc.RegisterConcrete(&MsgInitiateSeedPartnership{}, "partnerships/InitiateSeedPartnership", nil)
	cdc.RegisterConcrete(&MsgGraduateSeedPartnership{}, "partnerships/GraduateSeedPartnership", nil)
	cdc.RegisterConcrete(&MsgRegisterInPool{}, "partnerships/RegisterInPool", nil)
	cdc.RegisterConcrete(&MsgDeclareIntent{}, "partnerships/DeclareIntent", nil)
	cdc.RegisterConcrete(&MsgSponsorPartnership{}, "partnerships/SponsorPartnership", nil)
	// Core
	cdc.RegisterConcrete(&MsgInitiatePartnership{}, "partnerships/InitiatePartnership", nil)
	cdc.RegisterConcrete(&MsgAcceptPartnership{}, "partnerships/AcceptPartnership", nil)
	cdc.RegisterConcrete(&MsgDepositToPot{}, "partnerships/DepositToPot", nil)
	cdc.RegisterConcrete(&MsgDistributeReward{}, "partnerships/DistributeReward", nil)
	// Deliberation
	cdc.RegisterConcrete(&MsgProposeOperation{}, "partnerships/ProposeOperation", nil)
	cdc.RegisterConcrete(&MsgApproveOperation{}, "partnerships/ApproveOperation", nil)
	cdc.RegisterConcrete(&MsgRejectOperation{}, "partnerships/RejectOperation", nil)
	// Safety & exit
	cdc.RegisterConcrete(&MsgSafetyFreeze{}, "partnerships/SafetyFreeze", nil)
	cdc.RegisterConcrete(&MsgRaiseCoercionSignal{}, "partnerships/RaiseCoercionSignal", nil)
	cdc.RegisterConcrete(&MsgInitiateExit{}, "partnerships/InitiateExit", nil)
	// Governance
	cdc.RegisterConcrete(&MsgUpdateParams{}, "partnerships/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		// Cold-start
		&MsgInitiateSeedPartnership{},
		&MsgGraduateSeedPartnership{},
		&MsgRegisterInPool{},
		&MsgDeclareIntent{},
		&MsgSponsorPartnership{},
		// Core
		&MsgInitiatePartnership{},
		&MsgAcceptPartnership{},
		&MsgDepositToPot{},
		&MsgDistributeReward{},
		// Deliberation
		&MsgProposeOperation{},
		&MsgApproveOperation{},
		&MsgRejectOperation{},
		// Safety & exit
		&MsgSafetyFreeze{},
		&MsgRaiseCoercionSignal{},
		&MsgInitiateExit{},
		// Governance
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
