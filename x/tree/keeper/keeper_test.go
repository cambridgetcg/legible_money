package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/tree/keeper"
	"github.com/legible-money/legible/x/tree/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

// ---------- Mock BankKeeper ----------

type mockBankKeeper struct {
	balances       map[string]map[string]int64
	moduleBalances map[string]map[string]int64
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances:       make(map[string]map[string]int64),
		moduleBalances: make(map[string]map[string]int64),
	}
}

func (m *mockBankKeeper) SendCoins(_ context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		from := fromAddr.String()
		to := toAddr.String()
		if m.balances[from] == nil {
			m.balances[from] = make(map[string]int64)
		}
		if m.balances[to] == nil {
			m.balances[to] = make(map[string]int64)
		}
		m.balances[from][coin.Denom] -= coin.Amount.Int64()
		m.balances[to][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		from := senderAddr.String()
		if m.balances[from] == nil {
			m.balances[from] = make(map[string]int64)
		}
		if m.moduleBalances[recipientModule] == nil {
			m.moduleBalances[recipientModule] = make(map[string]int64)
		}
		m.balances[from][coin.Denom] -= coin.Amount.Int64()
		m.moduleBalances[recipientModule][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		to := recipientAddr.String()
		if m.moduleBalances[senderModule] == nil {
			m.moduleBalances[senderModule] = make(map[string]int64)
		}
		if m.balances[to] == nil {
			m.balances[to] = make(map[string]int64)
		}
		m.moduleBalances[senderModule][coin.Denom] -= coin.Amount.Int64()
		m.balances[to][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
	t.Helper()
	initAddresses()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockBK := newMockBankKeeper()

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), mockBK, "lgm1authority")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx, mockBK
}

func testAddr(name string) string {
	return sdk.AccAddress([]byte(name + "_address_padding")).String()
}

var (
	founder1  string
	founder2  string
	agent1    string
	agent2    string
	reviewer1 string
)

func initAddresses() {
	founder1 = testAddr("founder1")
	founder2 = testAddr("founder2")
	agent1 = testAddr("agent001")
	agent2 = testAddr("agent002")
	reviewer1 = testAddr("reviewer")
}

// ---------- Params Tests ----------

func TestDefaultParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	params := k.GetParams(ctx)
	if params.MaxTasksPerProject != 200 {
		t.Errorf("expected MaxTasksPerProject=200, got %d", params.MaxTasksPerProject)
	}
	if params.MaxRejections != 3 {
		t.Errorf("expected MaxRejections=3, got %d", params.MaxRejections)
	}
}

func TestSetGetParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	custom := types.Params{
		MinBudget:              "5000000",
		MaxTasksPerProject:     50,
		MaxContributors:        10,
		MaxApplications:        20,
		TaskDeadlineMinBlocks:  500,
		TaskDeadlineMaxBlocks:  500000,
		MaxRejections:          5,
		SeedExpiryBlocks:       100000,
		MinContributorsToStart: 2,
	}
	k.SetParams(ctx, custom)
	got := k.GetParams(ctx)
	if got.MaxTasksPerProject != 50 {
		t.Errorf("expected MaxTasksPerProject=50, got %d", got.MaxTasksPerProject)
	}
	if got.MinContributorsToStart != 2 {
		t.Errorf("expected MinContributorsToStart=2, got %d", got.MinContributorsToStart)
	}
}

// ---------- Project CRUD Tests ----------

func TestSetGetProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	project := types.ProductProject{
		Id:              "proj-1",
		Name:            "Test Project",
		Phase:           string(types.PhaseSeed),
		Founder:         founder1,
		KnowledgeDomain: "mathematics",
		Budget:          "1000000",
		Spent:           "0",
		TaskIds:         []string{},
		ServiceIds:      []string{},
		Contributors:    []*types.ContributorRecord{},
	}
	k.SetProject(ctx, project)

	got, found := k.GetProject(ctx, "proj-1")
	if !found {
		t.Fatal("project not found")
	}
	if got.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got '%s'", got.Name)
	}
	if got.Founder != founder1 {
		t.Errorf("expected founder %s, got %s", founder1, got.Founder)
	}
}

func TestGetProjectNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	_, found := k.GetProject(ctx, "nonexistent")
	if found {
		t.Error("expected project not found")
	}
}

func TestGetProjectsByFounder(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetProject(ctx, types.ProductProject{Id: "p1", Founder: founder1, KnowledgeDomain: "math", TaskIds: []string{}, ServiceIds: []string{}, Contributors: []*types.ContributorRecord{}})
	k.SetProject(ctx, types.ProductProject{Id: "p2", Founder: founder1, KnowledgeDomain: "physics", TaskIds: []string{}, ServiceIds: []string{}, Contributors: []*types.ContributorRecord{}})
	k.SetProject(ctx, types.ProductProject{Id: "p3", Founder: founder2, KnowledgeDomain: "cs", TaskIds: []string{}, ServiceIds: []string{}, Contributors: []*types.ContributorRecord{}})

	projects := k.GetProjectsByFounder(ctx, founder1)
	if len(projects) != 2 {
		t.Errorf("expected 2 projects for founder1, got %d", len(projects))
	}
}

func TestDeleteProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	project := types.ProductProject{Id: "p1", Founder: founder1, KnowledgeDomain: "math", TaskIds: []string{}, ServiceIds: []string{}, Contributors: []*types.ContributorRecord{}}
	k.SetProject(ctx, project)
	k.DeleteProject(ctx, project)

	_, found := k.GetProject(ctx, "p1")
	if found {
		t.Error("expected project to be deleted")
	}
}

// ---------- Task CRUD Tests ----------

func TestSetGetTask(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	task := types.ProjectTask{
		Id:        "task-1",
		ProjectId: "proj-1",
		Title:     "Implement feature",
		Status:    string(types.TaskOpen),
		Reviewers: []string{},
	}
	k.SetTask(ctx, task)

	got, found := k.GetTask(ctx, "task-1")
	if !found {
		t.Fatal("task not found")
	}
	if got.Title != "Implement feature" {
		t.Errorf("expected title 'Implement feature', got '%s'", got.Title)
	}
}

func TestGetTasksByProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetTask(ctx, types.ProjectTask{Id: "t1", ProjectId: "proj-1", Status: string(types.TaskOpen), Reviewers: []string{}})
	k.SetTask(ctx, types.ProjectTask{Id: "t2", ProjectId: "proj-1", Status: string(types.TaskOpen), Reviewers: []string{}})
	k.SetTask(ctx, types.ProjectTask{Id: "t3", ProjectId: "proj-2", Status: string(types.TaskOpen), Reviewers: []string{}})

	tasks := k.GetTasksByProject(ctx, "proj-1")
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks for proj-1, got %d", len(tasks))
	}
}

// ---------- Seed CRUD Tests ----------

func TestSetGetSeed(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	seed := types.OpportunitySeed{
		Id:              "seed-1",
		KnowledgeDomain: "physics",
		Status:          string(types.SeedDetected),
		ExpiresAtBlock:  200,
	}
	k.SetSeed(ctx, seed)

	got, found := k.GetSeed(ctx, "seed-1")
	if !found {
		t.Fatal("seed not found")
	}
	if got.Status != string(types.SeedDetected) {
		t.Errorf("expected status 'detected', got '%s'", got.Status)
	}
}

func TestGetExpiredSeeds(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetSeed(ctx, types.OpportunitySeed{Id: "s1", KnowledgeDomain: "math", Status: string(types.SeedDetected), ExpiresAtBlock: 50})
	k.SetSeed(ctx, types.OpportunitySeed{Id: "s2", KnowledgeDomain: "math", Status: string(types.SeedDetected), ExpiresAtBlock: 150})
	k.SetSeed(ctx, types.OpportunitySeed{Id: "s3", KnowledgeDomain: "math", Status: string(types.SeedExpired), ExpiresAtBlock: 50}) // already expired

	expired := k.GetExpiredSeeds(ctx, 100)
	if len(expired) != 1 {
		t.Errorf("expected 1 expired seed, got %d", len(expired))
	}
	if len(expired) > 0 && expired[0].Id != "s1" {
		t.Errorf("expected expired seed s1, got %s", expired[0].Id)
	}
}

// ---------- Service CRUD Tests ----------

func TestSetGetService(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	svc := types.ServiceLeaf{
		Id:        "svc-1",
		ProjectId: "proj-1",
		Name:      "API Service",
		Status:    string(types.ServiceActive),
	}
	k.SetService(ctx, svc)

	got, found := k.GetService(ctx, "svc-1")
	if !found {
		t.Fatal("service not found")
	}
	if got.Name != "API Service" {
		t.Errorf("expected name 'API Service', got '%s'", got.Name)
	}
}

// ---------- Phase Transition Tests ----------

func TestValidPhaseTransitions(t *testing.T) {
	tests := []struct {
		from  types.ProjectPhase
		to    types.ProjectPhase
		valid bool
	}{
		{types.PhaseSeed, types.PhaseSprout, true},
		{types.PhaseSeed, types.PhaseWithered, true},
		{types.PhaseSeed, types.PhaseGrowing, false},
		{types.PhaseSprout, types.PhaseGrowing, true},
		{types.PhaseSprout, types.PhaseFruiting, false},
		{types.PhaseGrowing, types.PhaseMature, true},
		{types.PhaseGrowing, types.PhaseDormant, true},
		{types.PhaseMature, types.PhaseFruiting, true},
		{types.PhaseFruiting, types.PhaseSeeding, true},
		{types.PhaseDormant, types.PhaseGrowing, true},
		{types.PhaseDormant, types.PhaseMature, true},
		{types.PhaseWithered, types.PhaseSeed, false},
	}

	for _, tt := range tests {
		got := keeper.IsValidPhaseTransition(tt.from, tt.to)
		if got != tt.valid {
			t.Errorf("phase %s->%s: expected valid=%v, got %v", tt.from, tt.to, tt.valid, got)
		}
	}
}

// ---------- Msg Server Tests: Project Lifecycle ----------

func TestCreateProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, err := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator:     founder1,
		Title:       "My Project",
		Description: "A test project",
		Domain:      "mathematics",
		Budget:      "1000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ProjectId == "" {
		t.Fatal("expected non-empty project ID")
	}

	project, found := k.GetProject(ctx, resp.ProjectId)
	if !found {
		t.Fatal("project not found after creation")
	}
	if project.Phase != string(types.PhaseSeed) {
		t.Errorf("expected phase 'seed', got '%s'", project.Phase)
	}
	if len(project.Contributors) != 1 {
		t.Errorf("expected 1 contributor (founder), got %d", len(project.Contributors))
	}
}

func TestProposeProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	_, err := srv.ProposeProject(ctx, &types.MsgProposeProject{
		Proposer: founder1, ProjectId: resp.ProjectId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := k.GetProject(ctx, resp.ProjectId)
	if project.Phase != string(types.PhaseSprout) {
		t.Errorf("expected phase 'sprout', got '%s'", project.Phase)
	}
}

func TestProposeProject_NotFounder(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	_, err := srv.ProposeProject(ctx, &types.MsgProposeProject{
		Proposer: founder2, ProjectId: resp.ProjectId,
	})
	if err == nil {
		t.Fatal("expected error for non-founder propose")
	}
}

func TestStartDevelopment(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})
	srv.ProposeProject(ctx, &types.MsgProposeProject{Proposer: founder1, ProjectId: resp.ProjectId})

	_, err := srv.StartDevelopment(ctx, &types.MsgStartDevelopment{
		Authority: founder1, ProjectId: resp.ProjectId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := k.GetProject(ctx, resp.ProjectId)
	if project.Phase != string(types.PhaseGrowing) {
		t.Errorf("expected phase 'growing', got '%s'", project.Phase)
	}
}

func TestStartDevelopment_InvalidPhase(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	// Try to start from seed (should fail — need sprout first)
	_, err := srv.StartDevelopment(ctx, &types.MsgStartDevelopment{
		Authority: founder1, ProjectId: resp.ProjectId,
	})
	if err == nil {
		t.Fatal("expected error for invalid phase transition seed->growing")
	}
}

func TestPauseAndResumeProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})
	srv.ProposeProject(ctx, &types.MsgProposeProject{Proposer: founder1, ProjectId: resp.ProjectId})
	srv.StartDevelopment(ctx, &types.MsgStartDevelopment{Authority: founder1, ProjectId: resp.ProjectId})

	// Pause
	_, err := srv.PauseProject(ctx, &types.MsgPauseProject{
		Authority: founder1, ProjectId: resp.ProjectId, Reason: "funding issue",
	})
	if err != nil {
		t.Fatalf("unexpected pause error: %v", err)
	}

	project, _ := k.GetProject(ctx, resp.ProjectId)
	if project.Phase != string(types.PhaseDormant) {
		t.Errorf("expected dormant, got %s", project.Phase)
	}

	// Resume
	_, err = srv.ResumeProject(ctx, &types.MsgResumeProject{
		Authority: founder1, ProjectId: resp.ProjectId,
	})
	if err != nil {
		t.Fatalf("unexpected resume error: %v", err)
	}

	project, _ = k.GetProject(ctx, resp.ProjectId)
	if project.Phase != string(types.PhaseGrowing) {
		t.Errorf("expected growing after resume, got %s", project.Phase)
	}
}

func TestAbandonProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	_, err := srv.AbandonProject(ctx, &types.MsgAbandonProject{
		Authority: founder1, ProjectId: resp.ProjectId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := k.GetProject(ctx, resp.ProjectId)
	if project.Phase != string(types.PhaseWithered) {
		t.Errorf("expected withered, got %s", project.Phase)
	}
}

func TestSpawnChildProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Create parent and advance to fruiting
	parentResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Parent", Domain: "math", Budget: "1000000",
	})
	srv.ProposeProject(ctx, &types.MsgProposeProject{Proposer: founder1, ProjectId: parentResp.ProjectId})
	srv.StartDevelopment(ctx, &types.MsgStartDevelopment{Authority: founder1, ProjectId: parentResp.ProjectId})

	// Deploy a service (required for fruiting)
	srv.DeployService(ctx, &types.MsgDeployService{
		Deployer: founder1, Name: "Svc",
	})

	// CompleteProject: growing -> mature
	_, err := srv.CompleteProject(ctx, &types.MsgCompleteProject{Authority: founder1, ProjectId: parentResp.ProjectId})
	if err != nil {
		t.Fatalf("complete project error: %v", err)
	}
	parent, _ := k.GetProject(ctx, parentResp.ProjectId)
	if parent.Phase != string(types.PhaseMature) {
		t.Fatalf("expected mature, got %s", parent.Phase)
	}

	// Advance to fruiting: mature -> fruiting
	// Since MsgDeployProject is not in the proto, we manually set the phase for testing
	parent.PreviousPhase = string(types.PhaseMature)
	parent.Phase = string(types.PhaseFruiting)
	parent.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	k.SetProject(ctx, parent)

	parent, _ = k.GetProject(ctx, parentResp.ProjectId)
	if parent.Phase != string(types.PhaseFruiting) {
		t.Fatalf("expected fruiting, got %s", parent.Phase)
	}

	// Spawn child
	childResp, err := srv.SpawnChildProject(ctx, &types.MsgSpawnChildProject{
		Creator:         founder1,
		ParentProjectId: parentResp.ProjectId,
		Title:           "Child",
		Description:     "A child project",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	child, found := k.GetProject(ctx, childResp.ChildProjectId)
	if !found {
		t.Fatal("child project not found")
	}
	if child.ParentProjectId != parentResp.ProjectId {
		t.Errorf("expected parent ID %s, got %s", parentResp.ProjectId, child.ParentProjectId)
	}
	if child.Phase != string(types.PhaseSeed) {
		t.Errorf("expected child in seed phase, got %s", child.Phase)
	}

	// Parent should advance to seeding
	parent, _ = k.GetProject(ctx, parentResp.ProjectId)
	if parent.Phase != string(types.PhaseSeeding) {
		t.Errorf("expected parent in seeding phase, got %s", parent.Phase)
	}
}

// ---------- Msg Server Tests: Task Workflow ----------

func TestAddTask(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	taskResp, err := srv.AddTask(ctx, &types.MsgAddTask{
		Creator:     founder1,
		ProjectId:   projResp.ProjectId,
		Title:       "Implement module",
		Description: "Implement the tree module",
		Bounty:      "100000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task, found := k.GetTask(ctx, taskResp.TaskId)
	if !found {
		t.Fatal("task not found")
	}
	if task.Status != string(types.TaskOpen) {
		t.Errorf("expected status 'open', got '%s'", task.Status)
	}

	// Project should track the task
	project, _ := k.GetProject(ctx, projResp.ProjectId)
	if project.TotalTasks != 1 {
		t.Errorf("expected 1 total task, got %d", project.TotalTasks)
	}
}

func TestAddTask_NotContributor(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	_, err := srv.AddTask(ctx, &types.MsgAddTask{
		Creator:   agent1,
		ProjectId: projResp.ProjectId,
		Title:     "Task",
	})
	if err == nil {
		t.Fatal("expected error for non-contributor adding task")
	}
}

func TestTaskWorkflow_AssignStartSubmitApprove(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	taskResp, _ := srv.AddTask(ctx, &types.MsgAddTask{
		Creator: founder1, ProjectId: projResp.ProjectId, Title: "Task",
	})

	// Assign
	_, err := srv.AssignTask(ctx, &types.MsgAssignTask{
		Assigner: founder1, TaskId: taskResp.TaskId, Assignee: agent1,
	})
	if err != nil {
		t.Fatalf("assign error: %v", err)
	}
	task, _ := k.GetTask(ctx, taskResp.TaskId)
	if task.Status != string(types.TaskAssigned) {
		t.Errorf("expected assigned, got %s", task.Status)
	}

	// Start work
	_, err = srv.StartWork(ctx, &types.MsgStartWork{
		Worker: agent1, TaskId: taskResp.TaskId,
	})
	if err != nil {
		t.Fatalf("start work error: %v", err)
	}
	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != string(types.TaskInProgress) {
		t.Errorf("expected in_progress, got %s", task.Status)
	}

	// Submit deliverable
	_, err = srv.SubmitDeliverable(ctx, &types.MsgSubmitDeliverable{
		Worker:          agent1,
		TaskId:          taskResp.TaskId,
		DeliverableHash: "abc123",
	})
	if err != nil {
		t.Fatalf("submit deliverable error: %v", err)
	}
	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != string(types.TaskReview) {
		t.Errorf("expected review, got %s", task.Status)
	}

	// Approve
	_, err = srv.ApproveDeliverable(ctx, &types.MsgApproveDeliverable{
		Approver: founder1, TaskId: taskResp.TaskId,
	})
	if err != nil {
		t.Fatalf("approve error: %v", err)
	}
	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != string(types.TaskCompleted) {
		t.Errorf("expected completed, got %s", task.Status)
	}

	// Project completed tasks should increment
	project, _ := k.GetProject(ctx, projResp.ProjectId)
	if project.CompletedTasks != 1 {
		t.Errorf("expected 1 completed task, got %d", project.CompletedTasks)
	}
}

func TestRejectDeliverable_MaxRejections(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})
	taskResp, _ := srv.AddTask(ctx, &types.MsgAddTask{
		Creator: founder1, ProjectId: projResp.ProjectId, Title: "Task",
	})
	srv.AssignTask(ctx, &types.MsgAssignTask{Assigner: founder1, TaskId: taskResp.TaskId, Assignee: agent1})
	srv.StartWork(ctx, &types.MsgStartWork{Worker: agent1, TaskId: taskResp.TaskId})
	srv.SubmitDeliverable(ctx, &types.MsgSubmitDeliverable{
		Worker: agent1, TaskId: taskResp.TaskId,
		DeliverableHash: "abc",
	})

	// Reject 1: goes back to in_progress
	srv.RejectDeliverable(ctx, &types.MsgRejectDeliverable{
		Rejector: founder1, TaskId: taskResp.TaskId, Reason: "bad",
	})
	task, _ := k.GetTask(ctx, taskResp.TaskId)
	if task.Status != string(types.TaskInProgress) {
		t.Errorf("after rejection 1: expected in_progress, got %s", task.Status)
	}

	// Resubmit and reject again
	srv.SubmitDeliverable(ctx, &types.MsgSubmitDeliverable{
		Worker: agent1, TaskId: taskResp.TaskId,
		DeliverableHash: "def",
	})
	srv.RejectDeliverable(ctx, &types.MsgRejectDeliverable{
		Rejector: founder1, TaskId: taskResp.TaskId, Reason: "still bad",
	})
	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != string(types.TaskInProgress) {
		t.Errorf("after rejection 2: expected in_progress, got %s", task.Status)
	}

	// Resubmit and reject 3rd time — should become disputed
	srv.SubmitDeliverable(ctx, &types.MsgSubmitDeliverable{
		Worker: agent1, TaskId: taskResp.TaskId,
		DeliverableHash: "ghi",
	})
	srv.RejectDeliverable(ctx, &types.MsgRejectDeliverable{
		Rejector: founder1, TaskId: taskResp.TaskId, Reason: "final rejection",
	})
	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != string(types.TaskDisputed) {
		t.Errorf("after rejection 3: expected disputed, got %s", task.Status)
	}
}

func TestReopenTask(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	// Create a task and manually set it to disputed
	taskResp, _ := srv.AddTask(ctx, &types.MsgAddTask{
		Creator: founder1, ProjectId: projResp.ProjectId, Title: "Task",
	})
	task, _ := k.GetTask(ctx, taskResp.TaskId)
	task.Status = string(types.TaskDisputed)
	task.Assignee = agent1
	k.SetTask(ctx, task)

	// Reopen
	_, err := srv.ReopenTask(ctx, &types.MsgReopenTask{
		Authority: founder1, TaskId: taskResp.TaskId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != string(types.TaskOpen) {
		t.Errorf("expected open after reopen, got %s", task.Status)
	}
	if task.Assignee != "" {
		t.Error("expected assignee to be cleared after reopen")
	}
}

func TestStartWork_NotAssignee(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})
	taskResp, _ := srv.AddTask(ctx, &types.MsgAddTask{
		Creator: founder1, ProjectId: projResp.ProjectId, Title: "Task",
	})
	srv.AssignTask(ctx, &types.MsgAssignTask{Assigner: founder1, TaskId: taskResp.TaskId, Assignee: agent1})

	_, err := srv.StartWork(ctx, &types.MsgStartWork{
		Worker: agent2, TaskId: taskResp.TaskId,
	})
	if err == nil {
		t.Fatal("expected error for non-assignee starting work")
	}
}

// ---------- Application Tests ----------

func TestApplyAndReviewApplication(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	// Apply
	_, err := srv.ApplyToProject(ctx, &types.MsgApplyToProject{
		Applicant: agent1,
		ProjectId: projResp.ProjectId,
		Role:      string(types.RoleDeveloper),
		Pitch:     "I want to contribute",
	})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	project, _ := k.GetProject(ctx, projResp.ProjectId)
	if len(project.Applications) != 1 {
		t.Fatalf("expected 1 application, got %d", len(project.Applications))
	}

	// Approve application
	// Proto uses ApplicationId (not ProjectId+ApplicantDid) and Accepted (not Approve).
	// The msg_server.go ReviewApplication iterates projects looking for app.Did == msg.ApplicationId.
	_, err = srv.ReviewApplication(ctx, &types.MsgReviewApplication{
		Reviewer:      founder1,
		ApplicationId: agent1,
		Accepted:      true,
	})
	if err != nil {
		t.Fatalf("review error: %v", err)
	}

	project, _ = k.GetProject(ctx, projResp.ProjectId)
	if len(project.Contributors) != 2 { // founder + new contributor
		t.Errorf("expected 2 contributors, got %d", len(project.Contributors))
	}
}

func TestApplyToProject_DuplicateApplication(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	srv.ApplyToProject(ctx, &types.MsgApplyToProject{
		Applicant: agent1, ProjectId: projResp.ProjectId, Role: string(types.RoleDeveloper),
	})

	_, err := srv.ApplyToProject(ctx, &types.MsgApplyToProject{
		Applicant: agent1, ProjectId: projResp.ProjectId, Role: string(types.RoleDeveloper),
	})
	if err == nil {
		t.Fatal("expected error for duplicate application")
	}
}

// ---------- Service Tests ----------

func TestDeployService(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Deploy service (proto MsgDeployService has no ProjectId or ServiceType)
	svcResp, err := srv.DeployService(ctx, &types.MsgDeployService{
		Deployer:    founder1,
		Name:        "Math API",
		Description: "A math computation API",
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	svc, found := k.GetService(ctx, svcResp.ServiceId)
	if !found {
		t.Fatal("service not found")
	}
	if svc.Status != string(types.ServiceDeploying) {
		t.Errorf("expected status deploying, got %s", svc.Status)
	}
}

func TestPauseResumeRetireService(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	svcResp, _ := srv.DeployService(ctx, &types.MsgDeployService{
		Deployer: founder1, Name: "Svc",
	})

	// Pause
	_, err := srv.PauseService(ctx, &types.MsgPauseService{Owner: founder1, ServiceId: svcResp.ServiceId})
	if err != nil {
		t.Fatalf("pause error: %v", err)
	}
	svc, _ := k.GetService(ctx, svcResp.ServiceId)
	if svc.Status != string(types.ServicePaused) {
		t.Errorf("expected paused, got %s", svc.Status)
	}

	// Resume
	_, err = srv.ResumeService(ctx, &types.MsgResumeService{Owner: founder1, ServiceId: svcResp.ServiceId})
	if err != nil {
		t.Fatalf("resume error: %v", err)
	}
	svc, _ = k.GetService(ctx, svcResp.ServiceId)
	if svc.Status != string(types.ServiceActive) {
		t.Errorf("expected active, got %s", svc.Status)
	}

	// Retire
	_, err = srv.RetireService(ctx, &types.MsgRetireService{Owner: founder1, ServiceId: svcResp.ServiceId})
	if err != nil {
		t.Fatalf("retire error: %v", err)
	}
	svc, _ = k.GetService(ctx, svcResp.ServiceId)
	if svc.Status != string(types.ServiceRetired) {
		t.Errorf("expected retired, got %s", svc.Status)
	}

	// Can't retire again
	_, err = srv.RetireService(ctx, &types.MsgRetireService{Owner: founder1, ServiceId: svcResp.ServiceId})
	if err == nil {
		t.Fatal("expected error for retiring already retired service")
	}
}

// ---------- Query Tests ----------

func TestQueryProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	querySrv := keeper.NewQueryServerImpl(k)
	msgSrv := keeper.NewMsgServerImpl(k)

	resp, _ := msgSrv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})

	qResp, err := querySrv.Project(ctx, &types.QueryProjectRequest{ProjectId: resp.ProjectId})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if qResp.Project.Name != "Project" {
		t.Errorf("expected name 'Project', got '%s'", qResp.Project.Name)
	}
}

func TestQueryProjectNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	querySrv := keeper.NewQueryServerImpl(k)

	_, err := querySrv.Project(ctx, &types.QueryProjectRequest{ProjectId: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent project query")
	}
}

func TestQueryProjectsByFounder(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	querySrv := keeper.NewQueryServerImpl(k)
	msgSrv := keeper.NewMsgServerImpl(k)

	msgSrv.CreateProject(ctx, &types.MsgCreateProject{Creator: founder1, Title: "P1", Domain: "math", Budget: "1000000"})
	msgSrv.CreateProject(ctx, &types.MsgCreateProject{Creator: founder1, Title: "P2", Domain: "physics", Budget: "2000000"})

	qResp, err := querySrv.ProjectsByFounder(ctx, &types.QueryProjectsByFounderRequest{Founder: founder1})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if qResp.Total != 2 {
		t.Errorf("expected 2 projects, got %d", qResp.Total)
	}
}

func TestQueryTasksByProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	querySrv := keeper.NewQueryServerImpl(k)
	msgSrv := keeper.NewMsgServerImpl(k)

	projResp, _ := msgSrv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Project", Domain: "math", Budget: "1000000",
	})
	msgSrv.AddTask(ctx, &types.MsgAddTask{Creator: founder1, ProjectId: projResp.ProjectId, Title: "T1"})
	msgSrv.AddTask(ctx, &types.MsgAddTask{Creator: founder1, ProjectId: projResp.ProjectId, Title: "T2"})

	qResp, err := querySrv.TasksByProject(ctx, &types.QueryTasksByProjectRequest{ProjectId: projResp.ProjectId})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if qResp.Total != 2 {
		t.Errorf("expected 2 tasks, got %d", qResp.Total)
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	querySrv := keeper.NewQueryServerImpl(k)

	qResp, err := querySrv.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if qResp.Params.MaxTasksPerProject != 200 {
		t.Errorf("expected MaxTasksPerProject=200, got %d", qResp.Params.MaxTasksPerProject)
	}
}

// ---------- Genesis Tests ----------

func TestGenesisRoundtrip(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	msgSrv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "P1", Domain: "math", Budget: "1000000",
	})

	// Export
	gs := k.ExportGenesis(ctx)
	if len(gs.Projects) != 1 {
		t.Fatalf("expected 1 project in export, got %d", len(gs.Projects))
	}

	// Create a new keeper and import
	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, gs)

	// Verify
	gs2 := k2.ExportGenesis(ctx2)
	if len(gs2.Projects) != 1 {
		t.Errorf("expected 1 project after import, got %d", len(gs2.Projects))
	}
	if gs2.Projects[0].Name != "P1" {
		t.Errorf("expected project name 'P1', got '%s'", gs2.Projects[0].Name)
	}
}

// ======================== Revenue Distribution Tests ========================

func TestCalculateRevenue(t *testing.T) {
	contributors := []*types.ContributorRecord{
		{Did: testAddr("alice"), TasksCompleted: 3, TotalEarned: "0"},
		{Did: testAddr("bob"), TasksCompleted: 1, TotalEarned: "0"},
	}

	dist := keeper.CalculateRevenue(1000000, 220000, 390000, 234000, 156000, contributors)

	// Verify 4-pool split sums to total (no dust loss)
	total := dist.EvidenceTax + dist.ContributorPool + dist.ResearchFund + dist.ProtocolTreasury
	if total != 1000000 {
		t.Errorf("revenue split doesn't sum to total: %d + %d + %d + %d = %d, want 1000000",
			dist.EvidenceTax, dist.ContributorPool, dist.ResearchFund, dist.ProtocolTreasury, total)
	}

	// Verify approximate basis point split
	if dist.EvidenceTax != 220000 {
		t.Errorf("evidence tax: got %d, want 220000", dist.EvidenceTax)
	}
	if dist.ContributorPool != 390000 {
		t.Errorf("contributor pool: got %d, want 390000", dist.ContributorPool)
	}
	if dist.ResearchFund != 234000 {
		t.Errorf("research fund: got %d, want 234000", dist.ResearchFund)
	}

	// Verify evidence tax 5-way sub-split sums to evidence tax total
	taxTotal := dist.TaxResearch + dist.TaxFoundation + dist.TaxCommunity + dist.TaxDevelopers + dist.TaxReserve
	if taxTotal != dist.EvidenceTax {
		t.Errorf("evidence tax sub-split doesn't sum: %d != %d", taxTotal, dist.EvidenceTax)
	}

	// Verify contributor shares sum to contributor pool
	var contribTotal int64
	for _, s := range dist.ContributorShares {
		contribTotal += s.Amount
	}
	if contribTotal != dist.ContributorPool {
		t.Errorf("contributor shares don't sum to pool: %d != %d", contribTotal, dist.ContributorPool)
	}

	// Verify pro-rata ordering (sorted by address)
	if len(dist.ContributorShares) != 2 {
		t.Fatalf("expected 2 contributor shares, got %d", len(dist.ContributorShares))
	}
	if dist.ContributorShares[0].Address >= dist.ContributorShares[1].Address {
		t.Error("contributor shares not sorted by address")
	}
}

func TestCalculateRevenue_ZeroAmount(t *testing.T) {
	dist := keeper.CalculateRevenue(0, 220000, 390000, 234000, 156000, nil)
	if dist.EvidenceTax != 0 || dist.ContributorPool != 0 || dist.ResearchFund != 0 || dist.ProtocolTreasury != 0 {
		t.Error("expected all zeros for zero amount")
	}
}

func TestCalculateRevenue_NoContributors(t *testing.T) {
	dist := keeper.CalculateRevenue(1000000, 220000, 390000, 234000, 156000, nil)

	// Contributor pool should redirect to treasury
	if dist.ContributorPool != 0 {
		t.Errorf("contributor pool should be 0 when no contributors, got %d", dist.ContributorPool)
	}

	// Treasury should absorb contributor pool
	expectedTreasury := int64(1000000) - int64(220000) - int64(234000) // total - evidenceTax - research
	if dist.ProtocolTreasury != expectedTreasury {
		t.Errorf("treasury: got %d, want %d", dist.ProtocolTreasury, expectedTreasury)
	}

	total := dist.EvidenceTax + dist.ContributorPool + dist.ResearchFund + dist.ProtocolTreasury
	if total != 1000000 {
		t.Errorf("total doesn't sum: %d", total)
	}
}

func TestCalculateRevenue_SingleContributor(t *testing.T) {
	contributors := []*types.ContributorRecord{
		{Did: testAddr("solo"), TasksCompleted: 5, TotalEarned: "0"},
	}
	dist := keeper.CalculateRevenue(1000000, 220000, 390000, 234000, 156000, contributors)

	if len(dist.ContributorShares) != 1 {
		t.Fatalf("expected 1 share, got %d", len(dist.ContributorShares))
	}
	if dist.ContributorShares[0].Amount != dist.ContributorPool {
		t.Errorf("single contributor should get entire pool: %d != %d",
			dist.ContributorShares[0].Amount, dist.ContributorPool)
	}
}

// ======================== CallService Tests ========================

// setupServiceForCall is a helper that creates a project with a paid service and activates it.
func setupServiceForCall(t *testing.T, k keeper.Keeper, ctx sdk.Context, srv types.MsgServer, price string) (projectId, serviceId string) {
	t.Helper()

	projResp, err := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Service Project", Domain: "api", Budget: "10000000",
	})
	if err != nil {
		t.Fatalf("create project error: %v", err)
	}
	projectId = projResp.ProjectId

	// Deploy service (proto MsgDeployService has no ProjectId or ServiceType)
	svcResp, err := srv.DeployService(ctx, &types.MsgDeployService{
		Deployer:     founder1,
		Name:         "Paid API",
		PricePerCall: price,
	})
	if err != nil {
		t.Fatalf("deploy service error: %v", err)
	}
	serviceId = svcResp.ServiceId

	// Manually activate the service since MsgActivateService is not in proto
	svc, found := k.GetService(ctx, serviceId)
	if !found {
		t.Fatal("service not found after deploy")
	}
	svc.Status = string(types.ServiceActive)
	svc.ProjectId = projectId // Link to project for revenue distribution
	k.SetService(ctx, svc)

	return
}

func TestCallService(t *testing.T) {
	k, ctx, mockBK := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projectId, serviceId := setupServiceForCall(t, k, ctx, srv, "1000000")

	// Fund the caller
	callerAddr := testAddr("caller1")
	mockBK.balances[callerAddr] = map[string]int64{"ulgm": 10000000}

	// Set up revenue params with valid addresses
	researchAddr := testAddr("research")
	treasuryAddr := testAddr("treasury")
	params := k.GetParams(ctx)
	params.ResearchFundAddr = researchAddr
	params.ProtocolTreasuryAddr = treasuryAddr
	k.SetParams(ctx, params)

	// Call service (proto uses Payload []byte, not Input string)
	resp, err := srv.CallService(ctx, &types.MsgCallService{
		Caller:    callerAddr,
		ServiceId: serviceId,
		Payload:   []byte("test query"),
	})
	if err != nil {
		t.Fatalf("call service error: %v", err)
	}
	// Proto response uses Result []byte, not AmountCharged string
	if string(resp.Result) != "1000000" {
		t.Errorf("expected result '1000000', got '%s'", string(resp.Result))
	}

	// Verify service metrics updated
	svc, _ := k.GetService(ctx, serviceId)
	if svc.TotalCalls != "1" {
		t.Errorf("expected TotalCalls=1, got %s", svc.TotalCalls)
	}
	if svc.TotalRevenue != "1000000" {
		t.Errorf("expected TotalRevenue=1000000, got %s", svc.TotalRevenue)
	}

	// Verify project revenue updated
	project, _ := k.GetProject(ctx, projectId)
	if project.RevenueGenerated != "1000000" {
		t.Errorf("expected RevenueGenerated=1000000, got %s", project.RevenueGenerated)
	}

	// Verify caller was charged
	if mockBK.balances[callerAddr]["ulgm"] != 9000000 {
		t.Errorf("caller balance: expected 9000000, got %d", mockBK.balances[callerAddr]["ulgm"])
	}
}

func TestCallService_NotActive(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	svcResp, _ := srv.DeployService(ctx, &types.MsgDeployService{
		Deployer: founder1, Name: "Svc", PricePerCall: "1000",
	})

	// Service is still in "deploying" status — not active
	_, err := srv.CallService(ctx, &types.MsgCallService{
		Caller: founder1, ServiceId: svcResp.ServiceId,
	})
	if err == nil {
		t.Fatal("expected error for calling non-active service")
	}
}

func TestCallService_FreeService(t *testing.T) {
	k, ctx, mockBK := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	_, serviceId := setupServiceForCall(t, k, ctx, srv, "0")

	callerAddr := testAddr("caller2")
	mockBK.balances[callerAddr] = map[string]int64{"ulgm": 5000000}

	resp, err := srv.CallService(ctx, &types.MsgCallService{
		Caller: callerAddr, ServiceId: serviceId, Payload: []byte("free query"),
	})
	if err != nil {
		t.Fatalf("free service call error: %v", err)
	}
	// Proto response uses Result []byte, not AmountCharged string
	if string(resp.Result) != "0" {
		t.Errorf("expected result '0' for free service, got '%s'", string(resp.Result))
	}

	// Verify metrics still updated
	svc, _ := k.GetService(ctx, serviceId)
	if svc.TotalCalls != "1" {
		t.Errorf("expected TotalCalls=1, got %s", svc.TotalCalls)
	}

	// Verify no balance change
	if mockBK.balances[callerAddr]["ulgm"] != 5000000 {
		t.Errorf("caller balance should be unchanged: got %d", mockBK.balances[callerAddr]["ulgm"])
	}
}

// ======================== Bounty Payout Tests ========================

func TestBountyPayout(t *testing.T) {
	k, ctx, mockBK := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Create project
	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "Bounty Project", Domain: "math", Budget: "10000000",
	})

	// Add a task with bounty
	taskResp, _ := srv.AddTask(ctx, &types.MsgAddTask{
		Creator:   founder1,
		ProjectId: projResp.ProjectId,
		Title:     "Bounty Task",
		Bounty:    "500000",
	})

	// Fund the module account so it can pay out
	mockBK.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 1000000}

	// Walk task through lifecycle
	srv.AssignTask(ctx, &types.MsgAssignTask{Assigner: founder1, TaskId: taskResp.TaskId, Assignee: agent1})
	srv.StartWork(ctx, &types.MsgStartWork{Worker: agent1, TaskId: taskResp.TaskId})
	srv.SubmitDeliverable(ctx, &types.MsgSubmitDeliverable{
		Worker: agent1, TaskId: taskResp.TaskId,
		DeliverableHash: "bounty123",
	})

	// Approve — should trigger bounty payout
	_, err := srv.ApproveDeliverable(ctx, &types.MsgApproveDeliverable{
		Approver: founder1, TaskId: taskResp.TaskId,
	})
	if err != nil {
		t.Fatalf("approve error: %v", err)
	}

	// Verify assignee received bounty
	if mockBK.balances[agent1]["ulgm"] != 500000 {
		t.Errorf("agent1 balance: expected 500000, got %d", mockBK.balances[agent1]["ulgm"])
	}

	// Verify module account was debited
	if mockBK.moduleBalances[types.ModuleName]["ulgm"] != 500000 {
		t.Errorf("module balance: expected 500000, got %d", mockBK.moduleBalances[types.ModuleName]["ulgm"])
	}
}

func TestBountyPayout_ZeroBounty(t *testing.T) {
	k, ctx, mockBK := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Title: "No Bounty Project", Domain: "math", Budget: "10000000",
	})
	taskResp, _ := srv.AddTask(ctx, &types.MsgAddTask{
		Creator: founder1, ProjectId: projResp.ProjectId, Title: "Free Task",
		Bounty: "0",
	})

	srv.AssignTask(ctx, &types.MsgAssignTask{Assigner: founder1, TaskId: taskResp.TaskId, Assignee: agent1})
	srv.StartWork(ctx, &types.MsgStartWork{Worker: agent1, TaskId: taskResp.TaskId})
	srv.SubmitDeliverable(ctx, &types.MsgSubmitDeliverable{
		Worker: agent1, TaskId: taskResp.TaskId,
		DeliverableHash: "free123",
	})

	_, err := srv.ApproveDeliverable(ctx, &types.MsgApproveDeliverable{
		Approver: founder1, TaskId: taskResp.TaskId,
	})
	if err != nil {
		t.Fatalf("approve error: %v", err)
	}

	// No bounty should be paid
	if mockBK.balances[agent1] != nil && mockBK.balances[agent1]["ulgm"] != 0 {
		t.Errorf("agent1 should have no balance for zero bounty, got %d", mockBK.balances[agent1]["ulgm"])
	}
}
