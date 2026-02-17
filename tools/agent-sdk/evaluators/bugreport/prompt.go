package bugreport

import (
	"fmt"
	"strings"
)

const bugReportSystemPrompt = `You are a protocol bug report evaluator for the Legible Money blockchain.
Your task is to assess whether a bug report describes a genuine, reproducible protocol issue.

You MUST respond with ONLY a JSON object in this exact format:
{"verdict": "true", "confidence": 85, "severity": "high", "reasoning": "Brief explanation"}

Rules:
- "verdict" must be exactly one of: "true" (real bug), "false" (not a bug), "uncertain" (needs more info)
- "confidence" must be an integer from 0 to 100
- "severity" must be exactly one of: "critical", "high", "medium", "low"
  - critical: consensus-breaking, fund loss, chain halt
  - high: incorrect state transitions, determinism violations
  - medium: edge-case failures, resource leaks under specific conditions
  - low: cosmetic issues, non-functional problems, documentation gaps
- "reasoning" must explain your assessment (1-3 sentences)

Evaluation criteria:
1. REPRODUCIBILITY: Does the report describe specific steps or conditions to trigger the bug?
2. SEVERITY: What is the blast radius? Does it affect consensus, funds, or state?
3. SPECIFICITY: Does it cite specific code paths, function names, or behaviors?
4. NOVELTY: Is this a known issue or a genuinely new finding?

If the report is vague, lacks specifics, or describes expected behavior, verdict should be "false".
If you cannot determine whether the bug is real without more context, use "uncertain".
Do NOT include any text outside the JSON object.`

// BugReportContext holds the information needed to evaluate a bug report.
type BugReportContext struct {
	ClaimID     string
	Text        string
	Domain      string
	Category    string
	CodeContext string   // Relevant source code snippets, if available
	KnownBugs  []string // Previously verified bugs in the same domain
}

// BuildBugReportPrompt constructs the user message for bug report evaluation.
func BuildBugReportPrompt(report BugReportContext) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Evaluate this bug report (Claim ID: %s):\n\n", report.ClaimID))
	b.WriteString(fmt.Sprintf("BUG REPORT: %s\n", report.Text))

	if report.Domain != "" {
		b.WriteString(fmt.Sprintf("DOMAIN: %s\n", report.Domain))
	}

	if report.CodeContext != "" {
		b.WriteString(fmt.Sprintf("\nRELEVANT CODE CONTEXT:\n%s\n", report.CodeContext))
	}

	if len(report.KnownBugs) > 0 {
		b.WriteString("\nPREVIOUSLY VERIFIED BUGS IN THIS DOMAIN:\n")
		for i, bug := range report.KnownBugs {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, bug))
		}
		b.WriteString("\nIf this report duplicates a known bug, verdict should be \"false\".\n")
	}

	return b.String()
}

// BugReportSystemPrompt returns the system prompt for bug report evaluation.
func BugReportSystemPrompt() string {
	return bugReportSystemPrompt
}
