package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// VestingCategory determines release curve parameters
// Each category has different half-life based on knowledge stability research
type VestingCategory string

const (
	CategoryAxiomatic    VestingCategory = "axiomatic"     // Mathematical axioms
	CategoryFormalProof  VestingCategory = "formal_proof"  // Verified proofs
	CategoryOnChain      VestingCategory = "on_chain"      // Blockchain-verifiable
	CategoryCryptographic VestingCategory = "cryptographic" // Cryptographic proofs
	CategoryComputational VestingCategory = "computational" // Reproducible computation
	CategoryPeerReviewed VestingCategory = "peer_reviewed"  // Published research
	CategoryReplicated   VestingCategory = "replicated"     // IRL-replicated
	CategoryOracleFeed   VestingCategory = "oracle_feed"    // Oracle price feeds
	CategoryAttestation  VestingCategory = "attestation"    // Human attestations
	CategoryContested    VestingCategory = "contested"      // Claims with active disputes
)

// VestingStatus tracks the lifecycle of a vesting schedule
type VestingStatus string

const (
	VestingStatusActive    VestingStatus = "vesting"    // Active, releasing rewards
	VestingStatusPaused    VestingStatus = "paused"     // Paused due to challenge
	VestingStatusCompleted VestingStatus = "completed"  // Reached max release
	VestingStatusFalsified VestingStatus = "falsified"  // Claim proven false
	VestingStatusAbandoned VestingStatus = "abandoned"  // Recipient abandoned
)

// RewardSource identifies where a reward originated
type RewardSource string

const (
	SourceBlockProduction RewardSource = "block_production"
	SourceVerification    RewardSource = "verification"
	SourceAPICall         RewardSource = "api_call"
	SourceCitation        RewardSource = "citation"
	SourceChallengeWin    RewardSource = "challenge_win"
	SourceChallengeDefense RewardSource = "challenge_defense"
	SourceBountyFulfillment RewardSource = "bounty_fulfillment"
)

// CategoryConfig defines release curve parameters for a vesting category
type CategoryConfig struct {
	Category       VestingCategory
	HalfLifeBlocks uint64 // blocks until half of reward is released
	CliffBlocks    uint64 // blocks before any release begins
	MaxRelease     uint64 // basis points (950000 = 95%, rest is permanent reserve)
}

func (c *CategoryConfig) Reset()         { *c = CategoryConfig{} }
func (c *CategoryConfig) String() string { return proto.CompactTextString(c) }
func (c *CategoryConfig) ProtoMessage()  {}

// VestingSchedule represents a truth-linked reward vesting record
type VestingSchedule struct {
	Id               string          // unique vesting ID
	ClaimId          string          // the claim this reward is for
	FactId           string          // the fact (once claim is accepted)
	Recipient        string          // bech32 address of reward recipient
	TotalAmount      string          // total reward amount (ulgm)
	ReleasedAmount   string          // amount already released
	ClaimableAmount  string          // amount currently claimable
	ReserveAmount    string          // permanent challenge reserve
	Category         VestingCategory // epistemic category
	Source           RewardSource    // reward origin
	Status           VestingStatus   // current status
	AcceptedAtBlock  uint64          // when claim was accepted
	CliffEndsAtBlock uint64          // when cliff period ends
	LastClaimBlock   uint64          // last block rewards were claimed
	TotalPausedBlocks uint64         // total blocks spent paused
	PausedAtBlock    uint64          // block when paused (0 if not paused)
	DefenseCount     uint32          // successful defenses (accelerates)
	ReplicationCount uint32          // independent replications (accelerates)
	CreatedAt        uint64          // block height
	UpdatedAt        uint64          // block height
}

func (v *VestingSchedule) Reset()         { *v = VestingSchedule{} }
func (v *VestingSchedule) String() string { return proto.CompactTextString(v) }
func (v *VestingSchedule) ProtoMessage()  {}

// RewardRouting represents a routed reward with research share deducted
type RewardRouting struct {
	Source          RewardSource // reward source
	OriginalAmount  string      // original reward amount
	ResearchShare   string      // 22% to research fund
	RecipientAmount string      // 78% to recipient
	Recipient       string      // bech32 recipient address
	FactId          string      // related fact
	BlockNumber     uint64      // block height
}

func (r *RewardRouting) Reset()         { *r = RewardRouting{} }
func (r *RewardRouting) String() string { return proto.CompactTextString(r) }
func (r *RewardRouting) ProtoMessage()  {}

// BlockRewardDistribution records block reward distribution
type BlockRewardDistribution struct {
	BlockHeight      uint64 // block number
	ProducerReward   string // reward to block producer
	ResearchShare    string // 22% to research fund
	TotalMinted      string // total new tokens
	ValidatorCount   uint32 // active validators
}

func (b *BlockRewardDistribution) Reset()         { *b = BlockRewardDistribution{} }
func (b *BlockRewardDistribution) String() string { return proto.CompactTextString(b) }
func (b *BlockRewardDistribution) ProtoMessage()  {}

// ClawbackRecord records a reward clawback due to falsification
type ClawbackRecord struct {
	Id               string // unique record ID
	VestingId        string // vesting schedule being clawed back
	ReleasedClawback string // amount clawed from already-released
	UnvestedForfeited string // unvested amount forfeited
	ReserveForfeited string // reserve forfeited
	ChallengerReward string // amount given to challenger
	BlockHeight      uint64 // when clawback occurred
}

func (c *ClawbackRecord) Reset()         { *c = ClawbackRecord{} }
func (c *ClawbackRecord) String() string { return proto.CompactTextString(c) }
func (c *ClawbackRecord) ProtoMessage()  {}

// Params for the vesting_rewards module
type Params struct {
	ResearchShareBps        uint64 // basis points (2200 = 22%)
	ResearchTreasuryAddress string // bech32 address for research fund
	BlockRewardAmount       string // base block reward in ulgm
	EmptyBlockRewardRate    uint64 // basis points of reward for empty blocks (0 = none)
	MinValidatorsForFullReward uint32 // target validator count for full block reward
	VestingEnabled          bool   // whether truth-linked vesting is active
	ReleasedClawbackRate    uint64 // basis points of released clawed back on falsification
}

func (p *Params) Reset()         { *p = Params{} }
func (p *Params) String() string { return proto.CompactTextString(p) }
func (p *Params) ProtoMessage()  {}

// GenesisState defines the vesting_rewards module's genesis state
type GenesisState struct {
	Params           Params
	CategoryConfigs  []CategoryConfig
	VestingSchedules []VestingSchedule
}

func (gs *GenesisState) Reset()         { *gs = GenesisState{} }
func (gs *GenesisState) String() string { return proto.CompactTextString(gs) }
func (gs *GenesisState) ProtoMessage()  {}

// --- Message Types ---

// MsgClaimVestedRewards claims available vested rewards
type MsgClaimVestedRewards struct {
	Sender    string // bech32 address
	VestingId string // specific vesting ID (empty = claim all)
}

func (m *MsgClaimVestedRewards) Reset()         { *m = MsgClaimVestedRewards{} }
func (m *MsgClaimVestedRewards) String() string { return proto.CompactTextString(m) }
func (m *MsgClaimVestedRewards) ProtoMessage()  {}

func (m *MsgClaimVestedRewards) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	return nil
}

func (m *MsgClaimVestedRewards) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{addr}
}

// MsgClaimVestedRewardsResponse is the response
type MsgClaimVestedRewardsResponse struct {
	ClaimedAmount string
	ScheduleCount uint32
}

func (m *MsgClaimVestedRewardsResponse) Reset()         { *m = MsgClaimVestedRewardsResponse{} }
func (m *MsgClaimVestedRewardsResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgClaimVestedRewardsResponse) ProtoMessage()  {}

// MsgFalsifyClaim triggers clawback for a falsified claim
type MsgFalsifyClaim struct {
	Authority string // module authority (governance)
	ClaimId   string // the falsified claim
	Challenger string // bech32 address of successful challenger
	Reason    string // reason for falsification
}

func (m *MsgFalsifyClaim) Reset()         { *m = MsgFalsifyClaim{} }
func (m *MsgFalsifyClaim) String() string { return proto.CompactTextString(m) }
func (m *MsgFalsifyClaim) ProtoMessage()  {}

func (m *MsgFalsifyClaim) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	if m.ClaimId == "" {
		return fmt.Errorf("claim ID required")
	}
	if m.Challenger == "" {
		return fmt.Errorf("challenger address required")
	}
	return nil
}

func (m *MsgFalsifyClaim) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// MsgFalsifyClaimResponse is the response
type MsgFalsifyClaimResponse struct {
	ClawbackAmount string
	ChallengerReward string
}

func (m *MsgFalsifyClaimResponse) Reset()         { *m = MsgFalsifyClaimResponse{} }
func (m *MsgFalsifyClaimResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgFalsifyClaimResponse) ProtoMessage()  {}

// MsgCreateVesting creates a new truth-linked vesting schedule.
// Only callable by authority (system/governance).
type MsgCreateVesting struct {
	Authority   string          // module authority
	ClaimId     string          // the claim this reward is for
	FactId      string          // the fact (once claim is accepted)
	Recipient   string          // bech32 address of reward recipient
	TotalAmount string          // total reward amount (ulgm)
	Category    VestingCategory // epistemic category
	Source      RewardSource    // reward origin
}

func (m *MsgCreateVesting) Reset()         { *m = MsgCreateVesting{} }
func (m *MsgCreateVesting) String() string { return proto.CompactTextString(m) }
func (m *MsgCreateVesting) ProtoMessage()  {}

func (m *MsgCreateVesting) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	if m.ClaimId == "" {
		return fmt.Errorf("claim ID required")
	}
	if _, err := sdk.AccAddressFromBech32(m.Recipient); err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	if m.TotalAmount == "" || m.TotalAmount == "0" {
		return fmt.Errorf("total amount must be positive")
	}
	return nil
}

func (m *MsgCreateVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// MsgCreateVestingResponse is the response.
type MsgCreateVestingResponse struct {
	VestingId string
}

func (m *MsgCreateVestingResponse) Reset()         { *m = MsgCreateVestingResponse{} }
func (m *MsgCreateVestingResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgCreateVestingResponse) ProtoMessage()  {}

// MsgPauseVesting pauses a vesting schedule (due to active challenge).
// Only callable by authority (governance/dispute module).
type MsgPauseVesting struct {
	Authority string // module authority
	VestingId string // vesting schedule to pause
	Reason    string // reason for pause
}

func (m *MsgPauseVesting) Reset()         { *m = MsgPauseVesting{} }
func (m *MsgPauseVesting) String() string { return proto.CompactTextString(m) }
func (m *MsgPauseVesting) ProtoMessage()  {}

func (m *MsgPauseVesting) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	if m.VestingId == "" {
		return fmt.Errorf("vesting ID required")
	}
	return nil
}

func (m *MsgPauseVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// MsgPauseVestingResponse is the response.
type MsgPauseVestingResponse struct{}

func (m *MsgPauseVestingResponse) Reset()         { *m = MsgPauseVestingResponse{} }
func (m *MsgPauseVestingResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgPauseVestingResponse) ProtoMessage()  {}

// MsgResumeVesting resumes a paused vesting schedule.
// Only callable by authority (governance/dispute module).
type MsgResumeVesting struct {
	Authority string // module authority
	VestingId string // vesting schedule to resume
}

func (m *MsgResumeVesting) Reset()         { *m = MsgResumeVesting{} }
func (m *MsgResumeVesting) String() string { return proto.CompactTextString(m) }
func (m *MsgResumeVesting) ProtoMessage()  {}

func (m *MsgResumeVesting) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	if m.VestingId == "" {
		return fmt.Errorf("vesting ID required")
	}
	return nil
}

func (m *MsgResumeVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// MsgResumeVestingResponse is the response.
type MsgResumeVestingResponse struct{}

func (m *MsgResumeVestingResponse) Reset()         { *m = MsgResumeVestingResponse{} }
func (m *MsgResumeVestingResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgResumeVestingResponse) ProtoMessage()  {}

// MsgAccelerateVesting records evidence that accelerates vesting.
// AccelerationType: "defense" (successful challenge defense) or "replication" (independent replication).
type MsgAccelerateVesting struct {
	Authority          string // module authority
	VestingId          string // vesting schedule to accelerate
	AccelerationType   string // "defense" or "replication"
	EvidenceReference  string // reference to evidence (claim ID, etc.)
}

func (m *MsgAccelerateVesting) Reset()         { *m = MsgAccelerateVesting{} }
func (m *MsgAccelerateVesting) String() string { return proto.CompactTextString(m) }
func (m *MsgAccelerateVesting) ProtoMessage()  {}

func (m *MsgAccelerateVesting) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority required")
	}
	if m.VestingId == "" {
		return fmt.Errorf("vesting ID required")
	}
	if m.AccelerationType != "defense" && m.AccelerationType != "replication" {
		return fmt.Errorf("acceleration type must be 'defense' or 'replication'")
	}
	return nil
}

func (m *MsgAccelerateVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// MsgAccelerateVestingResponse is the response.
type MsgAccelerateVestingResponse struct {
	DefenseCount     uint32
	ReplicationCount uint32
}

func (m *MsgAccelerateVestingResponse) Reset()         { *m = MsgAccelerateVestingResponse{} }
func (m *MsgAccelerateVestingResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgAccelerateVestingResponse) ProtoMessage()  {}

// MsgCompleteVesting marks a vesting schedule as completed.
// Can be called by recipient (when max release reached) or authority (governance).
type MsgCompleteVesting struct {
	Sender    string // bech32 sender (recipient or authority)
	VestingId string // vesting schedule to complete
}

func (m *MsgCompleteVesting) Reset()         { *m = MsgCompleteVesting{} }
func (m *MsgCompleteVesting) String() string { return proto.CompactTextString(m) }
func (m *MsgCompleteVesting) ProtoMessage()  {}

func (m *MsgCompleteVesting) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.VestingId == "" {
		return fmt.Errorf("vesting ID required")
	}
	return nil
}

func (m *MsgCompleteVesting) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{addr}
}

// MsgCompleteVestingResponse is the response.
type MsgCompleteVestingResponse struct {
	FinalReleasedAmount string
}

func (m *MsgCompleteVestingResponse) Reset()         { *m = MsgCompleteVestingResponse{} }
func (m *MsgCompleteVestingResponse) String() string { return proto.CompactTextString(m) }
func (m *MsgCompleteVestingResponse) ProtoMessage()  {}

// MsgServer is the msg server interface
type MsgServer interface {
	ClaimVestedRewards(context.Context, *MsgClaimVestedRewards) (*MsgClaimVestedRewardsResponse, error)
	FalsifyClaim(context.Context, *MsgFalsifyClaim) (*MsgFalsifyClaimResponse, error)
	CreateVesting(context.Context, *MsgCreateVesting) (*MsgCreateVestingResponse, error)
	PauseVesting(context.Context, *MsgPauseVesting) (*MsgPauseVestingResponse, error)
	ResumeVesting(context.Context, *MsgResumeVesting) (*MsgResumeVestingResponse, error)
	AccelerateVesting(context.Context, *MsgAccelerateVesting) (*MsgAccelerateVestingResponse, error)
	CompleteVesting(context.Context, *MsgCompleteVesting) (*MsgCompleteVestingResponse, error)
}
