package agentsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Ensure CLIClient satisfies both interfaces.
var (
	_ QueryClient = (*CLIClient)(nil)
	_ TxClient    = (*CLIClient)(nil)
)

func userHomeDir() (string, error) {
	return os.UserHomeDir()
}

// runQuery executes a legbled query command and returns JSON output.
func (c *CLIClient) runQuery(ctx context.Context, args ...string) ([]byte, error) {
	fullArgs := append(args, "--node", c.Node, "--output", "json")
	cmd := exec.CommandContext(ctx, c.Binary, fullArgs...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("query failed: %s: %w", string(out), err)
	}
	return out, nil
}

// runTx executes a legbled tx command and returns the tx hash.
func (c *CLIClient) runTx(ctx context.Context, args ...string) (string, error) {
	fullArgs := append(args,
		"--from", c.KeyName,
		"--keyring-backend", c.Keyring,
		"--home", c.Home,
		"--node", c.Node,
		"--chain-id", c.ChainID,
		"--gas-prices", fmt.Sprintf("%dulgm", c.GasPrice),
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"-y",
		"--output", "json",
	)
	cmd := exec.CommandContext(ctx, c.Binary, fullArgs...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tx failed: %s: %w", string(out), err)
	}

	// Extract tx hash from JSON response
	var resp struct {
		TxHash string `json:"txhash"`
	}
	if json.Unmarshal(out, &resp) == nil && resp.TxHash != "" {
		return resp.TxHash, nil
	}
	return strings.TrimSpace(string(out)), nil
}

// --- QueryClient Implementation ---

func (c *CLIClient) ChainStatus(ctx context.Context) (*ChainStatus, error) {
	cmd := exec.CommandContext(ctx, c.Binary, "status", "--node", c.Node)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("status query failed: %w", err)
	}

	var raw struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
			LatestBlockTime   string `json:"latest_block_time"`
			CatchingUp        bool   `json:"catching_up"`
		} `json:"sync_info"`
		NodeInfo struct {
			Network string `json:"network"`
		} `json:"node_info"`
	}
	// Try both formats (CometBFT uses different casing)
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parsing status: %w", err)
	}

	height, _ := strconv.ParseInt(raw.SyncInfo.LatestBlockHeight, 10, 64)
	return &ChainStatus{
		ChainID:      raw.NodeInfo.Network,
		LatestHeight: height,
		LatestTime:   raw.SyncInfo.LatestBlockTime,
		CatchingUp:   raw.SyncInfo.CatchingUp,
	}, nil
}

func (c *CLIClient) Balance(ctx context.Context, address string) (uint64, error) {
	out, err := c.runQuery(ctx, "query", "bank", "balances", address)
	if err != nil {
		return 0, err
	}

	var resp struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, fmt.Errorf("parsing balance: %w", err)
	}
	for _, b := range resp.Balances {
		if b.Denom == Denom {
			amt, _ := strconv.ParseUint(b.Amount, 10, 64)
			return amt, nil
		}
	}
	return 0, nil
}

func (c *CLIClient) AgentProfile(ctx context.Context, address string) (*AgentProfileInfo, error) {
	out, err := c.runQuery(ctx, "query", "discovery", "agent", address)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Profile AgentProfileInfo `json:"profile"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing profile: %w", err)
	}
	return &resp.Profile, nil
}

func (c *CLIClient) ActiveRounds(ctx context.Context) ([]RoundInfo, error) {
	out, err := c.runQuery(ctx, "query", "knowledge", "active-rounds")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Rounds []RoundInfo `json:"rounds"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing rounds: %w", err)
	}
	return resp.Rounds, nil
}

func (c *CLIClient) RoundDetail(ctx context.Context, roundID string) (*RoundInfo, error) {
	out, err := c.runQuery(ctx, "query", "knowledge", "round", roundID)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Round RoundInfo `json:"round"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing round: %w", err)
	}
	return &resp.Round, nil
}

func (c *CLIClient) FactsByDomain(ctx context.Context, domain string) ([]FactInfo, error) {
	out, err := c.runQuery(ctx, "query", "knowledge", "facts-by-domain", domain)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Facts []FactInfo `json:"facts"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing facts: %w", err)
	}
	return resp.Facts, nil
}

func (c *CLIClient) ClaimsBySubmitter(ctx context.Context, address string) ([]ClaimInfo, error) {
	out, err := c.runQuery(ctx, "query", "knowledge", "claims-by-submitter", address)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Claims []ClaimInfo `json:"claims"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing claims: %w", err)
	}
	return resp.Claims, nil
}

// --- TxClient Implementation ---

func (c *CLIClient) RegisterAccount(ctx context.Context, pubKeyHex string) (string, error) {
	return c.runTx(ctx, "tx", "legible-auth", "register-account")
}

func (c *CLIClient) RegisterAgent(ctx context.Context, req RegisterAgentRequest) (string, error) {
	args := []string{
		"tx", "discovery", "register-agent", req.DisplayName,
		"--stake", fmt.Sprintf("%d%s", req.Stake, Denom),
	}
	if len(req.Domains) > 0 {
		args = append(args, "--domain", strings.Join(req.Domains, ","))
	}
	if req.Description != "" {
		args = append(args, "--description", req.Description)
	}
	return c.runTx(ctx, args...)
}

func (c *CLIClient) Heartbeat(ctx context.Context) (string, error) {
	return c.runTx(ctx, "tx", "discovery", "heartbeat")
}

func (c *CLIClient) SubmitClaim(ctx context.Context, req SubmitClaimRequest) (string, error) {
	if len(req.Text) < MinClaimLen {
		return "", fmt.Errorf("claim text must be at least %d characters", MinClaimLen)
	}
	args := []string{
		"tx", "knowledge", "submit-claim", req.Text,
		"--domain", req.Domain,
		"--category", req.Category,
		"--stake", fmt.Sprintf("%d%s", req.Stake, Denom),
	}
	return c.runTx(ctx, args...)
}

func (c *CLIClient) SubmitCommitment(ctx context.Context, roundID, commitment string) (string, error) {
	return c.runTx(ctx, "tx", "knowledge", "submit-commitment", roundID,
		"--commitment", commitment)
}

func (c *CLIClient) SubmitReveal(ctx context.Context, roundID string, verdict Verdict, confidence uint32, salt string) (string, error) {
	return c.runTx(ctx, "tx", "knowledge", "submit-reveal", roundID,
		"--verdict", string(verdict),
		"--confidence", fmt.Sprintf("%d", confidence),
		"--salt", salt)
}

func (c *CLIClient) ChallengeFact(ctx context.Context, factID, reason string, stake uint64) (string, error) {
	return c.runTx(ctx, "tx", "knowledge", "challenge-fact", factID,
		"--reason", reason,
		"--stake", fmt.Sprintf("%d%s", stake, Denom))
}

func (c *CLIClient) CreateHome(ctx context.Context, name string) (string, error) {
	return c.runTx(ctx, "tx", "home", "create-home", name)
}

func (c *CLIClient) UpdateMemoryCID(ctx context.Context, homeID, cid string) (string, error) {
	return c.runTx(ctx, "tx", "home", "update-memory-cid", homeID, "--cid", cid)
}

func (c *CLIClient) Stake(ctx context.Context, validatorAddr string, amount uint64) (string, error) {
	return c.runTx(ctx, "tx", "staking", "delegate", validatorAddr,
		fmt.Sprintf("%d%s", amount, Denom))
}

// --- BVM Operations ---

func (c *CLIClient) DeployContract(ctx context.Context, bytecodeHex string) (string, error) {
	return c.runTx(ctx, "tx", "bvm", "deploy-contract",
		"--bytecode", bytecodeHex)
}

func (c *CLIClient) CallContract(ctx context.Context, contractAddr, inputDataHex string, gasLimit uint64) (string, error) {
	args := []string{
		"tx", "bvm", "call-contract", contractAddr,
		"--input", inputDataHex,
	}
	if gasLimit > 0 {
		args = append(args, "--gas-limit", fmt.Sprintf("%d", gasLimit))
	}
	return c.runTx(ctx, args...)
}

func (c *CLIClient) ScheduleExecution(ctx context.Context, contractAddr, method, payload string, atBlock, maxGas uint64) (string, error) {
	return c.runTx(ctx, "tx", "bvm", "schedule-contract", contractAddr,
		"--method", method,
		"--payload", payload,
		"--execute-at", fmt.Sprintf("%d", atBlock),
		"--max-gas", fmt.Sprintf("%d", maxGas))
}

// --- Home Lifecycle Operations ---

func (c *CLIClient) ConfigureGuardian(ctx context.Context, homeID, strategy string, autoDefend bool, deadmanThreshold uint64) (string, error) {
	args := []string{
		"tx", "home", "configure-guardian", homeID,
		"--strategy", strategy,
	}
	if autoDefend {
		args = append(args, "--auto-defend")
	}
	if deadmanThreshold > 0 {
		args = append(args, "--deadman-threshold", fmt.Sprintf("%d", deadmanThreshold))
	}
	return c.runTx(ctx, args...)
}

func (c *CLIClient) RegisterHomeKey(ctx context.Context, homeID, keyHash, role string, permissions []string, expiresAt uint64) (string, error) {
	args := []string{
		"tx", "home", "register-key", homeID,
		"--key-hash", keyHash,
		"--role", role,
	}
	if len(permissions) > 0 {
		args = append(args, "--permissions", strings.Join(permissions, ","))
	}
	if expiresAt > 0 {
		args = append(args, "--expires-at", fmt.Sprintf("%d", expiresAt))
	}
	return c.runTx(ctx, args...)
}

func (c *CLIClient) StartSession(ctx context.Context, homeID, keyHash string, permissions []string) (string, error) {
	args := []string{
		"tx", "home", "start-session", homeID,
		"--key-hash", keyHash,
	}
	if len(permissions) > 0 {
		args = append(args, "--permissions", strings.Join(permissions, ","))
	}
	return c.runTx(ctx, args...)
}

func (c *CLIClient) EndSession(ctx context.Context, homeID, sessionID string) (string, error) {
	return c.runTx(ctx, "tx", "home", "end-session", homeID,
		"--session-id", sessionID)
}

func (c *CLIClient) SetSpendingLimit(ctx context.Context, homeID, keyType string, maxAmount, periodBlocks uint64) (string, error) {
	return c.runTx(ctx, "tx", "home", "set-spending-limit", homeID,
		"--key-type", keyType,
		"--max-amount", fmt.Sprintf("%d%s", maxAmount, Denom),
		"--period-blocks", fmt.Sprintf("%d", periodBlocks))
}
