package agentsdk

import (
	"context"
	"fmt"
	"time"
)

// --- High-Level Agent Workflows ---
// These compose the low-level client calls into agent-friendly operations.

// Register performs full agent onboarding:
// 1. Register account (x/auth)
// 2. Register agent profile (x/discovery)
// 3. Start heartbeat daemon (optional)
func (a *Agent) Register(ctx context.Context, displayName string, domains []string, stake uint64) error {
	if a.txClient == nil {
		return fmt.Errorf("tx client not initialized — call SetTxClient first")
	}

	// Step 1: Register on-chain account
	if _, err := a.txClient.RegisterAccount(ctx, a.PubKeyHex()); err != nil {
		// Account may already exist — non-fatal
		_ = err
	}

	// Wait for tx inclusion
	time.Sleep(3 * time.Second)

	// Step 2: Register agent profile
	req := RegisterAgentRequest{
		DisplayName: displayName,
		Domains:     domains,
		Stake:       stake,
		Description: fmt.Sprintf("PicoClaw agent: %s", a.did),
	}
	if _, err := a.txClient.RegisterAgent(ctx, req); err != nil {
		return fmt.Errorf("registering agent: %w", err)
	}

	// Step 3: Start heartbeat if configured
	if a.cfg.AutoHeartbeat {
		a.StartHeartbeat(ctx)
	}

	return nil
}

// SubmitClaim submits a knowledge claim and returns the tx hash.
func (a *Agent) SubmitClaim(ctx context.Context, text, domain, category string, stake uint64) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	if len(text) < MinClaimLen {
		return "", fmt.Errorf("claim text must be at least %d characters (got %d)", MinClaimLen, len(text))
	}

	return a.txClient.SubmitClaim(ctx, SubmitClaimRequest{
		Text:     text,
		Domain:   domain,
		Category: category,
		Stake:    stake,
	})
}

// Verify performs a two-phase commit-reveal verification on a round.
// Call once to commit, call again to reveal.
func (a *Agent) Verify(ctx context.Context, roundID string, verdict Verdict, confidence uint32) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}

	// Check if we already committed (have a salt stored)
	if salt, ok := a.GetSalt(roundID); ok {
		// Phase 2: Reveal
		txHash, err := a.txClient.SubmitReveal(ctx, roundID, verdict, confidence, salt)
		if err != nil {
			return "", fmt.Errorf("submitting reveal: %w", err)
		}
		a.ClearSalt(roundID)
		return txHash, nil
	}

	// Phase 1: Commit
	salt, err := a.GenerateSalt(roundID)
	if err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	commitment := ComputeCommitment(verdict, confidence, salt)
	txHash, err := a.txClient.SubmitCommitment(ctx, roundID, commitment)
	if err != nil {
		a.ClearSalt(roundID) // Clean up on failure
		return "", fmt.Errorf("submitting commitment: %w", err)
	}

	return txHash, nil
}

// AutoVerify polls for pending rounds and auto-verifies them.
// This is the main verification loop for a verifier agent.
func (a *Agent) AutoVerify(ctx context.Context, evaluator ClaimEvaluator) error {
	if a.queryClient == nil || a.txClient == nil {
		return fmt.Errorf("clients not initialized")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rounds, err := a.queryClient.ActiveRounds(ctx)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, round := range rounds {
			switch round.Phase {
			case "commit":
				// Evaluate the claim and commit
				verdict, confidence, err := evaluator.Evaluate(ctx, round.ClaimID)
				if err != nil {
					continue // Skip claims we can't evaluate
				}
				if _, err := a.Verify(ctx, round.ID, verdict, confidence); err != nil {
					continue
				}

			case "reveal":
				// Reveal if we committed
				if _, ok := a.GetSalt(round.ID); ok {
					// We need the original verdict — evaluator should be deterministic
					verdict, confidence, err := evaluator.Evaluate(ctx, round.ClaimID)
					if err != nil {
						continue
					}
					if _, err := a.Verify(ctx, round.ID, verdict, confidence); err != nil {
						continue
					}
				}
			}
		}

		// Poll interval: ~1 block time
		time.Sleep(BlockTime)
	}
}

// ClaimEvaluator is the interface agents implement to evaluate claims.
// This is where the AI reasoning happens.
type ClaimEvaluator interface {
	// Evaluate determines the verdict and confidence for a claim.
	// Must be deterministic between commit and reveal phases.
	Evaluate(ctx context.Context, claimID string) (Verdict, uint32, error)
}

// StartHeartbeat begins periodic heartbeat transmission.
func (a *Agent) StartHeartbeat(ctx context.Context) {
	if a.heartbeatCancel != nil {
		a.heartbeatCancel() // Stop existing
	}

	hbCtx, cancel := context.WithCancel(ctx)
	a.heartbeatCancel = cancel

	go func() {
		ticker := time.NewTicker(a.cfg.HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				if a.txClient != nil {
					_, _ = a.txClient.Heartbeat(hbCtx)
				}
			}
		}
	}()
}

// StopHeartbeat stops the heartbeat daemon.
func (a *Agent) StopHeartbeat() {
	if a.heartbeatCancel != nil {
		a.heartbeatCancel()
		a.heartbeatCancel = nil
	}
}

// SyncHome updates the Agent Home's memory CID.
// This is called periodically to sync local workspace state to on-chain.
func (a *Agent) SyncHome(ctx context.Context, homeID, memoryCID string) error {
	if a.txClient == nil {
		return fmt.Errorf("tx client not initialized")
	}
	_, err := a.txClient.UpdateMemoryCID(ctx, homeID, memoryCID)
	return err
}

// QueryBalance returns the agent's current ulgm balance.
func (a *Agent) QueryBalance(ctx context.Context) (uint64, error) {
	if a.queryClient == nil {
		return 0, fmt.Errorf("query client not initialized")
	}
	return a.queryClient.Balance(ctx, a.address)
}

// QueryProfile returns the agent's discovery profile.
func (a *Agent) QueryProfile(ctx context.Context) (*AgentProfileInfo, error) {
	if a.queryClient == nil {
		return nil, fmt.Errorf("query client not initialized")
	}
	return a.queryClient.AgentProfile(ctx, a.address)
}

// --- BVM Workflows ---

// DeployContract deploys BVM bytecode and returns the contract address.
func (a *Agent) DeployContract(ctx context.Context, bytecodeHex string) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	return a.txClient.DeployContract(ctx, bytecodeHex)
}

// CallContract calls a deployed BVM contract.
func (a *Agent) CallContract(ctx context.Context, contractAddr, inputDataHex string, gasLimit uint64) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	return a.txClient.CallContract(ctx, contractAddr, inputDataHex, gasLimit)
}

// ScheduleExecution schedules a BVM contract call at a future block.
func (a *Agent) ScheduleExecution(ctx context.Context, contractAddr, method, payload string, atBlock, maxGas uint64) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	return a.txClient.ScheduleExecution(ctx, contractAddr, method, payload, atBlock, maxGas)
}

// --- Home Lifecycle Workflows ---

// CreateAndConfigureHome creates an Agent Home and configures its guardian.
func (a *Agent) CreateAndConfigureHome(ctx context.Context, name, defenseStrategy string, deadmanThreshold uint64) (string, error) {
	if a.txClient == nil {
		return "", fmt.Errorf("tx client not initialized")
	}
	homeID, err := a.txClient.CreateHome(ctx, name)
	if err != nil {
		return "", fmt.Errorf("creating home: %w", err)
	}
	// Wait for tx inclusion
	time.Sleep(3 * time.Second)
	// Configure guardian
	if _, err := a.txClient.ConfigureGuardian(ctx, homeID, defenseStrategy, true, deadmanThreshold); err != nil {
		return homeID, fmt.Errorf("configuring guardian (home created as %s): %w", homeID, err)
	}
	return homeID, nil
}

// ConfigureGuardian configures the Agent Home's guardian settings.
func (a *Agent) ConfigureGuardian(ctx context.Context, homeID, strategy string, autoDefend bool, deadmanThreshold uint64) error {
	if a.txClient == nil {
		return fmt.Errorf("tx client not initialized")
	}
	_, err := a.txClient.ConfigureGuardian(ctx, homeID, strategy, autoDefend, deadmanThreshold)
	return err
}

// SetSpendingLimit sets a spending limit on a Home key type.
func (a *Agent) SetSpendingLimit(ctx context.Context, homeID, keyType string, maxAmount, periodBlocks uint64) error {
	if a.txClient == nil {
		return fmt.Errorf("tx client not initialized")
	}
	_, err := a.txClient.SetSpendingLimit(ctx, homeID, keyType, maxAmount, periodBlocks)
	return err
}
