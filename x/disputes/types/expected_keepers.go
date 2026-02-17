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
// Fact mutations use domain-specific methods (MarkFactChallenged, ResolveFactDispute,
// CascadeFactConfidence) instead of raw SetFact to enforce knowledge module invariants.
type KnowledgeKeeper interface {
	// GetFact retrieves a fact by ID from the knowledge module.
	GetFact(ctx sdk.Context, factID string) (Fact, bool)
	// GetClaim retrieves a claim by ID.
	GetClaim(ctx sdk.Context, claimID string) (Claim, bool)
	// MarkFactChallenged transitions a fact to "challenged" status when a dispute is opened.
	MarkFactChallenged(ctx sdk.Context, factId string) error
	// ResolveFactDispute adjusts confidence and status based on dispute outcome.
	// outcome: "overturned", "upheld", or "inconclusive".
	// reductionBps: confidence reduction for overturned outcomes (basis points).
	// Returns the new confidence value.
	ResolveFactDispute(ctx sdk.Context, factId, outcome string, reductionBps uint64) (uint64, error)
	// CascadeFactConfidence reduces confidence of facts referencing a retracted fact.
	// reductionBps: cascade reduction in basis points (e.g. 220_000 = 22%).
	// Returns IDs of affected facts.
	CascadeFactConfidence(ctx sdk.Context, retractedFactId string, reductionBps uint64) ([]string, error)
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

// VestingKeeper defines the expected vesting_rewards keeper interface.
// Used to pause/resume vesting when disputes are opened/resolved.
type VestingKeeper interface {
	// PauseVestingByClaimId pauses the vesting schedule for a claim (if one exists).
	PauseVestingByClaimId(ctx sdk.Context, claimId string) error
	// ResumeVestingByClaimId resumes the vesting schedule for a claim (if one exists).
	ResumeVestingByClaimId(ctx sdk.Context, claimId string) error
	// FalsifyClaim triggers clawback when a claim is proven false via dispute.
	// Returns the clawback record with amounts forfeited/redistributed.
	FalsifyClaim(ctx sdk.Context, claimId string, challenger string) (ClawbackRecord, error)
}

// ClawbackRecord mirrors vesting_rewards types to avoid circular imports.
type ClawbackRecord struct {
	Id                string `json:"id"`
	ClaimId           string `json:"claim_id"`
	ReleasedClawback  string `json:"released_clawback"`
	UnvestedForfeited string `json:"unvested_forfeited"`
	ChallengerReward  string `json:"challenger_reward"`
	BlockHeight       uint64 `json:"block_height"`
}

func (c *ClawbackRecord) Reset()         {}
func (c *ClawbackRecord) String() string { return "" }
func (c *ClawbackRecord) ProtoMessage()  {}

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
	ChallengeCount     uint32   `json:"challenge_count,omitempty"`
}

func (f *Fact) Reset()         {}
func (f *Fact) String() string { return "" }
func (f *Fact) ProtoMessage()  {}

// Claim is a local mirror of the knowledge module's Claim type.
type Claim struct {
	Id        string `json:"id"`
	Submitter string `json:"submitter"`
	Status    string `json:"status"`
}

func (c *Claim) Reset()         {}
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

func (vi *ValidatorInfo) Reset()         {}
func (vi *ValidatorInfo) String() string { return "" }
func (vi *ValidatorInfo) ProtoMessage()  {}
