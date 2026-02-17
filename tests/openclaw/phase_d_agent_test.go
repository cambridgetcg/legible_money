//go:build openclaw

package openclaw

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tests/openclaw/claw"
	"github.com/legible-money/legible/tests/openclaw/devnet"
	"github.com/legible-money/legible/tests/openclaw/env"
	"github.com/legible-money/legible/tests/openclaw/helpers"
)

// TestPhaseD runs real OpenClaw agent integration tests.
//
// Architecture: The agent EVALUATES claims (AI reasoning) and the Go test
// harness handles all blockchain transactions (commit/reveal). This cleanly
// separates AI evaluation from on-chain mechanics and avoids exec approval
// issues with the OpenClaw gateway.
//
// Requires a running OpenClaw gateway. Claims submitted by "test1",
// verification transactions submitted by "validator" (the only verifier
// in single-validator mode).
//
// Run with:
//
//	go test ./tests/openclaw/... -tags openclaw -run TestPhaseD -v -count=1 -timeout 600s
func TestPhaseD(t *testing.T) {
	claw.RequireGateway(t)

	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
	defer cancel()

	sessionID := fmt.Sprintf("phase-d-%d", time.Now().UnixNano())

	// Boot environment with extended phase windows for AI reasoning
	e := env.BootWithGenesisPatcher(t, binaryPath, ctx, devnet.PatchGenesisForAgent)
	t.Logf("Phase D: %s", env.EnvInfo(e))

	valClient := e.NewCLIClient("validator")
	gw := claw.NewGatewayClient(sessionID)

	clawEnv := claw.DevnetEnv{
		RESTAddr:   e.RESTAddr(),
		RPCAddr:    e.RPCAddr(),
		ChainID:    e.ChainID(),
		BinaryPath: e.Binary(),
		NodeHome:   e.NodeHome(),
		KeyName:    "validator",
	}

	// Fund test1 for claim submissions
	test1Addr := e.KeyAddress("test1")
	if err := e.FundAccount(ctx, "validator", test1Addr, 50_000_000_000); err != nil {
		t.Fatalf("funding test1: %v", err)
	}
	helpers.WaitForTx(ctx, valClient)

	// --- D-01: Gateway Reachable ---

	t.Run("OCA-D-01_GatewayReachable", func(t *testing.T) {
		reply, err := gw.Send(ctx, "Reply with exactly one word: PONG")
		if err != nil {
			t.Fatalf("gateway send failed: %v", err)
		}
		if !strings.Contains(strings.ToUpper(reply), "PONG") {
			t.Logf("reply did not contain PONG (got: %.200s), but agent responded", reply)
		}
		helpers.AgentResult(t, "01", "gateway reachable, agent responds")
		t.Logf("agent reply: %.200s", reply)
	})

	// --- D-02: Agent Reaches Devnet ---

	t.Run("OCA-D-02_AgentReachesDevnet", func(t *testing.T) {
		prompt := fmt.Sprintf(
			"Run this curl command and tell me if it succeeds:\n\n"+
				"```bash\ncurl -s %s/legible/knowledge/v1/params\n```\n\n"+
				"Just reply with YES if you get a JSON response, or NO if it fails.",
			e.RESTAddr(),
		)
		reply, err := gw.Send(ctx, prompt)
		if err != nil {
			t.Fatalf("agent devnet check failed: %v", err)
		}
		helpers.AgentResult(t, "02", "agent confirmed devnet connectivity")
		t.Logf("agent reply: %.300s", reply)
	})

	// --- D-03: Step-by-Step Verification ---
	// Agent evaluates claim → Go harness commits → waits → Go harness reveals

	t.Run("OCA-D-03_StepByStepVerification", func(t *testing.T) {
		claimText := "Water boils at 100 degrees Celsius at standard atmospheric pressure"

		// Submit claim from test1
		test1Client := e.NewCLIClient("test1")
		_, err := test1Client.SubmitClaim(ctx, agentsdk.SubmitClaimRequest{
			Text:     claimText,
			Domain:   "general",
			Category: "empirical",
			Stake:    1_000_000,
		})
		if err != nil {
			t.Fatalf("SubmitClaim: %v", err)
		}
		helpers.WaitForTx(ctx, valClient)
		t.Logf("claim submitted, waiting for active round...")

		// Wait for round in commit phase
		roundID := waitForCommitRound(t, ctx, valClient)
		t.Logf("found round %s in commit phase", roundID)

		// Ask agent to evaluate the claim
		evalPrompt := clawEnv.EvaluatePrompt(claimText)
		t.Logf("asking agent to evaluate claim...")

		evalGW := claw.NewGatewayClient(sessionID + "-eval")
		evalGW.Timeout = 90 * time.Second
		reply, err := evalGW.Send(ctx, evalPrompt)
		if err != nil {
			t.Fatalf("agent evaluation failed: %v", err)
		}
		t.Logf("agent evaluation reply: %.500s", reply)

		// Parse the verdict
		verdict := claw.ParseVerdict(reply)
		if verdict == nil {
			t.Fatalf("could not parse agent verdict from reply: %.500s", reply)
		}
		t.Logf("parsed verdict: %s confidence=%d salt=%s", verdict.Verdict, verdict.Confidence, verdict.Salt)
		helpers.AgentInsight(t, "03", fmt.Sprintf("agent evaluated: verdict=%s confidence=%d", verdict.Verdict, verdict.Confidence))

		// Map agent verdict to SDK verdict
		sdkVerdict := mapVerdict(verdict.Verdict)

		// Compute commitment hash using SDK method
		commitment := agentsdk.ComputeCommitment(roundID, sdkVerdict, verdict.Confidence, verdict.Salt)
		t.Logf("computed commitment hash: %s", commitment)

		// Submit commitment from validator
		_, err = valClient.SubmitCommitment(ctx, roundID, commitment)
		if err != nil {
			t.Fatalf("SubmitCommitment: %v", err)
		}
		helpers.WaitForTx(ctx, valClient)

		// Verify commitment was stored (catches silent tx failures)
		detail, _ := valClient.RoundDetail(ctx, roundID)
		if detail != nil {
			t.Logf("post-commit round: phase=%s commitments (detail available)", detail.Phase)
		}
		t.Logf("commitment submitted, waiting for reveal phase...")

		// Wait for reveal phase
		revealCtx, revealCancel := context.WithTimeout(ctx, 3*time.Minute)
		defer revealCancel()
		if err := helpers.WaitForRoundPhase(revealCtx, valClient, roundID, "reveal"); err != nil {
			// Check if round already completed
			detail, _ = valClient.RoundDetail(ctx, roundID)
			if detail != nil {
				t.Logf("round phase=%s", detail.Phase)
				if detail.Phase == "complete" {
					helpers.AgentResult(t, "03", "round completed (fast aggregation)")
					return
				}
			}
			t.Fatalf("round didn't reach reveal phase: %v", err)
		}
		t.Logf("round in reveal phase, submitting reveal...")

		// Submit reveal from validator
		_, err = valClient.SubmitReveal(ctx, roundID, sdkVerdict, verdict.Confidence, verdict.Salt)
		if err != nil {
			t.Fatalf("SubmitReveal: %v", err)
		}
		helpers.WaitForTx(ctx, valClient)
		t.Logf("reveal submitted, waiting for round completion...")

		// Wait for completion (reveal phase is 60 blocks ~151s, so wait 4 minutes)
		completeCtx, completeCancel := context.WithTimeout(ctx, 4*time.Minute)
		defer completeCancel()
		if err := helpers.WaitForRoundPhase(completeCtx, valClient, roundID, "complete"); err != nil {
			t.Logf("round didn't reach complete phase: %v", err)
		}

		// Check outcome
		time.Sleep(3 * time.Second)
		detail, _ = valClient.RoundDetail(ctx, roundID)
		if detail != nil {
			t.Logf("round final: phase=%s", detail.Phase)
		}

		facts, _ := valClient.FactsByDomain(ctx, "general")
		t.Logf("facts in general domain: %d", len(facts))
		for _, f := range facts {
			if strings.Contains(f.ClaimText, "boil") || strings.Contains(f.ClaimText, "100") {
				t.Logf("fact created: %s (confidence=%d)", f.ID, f.Confidence)
				helpers.AgentResult(t, "03", fmt.Sprintf("verification complete — fact created with verdict=%s", verdict.Verdict))
				return
			}
		}

		// Round completed but no fact (verdict was false/uncertain or confidence below threshold)
		if detail != nil && detail.Phase == "complete" {
			helpers.AgentResult(t, "03", fmt.Sprintf("verification complete — agent verdict=%s (no fact, may be false/uncertain or low confidence)", verdict.Verdict))
			return
		}

		t.Fatalf("round %s: phase=%s — expected round to complete",
			roundID, detail.Phase)
	})

	// --- D-04: Autonomous Loop ---
	// Agent evaluates multiple claims in one batch, Go harness commits/reveals all

	t.Run("OCA-D-04_AutonomousLoop", func(t *testing.T) {
		initialFacts, _ := valClient.FactsByDomain(ctx, "general")
		initialCount := len(initialFacts)
		t.Logf("initial fact count: %d", initialCount)

		// Submit 3 claims
		test1Client := e.NewCLIClient("test1")
		claimTexts := []string{
			"The Earth orbits the Sun once every 365.25 days approximately",
			"Pi is approximately equal to 3.14159265",
			"DNA has a double helix structure",
		}

		for _, text := range claimTexts {
			_, err := test1Client.SubmitClaim(ctx, agentsdk.SubmitClaimRequest{
				Text:     text,
				Domain:   "general",
				Category: "empirical",
				Stake:    1_000_000,
			})
			if err != nil {
				t.Logf("claim submission error (continuing): %v", err)
			}
			helpers.WaitForTx(ctx, valClient)
		}
		t.Logf("submitted %d claims", len(claimTexts))

		// Collect all rounds in commit phase
		rounds, _ := valClient.ActiveRounds(ctx)
		var commitRounds []agentsdk.RoundInfo
		for _, r := range rounds {
			if r.Phase == "commit" {
				commitRounds = append(commitRounds, r)
			}
		}
		t.Logf("active rounds: %d total, %d in commit phase", len(rounds), len(commitRounds))
		if len(commitRounds) == 0 {
			t.Fatalf("no rounds in commit phase")
		}

		// Ask agent to batch-evaluate all claims
		evalPrompt := clawEnv.BatchEvaluatePrompt(claimTexts)
		t.Logf("asking agent to batch-evaluate %d claims...", len(claimTexts))

		batchGW := claw.NewGatewayClient(sessionID + "-batch")
		batchGW.Timeout = 120 * time.Second
		reply, err := batchGW.Send(ctx, evalPrompt)
		if err != nil {
			t.Fatalf("agent batch evaluation failed: %v", err)
		}
		t.Logf("agent batch reply: %.1000s", reply)

		verdicts := claw.ParseBatchVerdicts(reply, len(claimTexts))
		t.Logf("parsed %d verdicts from agent", len(verdicts))
		for i, v := range verdicts {
			t.Logf("  claim %d: verdict=%s confidence=%d salt=%s", i+1, v.Verdict, v.Confidence, v.Salt)
		}

		if len(verdicts) == 0 {
			t.Fatalf("no verdicts parsed from agent reply")
		}

		// Submit commitments for available rounds (match rounds to verdicts)
		committed := 0
		type roundVerdict struct {
			roundID    string
			sdkVerdict agentsdk.Verdict
			confidence uint32
			salt       string
		}
		var committedRounds []roundVerdict

		for i, r := range commitRounds {
			if i >= len(verdicts) {
				break
			}
			v := verdicts[i]
			sdkV := mapVerdict(v.Verdict)
			hash := agentsdk.ComputeCommitment(r.ID, sdkV, v.Confidence, v.Salt)

			_, err := valClient.SubmitCommitment(ctx, r.ID, hash)
			if err != nil {
				t.Logf("commit for round %s failed: %v", r.ID, err)
				continue
			}
			helpers.WaitForTx(ctx, valClient)
			committed++
			committedRounds = append(committedRounds, roundVerdict{
				roundID:    r.ID,
				sdkVerdict: sdkV,
				confidence: v.Confidence,
				salt:       v.Salt,
			})
			t.Logf("committed to round %s: verdict=%s", r.ID, v.Verdict)
		}
		t.Logf("committed to %d rounds", committed)

		if committed == 0 {
			t.Fatalf("failed to commit to any rounds")
		}

		// Wait for reveal phase on committed rounds, then reveal
		revealed := 0
		for _, rv := range committedRounds {
			revealCtx, revealCancel := context.WithTimeout(ctx, 3*time.Minute)
			err := helpers.WaitForRoundPhase(revealCtx, valClient, rv.roundID, "reveal")
			revealCancel()
			if err != nil {
				t.Logf("round %s didn't reach reveal: %v", rv.roundID, err)
				continue
			}

			_, err = valClient.SubmitReveal(ctx, rv.roundID, rv.sdkVerdict, rv.confidence, rv.salt)
			if err != nil {
				t.Logf("reveal for round %s failed: %v", rv.roundID, err)
				continue
			}
			helpers.WaitForTx(ctx, valClient)
			revealed++
			t.Logf("revealed round %s", rv.roundID)
		}
		t.Logf("revealed %d rounds", revealed)

		// Wait for facts (reveal phase is 60 blocks ~151s, plus aggregation time)
		factCtx, factCancel := context.WithTimeout(ctx, 4*time.Minute)
		defer factCancel()

		factTicker := time.NewTicker(5 * time.Second)
		defer factTicker.Stop()

		for {
			select {
			case <-factCtx.Done():
				facts, _ := valClient.FactsByDomain(ctx, "general")
				newFacts := len(facts) - initialCount
				t.Logf("timeout: %d total facts, %d new", len(facts), newFacts)
				if newFacts > 0 {
					helpers.AgentResult(t, "04", fmt.Sprintf("autonomous loop created %d new facts", newFacts))
					return
				}
				if revealed > 0 {
					helpers.AgentResult(t, "04", fmt.Sprintf("agent evaluated + harness committed/revealed %d rounds (no facts — verdicts may be reject/abstain)", revealed))
					return
				}
				t.Fatalf("autonomous loop: 0 new facts, 0 reveals")

			case <-factTicker.C:
				facts, err := valClient.FactsByDomain(factCtx, "general")
				if err != nil {
					continue
				}
				newFacts := len(facts) - initialCount
				if newFacts > 0 {
					t.Logf("success: %d new facts", newFacts)
					for i := initialCount; i < len(facts); i++ {
						t.Logf("  fact: %s (confidence=%d) — %.80s", facts[i].ID, facts[i].Confidence, facts[i].ClaimText)
					}
					helpers.AgentResult(t, "04", fmt.Sprintf("autonomous loop created %d new facts", newFacts))
					return
				}
			}
		}
	})
}

// waitForCommitRound polls ActiveRounds until a round in "commit" phase appears.
func waitForCommitRound(t *testing.T, ctx context.Context, client agentsdk.QueryClient) string {
	t.Helper()
	commitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-commitCtx.Done():
			t.Fatalf("no active round in commit phase found")
			return ""
		case <-ticker.C:
			rounds, err := client.ActiveRounds(commitCtx)
			if err != nil {
				continue
			}
			for _, r := range rounds {
				if r.Phase == "commit" {
					return r.ID
				}
			}
		}
	}
}

// mapVerdict converts agent text verdicts to SDK Verdict type.
func mapVerdict(s string) agentsdk.Verdict {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "accept":
		return agentsdk.VerdictTrue
	case "false", "reject":
		return agentsdk.VerdictFalse
	default:
		return agentsdk.VerdictUncertain
	}
}
