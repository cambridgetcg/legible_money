package types

import "fmt"

// ============ Auto-Governance Proposal Types (Loop 4 — Structural Evolution) ============
//
// When sustained hormonal signals indicate structural issues that multiplier
// adjustments alone cannot fix, the autopoiesis module drafts governance
// proposals (LIPs) for human/agent voting. Auto-proposals are suggestions,
// not submissions — they are emitted as events for off-chain agents to
// pick up and submit via MsgSubmitProposal with deposit.

// ProposalTriggerType identifies the kind of structural issue detected.
type ProposalTriggerType string

const (
	TriggerBoundHit        ProposalTriggerType = "bound_hit"         // multiplier stuck at bound
	TriggerDomainStress    ProposalTriggerType = "domain_stress"     // DVI negative for 10+ epochs
	TriggerSustainedStress ProposalTriggerType = "sustained_stress"  // SSI > moderate for 10+ epochs
	TriggerOpenClawCritical ProposalTriggerType = "openclaw_critical" // 3+ verified protocol findings
)

// Trigger thresholds (in epochs).
const (
	BoundHitThresholdEpochs        = 20 // multiplier at bound for 20+ epochs
	DomainStressThresholdEpochs    = 10 // DVI negative for 10+ epochs
	SustainedStressThresholdEpochs = 10 // SSI > moderate for 10+ epochs
	OpenClawCriticalThreshold      = 3  // 3+ verified "protocol" facts flagging same issue
)

// ProposalTrigger defines when an auto-proposal should be generated.
type ProposalTrigger struct {
	Type              ProposalTriggerType `json:"type"`
	Path              string              `json:"path,omitempty"`    // multiplier path (for bound_hit)
	Domain            string              `json:"domain,omitempty"`  // (for domain_stress)
	ConsecutiveEpochs uint64              `json:"consecutive_epochs"`
	Threshold         uint64              `json:"threshold"` // trigger threshold
	CurrentValue      uint64              `json:"current_value,omitempty"`
}

// AutoProposal is a governance proposal draft created by the autopoiesis module.
type AutoProposal struct {
	Trigger     ProposalTrigger `json:"trigger"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	ParamPath   string          `json:"param_path,omitempty"`  // param to change
	SuggestedValue uint64       `json:"suggested_value,omitempty"` // suggested new value (BPS)
	Epoch       uint64          `json:"epoch"`
}

// AutoProposalState tracks rate limiting for auto-proposals.
type AutoProposalState struct {
	LastProposalEpoch uint64 `json:"last_proposal_epoch"`
	ProposalsInWindow uint64 `json:"proposals_in_window"`
}

// ProposalTriggerState tracks sustained condition detection for a specific trigger.
type ProposalTriggerState struct {
	Type              ProposalTriggerType `json:"type"`
	Path              string              `json:"path,omitempty"`
	Domain            string              `json:"domain,omitempty"`
	ConsecutiveEpochs uint64              `json:"consecutive_epochs"`
	LastCheckedEpoch  uint64              `json:"last_checked_epoch"`
	// AcknowledgedAtBlock tracks the block height at which a count-based trigger
	// (e.g. openclaw_critical) last fired. Facts verified after this height are
	// counted toward the next trigger. Zero means never fired.
	AcknowledgedAtBlock uint64 `json:"acknowledged_at_block,omitempty"`
}

// TriggerStateKey returns a unique key suffix for a ProposalTriggerState.
func TriggerStateKey(triggerType ProposalTriggerType, path, domain string) string {
	return fmt.Sprintf("%s:%s:%s", triggerType, path, domain)
}
