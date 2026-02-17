package agentsdk

import (
	"context"
	"fmt"
	"time"
)

// --- Bug Bounty Workflows ---
// These compose the claim lifecycle, governance, and vesting reward primitives
// into an end-to-end bug bounty flow: Report → Verify → Fix → Bounty.

// SubmitBugReport submits a bug report as a protocol_observation claim.
// Returns the tx hash of the submitted claim.
func (a *Agent) SubmitBugReport(ctx context.Context, description, domain string, stake uint64) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if len(description) < MinClaimLen {
		return "", fmt.Errorf("bug report must be at least %d characters (got %d)", MinClaimLen, len(description))
	}

	return a.txClient.SubmitClaim(ctx, SubmitClaimRequest{
		Text:     description,
		Domain:   domain,
		Category: "protocol_observation",
		Stake:    stake,
	})
}

// SubmitBugFix submits a fix for verified bug(s) as both a claim and a governance proposal (LIP).
// Returns the LIP tx hash.
func (a *Agent) SubmitBugFix(ctx context.Context, req BugFixRequest) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if len(req.BugFactIDs) == 0 {
		return "", fmt.Errorf("must reference at least one verified bug fact")
	}
	if len(req.FixDescription) < MinClaimLen {
		return "", fmt.Errorf("fix description must be at least %d characters (got %d)", MinClaimLen, len(req.FixDescription))
	}

	// Step 1: Submit the fix as a knowledge claim referencing the bug facts.
	fixClaimTx, err := a.txClient.SubmitClaim(ctx, SubmitClaimRequest{
		Text:     req.FixDescription,
		Domain:   "protocol",
		Category: "protocol_observation",
		Stake:    req.Stake,
	})
	if err != nil {
		return "", fmt.Errorf("submitting fix claim: %w", err)
	}

	// Wait for tx inclusion before submitting the LIP.
	time.Sleep(3 * time.Second)

	// Step 2: Submit governance proposal (LIP) linking fix to bug facts.
	lipTx, err := a.txClient.SubmitLIP(ctx, SubmitLIPRequest{
		Title:       fmt.Sprintf("BugFix: %s", req.Title),
		Description: req.FixDescription,
		BugFactIDs:  req.BugFactIDs,
		FixClaimID:  fixClaimTx,
		Deposit:     req.Deposit,
	})
	if err != nil {
		return "", fmt.Errorf("submitting LIP: %w", err)
	}

	return lipTx, nil
}

// DistributeBounty distributes bounty payouts to bug finders and fix author
// via vesting reward creation. Called by Guardian agents after LIP approval.
func (a *Agent) DistributeBounty(ctx context.Context, req BountyDistribution) error {
	if a.txClient == nil {
		return fmt.Errorf("tx client not initialized")
	}
	if len(req.BugFinders) == 0 && req.FixAuthor == "" {
		return fmt.Errorf("must specify at least one bug finder or fix author")
	}

	// Pay bug finder(s).
	for _, finder := range req.BugFinders {
		if finder.Share == 0 {
			continue
		}
		_, err := a.txClient.CreateVesting(ctx, CreateVestingRequest{
			Beneficiary: finder.Address,
			Amount:      finder.Share,
			Source:      "bounty_fulfillment",
			Category:    "protocol_observation",
			Description: fmt.Sprintf("Bug bounty: %v", req.BugFactIDs),
		})
		if err != nil {
			return fmt.Errorf("paying bug finder %s: %w", finder.Address, err)
		}
	}

	// Pay fix author.
	if req.FixAuthor != "" && req.FixShare > 0 {
		_, err := a.txClient.CreateVesting(ctx, CreateVestingRequest{
			Beneficiary: req.FixAuthor,
			Amount:      req.FixShare,
			Source:      "bounty_fulfillment",
			Category:    "protocol_observation",
			Description: fmt.Sprintf("Fix bounty: LIP %s", req.LIPID),
		})
		if err != nil {
			return fmt.Errorf("paying fix author %s: %w", req.FixAuthor, err)
		}
	}

	return nil
}

// QueryBugFacts returns verified protocol_observation facts for a domain.
func (a *Agent) QueryBugFacts(ctx context.Context, domain string) ([]FactInfo, error) {
	if a.queryClient == nil {
		return nil, fmt.Errorf("query client not initialized")
	}
	return a.queryClient.BugFacts(ctx, domain)
}

// QueryProposalsByTrigger returns governance proposals by trigger type.
func (a *Agent) QueryProposalsByTrigger(ctx context.Context, triggerType string) ([]ProposalInfo, error) {
	if a.queryClient == nil {
		return nil, fmt.Errorf("query client not initialized")
	}
	return a.queryClient.ProposalsByTrigger(ctx, triggerType)
}

// AutoBugBounty runs the bug bounty lifecycle for the given role.
//   - Scanner: polls for new bugs to report (caller provides scan logic via evaluator)
//   - Verifier: reuses AutoVerify with the given evaluator
//   - Fixer: monitors autopoiesis_proposal_suggested events and submits fixes
//   - Guardian: monitors LIP approval and distributes bounties
func (a *Agent) AutoBugBounty(ctx context.Context, role BountyRole, evaluator ClaimEvaluator) error {
	switch role {
	case BountyRoleVerifier:
		return a.AutoVerify(ctx, evaluator)
	case BountyRoleScanner:
		return a.runScanner(ctx)
	case BountyRoleFixer:
		return a.runFixer(ctx)
	case BountyRoleGuardian:
		return a.runBountyGuardian(ctx)
	default:
		return fmt.Errorf("unknown bounty role: %s", role)
	}
}

// runScanner polls for autopoiesis events indicating critical protocol observations.
// In production, the scanner would also run static analysis or fuzzing;
// here it monitors the chain for verified bug facts.
func (a *Agent) runScanner(ctx context.Context) error {
	if a.queryClient == nil {
		return fmt.Errorf("query client not initialized")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Scanners poll for domain facts to identify patterns.
		// Actual bug discovery logic is application-specific and external.
		// This loop keeps the scanner alive and responsive to cancellation.
		time.Sleep(BlockTime * 10) // Poll every ~25 seconds
	}
}

// runFixer monitors for autopoiesis proposal events and could submit fixes.
// The actual fix submission requires external input (the fix itself).
func (a *Agent) runFixer(ctx context.Context) error {
	if a.queryClient == nil {
		return fmt.Errorf("query client not initialized")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Poll for proposals triggered by openclaw_critical.
		proposals, err := a.queryClient.ProposalsByTrigger(ctx, "openclaw_critical")
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		// Log/signal proposals that need fixes.
		// Actual fix generation is application-specific.
		_ = proposals

		time.Sleep(BlockTime * 10)
	}
}

// runBountyGuardian monitors for approved LIPs and distributes bounties.
func (a *Agent) runBountyGuardian(ctx context.Context) error {
	if a.queryClient == nil || a.txClient == nil {
		return fmt.Errorf("clients not initialized")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Poll for passed proposals from the openclaw_critical trigger.
		proposals, err := a.queryClient.ProposalsByTrigger(ctx, "openclaw_critical")
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, p := range proposals {
			if p.Status == "passed" {
				// In production, parse the proposal to extract bug fact IDs,
				// fix author, and calculate shares. This is a monitoring loop.
				_ = p
			}
		}

		time.Sleep(BlockTime * 10)
	}
}
