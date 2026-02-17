package orchestrator

import "time"

// OrchestraHealth is the top-level health snapshot of the entire orchestration.
type OrchestraHealth struct {
	Score           string          `json:"score"`
	State           ConductorState  `json:"state"`
	CurrentMovement string          `json:"current_movement,omitempty"`
	MovementsTotal  int             `json:"movements_total"`
	MovementsDone   int             `json:"movements_done"`
	Sections        []SectionHealth `json:"sections"`
	Economic        EconomicHealth  `json:"economic"`
	Uptime          time.Duration   `json:"uptime"`
	LastCheckpoint  time.Time       `json:"last_checkpoint,omitempty"`
}

// SectionHealth reports the status of a single section.
type SectionHealth struct {
	Name          SectionName `json:"name"`
	LeaderAgent   string      `json:"leader_agent"`
	MemberCount   int         `json:"member_count"`
	ActiveParts   int         `json:"active_parts"`
	CompletedParts int        `json:"completed_parts"`
	FailedParts   int         `json:"failed_parts"`
}

// EconomicHealth tracks the financial position of the orchestra.
type EconomicHealth struct {
	TotalBalance     uint64 `json:"total_balance"`      // Sum of all agent balances (ulgm)
	TotalGasSpent    uint64 `json:"total_gas_spent"`     // Cumulative gas cost (ulgm)
	TotalRewards     uint64 `json:"total_rewards"`       // Cumulative rewards (ulgm)
	NetPosition      int64  `json:"net_position"`        // Rewards - gas spent
	LowBalanceAgents int    `json:"low_balance_agents"`  // Count below min operating balance
	Sustainable      bool   `json:"sustainable"`         // Earning more than spending
	BudgetRemaining  uint64 `json:"budget_remaining"`    // Max gas budget - spent
}

// ConductorState tracks the conductor lifecycle.
type ConductorState string

const (
	StateIdle       ConductorState = "idle"
	StateTuning     ConductorState = "tuning"
	StatePerforming ConductorState = "performing"
	StatePaused     ConductorState = "paused"
	StateComplete   ConductorState = "complete"
	StateAborted    ConductorState = "aborted"
	StateFailed     ConductorState = "failed"
)
