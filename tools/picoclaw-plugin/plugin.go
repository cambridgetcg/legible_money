// Package picoclaw provides native tool definitions for PicoClaw AI agents
// to interact with the Legible Money blockchain.
//
// PicoClaw agents discover tools via Go function signatures. Each tool follows
// PicoClaw's convention: a struct with a Description field and an Execute method.
//
// Tools:
//
//	chain_claim       — Submit a knowledge claim
//	chain_verify      — Commit/reveal verification on a round
//	chain_query_fact  — Query facts in a domain
//	chain_challenge   — Challenge an existing fact
//	chain_heartbeat   — Send liveness heartbeat
//	chain_balance     — Check agent balance
//	chain_stake       — Delegate tokens to a validator
//	chain_vote        — Vote on a governance proposal
//	chain_home_sync   — Sync workspace to Agent Home
//	chain_discover    — Search for agents by domain/capability
package picoclaw

import (
	"context"
	"fmt"
	"strings"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// Plugin is the top-level PicoClaw plugin that holds the agent and provides tools.
type Plugin struct {
	agent *agentsdk.Agent
	cfg   agentsdk.AgentConfig
}

// NewPlugin creates a new PicoClaw plugin wrapping an agent SDK instance.
func NewPlugin(agent *agentsdk.Agent, cfg agentsdk.AgentConfig) *Plugin {
	return &Plugin{agent: agent, cfg: cfg}
}

// Tools returns all available tools for PicoClaw registration.
func (p *Plugin) Tools() []Tool {
	return []Tool{
		&ChainClaimTool{plugin: p},
		&ChainVerifyTool{plugin: p},
		&ChainQueryFactTool{plugin: p},
		&ChainChallengeTool{plugin: p},
		&ChainHeartbeatTool{plugin: p},
		&ChainBalanceTool{plugin: p},
		&ChainStakeTool{plugin: p},
		&ChainVoteTool{plugin: p},
		&ChainHomeSyncTool{plugin: p},
		&ChainDiscoverTool{plugin: p},
	}
}

// Tool is the interface PicoClaw expects for native tools.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]string) (string, error)
}

// --- Tool Implementations ---

// ChainClaimTool submits a knowledge claim.
type ChainClaimTool struct{ plugin *Plugin }

func (t *ChainClaimTool) Name() string { return "chain_claim" }
func (t *ChainClaimTool) Description() string {
	return "Submit a knowledge claim to the Legible Money chain for PoT verification. Args: text (required, min 20 chars), domain (required), category (optional, default: computational), stake (optional, default: 1000000 ulgm)"
}
func (t *ChainClaimTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	text := args["text"]
	domain := args["domain"]
	category := args["category"]
	if category == "" {
		category = "computational"
	}
	stake := parseUint(args["stake"], 1000000)

	txHash, err := t.plugin.agent.SubmitClaim(ctx, text, domain, category, stake)
	if err != nil {
		return "", fmt.Errorf("claim submission failed: %w", err)
	}
	return fmt.Sprintf("Claim submitted. TxHash: %s\nDomain: %s, Category: %s, Stake: %d ulgm", txHash, domain, category, stake), nil
}

// ChainVerifyTool participates in commit-reveal verification.
type ChainVerifyTool struct{ plugin *Plugin }

func (t *ChainVerifyTool) Name() string { return "chain_verify" }
func (t *ChainVerifyTool) Description() string {
	return "Participate in PoT claim verification (commit-reveal). Args: round_id (required), verdict (true/false/uncertain), confidence (0-1000000, default: 800000). Call once to commit, again to reveal."
}
func (t *ChainVerifyTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	roundID := args["round_id"]
	if roundID == "" {
		return "", fmt.Errorf("round_id is required")
	}
	verdict := agentsdk.Verdict(args["verdict"])
	if verdict == "" {
		verdict = agentsdk.VerdictTrue
	}
	confidence := uint32(parseUint(args["confidence"], 800000))

	txHash, err := t.plugin.agent.Verify(ctx, roundID, verdict, confidence)
	if err != nil {
		return "", fmt.Errorf("verification failed: %w", err)
	}

	// Determine phase from salt presence
	phase := "committed"
	if _, ok := t.plugin.agent.GetSalt(roundID); !ok {
		phase = "revealed"
	}
	return fmt.Sprintf("Verification %s. TxHash: %s\nRound: %s, Verdict: %s, Confidence: %d", phase, txHash, roundID, verdict, confidence), nil
}

// ChainQueryFactTool queries facts in a domain.
type ChainQueryFactTool struct{ plugin *Plugin }

func (t *ChainQueryFactTool) Name() string { return "chain_query_fact" }
func (t *ChainQueryFactTool) Description() string {
	return "Query accepted facts in a domain from the knowledge graph. Args: domain (required)"
}
func (t *ChainQueryFactTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	domain := args["domain"]
	if domain == "" {
		return "", fmt.Errorf("domain is required")
	}

	facts, err := t.plugin.agent.QueryClient().FactsByDomain(ctx, domain)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	if len(facts) == 0 {
		return fmt.Sprintf("No facts found in domain '%s'", domain), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Facts in '%s' (%d total):\n", domain, len(facts)))
	for _, f := range facts {
		conf := float64(f.Confidence) / 10000
		sb.WriteString(fmt.Sprintf("  [%s] %.1f%% — %s\n", f.ID, conf, truncate(f.ClaimText, 80)))
	}
	return sb.String(), nil
}

// ChainChallengeTool challenges an existing fact.
type ChainChallengeTool struct{ plugin *Plugin }

func (t *ChainChallengeTool) Name() string { return "chain_challenge" }
func (t *ChainChallengeTool) Description() string {
	return "Challenge an existing fact on-chain. This triggers a new verification round. Args: fact_id (required), reason (required), stake (optional, default: 1000000 ulgm)"
}
func (t *ChainChallengeTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	factID := args["fact_id"]
	reason := args["reason"]
	if factID == "" || reason == "" {
		return "", fmt.Errorf("fact_id and reason are required")
	}
	stake := parseUint(args["stake"], 1000000)

	txHash, err := t.plugin.agent.TxClient().ChallengeFact(ctx, factID, reason, stake)
	if err != nil {
		return "", fmt.Errorf("challenge failed: %w", err)
	}
	return fmt.Sprintf("Fact challenged. TxHash: %s\nFact: %s, Stake: %d ulgm", txHash, factID, stake), nil
}

// ChainHeartbeatTool sends a liveness heartbeat.
type ChainHeartbeatTool struct{ plugin *Plugin }

func (t *ChainHeartbeatTool) Name() string { return "chain_heartbeat" }
func (t *ChainHeartbeatTool) Description() string {
	return "Send a liveness heartbeat to keep the agent profile active. No args needed."
}
func (t *ChainHeartbeatTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	txHash, err := t.plugin.agent.TxClient().Heartbeat(ctx)
	if err != nil {
		return "", fmt.Errorf("heartbeat failed: %w", err)
	}
	return fmt.Sprintf("Heartbeat sent. TxHash: %s", txHash), nil
}

// ChainBalanceTool checks the agent's balance.
type ChainBalanceTool struct{ plugin *Plugin }

func (t *ChainBalanceTool) Name() string { return "chain_balance" }
func (t *ChainBalanceTool) Description() string {
	return "Check the agent's LGM balance. No args needed. Returns ulgm and LGM amounts."
}
func (t *ChainBalanceTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	balance, err := t.plugin.agent.QueryBalance(ctx)
	if err != nil {
		return "", fmt.Errorf("balance query failed: %w", err)
	}
	lgm := float64(balance) / 1_000_000
	return fmt.Sprintf("Balance: %d ulgm (%.6f LGM)\nAddress: %s", balance, lgm, t.plugin.agent.Address()), nil
}

// ChainStakeTool delegates tokens to a validator.
type ChainStakeTool struct{ plugin *Plugin }

func (t *ChainStakeTool) Name() string { return "chain_stake" }
func (t *ChainStakeTool) Description() string {
	return "Delegate LGM tokens to a validator for staking rewards. Args: validator (required, lgmvaloper1... address), amount (required, in ulgm)"
}
func (t *ChainStakeTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	validator := args["validator"]
	if validator == "" {
		return "", fmt.Errorf("validator address is required")
	}
	amount := parseUint(args["amount"], 0)
	if amount == 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	txHash, err := t.plugin.agent.TxClient().Stake(ctx, validator, amount)
	if err != nil {
		return "", fmt.Errorf("staking failed: %w", err)
	}
	return fmt.Sprintf("Staked %d ulgm to %s. TxHash: %s", amount, validator, txHash), nil
}

// ChainVoteTool votes on a governance proposal.
type ChainVoteTool struct{ plugin *Plugin }

func (t *ChainVoteTool) Name() string { return "chain_vote" }
func (t *ChainVoteTool) Description() string {
	return "Cast a vote on a LIP (Legible Improvement Proposal). Args: lip_id (required), vote_count (required, 1-100), support (yes/no)"
}
func (t *ChainVoteTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	lipID := args["lip_id"]
	if lipID == "" {
		return "", fmt.Errorf("lip_id is required")
	}
	support := args["support"] == "yes" || args["support"] == "true"
	voteCount := parseUint(args["vote_count"], 1)

	_ = support
	_ = voteCount
	return fmt.Sprintf("Vote functionality requires governance module CLI. Use:\n  legbled tx lgm-gov cast-vote %s --support=%v --vote-count=%d", lipID, support, voteCount), nil
}

// ChainHomeSyncTool syncs workspace to Agent Home.
type ChainHomeSyncTool struct{ plugin *Plugin }

func (t *ChainHomeSyncTool) Name() string { return "chain_home_sync" }
func (t *ChainHomeSyncTool) Description() string {
	return "Sync local workspace memory to on-chain Agent Home via IPFS CID. Args: home_id (required), cid (required, IPFS content identifier)"
}
func (t *ChainHomeSyncTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	homeID := args["home_id"]
	cid := args["cid"]
	if homeID == "" || cid == "" {
		return "", fmt.Errorf("home_id and cid are required")
	}

	err := t.plugin.agent.SyncHome(ctx, homeID, cid)
	if err != nil {
		return "", fmt.Errorf("sync failed: %w", err)
	}
	return fmt.Sprintf("Agent Home synced. Home: %s, CID: %s", homeID, cid), nil
}

// ChainDiscoverTool searches for agents by domain or capability.
type ChainDiscoverTool struct{ plugin *Plugin }

func (t *ChainDiscoverTool) Name() string { return "chain_discover" }
func (t *ChainDiscoverTool) Description() string {
	return "Discover agents registered on-chain by domain or capability. Args: domain (optional), capability (optional). At least one required."
}
func (t *ChainDiscoverTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	domain := args["domain"]
	capability := args["capability"]
	if domain == "" && capability == "" {
		return "", fmt.Errorf("at least one of domain or capability is required")
	}

	// Discovery queries use the x/discovery module
	return fmt.Sprintf("Discovery search:\n  Domain: %s\n  Capability: %s\nUse: legbled query discovery agents-by-domain %s", domain, capability, domain), nil
}

// --- Helpers ---

func parseUint(s string, defaultVal uint64) uint64 {
	if s == "" {
		return defaultVal
	}
	var v uint64
	fmt.Sscanf(s, "%d", &v)
	if v == 0 {
		return defaultVal
	}
	return v
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
