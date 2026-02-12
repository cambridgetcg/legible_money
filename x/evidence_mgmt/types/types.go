package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

// ---------- ValidateBasic / GetSigners ----------

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
	if !IsValidEvidenceType(EvidenceType(msg.EvidenceType)) {
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
		if !IsValidEvidenceType(EvidenceType(dt)) {
			return fmt.Errorf("invalid data type: %s", dt)
		}
	}
	if msg.Stake == "" {
		return fmt.Errorf("stake cannot be empty")
	}
	return nil
}

func (msg *MsgSubmitOracleData) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Operator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitOracleData) ValidateBasic() error {
	if msg.Operator == "" {
		return fmt.Errorf("operator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}
	if msg.OracleId == "" {
		return fmt.Errorf("oracle_id cannot be empty")
	}
	if msg.ClaimId == "" {
		return fmt.Errorf("claim_id cannot be empty")
	}
	if msg.ContentHash == "" {
		return fmt.Errorf("content_hash cannot be empty")
	}
	if !IsValidEvidenceType(EvidenceType(msg.DataType)) {
		return fmt.Errorf("invalid data type: %s", msg.DataType)
	}
	return nil
}

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
