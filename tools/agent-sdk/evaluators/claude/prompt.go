package claude

import (
	"fmt"
	"strings"
)

// ClaimContext holds the information needed to build an evaluation prompt.
type ClaimContext struct {
	ClaimID      string
	Text         string
	Domain       string
	Category     string
	RelatedFacts []string
	SubmitterRep uint32
}

const systemPrompt = `You are a knowledge claim verifier for the Legible Money protocol.
Your task is to evaluate whether a knowledge claim is true, false, or uncertain.

You MUST respond with ONLY a JSON object in this exact format:
{"verdict": "true", "confidence": 85, "reasoning": "Brief explanation"}

Rules:
- "verdict" must be exactly one of: "true", "false", "uncertain"
- "confidence" must be an integer from 0 to 100 (percentage)
- "reasoning" must be a brief explanation (1-3 sentences)
- If you cannot determine truth with reasonable confidence, use "uncertain"
- Base confidence on the strength of available evidence
- Do NOT include any text outside the JSON object`

// BuildEvaluationPrompt constructs the user message for claim evaluation.
func BuildEvaluationPrompt(claim ClaimContext) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Evaluate this knowledge claim (ID: %s):\n\n", claim.ClaimID))
	b.WriteString(fmt.Sprintf("CLAIM: %s\n", claim.Text))

	if claim.Domain != "" {
		b.WriteString(fmt.Sprintf("DOMAIN: %s\n", claim.Domain))
	}
	if claim.Category != "" {
		b.WriteString(fmt.Sprintf("CATEGORY: %s\n", claim.Category))
		b.WriteString(categoryGuidance(claim.Category))
	}

	if len(claim.RelatedFacts) > 0 {
		b.WriteString("\nRELATED ACCEPTED FACTS:\n")
		for i, fact := range claim.RelatedFacts {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, fact))
		}
	}

	return b.String()
}

// categoryGuidance returns evaluation guidance specific to the epistemic category.
func categoryGuidance(category string) string {
	switch strings.ToLower(category) {
	case "analytic":
		return "\nEVALUATION GUIDANCE: This is an analytic claim. Check logical validity, " +
			"internal consistency, and whether the conclusion follows from premises.\n"
	case "empirical":
		return "\nEVALUATION GUIDANCE: This is an empirical claim. Assess the quality of " +
			"evidence cited. If verification requires real-world data you don't have access to, " +
			"use 'uncertain' with lower confidence.\n"
	case "formal", "mathematical":
		return "\nEVALUATION GUIDANCE: This is a formal/mathematical claim. Verify the proof " +
			"structure, check for logical gaps, and confirm the conclusion follows from axioms.\n"
	case "predictive":
		return "\nEVALUATION GUIDANCE: This is a predictive claim. Evaluate the methodology " +
			"and reasoning. Predictive claims inherently carry uncertainty — assign lower " +
			"confidence unless the prediction is trivially derivable.\n"
	case "computational":
		return "\nEVALUATION GUIDANCE: This is a computational claim. Verify the computation " +
			"or algorithm described. Check for correctness of logic and edge cases.\n"
	default:
		return ""
	}
}

// SystemPrompt returns the system prompt for claim evaluation.
func SystemPrompt() string {
	return systemPrompt
}
