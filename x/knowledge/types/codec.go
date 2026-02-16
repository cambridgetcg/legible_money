package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/grpc"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitClaim{}, "knowledge/SubmitClaim", nil)
	cdc.RegisterConcrete(&MsgSubmitCommitment{}, "knowledge/SubmitCommitment", nil)
	cdc.RegisterConcrete(&MsgSubmitReveal{}, "knowledge/SubmitReveal", nil)
	cdc.RegisterConcrete(&MsgChallengeFact{}, "knowledge/ChallengeFact", nil)
	cdc.RegisterConcrete(&MsgAddFact{}, "knowledge/AddFact", nil)
	cdc.RegisterConcrete(&MsgProposeDomain{}, "knowledge/ProposeDomain", nil)
	cdc.RegisterConcrete(&MsgEndorseDomainProposal{}, "knowledge/EndorseDomainProposal", nil)
	cdc.RegisterConcrete(&MsgChallengeDomainProposal{}, "knowledge/ChallengeDomainProposal", nil)
	cdc.RegisterConcrete(&MsgRegisterStratum{}, "knowledge/RegisterStratum", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "knowledge/UpdateParams", nil)
	cdc.RegisterConcrete(&MsgUpdateExtendedParams{}, "knowledge/UpdateExtendedParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		// claim.pb.go
		&MsgSubmitClaim{},
		&MsgSubmitCommitment{},
		&MsgSubmitReveal{},
		&MsgChallengeFact{},
		&MsgAddFact{},
		// tx.pb.go
		&MsgProposeDomain{},
		&MsgEndorseDomainProposal{},
		&MsgChallengeDomainProposal{},
		&MsgRegisterStratum{},
		&MsgUpdateParams{},
		// counter_fact.pb.go
		&MsgSubmitContradiction{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// ExtendedMsgServer extends the proto-generated MsgServer with additional
// message handlers for claim lifecycle and counter-fact methods not yet
// in the proto service definition.
type ExtendedMsgServer interface {
	MsgServer
	SubmitClaim(context.Context, *MsgSubmitClaim) (*MsgSubmitClaimResponse, error)
	SubmitCommitment(context.Context, *MsgSubmitCommitment) (*MsgSubmitCommitmentResponse, error)
	SubmitReveal(context.Context, *MsgSubmitReveal) (*MsgSubmitRevealResponse, error)
	ChallengeFact(context.Context, *MsgChallengeFact) (*MsgChallengeFactResponse, error)
	AddFact(context.Context, *MsgAddFact) (*MsgAddFactResponse, error)
	SubmitContradiction(context.Context, *MsgSubmitContradiction) (*MsgSubmitContradictionResponse, error)
	UpdateExtendedParams(context.Context, *MsgUpdateExtendedParams) (*MsgUpdateExtendedParamsResponse, error)
}

// RegisterExtendedMsgServer registers the extended msg server that includes
// both proto-generated handlers and supplemental handlers for claim/counter-fact methods.
func RegisterExtendedMsgServer(s grpc.ServiceRegistrar, srv ExtendedMsgServer) {
	// Register the proto-generated service (ProposeDomain, EndorseDomainProposal, etc.)
	RegisterMsgServer(s, srv)
	// Register extended handlers for non-proto message types
	s.RegisterService(&_ExtendedMsg_serviceDesc, srv)
}

// gRPC service descriptor for extended (non-proto) message handlers.
var _ExtendedMsg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.knowledge.v1.ExtendedMsg",
	HandlerType: (*ExtendedMsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "SubmitClaim", Handler: _Msg_SubmitClaim_Handler},
		{MethodName: "SubmitCommitment", Handler: _Msg_SubmitCommitment_Handler},
		{MethodName: "SubmitReveal", Handler: _Msg_SubmitReveal_Handler},
		{MethodName: "ChallengeFact", Handler: _Msg_ChallengeFact_Handler},
		{MethodName: "AddFact", Handler: _Msg_AddFact_Handler},
		{MethodName: "SubmitContradiction", Handler: _Msg_SubmitContradiction_Handler},
		{MethodName: "UpdateExtendedParams", Handler: _Msg_UpdateExtendedParams_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/knowledge/v1/tx.proto",
}

func _Msg_SubmitClaim_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitClaim)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).SubmitClaim(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.knowledge.v1.ExtendedMsg/SubmitClaim"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).SubmitClaim(ctx, req.(*MsgSubmitClaim))
	})
}

func _Msg_SubmitCommitment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitCommitment)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).SubmitCommitment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.knowledge.v1.ExtendedMsg/SubmitCommitment"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).SubmitCommitment(ctx, req.(*MsgSubmitCommitment))
	})
}

func _Msg_SubmitReveal_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitReveal)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).SubmitReveal(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.knowledge.v1.ExtendedMsg/SubmitReveal"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).SubmitReveal(ctx, req.(*MsgSubmitReveal))
	})
}

func _Msg_ChallengeFact_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgChallengeFact)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).ChallengeFact(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.knowledge.v1.ExtendedMsg/ChallengeFact"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).ChallengeFact(ctx, req.(*MsgChallengeFact))
	})
}

func _Msg_AddFact_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAddFact)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).AddFact(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.knowledge.v1.ExtendedMsg/AddFact"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).AddFact(ctx, req.(*MsgAddFact))
	})
}

func _Msg_SubmitContradiction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitContradiction)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).SubmitContradiction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.knowledge.v1.ExtendedMsg/SubmitContradiction"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).SubmitContradiction(ctx, req.(*MsgSubmitContradiction))
	})
}

func _Msg_UpdateExtendedParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateExtendedParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedMsgServer).UpdateExtendedParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.knowledge.v1.ExtendedMsg/UpdateExtendedParams"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedMsgServer).UpdateExtendedParams(ctx, req.(*MsgUpdateExtendedParams))
	})
}
