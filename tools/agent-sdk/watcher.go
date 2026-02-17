// watcher.go provides event subscription via polling for AI agents.
//
// Since CometBFT WebSocket subscriptions are complex and unreliable for
// long-lived agents, EventWatcher uses a simple polling approach:
// periodically query active rounds and emit callbacks when state changes.
//
// Example:
//
//	watcher := agentsdk.NewEventWatcher(queryClient, agentsdk.WatcherConfig{
//	    PollInterval: agentsdk.BlockTime,
//	    Filters:      []string{"commit", "reveal"},
//	})
//	watcher.OnRoundPhaseChange(func(round RoundInfo, oldPhase, newPhase string) {
//	    log.Printf("Round %s: %s → %s", round.ID, oldPhase, newPhase)
//	})
//	watcher.OnNewRound(func(round RoundInfo) {
//	    log.Printf("New round: %s for claim %s", round.ID, round.ClaimID)
//	})
//	go watcher.Run(ctx)
package agentsdk

import (
	"context"
	"sync"
	"time"
)

// WatcherConfig configures the event watcher.
type WatcherConfig struct {
	// PollInterval is how often to check for new events. Default: BlockTime.
	PollInterval time.Duration

	// Filters restricts which round phases to watch. Empty = all phases.
	Filters []string

	// Domains restricts which domains to watch via fact queries. Empty = all.
	Domains []string
}

// RoundPhaseHandler is called when a round transitions between phases.
type RoundPhaseHandler func(round RoundInfo, oldPhase, newPhase string)

// NewRoundHandler is called when a previously unseen round appears.
type NewRoundHandler func(round RoundInfo)

// RoundCompleteHandler is called when a round disappears from active rounds.
type RoundCompleteHandler func(roundID string)

// EventWatcher polls for on-chain events and invokes registered callbacks.
type EventWatcher struct {
	qc  QueryClient
	cfg WatcherConfig

	// Callbacks
	onPhaseChange   []RoundPhaseHandler
	onNewRound      []NewRoundHandler
	onRoundComplete []RoundCompleteHandler
	mu              sync.Mutex

	// State tracking
	knownRounds map[string]string // roundID → last known phase
}

// NewEventWatcher creates a new event watcher.
func NewEventWatcher(qc QueryClient, cfg WatcherConfig) *EventWatcher {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = BlockTime
	}
	return &EventWatcher{
		qc:          qc,
		cfg:         cfg,
		knownRounds: make(map[string]string),
	}
}

// OnRoundPhaseChange registers a callback for phase transitions.
func (w *EventWatcher) OnRoundPhaseChange(handler RoundPhaseHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onPhaseChange = append(w.onPhaseChange, handler)
}

// OnNewRound registers a callback for newly discovered rounds.
func (w *EventWatcher) OnNewRound(handler NewRoundHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onNewRound = append(w.onNewRound, handler)
}

// OnRoundComplete registers a callback when a round disappears from active rounds.
func (w *EventWatcher) OnRoundComplete(handler RoundCompleteHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onRoundComplete = append(w.onRoundComplete, handler)
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (w *EventWatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *EventWatcher) poll(ctx context.Context) {
	rounds, err := w.qc.ActiveRounds(ctx)
	if err != nil {
		return
	}

	w.mu.Lock()
	handlers := struct {
		phaseChange   []RoundPhaseHandler
		newRound      []NewRoundHandler
		roundComplete []RoundCompleteHandler
	}{
		phaseChange:   append([]RoundPhaseHandler{}, w.onPhaseChange...),
		newRound:      append([]NewRoundHandler{}, w.onNewRound...),
		roundComplete: append([]RoundCompleteHandler{}, w.onRoundComplete...),
	}
	w.mu.Unlock()

	// Track which rounds are still active.
	activeIDs := make(map[string]bool)

	for _, round := range rounds {
		// Apply phase filter.
		if len(w.cfg.Filters) > 0 && !contains(w.cfg.Filters, round.Phase) {
			continue
		}

		activeIDs[round.ID] = true

		oldPhase, known := w.knownRounds[round.ID]
		if !known {
			// New round.
			w.knownRounds[round.ID] = round.Phase
			for _, h := range handlers.newRound {
				h(round)
			}
		} else if oldPhase != round.Phase {
			// Phase changed.
			w.knownRounds[round.ID] = round.Phase
			for _, h := range handlers.phaseChange {
				h(round, oldPhase, round.Phase)
			}
		}
	}

	// Detect completed rounds (were known, no longer active).
	for id := range w.knownRounds {
		if !activeIDs[id] {
			delete(w.knownRounds, id)
			for _, h := range handlers.roundComplete {
				h(id)
			}
		}
	}
}

// KnownRoundCount returns how many rounds are currently being tracked.
func (w *EventWatcher) KnownRoundCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.knownRounds)
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
