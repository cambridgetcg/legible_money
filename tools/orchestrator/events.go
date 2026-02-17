// Package orchestrator provides hierarchical multi-agent workflow orchestration
// for Legible Money's off-chain agent coordination. It composes agent-sdk and
// agent-swarm into structured "orchestra" workflows defined by declarative scores.
package orchestrator

import "time"

// EventType identifies the kind of orchestrator event.
type EventType string

const (
	// Conductor lifecycle events.
	EventTuneStarted      EventType = "tune_started"
	EventTuneCompleted    EventType = "tune_completed"
	EventPerformStarted   EventType = "perform_started"
	EventPerformCompleted EventType = "perform_completed"
	EventPaused           EventType = "paused"
	EventResumed          EventType = "resumed"
	EventAborted          EventType = "aborted"

	// Movement events.
	EventMovementStarted   EventType = "movement_started"
	EventMovementCompleted EventType = "movement_completed"
	EventMovementFailed    EventType = "movement_failed"
	EventGatePassed        EventType = "gate_passed"
	EventGateFailed        EventType = "gate_failed"

	// Section events.
	EventLeaderElected  EventType = "leader_elected"
	EventMemberJoined   EventType = "member_joined"
	EventMemberRemoved  EventType = "member_removed"
	EventSectionScaled  EventType = "section_scaled"

	// Part events.
	EventPartAssigned  EventType = "part_assigned"
	EventPartStarted   EventType = "part_started"
	EventPartCompleted EventType = "part_completed"
	EventPartFailed    EventType = "part_failed"
	EventPartRetried   EventType = "part_retried"

	// Economic events.
	EventBalanceLow     EventType = "balance_low"
	EventBalanceOK      EventType = "balance_ok"
	EventCheckpointSaved EventType = "checkpoint_saved"
)

// Event represents something that happened during orchestration.
type Event struct {
	Type      EventType         `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Movement  string            `json:"movement,omitempty"`  // Movement name if applicable
	Section   SectionName       `json:"section,omitempty"`   // Section if applicable
	Agent     string            `json:"agent,omitempty"`     // Agent name if applicable
	PartID    string            `json:"part_id,omitempty"`   // Part ID if applicable
	Message   string            `json:"message,omitempty"`   // Human-readable description
	Data      map[string]string `json:"data,omitempty"`      // Extra key-value data
}

// CommandType identifies an instruction sent to a section or agent.
type CommandType string

const (
	CmdExecutePart CommandType = "execute_part"
	CmdPause       CommandType = "pause"
	CmdResume      CommandType = "resume"
	CmdAbort       CommandType = "abort"
	CmdScale       CommandType = "scale"
)

// Command is an instruction dispatched by the conductor.
type Command struct {
	Type    CommandType       `json:"type"`
	Section SectionName       `json:"section,omitempty"`
	Agent   string            `json:"agent,omitempty"`
	PartID  string            `json:"part_id,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
}

// EventHandler receives events emitted during orchestration.
type EventHandler func(Event)
