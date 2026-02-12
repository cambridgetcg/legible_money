package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterCodec registers the module's types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgOpenDispute{}, "disputes/OpenDispute", nil)
	cdc.RegisterConcrete(&MsgCommitEvidence{}, "disputes/CommitEvidence", nil)
	cdc.RegisterConcrete(&MsgSubmitEvidence{}, "disputes/SubmitEvidence", nil)
	cdc.RegisterConcrete(&MsgVoteDispute{}, "disputes/VoteDispute", nil)
	cdc.RegisterConcrete(&MsgEscalateDispute{}, "disputes/EscalateDispute", nil)
	cdc.RegisterConcrete(&MsgSettleDispute{}, "disputes/SettleDispute", nil)
	cdc.RegisterConcrete(&MsgTimeoutDispute{}, "disputes/TimeoutDispute", nil)
}

// RegisterInterfaces registers the module's interface types with the interface registry.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// RegisterQueryServer is defined in query.go (re-export not needed).
