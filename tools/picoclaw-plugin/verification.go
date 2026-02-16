// verification.go provides the automated verification workflow for PicoClaw agents.
//
// The VerificationWorkflow manages the full lifecycle:
//   1. Poll for active verification rounds
//   2. Evaluate claims using the agent's ClaimEvaluator
//   3. Submit commit during commit phase
//   4. Submit reveal during reveal phase
//   5. Track rewards and accuracy metrics

package picoclaw

import (
	"context"
	"fmt"
	"sync"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// VerificationWorkflow manages automated claim verification for a PicoClaw agent.
type VerificationWorkflow struct {
	agent     *agentsdk.Agent
	evaluator agentsdk.ClaimEvaluator

	// Track rounds we've already processed to avoid double-commits
	processedRounds map[string]bool
	mu              sync.Mutex

	// Metrics
	metrics VerificationMetrics
}

// VerificationMetrics tracks the agent's verification performance.
type VerificationMetrics struct {
	mu               sync.Mutex
	TotalEvaluated   int    `json:"total_evaluated"`
	TotalCommitted   int    `json:"total_committed"`
	TotalRevealed    int    `json:"total_revealed"`
	TotalErrors      int    `json:"total_errors"`
	LastActivityTime string `json:"last_activity"`
}

// NewVerificationWorkflow creates a new workflow.
func NewVerificationWorkflow(agent *agentsdk.Agent, evaluator agentsdk.ClaimEvaluator) *VerificationWorkflow {
	return &VerificationWorkflow{
		agent:           agent,
		evaluator:       evaluator,
		processedRounds: make(map[string]bool),
	}
}

// Run starts the verification loop. It blocks until the context is cancelled.
func (vw *VerificationWorkflow) Run(ctx context.Context) error {
	qc := vw.agent.QueryClient()
	if qc == nil {
		return fmt.Errorf("query client not initialized")
	}

	ticker := time.NewTicker(agentsdk.BlockTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := vw.processRounds(ctx); err != nil {
				vw.metrics.mu.Lock()
				vw.metrics.TotalErrors++
				vw.metrics.mu.Unlock()
			}
		}
	}
}

// processRounds checks for active rounds and processes them.
func (vw *VerificationWorkflow) processRounds(ctx context.Context) error {
	qc := vw.agent.QueryClient()
	rounds, err := qc.ActiveRounds(ctx)
	if err != nil {
		return fmt.Errorf("querying rounds: %w", err)
	}

	for _, round := range rounds {
		vw.mu.Lock()
		roundKey := fmt.Sprintf("%s-%s", round.ID, round.Phase)
		if vw.processedRounds[roundKey] {
			vw.mu.Unlock()
			continue
		}
		vw.mu.Unlock()

		switch round.Phase {
		case "commit":
			if err := vw.handleCommitPhase(ctx, round); err != nil {
				continue
			}
		case "reveal":
			if err := vw.handleRevealPhase(ctx, round); err != nil {
				continue
			}
		}

		vw.mu.Lock()
		vw.processedRounds[roundKey] = true
		vw.mu.Unlock()

		vw.metrics.mu.Lock()
		vw.metrics.LastActivityTime = time.Now().UTC().Format(time.RFC3339)
		vw.metrics.mu.Unlock()
	}

	return nil
}

// handleCommitPhase evaluates a claim and submits a commitment.
func (vw *VerificationWorkflow) handleCommitPhase(ctx context.Context, round agentsdk.RoundInfo) error {
	// Evaluate the claim
	verdict, confidence, err := vw.evaluator.Evaluate(ctx, round.ClaimID)
	if err != nil {
		return fmt.Errorf("evaluating claim %s: %w", round.ClaimID, err)
	}

	vw.metrics.mu.Lock()
	vw.metrics.TotalEvaluated++
	vw.metrics.mu.Unlock()

	// Submit commitment
	_, err = vw.agent.Verify(ctx, round.ID, verdict, confidence)
	if err != nil {
		return fmt.Errorf("committing to round %s: %w", round.ID, err)
	}

	vw.metrics.mu.Lock()
	vw.metrics.TotalCommitted++
	vw.metrics.mu.Unlock()

	return nil
}

// handleRevealPhase reveals a previously committed verification.
func (vw *VerificationWorkflow) handleRevealPhase(ctx context.Context, round agentsdk.RoundInfo) error {
	// Check if we have a salt (means we committed)
	if _, ok := vw.agent.GetSalt(round.ID); !ok {
		return nil // We didn't commit to this round
	}

	// Re-evaluate (must be deterministic)
	verdict, confidence, err := vw.evaluator.Evaluate(ctx, round.ClaimID)
	if err != nil {
		return fmt.Errorf("re-evaluating claim %s: %w", round.ClaimID, err)
	}

	// Submit reveal
	_, err = vw.agent.Verify(ctx, round.ID, verdict, confidence)
	if err != nil {
		return fmt.Errorf("revealing for round %s: %w", round.ID, err)
	}

	vw.metrics.mu.Lock()
	vw.metrics.TotalRevealed++
	vw.metrics.mu.Unlock()

	return nil
}

// Metrics returns the current verification metrics.
func (vw *VerificationWorkflow) Metrics() VerificationMetrics {
	vw.metrics.mu.Lock()
	defer vw.metrics.mu.Unlock()
	return VerificationMetrics{
		TotalEvaluated:   vw.metrics.TotalEvaluated,
		TotalCommitted:   vw.metrics.TotalCommitted,
		TotalRevealed:    vw.metrics.TotalRevealed,
		TotalErrors:      vw.metrics.TotalErrors,
		LastActivityTime: vw.metrics.LastActivityTime,
	}
}

// --- Built-in Evaluators ---

// SimpleEvaluator is a basic evaluator that returns a fixed verdict.
// Useful for testing and as a starting point for custom evaluators.
type SimpleEvaluator struct {
	DefaultVerdict    agentsdk.Verdict
	DefaultConfidence uint32
}

func (e *SimpleEvaluator) Evaluate(ctx context.Context, claimID string) (agentsdk.Verdict, uint32, error) {
	return e.DefaultVerdict, e.DefaultConfidence, nil
}

// DomainEvaluator evaluates claims based on domain knowledge.
// This is the hook point for AI-powered evaluation.
type DomainEvaluator struct {
	// EvalFunc is called for each claim. Implement with your AI model.
	EvalFunc func(ctx context.Context, claimID string) (agentsdk.Verdict, uint32, error)
}

func (e *DomainEvaluator) Evaluate(ctx context.Context, claimID string) (agentsdk.Verdict, uint32, error) {
	if e.EvalFunc == nil {
		return agentsdk.VerdictUncertain, 500000, nil
	}
	return e.EvalFunc(ctx, claimID)
}
