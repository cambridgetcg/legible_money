package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/billing/types"
)

// -----------------------------------------------------------------------
// Tests: Subscription CRUD
// -----------------------------------------------------------------------

func TestSubscriptionSetGet(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	sub := &types.SubscriptionRecord{
		ProjectID: "proj-001",
		Plan:      types.PlanSeed,
		Status:    types.StatusActive,
		PeriodEnd: 50000,
		CreatedAt: 100,
	}
	k.SetSubscription(ctx, "proj-001", sub)

	got, found := k.GetSubscription(ctx, "proj-001")
	if !found {
		t.Fatal("expected subscription to be found")
	}
	if got.Plan != types.PlanSeed {
		t.Errorf("expected plan seed, got %s", got.Plan)
	}
	if got.Status != types.StatusActive {
		t.Errorf("expected status active, got %s", got.Status)
	}
	if got.PeriodEnd != 50000 {
		t.Errorf("expected period_end 50000, got %d", got.PeriodEnd)
	}
	if got.CreatedAt != 100 {
		t.Errorf("expected created_at 100, got %d", got.CreatedAt)
	}
}

func TestSubscriptionNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	_, found := k.GetSubscription(ctx, "nonexistent")
	if found {
		t.Error("expected subscription not found")
	}
}

func TestSubscriptionDelete(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	sub := &types.SubscriptionRecord{
		ProjectID: "proj-del",
		Plan:      types.PlanGrow,
		Status:    types.StatusActive,
		PeriodEnd: 60000,
		CreatedAt: 200,
	}
	k.SetSubscription(ctx, "proj-del", sub)

	// Verify exists
	_, found := k.GetSubscription(ctx, "proj-del")
	if !found {
		t.Fatal("expected subscription to exist before delete")
	}

	k.DeleteSubscription(ctx, "proj-del")

	_, found = k.GetSubscription(ctx, "proj-del")
	if found {
		t.Error("expected subscription to be deleted")
	}
}

func TestSubscriptionUpdate(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	sub := &types.SubscriptionRecord{
		ProjectID: "proj-upd",
		Plan:      types.PlanFree,
		Status:    types.StatusActive,
		PeriodEnd: 10000,
		CreatedAt: 50,
	}
	k.SetSubscription(ctx, "proj-upd", sub)

	// Upgrade plan
	sub.Plan = types.PlanScale
	sub.PeriodEnd = 80000
	k.SetSubscription(ctx, "proj-upd", sub)

	got, found := k.GetSubscription(ctx, "proj-upd")
	if !found {
		t.Fatal("expected subscription after update")
	}
	if got.Plan != types.PlanScale {
		t.Errorf("expected plan scale after update, got %s", got.Plan)
	}
	if got.PeriodEnd != 80000 {
		t.Errorf("expected period_end 80000 after update, got %d", got.PeriodEnd)
	}
}

// -----------------------------------------------------------------------
// Tests: Usage Increment + Limits
// -----------------------------------------------------------------------

func TestUsageIncrementFreePlan(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// No subscription set → defaults to free plan
	// Free: memory_ops = 100/day
	for i := 0; i < 100; i++ {
		used, limit, allowed := k.IncrementUsage(ctx, "proj-free", types.ResourceMemoryOps)
		if !allowed {
			t.Fatalf("expected allowed at usage %d (limit %d)", used, limit)
		}
	}

	// 101st should be denied
	used, limit, allowed := k.IncrementUsage(ctx, "proj-free", types.ResourceMemoryOps)
	if allowed {
		t.Errorf("expected denied at usage %d, limit %d", used, limit)
	}
	if limit != 100 {
		t.Errorf("expected limit 100, got %d", limit)
	}
	// Counter should not have incremented past limit
	if used != 100 {
		t.Errorf("expected used=100 (unchanged), got %d", used)
	}
}

func TestUsageIncrementSeedPlan(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	sub := &types.SubscriptionRecord{
		ProjectID: "proj-seed",
		Plan:      types.PlanSeed,
		Status:    types.StatusActive,
		PeriodEnd: 99999,
		CreatedAt: 1,
	}
	k.SetSubscription(ctx, "proj-seed", sub)

	// Seed: tool_calls = 500/day — increment 500 times
	for i := 0; i < 500; i++ {
		_, _, allowed := k.IncrementUsage(ctx, "proj-seed", types.ResourceToolCalls)
		if !allowed {
			t.Fatalf("expected allowed at iteration %d", i)
		}
	}

	// 501st denied
	_, limit, allowed := k.IncrementUsage(ctx, "proj-seed", types.ResourceToolCalls)
	if allowed {
		t.Error("expected denied at 501")
	}
	if limit != 500 {
		t.Errorf("expected limit 500, got %d", limit)
	}
}

func TestUsageScalePlanUnlimited(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	sub := &types.SubscriptionRecord{
		ProjectID: "proj-scale",
		Plan:      types.PlanScale,
		Status:    types.StatusActive,
		PeriodEnd: 99999,
		CreatedAt: 1,
	}
	k.SetSubscription(ctx, "proj-scale", sub)

	// Scale: unlimited (-1) — should always be allowed
	for i := 0; i < 1000; i++ {
		_, limit, allowed := k.IncrementUsage(ctx, "proj-scale", types.ResourceMemoryOps)
		if !allowed {
			t.Fatalf("scale plan should be unlimited, denied at iteration %d", i)
		}
		if limit != -1 {
			t.Fatalf("expected limit -1 (unlimited), got %d", limit)
		}
	}
}

func TestUsageCancelledFallsBackToFree(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	sub := &types.SubscriptionRecord{
		ProjectID: "proj-cancel",
		Plan:      types.PlanScale,
		Status:    types.StatusCancelled,
		PeriodEnd: 99999,
		CreatedAt: 1,
	}
	k.SetSubscription(ctx, "proj-cancel", sub)

	// Cancelled scale plan → free limits: verifications = 5/day
	for i := 0; i < 5; i++ {
		_, _, allowed := k.IncrementUsage(ctx, "proj-cancel", types.ResourceVerifications)
		if !allowed {
			t.Fatalf("expected allowed at iteration %d", i)
		}
	}

	_, limit, allowed := k.IncrementUsage(ctx, "proj-cancel", types.ResourceVerifications)
	if allowed {
		t.Error("cancelled subscription should fall back to free limits")
	}
	if limit != 5 {
		t.Errorf("expected free-tier verification limit 5, got %d", limit)
	}
}

// -----------------------------------------------------------------------
// Tests: GetUsage
// -----------------------------------------------------------------------

func TestGetUsage(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Empty initially
	usage := k.GetUsage(ctx, "proj-empty")
	if len(usage) != 0 {
		t.Errorf("expected empty usage map, got %d entries", len(usage))
	}

	// Increment some
	k.IncrementUsage(ctx, "proj-usage", types.ResourceMemoryOps)
	k.IncrementUsage(ctx, "proj-usage", types.ResourceMemoryOps)
	k.IncrementUsage(ctx, "proj-usage", types.ResourceToolCalls)

	usage = k.GetUsage(ctx, "proj-usage")
	if usage[types.ResourceMemoryOps] != 2 {
		t.Errorf("expected memory_ops=2, got %d", usage[types.ResourceMemoryOps])
	}
	if usage[types.ResourceToolCalls] != 1 {
		t.Errorf("expected tool_calls=1, got %d", usage[types.ResourceToolCalls])
	}
}

// -----------------------------------------------------------------------
// Tests: Usage Reset
// -----------------------------------------------------------------------

func TestResetUsage(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Increment
	k.IncrementUsage(ctx, "proj-reset", types.ResourceMemoryOps)
	k.IncrementUsage(ctx, "proj-reset", types.ResourceToolCalls)

	usage := k.GetUsage(ctx, "proj-reset")
	if usage[types.ResourceMemoryOps] != 1 {
		t.Fatalf("expected 1, got %d", usage[types.ResourceMemoryOps])
	}

	// Reset
	k.ResetUsage(ctx, "proj-reset")

	usage = k.GetUsage(ctx, "proj-reset")
	if usage[types.ResourceMemoryOps] != 0 {
		t.Errorf("expected 0 after reset, got %d", usage[types.ResourceMemoryOps])
	}
	if usage[types.ResourceToolCalls] != 0 {
		t.Errorf("expected 0 after reset, got %d", usage[types.ResourceToolCalls])
	}

	// Should be able to increment again after reset
	_, _, allowed := k.IncrementUsage(ctx, "proj-reset", types.ResourceMemoryOps)
	if !allowed {
		t.Error("expected allowed after reset")
	}
}

// -----------------------------------------------------------------------
// Tests: Plan Limits Validation
// -----------------------------------------------------------------------

func TestPlanLimitsDefaults(t *testing.T) {
	limits := types.DefaultPlanLimits()

	tests := []struct {
		plan     string
		resource string
		expected int64
	}{
		{types.PlanFree, types.ResourceMemoryOps, 100},
		{types.PlanFree, types.ResourceToolCalls, 10},
		{types.PlanFree, types.ResourceVerifications, 5},
		{types.PlanSeed, types.ResourceMemoryOps, 10_000},
		{types.PlanSeed, types.ResourceToolCalls, 500},
		{types.PlanSeed, types.ResourceVerifications, 100},
		{types.PlanGrow, types.ResourceMemoryOps, 100_000},
		{types.PlanGrow, types.ResourceToolCalls, 5_000},
		{types.PlanGrow, types.ResourceVerifications, 1_000},
		{types.PlanScale, types.ResourceMemoryOps, -1},
		{types.PlanScale, types.ResourceToolCalls, -1},
		{types.PlanScale, types.ResourceVerifications, -1},
	}

	for _, tc := range tests {
		limit := limits[tc.plan].GetLimit(tc.resource)
		if limit != tc.expected {
			t.Errorf("%s/%s: expected %d, got %d", tc.plan, tc.resource, tc.expected, limit)
		}
	}
}

// -----------------------------------------------------------------------
// Tests: SubscriptionRecord Validate
// -----------------------------------------------------------------------

func TestSubscriptionRecordValidate(t *testing.T) {
	tests := []struct {
		name    string
		sub     types.SubscriptionRecord
		wantErr bool
	}{
		{"valid", types.SubscriptionRecord{ProjectID: "p1", Plan: types.PlanSeed, Status: types.StatusActive}, false},
		{"empty project", types.SubscriptionRecord{ProjectID: "", Plan: types.PlanFree, Status: types.StatusActive}, true},
		{"bad plan", types.SubscriptionRecord{ProjectID: "p1", Plan: "enterprise", Status: types.StatusActive}, true},
		{"bad status", types.SubscriptionRecord{ProjectID: "p1", Plan: types.PlanFree, Status: "expired"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.sub.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests: BeginBlock Daily Reset
// -----------------------------------------------------------------------

func TestBeginBlockDailyReset(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Create two active subscriptions and one cancelled
	k.SetSubscription(ctx, "proj-a", &types.SubscriptionRecord{
		ProjectID: "proj-a", Plan: types.PlanSeed, Status: types.StatusActive,
		PeriodEnd: 999999, CreatedAt: 1,
	})
	k.SetSubscription(ctx, "proj-b", &types.SubscriptionRecord{
		ProjectID: "proj-b", Plan: types.PlanGrow, Status: types.StatusActive,
		PeriodEnd: 999999, CreatedAt: 1,
	})
	k.SetSubscription(ctx, "proj-c", &types.SubscriptionRecord{
		ProjectID: "proj-c", Plan: types.PlanFree, Status: types.StatusCancelled,
		PeriodEnd: 999999, CreatedAt: 1,
	})

	// Increment usage for all three
	k.IncrementUsage(ctx, "proj-a", types.ResourceMemoryOps)
	k.IncrementUsage(ctx, "proj-b", types.ResourceToolCalls)
	k.IncrementUsage(ctx, "proj-c", types.ResourceVerifications)

	// Verify usage exists
	if k.GetUsage(ctx, "proj-a")[types.ResourceMemoryOps] != 1 {
		t.Fatal("expected proj-a usage=1 before reset")
	}

	// ResetAllActiveUsage only resets active subs
	count := k.ResetAllActiveUsage(ctx)
	if count != 2 {
		t.Errorf("expected 2 active subscriptions reset, got %d", count)
	}

	// Active subs should be reset
	if k.GetUsage(ctx, "proj-a")[types.ResourceMemoryOps] != 0 {
		t.Error("expected proj-a usage=0 after reset")
	}
	if k.GetUsage(ctx, "proj-b")[types.ResourceToolCalls] != 0 {
		t.Error("expected proj-b usage=0 after reset")
	}

	// Cancelled sub should still have usage (not reset)
	if k.GetUsage(ctx, "proj-c")[types.ResourceVerifications] != 1 {
		t.Error("expected proj-c usage=1 (cancelled, not reset)")
	}
}

func TestLastResetBlock(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Default is 0
	if k.GetLastResetBlock(ctx) != 0 {
		t.Errorf("expected 0 initially, got %d", k.GetLastResetBlock(ctx))
	}

	k.SetLastResetBlock(ctx, 34272)
	if k.GetLastResetBlock(ctx) != 34272 {
		t.Errorf("expected 34272, got %d", k.GetLastResetBlock(ctx))
	}
}
