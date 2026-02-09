package types

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// LegibleAccount represents a Legible Money account with 4-layer key architecture
type LegibleAccount struct {
	Address               string       `json:"address"`                  // bech32: lgm1...
	Did                   string       `json:"did"`                      // did:lgm:{32-hex}
	PublicKey             string       `json:"public_key"`               // hex-encoded Ed25519 identity key
	AccountType           string       `json:"account_type"`             // agent, human, contract, system
	OperationalKeyHash    string       `json:"operational_key_hash"`     // hash of current operational key
	OperationalPublicKey  string       `json:"operational_public_key"`   // hex-encoded Ed25519 operational key
	OperationalKeyVersion uint32       `json:"operational_key_version"`  // rotation counter
	SessionKeyCount       uint32       `json:"session_key_count"`        // active session keys
	ReputationScore       uint64       `json:"reputation_score"`         // 0-1000000
	CreatedAtBlock        uint64       `json:"created_at_block"`
	LastActiveBlock       uint64       `json:"last_active_block"`
	Flags                 AccountFlags `json:"flags"`
}

// AccountFlags contains boolean flags for account capabilities and state
type AccountFlags struct {
	IsValidator      bool `json:"is_validator"`
	CanSubmitClaims  bool `json:"can_submit_claims"`
	CanChallenge     bool `json:"can_challenge"`
	InRecovery       bool `json:"in_recovery"`
	Frozen           bool `json:"frozen"`
}

// SessionKey represents an ephemeral key with limited capabilities
type SessionKey struct {
	KeyHash         string              `json:"key_hash"`
	PublicKey       string              `json:"public_key"`      // hex-encoded Ed25519 session public key
	Owner           string              `json:"owner"`           // bech32 address
	Capabilities    SessionCapabilities `json:"capabilities"`
	ExpiresAtBlock  uint64              `json:"expires_at_block"`
	CreatedAtBlock  uint64              `json:"created_at_block"`
}

// SessionCapabilities defines what a session key can do
type SessionCapabilities struct {
	CanTransfer       bool     `json:"can_transfer"`
	CanStake          bool     `json:"can_stake"`
	CanSubmitClaims   bool     `json:"can_submit_claims"`
	CanVote           bool     `json:"can_vote"`
	MaxTransferAmount string   `json:"max_transfer_amount"` // sdk.Int as string
	AllowedRecipients []string `json:"allowed_recipients"`  // empty = all
}

// DIDMapping maps DID to bech32 address
type DIDMapping struct {
	Did    string `json:"did"`     // did:lgm:{32-hex}
	Bech32 string `json:"bech32"`  // lgm1...
	PubKey string `json:"pub_key"` // hex-encoded Ed25519
}

// MsgRotateKey rotates operational or session key
type MsgRotateKey struct {
	Sender       string `json:"sender"`
	KeyType      string `json:"key_type"`       // "operational" or "session"
	NewKeyHash   string `json:"new_key_hash"`   // hash of new public key
	NewPublicKey string `json:"new_public_key"` // hex-encoded Ed25519 new public key
	Signature    string `json:"signature"`       // signed by old key
}

// MsgRotateKeyResponse is the response for MsgRotateKey
type MsgRotateKeyResponse struct{}

// MsgCreateSession creates a new session key
type MsgCreateSession struct {
	Owner             string              `json:"owner"`
	SessionKeyHash    string              `json:"session_key_hash"`
	SessionPublicKey  string              `json:"session_public_key"` // hex-encoded Ed25519 session public key
	Capabilities      SessionCapabilities `json:"capabilities"`
	Duration          uint64              `json:"duration"` // blocks
}

// MsgCreateSessionResponse is the response for MsgCreateSession
type MsgCreateSessionResponse struct {
	ExpiresAtBlock uint64 `json:"expires_at_block"`
}

// MsgRevokeSession revokes a session key
type MsgRevokeSession struct {
	Owner          string `json:"owner"`
	SessionKeyHash string `json:"session_key_hash"`
}

// MsgRevokeSessionResponse is the response for MsgRevokeSession
type MsgRevokeSessionResponse struct{}

// MsgRegisterAccount registers a new Legible account with DID mapping
type MsgRegisterAccount struct {
	Sender             string `json:"sender"`               // bech32 address
	Did                string `json:"did"`                  // did:lgm:{32-hex}
	PublicKey          string `json:"public_key"`           // hex-encoded Ed25519 identity key
	AccountType        string `json:"account_type"`         // agent, human, contract
	OperationalKeyHash string `json:"operational_key_hash"` // initial operational key hash
}

// MsgRegisterAccountResponse is the response for MsgRegisterAccount
type MsgRegisterAccountResponse struct{}

// RecoveryConfig defines the recovery configuration for an account
type RecoveryConfig struct {
	RecoveryKeyHash       string        `json:"recovery_key_hash"`        // hash of recovery public key
	Threshold             uint32        `json:"threshold"`                // min shards needed
	TotalShards           uint32        `json:"total_shards"`             // total shards created
	ShardHolders          []ShardHolder `json:"shard_holders"`            // who holds each shard
	RecoveryDelayBlocks   uint64        `json:"recovery_delay_blocks"`    // blocks to wait before executable
	ChallengePeriodBlocks uint64        `json:"challenge_period_blocks"`  // blocks during which recovery can be challenged
}

// ShardHolder identifies who holds a recovery shard
type ShardHolder struct {
	Type               string `json:"type"`                  // "agent", "operator", "guardian"
	Identifier         string `json:"identifier"`            // DID or bech32 address
	ShardIndex         uint32 `json:"shard_index"`           // 1-based shard index
	CanInitiateRecovery bool  `json:"can_initiate_recovery"`
}

// RecoveryRequest represents an in-progress recovery
type RecoveryRequest struct {
	AccountAddress     string `json:"account_address"`       // account being recovered
	InitiatedBy        string `json:"initiated_by"`          // address of initiator
	NewOperationalKey  string `json:"new_operational_key"`   // hex-encoded new Ed25519 pubkey
	ShardsProvided     []uint32 `json:"shards_provided"`     // indices of provided shards
	ShardsRequired     uint32 `json:"shards_required"`       // threshold
	InitiatedAtBlock   uint64 `json:"initiated_at_block"`
	DelayExpiresAt     uint64 `json:"delay_expires_at"`      // block when delay ends
	ChallengeExpiresAt uint64 `json:"challenge_expires_at"`  // block when challenge period ends
	Status             string `json:"status"`                // pending, delayed, challengeable, executable, completed, cancelled
	ChallengerAddress  string `json:"challenger_address,omitempty"`
	ChallengeReason    string `json:"challenge_reason,omitempty"`
}

func (rc RecoveryConfig) Reset()         {}
func (rc RecoveryConfig) String() string { return "RecoveryConfig{}" }
func (rc RecoveryConfig) ProtoMessage()  {}

func (rr RecoveryRequest) Reset()         {}
func (rr RecoveryRequest) String() string { return fmt.Sprintf("RecoveryRequest{%s}", rr.AccountAddress) }
func (rr RecoveryRequest) ProtoMessage()  {}

// Params defines module parameters
type Params struct {
	MaxSessionKeys        uint32 `json:"max_session_keys"`         // default 5
	MaxSessionDuration    uint64 `json:"max_session_duration"`     // default 34272 (1 day)
	KeyRotationCooldown   uint64 `json:"key_rotation_cooldown"`    // default 111 blocks
	RecoveryDelayBlocks   uint64 `json:"recovery_delay_blocks"`    // default 1000 (~42 min)
	ChallengePeriodBlocks uint64 `json:"challenge_period_blocks"`  // default 500 (~21 min)
}

// GenesisState is the state that must be provided at genesis
type GenesisState struct {
	Params          Params            `json:"params"`
	Accounts        []LegibleAccount  `json:"accounts"`
	DIDMappings     []DIDMapping      `json:"did_mappings"`
	SessionKeys     []SessionKey      `json:"session_keys"`
	RecoveryConfigs []RecoveryConfig  `json:"recovery_configs,omitempty"`
}

// DefaultGenesis returns default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		Accounts:    []LegibleAccount{},
		DIDMappings: []DIDMapping{},
		SessionKeys: []SessionKey{},
	}
}

// DefaultParams returns default module parameters
func DefaultParams() Params {
	return Params{
		MaxSessionKeys:        5,
		MaxSessionDuration:    34272, // 1 day in blocks
		KeyRotationCooldown:   111,
		RecoveryDelayBlocks:   1000,  // ~42 min
		ChallengePeriodBlocks: 500,   // ~21 min
	}
}

// Validate validates genesis state
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Validate accounts
	seenAddrs := make(map[string]bool)
	seenDIDs := make(map[string]bool)
	for _, acc := range gs.Accounts {
		if seenAddrs[acc.Address] {
			return fmt.Errorf("duplicate address: %s", acc.Address)
		}
		seenAddrs[acc.Address] = true

		if seenDIDs[acc.Did] {
			return fmt.Errorf("duplicate DID: %s", acc.Did)
		}
		seenDIDs[acc.Did] = true

		if err := ValidateDID(acc.Did); err != nil {
			return fmt.Errorf("invalid DID for account %s: %w", acc.Address, err)
		}
	}

	// Validate DID mappings
	for _, mapping := range gs.DIDMappings {
		if err := ValidateDID(mapping.Did); err != nil {
			return fmt.Errorf("invalid DID in mapping: %w", err)
		}
	}

	return nil
}

// Validate validates module parameters
func (p Params) Validate() error {
	if p.MaxSessionKeys == 0 {
		return fmt.Errorf("max_session_keys must be > 0")
	}
	if p.MaxSessionDuration == 0 {
		return fmt.Errorf("max_session_duration must be > 0")
	}
	return nil
}

// ValidateDID validates DID format: did:lgm:{32-hex}
func ValidateDID(did string) error {
	if !strings.HasPrefix(did, "did:lgm:") {
		return fmt.Errorf("DID must start with 'did:lgm:'")
	}
	suffix := strings.TrimPrefix(did, "did:lgm:")
	if len(suffix) != 64 { // 32 bytes = 64 hex chars
		return fmt.Errorf("DID suffix must be 64 hex characters")
	}
	// Could add hex validation here
	return nil
}

// sdk.Msg interface implementations

// GetSigners returns the signers for MsgRotateKey
func (msg MsgRotateKey) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

// ValidateBasic validates MsgRotateKey
func (msg MsgRotateKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.KeyType != "operational" && msg.KeyType != "session" {
		return fmt.Errorf("key_type must be 'operational' or 'session'")
	}
	if msg.NewKeyHash == "" {
		return fmt.Errorf("new_key_hash cannot be empty")
	}
	if msg.Signature == "" {
		return fmt.Errorf("signature cannot be empty")
	}
	return nil
}

// GetSigners returns the signers for MsgCreateSession
func (msg MsgCreateSession) GetSigners() []sdk.AccAddress {
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{owner}
}

// ValidateBasic validates MsgCreateSession
func (msg MsgCreateSession) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.SessionKeyHash == "" {
		return fmt.Errorf("session_key_hash cannot be empty")
	}
	if msg.Duration == 0 {
		return fmt.Errorf("duration must be > 0")
	}
	return nil
}

// GetSigners returns the signers for MsgRevokeSession
func (msg MsgRevokeSession) GetSigners() []sdk.AccAddress {
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{owner}
}

// ValidateBasic validates MsgRevokeSession
func (msg MsgRevokeSession) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.SessionKeyHash == "" {
		return fmt.Errorf("session_key_hash cannot be empty")
	}
	return nil
}

// proto.Message stub implementations (for compatibility)
func (msg MsgRotateKey) Reset()         {}
func (msg MsgRotateKey) String() string { return fmt.Sprintf("MsgRotateKey{%s}", msg.Sender) }
func (msg MsgRotateKey) ProtoMessage()  {}

func (msg MsgRotateKeyResponse) Reset()         {}
func (msg MsgRotateKeyResponse) String() string { return "MsgRotateKeyResponse{}" }
func (msg MsgRotateKeyResponse) ProtoMessage()  {}

func (msg MsgCreateSession) Reset()         {}
func (msg MsgCreateSession) String() string { return fmt.Sprintf("MsgCreateSession{%s}", msg.Owner) }
func (msg MsgCreateSession) ProtoMessage()  {}

func (msg MsgCreateSessionResponse) Reset()         {}
func (msg MsgCreateSessionResponse) String() string { return "MsgCreateSessionResponse{}" }
func (msg MsgCreateSessionResponse) ProtoMessage()  {}

func (msg MsgRevokeSession) Reset()         {}
func (msg MsgRevokeSession) String() string { return fmt.Sprintf("MsgRevokeSession{%s}", msg.Owner) }
func (msg MsgRevokeSession) ProtoMessage()  {}

func (msg MsgRevokeSessionResponse) Reset()         {}
func (msg MsgRevokeSessionResponse) String() string { return "MsgRevokeSessionResponse{}" }
func (msg MsgRevokeSessionResponse) ProtoMessage()  {}

// MsgFreezeAccount freezes an account (self-freeze or authority)
type MsgFreezeAccount struct {
	// Sender is the account owner (self-freeze) or module authority
	Sender  string `json:"sender"`
	Address string `json:"address"` // account to freeze
	Reason  string `json:"reason"`
}
type MsgFreezeAccountResponse struct{}

func (msg MsgFreezeAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}
func (msg MsgFreezeAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}
	return nil
}
func (msg MsgFreezeAccount) Reset()         {}
func (msg MsgFreezeAccount) String() string { return fmt.Sprintf("MsgFreezeAccount{%s}", msg.Address) }
func (msg MsgFreezeAccount) ProtoMessage()  {}
func (msg MsgFreezeAccountResponse) Reset()         {}
func (msg MsgFreezeAccountResponse) String() string { return "MsgFreezeAccountResponse{}" }
func (msg MsgFreezeAccountResponse) ProtoMessage()  {}

// MsgUnfreezeAccount unfreezes a frozen account (authority only)
type MsgUnfreezeAccount struct {
	Authority string `json:"authority"` // module authority (gov)
	Address   string `json:"address"`   // account to unfreeze
}
type MsgUnfreezeAccountResponse struct{}

func (msg MsgUnfreezeAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{sender}
}
func (msg MsgUnfreezeAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}
	return nil
}
func (msg MsgUnfreezeAccount) Reset()         {}
func (msg MsgUnfreezeAccount) String() string { return fmt.Sprintf("MsgUnfreezeAccount{%s}", msg.Address) }
func (msg MsgUnfreezeAccount) ProtoMessage()  {}
func (msg MsgUnfreezeAccountResponse) Reset()         {}
func (msg MsgUnfreezeAccountResponse) String() string { return "MsgUnfreezeAccountResponse{}" }
func (msg MsgUnfreezeAccountResponse) ProtoMessage()  {}

// MsgSetRecoveryConfig sets recovery configuration for an account
type MsgSetRecoveryConfig struct {
	Sender string         `json:"sender"`
	Config RecoveryConfig `json:"config"`
}
type MsgSetRecoveryConfigResponse struct{}

func (msg MsgSetRecoveryConfig) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}
func (msg MsgSetRecoveryConfig) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.Config.Threshold < 1 {
		return fmt.Errorf("threshold must be >= 1")
	}
	if msg.Config.TotalShards < msg.Config.Threshold {
		return fmt.Errorf("total_shards must be >= threshold")
	}
	if len(msg.Config.ShardHolders) != int(msg.Config.TotalShards) {
		return fmt.Errorf("shard_holders count must equal total_shards")
	}
	return nil
}
func (msg MsgSetRecoveryConfig) Reset()         {}
func (msg MsgSetRecoveryConfig) String() string { return fmt.Sprintf("MsgSetRecoveryConfig{%s}", msg.Sender) }
func (msg MsgSetRecoveryConfig) ProtoMessage()  {}
func (msg MsgSetRecoveryConfigResponse) Reset()         {}
func (msg MsgSetRecoveryConfigResponse) String() string { return "MsgSetRecoveryConfigResponse{}" }
func (msg MsgSetRecoveryConfigResponse) ProtoMessage()  {}

// MsgInitiateRecovery begins account recovery
type MsgInitiateRecovery struct {
	Sender            string `json:"sender"`              // initiator address
	AccountAddress    string `json:"account_address"`     // account being recovered
	NewOperationalKey string `json:"new_operational_key"` // hex-encoded Ed25519 pubkey
}
type MsgInitiateRecoveryResponse struct{}

func (msg MsgInitiateRecovery) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}
func (msg MsgInitiateRecovery) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.AccountAddress); err != nil {
		return fmt.Errorf("invalid account address: %w", err)
	}
	if msg.NewOperationalKey == "" || len(msg.NewOperationalKey) != 64 {
		return fmt.Errorf("new_operational_key must be 64 hex characters")
	}
	return nil
}
func (msg MsgInitiateRecovery) Reset()         {}
func (msg MsgInitiateRecovery) String() string { return fmt.Sprintf("MsgInitiateRecovery{%s}", msg.AccountAddress) }
func (msg MsgInitiateRecovery) ProtoMessage()  {}
func (msg MsgInitiateRecoveryResponse) Reset()         {}
func (msg MsgInitiateRecoveryResponse) String() string { return "MsgInitiateRecoveryResponse{}" }
func (msg MsgInitiateRecoveryResponse) ProtoMessage()  {}

// MsgSubmitRecoveryShard provides a shard to an active recovery request
type MsgSubmitRecoveryShard struct {
	Sender         string `json:"sender"`          // shard holder address
	AccountAddress string `json:"account_address"` // account being recovered
	ShardIndex     uint32 `json:"shard_index"`     // 1-based shard index
}
type MsgSubmitRecoveryShardResponse struct{}

func (msg MsgSubmitRecoveryShard) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}
func (msg MsgSubmitRecoveryShard) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.AccountAddress); err != nil {
		return fmt.Errorf("invalid account address: %w", err)
	}
	if msg.ShardIndex == 0 {
		return fmt.Errorf("shard_index must be >= 1")
	}
	return nil
}
func (msg MsgSubmitRecoveryShard) Reset()         {}
func (msg MsgSubmitRecoveryShard) String() string { return fmt.Sprintf("MsgSubmitRecoveryShard{shard=%d}", msg.ShardIndex) }
func (msg MsgSubmitRecoveryShard) ProtoMessage()  {}
func (msg MsgSubmitRecoveryShardResponse) Reset()         {}
func (msg MsgSubmitRecoveryShardResponse) String() string { return "MsgSubmitRecoveryShardResponse{}" }
func (msg MsgSubmitRecoveryShardResponse) ProtoMessage()  {}

// MsgChallengeRecovery challenges an active recovery request
type MsgChallengeRecovery struct {
	Sender         string `json:"sender"`          // challenger (typically account owner)
	AccountAddress string `json:"account_address"` // account being recovered
	Reason         string `json:"reason"`
}
type MsgChallengeRecoveryResponse struct{}

func (msg MsgChallengeRecovery) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}
func (msg MsgChallengeRecovery) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.AccountAddress); err != nil {
		return fmt.Errorf("invalid account address: %w", err)
	}
	if msg.Reason == "" {
		return fmt.Errorf("reason cannot be empty")
	}
	return nil
}
func (msg MsgChallengeRecovery) Reset()         {}
func (msg MsgChallengeRecovery) String() string { return fmt.Sprintf("MsgChallengeRecovery{%s}", msg.AccountAddress) }
func (msg MsgChallengeRecovery) ProtoMessage()  {}
func (msg MsgChallengeRecoveryResponse) Reset()         {}
func (msg MsgChallengeRecoveryResponse) String() string { return "MsgChallengeRecoveryResponse{}" }
func (msg MsgChallengeRecoveryResponse) ProtoMessage()  {}

// MsgExecuteRecovery completes a recovery after delay + challenge period
type MsgExecuteRecovery struct {
	Sender         string `json:"sender"`          // anyone can execute once executable
	AccountAddress string `json:"account_address"` // account being recovered
}
type MsgExecuteRecoveryResponse struct{}

func (msg MsgExecuteRecovery) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}
func (msg MsgExecuteRecovery) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.AccountAddress); err != nil {
		return fmt.Errorf("invalid account address: %w", err)
	}
	return nil
}
func (msg MsgExecuteRecovery) Reset()         {}
func (msg MsgExecuteRecovery) String() string { return fmt.Sprintf("MsgExecuteRecovery{%s}", msg.AccountAddress) }
func (msg MsgExecuteRecovery) ProtoMessage()  {}
func (msg MsgExecuteRecoveryResponse) Reset()         {}
func (msg MsgExecuteRecoveryResponse) String() string { return "MsgExecuteRecoveryResponse{}" }
func (msg MsgExecuteRecoveryResponse) ProtoMessage()  {}

// MsgServer interface for gRPC handlers
type MsgServer interface {
	RegisterAccount(context.Context, *MsgRegisterAccount) (*MsgRegisterAccountResponse, error)
	RotateKey(context.Context, *MsgRotateKey) (*MsgRotateKeyResponse, error)
	CreateSession(context.Context, *MsgCreateSession) (*MsgCreateSessionResponse, error)
	RevokeSession(context.Context, *MsgRevokeSession) (*MsgRevokeSessionResponse, error)
	FreezeAccount(context.Context, *MsgFreezeAccount) (*MsgFreezeAccountResponse, error)
	UnfreezeAccount(context.Context, *MsgUnfreezeAccount) (*MsgUnfreezeAccountResponse, error)
	SetRecoveryConfig(context.Context, *MsgSetRecoveryConfig) (*MsgSetRecoveryConfigResponse, error)
	InitiateRecovery(context.Context, *MsgInitiateRecovery) (*MsgInitiateRecoveryResponse, error)
	SubmitRecoveryShard(context.Context, *MsgSubmitRecoveryShard) (*MsgSubmitRecoveryShardResponse, error)
	ChallengeRecovery(context.Context, *MsgChallengeRecovery) (*MsgChallengeRecoveryResponse, error)
	ExecuteRecovery(context.Context, *MsgExecuteRecovery) (*MsgExecuteRecoveryResponse, error)
}

// sdk.Msg interface implementations for MsgRegisterAccount

// GetSigners returns the signers for MsgRegisterAccount
func (msg MsgRegisterAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

// ValidateBasic validates MsgRegisterAccount
func (msg MsgRegisterAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if err := ValidateDID(msg.Did); err != nil {
		return fmt.Errorf("invalid DID: %w", err)
	}
	if msg.PublicKey == "" {
		return fmt.Errorf("public_key cannot be empty")
	}
	if len(msg.PublicKey) != 64 {
		return fmt.Errorf("public_key must be 64 hex characters (32 bytes Ed25519)")
	}
	validTypes := map[string]bool{"agent": true, "human": true, "contract": true, "system": true}
	if !validTypes[msg.AccountType] {
		return fmt.Errorf("account_type must be agent, human, contract, or system")
	}
	return nil
}

func (msg MsgRegisterAccount) Reset()         {}
func (msg MsgRegisterAccount) String() string { return fmt.Sprintf("MsgRegisterAccount{%s}", msg.Sender) }
func (msg MsgRegisterAccount) ProtoMessage()  {}

func (msg MsgRegisterAccountResponse) Reset()         {}
func (msg MsgRegisterAccountResponse) String() string { return "MsgRegisterAccountResponse{}" }
func (msg MsgRegisterAccountResponse) ProtoMessage()  {}

var _ sdk.Msg = (*MsgRegisterAccount)(nil)
var _ proto.Message = (*MsgRegisterAccount)(nil)
var _ proto.Message = (*MsgRegisterAccountResponse)(nil)

// proto.Message implementations for data types
func (acc LegibleAccount) Reset()         {}
func (acc LegibleAccount) String() string { return fmt.Sprintf("LegibleAccount{%s}", acc.Address) }
func (acc LegibleAccount) ProtoMessage()  {}

func (f AccountFlags) Reset()         {}
func (f AccountFlags) String() string { return "AccountFlags{}" }
func (f AccountFlags) ProtoMessage()  {}

func (sk SessionKey) Reset()         {}
func (sk SessionKey) String() string { return fmt.Sprintf("SessionKey{%s}", sk.KeyHash) }
func (sk SessionKey) ProtoMessage()  {}

func (sc SessionCapabilities) Reset()         {}
func (sc SessionCapabilities) String() string { return "SessionCapabilities{}" }
func (sc SessionCapabilities) ProtoMessage()  {}

func (dm DIDMapping) Reset()         {}
func (dm DIDMapping) String() string { return fmt.Sprintf("DIDMapping{%s}", dm.Did) }
func (dm DIDMapping) ProtoMessage()  {}

func (p Params) Reset()         {}
func (p Params) String() string { return "Params{}" }
func (p Params) ProtoMessage()  {}

func (gs GenesisState) Reset()         {}
func (gs GenesisState) String() string { return "GenesisState{}" }
func (gs GenesisState) ProtoMessage()  {}

// Ensure messages implement proto.Message
var _ proto.Message = (*LegibleAccount)(nil)
var _ proto.Message = (*AccountFlags)(nil)
var _ proto.Message = (*SessionKey)(nil)
var _ proto.Message = (*SessionCapabilities)(nil)
var _ proto.Message = (*DIDMapping)(nil)
var _ proto.Message = (*Params)(nil)
var _ proto.Message = (*GenesisState)(nil)
var _ proto.Message = (*MsgRotateKey)(nil)
var _ proto.Message = (*MsgRotateKeyResponse)(nil)
var _ proto.Message = (*MsgCreateSession)(nil)
var _ proto.Message = (*MsgCreateSessionResponse)(nil)
var _ proto.Message = (*MsgRevokeSession)(nil)
var _ proto.Message = (*MsgRevokeSessionResponse)(nil)

// Ensure messages implement sdk.Msg
var _ sdk.Msg = (*MsgRotateKey)(nil)
var _ sdk.Msg = (*MsgCreateSession)(nil)
var _ sdk.Msg = (*MsgRevokeSession)(nil)
var _ sdk.Msg = (*MsgFreezeAccount)(nil)
var _ sdk.Msg = (*MsgUnfreezeAccount)(nil)
var _ sdk.Msg = (*MsgSetRecoveryConfig)(nil)
var _ sdk.Msg = (*MsgInitiateRecovery)(nil)
var _ sdk.Msg = (*MsgSubmitRecoveryShard)(nil)
var _ sdk.Msg = (*MsgChallengeRecovery)(nil)
var _ sdk.Msg = (*MsgExecuteRecovery)(nil)

var _ proto.Message = (*MsgFreezeAccount)(nil)
var _ proto.Message = (*MsgFreezeAccountResponse)(nil)
var _ proto.Message = (*MsgUnfreezeAccount)(nil)
var _ proto.Message = (*MsgUnfreezeAccountResponse)(nil)
var _ proto.Message = (*MsgSetRecoveryConfig)(nil)
var _ proto.Message = (*MsgSetRecoveryConfigResponse)(nil)
var _ proto.Message = (*MsgInitiateRecovery)(nil)
var _ proto.Message = (*MsgInitiateRecoveryResponse)(nil)
var _ proto.Message = (*MsgSubmitRecoveryShard)(nil)
var _ proto.Message = (*MsgSubmitRecoveryShardResponse)(nil)
var _ proto.Message = (*MsgChallengeRecovery)(nil)
var _ proto.Message = (*MsgChallengeRecoveryResponse)(nil)
var _ proto.Message = (*MsgExecuteRecovery)(nil)
var _ proto.Message = (*MsgExecuteRecoveryResponse)(nil)
var _ proto.Message = (*RecoveryConfig)(nil)
var _ proto.Message = (*RecoveryRequest)(nil)
