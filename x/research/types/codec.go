package types

import (
	"context"

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

// Note: RegisterMsgServer, _Msg_serviceDesc, and all _Msg_*_Handler functions
// are now defined in tx_grpc.pb.go (proto-generated).

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}

// ---------- gRPC Query Service Descriptor ----------

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.research.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Research", Handler: _Query_Research_Handler},
		{MethodName: "Submissions", Handler: _Query_Submissions_Handler},
		{MethodName: "Bounty", Handler: _Query_Bounty_Handler},
		{MethodName: "Params", Handler: _Query_Params_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/research/v1/query.proto",
}

func _Query_Research_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryResearchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Research(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.research.v1.Query/Research"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Research(ctx, req.(*QueryResearchRequest))
	})
}

func _Query_Submissions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuerySubmissionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Submissions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.research.v1.Query/Submissions"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Submissions(ctx, req.(*QuerySubmissionsRequest))
	})
}

func _Query_Bounty_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryBountyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Bounty(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.research.v1.Query/Bounty"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Bounty(ctx, req.(*QueryBountyRequest))
	})
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.research.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}
