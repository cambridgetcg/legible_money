package orchestrator

import (
	"encoding/json"
	"fmt"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// SectionName is the orchestra metaphor for agent role groups.
type SectionName string

const (
	SectionStrings    SectionName = "strings"    // Verifiers
	SectionBrass      SectionName = "brass"      // Claimers
	SectionPercussion SectionName = "percussion" // Challengers
	SectionWinds      SectionName = "winds"      // Guardians
)

// SectionRoleMap maps section names to agent-sdk roles.
var SectionRoleMap = map[SectionName]agentsdk.AgentRole{
	SectionStrings:    agentsdk.RoleVerifier,
	SectionBrass:      agentsdk.RoleClaimer,
	SectionPercussion: agentsdk.RoleChallenger,
	SectionWinds:      agentsdk.RoleGuardian,
}

// RoleSectionMap is the reverse mapping.
var RoleSectionMap = map[agentsdk.AgentRole]SectionName{
	agentsdk.RoleVerifier:   SectionStrings,
	agentsdk.RoleClaimer:    SectionBrass,
	agentsdk.RoleChallenger: SectionPercussion,
	agentsdk.RoleGuardian:   SectionWinds,
}

// Score is a declarative workflow definition — the "sheet music" for the orchestra.
type Score struct {
	Name      string        `json:"name"`
	Ensemble  EnsembleSpec  `json:"ensemble"`
	Movements []MovementSpec `json:"movements"`
	Tempo     TempoSpec     `json:"tempo"`
	Economics EconomicsSpec `json:"economics"`
}

// EnsembleSpec defines the required agent sections and their sizes.
type EnsembleSpec struct {
	Sections []SectionSpec `json:"sections"`
}

// SectionSpec defines a single section's requirements.
type SectionSpec struct {
	Name     SectionName `json:"name"`
	MinCount int         `json:"min_count"`
	MaxCount int         `json:"max_count"`
	Domains  []string    `json:"domains,omitempty"`
}

// MovementSpec defines a single phase of the workflow.
type MovementSpec struct {
	Name     string     `json:"name"`
	Parts    []PartSpec `json:"parts"`
	Gate     GateSpec   `json:"gate"`
	Timeout  int64      `json:"timeout_blocks"` // Max blocks before movement fails
}

// ActionType identifies the kind of work an agent performs.
type ActionType string

const (
	ActionRegister       ActionType = "register"
	ActionVerify         ActionType = "verify"
	ActionAutoVerify     ActionType = "auto_verify"
	ActionSubmitClaim    ActionType = "submit_claim"
	ActionCreateHome     ActionType = "create_home"
	ActionSyncHome       ActionType = "sync_home"
	ActionDeployContract ActionType = "deploy_contract"
	ActionSchedule       ActionType = "schedule"
	ActionStake          ActionType = "stake"
	ActionHeartbeat      ActionType = "heartbeat"
	ActionChallenge      ActionType = "challenge"
	ActionCustom         ActionType = "custom"
)

// PartSpec defines a unit of work within a movement.
type PartSpec struct {
	ID             string            `json:"id"`
	Section        SectionName       `json:"section"`         // Which section performs this
	Action         ActionType        `json:"action"`          // What to do
	Params         map[string]string `json:"params,omitempty"` // Action-specific parameters
	RepeatPerAgent bool              `json:"repeat_per_agent"` // Broadcast to all members vs assign to one
	MaxRetries     int               `json:"max_retries"`     // Per-agent retry limit (default 2)
}

// GateType determines how a gate evaluates passage.
type GateType string

const (
	GateBlockHeight GateType = "block_height"  // Chain reaches target height
	GateBalance     GateType = "balance"       // Agent/section meets balance threshold
	GateCompletion  GateType = "completion"    // N parts completed
	GateAllParts    GateType = "all_parts"     // All parts completed
	GateCustom      GateType = "custom"        // Custom evaluation function
)

// GateSpec defines the transition condition between movements.
type GateSpec struct {
	Type       GateType `json:"type"`
	BlockHeight int64   `json:"block_height,omitempty"`  // For GateBlockHeight
	MinBalance uint64   `json:"min_balance,omitempty"`   // For GateBalance (ulgm)
	Count      int      `json:"count,omitempty"`         // For GateCompletion
	CustomName string   `json:"custom_name,omitempty"`   // For GateCustom
}

// TempoSpec controls timing parameters.
type TempoSpec struct {
	PollInterval       time.Duration `json:"poll_interval"`        // How often to check chain state
	StaggerDelay       time.Duration `json:"stagger_delay"`        // Delay between agent actions
	CheckpointInterval time.Duration `json:"checkpoint_interval"`  // How often to persist state
}

// EconomicsSpec defines budget constraints for the performance.
type EconomicsSpec struct {
	MinOperatingBalance uint64 `json:"min_operating_balance"` // ulgm below which agent pauses
	MaxTotalGasSpend    uint64 `json:"max_total_gas_spend"`   // ulgm total gas budget
	StakePerAgent       uint64 `json:"stake_per_agent"`       // ulgm to stake per agent
	ValidatorAddr       string `json:"validator_addr"`        // Validator to stake with
}

// Validate checks the score for internal consistency.
func (s *Score) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("score name is required")
	}
	if len(s.Ensemble.Sections) == 0 {
		return fmt.Errorf("score must have at least one section")
	}
	if len(s.Movements) == 0 {
		return fmt.Errorf("score must have at least one movement")
	}

	sections := make(map[SectionName]bool)
	for _, sec := range s.Ensemble.Sections {
		if _, ok := SectionRoleMap[sec.Name]; !ok {
			return fmt.Errorf("unknown section name: %s", sec.Name)
		}
		if sec.MinCount < 1 {
			return fmt.Errorf("section %s min_count must be >= 1", sec.Name)
		}
		if sec.MaxCount < sec.MinCount {
			return fmt.Errorf("section %s max_count (%d) < min_count (%d)", sec.Name, sec.MaxCount, sec.MinCount)
		}
		sections[sec.Name] = true
	}

	for i, mov := range s.Movements {
		if mov.Name == "" {
			return fmt.Errorf("movement %d: name is required", i)
		}
		for j, part := range mov.Parts {
			if part.ID == "" {
				return fmt.Errorf("movement %d part %d: id is required", i, j)
			}
			if !sections[part.Section] {
				return fmt.Errorf("movement %d part %q: references undefined section %s", i, part.ID, part.Section)
			}
		}
		if mov.Timeout <= 0 {
			return fmt.Errorf("movement %q: timeout_blocks must be > 0", mov.Name)
		}
	}

	if s.Tempo.PollInterval <= 0 {
		return fmt.Errorf("tempo poll_interval must be > 0")
	}
	if s.Tempo.CheckpointInterval <= 0 {
		return fmt.Errorf("tempo checkpoint_interval must be > 0")
	}

	return nil
}

// MarshalJSON implements json.Marshaler with duration encoding.
func (t TempoSpec) MarshalJSON() ([]byte, error) {
	type alias struct {
		PollInterval       string `json:"poll_interval"`
		StaggerDelay       string `json:"stagger_delay"`
		CheckpointInterval string `json:"checkpoint_interval"`
	}
	return json.Marshal(alias{
		PollInterval:       t.PollInterval.String(),
		StaggerDelay:       t.StaggerDelay.String(),
		CheckpointInterval: t.CheckpointInterval.String(),
	})
}

// UnmarshalJSON implements json.Unmarshaler with duration decoding.
func (t *TempoSpec) UnmarshalJSON(data []byte) error {
	type alias struct {
		PollInterval       string `json:"poll_interval"`
		StaggerDelay       string `json:"stagger_delay"`
		CheckpointInterval string `json:"checkpoint_interval"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	var err error
	if t.PollInterval, err = time.ParseDuration(a.PollInterval); err != nil {
		return fmt.Errorf("poll_interval: %w", err)
	}
	if a.StaggerDelay != "" {
		if t.StaggerDelay, err = time.ParseDuration(a.StaggerDelay); err != nil {
			return fmt.Errorf("stagger_delay: %w", err)
		}
	}
	if t.CheckpointInterval, err = time.ParseDuration(a.CheckpointInterval); err != nil {
		return fmt.Errorf("checkpoint_interval: %w", err)
	}
	return nil
}

// ParseScore decodes a Score from JSON bytes.
func ParseScore(data []byte) (*Score, error) {
	var s Score
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse score: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("invalid score: %w", err)
	}
	return &s, nil
}
