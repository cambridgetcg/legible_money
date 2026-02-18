package types

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc"
)

// --- Constants ---

const (
	// Challenge stake and rewards (in ulgm)
	MinChallengeStake           = "77000000"  // 77 LGM
	CaptureExposureBonus        = "777000000" // 777 LGM bonus
	ChallengeStakeMultiplierBPS = 7_700_000   // 7.7x in basis points (/ 1,000,000)
	FailedChallengePenaltyBPS   = 770_000     // 77%
	ValidatorSlashOnUpheldBPS   = 300_000     // 30%

	// Evidence
	MinEvidencePieces                  = 3
	MaxEvidencePieces                  = 50
	DefaultEvidencePeriodBlocks uint64 = 777
	DefaultReviewPeriodBlocks   uint64 = 7_777

	// Bounty pools
	BaseBountyPool             = "2222000000" // 2222 LGM
	BountyReplenishmentRateBPS = 10_000       // 1% of verification rewards

	// Domain recovery
	DefaultDomainPauseDurationBlocks uint64 = 777
)

// --- Enums ---

// ChallengeStatus represents the lifecycle of a capture challenge.
type ChallengeStatus string

const (
	ChallengeSubmitted      ChallengeStatus = "submitted"
	ChallengeEvidencePeriod ChallengeStatus = "evidence_period"
	ChallengeUnderReview    ChallengeStatus = "under_review"
	ChallengeUpheld         ChallengeStatus = "upheld"
	ChallengeRejected       ChallengeStatus = "rejected"
	ChallengePartial        ChallengeStatus = "partial"
	ChallengeExpired        ChallengeStatus = "expired"
)

// EvidenceType categorizes the kind of capture evidence.
type EvidenceType string

const (
	EvidenceCoordinatedVoting    EvidenceType = "coordinated_voting"
	EvidenceTimingCorrelation    EvidenceType = "timing_correlation"
	EvidenceOffChainCoordination EvidenceType = "off_chain_coordination"
	EvidenceFalseFactAccepted    EvidenceType = "false_fact_accepted"
	EvidenceStakeManipulation    EvidenceType = "stake_manipulation"
	EvidenceQualificationFraud   EvidenceType = "qualification_fraud"
	EvidenceSystematicBias       EvidenceType = "systematic_bias"
)

// --- Params Validation ---
// Params struct is defined in genesis.pb.go (proto-generated).

// Validate performs basic param validation.
func (p Params) Validate() error {
	if p.EvidencePeriodBlocks == 0 {
		return fmt.Errorf("evidence period blocks must be positive")
	}
	if p.ReviewPeriodBlocks == 0 {
		return fmt.Errorf("review period blocks must be positive")
	}
	return nil
}

// --- Message Types ---

// MsgSubmitChallenge submits a new capture challenge.
type MsgSubmitChallenge struct {
	Challenger        string             `json:"challenger"`
	Domain            string             `json:"domain"`
	Evidence          []*CaptureEvidence `json:"evidence"`
	Stake             string             `json:"stake"`
	AccusedValidators []string           `json:"accused_validators"`
}

func (m *MsgSubmitChallenge) Reset()         { *m = MsgSubmitChallenge{} }
func (m *MsgSubmitChallenge) String() string { return proto.CompactTextString(m) }
func (m *MsgSubmitChallenge) ProtoMessage()  {}

func (m *MsgSubmitChallenge) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if len(m.Evidence) < MinEvidencePieces {
		return ErrInsufficientEvidence
	}
	return nil
}

func (m *MsgSubmitChallenge) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Challenger)
	return []sdk.AccAddress{addr}
}

// MsgSubmitChallengeResponse is the response.
type MsgSubmitChallengeResponse struct {
	ChallengeId string `json:"challenge_id"`
}

func (m *MsgSubmitChallengeResponse) Reset()         { *m = MsgSubmitChallengeResponse{} }
func (m *MsgSubmitChallengeResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgSubmitChallengeResponse) ProtoMessage()  {}

// MsgAddEvidence adds additional evidence to an active challenge.
type MsgAddEvidence struct {
	Challenger  string           `json:"challenger"`
	ChallengeId string           `json:"challenge_id"`
	Evidence    *CaptureEvidence `json:"evidence"`
}

func (m *MsgAddEvidence) Reset()         { *m = MsgAddEvidence{} }
func (m *MsgAddEvidence) String() string { return proto.CompactTextString(m) }
func (m *MsgAddEvidence) ProtoMessage()  {}

func (m *MsgAddEvidence) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if m.ChallengeId == "" {
		return fmt.Errorf("challenge_id is required")
	}
	return nil
}

func (m *MsgAddEvidence) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Challenger)
	return []sdk.AccAddress{addr}
}

// MsgAddEvidenceResponse is the response.
type MsgAddEvidenceResponse struct {
	EvidenceCount int `json:"evidence_count"`
}

func (m *MsgAddEvidenceResponse) Reset()         { *m = MsgAddEvidenceResponse{} }
func (m *MsgAddEvidenceResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgAddEvidenceResponse) ProtoMessage()  {}

// MsgResolveChallenge resolves a challenge (governance or auto).
type MsgResolveChallenge struct {
	Resolver      string `json:"resolver"` // authority address
	ChallengeId   string `json:"challenge_id"`
	Outcome       string `json:"outcome"` // "upheld", "rejected", "partial"
	Justification string `json:"justification"`
}

func (m *MsgResolveChallenge) Reset()         { *m = MsgResolveChallenge{} }
func (m *MsgResolveChallenge) String() string { return proto.CompactTextString(m) }
func (m *MsgResolveChallenge) ProtoMessage()  {}

func (m *MsgResolveChallenge) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Resolver); err != nil {
		return fmt.Errorf("invalid resolver address: %w", err)
	}
	if m.ChallengeId == "" {
		return fmt.Errorf("challenge_id is required")
	}
	switch m.Outcome {
	case "upheld", "rejected", "partial":
	default:
		return fmt.Errorf("invalid outcome: %s", m.Outcome)
	}
	return nil
}

func (m *MsgResolveChallenge) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Resolver)
	return []sdk.AccAddress{addr}
}

// MsgResolveChallengeResponse is the response.
type MsgResolveChallengeResponse struct {
	ChallengerReward string `json:"challenger_reward"`
}

func (m *MsgResolveChallengeResponse) Reset()         { *m = MsgResolveChallengeResponse{} }
func (m *MsgResolveChallengeResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgResolveChallengeResponse) ProtoMessage()  {}

// --- Service Interfaces ---

// MsgServer defines the capture challenge module's message server interface.
type MsgServer interface {
	SubmitChallenge(context.Context, *MsgSubmitChallenge) (*MsgSubmitChallengeResponse, error)
	AddEvidence(context.Context, *MsgAddEvidence) (*MsgAddEvidenceResponse, error)
	ResolveChallenge(context.Context, *MsgResolveChallenge) (*MsgResolveChallengeResponse, error)
}

// Query types (QueryServer, QueryChallengeRequest/Response, etc.) are now provided
// by the proto-generated query.pb.go and query_grpc.pb.go files.

// --- Codec Registration ---

// RegisterCodec registers the capture challenge module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitChallenge{}, "capture_challenge/SubmitChallenge", nil)
	cdc.RegisterConcrete(&MsgAddEvidence{}, "capture_challenge/AddEvidence", nil)
	cdc.RegisterConcrete(&MsgResolveChallenge{}, "capture_challenge/ResolveChallenge", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "capture_challenge/UpdateParams", nil)
}

// RegisterInterfaces registers the module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

// gRPC service descriptors for msg server
var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.capture_challenge.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "SubmitChallenge", Handler: _Msg_SubmitChallenge_Handler},
		{MethodName: "AddEvidence", Handler: _Msg_AddEvidence_Handler},
		{MethodName: "ResolveChallenge", Handler: _Msg_ResolveChallenge_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Msg_SubmitChallenge_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitChallenge)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SubmitChallenge(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.capture_challenge.v1.Msg/SubmitChallenge"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SubmitChallenge(ctx, req.(*MsgSubmitChallenge))
	})
}

func _Msg_AddEvidence_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAddEvidence)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AddEvidence(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.capture_challenge.v1.Msg/AddEvidence"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AddEvidence(ctx, req.(*MsgAddEvidence))
	})
}

func _Msg_ResolveChallenge_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgResolveChallenge)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ResolveChallenge(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.capture_challenge.v1.Msg/ResolveChallenge"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ResolveChallenge(ctx, req.(*MsgResolveChallenge))
	})
}

// RegisterMsgServer registers the capture challenge msg server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// Query gRPC service descriptor, handlers, and RegisterQueryServer are now provided
// by the proto-generated query_grpc.pb.go file.


// MsgUpdateParams is the governance message to update module params.
type MsgUpdateParams struct {
	Authority string  `json:"authority"`
	Params    *Params `json:"params"`
}

// MsgUpdateParamsResponse is the response type for MsgUpdateParams.
type MsgUpdateParamsResponse struct{}

func (*MsgUpdateParams) ProtoMessage()             {}
func (*MsgUpdateParams) Reset()                     {}
func (*MsgUpdateParams) String() string             { return "MsgUpdateParams" }
func (*MsgUpdateParamsResponse) ProtoMessage()      {}
func (*MsgUpdateParamsResponse) Reset()             {}
func (*MsgUpdateParamsResponse) String() string     { return "MsgUpdateParamsResponse" }

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return msg.Params.Validate()
}
