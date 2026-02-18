//go:build openclaw

package claw

import (
	"fmt"
	"strings"
)

// DevnetEnv holds connection info for an ephemeral devnet, used to build
// prompts that instruct the OpenClaw agent how to reach the chain.
type DevnetEnv struct {
	RESTAddr   string // e.g. http://localhost:54321
	RPCAddr    string // e.g. tcp://localhost:54322
	ChainID    string // e.g. lgm-oca-12345
	BinaryPath string // path to compiled legbled binary
	NodeHome   string // harness temp dir (keyring + config)
	KeyName    string // signing key name (e.g. "validator")
}

// CurlCmd returns a curl command that queries the devnet REST API.
func (d DevnetEnv) CurlCmd(path string) string {
	return fmt.Sprintf("curl -s %s%s", d.RESTAddr, path)
}

// EvaluatePrompt builds a prompt asking the agent to evaluate a single claim
// and return a structured verdict. The Go harness handles all blockchain
// transactions — the agent only needs to reason and reply.
func (d DevnetEnv) EvaluatePrompt(claimText string) string {
	var b strings.Builder

	b.WriteString("You are a knowledge verifier on the Legible Money protocol.\n\n")
	b.WriteString("Evaluate this claim and reply with your verdict.\n\n")
	b.WriteString(fmt.Sprintf("**Claim**: %q\n\n", claimText))
	b.WriteString("Reply in EXACTLY this format (3 lines, no extra text):\n\n")
	b.WriteString("```\n")
	b.WriteString("VERDICT=true\n")
	b.WriteString("CONFIDENCE=9000\n")
	b.WriteString("SALT=myRandomSalt123abc\n")
	b.WriteString("```\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- VERDICT must be: `true`, `false`, or `uncertain`\n")
	b.WriteString("- CONFIDENCE is 0-10000 BPS (10000 = 100%)\n")
	b.WriteString("- SALT is any random alphanumeric string, 16+ characters\n")
	b.WriteString("- Reply ONLY with the 3 lines above, nothing else\n")

	return b.String()
}

// BatchEvaluatePrompt builds a prompt asking the agent to evaluate multiple
// claims and return structured verdicts for each.
func (d DevnetEnv) BatchEvaluatePrompt(claims []string) string {
	var b strings.Builder

	b.WriteString("You are a knowledge verifier on the Legible Money protocol.\n\n")
	b.WriteString(fmt.Sprintf("Evaluate these %d claims and reply with your verdicts.\n\n", len(claims)))

	for i, c := range claims {
		b.WriteString(fmt.Sprintf("**Claim %d**: %q\n", i+1, c))
	}
	b.WriteString("\n")

	b.WriteString("Reply in EXACTLY this format (3 lines per claim, separated by blank lines):\n\n")
	b.WriteString("```\n")
	for i := range claims {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("CLAIM_%d_VERDICT=true\n", i+1))
		b.WriteString(fmt.Sprintf("CLAIM_%d_CONFIDENCE=9000\n", i+1))
		b.WriteString(fmt.Sprintf("CLAIM_%d_SALT=randomString%dabc\n", i+1, i+1))
	}
	b.WriteString("```\n\n")

	b.WriteString("Rules:\n")
	b.WriteString("- VERDICT must be: `true`, `false`, or `uncertain`\n")
	b.WriteString("- CONFIDENCE is 0-10000 BPS (10000 = 100%)\n")
	b.WriteString("- SALT is any random alphanumeric string, 16+ characters, UNIQUE per claim\n")
	b.WriteString("- Reply ONLY with the lines above, nothing else\n")

	return b.String()
}

// AgentVerdict holds a parsed evaluation from the agent.
type AgentVerdict struct {
	Verdict    string // "true", "false", or "uncertain"
	Confidence uint32 // 0-10000 BPS
	Salt       string
}

// ParseVerdict extracts VERDICT, CONFIDENCE, and SALT from agent reply text.
// Returns nil if parsing fails.
func ParseVerdict(reply string) *AgentVerdict {
	v := &AgentVerdict{}
	for _, line := range strings.Split(reply, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "VERDICT=") {
			v.Verdict = strings.TrimPrefix(line, "VERDICT=")
		} else if strings.HasPrefix(line, "CONFIDENCE=") {
			fmt.Sscanf(strings.TrimPrefix(line, "CONFIDENCE="), "%d", &v.Confidence)
		} else if strings.HasPrefix(line, "SALT=") {
			v.Salt = strings.TrimPrefix(line, "SALT=")
		}
	}
	if v.Verdict != "" && v.Salt != "" {
		return v
	}
	return nil
}

// ParseBatchVerdicts extracts numbered verdicts from a batch reply.
func ParseBatchVerdicts(reply string, count int) []*AgentVerdict {
	results := make([]*AgentVerdict, count)
	for i := range results {
		results[i] = &AgentVerdict{}
	}

	for _, line := range strings.Split(reply, "\n") {
		line = strings.TrimSpace(line)
		for i := 0; i < count; i++ {
			prefix := fmt.Sprintf("CLAIM_%d_", i+1)
			if strings.HasPrefix(line, prefix+"VERDICT=") {
				results[i].Verdict = strings.TrimPrefix(line, prefix+"VERDICT=")
			} else if strings.HasPrefix(line, prefix+"CONFIDENCE=") {
				fmt.Sscanf(strings.TrimPrefix(line, prefix+"CONFIDENCE="), "%d", &results[i].Confidence)
			} else if strings.HasPrefix(line, prefix+"SALT=") {
				results[i].Salt = strings.TrimPrefix(line, prefix+"SALT=")
			}
		}
	}

	// Filter to only valid verdicts
	var valid []*AgentVerdict
	for _, v := range results {
		if v.Verdict != "" && v.Salt != "" {
			valid = append(valid, v)
		}
	}
	return valid
}
