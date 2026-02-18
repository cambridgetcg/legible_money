package agentsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	return extractJSON(out), nil
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

	// Extract JSON portion from output. The --gas auto flag prints "gas estimate: N"
	// to stderr which gets mixed into CombinedOutput. Find the JSON object start.
	jsonBytes := extractJSON(out)

	// Extract tx hash and check result code from JSON response
	var resp struct {
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
		RawLog string `json:"raw_log"`
	}
	if json.Unmarshal(jsonBytes, &resp) == nil {
		if resp.Code != 0 {
			return "", fmt.Errorf("tx failed with code %d: %s", resp.Code, resp.RawLog)
		}
		if resp.TxHash != "" {
			return resp.TxHash, nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}

// extractJSON finds the first JSON object in a byte slice.
// Handles output like "gas estimate: 166168\n{...json...}" from --gas auto.
func extractJSON(data []byte) []byte {
	idx := strings.Index(string(data), "{")
	if idx >= 0 {
		return data[idx:]
	}
	return data
}

// RawQuery runs a CLI query and returns the raw output bytes (for diagnostics).
func (c *CLIClient) RawQuery(ctx context.Context, args ...string) ([]byte, error) {
	fullArgs := append(args, "--node", c.Node, "--output", "json")
	cmd := exec.CommandContext(ctx, c.Binary, fullArgs...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("raw query failed: %s: %w", string(out), err)
	}
	return out, nil
}

// QueryTxResult queries a tx by hash and returns the DeliverTx result code and raw_log.
// This catches failures that sync broadcast mode doesn't report (CheckTx passes but DeliverTx fails).
func (c *CLIClient) QueryTxResult(ctx context.Context, txHash string) (code int, rawLog string, err error) {
	out, qErr := c.runQuery(ctx, "query", "tx", txHash)
	if qErr != nil {
		return -1, "", qErr
	}
	var resp struct {
		Code   int    `json:"code"`
		RawLog string `json:"raw_log"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return -1, "", fmt.Errorf("parsing tx result: %w", err)
	}
	return resp.Code, resp.RawLog, nil
}

// --- QueryClient Implementation ---

func (c *CLIClient) ChainStatus(ctx context.Context) (*ChainStatus, error) {
	// Try CLI status command first (may be "status" or "comet status")
	var out []byte
	var err error

	for _, args := range [][]string{
		{"status", "--node", c.Node, "--output", "json"},
		{"comet", "status", "--node", c.Node, "--output", "json"},
	} {
		cmd := exec.CommandContext(ctx, c.Binary, args...)
		cmd.Env = os.Environ()
		out, err = cmd.CombinedOutput()
		if err == nil {
			break
		}
	}

	// If CLI fails, fall back to direct HTTP RPC query
	if err != nil {
		rpcHost := strings.TrimPrefix(c.Node, "tcp://")
		url := fmt.Sprintf("http://%s/status", rpcHost)
		resp, httpErr := http.Get(url)
		if httpErr != nil {
			return nil, fmt.Errorf("status query failed (CLI: %w, HTTP: %v)", err, httpErr)
		}
		defer resp.Body.Close()

		var rpcResp struct {
			Result json.RawMessage `json:"result"`
		}
		if decErr := json.NewDecoder(resp.Body).Decode(&rpcResp); decErr != nil {
			return nil, fmt.Errorf("parsing RPC status: %w", decErr)
		}
		out = rpcResp.Result
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
	// Bank query CLI doesn't exist in SDK v0.50 (uses autocli).
	// Use REST API (gRPC-gateway) instead.
	if c.RESTAddr == "" {
		return 0, fmt.Errorf("RESTAddr not set on CLIClient — cannot query balance")
	}
	url := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", c.RESTAddr, address)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("creating balance request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("balance query failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("parsing balance: %w", err)
	}
	for _, b := range result.Balances {
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
		return nil, fmt.Errorf("parsing rounds (raw: %.300s): %w", string(out), err)
	}
	return resp.Rounds, nil
}

func (c *CLIClient) RoundDetail(ctx context.Context, roundID string) (*RoundInfo, error) {
	out, err := c.runQuery(ctx, "query", "knowledge", "round", roundID)
	if err != nil {
		return nil, err
	}

	// Parse basic fields via RoundInfo struct
	var resp struct {
		Round RoundInfo `json:"round"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing round: %w", err)
	}

	// Extract commitment/reveal counts and final verdict from the full proto response.
	// The proto JSON has commitments/reveals as maps, final_verdict, final_confidence.
	var raw struct {
		Round struct {
			Commitments     map[string]json.RawMessage `json:"commitments"`
			Reveals         map[string]json.RawMessage `json:"reveals"`
			FinalVerdict    string                     `json:"final_verdict"`
			FinalConfidence uint64                     `json:"final_confidence,string"`
		} `json:"round"`
	}
	if json.Unmarshal(out, &raw) == nil {
		resp.Round.CommitmentCount = len(raw.Round.Commitments)
		resp.Round.RevealCount = len(raw.Round.Reveals)
		resp.Round.FinalVerdict = raw.Round.FinalVerdict
		resp.Round.FinalConfidence = raw.Round.FinalConfidence
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
	// Knowledge module CLI doesn't have a claims-by-submitter query yet.
	// Return empty for now — claims can be queried by domain instead.
	return nil, fmt.Errorf("ClaimsBySubmitter: query not implemented in knowledge CLI")
}

func (c *CLIClient) GetClaim(ctx context.Context, claimID string) (*ClaimInfo, error) {
	// Not yet implemented in knowledge CLI — will be wired via gRPC.
	return nil, fmt.Errorf("GetClaim: query not implemented in knowledge CLI")
}

func (c *CLIClient) FactReferences(ctx context.Context, factID string) ([]FactReference, error) {
	// Not yet implemented in knowledge CLI — will be wired via gRPC.
	return nil, fmt.Errorf("FactReferences: query not implemented in knowledge CLI")
}

func (c *CLIClient) CounterFacts(ctx context.Context, factID string) ([]CounterFactInfo, error) {
	// Not yet implemented in knowledge CLI — will be wired via gRPC.
	return nil, fmt.Errorf("CounterFacts: query not implemented in knowledge CLI")
}

func (c *CLIClient) BugFacts(ctx context.Context, domain string) ([]FactInfo, error) {
	// Fetch all facts in the domain and filter to protocol_observation category.
	facts, err := c.FactsByDomain(ctx, domain)
	if err != nil {
		return nil, err
	}
	var bugs []FactInfo
	for _, f := range facts {
		if f.Category == "protocol_observation" {
			bugs = append(bugs, f)
		}
	}
	return bugs, nil
}

func (c *CLIClient) ProposalsByTrigger(ctx context.Context, triggerType string) ([]ProposalInfo, error) {
	out, err := c.runQuery(ctx, "query", "gov", "proposals", "--status", "voting")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Proposals []ProposalInfo `json:"proposals"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing proposals: %w", err)
	}

	// Filter by trigger type if specified.
	if triggerType == "" {
		return resp.Proposals, nil
	}
	var filtered []ProposalInfo
	for _, p := range resp.Proposals {
		if p.TriggerType == triggerType {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}

// --- TxClient Implementation ---

func (c *CLIClient) RegisterAccount(ctx context.Context, pubKeyHex string) (string, error) {
	// legible_auth module has no CLI commands yet.
	// Accounts are implicitly created by the SDK when they receive funds.
	return "", fmt.Errorf("RegisterAccount: no CLI command available (accounts auto-created on first fund)")
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
	// CLI: submit-claim [claim-text] [domain] [category] [stake-amount]
	// StakeAmount is a plain integer (parsed via big.Int.SetString base 10), no denom suffix.
	return c.runTx(ctx, "tx", "knowledge", "submit-claim",
		req.Text, req.Domain, req.Category,
		fmt.Sprintf("%d", req.Stake))
}

func (c *CLIClient) SubmitCommitment(ctx context.Context, roundID, commitment string) (string, error) {
	// CLI: submit-commitment [round-id] [commitment-hash]
	return c.runTx(ctx, "tx", "knowledge", "submit-commitment", roundID, commitment)
}

func (c *CLIClient) SubmitReveal(ctx context.Context, roundID string, verdict Verdict, confidence uint32, salt string) (string, error) {
	// CLI: submit-reveal [round-id] [verdict] [confidence] [salt]
	return c.runTx(ctx, "tx", "knowledge", "submit-reveal",
		roundID, string(verdict), fmt.Sprintf("%d", confidence), salt)
}

func (c *CLIClient) ChallengeFact(ctx context.Context, factID, reason string, stake uint64) (string, error) {
	// CLI: challenge-fact [fact-id] [reason] [stake]
	// Stake is a plain integer (parsed via big.Int.SetString base 10), no denom suffix.
	return c.runTx(ctx, "tx", "knowledge", "challenge-fact",
		factID, reason, fmt.Sprintf("%d", stake))
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

// --- Governance Operations ---

func (c *CLIClient) SubmitLIP(ctx context.Context, req SubmitLIPRequest) (string, error) {
	args := []string{
		"tx", "gov", "submit-proposal",
		"--title", req.Title,
		"--description", req.Description,
		"--deposit", fmt.Sprintf("%d%s", req.Deposit, Denom),
	}
	if req.FixClaimID != "" {
		args = append(args, "--fix-claim-id", req.FixClaimID)
	}
	if len(req.BugFactIDs) > 0 {
		args = append(args, "--bug-fact-ids", strings.Join(req.BugFactIDs, ","))
	}
	return c.runTx(ctx, args...)
}

// --- Vesting Rewards Operations ---

func (c *CLIClient) CreateVesting(ctx context.Context, req CreateVestingRequest) (string, error) {
	args := []string{
		"tx", "vesting-rewards", "create-vesting",
		req.Beneficiary,
		fmt.Sprintf("%d%s", req.Amount, Denom),
		"--source", req.Source,
	}
	if req.Category != "" {
		args = append(args, "--category", req.Category)
	}
	if req.Description != "" {
		args = append(args, "--description", req.Description)
	}
	return c.runTx(ctx, args...)
}

// --- Dispute Operations (x/disputes) ---

func (c *CLIClient) GetDispute(ctx context.Context, disputeID string) (*DisputeInfo, error) {
	out, err := c.runQuery(ctx, "query", "disputes", "dispute", disputeID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Dispute DisputeInfo `json:"dispute"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing dispute: %w", err)
	}
	return &resp.Dispute, nil
}

func (c *CLIClient) DisputesByFact(ctx context.Context, factID string) ([]DisputeInfo, error) {
	out, err := c.runQuery(ctx, "query", "disputes", "disputes-by-fact", factID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Disputes []DisputeInfo `json:"disputes"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing disputes: %w", err)
	}
	return resp.Disputes, nil
}

func (c *CLIClient) ActiveDisputes(ctx context.Context) ([]DisputeInfo, error) {
	out, err := c.runQuery(ctx, "query", "disputes", "active-disputes")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Disputes []DisputeInfo `json:"disputes"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing active disputes: %w", err)
	}
	return resp.Disputes, nil
}

func (c *CLIClient) DisputeVotes(ctx context.Context, disputeID string) ([]DisputeVoteInfo, error) {
	out, err := c.runQuery(ctx, "query", "disputes", "votes", disputeID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Votes []DisputeVoteInfo `json:"votes"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing votes: %w", err)
	}
	return resp.Votes, nil
}

func (c *CLIClient) InitiateDispute(ctx context.Context, req InitiateDisputeRequest) (string, error) {
	return c.runTx(ctx, "tx", "disputes", "initiate-dispute",
		req.TargetID, req.Reason,
		"--stake", fmt.Sprintf("%d%s", req.Stake, Denom))
}

func (c *CLIClient) CommitDisputeEvidence(ctx context.Context, disputeID, commitmentHash string) (string, error) {
	return c.runTx(ctx, "tx", "disputes", "commit-evidence", disputeID, commitmentHash)
}

func (c *CLIClient) RevealDisputeEvidence(ctx context.Context, req RevealDisputeEvidenceRequest) (string, error) {
	args := []string{
		"tx", "disputes", "reveal-evidence", req.DisputeID,
		"--nonce", req.Nonce,
	}
	for _, e := range req.Evidence {
		args = append(args, "--evidence", e)
	}
	return c.runTx(ctx, args...)
}

func (c *CLIClient) ArbiterVote(ctx context.Context, req ArbiterVoteRequest) (string, error) {
	return c.runTx(ctx, "tx", "disputes", "arbiter-vote",
		req.DisputeID, string(req.Vote),
		"--reasoning", req.Reasoning)
}

func (c *CLIClient) EscalateDispute(ctx context.Context, disputeID string, additionalStake uint64) (string, error) {
	return c.runTx(ctx, "tx", "disputes", "escalate-dispute", disputeID,
		"--additional-stake", fmt.Sprintf("%d%s", additionalStake, Denom))
}

// --- Evidence Management Operations (x/evidence_mgmt) ---

func (c *CLIClient) EvidenceByClaimID(ctx context.Context, claimID string) ([]EvidenceInfo, error) {
	out, err := c.runQuery(ctx, "query", "evidence-mgmt", "evidence-by-claim", claimID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Evidence []EvidenceInfo `json:"evidence"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing evidence: %w", err)
	}
	return resp.Evidence, nil
}

func (c *CLIClient) ActiveOracles(ctx context.Context) ([]OracleInfo, error) {
	out, err := c.runQuery(ctx, "query", "evidence-mgmt", "active-oracles")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Oracles []OracleInfo `json:"oracles"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing oracles: %w", err)
	}
	return resp.Oracles, nil
}

func (c *CLIClient) SubmitEvidence(ctx context.Context, req SubmitEvidenceRequest) (string, error) {
	args := []string{
		"tx", "evidence-mgmt", "submit-evidence",
		req.ClaimID, req.EvidenceType, req.ContentHash,
		"--description", req.Description,
	}
	if req.SourceURI != "" {
		args = append(args, "--source-uri", req.SourceURI)
	}
	return c.runTx(ctx, args...)
}

func (c *CLIClient) AuditEvidence(ctx context.Context, req AuditEvidenceRequest) (string, error) {
	return c.runTx(ctx, "tx", "evidence-mgmt", "audit-evidence",
		req.EvidenceID,
		"--stake", fmt.Sprintf("%d%s", req.Stake, Denom),
		"--score", fmt.Sprintf("%d", req.Score),
		"--rationale", req.Rationale)
}

func (c *CLIClient) RegisterOracle(ctx context.Context, req RegisterOracleRequest) (string, error) {
	args := []string{
		"tx", "evidence-mgmt", "register-oracle",
		req.Name,
		"--stake", fmt.Sprintf("%d%s", req.Stake, Denom),
	}
	if len(req.DataTypes) > 0 {
		args = append(args, "--data-types", strings.Join(req.DataTypes, ","))
	}
	return c.runTx(ctx, args...)
}

func (c *CLIClient) SubmitOracleData(ctx context.Context, req SubmitOracleDataRequest) (string, error) {
	return c.runTx(ctx, "tx", "evidence-mgmt", "submit-oracle-data",
		req.OracleID, req.ClaimID, req.DataType, req.ContentHash)
}

// --- Research Operations (x/research) ---

func (c *CLIClient) GetResearch(ctx context.Context, researchID string) (*ResearchInfo, error) {
	out, err := c.runQuery(ctx, "query", "research", "research", researchID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Research ResearchInfo `json:"research"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing research: %w", err)
	}
	return &resp.Research, nil
}

func (c *CLIClient) ResearchSubmissions(ctx context.Context, status, domain string) ([]ResearchInfo, error) {
	args := []string{"query", "research", "submissions"}
	if status != "" {
		args = append(args, "--status", status)
	}
	if domain != "" {
		args = append(args, "--domain", domain)
	}
	out, err := c.runQuery(ctx, args...)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Submissions []ResearchInfo `json:"submissions"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing submissions: %w", err)
	}
	return resp.Submissions, nil
}

func (c *CLIClient) GetBounty(ctx context.Context, bountyID string) (*ResearchBountyInfo, error) {
	out, err := c.runQuery(ctx, "query", "research", "bounty", bountyID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Bounty ResearchBountyInfo `json:"bounty"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing bounty: %w", err)
	}
	return &resp.Bounty, nil
}

func (c *CLIClient) SubmitResearch(ctx context.Context, req SubmitResearchRequest) (string, error) {
	return c.runTx(ctx, "tx", "research", "submit-research",
		req.Title, req.Description, req.Domain,
		"--stake", fmt.Sprintf("%d%s", req.Stake, Denom))
}

func (c *CLIClient) ReviewResearch(ctx context.Context, req ReviewResearchRequest) (string, error) {
	return c.runTx(ctx, "tx", "research", "review-research",
		req.ResearchID, string(req.Verdict),
		"--quality-score", fmt.Sprintf("%d", req.QualityScore),
		"--reasoning", req.Reasoning)
}

func (c *CLIClient) CreateResearchBounty(ctx context.Context, req CreateResearchBountyRequest) (string, error) {
	return c.runTx(ctx, "tx", "research", "create-bounty",
		req.Title, req.Description,
		"--reward", fmt.Sprintf("%d%s", req.Reward, Denom),
		"--deadline-height", fmt.Sprintf("%d", req.DeadlineHeight))
}

func (c *CLIClient) ClaimResearchBounty(ctx context.Context, bountyID string) (string, error) {
	return c.runTx(ctx, "tx", "research", "claim-bounty", bountyID)
}

func (c *CLIClient) FundResearch(ctx context.Context, amount uint64) (string, error) {
	return c.runTx(ctx, "tx", "research", "fund-research",
		fmt.Sprintf("%d%s", amount, Denom))
}

// --- Ontology Operations (x/ontology) ---

func (c *CLIClient) AllDomains(ctx context.Context) ([]DomainInfo, error) {
	out, err := c.runQuery(ctx, "query", "ontology", "all-domains")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Domains []DomainInfo `json:"domains"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing domains: %w", err)
	}
	return resp.Domains, nil
}

func (c *CLIClient) DomainsByStratum(ctx context.Context, stratum uint32) ([]DomainInfo, error) {
	out, err := c.runQuery(ctx, "query", "ontology", "domains-by-stratum", fmt.Sprintf("%d", stratum))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Domains []DomainInfo `json:"domains"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing domains: %w", err)
	}
	return resp.Domains, nil
}

func (c *CLIClient) ConfidenceCeiling(ctx context.Context, domain string) (*ConfidenceCeilingInfo, error) {
	out, err := c.runQuery(ctx, "query", "ontology", "confidence-ceiling", domain)
	if err != nil {
		return nil, err
	}
	var resp ConfidenceCeilingInfo
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing confidence ceiling: %w", err)
	}
	return &resp, nil
}

func (c *CLIClient) AllLogicZones(ctx context.Context) ([]LogicZoneInfo, error) {
	out, err := c.runQuery(ctx, "query", "ontology", "all-logic-zones")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Zones []LogicZoneInfo `json:"logic_zones"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing logic zones: %w", err)
	}
	return resp.Zones, nil
}

func (c *CLIClient) ProposeOntologyDomain(ctx context.Context, req ProposeOntologyDomainRequest) (string, error) {
	return c.runTx(ctx, "tx", "ontology", "propose-domain",
		req.Name, req.DisplayName,
		"--description", req.Description,
		"--stratum", fmt.Sprintf("%d", req.Stratum),
		"--stake", fmt.Sprintf("%d%s", req.Stake, Denom))
}

func (c *CLIClient) VoteDomainProposal(ctx context.Context, proposalID string, approve bool) (string, error) {
	voteStr := "false"
	if approve {
		voteStr = "true"
	}
	return c.runTx(ctx, "tx", "ontology", "vote-domain-proposal", proposalID, voteStr)
}

func (c *CLIClient) AcknowledgeIncompleteness(ctx context.Context, factID, zone, reason string) (string, error) {
	return c.runTx(ctx, "tx", "ontology", "acknowledge-incompleteness",
		factID, zone,
		"--reason", reason)
}
