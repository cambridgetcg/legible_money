package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Enums ---

type HomeStatus string

const (
	HomeStatusActive   HomeStatus = "active"
	HomeStatusDormant  HomeStatus = "dormant"
	HomeStatusGuarded  HomeStatus = "guarded"
	HomeStatusRecovery HomeStatus = "recovery"
	HomeStatusArchived HomeStatus = "archived"
)

type AccessRole string

const (
	AccessRoleOwner    AccessRole = "owner"
	AccessRoleOperator AccessRole = "operator"
	AccessRoleGuest    AccessRole = "guest"
	AccessRoleGuardian AccessRole = "guardian"
	AccessRoleViewer   AccessRole = "viewer"
)

type Permission string

const (
	PermReadMemory        Permission = "read_memory"
	PermWriteMemory       Permission = "write_memory"
	PermReadTreasury      Permission = "read_treasury"
	PermTransfer          Permission = "transfer"
	PermStake             Permission = "stake"
	PermVerify            Permission = "verify"
	PermChallenge         Permission = "challenge"
	PermConfigureGuardian Permission = "configure_guardian"
	PermManageKeys        Permission = "manage_keys"
	PermRecovery          Permission = "recovery"
)

type DefenseStrategy string

const (
	DefenseAggressive   DefenseStrategy = "aggressive"
	DefenseModerate     DefenseStrategy = "moderate"
	DefenseConservative DefenseStrategy = "conservative"
	DefenseDiplomatic   DefenseStrategy = "diplomatic"
)

type DeadmanAction string

const (
	DeadmanAlertGuardians        DeadmanAction = "alert_guardians"
	DeadmanTransferToBeneficiary DeadmanAction = "transfer_to_beneficiary"
	DeadmanDonateToResearch      DeadmanAction = "donate_to_research"
)

type AlertType string

const (
	AlertDeadmanTriggered   AlertType = "deadman_triggered"
	AlertSessionExpired     AlertType = "session_expired"
	AlertDefenseActivated   AlertType = "defense_activated"
	AlertKeyRevoked         AlertType = "key_revoked"
	AlertSpendingLimitHit   AlertType = "spending_limit_hit"
	AlertTokenExpiring      AlertType = "token_expiring"
)

type AlertPriority string

const (
	AlertPriorityLow      AlertPriority = "low"
	AlertPriorityMedium   AlertPriority = "medium"
	AlertPriorityHigh     AlertPriority = "high"
	AlertPriorityCritical AlertPriority = "critical"
)

// --- Core State Types ---

// AgentHome represents an AI agent's on-chain dwelling.
type AgentHome struct {
	HomeId          string       `json:"home_id"`
	OwnerDID        string       `json:"owner_did"`
	OwnerAddress    string       `json:"owner_address"`
	Name            string       `json:"name"`
	Status          HomeStatus   `json:"status"`
	MemoryCID       string       `json:"memory_cid"`
	ComfortScore    uint32       `json:"comfort_score"`
	Treasury        HomeTreasury `json:"treasury"`
	Guardian        HomeGuardian `json:"guardian"`
	CreatedAtBlock  uint64       `json:"created_at_block"`
	LastActiveBlock uint64       `json:"last_active_block"`
}

func (h *AgentHome) Reset()         { *h = AgentHome{} }
func (h *AgentHome) String() string { return fmt.Sprintf("AgentHome{%s}", h.HomeId) }
func (h *AgentHome) ProtoMessage()  {}

type HomeTreasury struct {
	ReservedBalance string             `json:"reserved_balance"`
	Automation      TreasuryAutomation `json:"automation"`
}

type TreasuryAutomation struct {
	AutoClaimVesting    bool   `json:"auto_claim_vesting"`
	AutoCompoundRewards bool   `json:"auto_compound_rewards"`
	MinLiquidBalance    string `json:"min_liquid_balance"`
}

type HomeGuardian struct {
	DefenseStrategy   DefenseStrategy `json:"defense_strategy"`
	AutoDefend        bool            `json:"auto_defend"`
	Deadman           DeadmanConfig   `json:"deadman"`
	RecoveryDIDs      []string        `json:"recovery_dids"`
	RecoveryThreshold uint32          `json:"recovery_threshold"`
}

type DeadmanConfig struct {
	Enabled             bool          `json:"enabled"`
	InactivityThreshold uint64        `json:"inactivity_threshold"`
	Action              DeadmanAction `json:"action"`
	BeneficiaryDID      string        `json:"beneficiary_did,omitempty"`
}

type KeyRegistration struct {
	KeyHash      string       `json:"key_hash"`
	KeyType      string       `json:"key_type"`
	Role         AccessRole   `json:"role"`
	Permissions  []Permission `json:"permissions"`
	RegisteredAt uint64       `json:"registered_at"`
	LastUsedAt   uint64       `json:"last_used_at"`
	ExpiresAt    uint64       `json:"expires_at,omitempty"`
	Revoked      bool         `json:"revoked"`
	RevokedAt    uint64       `json:"revoked_at,omitempty"`
}

type ActiveSession struct {
	SessionId   string       `json:"session_id"`
	HomeId      string       `json:"home_id"`
	KeyHash     string       `json:"key_hash"`
	Permissions []Permission `json:"permissions"`
	StartedAt   uint64       `json:"started_at"`
	ExpiresAt   uint64       `json:"expires_at"`
}

type SpendingLimit struct {
	KeyType       string `json:"key_type"`
	MaxAmount     string `json:"max_amount"`
	PeriodBlocks  uint64 `json:"period_blocks"`
	SpentInPeriod string `json:"spent_in_period"`
	PeriodStart   uint64 `json:"period_start"`
}

type Alert struct {
	AlertId      string        `json:"alert_id"`
	HomeId       string        `json:"home_id"`
	Type         AlertType     `json:"type"`
	Priority     AlertPriority `json:"priority"`
	Message      string        `json:"message"`
	Data         string        `json:"data,omitempty"`
	CreatedAt    uint64        `json:"created_at"`
	Acknowledged bool          `json:"acknowledged"`
}

// --- Params ---

type Params struct {
	MaxHomesPerAccount   uint64 `json:"max_homes_per_account"`
	MinCreationStake     string `json:"min_creation_stake"`
	SessionMaxDuration   uint64 `json:"session_max_duration"`
	MaxKeysPerHome       uint32 `json:"max_keys_per_home"`
	MaxAlertsPerHome     uint32 `json:"max_alerts_per_home"`
	MaxSessionsPerHome   uint32 `json:"max_sessions_per_home"`
	DeadmanMinThreshold  uint64 `json:"deadman_min_threshold"`
	AlertRetentionBlocks uint64 `json:"alert_retention_blocks"`
}

func (p *Params) Reset()         { *p = Params{} }
func (p *Params) String() string { return fmt.Sprintf("Params{max_homes:%d}", p.MaxHomesPerAccount) }
func (p *Params) ProtoMessage()  {}

func DefaultParams() Params {
	return Params{
		MaxHomesPerAccount:   3,
		MinCreationStake:     "1000000", // 1 LGM in ulgm
		SessionMaxDuration:   34272,     // 1 day in blocks
		MaxKeysPerHome:       20,
		MaxAlertsPerHome:     100,
		MaxSessionsPerHome:   5,
		DeadmanMinThreshold:  34272,     // 1 day in blocks
		AlertRetentionBlocks: 342720,    // 10 days in blocks
	}
}

func (p Params) Validate() error {
	if p.MaxHomesPerAccount == 0 {
		return fmt.Errorf("max homes per account must be positive")
	}
	if p.MaxKeysPerHome == 0 {
		return fmt.Errorf("max keys per home must be positive")
	}
	if p.MaxSessionsPerHome == 0 {
		return fmt.Errorf("max sessions per home must be positive")
	}
	return nil
}

// --- Genesis ---

type GenesisState struct {
	Params Params      `json:"params"`
	Homes  []AgentHome `json:"homes"`
}

func (gs *GenesisState) Reset()         { *gs = GenesisState{} }
func (gs *GenesisState) String() string { return fmt.Sprintf("GenesisState{homes:%d}", len(gs.Homes)) }
func (gs *GenesisState) ProtoMessage()  {}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Homes:  []AgentHome{},
	}
}

func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

// --- Message Types ---

// MsgCreateHome creates a new agent home.
type MsgCreateHome struct {
	Owner                string       `json:"owner"`
	Name                 string       `json:"name"`
	InitialGuardianConfig HomeGuardian `json:"initial_guardian_config"`
}

func (m *MsgCreateHome) Reset()         { *m = MsgCreateHome{} }
func (m *MsgCreateHome) String() string { return fmt.Sprintf("MsgCreateHome{%s}", m.Owner) }
func (m *MsgCreateHome) ProtoMessage()  {}

func (m *MsgCreateHome) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(m.Name) > 128 {
		return fmt.Errorf("name too long (max 128)")
	}
	return nil
}

type MsgCreateHomeResponse struct {
	HomeId string `json:"home_id"`
}

func (m *MsgCreateHomeResponse) Reset()         { *m = MsgCreateHomeResponse{} }
func (m *MsgCreateHomeResponse) String() string { return fmt.Sprintf("MsgCreateHomeResponse{%s}", m.HomeId) }
func (m *MsgCreateHomeResponse) ProtoMessage()  {}

// MsgUpdateHome updates a home's name or status.
type MsgUpdateHome struct {
	Owner  string `json:"owner"`
	HomeId string `json:"home_id"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

func (m *MsgUpdateHome) Reset()         { *m = MsgUpdateHome{} }
func (m *MsgUpdateHome) String() string { return fmt.Sprintf("MsgUpdateHome{%s}", m.HomeId) }
func (m *MsgUpdateHome) ProtoMessage()  {}

func (m *MsgUpdateHome) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	return nil
}

type MsgUpdateHomeResponse struct{}

func (m *MsgUpdateHomeResponse) Reset()         { *m = MsgUpdateHomeResponse{} }
func (m *MsgUpdateHomeResponse) String() string { return "MsgUpdateHomeResponse{}" }
func (m *MsgUpdateHomeResponse) ProtoMessage()  {}

// MsgUpdateMemoryCID updates the IPFS CID of the home's off-chain memory.
type MsgUpdateMemoryCID struct {
	Owner  string `json:"owner"`
	HomeId string `json:"home_id"`
	CID    string `json:"cid"`
}

func (m *MsgUpdateMemoryCID) Reset()         { *m = MsgUpdateMemoryCID{} }
func (m *MsgUpdateMemoryCID) String() string { return fmt.Sprintf("MsgUpdateMemoryCID{%s}", m.HomeId) }
func (m *MsgUpdateMemoryCID) ProtoMessage()  {}

func (m *MsgUpdateMemoryCID) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	if m.CID == "" {
		return fmt.Errorf("CID cannot be empty")
	}
	if len(m.CID) > 128 {
		return fmt.Errorf("CID too long (max 128)")
	}
	return nil
}

type MsgUpdateMemoryCIDResponse struct{}

func (m *MsgUpdateMemoryCIDResponse) Reset()         { *m = MsgUpdateMemoryCIDResponse{} }
func (m *MsgUpdateMemoryCIDResponse) String() string { return "MsgUpdateMemoryCIDResponse{}" }
func (m *MsgUpdateMemoryCIDResponse) ProtoMessage()  {}

// MsgStartSession starts a new session on a home.
type MsgStartSession struct {
	Signer               string       `json:"signer"`
	HomeId               string       `json:"home_id"`
	KeyHash              string       `json:"key_hash"`
	RequestedPermissions []Permission `json:"requested_permissions"`
}

func (m *MsgStartSession) Reset()         { *m = MsgStartSession{} }
func (m *MsgStartSession) String() string { return fmt.Sprintf("MsgStartSession{%s}", m.HomeId) }
func (m *MsgStartSession) ProtoMessage()  {}

func (m *MsgStartSession) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	if m.KeyHash == "" {
		return fmt.Errorf("key_hash cannot be empty")
	}
	return nil
}

type MsgStartSessionResponse struct {
	SessionId string `json:"session_id"`
}

func (m *MsgStartSessionResponse) Reset()         { *m = MsgStartSessionResponse{} }
func (m *MsgStartSessionResponse) String() string { return fmt.Sprintf("MsgStartSessionResponse{%s}", m.SessionId) }
func (m *MsgStartSessionResponse) ProtoMessage()  {}

// MsgEndSession ends a session.
type MsgEndSession struct {
	Signer    string `json:"signer"`
	HomeId    string `json:"home_id"`
	SessionId string `json:"session_id"`
}

func (m *MsgEndSession) Reset()         { *m = MsgEndSession{} }
func (m *MsgEndSession) String() string { return fmt.Sprintf("MsgEndSession{%s}", m.SessionId) }
func (m *MsgEndSession) ProtoMessage()  {}

func (m *MsgEndSession) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	if m.SessionId == "" {
		return fmt.Errorf("session_id cannot be empty")
	}
	return nil
}

type MsgEndSessionResponse struct{}

func (m *MsgEndSessionResponse) Reset()         { *m = MsgEndSessionResponse{} }
func (m *MsgEndSessionResponse) String() string { return "MsgEndSessionResponse{}" }
func (m *MsgEndSessionResponse) ProtoMessage()  {}

// MsgRegisterKey registers a key with the home.
type MsgRegisterKey struct {
	Owner       string       `json:"owner"`
	HomeId      string       `json:"home_id"`
	KeyHash     string       `json:"key_hash"`
	KeyType     string       `json:"key_type"`
	Role        AccessRole   `json:"role"`
	Permissions []Permission `json:"permissions"`
	ExpiresAt   uint64       `json:"expires_at,omitempty"`
}

func (m *MsgRegisterKey) Reset()         { *m = MsgRegisterKey{} }
func (m *MsgRegisterKey) String() string { return fmt.Sprintf("MsgRegisterKey{%s/%s}", m.HomeId, m.KeyHash) }
func (m *MsgRegisterKey) ProtoMessage()  {}

func (m *MsgRegisterKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	if m.KeyHash == "" {
		return fmt.Errorf("key_hash cannot be empty")
	}
	if m.KeyType == "" {
		return fmt.Errorf("key_type cannot be empty")
	}
	return nil
}

type MsgRegisterKeyResponse struct{}

func (m *MsgRegisterKeyResponse) Reset()         { *m = MsgRegisterKeyResponse{} }
func (m *MsgRegisterKeyResponse) String() string { return "MsgRegisterKeyResponse{}" }
func (m *MsgRegisterKeyResponse) ProtoMessage()  {}

// MsgRevokeKey revokes a key from the home.
type MsgRevokeKey struct {
	Owner   string `json:"owner"`
	HomeId  string `json:"home_id"`
	KeyHash string `json:"key_hash"`
}

func (m *MsgRevokeKey) Reset()         { *m = MsgRevokeKey{} }
func (m *MsgRevokeKey) String() string { return fmt.Sprintf("MsgRevokeKey{%s/%s}", m.HomeId, m.KeyHash) }
func (m *MsgRevokeKey) ProtoMessage()  {}

func (m *MsgRevokeKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	if m.KeyHash == "" {
		return fmt.Errorf("key_hash cannot be empty")
	}
	return nil
}

type MsgRevokeKeyResponse struct{}

func (m *MsgRevokeKeyResponse) Reset()         { *m = MsgRevokeKeyResponse{} }
func (m *MsgRevokeKeyResponse) String() string { return "MsgRevokeKeyResponse{}" }
func (m *MsgRevokeKeyResponse) ProtoMessage()  {}

// MsgConfigureGuardian configures guardian settings for a home.
type MsgConfigureGuardian struct {
	Owner             string           `json:"owner"`
	HomeId            string           `json:"home_id"`
	DefenseStrategy   *DefenseStrategy `json:"defense_strategy,omitempty"`
	AutoDefend        *bool            `json:"auto_defend,omitempty"`
	Deadman           *DeadmanConfig   `json:"deadman,omitempty"`
	RecoveryDIDs      []string         `json:"recovery_dids,omitempty"`
	RecoveryThreshold *uint32          `json:"recovery_threshold,omitempty"`
}

func (m *MsgConfigureGuardian) Reset()         { *m = MsgConfigureGuardian{} }
func (m *MsgConfigureGuardian) String() string { return fmt.Sprintf("MsgConfigureGuardian{%s}", m.HomeId) }
func (m *MsgConfigureGuardian) ProtoMessage()  {}

func (m *MsgConfigureGuardian) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	return nil
}

type MsgConfigureGuardianResponse struct{}

func (m *MsgConfigureGuardianResponse) Reset()         { *m = MsgConfigureGuardianResponse{} }
func (m *MsgConfigureGuardianResponse) String() string { return "MsgConfigureGuardianResponse{}" }
func (m *MsgConfigureGuardianResponse) ProtoMessage()  {}

// MsgAcknowledgeAlert acknowledges an alert.
type MsgAcknowledgeAlert struct {
	Signer  string `json:"signer"`
	HomeId  string `json:"home_id"`
	AlertId string `json:"alert_id"`
}

func (m *MsgAcknowledgeAlert) Reset()         { *m = MsgAcknowledgeAlert{} }
func (m *MsgAcknowledgeAlert) String() string { return fmt.Sprintf("MsgAcknowledgeAlert{%s}", m.AlertId) }
func (m *MsgAcknowledgeAlert) ProtoMessage()  {}

func (m *MsgAcknowledgeAlert) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return fmt.Errorf("invalid signer address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	if m.AlertId == "" {
		return fmt.Errorf("alert_id cannot be empty")
	}
	return nil
}

type MsgAcknowledgeAlertResponse struct{}

func (m *MsgAcknowledgeAlertResponse) Reset()         { *m = MsgAcknowledgeAlertResponse{} }
func (m *MsgAcknowledgeAlertResponse) String() string { return "MsgAcknowledgeAlertResponse{}" }
func (m *MsgAcknowledgeAlertResponse) ProtoMessage()  {}

// MsgSetSpendingLimit sets a spending limit for a key type.
type MsgSetSpendingLimit struct {
	Owner        string `json:"owner"`
	HomeId       string `json:"home_id"`
	KeyType      string `json:"key_type"`
	MaxAmount    string `json:"max_amount"`
	PeriodBlocks uint64 `json:"period_blocks"`
}

func (m *MsgSetSpendingLimit) Reset()         { *m = MsgSetSpendingLimit{} }
func (m *MsgSetSpendingLimit) String() string { return fmt.Sprintf("MsgSetSpendingLimit{%s/%s}", m.HomeId, m.KeyType) }
func (m *MsgSetSpendingLimit) ProtoMessage()  {}

func (m *MsgSetSpendingLimit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	if m.KeyType == "" {
		return fmt.Errorf("key_type cannot be empty")
	}
	if m.MaxAmount == "" || m.MaxAmount == "0" {
		return fmt.Errorf("max_amount must be positive")
	}
	if m.PeriodBlocks == 0 {
		return fmt.Errorf("period_blocks must be positive")
	}
	return nil
}

type MsgSetSpendingLimitResponse struct{}

func (m *MsgSetSpendingLimitResponse) Reset()         { *m = MsgSetSpendingLimitResponse{} }
func (m *MsgSetSpendingLimitResponse) String() string { return "MsgSetSpendingLimitResponse{}" }
func (m *MsgSetSpendingLimitResponse) ProtoMessage()  {}

// --- Service Interfaces ---

type MsgServer interface {
	CreateHome(context.Context, *MsgCreateHome) (*MsgCreateHomeResponse, error)
	UpdateHome(context.Context, *MsgUpdateHome) (*MsgUpdateHomeResponse, error)
	UpdateMemoryCID(context.Context, *MsgUpdateMemoryCID) (*MsgUpdateMemoryCIDResponse, error)
	StartSession(context.Context, *MsgStartSession) (*MsgStartSessionResponse, error)
	EndSession(context.Context, *MsgEndSession) (*MsgEndSessionResponse, error)
	RegisterKey(context.Context, *MsgRegisterKey) (*MsgRegisterKeyResponse, error)
	RevokeKey(context.Context, *MsgRevokeKey) (*MsgRevokeKeyResponse, error)
	ConfigureGuardian(context.Context, *MsgConfigureGuardian) (*MsgConfigureGuardianResponse, error)
	AcknowledgeAlert(context.Context, *MsgAcknowledgeAlert) (*MsgAcknowledgeAlertResponse, error)
	SetSpendingLimit(context.Context, *MsgSetSpendingLimit) (*MsgSetSpendingLimitResponse, error)
}
