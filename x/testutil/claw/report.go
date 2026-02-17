package claw

import "testing"

// Insight logs a structured finding from a probe.
func Insight(t *testing.T, probeID, msg string) {
	t.Helper()
	t.Logf("%s INSIGHT: %s", probeID, msg)
}

// Economics logs attack cost/reward analysis.
func Economics(t *testing.T, probeID string, cost, reward, damage int64) {
	t.Helper()
	profit := reward - cost
	t.Logf("%s ECONOMICS: cost=%d ulgm, reward=%d ulgm, damage=%d ulgm, profit=%d ulgm",
		probeID, cost, reward, damage, profit)
}

// Defense logs a defense mechanism observation.
func Defense(t *testing.T, probeID, defense, evidence string) {
	t.Helper()
	t.Logf("%s DEFENSE: %s — %s", probeID, defense, evidence)
}

// Result logs the probe outcome with a status classification.
func Result(t *testing.T, probeID string, status Status, verdict string) {
	t.Helper()
	t.Logf("%s RESULT [%s]: %s", probeID, status, verdict)
}
