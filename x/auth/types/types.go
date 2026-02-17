package types

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// ============================================================
// Validation helpers
// ============================================================

// Validate validates module parameters (extends proto-generated Params).
func (p *Params) Validate() error {
	if p.MaxSessionKeys == 0 {
		return fmt.Errorf("max_session_keys must be > 0")
	}
	if p.MaxSessionDuration == 0 {
		return fmt.Errorf("max_session_duration must be > 0")
	}
	return nil
}

// ValidateDID validates DID format: did:lgm:{hex}
// Accepts 32-char (TypeScript canonical) or 64-char (full pubkey) hex suffixes.
func ValidateDID(did string) error {
	if !strings.HasPrefix(did, "did:lgm:") {
		return fmt.Errorf("DID must start with 'did:lgm:'")
	}
	suffix := strings.TrimPrefix(did, "did:lgm:")
	if len(suffix) != 32 && len(suffix) != 64 {
		return fmt.Errorf("DID suffix must be 32 or 64 hex characters, got %d", len(suffix))
	}
	for _, c := range suffix {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("DID suffix contains non-hex character: %c", c)
		}
	}
	return nil
}

// PublicKeyToDID derives the canonical DID from a hex-encoded Ed25519 public key.
// Matches TypeScript canonical format: did:lgm:{first 32 hex chars of pubkey}
func PublicKeyToDID(pubKeyHex string) string {
	if len(pubKeyHex) < 32 {
		return ""
	}
	return "did:lgm:" + strings.ToLower(pubKeyHex[:32])
}

// ValidateDIDDerivation checks that a DID correctly derives from the given public key.
// Supports both 32-char (canonical) and 64-char (full pubkey) DID suffixes.
func ValidateDIDDerivation(did string, pubKeyHex string) error {
	if len(pubKeyHex) != 64 {
		return fmt.Errorf("public key must be 64 hex characters, got %d", len(pubKeyHex))
	}
	suffix := strings.TrimPrefix(did, "did:lgm:")
	switch len(suffix) {
	case 32:
		expected := strings.ToLower(pubKeyHex[:32])
		if strings.ToLower(suffix) != expected {
			return fmt.Errorf("DID does not derive from public key: expected did:lgm:%s, got %s", expected, did)
		}
	case 64:
		if !strings.EqualFold(suffix, pubKeyHex) {
			return fmt.Errorf("DID does not match full public key: expected did:lgm:%s, got %s", strings.ToLower(pubKeyHex), did)
		}
	default:
		return fmt.Errorf("DID suffix must be 32 or 64 hex characters, got %d", len(suffix))
	}
	return nil
}

// ============================================================
// sdk.Msg interface implementations for proto-generated types
// (ValidateBasic + GetSigners extend the proto types)
// ============================================================

// GetSigners returns the signers for MsgRotateKey
func (msg *MsgRotateKey) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

// ValidateBasic validates MsgRotateKey
func (msg *MsgRotateKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if len(msg.NewOperationalKey) == 0 {
		return fmt.Errorf("new_operational_key cannot be empty")
	}
	if len(msg.AuthorizationSignature) == 0 {
		return fmt.Errorf("authorization_signature cannot be empty")
	}
	return nil
}

// GetSigners returns the signers for MsgCreateSession
func (msg *MsgCreateSession) GetSigners() []sdk.AccAddress {
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{owner}
}

// ValidateBasic validates MsgCreateSession
func (msg *MsgCreateSession) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if len(msg.SessionPubKey) == 0 {
		return fmt.Errorf("session_pub_key cannot be empty")
	}
	return nil
}

// GetSigners returns the signers for MsgRevokeSession
func (msg *MsgRevokeSession) GetSigners() []sdk.AccAddress {
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{owner}
}

// ValidateBasic validates MsgRevokeSession
func (msg *MsgRevokeSession) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.SessionId == "" {
		return fmt.Errorf("session_id cannot be empty")
	}
	return nil
}

// GetSigners returns the signers for MsgUpdateParams
func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic validates MsgUpdateParams
func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Params != nil {
		return msg.Params.Validate()
	}
	return nil
}

// ============================================================
// Hand-written message types NOT in proto (kept as-is)
// ============================================================

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

func (msg MsgRegisterAccount) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

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
	// Verify DID derives from public key (prevents DID spoofing)
	if err := ValidateDIDDerivation(msg.Did, msg.PublicKey); err != nil {
		return fmt.Errorf("DID derivation mismatch: %w", err)
	}
	validTypes := map[string]bool{"agent": true, "human": true, "contract": true, "system": true}
	if !validTypes[msg.AccountType] {
		return fmt.Errorf("account_type must be agent, human, contract, or system")
	}
	return nil
}

func (msg MsgRegisterAccount) Reset() {}
func (msg MsgRegisterAccount) String() string {
	return fmt.Sprintf("MsgRegisterAccount{%s}", msg.Sender)
}
func (msg MsgRegisterAccount) ProtoMessage() {}

func (msg MsgRegisterAccountResponse) Reset() {}
func (msg MsgRegisterAccountResponse) String() string {
	return "MsgRegisterAccountResponse{}"
}
func (msg MsgRegisterAccountResponse) ProtoMessage() {}

var _ sdk.Msg = (*MsgRegisterAccount)(nil)
var _ proto.Message = (*MsgRegisterAccount)(nil)
var _ proto.Message = (*MsgRegisterAccountResponse)(nil)

// MsgFreezeAccount freezes an account (self-freeze or authority)
type MsgFreezeAccount struct {
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
func (msg MsgFreezeAccount) Reset() {}
func (msg MsgFreezeAccount) String() string {
	return fmt.Sprintf("MsgFreezeAccount{%s}", msg.Address)
}
func (msg MsgFreezeAccount) ProtoMessage()  {}
func (msg MsgFreezeAccountResponse) Reset() {}
func (msg MsgFreezeAccountResponse) String() string {
	return "MsgFreezeAccountResponse{}"
}
func (msg MsgFreezeAccountResponse) ProtoMessage() {}

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
func (msg MsgUnfreezeAccount) Reset() {}
func (msg MsgUnfreezeAccount) String() string {
	return fmt.Sprintf("MsgUnfreezeAccount{%s}", msg.Address)
}
func (msg MsgUnfreezeAccount) ProtoMessage()  {}
func (msg MsgUnfreezeAccountResponse) Reset() {}
func (msg MsgUnfreezeAccountResponse) String() string {
	return "MsgUnfreezeAccountResponse{}"
}
func (msg MsgUnfreezeAccountResponse) ProtoMessage() {}

// MsgSetRecoveryConfig sets recovery configuration for an account
type MsgSetRecoveryConfig struct {
	Sender string          `json:"sender"`
	Config *RecoveryConfig `json:"config"`
}
type MsgSetRecoveryConfigResponse struct{}

func (msg *MsgSetRecoveryConfig) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}
func (msg *MsgSetRecoveryConfig) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.Config == nil {
		return fmt.Errorf("config cannot be nil")
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
func (msg *MsgSetRecoveryConfig) Reset() {}
func (msg *MsgSetRecoveryConfig) String() string {
	return fmt.Sprintf("MsgSetRecoveryConfig{%s}", msg.Sender)
}
func (msg *MsgSetRecoveryConfig) ProtoMessage() {}
func (msg MsgSetRecoveryConfigResponse) Reset() {}
func (msg MsgSetRecoveryConfigResponse) String() string {
	return "MsgSetRecoveryConfigResponse{}"
}
func (msg MsgSetRecoveryConfigResponse) ProtoMessage() {}

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
func (msg MsgInitiateRecovery) Reset() {}
func (msg MsgInitiateRecovery) String() string {
	return fmt.Sprintf("MsgInitiateRecovery{%s}", msg.AccountAddress)
}
func (msg MsgInitiateRecovery) ProtoMessage()  {}
func (msg MsgInitiateRecoveryResponse) Reset() {}
func (msg MsgInitiateRecoveryResponse) String() string {
	return "MsgInitiateRecoveryResponse{}"
}
func (msg MsgInitiateRecoveryResponse) ProtoMessage() {}

// MsgSubmitRecoveryShard provides a shard to an active recovery request
type MsgSubmitRecoveryShard struct {
	Sender         string `json:"sender"`          // shard holder address
	AccountAddress string `json:"account_address"` // account being recovered
	ShardIndex     uint32 `json:"shard_index"`     // 1-based shard index
	EncryptedShard []byte `json:"encrypted_shard"` // H-2: encrypted shard data (optional for backward compat)
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
func (msg MsgSubmitRecoveryShard) Reset() {}
func (msg MsgSubmitRecoveryShard) String() string {
	return fmt.Sprintf("MsgSubmitRecoveryShard{shard=%d}", msg.ShardIndex)
}
func (msg MsgSubmitRecoveryShard) ProtoMessage()  {}
func (msg MsgSubmitRecoveryShardResponse) Reset() {}
func (msg MsgSubmitRecoveryShardResponse) String() string {
	return "MsgSubmitRecoveryShardResponse{}"
}
func (msg MsgSubmitRecoveryShardResponse) ProtoMessage() {}

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
func (msg MsgChallengeRecovery) Reset() {}
func (msg MsgChallengeRecovery) String() string {
	return fmt.Sprintf("MsgChallengeRecovery{%s}", msg.AccountAddress)
}
func (msg MsgChallengeRecovery) ProtoMessage()  {}
func (msg MsgChallengeRecoveryResponse) Reset() {}
func (msg MsgChallengeRecoveryResponse) String() string {
	return "MsgChallengeRecoveryResponse{}"
}
func (msg MsgChallengeRecoveryResponse) ProtoMessage() {}

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
func (msg MsgExecuteRecovery) Reset() {}
func (msg MsgExecuteRecovery) String() string {
	return fmt.Sprintf("MsgExecuteRecovery{%s}", msg.AccountAddress)
}
func (msg MsgExecuteRecovery) ProtoMessage()  {}
func (msg MsgExecuteRecoveryResponse) Reset() {}
func (msg MsgExecuteRecoveryResponse) String() string {
	return "MsgExecuteRecoveryResponse{}"
}
func (msg MsgExecuteRecoveryResponse) ProtoMessage() {}

// ============================================================
// Recovery shard storage (H-2: on-chain encrypted shard data)
// ============================================================

// RecoveryShard stores an encrypted Shamir shard on-chain during recovery.
// The shard data is encrypted with the shard holder's key before submission.
type RecoveryShard struct {
	OwnerAddress   string `json:"owner_address"`     // Account being recovered
	ShardIndex     uint32 `json:"shard_index"`       // 1-based shard index
	SubmitterAddr  string `json:"submitter_address"` // Who submitted this shard
	EncryptedData  []byte `json:"encrypted_data"`    // Encrypted shard bytes
	SubmittedBlock uint64 `json:"submitted_block"`   // Block height when submitted
}

func (s RecoveryShard) Reset() {}
func (s RecoveryShard) String() string {
	return fmt.Sprintf("RecoveryShard{%s/%d}", s.OwnerAddress, s.ShardIndex)
}
func (s RecoveryShard) ProtoMessage() {}

// ============================================================
// Extended MsgServer for methods not in the proto service definition
// ============================================================

// ExtendedMsgServer extends the generated MsgServer with additional
// message handlers that are not yet in the proto service definition.
type ExtendedMsgServer interface {
	MsgServer
	RegisterAccount(context.Context, *MsgRegisterAccount) (*MsgRegisterAccountResponse, error)
	FreezeAccount(context.Context, *MsgFreezeAccount) (*MsgFreezeAccountResponse, error)
	UnfreezeAccount(context.Context, *MsgUnfreezeAccount) (*MsgUnfreezeAccountResponse, error)
	SetRecoveryConfig(context.Context, *MsgSetRecoveryConfig) (*MsgSetRecoveryConfigResponse, error)
	InitiateRecovery(context.Context, *MsgInitiateRecovery) (*MsgInitiateRecoveryResponse, error)
	SubmitRecoveryShard(context.Context, *MsgSubmitRecoveryShard) (*MsgSubmitRecoveryShardResponse, error)
	ChallengeRecovery(context.Context, *MsgChallengeRecovery) (*MsgChallengeRecoveryResponse, error)
	ExecuteRecovery(context.Context, *MsgExecuteRecovery) (*MsgExecuteRecoveryResponse, error)
}

// ============================================================
// Interface compliance assertions (non-proto types only)
// ============================================================

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
