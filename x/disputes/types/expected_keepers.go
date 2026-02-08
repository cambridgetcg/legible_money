package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank keeper interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
}

// KnowledgeKeeper defines the expected knowledge keeper interface.
// Used to look up facts and claims that are being disputed.
type KnowledgeKeeper interface {
	// GetFact retrieves a fact by ID from the knowledge module.
	GetFact(ctx sdk.Context, factID string) (Fact, bool)
	// SetFact stores a fact (used to update status during dispute resolution).
	SetFact(ctx sdk.Context, fact Fact)
	// GetClaim retrieves a claim by ID.
	GetClaim(ctx sdk.Context, claimID string) (Claim, bool)
}

// StakingKeeper defines the expected staking keeper interface.
// Used to query validator info for arbiter selection.
type StakingKeeper interface {
	// GetActiveValidatorInfos returns all active validators as ValidatorInfo structs.
	GetActiveValidatorInfos(ctx context.Context) []ValidatorInfo
	// GetValidatorInfo returns info for a single validator by operator address.
	GetValidatorInfo(ctx context.Context, operatorAddr string) (ValidatorInfo, bool)
	// GetEffectiveStake returns the effective stake for a validator.
	GetEffectiveStake(ctx context.Context, operatorAddr string) uint64
}

// Fact is a local mirror of the knowledge module's Fact type.
// Avoids circular imports by defining a compatible subset here.
type Fact struct {
	Id                 string   `json:"id"`
	ClaimText          string   `json:"claim_text"`
	Domain             string   `json:"domain"`
	EpistemicCategory  string   `json:"epistemic_category"`
	Confidence         uint64   `json:"confidence"`
	Status             string   `json:"status"`
	Submitter          string   `json:"submitter"`
	References         []string `json:"references"`
	VerifiedAtHeight   uint64   `json:"verified_at_height"`
	VerifierCount      uint32   `json:"verifier_count"`
	LogicZone          string   `json:"logic_zone"`
	CreatedAtHeight    uint64   `json:"created_at_height"`
	LastVerifiedHeight uint64   `json:"last_verified_height"`
}

func (f *Fact) Reset()        {}
func (f *Fact) String() string { return "" }
func (f *Fact) ProtoMessage()  {}

// Claim is a local mirror of the knowledge module's Claim type.
type Claim struct {
	Id        string `json:"id"`
	Submitter string `json:"submitter"`
	Status    string `json:"status"`
}

func (c *Claim) Reset()        {}
func (c *Claim) String() string { return "" }
func (c *Claim) ProtoMessage()  {}

// ValidatorInfo is a local mirror of the knowledge module's ValidatorInfo type.
type ValidatorInfo struct {
	Address    string `json:"address"`
	Stake      uint64 `json:"stake"`
	Tier       uint32 `json:"tier"`
	PublicKey  []byte `json:"public_key"`
	Reputation uint64 `json:"reputation"`
}

func (vi *ValidatorInfo) Reset()        {}
func (vi *ValidatorInfo) String() string { return "" }
func (vi *ValidatorInfo) ProtoMessage()  {}
