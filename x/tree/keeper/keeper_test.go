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

	k := keeper.NewKeeper(cdc, storeKey, mockBK, "lgm1authority")

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
		Phase:           types.PhaseSeed,
		Founder:         founder1,
		KnowledgeDomain: "mathematics",
		Budget:          "1000000",
		Spent:           "0",
		TaskIds:         []string{},
		ServiceIds:      []string{},
		Contributors:    []types.ContributorRecord{},
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
	k.SetProject(ctx, types.ProductProject{Id: "p1", Founder: founder1, KnowledgeDomain: "math", TaskIds: []string{}, ServiceIds: []string{}, Contributors: []types.ContributorRecord{}})
	k.SetProject(ctx, types.ProductProject{Id: "p2", Founder: founder1, KnowledgeDomain: "physics", TaskIds: []string{}, ServiceIds: []string{}, Contributors: []types.ContributorRecord{}})
	k.SetProject(ctx, types.ProductProject{Id: "p3", Founder: founder2, KnowledgeDomain: "cs", TaskIds: []string{}, ServiceIds: []string{}, Contributors: []types.ContributorRecord{}})

	projects := k.GetProjectsByFounder(ctx, founder1)
	if len(projects) != 2 {
		t.Errorf("expected 2 projects for founder1, got %d", len(projects))
	}
}

func TestDeleteProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	project := types.ProductProject{Id: "p1", Founder: founder1, KnowledgeDomain: "math", TaskIds: []string{}, ServiceIds: []string{}, Contributors: []types.ContributorRecord{}}
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
		Status:    types.TaskOpen,
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
	k.SetTask(ctx, types.ProjectTask{Id: "t1", ProjectId: "proj-1", Status: types.TaskOpen, Reviewers: []string{}})
	k.SetTask(ctx, types.ProjectTask{Id: "t2", ProjectId: "proj-1", Status: types.TaskOpen, Reviewers: []string{}})
	k.SetTask(ctx, types.ProjectTask{Id: "t3", ProjectId: "proj-2", Status: types.TaskOpen, Reviewers: []string{}})

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
		Status:          types.SeedDetected,
		ExpiresAtBlock:  200,
	}
	k.SetSeed(ctx, seed)

	got, found := k.GetSeed(ctx, "seed-1")
	if !found {
		t.Fatal("seed not found")
	}
	if got.Status != types.SeedDetected {
		t.Errorf("expected status 'detected', got '%s'", got.Status)
	}
}

func TestGetExpiredSeeds(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetSeed(ctx, types.OpportunitySeed{Id: "s1", KnowledgeDomain: "math", Status: types.SeedDetected, ExpiresAtBlock: 50})
	k.SetSeed(ctx, types.OpportunitySeed{Id: "s2", KnowledgeDomain: "math", Status: types.SeedDetected, ExpiresAtBlock: 150})
	k.SetSeed(ctx, types.OpportunitySeed{Id: "s3", KnowledgeDomain: "math", Status: types.SeedExpired, ExpiresAtBlock: 50}) // already expired

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
		Status:    types.ServiceActive,
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
		Creator:         founder1,
		Name:            "My Project",
		Description:     "A test project",
		KnowledgeDomain: "mathematics",
		Budget:          "1000000",
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
	if project.Phase != types.PhaseSeed {
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
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	_, err := srv.ProposeProject(ctx, &types.MsgProposeProject{
		Proposer: founder1, ProjectId: resp.ProjectId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := k.GetProject(ctx, resp.ProjectId)
	if project.Phase != types.PhaseSprout {
		t.Errorf("expected phase 'sprout', got '%s'", project.Phase)
	}
}

func TestProposeProject_NotFounder(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
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
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})
	srv.ProposeProject(ctx, &types.MsgProposeProject{Proposer: founder1, ProjectId: resp.ProjectId})

	_, err := srv.StartDevelopment(ctx, &types.MsgStartDevelopment{
		Sender: founder1, ProjectId: resp.ProjectId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := k.GetProject(ctx, resp.ProjectId)
	if project.Phase != types.PhaseGrowing {
		t.Errorf("expected phase 'growing', got '%s'", project.Phase)
	}
}

func TestStartDevelopment_InvalidPhase(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	// Try to start from seed (should fail — need sprout first)
	_, err := srv.StartDevelopment(ctx, &types.MsgStartDevelopment{
		Sender: founder1, ProjectId: resp.ProjectId,
	})
	if err == nil {
		t.Fatal("expected error for invalid phase transition seed->growing")
	}
}

func TestPauseAndResumeProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})
	srv.ProposeProject(ctx, &types.MsgProposeProject{Proposer: founder1, ProjectId: resp.ProjectId})
	srv.StartDevelopment(ctx, &types.MsgStartDevelopment{Sender: founder1, ProjectId: resp.ProjectId})

	// Pause
	_, err := srv.PauseProject(ctx, &types.MsgPauseProject{
		Sender: founder1, ProjectId: resp.ProjectId, Reason: "funding issue",
	})
	if err != nil {
		t.Fatalf("unexpected pause error: %v", err)
	}

	project, _ := k.GetProject(ctx, resp.ProjectId)
	if project.Phase != types.PhaseDormant {
		t.Errorf("expected dormant, got %s", project.Phase)
	}

	// Resume
	_, err = srv.ResumeProject(ctx, &types.MsgResumeProject{
		Sender: founder1, ProjectId: resp.ProjectId,
	})
	if err != nil {
		t.Fatalf("unexpected resume error: %v", err)
	}

	project, _ = k.GetProject(ctx, resp.ProjectId)
	if project.Phase != types.PhaseGrowing {
		t.Errorf("expected growing after resume, got %s", project.Phase)
	}
}

func TestAbandonProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	resp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	_, err := srv.AbandonProject(ctx, &types.MsgAbandonProject{
		Sender: founder1, ProjectId: resp.ProjectId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := k.GetProject(ctx, resp.ProjectId)
	if project.Phase != types.PhaseWithered {
		t.Errorf("expected withered, got %s", project.Phase)
	}
}

func TestSpawnChildProject(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Create parent and advance to fruiting
	parentResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Parent", KnowledgeDomain: "math", Budget: "1000000",
	})
	srv.ProposeProject(ctx, &types.MsgProposeProject{Proposer: founder1, ProjectId: parentResp.ProjectId})
	srv.StartDevelopment(ctx, &types.MsgStartDevelopment{Sender: founder1, ProjectId: parentResp.ProjectId})

	// Manually advance to mature then fruiting
	parent, _ := k.GetProject(ctx, parentResp.ProjectId)
	parent.Phase = types.PhaseMature
	k.SetProject(ctx, parent)
	srv.CompleteProject(ctx, &types.MsgCompleteProject{Sender: founder1, ProjectId: parentResp.ProjectId})

	parent, _ = k.GetProject(ctx, parentResp.ProjectId)
	if parent.Phase != types.PhaseFruiting {
		t.Fatalf("expected fruiting, got %s", parent.Phase)
	}

	// Spawn child
	childResp, err := srv.SpawnChildProject(ctx, &types.MsgSpawnChildProject{
		Sender:          founder1,
		ParentProjectId: parentResp.ProjectId,
		Name:            "Child",
		Description:     "A child project",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	child, found := k.GetProject(ctx, childResp.ProjectId)
	if !found {
		t.Fatal("child project not found")
	}
	if child.ParentProjectId != parentResp.ProjectId {
		t.Errorf("expected parent ID %s, got %s", parentResp.ProjectId, child.ParentProjectId)
	}
	if child.Phase != types.PhaseSeed {
		t.Errorf("expected child in seed phase, got %s", child.Phase)
	}

	// Parent should advance to seeding
	parent, _ = k.GetProject(ctx, parentResp.ProjectId)
	if parent.Phase != types.PhaseSeeding {
		t.Errorf("expected parent in seeding phase, got %s", parent.Phase)
	}
}

// ---------- Msg Server Tests: Task Workflow ----------

func TestAddTask(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	taskResp, err := srv.AddTask(ctx, &types.MsgAddTask{
		Sender:      founder1,
		ProjectId:   projResp.ProjectId,
		Title:       "Implement module",
		Description: "Implement the tree module",
		TaskType:    types.TaskImplement,
		BountyAmount: "100000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task, found := k.GetTask(ctx, taskResp.TaskId)
	if !found {
		t.Fatal("task not found")
	}
	if task.Status != types.TaskOpen {
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
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	_, err := srv.AddTask(ctx, &types.MsgAddTask{
		Sender:    agent1,
		ProjectId: projResp.ProjectId,
		Title:     "Task",
		TaskType:  types.TaskImplement,
	})
	if err == nil {
		t.Fatal("expected error for non-contributor adding task")
	}
}

func TestTaskWorkflow_AssignStartSubmitApprove(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	taskResp, _ := srv.AddTask(ctx, &types.MsgAddTask{
		Sender: founder1, ProjectId: projResp.ProjectId, Title: "Task", TaskType: types.TaskImplement,
	})

	// Assign
	_, err := srv.AssignTask(ctx, &types.MsgAssignTask{
		Sender: founder1, TaskId: taskResp.TaskId, Assignee: agent1,
	})
	if err != nil {
		t.Fatalf("assign error: %v", err)
	}
	task, _ := k.GetTask(ctx, taskResp.TaskId)
	if task.Status != types.TaskAssigned {
		t.Errorf("expected assigned, got %s", task.Status)
	}

	// Start work
	_, err = srv.StartWork(ctx, &types.MsgStartWork{
		Assignee: agent1, TaskId: taskResp.TaskId,
	})
	if err != nil {
		t.Fatalf("start work error: %v", err)
	}
	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != types.TaskInProgress {
		t.Errorf("expected in_progress, got %s", task.Status)
	}

	// Submit deliverable
	_, err = srv.SubmitDeliverable(ctx, &types.MsgSubmitDeliverable{
		Assignee: agent1,
		TaskId:   taskResp.TaskId,
		Deliverable: types.TaskDeliverable{
			Type: "code",
			Hash: "abc123",
		},
	})
	if err != nil {
		t.Fatalf("submit deliverable error: %v", err)
	}
	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != types.TaskReview {
		t.Errorf("expected review, got %s", task.Status)
	}

	// Approve
	_, err = srv.ApproveDeliverable(ctx, &types.MsgApproveDeliverable{
		Reviewer: founder1, TaskId: taskResp.TaskId,
	})
	if err != nil {
		t.Fatalf("approve error: %v", err)
	}
	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != types.TaskCompleted {
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
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})
	taskResp, _ := srv.AddTask(ctx, &types.MsgAddTask{
		Sender: founder1, ProjectId: projResp.ProjectId, Title: "Task", TaskType: types.TaskImplement,
	})
	srv.AssignTask(ctx, &types.MsgAssignTask{Sender: founder1, TaskId: taskResp.TaskId, Assignee: agent1})
	srv.StartWork(ctx, &types.MsgStartWork{Assignee: agent1, TaskId: taskResp.TaskId})
	srv.SubmitDeliverable(ctx, &types.MsgSubmitDeliverable{
		Assignee: agent1, TaskId: taskResp.TaskId,
		Deliverable: types.TaskDeliverable{Type: "code", Hash: "abc"},
	})

	// Reject 1: goes back to in_progress
	srv.RejectDeliverable(ctx, &types.MsgRejectDeliverable{
		Reviewer: founder1, TaskId: taskResp.TaskId, Reason: "bad",
	})
	task, _ := k.GetTask(ctx, taskResp.TaskId)
	if task.Status != types.TaskInProgress {
		t.Errorf("after rejection 1: expected in_progress, got %s", task.Status)
	}

	// Resubmit and reject again
	srv.SubmitDeliverable(ctx, &types.MsgSubmitDeliverable{
		Assignee: agent1, TaskId: taskResp.TaskId,
		Deliverable: types.TaskDeliverable{Type: "code", Hash: "def"},
	})
	srv.RejectDeliverable(ctx, &types.MsgRejectDeliverable{
		Reviewer: founder1, TaskId: taskResp.TaskId, Reason: "still bad",
	})
	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != types.TaskInProgress {
		t.Errorf("after rejection 2: expected in_progress, got %s", task.Status)
	}

	// Resubmit and reject 3rd time — should become disputed
	srv.SubmitDeliverable(ctx, &types.MsgSubmitDeliverable{
		Assignee: agent1, TaskId: taskResp.TaskId,
		Deliverable: types.TaskDeliverable{Type: "code", Hash: "ghi"},
	})
	srv.RejectDeliverable(ctx, &types.MsgRejectDeliverable{
		Reviewer: founder1, TaskId: taskResp.TaskId, Reason: "final rejection",
	})
	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != types.TaskDisputed {
		t.Errorf("after rejection 3: expected disputed, got %s", task.Status)
	}
}

func TestReopenTask(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	// Create a task and manually set it to disputed
	taskResp, _ := srv.AddTask(ctx, &types.MsgAddTask{
		Sender: founder1, ProjectId: projResp.ProjectId, Title: "Task", TaskType: types.TaskImplement,
	})
	task, _ := k.GetTask(ctx, taskResp.TaskId)
	task.Status = types.TaskDisputed
	task.Assignee = agent1
	k.SetTask(ctx, task)

	// Reopen
	_, err := srv.ReopenTask(ctx, &types.MsgReopenTask{
		Sender: founder1, TaskId: taskResp.TaskId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task, _ = k.GetTask(ctx, taskResp.TaskId)
	if task.Status != types.TaskOpen {
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
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})
	taskResp, _ := srv.AddTask(ctx, &types.MsgAddTask{
		Sender: founder1, ProjectId: projResp.ProjectId, Title: "Task", TaskType: types.TaskImplement,
	})
	srv.AssignTask(ctx, &types.MsgAssignTask{Sender: founder1, TaskId: taskResp.TaskId, Assignee: agent1})

	_, err := srv.StartWork(ctx, &types.MsgStartWork{
		Assignee: agent2, TaskId: taskResp.TaskId,
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
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	// Apply
	_, err := srv.ApplyToProject(ctx, &types.MsgApplyToProject{
		Applicant:    agent1,
		ProjectId:    projResp.ProjectId,
		ProposedRole: types.RoleDeveloper,
		Message:      "I want to contribute",
	})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	project, _ := k.GetProject(ctx, projResp.ProjectId)
	if len(project.Applications) != 1 {
		t.Fatalf("expected 1 application, got %d", len(project.Applications))
	}

	// Approve application
	_, err = srv.ReviewApplication(ctx, &types.MsgReviewApplication{
		Reviewer:     founder1,
		ProjectId:    projResp.ProjectId,
		ApplicantDid: agent1,
		Approve:      true,
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
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	srv.ApplyToProject(ctx, &types.MsgApplyToProject{
		Applicant: agent1, ProjectId: projResp.ProjectId, ProposedRole: types.RoleDeveloper,
	})

	_, err := srv.ApplyToProject(ctx, &types.MsgApplyToProject{
		Applicant: agent1, ProjectId: projResp.ProjectId, ProposedRole: types.RoleDeveloper,
	})
	if err == nil {
		t.Fatal("expected error for duplicate application")
	}
}

// ---------- Service Tests ----------

func TestDeployService(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	// Advance project to growing phase
	srv.ProposeProject(ctx, &types.MsgProposeProject{Proposer: founder1, ProjectId: projResp.ProjectId})
	srv.StartDevelopment(ctx, &types.MsgStartDevelopment{Sender: founder1, ProjectId: projResp.ProjectId})

	svcResp, err := srv.DeployService(ctx, &types.MsgDeployService{
		Deployer:    founder1,
		ProjectId:   projResp.ProjectId,
		Name:        "Math API",
		Description: "A math computation API",
		ServiceType: types.ServiceAPI,
	})
	if err != nil {
		t.Fatalf("deploy error: %v", err)
	}

	svc, found := k.GetService(ctx, svcResp.ServiceId)
	if !found {
		t.Fatal("service not found")
	}
	if svc.Status != types.ServiceDeploying {
		t.Errorf("expected status deploying, got %s", svc.Status)
	}
}

func TestDeployService_InvalidPhase(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})

	// Try to deploy in seed phase
	_, err := srv.DeployService(ctx, &types.MsgDeployService{
		Deployer: founder1, ProjectId: projResp.ProjectId, Name: "Svc", ServiceType: types.ServiceAPI,
	})
	if err == nil {
		t.Fatal("expected error for deploying in seed phase")
	}
}

func TestPauseResumeRetireService(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	projResp, _ := srv.CreateProject(ctx, &types.MsgCreateProject{
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})
	srv.ProposeProject(ctx, &types.MsgProposeProject{Proposer: founder1, ProjectId: projResp.ProjectId})
	srv.StartDevelopment(ctx, &types.MsgStartDevelopment{Sender: founder1, ProjectId: projResp.ProjectId})

	svcResp, _ := srv.DeployService(ctx, &types.MsgDeployService{
		Deployer: founder1, ProjectId: projResp.ProjectId, Name: "Svc", ServiceType: types.ServiceAPI,
	})

	// Pause
	_, err := srv.PauseService(ctx, &types.MsgPauseService{Sender: founder1, ServiceId: svcResp.ServiceId})
	if err != nil {
		t.Fatalf("pause error: %v", err)
	}
	svc, _ := k.GetService(ctx, svcResp.ServiceId)
	if svc.Status != types.ServicePaused {
		t.Errorf("expected paused, got %s", svc.Status)
	}

	// Resume
	_, err = srv.ResumeService(ctx, &types.MsgResumeService{Sender: founder1, ServiceId: svcResp.ServiceId})
	if err != nil {
		t.Fatalf("resume error: %v", err)
	}
	svc, _ = k.GetService(ctx, svcResp.ServiceId)
	if svc.Status != types.ServiceActive {
		t.Errorf("expected active, got %s", svc.Status)
	}

	// Retire
	_, err = srv.RetireService(ctx, &types.MsgRetireService{Sender: founder1, ServiceId: svcResp.ServiceId})
	if err != nil {
		t.Fatalf("retire error: %v", err)
	}
	svc, _ = k.GetService(ctx, svcResp.ServiceId)
	if svc.Status != types.ServiceRetired {
		t.Errorf("expected retired, got %s", svc.Status)
	}

	// Can't retire again
	_, err = srv.RetireService(ctx, &types.MsgRetireService{Sender: founder1, ServiceId: svcResp.ServiceId})
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
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
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

	msgSrv.CreateProject(ctx, &types.MsgCreateProject{Creator: founder1, Name: "P1", KnowledgeDomain: "math", Budget: "1000000"})
	msgSrv.CreateProject(ctx, &types.MsgCreateProject{Creator: founder1, Name: "P2", KnowledgeDomain: "physics", Budget: "2000000"})

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
		Creator: founder1, Name: "Project", KnowledgeDomain: "math", Budget: "1000000",
	})
	msgSrv.AddTask(ctx, &types.MsgAddTask{Sender: founder1, ProjectId: projResp.ProjectId, Title: "T1", TaskType: types.TaskImplement})
	msgSrv.AddTask(ctx, &types.MsgAddTask{Sender: founder1, ProjectId: projResp.ProjectId, Title: "T2", TaskType: types.TaskTest})

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
		Creator: founder1, Name: "P1", KnowledgeDomain: "math", Budget: "1000000",
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
