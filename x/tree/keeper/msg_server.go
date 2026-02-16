package keeper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tree/types"
)

type msgServer struct {
	Keeper
	types.UnimplementedMsgServer
}

// NewMsgServerImpl returns a MsgServer implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// ======================== Project Messages ========================

// CreateProject creates a new project in seed phase.
// Proto fields: Creator, Title, Description, Domain, Budget, RequiredCapabilities
func (m msgServer) CreateProject(goCtx context.Context, msg *types.MsgCreateProject) (*types.MsgCreateProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentBlock := uint64(ctx.BlockHeight())

	counter := m.Keeper.GetNextProjectId(ctx)
	projectId := fmt.Sprintf("proj-%d-%d", currentBlock, counter)

	project := types.ProductProject{
		Id:                  projectId,
		Name:                msg.Title,
		Description:         msg.Description,
		Phase:               string(types.PhaseSeed),
		CreatedAtBlock:      currentBlock,
		PhaseChangedAtBlock: currentBlock,
		KnowledgeDomain:     msg.Domain,
		Founder:             msg.Creator,
		Contributors: []*types.ContributorRecord{
			{
				Did:           msg.Creator,
				Role:          string(types.RoleFounder),
				JoinedAtBlock: currentBlock,
			},
		},
		Budget:     msg.Budget,
		Spent:      "0",
		TaskIds:    []string{},
		ServiceIds: []string{},
	}

	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_create_project",
		sdk.NewAttribute("project_id", projectId),
		sdk.NewAttribute("founder", msg.Creator),
		sdk.NewAttribute("domain", msg.Domain),
	))

	return &types.MsgCreateProjectResponse{ProjectId: projectId}, nil
}

// ProposeProject advances a project from seed to sprout.
func (m msgServer) ProposeProject(goCtx context.Context, msg *types.MsgProposeProject) (*types.MsgProposeProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Proposer {
		return nil, types.ErrNotFounder
	}
	if !IsValidPhaseTransition(types.ProjectPhase(project.Phase), types.PhaseSprout) {
		return nil, types.ErrInvalidPhaseTransition
	}

	project.Phase = string(types.PhaseSprout)
	project.PreviousPhase = string(types.PhaseSeed)
	project.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_propose_project",
		sdk.NewAttribute("project_id", msg.ProjectId),
	))

	return &types.MsgProposeProjectResponse{}, nil
}

// StartDevelopment advances a project from sprout to growing.
// Proto field: Authority (was Sender)
func (m msgServer) StartDevelopment(goCtx context.Context, msg *types.MsgStartDevelopment) (*types.MsgStartDevelopmentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Authority {
		return nil, types.ErrNotFounder
	}
	if !IsValidPhaseTransition(types.ProjectPhase(project.Phase), types.PhaseGrowing) {
		return nil, types.ErrInvalidPhaseTransition
	}

	// Check min contributors
	params := m.Keeper.GetParams(ctx)
	if uint32(len(project.Contributors)) < params.MinContributorsToStart {
		return nil, fmt.Errorf("need at least %d contributors to start development", params.MinContributorsToStart)
	}

	project.Phase = string(types.PhaseGrowing)
	project.PreviousPhase = string(types.PhaseSprout)
	project.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_start_development",
		sdk.NewAttribute("project_id", msg.ProjectId),
	))

	return &types.MsgStartDevelopmentResponse{}, nil
}

// CompleteProject advances a project from growing to mature phase.
// Proto field: Authority (was Sender)
func (m msgServer) CompleteProject(goCtx context.Context, msg *types.MsgCompleteProject) (*types.MsgCompleteProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Authority {
		return nil, types.ErrNotFounder
	}
	if !IsValidPhaseTransition(types.ProjectPhase(project.Phase), types.PhaseMature) {
		return nil, types.ErrInvalidPhaseTransition
	}

	project.PreviousPhase = project.Phase
	project.Phase = string(types.PhaseMature)
	project.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_complete_project",
		sdk.NewAttribute("project_id", msg.ProjectId),
	))

	return &types.MsgCompleteProjectResponse{}, nil
}

// DeployProject advances a mature project to fruiting phase (requires at least one service).
// NOTE: MsgDeployProject is NOT in the proto-generated set (proto has no DeployProject).
// We keep this as a no-op stub or handle via a different mechanism.
// For now, we use Authority field since the hand-written type is removed.
// Actually, MsgDeployProject is NOT in tx.pb.go. We need to check if it's still referenced.
// Since it's not proto-generated, we skip this method. The proto MsgServer doesn't include it.

// PauseProject moves a project to dormant phase.
// Proto field: Authority (was Sender)
func (m msgServer) PauseProject(goCtx context.Context, msg *types.MsgPauseProject) (*types.MsgPauseProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Authority {
		return nil, types.ErrNotFounder
	}
	if !IsValidPhaseTransition(types.ProjectPhase(project.Phase), types.PhaseDormant) {
		return nil, types.ErrInvalidPhaseTransition
	}

	project.PreviousPhase = project.Phase
	project.Phase = string(types.PhaseDormant)
	project.PauseReason = msg.Reason
	project.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_pause_project",
		sdk.NewAttribute("project_id", msg.ProjectId),
		sdk.NewAttribute("reason", msg.Reason),
	))

	return &types.MsgPauseProjectResponse{}, nil
}

// ResumeProject resumes a dormant project to its previous phase.
// Proto field: Authority (was Sender)
func (m msgServer) ResumeProject(goCtx context.Context, msg *types.MsgResumeProject) (*types.MsgResumeProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Authority {
		return nil, types.ErrNotFounder
	}
	if project.Phase != string(types.PhaseDormant) {
		return nil, types.ErrInvalidPhaseTransition
	}

	// Resume to previous phase if valid
	resumeTo := project.PreviousPhase
	if resumeTo == "" || resumeTo == string(types.PhaseDormant) {
		resumeTo = string(types.PhaseGrowing)
	}
	if !IsValidPhaseTransition(types.ProjectPhase(project.Phase), types.ProjectPhase(resumeTo)) {
		return nil, types.ErrInvalidPhaseTransition
	}

	project.PreviousPhase = string(types.PhaseDormant)
	project.Phase = resumeTo
	project.PauseReason = ""
	project.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_resume_project",
		sdk.NewAttribute("project_id", msg.ProjectId),
		sdk.NewAttribute("resumed_to", string(resumeTo)),
	))

	return &types.MsgResumeProjectResponse{}, nil
}

// AbandonProject moves a project to withered phase.
// Proto field: Authority (was Sender)
func (m msgServer) AbandonProject(goCtx context.Context, msg *types.MsgAbandonProject) (*types.MsgAbandonProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Authority {
		return nil, types.ErrNotFounder
	}
	if !IsValidPhaseTransition(types.ProjectPhase(project.Phase), types.PhaseWithered) {
		return nil, types.ErrInvalidPhaseTransition
	}

	project.PreviousPhase = project.Phase
	project.Phase = string(types.PhaseWithered)
	project.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_abandon_project",
		sdk.NewAttribute("project_id", msg.ProjectId),
	))

	return &types.MsgAbandonProjectResponse{}, nil
}

// SpawnChildProject creates a child project from a fruiting/seeding parent.
// Proto field: Creator (was Sender)
func (m msgServer) SpawnChildProject(goCtx context.Context, msg *types.MsgSpawnChildProject) (*types.MsgSpawnChildProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	parent, found := m.Keeper.GetProject(ctx, msg.ParentProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(parent, msg.Creator) {
		return nil, types.ErrNotContributor
	}
	// Parent must be in fruiting or seeding phase
	if parent.Phase != string(types.PhaseFruiting) && parent.Phase != string(types.PhaseSeeding) {
		return nil, types.ErrInvalidPhaseTransition
	}

	currentBlock := uint64(ctx.BlockHeight())
	counter := m.Keeper.GetNextProjectId(ctx)
	childId := fmt.Sprintf("proj-%d-%d", currentBlock, counter)

	child := types.ProductProject{
		Id:                  childId,
		Name:                msg.Title,
		Description:         msg.Description,
		Phase:               string(types.PhaseSeed),
		CreatedAtBlock:      currentBlock,
		PhaseChangedAtBlock: currentBlock,
		KnowledgeDomain:     parent.KnowledgeDomain,
		ParentProjectId:     msg.ParentProjectId,
		Founder:             msg.Creator,
		Contributors: []*types.ContributorRecord{
			{
				Did:           msg.Creator,
				Role:          string(types.RoleFounder),
				JoinedAtBlock: currentBlock,
			},
		},
		Budget:     msg.Budget,
		Spent:      "0",
		TaskIds:    []string{},
		ServiceIds: []string{},
	}

	m.Keeper.SetProject(ctx, child)

	// Advance parent to seeding if it was fruiting
	if parent.Phase == string(types.PhaseFruiting) {
		parent.PreviousPhase = string(types.PhaseFruiting)
		parent.Phase = string(types.PhaseSeeding)
		parent.PhaseChangedAtBlock = currentBlock
		m.Keeper.SetProject(ctx, parent)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_spawn_child_project",
		sdk.NewAttribute("child_id", childId),
		sdk.NewAttribute("parent_id", msg.ParentProjectId),
	))

	return &types.MsgSpawnChildProjectResponse{ChildProjectId: childId}, nil
}

// ======================== Task Messages ========================

// AddTask adds a new task to a project.
// Proto field: Creator (was Sender), Bounty (was BountyAmount)
func (m msgServer) AddTask(goCtx context.Context, msg *types.MsgAddTask) (*types.MsgAddTaskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Creator) {
		return nil, types.ErrNotContributor
	}

	params := m.Keeper.GetParams(ctx)
	if project.TotalTasks >= params.MaxTasksPerProject {
		return nil, types.ErrMaxTasksReached
	}

	currentBlock := uint64(ctx.BlockHeight())
	counter := m.Keeper.GetNextTaskId(ctx)
	taskId := fmt.Sprintf("task-%d-%d", currentBlock, counter)

	task := types.ProjectTask{
		Id:                   taskId,
		ProjectId:            msg.ProjectId,
		Title:                msg.Title,
		Description:          msg.Description,
		RequiredCapabilities: msg.RequiredCapabilities,
		Status:               string(types.TaskOpen),
		CreatedAtBlock:       currentBlock,
		BountyAmount:         msg.Bounty,
		Reviewers:            []string{},
	}

	m.Keeper.SetTask(ctx, task)

	// Update project task tracking
	project.TaskIds = append(project.TaskIds, taskId)
	project.TotalTasks++
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_add_task",
		sdk.NewAttribute("task_id", taskId),
		sdk.NewAttribute("project_id", msg.ProjectId),
	))

	return &types.MsgAddTaskResponse{TaskId: taskId}, nil
}

// AssignTask assigns a task to an agent.
// Proto field: Assigner (was Sender)
func (m msgServer) AssignTask(goCtx context.Context, msg *types.MsgAssignTask) (*types.MsgAssignTaskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.Keeper.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}

	// Verify sender is founder or contributor of the project
	project, found := m.Keeper.GetProject(ctx, task.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Assigner) {
		return nil, types.ErrNotContributor
	}

	if task.Status != string(types.TaskOpen) {
		return nil, types.ErrInvalidTaskState
	}

	task.Assignee = msg.Assignee
	task.Status = string(types.TaskAssigned)
	m.Keeper.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_assign_task",
		sdk.NewAttribute("task_id", msg.TaskId),
		sdk.NewAttribute("assignee", msg.Assignee),
	))

	return &types.MsgAssignTaskResponse{}, nil
}

// StartWork moves a task from assigned to in_progress.
// Proto field: Worker (was Assignee)
func (m msgServer) StartWork(goCtx context.Context, msg *types.MsgStartWork) (*types.MsgStartWorkResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.Keeper.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}
	if task.Assignee != msg.Worker {
		return nil, types.ErrNotAssignee
	}
	if task.Status != string(types.TaskAssigned) {
		return nil, types.ErrInvalidTaskState
	}

	task.Status = string(types.TaskInProgress)
	m.Keeper.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_start_work",
		sdk.NewAttribute("task_id", msg.TaskId),
	))

	return &types.MsgStartWorkResponse{}, nil
}

// SubmitDeliverable submits work for review.
// Proto fields: Worker (was Assignee), DeliverableHash, DeliverableUri (was Deliverable struct)
func (m msgServer) SubmitDeliverable(goCtx context.Context, msg *types.MsgSubmitDeliverable) (*types.MsgSubmitDeliverableResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.Keeper.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}
	if task.Assignee != msg.Worker {
		return nil, types.ErrNotAssignee
	}
	if task.Status != string(types.TaskInProgress) {
		return nil, types.ErrInvalidTaskState
	}

	deliverable := types.TaskDeliverable{
		Hash:             msg.DeliverableHash,
		SubmittedAtBlock: uint64(ctx.BlockHeight()),
	}
	task.Deliverable = &deliverable
	task.Status = string(types.TaskReview)
	m.Keeper.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_submit_deliverable",
		sdk.NewAttribute("task_id", msg.TaskId),
		sdk.NewAttribute("hash", msg.DeliverableHash),
	))

	return &types.MsgSubmitDeliverableResponse{}, nil
}

// ApproveDeliverable approves a task deliverable.
// Proto field: Approver (was Reviewer)
func (m msgServer) ApproveDeliverable(goCtx context.Context, msg *types.MsgApproveDeliverable) (*types.MsgApproveDeliverableResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.Keeper.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}
	if task.Status != string(types.TaskReview) {
		return nil, types.ErrInvalidTaskState
	}

	// Reviewer cannot be the assignee (prevent self-review)
	if msg.Approver == task.Assignee {
		return nil, types.ErrSelfReview
	}

	// Reviewer must be a project contributor
	project, found := m.Keeper.GetProject(ctx, task.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Approver) {
		return nil, types.ErrNotContributor
	}

	task.Status = string(types.TaskCompleted)
	task.CompletedAtBlock = uint64(ctx.BlockHeight())
	if task.Deliverable != nil {
		task.Deliverable.VerifiedBy = append(task.Deliverable.VerifiedBy, msg.Approver)
	}
	m.Keeper.SetTask(ctx, task)

	// Update project completed tasks count
	project.CompletedTasks++

	// Pay bounty if task has one
	if task.BountyAmount != "" && task.BountyAmount != "0" {
		bounty, ok := math.NewIntFromString(task.BountyAmount)
		if ok && bounty.IsPositive() {
			assigneeAddr, addrErr := sdk.AccAddressFromBech32(task.Assignee)
			if addrErr == nil {
				coins := sdk.NewCoins(sdk.NewCoin("ulgm", bounty))
				err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, assigneeAddr, coins)
				if err != nil {
					return nil, fmt.Errorf("bounty payout failed: %w", err)
				}

				// Update contributor's TasksCompleted and TotalEarned
				for i, c := range project.Contributors {
					if c.Did == task.Assignee {
						project.Contributors[i].TasksCompleted++
						existing, _ := strconv.ParseInt(project.Contributors[i].TotalEarned, 10, 64)
						project.Contributors[i].TotalEarned = strconv.FormatInt(existing+bounty.Int64(), 10)
						break
					}
				}
			}
		}
	}

	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_approve_deliverable",
		sdk.NewAttribute("task_id", msg.TaskId),
		sdk.NewAttribute("approver", msg.Approver),
	))

	return &types.MsgApproveDeliverableResponse{}, nil
}

// RejectDeliverable rejects a task deliverable.
// Proto field: Rejector (was Reviewer)
func (m msgServer) RejectDeliverable(goCtx context.Context, msg *types.MsgRejectDeliverable) (*types.MsgRejectDeliverableResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.Keeper.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}
	if task.Status != string(types.TaskReview) {
		return nil, types.ErrInvalidTaskState
	}

	project, found := m.Keeper.GetProject(ctx, task.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Rejector) {
		return nil, types.ErrNotContributor
	}

	params := m.Keeper.GetParams(ctx)
	task.RejectionCount++

	if task.RejectionCount >= params.MaxRejections {
		task.Status = string(types.TaskDisputed)
	} else {
		// Send back to in_progress for rework
		task.Status = string(types.TaskInProgress)
	}
	m.Keeper.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_reject_deliverable",
		sdk.NewAttribute("task_id", msg.TaskId),
		sdk.NewAttribute("reason", msg.Reason),
	))

	return &types.MsgRejectDeliverableResponse{}, nil
}

// ReopenTask reopens a rejected or disputed task.
// Proto field: Authority (was Sender)
func (m msgServer) ReopenTask(goCtx context.Context, msg *types.MsgReopenTask) (*types.MsgReopenTaskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.Keeper.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}

	project, found := m.Keeper.GetProject(ctx, task.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Authority {
		return nil, types.ErrNotFounder
	}

	if task.Status != string(types.TaskRejected) && task.Status != string(types.TaskDisputed) {
		return nil, types.ErrInvalidTaskState
	}

	task.Status = string(types.TaskOpen)
	task.Assignee = ""
	task.Deliverable = nil
	task.RejectionCount = 0
	m.Keeper.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_reopen_task",
		sdk.NewAttribute("task_id", msg.TaskId),
	))

	return &types.MsgReopenTaskResponse{}, nil
}

// ======================== Application Messages ========================

// ApplyToProject submits an application to join a project.
// Proto fields: Applicant, ProjectId, Role (was ProposedRole), Pitch (was Message), Capabilities
func (m msgServer) ApplyToProject(goCtx context.Context, msg *types.MsgApplyToProject) (*types.MsgApplyToProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}

	params := m.Keeper.GetParams(ctx)
	if uint32(len(project.Applications)) >= params.MaxApplications {
		return nil, types.ErrMaxApplicationsReached
	}

	// Check for duplicate application
	for _, app := range project.Applications {
		if app.Did == msg.Applicant && app.Status == string(types.AppPending) {
			return nil, types.ErrAlreadyApplied
		}
	}

	project.Applications = append(project.Applications, &types.ProjectApplication{
		Did:            msg.Applicant,
		ProposedRole:   msg.Role,
		Message:        msg.Pitch,
		Capabilities:   msg.Capabilities,
		AppliedAtBlock: uint64(ctx.BlockHeight()),
		Status:         string(types.AppPending),
	})
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_apply_to_project",
		sdk.NewAttribute("project_id", msg.ProjectId),
		sdk.NewAttribute("applicant", msg.Applicant),
	))

	return &types.MsgApplyToProjectResponse{}, nil
}

// ReviewApplication approves or rejects a project application.
// Proto fields: Reviewer, ApplicationId (was ProjectId+ApplicantDid), Accepted (was Approve), Reason
func (m msgServer) ReviewApplication(goCtx context.Context, msg *types.MsgReviewApplication) (*types.MsgReviewApplicationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// The proto uses ApplicationId instead of ProjectId + ApplicantDid.
	// We need to search all projects for the application. For backward compat,
	// we iterate projects to find the matching application.
	// In a production implementation, we'd have an application index.
	// For now, we search all projects since ApplicationId format is not yet established.
	// This is a temporary approach until application indexing is added.

	// Try to find application by iterating projects
	var targetProject *types.ProductProject
	var appIndex int = -1
	var projectId string

	m.Keeper.IterateProjects(ctx, func(p types.ProductProject) bool {
		for i, app := range p.Applications {
			if app.Did == msg.ApplicationId && app.Status == string(types.AppPending) {
				proj := p
				targetProject = &proj
				appIndex = i
				projectId = p.Id
				return true
			}
		}
		return false
	})

	if targetProject == nil || appIndex < 0 {
		return nil, types.ErrApplicationNotFound
	}
	if targetProject.Founder != msg.Reviewer {
		return nil, types.ErrNotFounder
	}

	if msg.Accepted {
		targetProject.Applications[appIndex].Status = string(types.AppApproved)

		// Check max contributors
		params := m.Keeper.GetParams(ctx)
		if uint32(len(targetProject.Contributors)) >= params.MaxContributors {
			return nil, types.ErrMaxContributorsReached
		}

		// Add as contributor
		app := targetProject.Applications[appIndex]
		targetProject.Contributors = append(targetProject.Contributors, &types.ContributorRecord{
			Did:           app.Did,
			Role:          app.ProposedRole,
			JoinedAtBlock: uint64(ctx.BlockHeight()),
		})
	} else {
		targetProject.Applications[appIndex].Status = string(types.AppRejected)
	}

	m.Keeper.SetProject(ctx, *targetProject)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_review_application",
		sdk.NewAttribute("project_id", projectId),
		sdk.NewAttribute("applicant", msg.ApplicationId),
		sdk.NewAttribute("accepted", fmt.Sprintf("%v", msg.Accepted)),
	))

	return &types.MsgReviewApplicationResponse{}, nil
}

// SetAvailability records an agent's availability for project work.
func (m msgServer) SetAvailability(goCtx context.Context, msg *types.MsgSetAvailability) (*types.MsgSetAvailabilityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_set_availability",
		sdk.NewAttribute("agent", msg.Agent),
		sdk.NewAttribute("available", fmt.Sprintf("%v", msg.Available)),
	))

	return &types.MsgSetAvailabilityResponse{}, nil
}

// ======================== Service Messages ========================

// DeployService deploys a new service from a project.
// Proto: Deployer, Name, Description, Endpoint, PricePerCall, SubscriptionPrice, Capabilities
// Note: proto MsgDeployService doesn't have ProjectId, ServiceType, or ContractAddress.
// We use Deployer to find the project or use the service without project linkage.
func (m msgServer) DeployService(goCtx context.Context, msg *types.MsgDeployService) (*types.MsgDeployServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	currentBlock := uint64(ctx.BlockHeight())
	counter := m.Keeper.GetNextServiceId(ctx)
	serviceId := fmt.Sprintf("svc-%d-%d", currentBlock, counter)

	pricePerCall := msg.PricePerCall
	if pricePerCall == "" {
		pricePerCall = "0"
	}

	service := types.ServiceLeaf{
		Id:              serviceId,
		Name:            msg.Name,
		Description:     msg.Description,
		ContractAddress: msg.Endpoint,
		TotalCalls:      "0",
		TotalRevenue:    "0",
		UptimeBlocks:    "0",
		Status:          string(types.ServiceDeploying),
		DeployedAtBlock: currentBlock,
		PricePerCall:    pricePerCall,
	}

	m.Keeper.SetService(ctx, service)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_deploy_service",
		sdk.NewAttribute("service_id", serviceId),
		sdk.NewAttribute("deployer", msg.Deployer),
	))

	return &types.MsgDeployServiceResponse{ServiceId: serviceId}, nil
}

// PauseService pauses an active service.
// Proto field: Owner (was Sender)
func (m msgServer) PauseService(goCtx context.Context, msg *types.MsgPauseService) (*types.MsgPauseServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	service, found := m.Keeper.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, types.ErrServiceNotFound
	}

	if service.Status != string(types.ServiceActive) && service.Status != string(types.ServiceDeploying) && service.Status != string(types.ServiceDegraded) {
		return nil, types.ErrInvalidServiceState
	}

	service.Status = string(types.ServicePaused)
	m.Keeper.SetService(ctx, service)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_pause_service",
		sdk.NewAttribute("service_id", msg.ServiceId),
	))

	return &types.MsgPauseServiceResponse{}, nil
}

// ResumeService resumes a paused service.
// Proto field: Owner (was Sender)
func (m msgServer) ResumeService(goCtx context.Context, msg *types.MsgResumeService) (*types.MsgResumeServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	service, found := m.Keeper.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, types.ErrServiceNotFound
	}

	if service.Status != string(types.ServicePaused) {
		return nil, types.ErrInvalidServiceState
	}

	service.Status = string(types.ServiceActive)
	m.Keeper.SetService(ctx, service)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_resume_service",
		sdk.NewAttribute("service_id", msg.ServiceId),
	))

	return &types.MsgResumeServiceResponse{}, nil
}

// RetireService permanently retires a service.
// Proto field: Owner (was Sender)
func (m msgServer) RetireService(goCtx context.Context, msg *types.MsgRetireService) (*types.MsgRetireServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	service, found := m.Keeper.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, types.ErrServiceNotFound
	}

	if service.Status == string(types.ServiceRetired) {
		return nil, types.ErrInvalidServiceState
	}

	service.Status = string(types.ServiceRetired)
	m.Keeper.SetService(ctx, service)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_retire_service",
		sdk.NewAttribute("service_id", msg.ServiceId),
	))

	return &types.MsgRetireServiceResponse{}, nil
}

// ======================== CallService ========================

// CallService calls an active service, charging the caller and distributing revenue.
func (m msgServer) CallService(goCtx context.Context, msg *types.MsgCallService) (*types.MsgCallServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 1. Look up service and verify active
	service, found := m.Keeper.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, types.ErrServiceNotFound
	}
	if service.Status != string(types.ServiceActive) {
		return nil, types.ErrServiceNotActive
	}

	// 2. Check for active subscription — subscribers skip per-call payment
	hasSubscription := m.Keeper.HasActiveSubscription(ctx, msg.ServiceId, msg.Caller)

	// 3. Parse price
	price, ok := math.NewIntFromString(service.PricePerCall)
	if !ok {
		price = math.ZeroInt()
	}

	// 4. Determine payment method: subscription, channel, or direct
	channelPayment := false
	if price.IsPositive() && !hasSubscription {
		// Check if MaxFee specifies a payment channel (format: "channel:<channelId>")
		if strings.HasPrefix(msg.MaxFee, "channel:") && m.Keeper.channelsKeeper != nil {
			channelId := strings.TrimPrefix(msg.MaxFee, "channel:")

			// Verify channel: caller must be payer, channel must be open
			payer, _, _, status, chFound := m.Keeper.channelsKeeper.GetChannelInfo(ctx, channelId)
			if !chFound {
				return nil, fmt.Errorf("payment channel %s not found", channelId)
			}
			if status != "open" {
				return nil, fmt.Errorf("payment channel %s is not open (status: %s)", channelId, status)
			}
			if payer != msg.Caller {
				return nil, fmt.Errorf("caller %s is not the payer of channel %s", msg.Caller, channelId)
			}

			// Debit the channel — transfers funds from channels module to tree module
			if err := m.Keeper.channelsKeeper.SpendFromChannel(ctx, channelId, price.String(), types.ModuleName); err != nil {
				return nil, types.ErrInsufficientPayment.Wrap(err.Error())
			}
			channelPayment = true
		}

		if !channelPayment {
			// Direct payment from caller's account
			callerAddr, err := sdk.AccAddressFromBech32(msg.Caller)
			if err != nil {
				return nil, fmt.Errorf("invalid caller address: %w", err)
			}

			paymentCoins := sdk.NewCoins(sdk.NewCoin("ulgm", price))
			err = m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, callerAddr, types.ModuleName, paymentCoins)
			if err != nil {
				return nil, types.ErrInsufficientPayment.Wrap(err.Error())
			}
		}

		// Revenue distribution (same path for both channel and direct payments —
		// funds are now in the tree module account either way)
		project, projFound := m.Keeper.GetProject(ctx, service.ProjectId)
		if projFound {
			params := m.Keeper.GetParams(ctx)

			dist := CalculateRevenue(
				price.Int64(),
				params.EvidenceTaxBp,
				params.ContributorsBp,
				params.ResearchFundBp,
				params.ProtocolTreasuryBp,
				project.Contributors,
			)

			taxRecipients := []struct {
				addr   string
				amount int64
			}{
				{params.ResearchFundAddr, dist.TaxResearch},
				{params.ProtocolTreasuryAddr, dist.TaxFoundation},
				{params.ProtocolTreasuryAddr, dist.TaxCommunity},
				{params.ProtocolTreasuryAddr, dist.TaxDevelopers},
				{params.ProtocolTreasuryAddr, dist.TaxReserve},
			}

			for _, r := range taxRecipients {
				if r.amount > 0 && r.addr != "" {
					recipientAddr, addrErr := sdk.AccAddressFromBech32(r.addr)
					if addrErr == nil {
						coins := sdk.NewCoins(sdk.NewCoin("ulgm", math.NewInt(r.amount)))
						_ = m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, coins)
					}
				}
			}

			for _, share := range dist.ContributorShares {
				if share.Amount > 0 {
					addr, addrErr := sdk.AccAddressFromBech32(share.Address)
					if addrErr == nil {
						coins := sdk.NewCoins(sdk.NewCoin("ulgm", math.NewInt(share.Amount)))
						_ = m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins)
					}
				}
			}

			if dist.ResearchFund > 0 && params.ResearchFundAddr != "" {
				addr, addrErr := sdk.AccAddressFromBech32(params.ResearchFundAddr)
				if addrErr == nil {
					coins := sdk.NewCoins(sdk.NewCoin("ulgm", math.NewInt(dist.ResearchFund)))
					_ = m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins)
				}
			}

			if dist.ProtocolTreasury > 0 && params.ProtocolTreasuryAddr != "" {
				addr, addrErr := sdk.AccAddressFromBech32(params.ProtocolTreasuryAddr)
				if addrErr == nil {
					coins := sdk.NewCoins(sdk.NewCoin("ulgm", math.NewInt(dist.ProtocolTreasury)))
					_ = m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins)
				}
			}

			existingRevenue, _ := strconv.ParseInt(project.RevenueGenerated, 10, 64)
			project.RevenueGenerated = strconv.FormatInt(existingRevenue+price.Int64(), 10)
			m.Keeper.SetProject(ctx, project)
		}
	}

	// 5. Update service metrics
	existingCalls, _ := strconv.ParseInt(service.TotalCalls, 10, 64)
	service.TotalCalls = strconv.FormatInt(existingCalls+1, 10)

	// Revenue only counted when actually paid (not subscription-covered)
	if !hasSubscription {
		existingRevenue, _ := strconv.ParseInt(service.TotalRevenue, 10, 64)
		service.TotalRevenue = strconv.FormatInt(existingRevenue+price.Int64(), 10)
	}

	service.LastCalledBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetService(ctx, service)

	paymentMethod := "direct"
	if hasSubscription {
		paymentMethod = "subscription"
	} else if channelPayment {
		paymentMethod = "channel"
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_call_service",
		sdk.NewAttribute("service_id", msg.ServiceId),
		sdk.NewAttribute("caller", msg.Caller),
		sdk.NewAttribute("amount", price.String()),
		sdk.NewAttribute("payment_method", paymentMethod),
	))

	return &types.MsgCallServiceResponse{
		Result: []byte(price.String()),
	}, nil
}

// ======================== Contributor Messages ========================

// AddContributor directly adds a contributor to a project (founder-only, bypasses application).
// Proto fields: Authority, ProjectId, Contributor, Role
func (m msgServer) AddContributor(goCtx context.Context, msg *types.MsgAddContributor) (*types.MsgAddContributorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Authority {
		return nil, types.ErrNotFounder
	}

	params := m.Keeper.GetParams(ctx)
	if uint32(len(project.Contributors)) >= params.MaxContributors {
		return nil, types.ErrMaxContributorsReached
	}

	// Check for duplicate contributor
	for _, c := range project.Contributors {
		if c.Did == msg.Contributor {
			return nil, types.ErrAlreadyApplied.Wrap("address is already a contributor")
		}
	}

	project.Contributors = append(project.Contributors, &types.ContributorRecord{
		Did:           msg.Contributor,
		Role:          msg.Role,
		JoinedAtBlock: uint64(ctx.BlockHeight()),
	})
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_add_contributor",
		sdk.NewAttribute("project_id", msg.ProjectId),
		sdk.NewAttribute("contributor", msg.Contributor),
		sdk.NewAttribute("role", msg.Role),
	))

	return &types.MsgAddContributorResponse{}, nil
}

// ======================== Subscription Messages ========================

// SubscribeService creates a time-bounded subscription to an active service.
// Proto fields: Subscriber, ServiceId, DurationBlocks
func (m msgServer) SubscribeService(goCtx context.Context, msg *types.MsgSubscribeService) (*types.MsgSubscribeServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	service, found := m.Keeper.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, types.ErrServiceNotFound
	}
	if service.Status != string(types.ServiceActive) {
		return nil, types.ErrServiceNotActive
	}

	currentBlock := uint64(ctx.BlockHeight())
	counter := m.Keeper.GetNextSubscriptionId(ctx)
	subId := fmt.Sprintf("sub-%d-%d", currentBlock, counter)

	sub := types.ServiceSubscription{
		Id:             subId,
		ServiceId:      msg.ServiceId,
		Subscriber:     msg.Subscriber,
		StartBlock:     currentBlock,
		DurationBlocks: msg.DurationBlocks,
		ExpiresAtBlock: currentBlock + msg.DurationBlocks,
	}

	m.Keeper.SetSubscription(ctx, sub)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_subscribe_service",
		sdk.NewAttribute("subscription_id", subId),
		sdk.NewAttribute("service_id", msg.ServiceId),
		sdk.NewAttribute("subscriber", msg.Subscriber),
		sdk.NewAttribute("duration_blocks", strconv.FormatUint(msg.DurationBlocks, 10)),
	))

	return &types.MsgSubscribeServiceResponse{SubscriptionId: subId}, nil
}

// ======================== Seeding Messages ========================

// BeginSeeding transitions a fruiting project to seeding phase and creates an opportunity seed.
// Proto fields: Seeder, ProjectId, Domain
func (m msgServer) BeginSeeding(goCtx context.Context, msg *types.MsgBeginSeeding) (*types.MsgBeginSeedingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Seeder) {
		return nil, types.ErrNotContributor
	}
	if !IsValidPhaseTransition(types.ProjectPhase(project.Phase), types.PhaseSeeding) {
		return nil, types.ErrInvalidPhaseTransition
	}

	currentBlock := uint64(ctx.BlockHeight())

	// Transition project to seeding phase
	project.PreviousPhase = project.Phase
	project.Phase = string(types.PhaseSeeding)
	project.PhaseChangedAtBlock = currentBlock
	m.Keeper.SetProject(ctx, project)

	// Create an opportunity seed in the specified domain
	params := m.Keeper.GetParams(ctx)
	seedCounter := m.Keeper.GetNextSeedId(ctx)
	seedId := fmt.Sprintf("seed-%d-%d", currentBlock, seedCounter)

	domain := msg.Domain
	if domain == "" {
		domain = project.KnowledgeDomain
	}

	seed := types.OpportunitySeed{
		Id:              seedId,
		DetectedAtBlock: currentBlock,
		KnowledgeDomain: domain,
		Status:          string(types.SeedDetected),
		ProjectId:       msg.ProjectId,
		ExpiresAtBlock:  currentBlock + params.SeedExpiryBlocks,
	}
	m.Keeper.SetSeed(ctx, seed)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_begin_seeding",
		sdk.NewAttribute("project_id", msg.ProjectId),
		sdk.NewAttribute("seed_id", seedId),
		sdk.NewAttribute("domain", domain),
	))

	return &types.MsgBeginSeedingResponse{}, nil
}

// ======================== Opportunity Messages ========================

// DetectOpportunity records a new opportunity seed detected by an agent.
// Proto fields: Detector, Domain, Description, RelatedFacts
func (m msgServer) DetectOpportunity(goCtx context.Context, msg *types.MsgDetectOpportunity) (*types.MsgDetectOpportunityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	currentBlock := uint64(ctx.BlockHeight())
	params := m.Keeper.GetParams(ctx)
	counter := m.Keeper.GetNextSeedId(ctx)
	seedId := fmt.Sprintf("seed-%d-%d", currentBlock, counter)

	seed := types.OpportunitySeed{
		Id:              seedId,
		DetectedAtBlock: currentBlock,
		KnowledgeDomain: msg.Domain,
		Confidence:      "0.5",
		Status:          string(types.SeedDetected),
		ExpiresAtBlock:  currentBlock + params.SeedExpiryBlocks,
	}

	// Store demand signal with description and related facts
	if msg.Description != "" || len(msg.RelatedFacts) > 0 {
		evidence := make(map[string]string)
		evidence["description"] = msg.Description
		for i, fact := range msg.RelatedFacts {
			evidence[fmt.Sprintf("related_fact_%d", i)] = fact
		}
		seed.Signal = &types.DemandSignal{
			Type:     "agent_detection",
			Evidence: evidence,
			Strength: "0.5",
		}
	}

	m.Keeper.SetSeed(ctx, seed)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_detect_opportunity",
		sdk.NewAttribute("opportunity_id", seedId),
		sdk.NewAttribute("detector", msg.Detector),
		sdk.NewAttribute("domain", msg.Domain),
	))

	return &types.MsgDetectOpportunityResponse{OpportunityId: seedId}, nil
}

// ClaimOpportunity claims a detected opportunity seed by staking on it.
// Proto fields: Claimer, OpportunityId, Stake
func (m msgServer) ClaimOpportunity(goCtx context.Context, msg *types.MsgClaimOpportunity) (*types.MsgClaimOpportunityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	seed, found := m.Keeper.GetSeed(ctx, msg.OpportunityId)
	if !found {
		return nil, types.ErrSeedNotFound
	}
	if seed.Status == string(types.SeedClaimed) {
		return nil, types.ErrSeedAlreadyClaimed
	}
	if seed.Status == string(types.SeedExpired) {
		return nil, types.ErrSeedExpired
	}

	currentBlock := uint64(ctx.BlockHeight())
	if seed.ExpiresAtBlock > 0 && currentBlock >= seed.ExpiresAtBlock {
		// Seed has expired — update status and reject
		seed.Status = string(types.SeedExpired)
		m.Keeper.SetSeed(ctx, seed)
		return nil, types.ErrSeedExpired
	}

	// Escrow stake if provided
	if msg.Stake != "" && msg.Stake != "0" {
		stakeAmount, ok := math.NewIntFromString(msg.Stake)
		if ok && stakeAmount.IsPositive() {
			claimerAddr, err := sdk.AccAddressFromBech32(msg.Claimer)
			if err != nil {
				return nil, fmt.Errorf("invalid claimer address: %w", err)
			}
			coins := sdk.NewCoins(sdk.NewCoin("ulgm", stakeAmount))
			err = m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, claimerAddr, types.ModuleName, coins)
			if err != nil {
				return nil, types.ErrInsufficientPayment.Wrap(err.Error())
			}
		}
	}

	seed.Status = string(types.SeedClaimed)
	seed.ClaimedBy = msg.Claimer
	m.Keeper.SetSeed(ctx, seed)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_claim_opportunity",
		sdk.NewAttribute("opportunity_id", msg.OpportunityId),
		sdk.NewAttribute("claimer", msg.Claimer),
		sdk.NewAttribute("stake", msg.Stake),
	))

	return &types.MsgClaimOpportunityResponse{}, nil
}
