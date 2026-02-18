//go:build openclaw

package helpers

import (
	"context"
	"strings"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// AssertBalanceGTE asserts that an address holds at least minBalance ulgm.
func AssertBalanceGTE(t *testing.T, client agentsdk.QueryClient, addr string, minBalance uint64) {
	t.Helper()
	ctx := context.Background()
	bal, err := client.Balance(ctx, addr)
	if err != nil {
		t.Fatalf("querying balance for %s: %v", addr, err)
	}
	if bal < minBalance {
		t.Errorf("balance too low: %s has %d ulgm, want >= %d", addr, bal, minBalance)
	}
}

// AssertBalanceDelta asserts that an address's balance changed by expectedDelta
// compared to a previously captured balance.
func AssertBalanceDelta(t *testing.T, client agentsdk.QueryClient, addr string, before uint64, expectedDelta int64) {
	t.Helper()
	ctx := context.Background()
	after, err := client.Balance(ctx, addr)
	if err != nil {
		t.Fatalf("querying balance for %s: %v", addr, err)
	}
	actual := int64(after) - int64(before)
	if actual != expectedDelta {
		t.Errorf("balance delta for %s: got %d, want %d (before=%d, after=%d)",
			addr, actual, expectedDelta, before, after)
	}
}

// AssertFactExists asserts that a fact exists in the given domain containing claimSubstring.
func AssertFactExists(t *testing.T, client agentsdk.QueryClient, domain, claimSubstring string) *agentsdk.FactInfo {
	t.Helper()
	ctx := context.Background()
	facts, err := client.FactsByDomain(ctx, domain)
	if err != nil {
		t.Fatalf("querying facts for domain %s: %v", domain, err)
	}
	for i, f := range facts {
		if strings.Contains(f.ClaimText, claimSubstring) {
			return &facts[i]
		}
	}
	t.Errorf("no fact found in domain %q containing %q (found %d facts)", domain, claimSubstring, len(facts))
	return nil
}

// AssertRoundCompleted asserts that a round completed with the expected verdict.
func AssertRoundCompleted(t *testing.T, client agentsdk.QueryClient, roundID, expectedVerdict string) {
	t.Helper()
	ctx := context.Background()
	detail, err := client.RoundDetail(ctx, roundID)
	if err != nil {
		t.Fatalf("querying round %s: %v", roundID, err)
	}
	if detail.Phase != "complete" {
		t.Errorf("round %s phase: got %s, want complete", roundID, detail.Phase)
	}
	_ = expectedVerdict
}
