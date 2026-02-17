package claude

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrRateLimited is returned when the per-minute request limit is exceeded.
	ErrRateLimited = errors.New("claude: rate limit exceeded")
	// ErrBudgetExceeded is returned when the daily token budget is exhausted.
	ErrBudgetExceeded = errors.New("claude: daily token budget exceeded")
)

// UsageStats holds current usage counters.
type UsageStats struct {
	InputTokensUsed  int64     `json:"input_tokens_used"`
	OutputTokensUsed int64     `json:"output_tokens_used"`
	RequestCount     int64     `json:"request_count"`
	WindowStart      time.Time `json:"window_start"`
}

// SubscriptionManager tracks API usage against Claude Max subscription limits.
// It enforces per-minute rate limits and daily token budgets.
type SubscriptionManager struct {
	mu sync.Mutex

	// Accumulated usage in the current window.
	inputTokensUsed  int64
	outputTokensUsed int64
	requestCount     int64
	windowStart      time.Time

	// Per-minute sliding window for rate limiting.
	minuteRequests []time.Time

	// Configurable limits.
	MaxRequestsPerMinute int
	MaxTokensPerDay      int64

	// Clock function for testability.
	now func() time.Time
}

// NewSubscriptionManager creates a manager with the given limits.
func NewSubscriptionManager(maxReqPerMin int, maxTokensPerDay int64) *SubscriptionManager {
	return &SubscriptionManager{
		MaxRequestsPerMinute: maxReqPerMin,
		MaxTokensPerDay:      maxTokensPerDay,
		windowStart:          time.Now(),
		now:                  time.Now,
	}
}

// CanRequest checks whether a request with the estimated token count is allowed.
// Returns nil if OK, ErrRateLimited or ErrBudgetExceeded otherwise.
func (sm *SubscriptionManager) CanRequest(estimatedTokens int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := sm.now()

	// Reset daily window if >24h since window start.
	if now.Sub(sm.windowStart) > 24*time.Hour {
		sm.inputTokensUsed = 0
		sm.outputTokensUsed = 0
		sm.requestCount = 0
		sm.windowStart = now
		sm.minuteRequests = nil
	}

	// Prune per-minute sliding window.
	cutoff := now.Add(-1 * time.Minute)
	pruned := sm.minuteRequests[:0]
	for _, t := range sm.minuteRequests {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	sm.minuteRequests = pruned

	// Check per-minute rate.
	if sm.MaxRequestsPerMinute > 0 && len(sm.minuteRequests) >= sm.MaxRequestsPerMinute {
		return ErrRateLimited
	}

	// Check daily budget.
	totalUsed := sm.inputTokensUsed + sm.outputTokensUsed
	if sm.MaxTokensPerDay > 0 && totalUsed+int64(estimatedTokens) > sm.MaxTokensPerDay {
		return ErrBudgetExceeded
	}

	return nil
}

// RecordUsage records tokens consumed by a completed request.
func (sm *SubscriptionManager) RecordUsage(inputTokens, outputTokens int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := sm.now()
	sm.inputTokensUsed += int64(inputTokens)
	sm.outputTokensUsed += int64(outputTokens)
	sm.requestCount++
	sm.minuteRequests = append(sm.minuteRequests, now)
}

// Usage returns a snapshot of current usage stats.
func (sm *SubscriptionManager) Usage() UsageStats {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return UsageStats{
		InputTokensUsed:  sm.inputTokensUsed,
		OutputTokensUsed: sm.outputTokensUsed,
		RequestCount:     sm.requestCount,
		WindowStart:      sm.windowStart,
	}
}

// ResetWindow resets all usage counters. Useful for testing or manual reset.
func (sm *SubscriptionManager) ResetWindow() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.inputTokensUsed = 0
	sm.outputTokensUsed = 0
	sm.requestCount = 0
	sm.minuteRequests = nil
	sm.windowStart = sm.now()
}
