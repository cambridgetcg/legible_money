package orchestrator

import (
	"context"
	"fmt"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// GuardianAction describes a guardian operation.
type GuardianAction struct {
	Type   string            `json:"type"` // "create_home", "configure_guardian", "set_spending_limit"
	Params map[string]string `json:"params"`
}

// DefenseStrategy controls how the guardian protects Agent Homes.
type DefenseStrategy int

const (
	// DefensePassive monitors only — no automatic defense actions.
	DefensePassive DefenseStrategy = iota
	// DefenseActive automatically defends against detected threats.
	DefenseActive
	// DefenseAggressive proactively strengthens defenses.
	DefenseAggressive
)

// defenseStrategyName maps strategy to on-chain config string.
var defenseStrategyName = map[DefenseStrategy]string{
	DefensePassive:    "passive",
	DefenseActive:     "active",
	DefenseAggressive: "aggressive",
}

// WindInstrument implements guardian behavior (the "winds" section).
type WindInstrument struct {
	strategy         DefenseStrategy
	deadmanThreshold uint64            // blocks before deadman switch triggers
	spendingLimits   map[string]uint64 // keyType → maxAmount in ulgm
}

// NewWindInstrument creates a guardian instrument.
func NewWindInstrument(strategy DefenseStrategy, deadmanThreshold uint64) *WindInstrument {
	return &WindInstrument{
		strategy:         strategy,
		deadmanThreshold: deadmanThreshold,
		spendingLimits:   make(map[string]uint64),
	}
}

// SetSpendingLimit configures a spending limit for a key type.
func (w *WindInstrument) SetSpendingLimit(keyType string, maxAmount uint64) {
	w.spendingLimits[keyType] = maxAmount
}

// Name returns the instrument name.
func (w *WindInstrument) Name() string { return "wind" }

// Section returns the winds section.
func (w *WindInstrument) Section() SectionName { return SectionWinds }

// PrepareAction injects guardian-specific parameters before execution.
func (w *WindInstrument) PrepareAction(_ context.Context, _ *agentsdk.Agent, action ActionType, params map[string]string) (map[string]string, error) {
	switch action {
	case ActionCreateHome:
		result := copyParams(params)
		result["defense_strategy"] = defenseStrategyName[w.strategy]
		if w.deadmanThreshold > 0 {
			result["deadman_threshold"] = fmt.Sprintf("%d", w.deadmanThreshold)
		}
		// Set auto_defend based on strategy
		if w.strategy >= DefenseActive {
			result["auto_defend"] = "true"
		}
		return result, nil

	case ActionSyncHome:
		result := copyParams(params)
		// Inject defense strategy for sync operations
		result["defense_strategy"] = defenseStrategyName[w.strategy]
		return result, nil

	default:
		return params, nil
	}
}

// GenerateGuardianActions returns a list of guardian actions based on the strategy.
func (w *WindInstrument) GenerateGuardianActions(homeID string) []GuardianAction {
	var actions []GuardianAction

	// Configure guardian
	actions = append(actions, GuardianAction{
		Type: "configure_guardian",
		Params: map[string]string{
			"home_id":           homeID,
			"strategy":          defenseStrategyName[w.strategy],
			"deadman_threshold": fmt.Sprintf("%d", w.deadmanThreshold),
		},
	})

	// Apply spending limits
	for keyType, maxAmount := range w.spendingLimits {
		actions = append(actions, GuardianAction{
			Type: "set_spending_limit",
			Params: map[string]string{
				"home_id":    homeID,
				"key_type":   keyType,
				"max_amount": fmt.Sprintf("%d", maxAmount),
			},
		})
	}

	return actions
}

// Compile-time check
var _ Instrument = (*WindInstrument)(nil)
