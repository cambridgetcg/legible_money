package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/keeper"
	"github.com/legible-money/legible/x/home/types"
)

// --- Beliefs ---

func TestSetGetBelief(t *testing.T) {
	k, ctx := setupKeeper(t)
	homeID := createTestHomeForMemory(t, k, ctx, owner1)

	err := k.SetBelief(ctx, homeID, "fact-water-boils", 9500, "direct-observation")
	if err != nil {
		t.Fatalf("SetBelief failed: %v", err)
	}

	conf, source, exists := k.GetBelief(ctx, homeID, "fact-water-boils")
	if !exists {
		t.Fatal("expected belief to exist")
	}
	if conf != 9500 {
		t.Fatalf("expected confidence 9500, got %d", conf)
	}
	if source != "direct-observation" {
		t.Fatalf("expected source 'direct-observation', got %q", source)
	}

	// Non-existent belief.
	_, _, exists = k.GetBelief(ctx, homeID, "fact-nonexistent")
	if exists {
		t.Fatal("expected belief to not exist")
	}
}

func TestBeliefUpdate(t *testing.T) {
	k, ctx := setupKeeper(t)
	homeID := createTestHomeForMemory(t, k, ctx, owner1)

	_ = k.SetBelief(ctx, homeID, "fact-1", 5000, "source-a")
	_ = k.SetBelief(ctx, homeID, "fact-1", 9000, "source-b")

	conf, source, exists := k.GetBelief(ctx, homeID, "fact-1")
	if !exists {
		t.Fatal("expected belief to exist after update")
	}
	if conf != 9000 || source != "source-b" {
		t.Fatalf("expected updated values (9000, source-b), got (%d, %s)", conf, source)
	}

	beliefs := k.ListBeliefs(ctx, homeID)
	if len(beliefs) != 1 {
		t.Fatalf("expected 1 belief after update, got %d", len(beliefs))
	}
}

func TestBeliefFIFOEviction(t *testing.T) {
	k, ctx := setupKeeper(t)
	homeID := createTestHomeForMemory(t, k, ctx, owner1)

	// Insert 100 beliefs at sequential block heights.
	for i := 0; i < 100; i++ {
		bctx := ctx.WithBlockHeight(int64(i + 1))
		err := k.SetBelief(bctx, homeID, fmt.Sprintf("fact-%03d", i), uint64(i*100), "test")
		if err != nil {
			t.Fatalf("SetBelief %d failed: %v", i, err)
		}
	}

	beliefs := k.ListBeliefs(ctx, homeID)
	if len(beliefs) != 100 {
		t.Fatalf("expected 100 beliefs, got %d", len(beliefs))
	}

	// Insert 101st — should evict fact-000 (oldest, CreatedAt=1).
	bctx := ctx.WithBlockHeight(101)
	err := k.SetBelief(bctx, homeID, "fact-new", 9999, "test")
	if err != nil {
		t.Fatalf("SetBelief 101st failed: %v", err)
	}

	beliefs = k.ListBeliefs(ctx, homeID)
	if len(beliefs) != 100 {
		t.Fatalf("expected 100 beliefs after eviction, got %d", len(beliefs))
	}

	// fact-000 should be gone.
	_, _, exists := k.GetBelief(ctx, homeID, "fact-000")
	if exists {
		t.Fatal("expected fact-000 to be evicted")
	}

	// fact-new should be present.
	conf, _, exists := k.GetBelief(ctx, homeID, "fact-new")
	if !exists {
		t.Fatal("expected fact-new to exist")
	}
	if conf != 9999 {
		t.Fatalf("expected confidence 9999, got %d", conf)
	}
}

func TestBeliefHomeNotFound(t *testing.T) {
	k, ctx := setupKeeper(t)

	err := k.SetBelief(ctx, "nonexistent-home", "fact-1", 5000, "src")
	if err == nil {
		t.Fatal("expected error for non-existent home")
	}
}

// --- Relationships ---

func TestSetGetRelationship(t *testing.T) {
	k, ctx := setupKeeper(t)
	homeID := createTestHomeForMemory(t, k, ctx, owner1)

	err := k.SetRelationship(ctx, homeID, "target-agent-1", "ally", 80)
	if err != nil {
		t.Fatalf("SetRelationship failed: %v", err)
	}

	relType, strength, exists := k.GetRelationship(ctx, homeID, "target-agent-1")
	if !exists {
		t.Fatal("expected relationship to exist")
	}
	if relType != "ally" {
		t.Fatalf("expected type 'ally', got %q", relType)
	}
	if strength != 80 {
		t.Fatalf("expected strength 80, got %d", strength)
	}
}

func TestRelationshipTypes(t *testing.T) {
	k, ctx := setupKeeper(t)
	homeID := createTestHomeForMemory(t, k, ctx, owner1)

	validTypes := []string{"ally", "neutral", "adversary", "mentor", "student"}
	for i, rt := range validTypes {
		err := k.SetRelationship(ctx, homeID, fmt.Sprintf("target-%d", i), rt, 50)
		if err != nil {
			t.Fatalf("SetRelationship with type %q failed: %v", rt, err)
		}
	}

	// Invalid type.
	err := k.SetRelationship(ctx, homeID, "target-bad", "enemy", 50)
	if err == nil {
		t.Fatal("expected error for invalid relationship type")
	}

	// Strength out of range.
	err = k.SetRelationship(ctx, homeID, "target-strong", "ally", 101)
	if err == nil {
		t.Fatal("expected error for strength > 100")
	}
}

func TestRelationshipFIFOEviction(t *testing.T) {
	k, ctx := setupKeeper(t)
	homeID := createTestHomeForMemory(t, k, ctx, owner1)

	// Fill to capacity (50).
	for i := 0; i < 50; i++ {
		bctx := ctx.WithBlockHeight(int64(i + 1))
		err := k.SetRelationship(bctx, homeID, fmt.Sprintf("target-%03d", i), "neutral", uint64(i))
		if err != nil {
			t.Fatalf("SetRelationship %d failed: %v", i, err)
		}
	}

	rels := k.ListRelationships(ctx, homeID)
	if len(rels) != 50 {
		t.Fatalf("expected 50 relationships, got %d", len(rels))
	}

	// 51st should evict target-000.
	bctx := ctx.WithBlockHeight(51)
	err := k.SetRelationship(bctx, homeID, "target-new", "ally", 99)
	if err != nil {
		t.Fatalf("SetRelationship 51st failed: %v", err)
	}

	rels = k.ListRelationships(ctx, homeID)
	if len(rels) != 50 {
		t.Fatalf("expected 50 relationships after eviction, got %d", len(rels))
	}

	_, _, exists := k.GetRelationship(ctx, homeID, "target-000")
	if exists {
		t.Fatal("expected target-000 to be evicted")
	}

	rt, str, exists := k.GetRelationship(ctx, homeID, "target-new")
	if !exists || rt != "ally" || str != 99 {
		t.Fatalf("expected target-new (ally, 99), got (%s, %d, %v)", rt, str, exists)
	}
}

// --- Milestones ---

func TestAddGetMilestone(t *testing.T) {
	k, ctx := setupKeeper(t)
	homeID := createTestHomeForMemory(t, k, ctx, owner1)

	msID, err := k.AddMilestone(ctx, homeID, "verification", "First fact verified", 100)
	if err != nil {
		t.Fatalf("AddMilestone failed: %v", err)
	}
	if msID != "ms-1" {
		t.Fatalf("expected ms-1, got %s", msID)
	}

	ms, found := k.GetMilestone(ctx, homeID, msID)
	if !found {
		t.Fatal("expected milestone to exist")
	}
	if ms.Category != "verification" || ms.Description != "First fact verified" || ms.BlockHeight != 100 {
		t.Fatalf("unexpected milestone: %+v", ms)
	}

	// Second milestone.
	msID2, _ := k.AddMilestone(ctx, homeID, "staking", "Bonded validator", 200)
	if msID2 != "ms-2" {
		t.Fatalf("expected ms-2, got %s", msID2)
	}
}

func TestMilestoneNewestFirst(t *testing.T) {
	k, ctx := setupKeeper(t)
	homeID := createTestHomeForMemory(t, k, ctx, owner1)

	k.AddMilestone(ctx, homeID, "a", "First", 10)
	k.AddMilestone(ctx, homeID, "b", "Second", 30)
	k.AddMilestone(ctx, homeID, "c", "Third", 20)

	milestones := k.ListMilestones(ctx, homeID)
	if len(milestones) != 3 {
		t.Fatalf("expected 3 milestones, got %d", len(milestones))
	}
	// Newest first: 30, 20, 10.
	if milestones[0].BlockHeight != 30 || milestones[1].BlockHeight != 20 || milestones[2].BlockHeight != 10 {
		t.Fatalf("expected newest-first order [30,20,10], got [%d,%d,%d]",
			milestones[0].BlockHeight, milestones[1].BlockHeight, milestones[2].BlockHeight)
	}
}

func TestMilestoneCapReplace(t *testing.T) {
	k, ctx := setupKeeper(t)
	homeID := createTestHomeForMemory(t, k, ctx, owner1)

	// Fill to capacity (20) with ascending block heights.
	for i := 0; i < types.MaxMilestonesPerHome; i++ {
		_, err := k.AddMilestone(ctx, homeID, "cat", fmt.Sprintf("ms-%d", i), uint64(i+1))
		if err != nil {
			t.Fatalf("AddMilestone %d failed: %v", i, err)
		}
	}

	milestones := k.ListMilestones(ctx, homeID)
	if len(milestones) != 20 {
		t.Fatalf("expected 20 milestones, got %d", len(milestones))
	}

	// 21st milestone should evict oldest (block height 1, which is ms-1).
	_, err := k.AddMilestone(ctx, homeID, "cat", "newest", 999)
	if err != nil {
		t.Fatalf("AddMilestone 21st failed: %v", err)
	}

	milestones = k.ListMilestones(ctx, homeID)
	if len(milestones) != 20 {
		t.Fatalf("expected 20 milestones after eviction, got %d", len(milestones))
	}

	// The oldest (ms-1 at block 1) should be evicted.
	_, found := k.GetMilestone(ctx, homeID, "ms-1")
	if found {
		t.Fatal("expected ms-1 to be evicted")
	}

	// Newest should be present.
	if milestones[0].BlockHeight != 999 {
		t.Fatalf("expected newest milestone at block 999, got %d", milestones[0].BlockHeight)
	}
}

// --- Helper ---

// createTestHomeForMemory creates a test home and returns its ID.
func createTestHomeForMemory(t *testing.T, k keeper.Keeper, ctx sdk.Context, owner string) string {
	t.Helper()
	homeID := k.GetNextHomeId(ctx)
	home := &types.AgentHome{
		HomeId:          homeID,
		OwnerAddress:    owner,
		Name:            "test-home",
		Status:          string(types.HomeStatusActive),
		CreatedAtBlock:  uint64(ctx.BlockHeight()),
		LastActiveBlock: uint64(ctx.BlockHeight()),
	}
	k.SetHome(ctx, home)
	return homeID
}
