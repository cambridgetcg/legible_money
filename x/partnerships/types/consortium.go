package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Consortium represents a multi-party partnership (2-11 members).
type Consortium struct {
	Id                string              `json:"id"`
	Name              string              `json:"name"`
	Status            string              `json:"status"` // forming, active, cooling, dissolved
	Members           []*ConsortiumMember `json:"members"`
	CommonPotBalance  string              `json:"common_pot_balance"`
	TotalEarned       string              `json:"total_earned"`
	CommonPotShareBps uint64              `json:"common_pot_share_bps"` // default 220000 (22%)
	LockTier          uint32              `json:"lock_tier"`
	LockExpiresAt     uint64              `json:"lock_expires_at"`
	FormedAtBlock     uint64              `json:"formed_at_block"`
	RequiredApprovals uint32              `json:"required_approvals"` // for pot spending (default: all)
	CooperationScore  uint64              `json:"cooperation_score"`
	ExitState         *ConsortiumExitState `json:"exit_state,omitempty"`
}

// ConsortiumMember represents one member in a consortium.
type ConsortiumMember struct {
	Address     string `json:"address"`
	AccountType string `json:"account_type"` // "human" or "agent"
	ShareBps    uint64 `json:"share_bps"`    // member's share of non-pot rewards (must sum to 1M)
	JoinedAt    uint64 `json:"joined_at"`    // block height
	Role        string `json:"role"`         // "founder", "member", "observer"
	Accepted    bool   `json:"accepted"`     // formation acceptance flag
}

// ConsortiumExitState tracks dissolution state for a consortium.
type ConsortiumExitState struct {
	InitiatedBy string `json:"initiated_by"`
	InitiatedAt uint64 `json:"initiated_at"`
	CooldownEnd uint64 `json:"cooldown_end"`
}

// Consortium constants.
const (
	ConsortiumMinMembers      = 2
	ConsortiumMaxMembers      = 11
	ConsortiumFormationWindow = 154 // 2x partnership formation (77*2)
)

// ValidateConsortium validates a consortium's fields.
func ValidateConsortium(c *Consortium) error {
	if c.Id == "" {
		return fmt.Errorf("consortium id cannot be empty")
	}
	if c.Name == "" {
		return fmt.Errorf("consortium name cannot be empty")
	}
	if c.Status != StatusForming && c.Status != StatusActive &&
		c.Status != StatusCooling && c.Status != StatusDissolved {
		return fmt.Errorf("invalid consortium status: %s", c.Status)
	}
	if len(c.Members) < ConsortiumMinMembers {
		return ErrConsortiumTooSmall
	}
	if len(c.Members) > ConsortiumMaxMembers {
		return ErrConsortiumFull
	}

	// Validate all members and check share sum
	var totalShareBps uint64
	addresses := make(map[string]bool)
	for _, m := range c.Members {
		if err := ValidateConsortiumMember(m); err != nil {
			return err
		}
		if addresses[m.Address] {
			return fmt.Errorf("duplicate member address: %s", m.Address)
		}
		addresses[m.Address] = true
		totalShareBps += m.ShareBps
	}
	if totalShareBps != 1000000 {
		return fmt.Errorf("%w: got %d", ErrShareSumInvalid, totalShareBps)
	}

	return nil
}

// ValidateConsortiumMember validates a single member.
func ValidateConsortiumMember(m *ConsortiumMember) error {
	if m.Address == "" {
		return fmt.Errorf("member address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Address); err != nil {
		return fmt.Errorf("invalid member address %s: %w", m.Address, err)
	}
	if m.AccountType != "human" && m.AccountType != "agent" {
		return fmt.Errorf("invalid account_type: %s (must be human or agent)", m.AccountType)
	}
	if m.Role != "founder" && m.Role != "member" && m.Role != "observer" {
		return fmt.Errorf("invalid role: %s (must be founder, member, or observer)", m.Role)
	}
	return nil
}
