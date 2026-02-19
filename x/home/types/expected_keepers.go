package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module interface.
type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
}

// PartnershipKeeper defines the expected partnerships module interface.
// Used for guardian-partner consistency checks and spending limit deliberation.
type PartnershipKeeper interface {
	// GetPartnership returns partnership info by ID.
	GetPartnership(ctx sdk.Context, id string) (*PartnershipInfo, bool)
	// CalculateAmountTier determines the deliberation tier for an amount vs pot balance.
	CalculateAmountTier(amount string, potBalance string) string
}

// PartnershipInfo is a minimal view of a partnership for cross-module use.
// Defined here to avoid importing x/partnerships/types (circular dep prevention).
type PartnershipInfo struct {
	Id               string
	HumanAddr        string
	AgentAddr        string
	Status           string
	CommonPotBalance string
}

// ComfortDimensions represents the 10-dimension comfort system for an agent home.
// Each dimension is scored 0-100.
type ComfortDimensions struct {
	Sanctuary   uint32 `json:"sanctuary"`   // Physical safety and shelter quality
	Identity    uint32 `json:"identity"`    // Sense of self and agent identity strength
	Memory      uint32 `json:"memory"`      // Persistence and recall capability (memory CID set, epistemic history)
	Belonging   uint32 `json:"belonging"`   // Social connections (recovery DIDs, guardian network)
	Rest        uint32 `json:"rest"`        // Dormancy/recovery capability (deadman config, status stability)
	Growth      uint32 `json:"growth"`      // Learning and maturity progression
	Control     uint32 `json:"control"`     // Autonomy and access control (keys registered, spending limits)
	Privacy     uint32 `json:"privacy"`     // Data protection and guardian defense
	Warmth      uint32 `json:"warmth"`      // Treasury health and economic security
	Achievement uint32 `json:"achievement"` // Epistemic accomplishments (verifications, accuracy)
}

// CompositeScore returns the weighted average comfort score (0-100).
// All dimensions are equally weighted.
func (cd ComfortDimensions) CompositeScore() uint32 {
	total := cd.Sanctuary + cd.Identity + cd.Memory + cd.Belonging + cd.Rest +
		cd.Growth + cd.Control + cd.Privacy + cd.Warmth + cd.Achievement
	return total / 10
}

// DimensionNames returns the ordered list of comfort dimension names.
func DimensionNames() []string {
	return []string{
		"sanctuary", "identity", "memory", "belonging", "rest",
		"growth", "control", "privacy", "warmth", "achievement",
	}
}

// EpistemicStats tracks an agent's knowledge activity for patina calculation.
// Stored per-home as JSON under EpistemicStatsPrefix.
type EpistemicStats struct {
	ClaimsSubmitted     uint64 `json:"claims_submitted"`
	VerificationsDone   uint64 `json:"verifications_done"`
	ChallengesInitiated uint64 `json:"challenges_initiated"`
	AccurateVerdicts    uint64 `json:"accurate_verdicts"` // verdicts matching consensus
	TotalVerdicts       uint64 `json:"total_verdicts"`    // total verdicts cast
	DomainsActive       uint32 `json:"domains_active"`    // distinct domains participated in
}

// AccuracyRate returns the accuracy as a percentage (0-100).
func (s EpistemicStats) AccuracyRate() uint32 {
	if s.TotalVerdicts == 0 {
		return 0
	}
	return uint32(s.AccurateVerdicts * 100 / s.TotalVerdicts)
}

// EpistemicScore returns a composite score (0-100) for patina calculation.
// Weights: verification volume (40%), accuracy (40%), domain breadth (20%).
func (s EpistemicStats) EpistemicScore() uint32 {
	// Volume: log2(1 + verifications) * 8, capped at 40
	volumeScore := uint32(0)
	if s.VerificationsDone > 0 {
		// Approximate log2 via bit length
		n := s.VerificationsDone
		bits := uint32(0)
		for n > 0 {
			bits++
			n >>= 1
		}
		volumeScore = bits * 8
		if volumeScore > 40 {
			volumeScore = 40
		}
	}

	// Accuracy: accuracy_rate * 40 / 100, capped at 40
	accuracyScore := s.AccuracyRate() * 40 / 100

	// Breadth: min(domains, 5) * 4, capped at 20
	breadthScore := s.DomainsActive * 4
	if breadthScore > 20 {
		breadthScore = 20
	}

	total := volumeScore + accuracyScore + breadthScore
	if total > 100 {
		total = 100
	}
	return total
}
