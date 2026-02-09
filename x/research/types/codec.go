package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"

	"google.golang.org/grpc"
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
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.research.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "SubmitResearch", Handler: nil},
		{MethodName: "ChallengeResearch", Handler: nil},
		{MethodName: "ReviewResearch", Handler: nil},
		{MethodName: "ResolveResearch", Handler: nil},
		{MethodName: "CreateBounty", Handler: nil},
		{MethodName: "ClaimBounty", Handler: nil},
		{MethodName: "FulfillBounty", Handler: nil},
		{MethodName: "FundResearch", Handler: nil},
		{MethodName: "UpdateParams", Handler: nil},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/research/v1/tx.proto",
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.research.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Research", Handler: nil},
		{MethodName: "Submissions", Handler: nil},
		{MethodName: "Bounty", Handler: nil},
		{MethodName: "Params", Handler: nil},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/research/v1/query.proto",
}
