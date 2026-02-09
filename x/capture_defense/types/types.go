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

// --- Enums ---

// QualificationPathway represents how a validator became qualified for a domain.
type QualificationPathway string

const (
	PathwayStakeCommitment   QualificationPathway = "stake_commitment"
	PathwayTrackRecord       QualificationPathway = "track_record"
	PathwayCrossReference    QualificationPathway = "cross_reference"
	PathwayStratumInheritance QualificationPathway = "stratum_inheritance"
	PathwayGenesis           QualificationPathway = "genesis"
)

// QualificationStatus represents the current status of a domain qualification.
type QualificationStatus string

const (
	StatusActive       QualificationStatus = "active"
	StatusProbationary QualificationStatus = "probationary"
	StatusSuspended    QualificationStatus = "suspended"
	StatusRevoked      QualificationStatus = "revoked"
	StatusExpired      QualificationStatus = "expired"
)

// ReputationSource indicates which reputation layer was used.
type ReputationSource string

const (
	SourceDomain  ReputationSource = "domain"
	SourceStratum ReputationSource = "stratum"
	SourceGlobal  ReputationSource = "global"
)

// --- Configuration Constants ---
// All floating-point values stored as uint64 basis points (1,000,000 = 100%).

const (
	// Stake requirements (in ulgm = micro-LGM)
	StakeLeafDomain  = "22000000"  // 22 LGM for leaf domains (depth > 2)
	StakeMidDomain   = "222000000" // 222 LGM for mid-level (depth 2)
	StakeTopDomain   = "2222000000" // 2222 LGM for top-level (depth 1)

	// Lock and timing (in blocks)
	StakeLockPeriod             uint64 = 7_777
	MinVerificationsForQual     uint64 = 19
	MinAccuracyForQual          uint64 = 770_000 // 77% in basis points
	MinEndorsersForQual         uint64 = 3
	EndorsementDecayBlocks      uint64 = 77_777
	InactivityExpirationBlocks  uint64 = 777_777
	ProbationaryPeriodBlocks    uint64 = 777
	MaxQualificationDepth       uint64 = 7

	// Weights (basis points, 1,000,000 = 100%)
	ProbationaryWeight          uint64 = 500_000  // 50%
	StratumInheritanceDiscount  uint64 = 700_000  // 70%
	AccuracyThresholdSuspension uint64 = 500_000  // 50%
	ConsecutiveIncorrectSuspend uint64 = 5

	// Reputation config
	ReputationHalfLifeBlocks uint64 = 77_777
	BaseScore                uint64 = 500
	MaxScore                 uint64 = 1000
	MinDecayedScore          uint64 = 300
	GlobalReputationDiscount uint64 = 500_000  // 50%
	StratumReputationDiscount uint64 = 700_000 // 70%
	NewDomainDiscount        uint64 = 300_000  // 30%
	MinVerificationsForDomainRep uint64 = 19

	// Score component weights (must sum to 1,000,000)
	AccuracyWeight         uint64 = 500_000 // 50%
	ConsistencyWeight      uint64 = 250_000 // 25%
	ChallengeDefenseWeight uint64 = 150_000 // 15%
	CalibrationWeight      uint64 = 100_000 // 10%

	// Capture detection thresholds
	HerfindahlTriggerThreshold uint64 = 400_000 // 0.4
	HerfindahlPauseThreshold   uint64 = 600_000 // 0.6
	MinVerificationsForAnalysis uint64 = 19
	DetectionIntervalBlocks     uint64 = 777
	DecayIntervalBlocks         uint64 = 7_777
	MaxExpirationsPerBlock      uint64 = 100

	// Cross-stratum
	CrossStratumAgreementThreshold uint64 = 700_000 // 70%
	LowerStratumVetoWeight         uint64 = 300_000 // 30%
	PartialCoverageConfPenalty     uint64 = 200_000 // 20%
	MaxStratumDistance              uint64 = 2
)

// --- Core Types ---

// QualificationMetrics tracks verification performance per domain.
type QualificationMetrics struct {
	TotalVerifications     uint64 `json:"total_verifications"`
	CorrectVerifications   uint64 `json:"correct_verifications"`
	Accuracy               uint64 `json:"accuracy"`                // basis points
	LastVerificationBlock  uint64 `json:"last_verification_block"`
	ConsecutiveCorrect     uint64 `json:"consecutive_correct"`
	ConsecutiveIncorrect   uint64 `json:"consecutive_incorrect"`
	ChallengesReceived     uint64 `json:"challenges_received"`
	ChallengesLost         uint64 `json:"challenges_lost"`
	EffectiveWeight        uint64 `json:"effective_weight"` // basis points
}

// DomainQualification represents a validator's qualification for a specific domain.
type DomainQualification struct {
	ValidatorAddr       string               `json:"validator_addr"`
	Domain              string               `json:"domain"`
	Stratum             uint32               `json:"stratum"`
	Pathway             QualificationPathway `json:"pathway"`
	Status              QualificationStatus  `json:"status"`
	EarnedAtBlock       uint64               `json:"earned_at_block"`
	ExpiresAtBlock      uint64               `json:"expires_at_block"` // 0 = no expiry
	Metrics             QualificationMetrics `json:"metrics"`
	LockedStake         string               `json:"locked_stake,omitempty"`
	StakeUnlocksAtBlock uint64               `json:"stake_unlocks_at_block,omitempty"`
	Endorsers           []string             `json:"endorsers,omitempty"`
	InheritedFrom       string               `json:"inherited_from,omitempty"`
	InheritanceDiscount uint64               `json:"inheritance_discount"` // basis points
}

func (q *DomainQualification) Reset()         { *q = DomainQualification{} }
func (q *DomainQualification) String() string { return proto.CompactTextString(q) }
func (q *DomainQualification) ProtoMessage()  {}

// GlobalReputation tracks a validator's overall reputation.
type GlobalReputation struct {
	Score              uint64 `json:"score"` // 0-1000
	TotalVerifications uint64 `json:"total_verifications"`
	CorrectVerifications uint64 `json:"correct_verifications"`
	Accuracy           uint64 `json:"accuracy"` // basis points
}

// StratumReputation tracks a validator's reputation within a stratum.
type StratumReputation struct {
	Stratum            uint32 `json:"stratum"`
	Score              uint64 `json:"score"` // 0-1000
	TotalVerifications uint64 `json:"total_verifications"`
	CorrectVerifications uint64 `json:"correct_verifications"`
	Accuracy           uint64 `json:"accuracy"` // basis points
	DomainsVerified    uint64 `json:"domains_verified"`
	FirstVerificationBlock uint64 `json:"first_verification_block"`
	LastActivityBlock  uint64 `json:"last_activity_block"`
	LastDecayBlock     uint64 `json:"last_decay_block"`
}

// DomainReputation tracks a validator's reputation within a specific domain.
type DomainReputation struct {
	Domain             string `json:"domain"`
	Stratum            uint32 `json:"stratum"`
	Score              uint64 `json:"score"` // 0-1000
	TotalVerifications uint64 `json:"total_verifications"`
	CorrectVerifications uint64 `json:"correct_verifications"`
	Accuracy           uint64 `json:"accuracy"` // basis points
	VerdictConsistency uint64 `json:"verdict_consistency"` // basis points
	ConfidenceCalibration uint64 `json:"confidence_calibration"` // basis points
	ChallengesReceived uint64 `json:"challenges_received"`
	ChallengesLost     uint64 `json:"challenges_lost"`
	ChallengeDefenseRate uint64 `json:"challenge_defense_rate"` // basis points
	ScoreConfidence    uint64 `json:"score_confidence"` // basis points, increases with verifications
	FirstVerificationBlock uint64 `json:"first_verification_block"`
	LastVerificationBlock  uint64 `json:"last_verification_block"`
	DecayedScore       uint64 `json:"decayed_score"`
	LastDecayBlock     uint64 `json:"last_decay_block"`
}

// IsolatedReputation holds all reputation data for a single validator.
type IsolatedReputation struct {
	ValidatorAddr     string                       `json:"validator_addr"`
	GlobalReputation  GlobalReputation             `json:"global_reputation"`
	StratumReputations map[uint32]StratumReputation `json:"stratum_reputations"`
	DomainReputations map[string]DomainReputation  `json:"domain_reputations"`
	CreatedAtBlock    uint64                       `json:"created_at_block"`
	LastUpdatedBlock  uint64                       `json:"last_updated_block"`
}

func (r *IsolatedReputation) Reset()         { *r = IsolatedReputation{} }
func (r *IsolatedReputation) String() string { return proto.CompactTextString(r) }
func (r *IsolatedReputation) ProtoMessage()  {}

// CaptureMetrics holds domain-level capture risk metrics.
type CaptureMetrics struct {
	Domain             string `json:"domain"`
	HerfindahlIndex    uint64 `json:"herfindahl_index"` // basis points
	TimingCorrelation  uint64 `json:"timing_correlation"` // basis points
	VerdictCorrelation uint64 `json:"verdict_correlation"` // basis points
	UniqueValidators   uint64 `json:"unique_validators"`
	TotalVerifications uint64 `json:"total_verifications"`
	Top3ValidatorShare uint64 `json:"top3_validator_share"` // basis points
	AnalyzedAtBlock    uint64 `json:"analyzed_at_block"`
	RiskScore          uint64 `json:"risk_score"` // basis points (0-1,000,000)
}

func (m *CaptureMetrics) Reset()         { *m = CaptureMetrics{} }
func (m *CaptureMetrics) String() string { return proto.CompactTextString(m) }
func (m *CaptureMetrics) ProtoMessage()  {}

// VerificationHistoryEntry records a single verification round for capture analysis.
type VerificationHistoryEntry struct {
	ClaimId              string   `json:"claim_id"`
	BlockNumber          uint64   `json:"block_number"`
	Validators           []string `json:"validators"`
	Verdicts             []string `json:"verdicts"` // "confirmed", "rejected", "uncertain"
	FinalOutcome         string   `json:"final_outcome"` // "accepted", "rejected", "contested"
	SubmissionTimestamps []uint64 `json:"submission_timestamps"`
}

// VerificationHistory holds the recent verification history for a domain.
type VerificationHistory struct {
	Domain  string                     `json:"domain"`
	Entries []VerificationHistoryEntry `json:"entries"`
}

// CrossStratumRequirement defines what strata must be covered for a claim stratum.
type CrossStratumRequirement struct {
	ClaimStratum              uint32            `json:"claim_stratum"`
	RequiredStrata            []uint32          `json:"required_strata"`
	MinValidatorsPerStratum   map[uint32]uint32 `json:"min_validators_per_stratum"`
	LowerStratumWeightBonus   uint64            `json:"lower_stratum_weight_bonus"` // basis points (1,000,000 = 1.0x)
}

// EffectiveReputationResult is returned by GetEffectiveReputation.
type EffectiveReputationResult struct {
	Score      uint64           `json:"score"`
	Source     ReputationSource `json:"source"`
	Discount   uint64           `json:"discount"` // basis points
	Confidence uint64           `json:"confidence"` // basis points
}

// --- Params ---

// Params defines the capture defense module parameters.
type Params struct {
	StakeLockPeriod            uint64 `json:"stake_lock_period"`
	MinVerificationsForQual    uint64 `json:"min_verifications_for_qual"`
	MinAccuracyForQual         uint64 `json:"min_accuracy_for_qual"`
	MinEndorsersForQual        uint64 `json:"min_endorsers_for_qual"`
	InactivityExpirationBlocks uint64 `json:"inactivity_expiration_blocks"`
	ProbationaryPeriodBlocks   uint64 `json:"probationary_period_blocks"`
	ReputationHalfLifeBlocks   uint64 `json:"reputation_half_life_blocks"`
	DetectionIntervalBlocks    uint64 `json:"detection_interval_blocks"`
	DecayIntervalBlocks        uint64 `json:"decay_interval_blocks"`
	HerfindahlTriggerThreshold uint64 `json:"herfindahl_trigger_threshold"`
	MaxExpirationsPerBlock     uint64 `json:"max_expirations_per_block"`
}

func (p *Params) Reset()         { *p = Params{} }
func (p *Params) String() string { return proto.CompactTextString(p) }
func (p *Params) ProtoMessage()  {}

// DefaultParams returns the default capture defense parameters.
func DefaultParams() Params {
	return Params{
		StakeLockPeriod:            StakeLockPeriod,
		MinVerificationsForQual:    MinVerificationsForQual,
		MinAccuracyForQual:         MinAccuracyForQual,
		MinEndorsersForQual:        MinEndorsersForQual,
		InactivityExpirationBlocks: InactivityExpirationBlocks,
		ProbationaryPeriodBlocks:   ProbationaryPeriodBlocks,
		ReputationHalfLifeBlocks:   ReputationHalfLifeBlocks,
		DetectionIntervalBlocks:    DetectionIntervalBlocks,
		DecayIntervalBlocks:        DecayIntervalBlocks,
		HerfindahlTriggerThreshold: HerfindahlTriggerThreshold,
		MaxExpirationsPerBlock:     MaxExpirationsPerBlock,
	}
}

// Validate performs basic param validation.
func (p Params) Validate() error {
	if p.StakeLockPeriod == 0 {
		return fmt.Errorf("stake lock period must be positive")
	}
	if p.MinAccuracyForQual > 1_000_000 {
		return fmt.Errorf("min accuracy must be <= 1,000,000")
	}
	if p.HerfindahlTriggerThreshold > 1_000_000 {
		return fmt.Errorf("herfindahl trigger must be <= 1,000,000")
	}
	return nil
}

// --- Genesis ---

// GenesisState defines the capture defense module's genesis state.
type GenesisState struct {
	Params         Params                `json:"params"`
	Qualifications []DomainQualification `json:"qualifications"`
	Reputations    []IsolatedReputation  `json:"reputations"`
	Metrics        []CaptureMetrics      `json:"metrics"`
}

func (gs *GenesisState) Reset()         { *gs = GenesisState{} }
func (gs *GenesisState) String() string { return proto.CompactTextString(gs) }
func (gs *GenesisState) ProtoMessage()  {}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		Qualifications: []DomainQualification{},
		Reputations:    []IsolatedReputation{},
		Metrics:        []CaptureMetrics{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

// --- Message Types ---

// MsgRequestQualification requests qualification for a domain.
type MsgRequestQualification struct {
	ValidatorAddr string               `json:"validator_addr"`
	Domain        string               `json:"domain"`
	Pathway       QualificationPathway `json:"pathway"`
	Stake         string               `json:"stake,omitempty"`
	Endorsers     []string             `json:"endorsers,omitempty"`
	InheritFrom   string               `json:"inherit_from,omitempty"`
}

func (m *MsgRequestQualification) Reset()         { *m = MsgRequestQualification{} }
func (m *MsgRequestQualification) String() string { return proto.CompactTextString(m) }
func (m *MsgRequestQualification) ProtoMessage()  {}

func (m *MsgRequestQualification) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.ValidatorAddr); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	switch m.Pathway {
	case PathwayStakeCommitment, PathwayTrackRecord, PathwayCrossReference, PathwayStratumInheritance:
	default:
		return fmt.Errorf("invalid pathway: %s", m.Pathway)
	}
	return nil
}

func (m *MsgRequestQualification) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.ValidatorAddr)
	return []sdk.AccAddress{addr}
}

// MsgRequestQualificationResponse is the response.
type MsgRequestQualificationResponse struct {
	Granted bool   `json:"granted"`
	Error   string `json:"error,omitempty"`
}

func (m *MsgRequestQualificationResponse) Reset()         { *m = MsgRequestQualificationResponse{} }
func (m *MsgRequestQualificationResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgRequestQualificationResponse) ProtoMessage()  {}

// MsgEndorseQualification endorses another validator's qualification.
type MsgEndorseQualification struct {
	Endorser        string `json:"endorser"`
	TargetValidator string `json:"target_validator"`
	Domain          string `json:"domain"`
	Stake           string `json:"stake"`
}

func (m *MsgEndorseQualification) Reset()         { *m = MsgEndorseQualification{} }
func (m *MsgEndorseQualification) String() string { return proto.CompactTextString(m) }
func (m *MsgEndorseQualification) ProtoMessage()  {}

func (m *MsgEndorseQualification) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Endorser); err != nil {
		return fmt.Errorf("invalid endorser address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.TargetValidator); err != nil {
		return fmt.Errorf("invalid target validator address: %w", err)
	}
	if m.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if m.Endorser == m.TargetValidator {
		return ErrSelfEndorsement
	}
	return nil
}

func (m *MsgEndorseQualification) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Endorser)
	return []sdk.AccAddress{addr}
}

// MsgEndorseQualificationResponse is the response.
type MsgEndorseQualificationResponse struct {
	Success bool `json:"success"`
}

func (m *MsgEndorseQualificationResponse) Reset()         { *m = MsgEndorseQualificationResponse{} }
func (m *MsgEndorseQualificationResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgEndorseQualificationResponse) ProtoMessage()  {}

// --- Service Interfaces ---

// MsgServer defines the capture defense module's message server interface.
type MsgServer interface {
	RequestQualification(context.Context, *MsgRequestQualification) (*MsgRequestQualificationResponse, error)
	EndorseQualification(context.Context, *MsgEndorseQualification) (*MsgEndorseQualificationResponse, error)
}

// QueryServer defines the capture defense module's query server interface.
type QueryServer interface {
	Qualification(context.Context, *QueryQualificationRequest) (*QueryQualificationResponse, error)
	ValidatorQualifications(context.Context, *QueryValidatorQualificationsRequest) (*QueryValidatorQualificationsResponse, error)
	DomainReputation(context.Context, *QueryDomainReputationRequest) (*QueryDomainReputationResponse, error)
	CaptureRisk(context.Context, *QueryCaptureRiskRequest) (*QueryCaptureRiskResponse, error)
}

// --- Query Types ---

type QueryQualificationRequest struct {
	ValidatorAddr string `json:"validator_addr"`
	Domain        string `json:"domain"`
}

type QueryQualificationResponse struct {
	Qualification *DomainQualification `json:"qualification,omitempty"`
	Found         bool                 `json:"found"`
}

type QueryValidatorQualificationsRequest struct {
	ValidatorAddr string `json:"validator_addr"`
}

type QueryValidatorQualificationsResponse struct {
	Qualifications []DomainQualification `json:"qualifications"`
}

type QueryDomainReputationRequest struct {
	ValidatorAddr string `json:"validator_addr"`
	Domain        string `json:"domain"`
}

type QueryDomainReputationResponse struct {
	Reputation EffectiveReputationResult `json:"reputation"`
}

type QueryCaptureRiskRequest struct {
	Domain string `json:"domain"`
}

type QueryCaptureRiskResponse struct {
	Metrics *CaptureMetrics `json:"metrics,omitempty"`
	Found   bool            `json:"found"`
}

// --- Codec Registration ---

// RegisterCodec registers the capture defense module's types with the amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRequestQualification{}, "capture_defense/RequestQualification", nil)
	cdc.RegisterConcrete(&MsgEndorseQualification{}, "capture_defense/EndorseQualification", nil)
}

// RegisterInterfaces registers the module's interfaces.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Hand-written types lack proto typeURLs; registration deferred to proto codegen.
}

// gRPC service descriptors for msg server
var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.capture_defense.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "RequestQualification", Handler: _Msg_RequestQualification_Handler},
		{MethodName: "EndorseQualification", Handler: _Msg_EndorseQualification_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Msg_RequestQualification_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRequestQualification)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RequestQualification(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.capture_defense.v1.Msg/RequestQualification"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RequestQualification(ctx, req.(*MsgRequestQualification))
	})
}

func _Msg_EndorseQualification_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgEndorseQualification)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).EndorseQualification(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.capture_defense.v1.Msg/EndorseQualification"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).EndorseQualification(ctx, req.(*MsgEndorseQualification))
	})
}

// RegisterMsgServer registers the capture defense msg server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// gRPC service descriptors for query server
var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.capture_defense.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Qualification", Handler: _Query_Qualification_Handler},
		{MethodName: "ValidatorQualifications", Handler: _Query_ValidatorQualifications_Handler},
		{MethodName: "DomainReputation", Handler: _Query_DomainReputation_Handler},
		{MethodName: "CaptureRisk", Handler: _Query_CaptureRisk_Handler},
	},
	Streams: []grpc.StreamDesc{},
}

func _Query_Qualification_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryQualificationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Qualification(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.capture_defense.v1.Query/Qualification"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Qualification(ctx, req.(*QueryQualificationRequest))
	})
}

func _Query_ValidatorQualifications_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryValidatorQualificationsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ValidatorQualifications(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.capture_defense.v1.Query/ValidatorQualifications"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ValidatorQualifications(ctx, req.(*QueryValidatorQualificationsRequest))
	})
}

func _Query_DomainReputation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDomainReputationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).DomainReputation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.capture_defense.v1.Query/DomainReputation"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).DomainReputation(ctx, req.(*QueryDomainReputationRequest))
	})
}

func _Query_CaptureRisk_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryCaptureRiskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).CaptureRisk(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.capture_defense.v1.Query/CaptureRisk"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).CaptureRisk(ctx, req.(*QueryCaptureRiskRequest))
	})
}

// RegisterQueryServer registers the capture defense query server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
