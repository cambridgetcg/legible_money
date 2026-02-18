package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the governance module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitLIP{}, "lgm_gov/SubmitLIP", nil)
	cdc.RegisterConcrete(&MsgStakeLIP{}, "lgm_gov/StakeLIP", nil)
	cdc.RegisterConcrete(&MsgCastVote{}, "lgm_gov/CastVote", nil)
	cdc.RegisterConcrete(&MsgAdvanceLIPStage{}, "lgm_gov/AdvanceLIPStage", nil)
	cdc.RegisterConcrete(&MsgWithdrawLIP{}, "lgm_gov/WithdrawLIP", nil)
	cdc.RegisterConcrete(&MsgSubmitDisbursement{}, "lgm_gov/SubmitDisbursement", nil)
	cdc.RegisterConcrete(&MsgExecuteDisbursement{}, "lgm_gov/ExecuteDisbursement", nil)
	cdc.RegisterConcrete(&MsgCancelDisbursement{}, "lgm_gov/CancelDisbursement", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "lgm_gov/UpdateParams", nil)
	cdc.RegisterConcrete(&MsgAttachUpgradePlan{}, "lgm_gov/AttachUpgradePlan", nil)
}

// RegisterInterfaces registers the governance module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitLIP{},
		&MsgStakeLIP{},
		&MsgAmendLIP{},
		&MsgAdvanceLIPStage{},
		&MsgWithdrawLIP{},
		&MsgCastVote{},
		&MsgLockVote{},
		&MsgUnlockVote{},
		&MsgSwitchVote{},
		&MsgCommitReview{},
		&MsgRevealReview{},
		&MsgFinalizeReview{},
		&MsgSubmitDisbursement{},
		&MsgStakeDisbursement{},
		&MsgAdvanceDisbursement{},
		&MsgExecuteDisbursement{},
		&MsgCancelDisbursement{},
		&MsgRegisterOperator{},
		&MsgAddAgent{},
		&MsgRemoveAgent{},
		&MsgSlashOperator{},
		&MsgCreateDeployment{},
		&MsgAdvanceDeployment{},
		&MsgApproveDeployment{},
		&MsgRollbackDeployment{},
		// Hand-written (types.go) — have XXX_MessageName in proto_names.go
		&MsgUpdateParams{},
		&MsgAttachUpgradePlan{},
	)
}

// Query service registration is now handled by proto-generated query_grpc.pb.go.
