package orchestrator

import (
	"context"
	"fmt"
	"strconv"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// ActionExecutor bridges part actions to agent-sdk method calls.
type ActionExecutor interface {
	Execute(ctx context.Context, agent *agentsdk.Agent, action ActionType, params map[string]string) (*PartResult, error)
}

// StandardExecutor dispatches actions to the corresponding agent-sdk methods.
type StandardExecutor struct {
	// OpenClaw is optional — only used for ActionCustom.
	OpenClaw *OpenClawManager
}

// NewStandardExecutor creates an executor with optional OpenClaw support.
func NewStandardExecutor(ocm *OpenClawManager) *StandardExecutor {
	return &StandardExecutor{OpenClaw: ocm}
}

// Execute dispatches to the appropriate agent-sdk method based on action type.
func (e *StandardExecutor) Execute(ctx context.Context, agent *agentsdk.Agent, action ActionType, params map[string]string) (*PartResult, error) {
	switch action {
	case ActionRegister:
		return e.doRegister(ctx, agent, params)
	case ActionVerify:
		return e.doVerify(ctx, agent, params)
	case ActionAutoVerify:
		return e.doAutoVerify(ctx, agent, params)
	case ActionSubmitClaim:
		return e.doSubmitClaim(ctx, agent, params)
	case ActionCreateHome:
		return e.doCreateHome(ctx, agent, params)
	case ActionSyncHome:
		return e.doSyncHome(ctx, agent, params)
	case ActionDeployContract:
		return e.doDeployContract(ctx, agent, params)
	case ActionSchedule:
		return e.doSchedule(ctx, agent, params)
	case ActionStake:
		return e.doStake(ctx, agent, params)
	case ActionHeartbeat:
		return e.doHeartbeat(ctx, agent)
	case ActionCustom:
		return e.doCustom(ctx, agent, params)
	default:
		return nil, fmt.Errorf("unknown action type: %s", action)
	}
}

func (e *StandardExecutor) doRegister(ctx context.Context, agent *agentsdk.Agent, params map[string]string) (*PartResult, error) {
	displayName := params["display_name"]
	if displayName == "" {
		displayName = "orchestrated-agent"
	}
	domains := splitCSV(params["domains"])
	stake := parseUint64(params["stake"])

	err := agent.Register(ctx, displayName, domains, stake)
	if err != nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{
		Success: true,
		Data:    map[string]string{"address": agent.Address(), "did": agent.DID()},
	}, nil
}

func (e *StandardExecutor) doVerify(ctx context.Context, agent *agentsdk.Agent, params map[string]string) (*PartResult, error) {
	roundID := params["round_id"]
	verdict := agentsdk.Verdict(params["verdict"])
	confidence := uint32(parseUint64(params["confidence"]))

	txHash, err := agent.Verify(ctx, roundID, verdict, confidence)
	if err != nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{Success: true, TxHash: txHash}, nil
}

func (e *StandardExecutor) doAutoVerify(ctx context.Context, agent *agentsdk.Agent, params map[string]string) (*PartResult, error) {
	// AutoVerify is a long-running loop — we start it and report success.
	// The caller should use a context with cancellation for lifecycle control.
	evaluatorName := params["evaluator"]
	_ = evaluatorName // evaluator selection handled by caller

	err := agent.AutoVerify(ctx, nil)
	if err != nil && ctx.Err() == nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{Success: true}, nil
}

func (e *StandardExecutor) doSubmitClaim(ctx context.Context, agent *agentsdk.Agent, params map[string]string) (*PartResult, error) {
	text := params["text"]
	domain := params["domain"]
	category := params["category"]
	stake := parseUint64(params["stake"])

	txHash, err := agent.SubmitClaim(ctx, text, domain, category, stake)
	if err != nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{Success: true, TxHash: txHash}, nil
}

func (e *StandardExecutor) doCreateHome(ctx context.Context, agent *agentsdk.Agent, params map[string]string) (*PartResult, error) {
	name := params["name"]
	if name == "" {
		name = "default"
	}
	strategy := params["defense_strategy"]
	threshold := parseUint64(params["deadman_threshold"])

	txHash, err := agent.CreateAndConfigureHome(ctx, name, strategy, threshold)
	if err != nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{Success: true, TxHash: txHash}, nil
}

func (e *StandardExecutor) doSyncHome(ctx context.Context, agent *agentsdk.Agent, params map[string]string) (*PartResult, error) {
	homeID := params["home_id"]
	memoryCID := params["memory_cid"]

	err := agent.SyncHome(ctx, homeID, memoryCID)
	if err != nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{Success: true}, nil
}

func (e *StandardExecutor) doDeployContract(ctx context.Context, agent *agentsdk.Agent, params map[string]string) (*PartResult, error) {
	bytecodeHex := params["bytecode_hex"]

	txHash, err := agent.DeployContract(ctx, bytecodeHex)
	if err != nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{
		Success: true,
		TxHash:  txHash,
		Data:    map[string]string{"contract_tx": txHash},
	}, nil
}

func (e *StandardExecutor) doSchedule(ctx context.Context, agent *agentsdk.Agent, params map[string]string) (*PartResult, error) {
	contractAddr := params["contract_addr"]
	method := params["method"]
	payload := params["payload"]
	atBlock := parseUint64(params["at_block"])
	maxGas := parseUint64(params["max_gas"])

	txHash, err := agent.ScheduleExecution(ctx, contractAddr, method, payload, atBlock, maxGas)
	if err != nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{Success: true, TxHash: txHash}, nil
}

func (e *StandardExecutor) doStake(ctx context.Context, agent *agentsdk.Agent, params map[string]string) (*PartResult, error) {
	validatorAddr := params["validator_addr"]
	amount := parseUint64(params["amount"])

	tc := agent.TxClient()
	if tc == nil {
		return &PartResult{Success: false, Error: "no tx client configured"}, nil
	}
	txHash, err := tc.Stake(ctx, validatorAddr, amount)
	if err != nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{Success: true, TxHash: txHash}, nil
}

func (e *StandardExecutor) doHeartbeat(ctx context.Context, agent *agentsdk.Agent) (*PartResult, error) {
	tc := agent.TxClient()
	if tc == nil {
		return &PartResult{Success: false, Error: "no tx client configured"}, nil
	}
	txHash, err := tc.Heartbeat(ctx)
	if err != nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{Success: true, TxHash: txHash}, nil
}

func (e *StandardExecutor) doCustom(ctx context.Context, agent *agentsdk.Agent, params map[string]string) (*PartResult, error) {
	if e.OpenClaw == nil {
		return nil, fmt.Errorf("custom action requires OpenClawManager")
	}
	command := params["command"]
	sessionID := params["session_id"]
	if sessionID == "" {
		return nil, fmt.Errorf("custom action requires session_id param")
	}
	resp, err := e.OpenClaw.SendCommand(ctx, sessionID, command)
	if err != nil {
		return &PartResult{Success: false, Error: err.Error()}, nil
	}
	return &PartResult{
		Success: true,
		Data:    map[string]string{"response": resp},
	}, nil
}

// splitCSV splits a comma-separated string into trimmed parts.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			// Trim spaces.
			for len(part) > 0 && part[0] == ' ' {
				part = part[1:]
			}
			for len(part) > 0 && part[len(part)-1] == ' ' {
				part = part[:len(part)-1]
			}
			if len(part) > 0 {
				parts = append(parts, part)
			}
			start = i + 1
		}
	}
	return parts
}

func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
