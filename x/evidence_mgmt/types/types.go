package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// ---------- Enumerations ----------

// EvidenceType categorizes evidence sources.
type EvidenceType string

const (
	EvidenceTypeOnChain     EvidenceType = "on_chain"
	EvidenceTypeOracleFeed  EvidenceType = "oracle_feed"
	EvidenceTypeAttestation EvidenceType = "attestation"
	EvidenceTypeComputation EvidenceType = "computation"
	EvidenceTypeDocument    EvidenceType = "document"
	EvidenceTypeExperiment  EvidenceType = "experiment"
)

// ValidEvidenceTypes returns all valid evidence types.
func ValidEvidenceTypes() []EvidenceType {
	return []EvidenceType{
		EvidenceTypeOnChain,
		EvidenceTypeOracleFeed,
		EvidenceTypeAttestation,
		EvidenceTypeComputation,
		EvidenceTypeDocument,
		EvidenceTypeExperiment,
	}
}

// IsValidEvidenceType checks if the given type is valid.
func IsValidEvidenceType(t EvidenceType) bool {
	for _, valid := range ValidEvidenceTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// EvidenceStatus tracks evidence lifecycle.
type EvidenceStatus string

const (
	EvidenceStatusSubmitted EvidenceStatus = "submitted"
	EvidenceStatusVerified  EvidenceStatus = "verified"
	EvidenceStatusRejected  EvidenceStatus = "rejected"
	EvidenceStatusExpired   EvidenceStatus = "expired"
)

// AuditStatus tracks audit lifecycle.
type AuditStatus string

const (
	AuditStatusPending    AuditStatus = "pending"
	AuditStatusAccurate   AuditStatus = "accurate"
	AuditStatusInaccurate AuditStatus = "inaccurate"
	AuditStatusSlashed    AuditStatus = "slashed"
	AuditStatusRewarded   AuditStatus = "rewarded"
	AuditStatusExpired    AuditStatus = "expired"
)

// ---------- Core Types ----------

// Evidence represents a piece of evidence supporting or contradicting a claim/fact.
type Evidence struct {
	Id           string         `json:"id"`
	ClaimId      string         `json:"claim_id"`
	FactId       string         `json:"fact_id"`
	Submitter    string         `json:"submitter"`
	EvidenceType EvidenceType   `json:"evidence_type"`
	ContentHash  string         `json:"content_hash"`
	Description  string         `json:"description"`
	SourceURI    string         `json:"source_uri"`
	Status       EvidenceStatus `json:"status"`
	Reliability  uint64         `json:"reliability"`
	SubmittedAt  uint64         `json:"submitted_at"`
	VerifiedAt   uint64         `json:"verified_at"`
	AuditCount   uint32         `json:"audit_count"`
}

func (e *Evidence) Reset()        {}
func (e *Evidence) String() string { return fmt.Sprintf("%+v", *e) }
func (e *Evidence) ProtoMessage()  {}

// EvidenceAudit represents an auditor's assessment of evidence quality.
type EvidenceAudit struct {
	Id            string      `json:"id"`
	EvidenceId    string      `json:"evidence_id"`
	Auditor       string      `json:"auditor"`
	Stake         string      `json:"stake"`
	Score         uint64      `json:"score"`
	Rationale     string      `json:"rationale"`
	Status        AuditStatus `json:"status"`
	SubmittedAt   uint64      `json:"submitted_at"`
	ResolvedAt    uint64      `json:"resolved_at"`
	RewardAmount  string      `json:"reward_amount"`
	SlashAmount   string      `json:"slash_amount"`
	AccuracyDelta uint64      `json:"accuracy_delta"`
}

func (a *EvidenceAudit) Reset()        {}
func (a *EvidenceAudit) String() string { return fmt.Sprintf("%+v", *a) }
func (a *EvidenceAudit) ProtoMessage()  {}

// AuditConsensus represents the aggregated audit scores for evidence.
type AuditConsensus struct {
	EvidenceId     string `json:"evidence_id"`
	WeightedScore  uint64 `json:"weighted_score"`
	TotalStake     string `json:"total_stake"`
	AuditCount     uint32 `json:"audit_count"`
	Variance       uint64 `json:"variance"`
	IsHighVariance bool   `json:"is_high_variance"`
	ComputedAt     uint64 `json:"computed_at"`
}

func (ac *AuditConsensus) Reset()        {}
func (ac *AuditConsensus) String() string { return fmt.Sprintf("%+v", *ac) }
func (ac *AuditConsensus) ProtoMessage()  {}

// Oracle represents a registered external data oracle.
type Oracle struct {
	Id              string         `json:"id"`
	Operator        string         `json:"operator"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	DataTypes       []EvidenceType `json:"data_types"`
	Stake           string         `json:"stake"`
	ReputationScore uint64         `json:"reputation_score"`
	DataPoints      uint64         `json:"data_points"`
	AccuracyRate    uint64         `json:"accuracy_rate"`
	RegisteredAt    uint64         `json:"registered_at"`
	IsActive        bool           `json:"is_active"`
}

func (o *Oracle) Reset()        {}
func (o *Oracle) String() string { return fmt.Sprintf("%+v", *o) }
func (o *Oracle) ProtoMessage()  {}

// Params defines the evidence_mgmt module parameters.
type Params struct {
	MinAuditStake         string `json:"min_audit_stake"`
	AuditWindow           uint64 `json:"audit_window"`
	MinAuditsForConsensus uint32 `json:"min_audits_for_consensus"`
	AccuracyThreshold     uint64 `json:"accuracy_threshold"`
	SlashRate             uint64 `json:"slash_rate"`
	RewardRate            uint64 `json:"reward_rate"`
	MinOracleStake        string `json:"min_oracle_stake"`
	OracleSlashRate       uint64 `json:"oracle_slash_rate"`
	MaxEvidencePerClaim   uint32 `json:"max_evidence_per_claim"`
}

func (p *Params) Reset()        {}
func (p *Params) String() string { return fmt.Sprintf("%+v", *p) }
func (p *Params) ProtoMessage()  {}

// Validate performs parameter validation.
func (p Params) Validate() error {
	if p.AccuracyThreshold > 1000000 {
		return fmt.Errorf("%w: accuracy_threshold cannot exceed 1000000", ErrInvalidParams)
	}
	if p.SlashRate > 1000000 {
		return fmt.Errorf("%w: slash_rate cannot exceed 1000000", ErrInvalidParams)
	}
	if p.RewardRate > 1000000 {
		return fmt.Errorf("%w: reward_rate cannot exceed 1000000", ErrInvalidParams)
	}
	if p.OracleSlashRate > 1000000 {
		return fmt.Errorf("%w: oracle_slash_rate cannot exceed 1000000", ErrInvalidParams)
	}
	if p.MinAuditsForConsensus == 0 {
		return fmt.Errorf("%w: min_audits_for_consensus must be > 0", ErrInvalidParams)
	}
	return nil
}

// ---------- Message Types ----------

// MsgSubmitEvidence submits evidence for a claim.
type MsgSubmitEvidence struct {
	Submitter    string       `json:"submitter"`
	ClaimId      string       `json:"claim_id"`
	EvidenceType EvidenceType `json:"evidence_type"`
	ContentHash  string       `json:"content_hash"`
	Description  string       `json:"description"`
	SourceURI    string       `json:"source_uri"`
}

func (msg *MsgSubmitEvidence) Reset()        {}
func (msg *MsgSubmitEvidence) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitEvidence) ProtoMessage()  {}

func (msg *MsgSubmitEvidence) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitEvidence) ValidateBasic() error {
	if msg.Submitter == "" {
		return fmt.Errorf("submitter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if msg.ClaimId == "" {
		return fmt.Errorf("claim_id cannot be empty")
	}
	if !IsValidEvidenceType(msg.EvidenceType) {
		return fmt.Errorf("invalid evidence type: %s", msg.EvidenceType)
	}
	if msg.ContentHash == "" {
		return fmt.Errorf("content_hash cannot be empty")
	}
	if msg.Description == "" {
		return fmt.Errorf("description cannot be empty")
	}
	return nil
}

type MsgSubmitEvidenceResponse struct {
	EvidenceId string `json:"evidence_id"`
}

func (msg *MsgSubmitEvidenceResponse) Reset()        {}
func (msg *MsgSubmitEvidenceResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitEvidenceResponse) ProtoMessage()  {}

// MsgAuditEvidence submits an audit score for evidence.
type MsgAuditEvidence struct {
	Auditor    string `json:"auditor"`
	EvidenceId string `json:"evidence_id"`
	Score      uint64 `json:"score"`
	Rationale  string `json:"rationale"`
	Stake      string `json:"stake"`
}

func (msg *MsgAuditEvidence) Reset()        {}
func (msg *MsgAuditEvidence) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgAuditEvidence) ProtoMessage()  {}

func (msg *MsgAuditEvidence) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Auditor)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAuditEvidence) ValidateBasic() error {
	if msg.Auditor == "" {
		return fmt.Errorf("auditor cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Auditor); err != nil {
		return fmt.Errorf("invalid auditor address: %w", err)
	}
	if msg.EvidenceId == "" {
		return fmt.Errorf("evidence_id cannot be empty")
	}
	if msg.Score > 1000000 {
		return fmt.Errorf("score must be 0-1000000")
	}
	if msg.Rationale == "" {
		return fmt.Errorf("rationale cannot be empty")
	}
	if msg.Stake == "" {
		return fmt.Errorf("stake cannot be empty")
	}
	return nil
}

type MsgAuditEvidenceResponse struct {
	AuditId string `json:"audit_id"`
}

func (msg *MsgAuditEvidenceResponse) Reset()        {}
func (msg *MsgAuditEvidenceResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgAuditEvidenceResponse) ProtoMessage()  {}

// MsgRegisterOracle registers a new oracle operator.
type MsgRegisterOracle struct {
	Operator    string         `json:"operator"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	DataTypes   []EvidenceType `json:"data_types"`
	Stake       string         `json:"stake"`
}

func (msg *MsgRegisterOracle) Reset()        {}
func (msg *MsgRegisterOracle) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRegisterOracle) ProtoMessage()  {}

func (msg *MsgRegisterOracle) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Operator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterOracle) ValidateBasic() error {
	if msg.Operator == "" {
		return fmt.Errorf("operator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}
	if msg.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(msg.DataTypes) == 0 {
		return fmt.Errorf("data_types cannot be empty")
	}
	for _, dt := range msg.DataTypes {
		if !IsValidEvidenceType(dt) {
			return fmt.Errorf("invalid data type: %s", dt)
		}
	}
	if msg.Stake == "" {
		return fmt.Errorf("stake cannot be empty")
	}
	return nil
}

type MsgRegisterOracleResponse struct {
	OracleId string `json:"oracle_id"`
}

func (msg *MsgRegisterOracleResponse) Reset()        {}
func (msg *MsgRegisterOracleResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRegisterOracleResponse) ProtoMessage()  {}

// MsgSubmitOracleData submits evidence from a registered oracle.
type MsgSubmitOracleData struct {
	OracleId    string       `json:"oracle_id"`
	ClaimId     string       `json:"claim_id"`
	ContentHash string       `json:"content_hash"`
	DataType    EvidenceType `json:"data_type"`
}

func (msg *MsgSubmitOracleData) Reset()        {}
func (msg *MsgSubmitOracleData) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitOracleData) ProtoMessage()  {}

func (msg *MsgSubmitOracleData) GetSigners() []sdk.AccAddress {
	// Oracle data is signed by the oracle operator, resolved at msg_server level.
	return nil
}

func (msg *MsgSubmitOracleData) ValidateBasic() error {
	if msg.OracleId == "" {
		return fmt.Errorf("oracle_id cannot be empty")
	}
	if msg.ClaimId == "" {
		return fmt.Errorf("claim_id cannot be empty")
	}
	if msg.ContentHash == "" {
		return fmt.Errorf("content_hash cannot be empty")
	}
	if !IsValidEvidenceType(msg.DataType) {
		return fmt.Errorf("invalid data type: %s", msg.DataType)
	}
	return nil
}

type MsgSubmitOracleDataResponse struct {
	EvidenceId string `json:"evidence_id"`
}

func (msg *MsgSubmitOracleDataResponse) Reset()        {}
func (msg *MsgSubmitOracleDataResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgSubmitOracleDataResponse) ProtoMessage()  {}

// MsgUpdateParams updates evidence_mgmt module parameters.
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (msg *MsgUpdateParams) Reset()        {}
func (msg *MsgUpdateParams) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgUpdateParams) ProtoMessage()  {}

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
	return msg.Params.Validate()
}

type MsgUpdateParamsResponse struct{}

func (msg *MsgUpdateParamsResponse) Reset()        {}
func (msg *MsgUpdateParamsResponse) String() string { return "" }
func (msg *MsgUpdateParamsResponse) ProtoMessage()  {}

// ---------- MsgServer Interface ----------

// MsgServer is the service interface for the evidence_mgmt module.
type MsgServer interface {
	SubmitEvidence(context.Context, *MsgSubmitEvidence) (*MsgSubmitEvidenceResponse, error)
	AuditEvidence(context.Context, *MsgAuditEvidence) (*MsgAuditEvidenceResponse, error)
	RegisterOracle(context.Context, *MsgRegisterOracle) (*MsgRegisterOracleResponse, error)
	SubmitOracleData(context.Context, *MsgSubmitOracleData) (*MsgSubmitOracleDataResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}

// Placeholder service descriptor for codec registration.
var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.evidence_mgmt.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "SubmitEvidence", Handler: nil},
		{MethodName: "AuditEvidence", Handler: nil},
		{MethodName: "RegisterOracle", Handler: nil},
		{MethodName: "SubmitOracleData", Handler: nil},
		{MethodName: "UpdateParams", Handler: nil},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/evidence_mgmt/v1/tx.proto",
}
