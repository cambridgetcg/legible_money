package keeper_test

// ==================================================================================
// OpenClaw Tree Adversarial Probe Suite (OC-TREE-1 through OC-TREE-3)
//
// Attack surfaces tested:
// - Non-founder proposes project (phase transition hijack)
// - Non-founder approves contributor
// - Invalid phase transition (skip phases)
// ==================================================================================

import (
	"testing"

	"github.com/legible-money/legible/x/tree/keeper"
	"github.com/legible-money/legible/x/tree/types"
)

// ==================================================================================
// OC-TREE-1: Non-Founder Phase Transition Hijack
//
// ATTACK: Someone other than the project founder tries to advance the project
// from seed to sprout phase.
// If unguarded, an attacker could rush a project through phases.
//
// DEFENSE: ProposeProject checks msg.Proposer == project.Founder.
// ==================================================================================

func TestOC_TREE1_NonFounderPhaseTransition(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create a project with founder1
	resp, err := msgServer.CreateProject(ctx, &types.MsgCreateProject{
		Creator:     founder1,
		Title:       "Test Project",
		Description: "A test project",
		Domain:      "mathematics",
		Budget:      "1000000",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// ATTACK: Non-founder tries to propose (advance to sprout)
	_, err = msgServer.ProposeProject(ctx, &types.MsgProposeProject{
		Proposer:  founder2, // not the founder
		ProjectId: resp.ProjectId,
	})
	if err == nil {
		t.Fatal("OC-TREE-1 FAIL: non-founder was able to advance project phase!")
	}

	// Verify project still in seed phase
	project, found := k.GetProject(ctx, resp.ProjectId)
	if !found {
		t.Fatal("project not found")
	}
	if project.Phase != string(types.PhaseSeed) {
		t.Fatalf("OC-TREE-1 FAIL: project phase changed by non-founder: %s", project.Phase)
	}

	t.Logf("OC-TREE-1 PASS: non-founder phase transition correctly rejected")
}

// ==================================================================================
// OC-TREE-2: Non-Founder Adds Contributor
//
// ATTACK: Someone other than the founder approves a contributor into a project.
// If unguarded, attackers could stuff projects with accomplices.
//
// DEFENSE: ReviewApplication checks msg.Reviewer == project.Founder.
// ==================================================================================

func TestOC_TREE2_NonFounderAddsContributor(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create a project
	resp, err := msgServer.CreateProject(ctx, &types.MsgCreateProject{
		Creator:     founder1,
		Title:       "Test Project",
		Description: "A test project",
		Domain:      "mathematics",
		Budget:      "1000000",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Agent applies
	_, err = msgServer.ApplyToProject(ctx, &types.MsgApplyToProject{
		Applicant: agent1,
		ProjectId: resp.ProjectId,
		Role:      "developer",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	// ATTACK: Non-founder tries to review/approve the application
	_, err = msgServer.ReviewApplication(ctx, &types.MsgReviewApplication{
		Reviewer:      founder2, // not the actual founder
		ApplicationId: agent1,   // ApplicationId is the applicant's DID
		Accepted:      true,
		Reason:        "Approving from wrong account",
	})
	if err == nil {
		t.Fatal("OC-TREE-2 FAIL: non-founder was able to approve contributor!")
	}

	// Verify contributor not promoted
	project, _ := k.GetProject(ctx, resp.ProjectId)
	for _, c := range project.Contributors {
		if c.Did == agent1 {
			t.Fatalf("OC-TREE-2 FAIL: agent was approved by non-founder: role=%s", c.Role)
		}
	}

	t.Logf("OC-TREE-2 PASS: non-founder contributor approval correctly rejected")
}

// ==================================================================================
// OC-TREE-3: Invalid Phase Transition (Skip Phase)
//
// ATTACK: Try to seed a project that is still in seed phase (must be in sprout).
// If unguarded, projects could skip review/sprout phases.
//
// DEFENSE: Phase transition validation checks IsValidPhaseTransition.
// ==================================================================================

func TestOC_TREE3_InvalidPhaseTransition(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create a project (starts in seed phase)
	resp, err := msgServer.CreateProject(ctx, &types.MsgCreateProject{
		Creator:     founder1,
		Title:       "Test Project",
		Description: "A test project",
		Domain:      "mathematics",
		Budget:      "1000000",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// ATTACK: Try to seed the project (which requires sprout phase with enough contributors)
	_, err = msgServer.SeedProject(ctx, &types.MsgSeedProject{
		Founder:   founder1,
		ProjectId: resp.ProjectId,
	})
	if err == nil {
		t.Fatal("OC-TREE-3 FAIL: phase skip allowed (seed -> grow directly)!")
	}

	// Verify still in seed phase
	project, _ := k.GetProject(ctx, resp.ProjectId)
	if project.Phase != string(types.PhaseSeed) {
		t.Fatalf("OC-TREE-3 FAIL: phase changed from seed to %s", project.Phase)
	}

	t.Logf("OC-TREE-3 PASS: invalid phase transition correctly rejected")
}
