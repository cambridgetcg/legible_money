package keeper_test

import (
	"fmt"
	"testing"

	"github.com/legible-money/legible/x/home/types"
)

func TestCreateNeighborhood(t *testing.T) {
	k, ctx := setupKeeper(t)
	founderID := createTestHomeForMemory(t, k, ctx, owner1)

	nbrID, err := k.CreateNeighborhood(ctx, founderID, "physics")
	if err != nil {
		t.Fatalf("CreateNeighborhood failed: %v", err)
	}
	if nbrID != "nbr-1" {
		t.Fatalf("expected nbr-1, got %s", nbrID)
	}

	nbr, found := k.GetNeighborhood(ctx, nbrID)
	if !found {
		t.Fatal("expected neighborhood to exist")
	}
	if nbr.FounderHomeID != founderID {
		t.Fatalf("expected founder %s, got %s", founderID, nbr.FounderHomeID)
	}
	if nbr.Domain != "physics" {
		t.Fatalf("expected domain physics, got %s", nbr.Domain)
	}
	if len(nbr.Members) != 1 || nbr.Members[0] != founderID {
		t.Fatalf("expected [%s], got %v", founderID, nbr.Members)
	}

	// Second neighborhood gets incremented ID.
	nbrID2, _ := k.CreateNeighborhood(ctx, founderID, "chemistry")
	if nbrID2 != "nbr-2" {
		t.Fatalf("expected nbr-2, got %s", nbrID2)
	}
}

func TestJoinAndLeaveNeighborhood(t *testing.T) {
	k, ctx := setupKeeper(t)
	h1 := createTestHomeForMemory(t, k, ctx, owner1)
	h2 := createTestHomeForMemory(t, k, ctx, owner2)
	h3 := createTestHomeForMemory(t, k, ctx, user1)

	nbrID, _ := k.CreateNeighborhood(ctx, h1, "math")

	// Join.
	err := k.JoinNeighborhood(ctx, h2, nbrID)
	if err != nil {
		t.Fatalf("JoinNeighborhood failed: %v", err)
	}
	err = k.JoinNeighborhood(ctx, h3, nbrID)
	if err != nil {
		t.Fatalf("JoinNeighborhood h3 failed: %v", err)
	}

	members := k.ListNeighborhoodMembers(ctx, nbrID)
	if len(members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(members))
	}

	// Duplicate join.
	err = k.JoinNeighborhood(ctx, h2, nbrID)
	if err == nil {
		t.Fatal("expected error for duplicate join")
	}

	// Leave.
	err = k.LeaveNeighborhood(ctx, h2, nbrID)
	if err != nil {
		t.Fatalf("LeaveNeighborhood failed: %v", err)
	}

	members = k.ListNeighborhoodMembers(ctx, nbrID)
	if len(members) != 2 {
		t.Fatalf("expected 2 members after leave, got %d", len(members))
	}

	// Leave non-member.
	err = k.LeaveNeighborhood(ctx, h2, nbrID)
	if err == nil {
		t.Fatal("expected error for leaving when not a member")
	}
}

func TestNeighborhoodMaxMembers(t *testing.T) {
	k, ctx := setupKeeper(t)
	founder := createTestHomeForMemory(t, k, ctx, owner1)
	nbrID, _ := k.CreateNeighborhood(ctx, founder, "ai")

	// Fill to max (founder counts as 1, need 21 more).
	for i := 0; i < types.MaxNeighborhoodMembers-1; i++ {
		addr := testAddr(fmt.Sprintf("member%03d", i))
		hID := createTestHomeForMemory(t, k, ctx, addr)
		err := k.JoinNeighborhood(ctx, hID, nbrID)
		if err != nil {
			t.Fatalf("JoinNeighborhood member %d failed: %v", i, err)
		}
	}

	members := k.ListNeighborhoodMembers(ctx, nbrID)
	if len(members) != types.MaxNeighborhoodMembers {
		t.Fatalf("expected %d members, got %d", types.MaxNeighborhoodMembers, len(members))
	}

	// One more should fail.
	extraAddr := testAddr("extra000")
	extraID := createTestHomeForMemory(t, k, ctx, extraAddr)
	err := k.JoinNeighborhood(ctx, extraID, nbrID)
	if err == nil {
		t.Fatal("expected error when neighborhood is full")
	}
}

func TestNeighborhoodBonus(t *testing.T) {
	k, ctx := setupKeeper(t)
	h1 := createTestHomeForMemory(t, k, ctx, owner1)
	h2 := createTestHomeForMemory(t, k, ctx, owner2)
	h3 := createTestHomeForMemory(t, k, ctx, user1)

	// No neighborhood — bonus is 0.
	bonus := k.NeighborhoodBonus(ctx, h1)
	if bonus != 0 {
		t.Fatalf("expected 0 bonus with no neighborhood, got %d", bonus)
	}

	// Create neighborhood with only 1 member (founder).
	nbrID, _ := k.CreateNeighborhood(ctx, h1, "bio")
	bonus = k.NeighborhoodBonus(ctx, h1)
	if bonus != 0 {
		t.Fatalf("expected 0 bonus with 1 member, got %d", bonus)
	}

	// Add second member — still < MinNeighborhoodMembers (3).
	k.JoinNeighborhood(ctx, h2, nbrID)
	bonus = k.NeighborhoodBonus(ctx, h1)
	if bonus != 0 {
		t.Fatalf("expected 0 bonus with 2 members, got %d", bonus)
	}

	// Add third — now >= 3, bonus kicks in.
	k.JoinNeighborhood(ctx, h3, nbrID)
	bonus = k.NeighborhoodBonus(ctx, h1)
	if bonus != types.NeighborhoodBonusBps {
		t.Fatalf("expected %d bonus with 3 members, got %d", types.NeighborhoodBonusBps, bonus)
	}

	// All members get the bonus.
	bonus2 := k.NeighborhoodBonus(ctx, h2)
	bonus3 := k.NeighborhoodBonus(ctx, h3)
	if bonus2 != types.NeighborhoodBonusBps || bonus3 != types.NeighborhoodBonusBps {
		t.Fatalf("expected all members to get bonus, got h2=%d h3=%d", bonus2, bonus3)
	}
}

func TestCreateNeighborhoodValidation(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Non-existent home.
	_, err := k.CreateNeighborhood(ctx, "nonexistent", "physics")
	if err == nil {
		t.Fatal("expected error for non-existent founder")
	}

	// Empty domain.
	founderID := createTestHomeForMemory(t, k, ctx, owner1)
	_, err = k.CreateNeighborhood(ctx, founderID, "")
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
}

func TestNeighborhoodMembersSorted(t *testing.T) {
	k, ctx := setupKeeper(t)
	h1 := createTestHomeForMemory(t, k, ctx, owner1)
	h2 := createTestHomeForMemory(t, k, ctx, owner2)
	h3 := createTestHomeForMemory(t, k, ctx, user1)

	nbrID, _ := k.CreateNeighborhood(ctx, h1, "logic")
	k.JoinNeighborhood(ctx, h3, nbrID)
	k.JoinNeighborhood(ctx, h2, nbrID)

	members := k.ListNeighborhoodMembers(ctx, nbrID)
	if len(members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(members))
	}

	// Members should be sorted.
	for i := 1; i < len(members); i++ {
		if members[i-1] > members[i] {
			t.Fatalf("members not sorted: %v", members)
		}
	}
}
