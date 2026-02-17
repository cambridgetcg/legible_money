//go:build openclaw

package helpers

import (
	"testing"

	"github.com/legible-money/legible/x/testutil/claw"
)

// Insight logs a structured insight from a test probe.
// Delegates to claw.Insight with "OCA-" prefix for devnet integration probes.
func Insight(t *testing.T, testID, msg string) {
	t.Helper()
	claw.Insight(t, "OCA-"+testID, msg)
}

// Economics logs economic analysis of an attack/defense scenario.
func Economics(t *testing.T, testID string, cost, reward, damage int64) {
	t.Helper()
	claw.Economics(t, "OCA-"+testID, cost, reward, damage)
}

// Defense logs a defense mechanism observation.
func Defense(t *testing.T, testID, defense, evidence string) {
	t.Helper()
	claw.Defense(t, "OCA-"+testID, defense, evidence)
}

// Result logs the outcome of a test scenario.
func Result(t *testing.T, testID, outcome string) {
	t.Helper()
	claw.Result(t, "OCA-"+testID, claw.Defended, outcome)
}
