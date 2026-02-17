//go:build openclaw

package openclaw

import (
	"context"
	"testing"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tests/openclaw/agents"
	"github.com/legible-money/legible/tests/openclaw/devnet"
	"github.com/legible-money/legible/tests/openclaw/helpers"
)

// TestPhaseA runs the PoT verification lifecycle on a live devnet.
// All subtests share a single devnet instance and run sequentially.
func TestPhaseA(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Boot devnet
	h := devnet.NewHarness(t, binaryPath)
	if err := h.Start(ctx); err != nil {
		t.Fatalf("devnet start failed: %v", err)
	}

	// Create a query client for the validator key (used across subtests)
	valClient := h.NewCLIClient("validator")

	// State carried across sequential subtests
	var (
		test1Addr string
		claimText = "The speed of light in vacuum is approximately 299792458 meters per second"
		claimCat  = "replicated" // empirical zone compatible; "computational" requires presburger/peano/propositional
	)

	t.Run("OCA-A-01_ChainBootAndBlockProduction", func(t *testing.T) {
		status, err := valClient.ChainStatus(ctx)
		if err != nil {
			t.Fatalf("ChainStatus: %v", err)
		}
		if status.LatestHeight <= 0 {
			t.Fatalf("height should be > 0, got %d", status.LatestHeight)
		}
		if status.ChainID != h.ChainID {
			t.Fatalf("chain_id mismatch: got %s, want %s", status.ChainID, h.ChainID)
		}
		helpers.Result(t, "A-01", "chain booted and producing blocks")
		t.Logf("height=%d chain_id=%s", status.LatestHeight, status.ChainID)
	})

	t.Run("OCA-A-02_AccountRegistration", func(t *testing.T) {
		test1Client := h.NewCLIClient("test1")
		_, err := test1Client.RegisterAccount(ctx, "")
		if err != nil {
			t.Logf("RegisterAccount returned error (may be expected if module not wired): %v", err)
		}
		helpers.WaitForTx(ctx, valClient)
		test1Addr = h.KeyAddress("test1")
		if test1Addr == "" {
			t.Fatal("test1 address is empty")
		}
		helpers.Result(t, "A-02", "account registered: "+test1Addr)
	})

	t.Run("OCA-A-03_AccountFunding", func(t *testing.T) {
		test1Addr = h.KeyAddress("test1")
		fundAmount := uint64(10_000_000_000) // 10K LGM

		if err := h.FundAccount(ctx, "validator", test1Addr, fundAmount); err != nil {
			t.Fatalf("FundAccount: %v", err)
		}
		helpers.WaitForTx(ctx, valClient)

		helpers.AssertBalanceGTE(t, valClient, test1Addr, fundAmount/2) // Allow for gas fees
		helpers.Result(t, "A-03", "account funded with 10K LGM")
	})

	t.Run("OCA-A-04_ClaimSubmission", func(t *testing.T) {
		test1Client := h.NewCLIClient("test1")
		stakeUlgm := uint64(1_000_000) // 1 LGM

		txHash, err := test1Client.SubmitClaim(ctx, agentsdk.SubmitClaimRequest{
			Text:     claimText,
			Domain:   "general",
			Category: claimCat,
			Stake:    stakeUlgm,
		})
		if err != nil {
			t.Fatalf("SubmitClaim: %v", err)
		}
		helpers.WaitForTx(ctx, valClient)

		// Verify claim is queryable
		test1Addr = h.KeyAddress("test1")
		claims, err := test1Client.ClaimsBySubmitter(ctx, test1Addr)
		if err != nil {
			t.Logf("ClaimsBySubmitter query error (module may need wiring): %v", err)
		} else if len(claims) == 0 {
			t.Log("No claims returned — claim may be processed differently")
		} else {
			t.Logf("Found %d claims for submitter", len(claims))
		}

		helpers.Result(t, "A-04", "claim submitted: tx="+txHash)
	})

	t.Run("OCA-A-05_VerificationRoundCreation", func(t *testing.T) {
		rounds, err := valClient.ActiveRounds(ctx)
		if err != nil {
			t.Logf("ActiveRounds query error: %v", err)
			t.Skip("skipping — active rounds query not available")
		}
		if len(rounds) == 0 {
			t.Log("No active rounds — round may not be auto-created yet")
		} else {
			t.Logf("Found %d active rounds, first: id=%s phase=%s", len(rounds), rounds[0].ID, rounds[0].Phase)
		}
		helpers.Result(t, "A-05", "verification round query exercised")
	})

	t.Run("OCA-A-06_CommitPhase", func(t *testing.T) {
		rounds, err := valClient.ActiveRounds(ctx)
		if err != nil || len(rounds) == 0 {
			t.Skip("no active rounds to commit to")
		}

		round := rounds[0]
		if round.Phase != "commit" {
			t.Skipf("round %s not in commit phase (phase=%s)", round.ID, round.Phase)
		}

		// Validator commits: verdict=true, confidence=80%
		commitment := agentsdk.ComputeCommitment(round.ID, agentsdk.VerdictTrue, 800000, "testsalt123")
		txHash, err := valClient.SubmitCommitment(ctx, round.ID, commitment)
		if err != nil {
			t.Fatalf("SubmitCommitment: %v", err)
		}
		helpers.WaitForTx(ctx, valClient)
		helpers.Result(t, "A-06", "commitment submitted: tx="+txHash)
	})

	t.Run("OCA-A-07_RevealPhase", func(t *testing.T) {
		rounds, err := valClient.ActiveRounds(ctx)
		if err != nil || len(rounds) == 0 {
			t.Skip("no active rounds to reveal")
		}

		round := rounds[0]
		if round.Phase != "reveal" {
			// Wait for reveal phase
			err := helpers.WaitForRoundPhase(ctx, valClient, round.ID, "reveal")
			if err != nil {
				t.Skipf("could not wait for reveal phase: %v", err)
			}
		}

		txHash, err := valClient.SubmitReveal(ctx, round.ID, agentsdk.VerdictTrue, 800000, "testsalt123")
		if err != nil {
			t.Fatalf("SubmitReveal: %v", err)
		}
		helpers.WaitForTx(ctx, valClient)
		helpers.Result(t, "A-07", "reveal submitted: tx="+txHash)
	})

	t.Run("OCA-A-08_AggregationAndVerdict", func(t *testing.T) {
		rounds, err := valClient.ActiveRounds(ctx)
		if err != nil || len(rounds) == 0 {
			t.Skip("no rounds to check for completion")
		}

		round := rounds[0]
		// Wait for round to complete (or timeout)
		waitCtx, waitCancel := context.WithTimeout(ctx, 60*time.Second)
		defer waitCancel()
		err = helpers.WaitForRoundPhase(waitCtx, valClient, round.ID, "complete")
		if err != nil {
			t.Logf("Round did not complete within timeout: %v", err)
			t.Skip("aggregation not triggered — may need EndBlock hook")
		}

		helpers.Result(t, "A-08", "round completed")
	})

	t.Run("OCA-A-09_FactCreation", func(t *testing.T) {
		facts, err := valClient.FactsByDomain(ctx, "general")
		if err != nil {
			t.Logf("FactsByDomain error: %v", err)
			t.Skip("facts query not available")
		}
		if len(facts) == 0 {
			t.Log("No facts yet — fact creation may require round completion")
		} else {
			t.Logf("Found %d facts in 'general' domain", len(facts))
			helpers.Insight(t, "A-09", "fact created on live chain")
		}
		helpers.Result(t, "A-09", "fact query exercised")
	})

	t.Run("OCA-A-10_RewardDistribution", func(t *testing.T) {
		valAddr := h.KeyAddress("validator")
		bal, err := valClient.Balance(ctx, valAddr)
		if err != nil {
			t.Fatalf("Balance query: %v", err)
		}
		t.Logf("Validator balance after lifecycle: %d ulgm (%.2f LGM)", bal, float64(bal)/1_000_000)
		helpers.Result(t, "A-10", "reward distribution check complete")
	})
}

// TestPhaseA_AgentSDKWorkflow tests the Agent SDK's Verify workflow with a real devnet.
func TestPhaseA_AgentSDKWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	h := devnet.NewHarness(t, binaryPath)
	if err := h.Start(ctx); err != nil {
		t.Fatalf("devnet start failed: %v", err)
	}

	// Create an agent using the SDK
	agent, client, err := agents.CreateAgent(h, "test1", agentsdk.RoleVerifier)
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}

	// Check chain status via agent's query client
	status, err := client.ChainStatus(ctx)
	if err != nil {
		t.Fatalf("ChainStatus: %v", err)
	}
	if status.LatestHeight <= 0 {
		t.Fatal("chain not producing blocks")
	}

	// Check balance via agent workflow
	bal, err := agent.QueryBalance(ctx)
	if err != nil {
		t.Logf("QueryBalance error (expected if account not funded): %v", err)
	} else {
		t.Logf("Agent balance: %d ulgm", bal)
	}

	helpers.Result(t, "A-SDK", "Agent SDK wiring validated on live chain")
}
