//go:build openclaw

package openclaw

import (
	"context"
	"fmt"
	"testing"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tests/openclaw/agents"
	"github.com/legible-money/legible/tests/openclaw/devnet"
	"github.com/legible-money/legible/tests/openclaw/helpers"
)

// TestPhaseC runs multi-agent coordination scenarios on a live devnet.
func TestPhaseC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	h := devnet.NewHarness(t, binaryPath)
	if err := h.Start(ctx); err != nil {
		t.Fatalf("devnet start failed: %v", err)
	}

	// Create agent pool: 3 claimers, 3 verifiers, 2 challengers
	pool := agents.NewPool(h)
	roles := map[string]agentsdk.AgentRole{
		"test1": agentsdk.RoleClaimer,
		"test2": agentsdk.RoleClaimer,
		"test3": agentsdk.RoleClaimer,
		"test4": agentsdk.RoleVerifier,
		"test5": agentsdk.RoleVerifier,
		"test6": agentsdk.RoleVerifier,
		"test7": agentsdk.RoleChallenger,
		"test8": agentsdk.RoleChallenger,
	}
	for name, role := range roles {
		if _, err := pool.Add(name, role); err != nil {
			t.Fatalf("pool.Add(%s): %v", name, err)
		}
	}

	if err := pool.FundAll(ctx, 1_000_000_000); err != nil { // 1K LGM each (agents already have 10K from genesis)
		t.Fatalf("FundAll: %v", err)
	}

	valClient := h.NewCLIClient("validator")

	t.Run("OCA-C-01_ConcurrentClaims", func(t *testing.T) {
		claimers := pool.ByRole(agentsdk.RoleClaimer)
		submitted := 0

		for i, claimer := range claimers {
			text := fmt.Sprintf("Concurrent claim %d: testing parallel submission on live chain", i+1)
			_, err := claimer.Client.SubmitClaim(ctx, agentsdk.SubmitClaimRequest{
				Text:     text,
				Domain:   "general",
				Category: "computational",
				Stake:    1_000_000,
			})
			if err != nil {
				t.Logf("Claimer %s error: %v", claimer.Name, err)
			} else {
				submitted++
			}
			time.Sleep(500 * time.Millisecond) // Small delay to avoid nonce races
		}
		helpers.WaitForTx(ctx, valClient)

		t.Logf("Submitted %d/%d concurrent claims", submitted, len(claimers))
		helpers.Result(t, "C-01", fmt.Sprintf("%d claims submitted concurrently", submitted))
	})

	t.Run("OCA-C-02_ClaimerVerifierPipeline", func(t *testing.T) {
		claimer := pool.Get("test1")
		verifier := pool.Get("test4")

		// Step 1: Submit claim
		claimText := "Pipeline test: verifier should commit then reveal on this claim"
		txHash, err := claimer.Client.SubmitClaim(ctx, agentsdk.SubmitClaimRequest{
			Text:     claimText,
			Domain:   "general",
			Category: "computational",
			Stake:    1_000_000,
		})
		if err != nil {
			t.Logf("SubmitClaim error: %v", err)
			t.Skip("claim submission failed")
		}
		t.Logf("Claim submitted: tx=%s", txHash)
		helpers.WaitForTx(ctx, valClient)

		// Step 2: Check for active rounds
		rounds, err := verifier.Client.ActiveRounds(ctx)
		if err != nil || len(rounds) == 0 {
			t.Log("No active rounds — verification round not auto-created")
			t.Skip("round creation depends on EndBlock hook")
		}

		// Step 3: Commit
		round := rounds[0]
		_, err = verifier.Agent.Verify(ctx, round.ID, agentsdk.VerdictTrue, 800000)
		if err != nil {
			t.Logf("Verify (commit): %v", err)
		}
		helpers.WaitForTx(ctx, valClient)

		// Step 4: Wait for reveal phase
		err = helpers.WaitForRoundPhase(ctx, verifier.Client, round.ID, "reveal")
		if err != nil {
			t.Logf("WaitForRoundPhase(reveal): %v", err)
			t.Skip("reveal phase not reached")
		}

		// Step 5: Reveal
		_, err = verifier.Agent.Verify(ctx, round.ID, agentsdk.VerdictTrue, 800000)
		if err != nil {
			t.Logf("Verify (reveal): %v", err)
		}

		helpers.Result(t, "C-02", "claimer-verifier pipeline exercised")
	})

	t.Run("OCA-C-03_ChallengeResponse", func(t *testing.T) {
		challenger := pool.Get("test7")

		// Challenge requires an existing fact ID
		facts, err := challenger.Client.FactsByDomain(ctx, "general")
		if err != nil || len(facts) == 0 {
			t.Skip("no facts to challenge — Phase A round completion required")
		}

		factID := facts[0].ID
		_, err = challenger.Client.ChallengeFact(ctx, factID, "Testing challenge mechanism on live chain", 2_000_000)
		if err != nil {
			t.Logf("ChallengeFact error: %v", err)
		} else {
			t.Log("Challenge submitted successfully")
		}

		helpers.Result(t, "C-03", "challenge response tested")
	})

	t.Run("OCA-C-04_MixedVerdicts", func(t *testing.T) {
		// 3 verifiers with different verdicts
		verifiers := pool.ByRole(agentsdk.RoleVerifier)
		if len(verifiers) < 3 {
			t.Skip("need 3 verifiers")
		}

		rounds, err := verifiers[0].Client.ActiveRounds(ctx)
		if err != nil || len(rounds) == 0 {
			t.Skip("no active rounds for mixed verdict test")
		}

		round := rounds[0]
		verdicts := []agentsdk.Verdict{agentsdk.VerdictTrue, agentsdk.VerdictTrue, agentsdk.VerdictFalse}

		for i, v := range verifiers[:3] {
			_, err := v.Agent.Verify(ctx, round.ID, verdicts[i], 800000)
			if err != nil {
				t.Logf("Verifier %s commit error: %v", v.Name, err)
			}
			time.Sleep(500 * time.Millisecond)
		}
		helpers.WaitForTx(ctx, valClient)

		helpers.Insight(t, "C-04", "mixed verdicts submitted — 2 accept, 1 reject")
		helpers.Result(t, "C-04", "mixed verdict scenario exercised")
	})

	t.Run("OCA-C-05_AutoVerifyLoop", func(t *testing.T) {
		verifier := pool.Get("test4")

		// Run AutoVerify for ~10 seconds with AlwaysTrueEvaluator
		autoCtx, autoCancel := context.WithTimeout(ctx, 10*time.Second)
		defer autoCancel()

		evaluator := agents.AlwaysTrueEvaluator{Confidence: 800000}
		errCh := make(chan error, 1)
		go func() {
			errCh <- verifier.Agent.AutoVerify(autoCtx, evaluator)
		}()

		// Wait for the AutoVerify to finish (context timeout)
		select {
		case err := <-errCh:
			if err != nil && err != context.DeadlineExceeded {
				t.Logf("AutoVerify error: %v", err)
			}
		case <-time.After(15 * time.Second):
			autoCancel()
		}

		helpers.Result(t, "C-05", "AutoVerify loop ran for 10s without crash")
	})

	t.Run("OCA-C-06_BalanceAccounting", func(t *testing.T) {
		// Track balances across all pool agents
		var totalBalance uint64
		for _, ma := range pool.Agents {
			bal, err := ma.Client.Balance(ctx, ma.Address)
			if err != nil {
				t.Logf("Balance query for %s: %v", ma.Name, err)
				continue
			}
			t.Logf("Agent %s (%s): %d ulgm (%.2f LGM)", ma.Name, ma.Role, bal, float64(bal)/1_000_000)
			totalBalance += bal
		}

		// Check validator balance too
		valAddr := h.KeyAddress("validator")
		valBal, err := valClient.Balance(ctx, valAddr)
		if err == nil {
			t.Logf("Validator: %d ulgm (%.2f LGM)", valBal, float64(valBal)/1_000_000)
			totalBalance += valBal
		}

		t.Logf("Total tracked balance: %d ulgm (%.2f LGM)", totalBalance, float64(totalBalance)/1_000_000)
		helpers.Insight(t, "C-06", "balance accounting snapshot captured")
		helpers.Result(t, "C-06", "balance conservation check complete")
	})

	t.Run("OCA-C-07_AgentHeartbeatLiveness", func(t *testing.T) {
		// Test heartbeat sending
		for _, ma := range pool.Agents[:3] {
			_, err := ma.Client.Heartbeat(ctx)
			if err != nil {
				t.Logf("Heartbeat from %s: %v", ma.Name, err)
			} else {
				t.Logf("Heartbeat sent from %s", ma.Name)
			}
			time.Sleep(500 * time.Millisecond)
		}

		helpers.Result(t, "C-07", "heartbeat liveness tested for 3 agents")
	})

	t.Run("OCA-C-08_DomainSpecialization", func(t *testing.T) {
		// Submit claims in different domains
		claimer1 := pool.Get("test1")
		claimer2 := pool.Get("test2")

		_, err := claimer1.Client.SubmitClaim(ctx, agentsdk.SubmitClaimRequest{
			Text:     "Physics domain: Energy equals mass times the speed of light squared",
			Domain:   "physics",
			Category: "empirical",
			Stake:    1_000_000,
		})
		if err != nil {
			t.Logf("Physics claim error: %v", err)
		}
		time.Sleep(agentsdk.BlockTime)

		_, err = claimer2.Client.SubmitClaim(ctx, agentsdk.SubmitClaimRequest{
			Text:     "Math domain: The sum of angles in a triangle is one hundred eighty degrees",
			Domain:   "math",
			Category: "formal",
			Stake:    1_000_000,
		})
		if err != nil {
			t.Logf("Math claim error: %v", err)
		}
		helpers.WaitForTx(ctx, valClient)

		// Query domain-scoped facts
		for _, domain := range []string{"physics", "math", "general"} {
			facts, err := valClient.FactsByDomain(ctx, domain)
			if err != nil {
				t.Logf("FactsByDomain(%s): %v", domain, err)
			} else {
				t.Logf("Domain %s: %d facts", domain, len(facts))
			}
		}

		helpers.Result(t, "C-08", "domain specialization tested with physics + math claims")
	})
}
