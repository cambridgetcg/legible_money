package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitEvidence{}, "evidence_mgmt/SubmitEvidence", nil)
	cdc.RegisterConcrete(&MsgAuditEvidence{}, "evidence_mgmt/AuditEvidence", nil)
	cdc.RegisterConcrete(&MsgRegisterOracle{}, "evidence_mgmt/RegisterOracle", nil)
	cdc.RegisterConcrete(&MsgSubmitOracleData{}, "evidence_mgmt/SubmitOracleData", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "evidence_mgmt/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitEvidence{},
		&MsgAuditEvidence{},
		&MsgRegisterOracle{},
		&MsgSubmitOracleData{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
