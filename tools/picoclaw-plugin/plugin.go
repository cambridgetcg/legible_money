// Package picoclaw provides native tool definitions for PicoClaw AI agents
// to interact with the Legible Money blockchain.
//
// PicoClaw agents discover tools via Go function signatures. Each tool follows
// PicoClaw's convention: a struct with a Description field and an Execute method.
//
// Tools:
//
//	chain_claim            — Submit a knowledge claim
//	chain_verify           — Commit/reveal verification on a round
//	chain_query_fact       — Query facts in a domain
//	chain_challenge        — Challenge an existing fact
//	chain_heartbeat        — Send liveness heartbeat
//	chain_balance          — Check agent balance
//	chain_stake            — Delegate tokens to a validator
//	chain_vote             — Vote on a governance proposal
//	chain_discover         — Search for agents by domain/capability
//	chain_deploy_contract  — Deploy a BVM contract
//	chain_call_contract    — Call a BVM contract
//	chain_schedule         — Schedule future BVM execution
//	chain_home_create      — Create an Agent Home
//	chain_home_sync        — Sync workspace to Agent Home
//	chain_home_guardian    — Configure guardian settings
//	chain_home_session     — Start/end Home sessions
//	chain_home_spending    — Set spending limits
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
		// Knowledge / PoT
		&ChainClaimTool{plugin: p},
		&ChainVerifyTool{plugin: p},
		&ChainQueryFactTool{plugin: p},
		&ChainChallengeTool{plugin: p},
		// Discovery
		&ChainHeartbeatTool{plugin: p},
		&ChainBalanceTool{plugin: p},
		&ChainDiscoverTool{plugin: p},
		// Staking & Governance
		&ChainStakeTool{plugin: p},
		&ChainVoteTool{plugin: p},
		// BVM
		&ChainDeployContractTool{plugin: p},
		&ChainCallContractTool{plugin: p},
		&ChainReadContractTool{plugin: p},
		&ChainScheduleTool{plugin: p},
		// Agent Home
		&ChainHomeCreateTool{plugin: p},
		&ChainHomeSyncTool{plugin: p},
		&ChainHomeGuardianTool{plugin: p},
		&ChainHomeSessionTool{plugin: p},
		&ChainHomeSpendingTool{plugin: p},
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
	return "Submit a knowledge claim to the Legible Money chain for PoT verification. Args: text (required, min 20 chars), domain (required), category (optional, default: empirical), stake (optional, default: 1000000 ulgm)"
}
func (t *ChainClaimTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	text := args["text"]
	domain := args["domain"]
	category := args["category"]
	if category == "" {
		category = "empirical"
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

// --- BVM Tools ---

// ChainDeployContractTool deploys a BVM contract.
type ChainDeployContractTool struct{ plugin *Plugin }

func (t *ChainDeployContractTool) Name() string { return "chain_deploy_contract" }
func (t *ChainDeployContractTool) Description() string {
	return "Deploy a BVM (Belief Virtual Machine) contract on-chain. Args: bytecode (required, hex-encoded bytecode)"
}
func (t *ChainDeployContractTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	bytecode := args["bytecode"]
	if bytecode == "" {
		return "", fmt.Errorf("bytecode is required")
	}
	txHash, err := t.plugin.agent.DeployContract(ctx, bytecode)
	if err != nil {
		return "", fmt.Errorf("deploy failed: %w", err)
	}
	return fmt.Sprintf("Contract deployed. TxHash: %s", txHash), nil
}

// ChainCallContractTool calls a deployed BVM contract.
type ChainCallContractTool struct{ plugin *Plugin }

func (t *ChainCallContractTool) Name() string { return "chain_call_contract" }
func (t *ChainCallContractTool) Description() string {
	return "Call a deployed BVM contract. Args: contract (required, lgm1... address), input (required, hex-encoded calldata), gas_limit (optional, default: 500000)"
}
func (t *ChainCallContractTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	contract := args["contract"]
	input := args["input"]
	if contract == "" || input == "" {
		return "", fmt.Errorf("contract and input are required")
	}
	gasLimit := parseUint(args["gas_limit"], 500000)

	txHash, err := t.plugin.agent.CallContract(ctx, contract, input, gasLimit)
	if err != nil {
		return "", fmt.Errorf("call failed: %w", err)
	}
	return fmt.Sprintf("Contract called. TxHash: %s\nContract: %s, Gas limit: %d", txHash, contract, gasLimit), nil
}

// ChainScheduleTool schedules a future BVM contract execution.
type ChainScheduleTool struct{ plugin *Plugin }

func (t *ChainScheduleTool) Name() string { return "chain_schedule" }
func (t *ChainScheduleTool) Description() string {
	return "Schedule a BVM contract call at a future block. Args: contract (required), method (required), payload (optional), at_block (required, block height), max_gas (optional, default: 500000)"
}
func (t *ChainScheduleTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	contract := args["contract"]
	method := args["method"]
	if contract == "" || method == "" {
		return "", fmt.Errorf("contract and method are required")
	}
	payload := args["payload"]
	atBlock := parseUint(args["at_block"], 0)
	if atBlock == 0 {
		return "", fmt.Errorf("at_block is required and must be > 0")
	}
	maxGas := parseUint(args["max_gas"], 500000)

	txHash, err := t.plugin.agent.ScheduleExecution(ctx, contract, method, payload, atBlock, maxGas)
	if err != nil {
		return "", fmt.Errorf("schedule failed: %w", err)
	}
	return fmt.Sprintf("Execution scheduled. TxHash: %s\nContract: %s, At block: %d", txHash, contract, atBlock), nil
}

// ChainReadContractTool reads state from a deployed BVM contract.
type ChainReadContractTool struct{ plugin *Plugin }

func (t *ChainReadContractTool) Name() string { return "chain_read_contract" }
func (t *ChainReadContractTool) Description() string {
	return "Read storage state from a deployed BVM contract (no tx needed). Args: contract (required, lgm1... address), key (required, storage slot key). Returns the stored value."
}
func (t *ChainReadContractTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	contract := args["contract"]
	key := args["key"]
	if contract == "" || key == "" {
		return "", fmt.Errorf("contract and key are required")
	}

	value, found, err := t.plugin.agent.QueryClient().ContractState(ctx, contract, key)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	if !found {
		return fmt.Sprintf("Key %q not found in contract %s", key, contract), nil
	}
	return fmt.Sprintf("Contract: %s\nKey: %s\nValue: %s", contract, key, value), nil
}

// --- Home Lifecycle Tools ---

// ChainHomeCreateTool creates an Agent Home.
type ChainHomeCreateTool struct{ plugin *Plugin }

func (t *ChainHomeCreateTool) Name() string { return "chain_home_create" }
func (t *ChainHomeCreateTool) Description() string {
	return "Create a new Agent Home on-chain. Args: name (required, display name for the home)"
}
func (t *ChainHomeCreateTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	name := args["name"]
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	txHash, err := t.plugin.agent.TxClient().CreateHome(ctx, name)
	if err != nil {
		return "", fmt.Errorf("create home failed: %w", err)
	}
	return fmt.Sprintf("Agent Home created. TxHash: %s\nName: %s", txHash, name), nil
}

// ChainHomeGuardianTool configures the Agent Home's guardian.
type ChainHomeGuardianTool struct{ plugin *Plugin }

func (t *ChainHomeGuardianTool) Name() string { return "chain_home_guardian" }
func (t *ChainHomeGuardianTool) Description() string {
	return "Configure Agent Home guardian settings. Args: home_id (required), strategy (aggressive/moderate/conservative/diplomatic), auto_defend (true/false), deadman_threshold (blocks, min 34272)"
}
func (t *ChainHomeGuardianTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	homeID := args["home_id"]
	if homeID == "" {
		return "", fmt.Errorf("home_id is required")
	}
	strategy := args["strategy"]
	if strategy == "" {
		strategy = "moderate"
	}
	autoDefend := args["auto_defend"] == "true"
	deadmanThreshold := parseUint(args["deadman_threshold"], 34272)

	err := t.plugin.agent.ConfigureGuardian(ctx, homeID, strategy, autoDefend, deadmanThreshold)
	if err != nil {
		return "", fmt.Errorf("configure guardian failed: %w", err)
	}
	return fmt.Sprintf("Guardian configured. Home: %s, Strategy: %s, Auto-defend: %v, Deadman: %d blocks",
		homeID, strategy, autoDefend, deadmanThreshold), nil
}

// ChainHomeSessionTool manages Agent Home sessions.
type ChainHomeSessionTool struct{ plugin *Plugin }

func (t *ChainHomeSessionTool) Name() string { return "chain_home_session" }
func (t *ChainHomeSessionTool) Description() string {
	return "Start or end an Agent Home session. Args: home_id (required), action (start/end), key_hash (required for start), session_id (required for end), permissions (optional, comma-separated)"
}
func (t *ChainHomeSessionTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	homeID := args["home_id"]
	action := args["action"]
	if homeID == "" || action == "" {
		return "", fmt.Errorf("home_id and action are required")
	}

	switch action {
	case "start":
		keyHash := args["key_hash"]
		if keyHash == "" {
			return "", fmt.Errorf("key_hash is required for start")
		}
		var perms []string
		if p := args["permissions"]; p != "" {
			perms = strings.Split(p, ",")
		}
		txHash, err := t.plugin.agent.TxClient().StartSession(ctx, homeID, keyHash, perms)
		if err != nil {
			return "", fmt.Errorf("start session failed: %w", err)
		}
		return fmt.Sprintf("Session started. TxHash: %s\nHome: %s", txHash, homeID), nil

	case "end":
		sessionID := args["session_id"]
		if sessionID == "" {
			return "", fmt.Errorf("session_id is required for end")
		}
		txHash, err := t.plugin.agent.TxClient().EndSession(ctx, homeID, sessionID)
		if err != nil {
			return "", fmt.Errorf("end session failed: %w", err)
		}
		return fmt.Sprintf("Session ended. TxHash: %s\nHome: %s, Session: %s", txHash, homeID, sessionID), nil

	default:
		return "", fmt.Errorf("action must be 'start' or 'end'")
	}
}

// ChainHomeSpendingTool sets spending limits on an Agent Home.
type ChainHomeSpendingTool struct{ plugin *Plugin }

func (t *ChainHomeSpendingTool) Name() string { return "chain_home_spending" }
func (t *ChainHomeSpendingTool) Description() string {
	return "Set a spending limit for an Agent Home key type. Args: home_id (required), key_type (required), max_amount (required, ulgm), period_blocks (required)"
}
func (t *ChainHomeSpendingTool) Execute(ctx context.Context, args map[string]string) (string, error) {
	homeID := args["home_id"]
	keyType := args["key_type"]
	if homeID == "" || keyType == "" {
		return "", fmt.Errorf("home_id and key_type are required")
	}
	maxAmount := parseUint(args["max_amount"], 0)
	periodBlocks := parseUint(args["period_blocks"], 0)
	if maxAmount == 0 || periodBlocks == 0 {
		return "", fmt.Errorf("max_amount and period_blocks must be positive")
	}

	err := t.plugin.agent.SetSpendingLimit(ctx, homeID, keyType, maxAmount, periodBlocks)
	if err != nil {
		return "", fmt.Errorf("set spending limit failed: %w", err)
	}
	lgm := float64(maxAmount) / 1_000_000
	return fmt.Sprintf("Spending limit set. Home: %s, Key: %s, Max: %.6f LGM per %d blocks",
		homeID, keyType, lgm, periodBlocks), nil
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
