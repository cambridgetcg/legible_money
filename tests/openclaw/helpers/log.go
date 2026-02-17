//go:build openclaw

package helpers

import "testing"

// Insight logs a structured insight from a test probe.
// Pattern: extends "OC-1 INSIGHT: ..." convention from openclaw_test.go.
func Insight(t *testing.T, testID, msg string) {
	t.Helper()
	t.Logf("OCA-%s INSIGHT: %s", testID, msg)
}

// Economics logs economic analysis of an attack/defense scenario.
func Economics(t *testing.T, testID string, cost, reward, damage int64) {
	t.Helper()
	t.Logf("OCA-%s ECONOMICS: cost=%d ulgm, reward=%d ulgm, damage=%d ulgm, profit=%d ulgm",
		testID, cost, reward, damage, reward-cost)
}

// Defense logs a defense mechanism observation.
func Defense(t *testing.T, testID, defense, evidence string) {
	t.Helper()
	t.Logf("OCA-%s DEFENSE: %s — %s", testID, defense, evidence)
}

// Result logs the outcome of a test scenario.
func Result(t *testing.T, testID, outcome string) {
	t.Helper()
	t.Logf("OCA-%s RESULT: %s", testID, outcome)
}
