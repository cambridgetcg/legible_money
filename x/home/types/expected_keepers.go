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
