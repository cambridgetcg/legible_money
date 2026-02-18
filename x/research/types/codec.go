package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitResearch{}, "research/SubmitResearch", nil)
	cdc.RegisterConcrete(&MsgChallengeResearch{}, "research/ChallengeResearch", nil)
	cdc.RegisterConcrete(&MsgReviewResearch{}, "research/ReviewResearch", nil)
	cdc.RegisterConcrete(&MsgResolveResearch{}, "research/ResolveResearch", nil)
	cdc.RegisterConcrete(&MsgCreateBounty{}, "research/CreateBounty", nil)
	cdc.RegisterConcrete(&MsgClaimBounty{}, "research/ClaimBounty", nil)
	cdc.RegisterConcrete(&MsgFulfillBounty{}, "research/FulfillBounty", nil)
	cdc.RegisterConcrete(&MsgFundResearch{}, "research/FundResearch", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "research/UpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitResearch{},
		&MsgChallengeResearch{},
		&MsgReviewResearch{},
		&MsgResolveResearch{},
		&MsgCreateBounty{},
		&MsgClaimBounty{},
		&MsgFulfillBounty{},
		&MsgFundResearch{},
		&MsgUpdateParams{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// Note: RegisterMsgServer, _Msg_serviceDesc, and all _Msg_*_Handler functions
// are now defined in tx_grpc.pb.go (proto-generated).

// Note: RegisterQueryServer, _Query_serviceDesc, Query_ServiceDesc, and all
// _Query_*_Handler functions are now defined in query_grpc.pb.go (proto-generated).
