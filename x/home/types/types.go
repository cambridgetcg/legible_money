package types

import (
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
	AlertDeadmanTriggered AlertType = "deadman_triggered"
	AlertSessionExpired   AlertType = "session_expired"
	AlertDefenseActivated AlertType = "defense_activated"
	AlertKeyRevoked       AlertType = "key_revoked"
	AlertSpendingLimitHit AlertType = "spending_limit_hit"
	AlertTokenExpiring    AlertType = "token_expiring"
)

type AlertPriority string

const (
	AlertPriorityLow      AlertPriority = "low"
	AlertPriorityMedium   AlertPriority = "medium"
	AlertPriorityHigh     AlertPriority = "high"
	AlertPriorityCritical AlertPriority = "critical"
)

// --- Params Validation ---

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

// --- ValidateBasic for proto-generated Msg types ---

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

func (m *MsgUpdateHome) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	return nil
}

func (m *MsgUpdateMemoryCID) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	if m.Cid == "" {
		return fmt.Errorf("CID cannot be empty")
	}
	if len(m.Cid) > 128 {
		return fmt.Errorf("CID too long (max 128)")
	}
	return nil
}

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

func (m *MsgConfigureGuardian) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	return nil
}

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

// --- GetSigners ---

func (m *MsgCreateHome) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Owner)
	return []sdk.AccAddress{addr}
}

func (m *MsgUpdateHome) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Owner)
	return []sdk.AccAddress{addr}
}

func (m *MsgUpdateMemoryCID) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Owner)
	return []sdk.AccAddress{addr}
}

func (m *MsgStartSession) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Signer)
	return []sdk.AccAddress{addr}
}

func (m *MsgEndSession) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Signer)
	return []sdk.AccAddress{addr}
}

func (m *MsgRegisterKey) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Owner)
	return []sdk.AccAddress{addr}
}

func (m *MsgRevokeKey) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Owner)
	return []sdk.AccAddress{addr}
}

func (m *MsgConfigureGuardian) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Owner)
	return []sdk.AccAddress{addr}
}

func (m *MsgAcknowledgeAlert) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Signer)
	return []sdk.AccAddress{addr}
}

func (m *MsgSetSpendingLimit) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Owner)
	return []sdk.AccAddress{addr}
}
