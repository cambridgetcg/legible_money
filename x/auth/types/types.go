package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// LegibleAccount represents a Legible Money account with 4-layer key architecture
type LegibleAccount struct {
	Address              string       `json:"address"`                // bech32: lgm1...
	Did                  string       `json:"did"`                    // did:lgm:{32-hex}
	PublicKey            string       `json:"public_key"`             // hex-encoded Ed25519
	AccountType          string       `json:"account_type"`           // agent, human, contract, system
	OperationalKeyHash   string       `json:"operational_key_hash"`   // current operational key
	OperationalKeyVersion uint32      `json:"operational_key_version"` // rotation counter
	SessionKeyCount      uint32       `json:"session_key_count"`      // active session keys
	ReputationScore      uint64       `json:"reputation_score"`       // 0-1000000
	CreatedAtBlock       uint64       `json:"created_at_block"`
	LastActiveBlock      uint64       `json:"last_active_block"`
	Flags                AccountFlags `json:"flags"`
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
	Sender     string `json:"sender"`
	KeyType    string `json:"key_type"`     // "operational" or "session"
	NewKeyHash string `json:"new_key_hash"` // hash of new public key
	Signature  string `json:"signature"`    // signed by old key
}

// MsgRotateKeyResponse is the response for MsgRotateKey
type MsgRotateKeyResponse struct{}

// MsgCreateSession creates a new session key
type MsgCreateSession struct {
	Owner          string              `json:"owner"`
	SessionKeyHash string              `json:"session_key_hash"`
	Capabilities   SessionCapabilities `json:"capabilities"`
	Duration       uint64              `json:"duration"` // blocks
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

// Params defines module parameters
type Params struct {
	MaxSessionKeys       uint32 `json:"max_session_keys"`        // default 5
	MaxSessionDuration   uint64 `json:"max_session_duration"`    // default 34272 (1 day)
	KeyRotationCooldown  uint64 `json:"key_rotation_cooldown"`   // default 111 blocks
}

// GenesisState is the state that must be provided at genesis
type GenesisState struct {
	Params      Params            `json:"params"`
	Accounts    []LegibleAccount  `json:"accounts"`
	DIDMappings []DIDMapping      `json:"did_mappings"`
	SessionKeys []SessionKey      `json:"session_keys"`
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
		MaxSessionKeys:      5,
		MaxSessionDuration:  34272, // 1 day in blocks
		KeyRotationCooldown: 111,
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

// MsgServer interface for gRPC handlers
type MsgServer interface {
	RotateKey(ctx sdk.Context, msg *MsgRotateKey) (*MsgRotateKeyResponse, error)
	CreateSession(ctx sdk.Context, msg *MsgCreateSession) (*MsgCreateSessionResponse, error)
	RevokeSession(ctx sdk.Context, msg *MsgRevokeSession) (*MsgRevokeSessionResponse, error)
}

// Service descriptor stub (normally generated by protoc)
var _Msg_serviceDesc = struct {
	ServiceName string
	HandlerType interface{}
	Methods     []interface{}
	Streams     []interface{}
	Metadata    string
}{
	ServiceName: "legible.auth.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods:     []interface{}{},
	Streams:     []interface{}{},
	Metadata:    "legible/auth/v1/tx.proto",
}

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
