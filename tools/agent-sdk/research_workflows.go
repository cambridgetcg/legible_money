package agentsdk

import (
	"context"
	"fmt"
	"time"
)

// --- Research & Evidence Workflows ---
// These compose research, evidence, and ontology primitives into
// agent-friendly operations for submitting research, peer reviewing,
// managing bounties, and submitting supporting evidence.

// SubmitResearchWithEvidence submits research and attaches supporting evidence.
// Returns the research submission tx hash.
func (a *Agent) SubmitResearchWithEvidence(ctx context.Context, req SubmitResearchRequest, evidence []SubmitEvidenceRequest) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if req.Title == "" {
		return "", fmt.Errorf("research title is required")
	}
	if len(req.Description) < MinClaimLen {
		return "", fmt.Errorf("research description must be at least %d characters (got %d)", MinClaimLen, len(req.Description))
	}

	// Step 1: Submit the research.
	resTx, err := a.txClient.SubmitResearch(ctx, req)
	if err != nil {
		return "", fmt.Errorf("submitting research: %w", err)
	}

	// Step 2: Attach supporting evidence (if any).
	if len(evidence) > 0 {
		time.Sleep(3 * time.Second)
		for i, ev := range evidence {
			if _, err := a.txClient.SubmitEvidence(ctx, ev); err != nil {
				return resTx, fmt.Errorf("submitting evidence %d: %w", i, err)
			}
		}
	}

	return resTx, nil
}

// PeerReview submits a peer review for a research entry.
func (a *Agent) PeerReview(ctx context.Context, req ReviewResearchRequest) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if req.ResearchID == "" {
		return "", fmt.Errorf("research ID is required")
	}
	if req.QualityScore > 100 {
		return "", fmt.Errorf("quality score must be 0-100 (got %d)", req.QualityScore)
	}
	if req.Reasoning == "" {
		return "", fmt.Errorf("review reasoning is required")
	}

	return a.txClient.ReviewResearch(ctx, req)
}

// CreateAndMonitorBounty creates a research bounty and optionally monitors it.
// Returns the bounty creation tx hash.
func (a *Agent) CreateAndMonitorBounty(ctx context.Context, req CreateResearchBountyRequest) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if req.Title == "" {
		return "", fmt.Errorf("bounty title is required")
	}
	if req.Reward == 0 {
		return "", fmt.Errorf("bounty reward must be > 0")
	}
	if req.DeadlineHeight == 0 {
		return "", fmt.Errorf("bounty deadline height must be > 0")
	}

	return a.txClient.CreateResearchBounty(ctx, req)
}

// SubmitSupportingEvidence submits evidence supporting a claim.
func (a *Agent) SubmitSupportingEvidence(ctx context.Context, req SubmitEvidenceRequest) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if req.ClaimID == "" {
		return "", fmt.Errorf("claim ID is required")
	}
	if req.ContentHash == "" {
		return "", fmt.Errorf("content hash is required")
	}
	if req.EvidenceType == "" {
		return "", fmt.Errorf("evidence type is required")
	}

	return a.txClient.SubmitEvidence(ctx, req)
}

// QueryResearchByDomain returns research submissions in a domain.
func (a *Agent) QueryResearchByDomain(ctx context.Context, domain string) ([]ResearchInfo, error) {
	if a.queryClient == nil {
		return nil, fmt.Errorf("query client not initialized")
	}
	return a.queryClient.ResearchSubmissions(ctx, "", domain)
}

// QueryOpenBounties returns open research bounties.
func (a *Agent) QueryOpenBounties(ctx context.Context) ([]ResearchInfo, error) {
	if a.queryClient == nil {
		return nil, fmt.Errorf("query client not initialized")
	}
	// Use status filter to get only open bounties via submissions query.
	return a.queryClient.ResearchSubmissions(ctx, "submitted", "")
}

// --- Ontology Workflows ---

// ProposeNewDomain proposes a new knowledge domain in the ontology.
func (a *Agent) ProposeNewDomain(ctx context.Context, req ProposeOntologyDomainRequest) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if req.Name == "" {
		return "", fmt.Errorf("domain name is required")
	}
	if req.Stratum > 6 {
		return "", fmt.Errorf("stratum must be 0-6 (got %d)", req.Stratum)
	}
	if req.Stake == 0 {
		return "", fmt.Errorf("proposal stake must be > 0")
	}

	return a.txClient.ProposeOntologyDomain(ctx, req)
}

// AcknowledgeGodelianIncompleteness records a Gödelian acknowledgment for a fact.
// This is used when a fact exists in a logically incomplete zone (Peano, set theory).
func (a *Agent) AcknowledgeGodelianIncompleteness(ctx context.Context, factID, zone, reason string) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if factID == "" {
		return "", fmt.Errorf("fact ID is required")
	}
	if zone == "" {
		return "", fmt.Errorf("logic zone is required")
	}
	if reason == "" {
		return "", fmt.Errorf("acknowledgment reason is required")
	}

	return a.txClient.AcknowledgeIncompleteness(ctx, factID, zone, reason)
}

// QueryDomainConfidence returns the confidence ceiling for a domain.
func (a *Agent) QueryDomainConfidence(ctx context.Context, domain string) (*ConfidenceCeilingInfo, error) {
	if a.queryClient == nil {
		return nil, fmt.Errorf("query client not initialized")
	}
	return a.queryClient.ConfidenceCeiling(ctx, domain)
}

// QueryLogicZones returns all registered logic zones.
func (a *Agent) QueryLogicZones(ctx context.Context) ([]LogicZoneInfo, error) {
	if a.queryClient == nil {
		return nil, fmt.Errorf("query client not initialized")
	}
	return a.queryClient.AllLogicZones(ctx)
}

// AutoResearchReviewer polls for research submissions needing review.
// This is the main loop for reviewer agents.
func (a *Agent) AutoResearchReviewer(ctx context.Context, evaluator ResearchEvaluator) error {
	if a.queryClient == nil || a.txClient == nil {
		return fmt.Errorf("clients not initialized")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		submissions, err := a.queryClient.ResearchSubmissions(ctx, "under_review", "")
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, s := range submissions {
			verdict, score, reasoning, err := evaluator.EvaluateResearch(ctx, s.ID)
			if err != nil {
				continue
			}

			if _, err := a.txClient.ReviewResearch(ctx, ReviewResearchRequest{
				ResearchID:   s.ID,
				Verdict:      verdict,
				QualityScore: score,
				Reasoning:    reasoning,
			}); err != nil {
				continue
			}
		}

		time.Sleep(BlockTime * 10)
	}
}

// ResearchEvaluator is the interface reviewer agents implement.
type ResearchEvaluator interface {
	// EvaluateResearch determines the verdict, quality score, and reasoning.
	EvaluateResearch(ctx context.Context, researchID string) (ReviewVerdict, uint32, string, error)
}
