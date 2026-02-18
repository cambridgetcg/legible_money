package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/knowledge/types"
)

// ---------- MarkFactChallenged Tests ----------

func TestMarkFactChallenged_Verified(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id:         "fact-challenge-1",
		ClaimText:  "Test claim",
		Domain:     "physics",
		Status:     "verified",
		Confidence: 800_000,
		Submitter:  testSubmitterAddr,
	}
	k.SetFact(ctx, &fact)

	err := k.MarkFactChallenged(ctx, "fact-challenge-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, found := k.GetFact(ctx, "fact-challenge-1")
	if !found {
		t.Fatal("fact not found after challenge")
	}
	if updated.Status != "challenged" {
		t.Fatalf("expected status 'challenged', got '%s'", updated.Status)
	}
}

func TestMarkFactChallenged_Active(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id:     "fact-challenge-2",
		Domain: "physics",
		Status: "active",
	}
	k.SetFact(ctx, &fact)

	err := k.MarkFactChallenged(ctx, "fact-challenge-2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := k.GetFact(ctx, "fact-challenge-2")
	if updated.Status != "challenged" {
		t.Fatalf("expected status 'challenged', got '%s'", updated.Status)
	}
}

func TestMarkFactChallenged_InvalidStatus(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id:     "fact-challenge-3",
		Domain: "physics",
		Status: "contested",
	}
	k.SetFact(ctx, &fact)

	err := k.MarkFactChallenged(ctx, "fact-challenge-3")
	if err == nil {
		t.Fatal("expected error for contested fact, got nil")
	}
}

func TestMarkFactChallenged_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	err := k.MarkFactChallenged(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent fact, got nil")
	}
}

// ---------- ResolveFactDispute Tests ----------

func TestResolveFactDispute_Overturned(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id:         "fact-resolve-1",
		Domain:     "physics",
		Status:     "challenged",
		Confidence: 800_000,
	}
	k.SetFact(ctx, &fact)

	newConf, err := k.ResolveFactDispute(ctx, "fact-resolve-1", "overturned", 550_000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 800_000 * (1_000_000 - 550_000) / 1_000_000 = 360_000
	expectedConf := uint64(800_000) * (1_000_000 - 550_000) / 1_000_000
	if newConf != expectedConf {
		t.Fatalf("expected confidence %d, got %d", expectedConf, newConf)
	}

	updated, _ := k.GetFact(ctx, "fact-resolve-1")
	if updated.Status != "contested" {
		t.Fatalf("expected status 'contested', got '%s'", updated.Status)
	}
	if updated.ChallengeCount != 1 {
		t.Fatalf("expected challenge count 1, got %d", updated.ChallengeCount)
	}
}

func TestResolveFactDispute_OverturnedRevoked(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id:         "fact-resolve-2",
		Domain:     "physics",
		Status:     "challenged",
		Confidence: 150_000, // Low confidence
	}
	k.SetFact(ctx, &fact)

	newConf, err := k.ResolveFactDispute(ctx, "fact-resolve-2", "overturned", 550_000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 150_000 * 450_000 / 1_000_000 = 67_500 < 100_000 threshold
	expectedConf := uint64(150_000) * (1_000_000 - 550_000) / 1_000_000
	if newConf != expectedConf {
		t.Fatalf("expected confidence %d, got %d", expectedConf, newConf)
	}

	updated, _ := k.GetFact(ctx, "fact-resolve-2")
	if updated.Status != "revoked" {
		t.Fatalf("expected status 'revoked', got '%s'", updated.Status)
	}
}

func TestResolveFactDispute_Upheld(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id:         "fact-resolve-3",
		Domain:     "physics",
		Status:     "challenged",
		Confidence: 800_000,
	}
	k.SetFact(ctx, &fact)

	newConf, err := k.ResolveFactDispute(ctx, "fact-resolve-3", "upheld", 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 800_000 * 1_110_000 / 1_000_000 = 888_000
	expectedConf := uint64(800_000) * (1_000_000 + 110_000) / 1_000_000
	if newConf != expectedConf {
		t.Fatalf("expected confidence %d, got %d", expectedConf, newConf)
	}

	updated, _ := k.GetFact(ctx, "fact-resolve-3")
	if updated.Status != "active" {
		t.Fatalf("expected status 'active', got '%s'", updated.Status)
	}
}

func TestResolveFactDispute_UpheldCapped(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id:         "fact-resolve-4",
		Domain:     "physics",
		Status:     "contested",
		Confidence: 950_000,
	}
	k.SetFact(ctx, &fact)

	newConf, err := k.ResolveFactDispute(ctx, "fact-resolve-4", "upheld", 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 950_000 * 1_110_000 / 1_000_000 = 1_054_500 → capped at 1_000_000
	if newConf != 1_000_000 {
		t.Fatalf("expected confidence capped at 1_000_000, got %d", newConf)
	}
}

func TestResolveFactDispute_Inconclusive(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	fact := types.Fact{
		Id:         "fact-resolve-5",
		Domain:     "physics",
		Status:     "challenged",
		Confidence: 800_000,
	}
	k.SetFact(ctx, &fact)

	newConf, err := k.ResolveFactDispute(ctx, "fact-resolve-5", "inconclusive", 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// No confidence change
	if newConf != 800_000 {
		t.Fatalf("expected confidence unchanged at 800_000, got %d", newConf)
	}

	updated, _ := k.GetFact(ctx, "fact-resolve-5")
	if updated.Status != "contested" {
		t.Fatalf("expected status 'contested' (was challenged), got '%s'", updated.Status)
	}
}

func TestResolveFactDispute_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	_, err := k.ResolveFactDispute(ctx, "nonexistent", "upheld", 0)
	if err == nil {
		t.Fatal("expected error for nonexistent fact")
	}
}

// ---------- CascadeFactConfidence Tests ----------

func TestCascadeFactConfidence_MultiFact(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Create the retracted fact
	retracted := types.Fact{
		Id:         "fact-retracted",
		Domain:     "physics",
		Status:     "revoked",
		Confidence: 50_000,
	}
	k.SetFact(ctx, &retracted)

	// Create two facts that reference the retracted one
	referrer1 := types.Fact{
		Id:         "fact-ref-1",
		Domain:     "physics",
		Status:     "active",
		Confidence: 700_000,
	}
	k.SetFact(ctx, &referrer1)

	referrer2 := types.Fact{
		Id:         "fact-ref-2",
		Domain:     "physics",
		Status:     "active",
		Confidence: 500_000,
	}
	k.SetFact(ctx, &referrer2)

	// Set up references: referrer1 → retracted, referrer2 → retracted
	k.SetFactReference(ctx, &types.FactReference{
		FromFactId:    "fact-ref-1",
		ToFactId:      "fact-retracted",
		ReferenceType: "supports",
	})
	k.SetFactReference(ctx, &types.FactReference{
		FromFactId:    "fact-ref-2",
		ToFactId:      "fact-retracted",
		ReferenceType: "supports",
	})

	affected, err := k.CascadeFactConfidence(ctx, "fact-retracted", 220_000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(affected) != 2 {
		t.Fatalf("expected 2 affected facts, got %d", len(affected))
	}

	// referrer1: 700_000 * (1_000_000 - 220_000) / 1_000_000 = 546_000
	r1, _ := k.GetFact(ctx, "fact-ref-1")
	expectedConf1 := uint64(700_000) * (1_000_000 - 220_000) / 1_000_000
	if r1.Confidence != expectedConf1 {
		t.Fatalf("referrer1: expected confidence %d, got %d", expectedConf1, r1.Confidence)
	}
	if r1.Status != "contested" {
		t.Fatalf("referrer1: expected status 'contested', got '%s'", r1.Status)
	}

	// referrer2: 500_000 * 780_000 / 1_000_000 = 390_000
	r2, _ := k.GetFact(ctx, "fact-ref-2")
	expectedConf2 := uint64(500_000) * (1_000_000 - 220_000) / 1_000_000
	if r2.Confidence != expectedConf2 {
		t.Fatalf("referrer2: expected confidence %d, got %d", expectedConf2, r2.Confidence)
	}
}

func TestCascadeFactConfidence_SkipRevoked(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	retracted := types.Fact{
		Id:     "fact-retracted-2",
		Domain: "physics",
		Status: "revoked",
	}
	k.SetFact(ctx, &retracted)

	alreadyRevoked := types.Fact{
		Id:         "fact-already-revoked",
		Domain:     "physics",
		Status:     "revoked",
		Confidence: 50_000,
	}
	k.SetFact(ctx, &alreadyRevoked)

	k.SetFactReference(ctx, &types.FactReference{
		FromFactId:    "fact-already-revoked",
		ToFactId:      "fact-retracted-2",
		ReferenceType: "supports",
	})

	affected, err := k.CascadeFactConfidence(ctx, "fact-retracted-2", 220_000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(affected) != 0 {
		t.Fatalf("expected 0 affected facts (all revoked), got %d", len(affected))
	}
}

func TestCascadeFactConfidence_NoReferences(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	retracted := types.Fact{
		Id:     "fact-no-refs",
		Domain: "physics",
		Status: "revoked",
	}
	k.SetFact(ctx, &retracted)

	affected, err := k.CascadeFactConfidence(ctx, "fact-no-refs", 220_000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if affected != nil {
		t.Fatalf("expected nil for no references, got %v", affected)
	}
}
