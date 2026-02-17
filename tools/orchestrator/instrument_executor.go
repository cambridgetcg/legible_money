package orchestrator

import (
	"context"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// InstrumentExecutor wraps an ActionExecutor and calls the appropriate
// instrument's PrepareAction before delegating to the inner executor.
// This is the decorator pattern — composable and testable.
type InstrumentExecutor struct {
	inner       ActionExecutor
	instruments map[SectionName]Instrument
}

// NewInstrumentExecutor creates an executor that applies instruments before dispatching.
func NewInstrumentExecutor(inner ActionExecutor, instruments map[SectionName]Instrument) *InstrumentExecutor {
	if instruments == nil {
		instruments = make(map[SectionName]Instrument)
	}
	return &InstrumentExecutor{
		inner:       inner,
		instruments: instruments,
	}
}

// SetInstrument assigns an instrument to a section.
func (ie *InstrumentExecutor) SetInstrument(section SectionName, inst Instrument) {
	ie.instruments[section] = inst
}

// Execute applies the instrument's PrepareAction, then delegates to the inner executor.
// If no instrument is registered for the action's section, it falls through directly.
func (ie *InstrumentExecutor) Execute(ctx context.Context, agent *agentsdk.Agent, action ActionType, params map[string]string) (*PartResult, error) {
	// Determine which section this action belongs to
	section := actionToSection(action)

	// Look up instrument
	inst, hasInstrument := ie.instruments[section]
	if hasInstrument {
		prepared, err := inst.PrepareAction(ctx, agent, action, params)
		if err != nil {
			return &PartResult{
				Success: false,
				Error:   err.Error(),
			}, err
		}
		params = prepared
	}

	return ie.inner.Execute(ctx, agent, action, params)
}

// actionToSection maps an action type to its canonical section.
func actionToSection(action ActionType) SectionName {
	switch action {
	case ActionVerify, ActionAutoVerify:
		return SectionStrings
	case ActionSubmitClaim:
		return SectionBrass
	case ActionChallenge:
		return SectionPercussion
	case ActionCreateHome, ActionSyncHome:
		return SectionWinds
	default:
		// Generic actions (register, stake, heartbeat, etc.) don't have a section
		return ""
	}
}

// Compile-time check
var _ ActionExecutor = (*InstrumentExecutor)(nil)
