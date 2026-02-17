package orchestrator

import (
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	swarm "github.com/legible-money/legible/tools/agent-swarm"
)

func TestNewSection(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 3, agentsdk.RoleVerifier)
	defer cleanup()

	spec := SectionSpec{Name: SectionStrings, MinCount: 2, MaxCount: 5}
	sec, err := NewSection(spec, agents)
	if err != nil {
		t.Fatalf("NewSection: %v", err)
	}

	if sec.Name != SectionStrings {
		t.Errorf("name = %s, want %s", sec.Name, SectionStrings)
	}
	if len(sec.Members) != 3 {
		t.Errorf("members = %d, want 3", len(sec.Members))
	}
	if sec.Leader == nil {
		t.Fatal("leader is nil")
	}
}

func TestNewSection_TooFewAgents(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 1, agentsdk.RoleVerifier)
	defer cleanup()

	spec := SectionSpec{Name: SectionStrings, MinCount: 3, MaxCount: 5}
	_, err := NewSection(spec, agents)
	if err == nil {
		t.Fatal("expected error for too few agents")
	}
}

func TestElectLeader_DeterministicTiebreak(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 3, agentsdk.RoleVerifier)
	defer cleanup()

	// All agents have same rewards (0), so tiebreak by name.
	spec := SectionSpec{Name: SectionStrings, MinCount: 1, MaxCount: 5}
	sec, err := NewSection(spec, agents)
	if err != nil {
		t.Fatalf("NewSection: %v", err)
	}

	// Leader should be deterministic given same rewards.
	leader1 := sec.Leader.Managed.Spec.Name

	sec.ElectLeader()
	leader2 := sec.Leader.Managed.Spec.Name

	if leader1 != leader2 {
		t.Errorf("leader changed between elections: %s → %s", leader1, leader2)
	}
}

func TestElectLeader_HigherRewardsWins(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 3, agentsdk.RoleVerifier)
	defer cleanup()

	// Give the last agent higher rewards.
	agents[2].Metrics.RewardsEarned = 1000

	spec := SectionSpec{Name: SectionStrings, MinCount: 1, MaxCount: 5}
	sec, err := NewSection(spec, agents)
	if err != nil {
		t.Fatalf("NewSection: %v", err)
	}

	if sec.Leader.Managed.Spec.Name != agents[2].Spec.Name {
		t.Errorf("leader = %s, want %s (highest rewards)", sec.Leader.Managed.Spec.Name, agents[2].Spec.Name)
	}
}

func TestAssignPart_LeastLoaded(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 3, agentsdk.RoleVerifier)
	defer cleanup()

	spec := SectionSpec{Name: SectionStrings, MinCount: 1, MaxCount: 5}
	sec, err := NewSection(spec, agents)
	if err != nil {
		t.Fatalf("NewSection: %v", err)
	}

	// Load members[0] and members[1].
	sec.Members[0].ActiveParts = 5
	sec.Members[1].ActiveParts = 3

	member, err := sec.AssignPart("test-part")
	if err != nil {
		t.Fatalf("AssignPart: %v", err)
	}

	if member.ActiveParts != 0 {
		t.Errorf("assigned to member with %d active parts, want 0 (least loaded)", member.ActiveParts)
	}
}

func TestAssignPart_SkipsPaused(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 2, agentsdk.RoleVerifier)
	defer cleanup()

	spec := SectionSpec{Name: SectionStrings, MinCount: 1, MaxCount: 5}
	sec, err := NewSection(spec, agents)
	if err != nil {
		t.Fatalf("NewSection: %v", err)
	}

	sec.Members[0].Paused = true

	member, err := sec.AssignPart("test-part")
	if err != nil {
		t.Fatalf("AssignPart: %v", err)
	}

	if member.Managed.Spec.Name == sec.Members[0].Managed.Spec.Name {
		t.Error("assigned to paused member")
	}
}

func TestAssignPart_NoAvailable(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 2, agentsdk.RoleVerifier)
	defer cleanup()

	spec := SectionSpec{Name: SectionStrings, MinCount: 1, MaxCount: 5}
	sec, err := NewSection(spec, agents)
	if err != nil {
		t.Fatalf("NewSection: %v", err)
	}

	sec.Members[0].Paused = true
	sec.Members[1].Managed.Status = swarm.StatusFailed

	_, err = sec.AssignPart("test-part")
	if err == nil {
		t.Fatal("expected error when no members available")
	}
}

func TestBroadcastPart(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 3, agentsdk.RoleVerifier)
	defer cleanup()

	spec := SectionSpec{Name: SectionStrings, MinCount: 1, MaxCount: 5}
	sec, err := NewSection(spec, agents)
	if err != nil {
		t.Fatalf("NewSection: %v", err)
	}

	sec.Members[1].Paused = true

	active := sec.BroadcastPart()
	if len(active) != 2 {
		t.Errorf("broadcast to %d members, want 2 (3 - 1 paused)", len(active))
	}
}

func TestAddRemoveMember(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 3, agentsdk.RoleVerifier)
	defer cleanup()

	spec := SectionSpec{Name: SectionStrings, MinCount: 2, MaxCount: 4}
	sec, err := NewSection(spec, agents)
	if err != nil {
		t.Fatalf("NewSection: %v", err)
	}

	// Add a new member.
	newAgents, cleanup2 := testManagedAgents(t, 1, agentsdk.RoleVerifier)
	defer cleanup2()

	err = sec.AddMember(newAgents[0])
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if len(sec.Members) != 4 {
		t.Errorf("members = %d, want 4", len(sec.Members))
	}

	// Adding beyond max should fail.
	err = sec.AddMember(newAgents[0])
	if err == nil {
		t.Fatal("expected error when adding beyond max")
	}

	// Remove a non-leader member.
	leaderName := sec.Leader.Managed.Spec.Name
	var removeName string
	for _, m := range sec.Members {
		if m.Managed.Spec.Name != leaderName {
			removeName = m.Managed.Spec.Name
			break
		}
	}

	err = sec.RemoveMember(removeName)
	if err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	if len(sec.Members) != 3 {
		t.Errorf("members = %d, want 3", len(sec.Members))
	}

	// Cannot remove leader.
	err = sec.RemoveMember(leaderName)
	if err == nil {
		t.Fatal("expected error when removing leader")
	}
}

func TestSectionHealth(t *testing.T) {
	agents, cleanup := testManagedAgents(t, 2, agentsdk.RoleVerifier)
	defer cleanup()

	spec := SectionSpec{Name: SectionStrings, MinCount: 1, MaxCount: 5}
	sec, err := NewSection(spec, agents)
	if err != nil {
		t.Fatalf("NewSection: %v", err)
	}

	sec.Members[0].TotalDone = 5
	sec.Members[0].TotalFailed = 1
	sec.Members[1].ActiveParts = 2

	h := sec.Health()
	if h.MemberCount != 2 {
		t.Errorf("member_count = %d, want 2", h.MemberCount)
	}
	if h.CompletedParts != 5 {
		t.Errorf("completed_parts = %d, want 5", h.CompletedParts)
	}
	if h.FailedParts != 1 {
		t.Errorf("failed_parts = %d, want 1", h.FailedParts)
	}
	if h.ActiveParts != 2 {
		t.Errorf("active_parts = %d, want 2", h.ActiveParts)
	}
}
