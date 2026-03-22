package types

import "fmt"

// BlocksPerDay is ~86400s / 2.521s block time.
const BlocksPerDay uint64 = 34272

// Plan names.
const (
	PlanFree  = "free"
	PlanSeed  = "seed"
	PlanGrow  = "grow"
	PlanScale = "scale"
)

// Subscription statuses.
const (
	StatusActive    = "active"
	StatusCancelled = "cancelled"
	StatusPastDue   = "past_due"
)

// SubscriptionRecord tracks a project's billing plan on-chain.
// Hand-written type — serialized with json.Marshal (no proto registration).
type SubscriptionRecord struct {
	ProjectID string `json:"project_id"`
	Plan      string `json:"plan"`       // free, seed, grow, scale
	Status    string `json:"status"`     // active, cancelled, past_due
	PeriodEnd uint64 `json:"period_end"` // block height when current period ends
	CreatedAt uint64 `json:"created_at"` // block height
}

// Validate checks the subscription record fields.
func (s *SubscriptionRecord) Validate() error {
	if s.ProjectID == "" {
		return fmt.Errorf("project_id cannot be empty")
	}
	switch s.Plan {
	case PlanFree, PlanSeed, PlanGrow, PlanScale:
	default:
		return fmt.Errorf("invalid plan: %s", s.Plan)
	}
	switch s.Status {
	case StatusActive, StatusCancelled, StatusPastDue:
	default:
		return fmt.Errorf("invalid status: %s", s.Status)
	}
	return nil
}

// UsageCounter tracks per-project resource usage within a billing period.
// Hand-written type — serialized with json.Marshal.
type UsageCounter struct {
	ProjectID   string            `json:"project_id"`
	Counters    map[string]uint64 `json:"counters"`     // resource -> count
	PeriodStart uint64            `json:"period_start"` // block when counters were last reset
}

// ResourceLimit defines a per-resource daily limit. -1 means unlimited.
type ResourceLimit struct {
	MemoryOps     int64 `json:"memory_ops"`
	ToolCalls     int64 `json:"tool_calls"`
	Verifications int64 `json:"verifications"`
}

// Resource keys.
const (
	ResourceMemoryOps     = "memory_ops"
	ResourceToolCalls     = "tool_calls"
	ResourceVerifications = "verifications"
)

// DefaultPlanLimits returns limits for each plan.
// -1 = unlimited.
func DefaultPlanLimits() map[string]ResourceLimit {
	return map[string]ResourceLimit{
		PlanFree:  {MemoryOps: 100, ToolCalls: 10, Verifications: 5},
		PlanSeed:  {MemoryOps: 10_000, ToolCalls: 500, Verifications: 100},
		PlanGrow:  {MemoryOps: 100_000, ToolCalls: 5_000, Verifications: 1_000},
		PlanScale: {MemoryOps: -1, ToolCalls: -1, Verifications: -1},
	}
}

// GetLimit returns the limit for a given resource within a plan's limits.
// Returns -1 (unlimited) if the resource is unknown.
func (r ResourceLimit) GetLimit(resource string) int64 {
	switch resource {
	case ResourceMemoryOps:
		return r.MemoryOps
	case ResourceToolCalls:
		return r.ToolCalls
	case ResourceVerifications:
		return r.Verifications
	default:
		return -1
	}
}
