package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterCodec registers the tree module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	// Projects
	cdc.RegisterConcrete(&MsgCreateProject{}, "tree/CreateProject", nil)
	cdc.RegisterConcrete(&MsgProposeProject{}, "tree/ProposeProject", nil)
	cdc.RegisterConcrete(&MsgStartDevelopment{}, "tree/StartDevelopment", nil)
	cdc.RegisterConcrete(&MsgCompleteProject{}, "tree/CompleteProject", nil)
	cdc.RegisterConcrete(&MsgPauseProject{}, "tree/PauseProject", nil)
	cdc.RegisterConcrete(&MsgResumeProject{}, "tree/ResumeProject", nil)
	cdc.RegisterConcrete(&MsgAbandonProject{}, "tree/AbandonProject", nil)
	cdc.RegisterConcrete(&MsgSpawnChildProject{}, "tree/SpawnChildProject", nil)
	// Tasks
	cdc.RegisterConcrete(&MsgAddTask{}, "tree/AddTask", nil)
	cdc.RegisterConcrete(&MsgAssignTask{}, "tree/AssignTask", nil)
	cdc.RegisterConcrete(&MsgStartWork{}, "tree/StartWork", nil)
	cdc.RegisterConcrete(&MsgSubmitDeliverable{}, "tree/SubmitDeliverable", nil)
	cdc.RegisterConcrete(&MsgApproveDeliverable{}, "tree/ApproveDeliverable", nil)
	cdc.RegisterConcrete(&MsgRejectDeliverable{}, "tree/RejectDeliverable", nil)
	cdc.RegisterConcrete(&MsgReopenTask{}, "tree/ReopenTask", nil)
	// Applications
	cdc.RegisterConcrete(&MsgApplyToProject{}, "tree/ApplyToProject", nil)
	cdc.RegisterConcrete(&MsgReviewApplication{}, "tree/ReviewApplication", nil)
	cdc.RegisterConcrete(&MsgSetAvailability{}, "tree/SetAvailability", nil)
	// Services
	cdc.RegisterConcrete(&MsgDeployService{}, "tree/DeployService", nil)
	cdc.RegisterConcrete(&MsgPauseService{}, "tree/PauseService", nil)
	cdc.RegisterConcrete(&MsgResumeService{}, "tree/ResumeService", nil)
	cdc.RegisterConcrete(&MsgRetireService{}, "tree/RetireService", nil)
	cdc.RegisterConcrete(&MsgCallService{}, "tree/CallService", nil)
}

// RegisterInterfaces registers the tree module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
