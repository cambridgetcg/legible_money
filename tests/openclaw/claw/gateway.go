//go:build openclaw

// Package claw provides the OpenClaw gateway client for Phase D agent tests.
// It wraps the `openclaw agent` CLI to send messages to a running OpenClaw
// gateway and receive AI-generated replies.
package claw

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// GatewayClient sends messages to the OpenClaw agent via the gateway CLI.
type GatewayClient struct {
	// Binary is the path to the openclaw binary. Defaults to "openclaw".
	Binary string
	// Timeout per agent call. Defaults to 120s.
	Timeout time.Duration
	// SessionID isolates agent memory across test runs.
	SessionID string
}

// NewGatewayClient creates a client with sensible defaults.
func NewGatewayClient(sessionID string) *GatewayClient {
	return &GatewayClient{
		Binary:    "openclaw",
		Timeout:   120 * time.Second,
		SessionID: sessionID,
	}
}

// agentPayload is a single reply payload from the agent.
type agentPayload struct {
	Text string `json:"text"`
}

// agentResult holds the structured result from the agent.
type agentResult struct {
	Payloads []agentPayload `json:"payloads"`
	Meta     json.RawMessage `json:"meta,omitempty"`
}

// agentResponse is the JSON envelope from `openclaw agent --json`.
// The actual schema is: { status, summary, result: { payloads: [{ text }] } }
type agentResponse struct {
	Status  string      `json:"status"`
	Summary string      `json:"summary"`
	Result  agentResult `json:"result"`
	// Legacy/fallback fields
	Reply   string `json:"reply"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Send dispatches a message to the OpenClaw agent and returns the reply text.
// It shells out to `openclaw agent -m "..." --json --session-id <id> --timeout <s>`.
func (g *GatewayClient) Send(ctx context.Context, message string) (string, error) {
	binary := g.Binary
	if binary == "" {
		binary = "openclaw"
	}
	timeout := g.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	args := []string{
		"agent",
		"-m", message,
		"--json",
		"--timeout", fmt.Sprintf("%d", int(timeout.Seconds())),
	}
	if g.SessionID != "" {
		args = append(args, "--session-id", g.SessionID)
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("openclaw agent: %w\noutput: %s", err, string(out))
	}

	return parseAgentOutput(out)
}

// parseAgentOutput extracts the reply text from --json output.
// Primary path: result.payloads[0].text (OpenClaw gateway format).
// Falls back to legacy fields and raw stdout.
func parseAgentOutput(raw []byte) (string, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", fmt.Errorf("empty response from gateway")
	}

	var resp agentResponse
	if err := json.Unmarshal([]byte(text), &resp); err == nil {
		if resp.Error != "" {
			return "", fmt.Errorf("agent error: %s", resp.Error)
		}
		// Primary: result.payloads[0].text
		if len(resp.Result.Payloads) > 0 && resp.Result.Payloads[0].Text != "" {
			// Concatenate all payloads if multiple
			var parts []string
			for _, p := range resp.Result.Payloads {
				if p.Text != "" {
					parts = append(parts, p.Text)
				}
			}
			return strings.Join(parts, "\n"), nil
		}
		// Legacy fields
		if resp.Reply != "" {
			return resp.Reply, nil
		}
		if resp.Message != "" {
			return resp.Message, nil
		}
	}

	// Fallback: treat the entire output as the reply.
	return text, nil
}

// RequireGateway checks that the OpenClaw gateway is reachable.
// Calls t.Skip if the gateway is not running.
func RequireGateway(t *testing.T) {
	t.Helper()

	binary := "openclaw"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "health", "--json", "--timeout", "10000")
	out, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))

	// The health command may get killed by the context timeout after printing
	// valid JSON. Check for "ok":true in the output regardless of exit code.
	if strings.Contains(outStr, `"ok":true`) || strings.Contains(outStr, `"ok": true`) {
		t.Logf("gateway health: ok")
		return
	}

	if err != nil {
		t.Skipf("openclaw gateway not running (health check failed: %v)\noutput: %s", err, outStr)
	}
	t.Logf("gateway health: %s", outStr)
}
