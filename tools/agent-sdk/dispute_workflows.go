package agentsdk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// --- Dispute Workflows ---
// These compose dispute primitives into agent-friendly operations
// for challenging facts, submitting evidence, and arbitrating disputes.

// DisputeFact initiates a dispute against a fact with a reason and evidence.
// Performs: initiate → commit evidence → (wait) → reveal evidence.
// Returns the dispute tx hash.
func (a *Agent) DisputeFact(ctx context.Context, factID, reason string, stake uint64, evidence []string) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if reason == "" {
		return "", fmt.Errorf("dispute reason is required")
	}
	if stake == 0 {
		return "", fmt.Errorf("dispute stake must be > 0")
	}

	// Step 1: Initiate the dispute.
	disputeTx, err := a.txClient.InitiateDispute(ctx, InitiateDisputeRequest{
		TargetID: factID,
		Reason:   reason,
		Stake:    stake,
	})
	if err != nil {
		return "", fmt.Errorf("initiating dispute: %w", err)
	}

	// If no evidence to submit, return immediately.
	if len(evidence) == 0 {
		return disputeTx, nil
	}

	// Wait for tx inclusion before committing evidence.
	time.Sleep(3 * time.Second)

	// Step 2: Commit evidence hash.
	nonce, err := a.GenerateSalt("dispute-" + disputeTx)
	if err != nil {
		return disputeTx, fmt.Errorf("generating evidence nonce: %w", err)
	}
	commitHash := computeEvidenceCommitment(evidence, nonce)
	if _, err := a.txClient.CommitDisputeEvidence(ctx, disputeTx, commitHash); err != nil {
		return disputeTx, fmt.Errorf("committing evidence: %w", err)
	}

	// Wait for commit to be included, then reveal.
	time.Sleep(3 * time.Second)

	// Step 3: Reveal evidence.
	if _, err := a.txClient.RevealDisputeEvidence(ctx, RevealDisputeEvidenceRequest{
		DisputeID: disputeTx,
		Evidence:  evidence,
		Nonce:     nonce,
	}); err != nil {
		return disputeTx, fmt.Errorf("revealing evidence: %w", err)
	}

	return disputeTx, nil
}

// QueryActiveDisputes returns all non-resolved disputes.
func (a *Agent) QueryActiveDisputes(ctx context.Context) ([]DisputeInfo, error) {
	if a.queryClient == nil {
		return nil, fmt.Errorf("query client not initialized")
	}
	return a.queryClient.ActiveDisputes(ctx)
}

// QueryDisputesByFact returns all disputes targeting a specific fact.
func (a *Agent) QueryDisputesByFact(ctx context.Context, factID string) ([]DisputeInfo, error) {
	if a.queryClient == nil {
		return nil, fmt.Errorf("query client not initialized")
	}
	return a.queryClient.DisputesByFact(ctx, factID)
}

// VoteOnDispute casts an arbiter vote on a dispute.
func (a *Agent) VoteOnDispute(ctx context.Context, disputeID string, vote DisputeVote, reasoning string) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if reasoning == "" {
		return "", fmt.Errorf("arbiter reasoning is required")
	}
	return a.txClient.ArbiterVote(ctx, ArbiterVoteRequest{
		DisputeID: disputeID,
		Vote:      vote,
		Reasoning: reasoning,
	})
}

// AutoArbiter polls for disputes in the voting phase and evaluates them.
// This is the main loop for arbiter agents, analogous to AutoVerify for verifiers.
func (a *Agent) AutoArbiter(ctx context.Context, evaluator DisputeEvaluator) error {
	if a.queryClient == nil || a.txClient == nil {
		return fmt.Errorf("clients not initialized")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		disputes, err := a.queryClient.ActiveDisputes(ctx)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, d := range disputes {
			if d.Status != "voting" {
				continue
			}

			vote, reasoning, err := evaluator.EvaluateDispute(ctx, d.ID)
			if err != nil {
				continue // Skip disputes we can't evaluate
			}

			if _, err := a.txClient.ArbiterVote(ctx, ArbiterVoteRequest{
				DisputeID: d.ID,
				Vote:      vote,
				Reasoning: reasoning,
			}); err != nil {
				continue
			}
		}

		time.Sleep(BlockTime * 5)
	}
}

// DisputeEvaluator is the interface arbiter agents implement to evaluate disputes.
type DisputeEvaluator interface {
	// EvaluateDispute determines the vote and reasoning for a dispute.
	EvaluateDispute(ctx context.Context, disputeID string) (DisputeVote, string, error)
}

// computeEvidenceCommitment computes a commitment hash for dispute evidence.
func computeEvidenceCommitment(evidence []string, nonce string) string {
	h := sha256.New()
	h.Write([]byte("LGM.dispute-evidence.v1:"))
	for _, e := range evidence {
		writeLP(h, []byte(e))
	}
	writeLP(h, []byte(nonce))
	return hex.EncodeToString(h.Sum(nil))
}
