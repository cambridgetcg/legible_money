package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tree/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns a MsgServer implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// ======================== Project Messages ========================

// CreateProject creates a new project in seed phase.
func (m msgServer) CreateProject(goCtx context.Context, msg *types.MsgCreateProject) (*types.MsgCreateProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentBlock := uint64(ctx.BlockHeight())

	counter := m.Keeper.GetNextProjectId(ctx)
	projectId := fmt.Sprintf("proj-%d-%d", currentBlock, counter)

	project := types.ProductProject{
		Id:                  projectId,
		Name:                msg.Name,
		Description:         msg.Description,
		Phase:               types.PhaseSeed,
		CreatedAtBlock:      currentBlock,
		PhaseChangedAtBlock: currentBlock,
		KnowledgeDomain:     msg.KnowledgeDomain,
		Founder:             msg.Creator,
		Contributors: []types.ContributorRecord{
			{
				Did:           msg.Creator,
				Role:          types.RoleFounder,
				JoinedAtBlock: currentBlock,
			},
		},
		ProductSpec:   msg.ProductSpec,
		FundingSource: msg.FundingSource,
		Budget:        msg.Budget,
		Spent:         "0",
		TaskIds:       []string{},
		ServiceIds:    []string{},
	}

	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_create_project",
		sdk.NewAttribute("project_id", projectId),
		sdk.NewAttribute("founder", msg.Creator),
		sdk.NewAttribute("domain", msg.KnowledgeDomain),
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
	if !IsValidPhaseTransition(project.Phase, types.PhaseSprout) {
		return nil, types.ErrInvalidPhaseTransition
	}

	project.Phase = types.PhaseSprout
	project.PreviousPhase = types.PhaseSeed
	project.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_propose_project",
		sdk.NewAttribute("project_id", msg.ProjectId),
	))

	return &types.MsgProposeProjectResponse{}, nil
}

// StartDevelopment advances a project from sprout to growing.
func (m msgServer) StartDevelopment(goCtx context.Context, msg *types.MsgStartDevelopment) (*types.MsgStartDevelopmentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Sender {
		return nil, types.ErrNotFounder
	}
	if !IsValidPhaseTransition(project.Phase, types.PhaseGrowing) {
		return nil, types.ErrInvalidPhaseTransition
	}

	// Check min contributors
	params := m.Keeper.GetParams(ctx)
	if uint32(len(project.Contributors)) < params.MinContributorsToStart {
		return nil, fmt.Errorf("need at least %d contributors to start development", params.MinContributorsToStart)
	}

	project.Phase = types.PhaseGrowing
	project.PreviousPhase = types.PhaseSprout
	project.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_start_development",
		sdk.NewAttribute("project_id", msg.ProjectId),
	))

	return &types.MsgStartDevelopmentResponse{}, nil
}

// CompleteProject advances a project to fruiting phase.
func (m msgServer) CompleteProject(goCtx context.Context, msg *types.MsgCompleteProject) (*types.MsgCompleteProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Sender {
		return nil, types.ErrNotFounder
	}
	if !IsValidPhaseTransition(project.Phase, types.PhaseFruiting) {
		return nil, types.ErrInvalidPhaseTransition
	}

	project.PreviousPhase = project.Phase
	project.Phase = types.PhaseFruiting
	project.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_complete_project",
		sdk.NewAttribute("project_id", msg.ProjectId),
	))

	return &types.MsgCompleteProjectResponse{}, nil
}

// PauseProject moves a project to dormant phase.
func (m msgServer) PauseProject(goCtx context.Context, msg *types.MsgPauseProject) (*types.MsgPauseProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Sender {
		return nil, types.ErrNotFounder
	}
	if !IsValidPhaseTransition(project.Phase, types.PhaseDormant) {
		return nil, types.ErrInvalidPhaseTransition
	}

	project.PreviousPhase = project.Phase
	project.Phase = types.PhaseDormant
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
func (m msgServer) ResumeProject(goCtx context.Context, msg *types.MsgResumeProject) (*types.MsgResumeProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Sender {
		return nil, types.ErrNotFounder
	}
	if project.Phase != types.PhaseDormant {
		return nil, types.ErrInvalidPhaseTransition
	}

	// Resume to previous phase if valid
	resumeTo := project.PreviousPhase
	if resumeTo == "" || resumeTo == types.PhaseDormant {
		resumeTo = types.PhaseGrowing
	}
	if !IsValidPhaseTransition(project.Phase, resumeTo) {
		return nil, types.ErrInvalidPhaseTransition
	}

	project.PreviousPhase = types.PhaseDormant
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
func (m msgServer) AbandonProject(goCtx context.Context, msg *types.MsgAbandonProject) (*types.MsgAbandonProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Sender {
		return nil, types.ErrNotFounder
	}
	if !IsValidPhaseTransition(project.Phase, types.PhaseWithered) {
		return nil, types.ErrInvalidPhaseTransition
	}

	project.PreviousPhase = project.Phase
	project.Phase = types.PhaseWithered
	project.PhaseChangedAtBlock = uint64(ctx.BlockHeight())
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_abandon_project",
		sdk.NewAttribute("project_id", msg.ProjectId),
	))

	return &types.MsgAbandonProjectResponse{}, nil
}

// SpawnChildProject creates a child project from a fruiting/seeding parent.
func (m msgServer) SpawnChildProject(goCtx context.Context, msg *types.MsgSpawnChildProject) (*types.MsgSpawnChildProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	parent, found := m.Keeper.GetProject(ctx, msg.ParentProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(parent, msg.Sender) {
		return nil, types.ErrNotContributor
	}
	// Parent must be in fruiting or seeding phase
	if parent.Phase != types.PhaseFruiting && parent.Phase != types.PhaseSeeding {
		return nil, types.ErrInvalidPhaseTransition
	}

	currentBlock := uint64(ctx.BlockHeight())
	counter := m.Keeper.GetNextProjectId(ctx)
	childId := fmt.Sprintf("proj-%d-%d", currentBlock, counter)

	child := types.ProductProject{
		Id:                  childId,
		Name:                msg.Name,
		Description:         msg.Description,
		Phase:               types.PhaseSeed,
		CreatedAtBlock:      currentBlock,
		PhaseChangedAtBlock: currentBlock,
		KnowledgeDomain:     parent.KnowledgeDomain,
		ParentProjectId:     msg.ParentProjectId,
		Founder:             msg.Sender,
		Contributors: []types.ContributorRecord{
			{
				Did:           msg.Sender,
				Role:          types.RoleFounder,
				JoinedAtBlock: currentBlock,
			},
		},
		ProductSpec: msg.ProductSpec,
		Budget:      "0",
		Spent:       "0",
		TaskIds:     []string{},
		ServiceIds:  []string{},
	}

	m.Keeper.SetProject(ctx, child)

	// Advance parent to seeding if it was fruiting
	if parent.Phase == types.PhaseFruiting {
		parent.PreviousPhase = types.PhaseFruiting
		parent.Phase = types.PhaseSeeding
		parent.PhaseChangedAtBlock = currentBlock
		m.Keeper.SetProject(ctx, parent)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_spawn_child_project",
		sdk.NewAttribute("child_id", childId),
		sdk.NewAttribute("parent_id", msg.ParentProjectId),
	))

	return &types.MsgSpawnChildProjectResponse{ProjectId: childId}, nil
}

// ======================== Task Messages ========================

// AddTask adds a new task to a project.
func (m msgServer) AddTask(goCtx context.Context, msg *types.MsgAddTask) (*types.MsgAddTaskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Sender) {
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
		TaskType:             msg.TaskType,
		RequiredCapabilities: msg.RequiredCapabilities,
		Status:               types.TaskOpen,
		CreatedAtBlock:       currentBlock,
		DeadlineBlock:        msg.DeadlineBlock,
		BountyAmount:         msg.BountyAmount,
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
	if !IsFounderOrContributor(project, msg.Sender) {
		return nil, types.ErrNotContributor
	}

	if task.Status != types.TaskOpen {
		return nil, types.ErrInvalidTaskState
	}

	task.Assignee = msg.Assignee
	task.Status = types.TaskAssigned
	m.Keeper.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_assign_task",
		sdk.NewAttribute("task_id", msg.TaskId),
		sdk.NewAttribute("assignee", msg.Assignee),
	))

	return &types.MsgAssignTaskResponse{}, nil
}

// StartWork moves a task from assigned to in_progress.
func (m msgServer) StartWork(goCtx context.Context, msg *types.MsgStartWork) (*types.MsgStartWorkResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.Keeper.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}
	if task.Assignee != msg.Assignee {
		return nil, types.ErrNotAssignee
	}
	if task.Status != types.TaskAssigned {
		return nil, types.ErrInvalidTaskState
	}

	task.Status = types.TaskInProgress
	m.Keeper.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_start_work",
		sdk.NewAttribute("task_id", msg.TaskId),
	))

	return &types.MsgStartWorkResponse{}, nil
}

// SubmitDeliverable submits work for review.
func (m msgServer) SubmitDeliverable(goCtx context.Context, msg *types.MsgSubmitDeliverable) (*types.MsgSubmitDeliverableResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.Keeper.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}
	if task.Assignee != msg.Assignee {
		return nil, types.ErrNotAssignee
	}
	if task.Status != types.TaskInProgress {
		return nil, types.ErrInvalidTaskState
	}

	deliverable := msg.Deliverable
	deliverable.SubmittedAtBlock = uint64(ctx.BlockHeight())
	task.Deliverable = &deliverable
	task.Status = types.TaskReview
	m.Keeper.SetTask(ctx, task)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_submit_deliverable",
		sdk.NewAttribute("task_id", msg.TaskId),
		sdk.NewAttribute("hash", deliverable.Hash),
	))

	return &types.MsgSubmitDeliverableResponse{}, nil
}

// ApproveDeliverable approves a task deliverable.
func (m msgServer) ApproveDeliverable(goCtx context.Context, msg *types.MsgApproveDeliverable) (*types.MsgApproveDeliverableResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.Keeper.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}
	if task.Status != types.TaskReview {
		return nil, types.ErrInvalidTaskState
	}

	// Reviewer must be a project contributor (not the assignee)
	project, found := m.Keeper.GetProject(ctx, task.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Reviewer) {
		return nil, types.ErrNotContributor
	}

	task.Status = types.TaskCompleted
	task.CompletedAtBlock = uint64(ctx.BlockHeight())
	if task.Deliverable != nil {
		task.Deliverable.VerifiedBy = append(task.Deliverable.VerifiedBy, msg.Reviewer)
	}
	m.Keeper.SetTask(ctx, task)

	// Update project completed tasks count
	project.CompletedTasks++
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_approve_deliverable",
		sdk.NewAttribute("task_id", msg.TaskId),
		sdk.NewAttribute("reviewer", msg.Reviewer),
	))

	return &types.MsgApproveDeliverableResponse{}, nil
}

// RejectDeliverable rejects a task deliverable.
func (m msgServer) RejectDeliverable(goCtx context.Context, msg *types.MsgRejectDeliverable) (*types.MsgRejectDeliverableResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	task, found := m.Keeper.GetTask(ctx, msg.TaskId)
	if !found {
		return nil, types.ErrTaskNotFound
	}
	if task.Status != types.TaskReview {
		return nil, types.ErrInvalidTaskState
	}

	project, found := m.Keeper.GetProject(ctx, task.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Reviewer) {
		return nil, types.ErrNotContributor
	}

	params := m.Keeper.GetParams(ctx)
	task.RejectionCount++

	if task.RejectionCount >= params.MaxRejections {
		task.Status = types.TaskDisputed
	} else {
		// Send back to in_progress for rework
		task.Status = types.TaskInProgress
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
	if project.Founder != msg.Sender {
		return nil, types.ErrNotFounder
	}

	if task.Status != types.TaskRejected && task.Status != types.TaskDisputed {
		return nil, types.ErrInvalidTaskState
	}

	task.Status = types.TaskOpen
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
		if app.Did == msg.Applicant && app.Status == types.AppPending {
			return nil, types.ErrAlreadyApplied
		}
	}

	project.Applications = append(project.Applications, types.ProjectApplication{
		Did:            msg.Applicant,
		ProposedRole:   msg.ProposedRole,
		Message:        msg.Message,
		Capabilities:   msg.Capabilities,
		AppliedAtBlock: uint64(ctx.BlockHeight()),
		Status:         types.AppPending,
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
func (m msgServer) ReviewApplication(goCtx context.Context, msg *types.MsgReviewApplication) (*types.MsgReviewApplicationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if project.Founder != msg.Reviewer {
		return nil, types.ErrNotFounder
	}

	appFound := false
	for i, app := range project.Applications {
		if app.Did == msg.ApplicantDid && app.Status == types.AppPending {
			appFound = true
			if msg.Approve {
				project.Applications[i].Status = types.AppApproved

				// Check max contributors
				params := m.Keeper.GetParams(ctx)
				if uint32(len(project.Contributors)) >= params.MaxContributors {
					return nil, types.ErrMaxContributorsReached
				}

				// Add as contributor
				project.Contributors = append(project.Contributors, types.ContributorRecord{
					Did:           app.Did,
					Role:          app.ProposedRole,
					JoinedAtBlock: uint64(ctx.BlockHeight()),
				})
			} else {
				project.Applications[i].Status = types.AppRejected
			}
			break
		}
	}
	if !appFound {
		return nil, types.ErrApplicationNotFound
	}

	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_review_application",
		sdk.NewAttribute("project_id", msg.ProjectId),
		sdk.NewAttribute("applicant", msg.ApplicantDid),
		sdk.NewAttribute("approved", fmt.Sprintf("%v", msg.Approve)),
	))

	return &types.MsgReviewApplicationResponse{}, nil
}

// SetAvailability records an agent's availability for project work.
// This is stored as an event for now — no persistent state beyond events.
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
func (m msgServer) DeployService(goCtx context.Context, msg *types.MsgDeployService) (*types.MsgDeployServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	project, found := m.Keeper.GetProject(ctx, msg.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Deployer) {
		return nil, types.ErrNotContributor
	}

	// Project must be at least in growing phase
	if project.Phase == types.PhaseSeed || project.Phase == types.PhaseSprout || project.Phase == types.PhaseWithered {
		return nil, types.ErrInvalidPhaseTransition
	}

	currentBlock := uint64(ctx.BlockHeight())
	counter := m.Keeper.GetNextServiceId(ctx)
	serviceId := fmt.Sprintf("svc-%d-%d", currentBlock, counter)

	service := types.ServiceLeaf{
		Id:              serviceId,
		ProjectId:       msg.ProjectId,
		Name:            msg.Name,
		Description:     msg.Description,
		ServiceType:     msg.ServiceType,
		ContractAddress: msg.ContractAddress,
		TotalCalls:      "0",
		TotalRevenue:    "0",
		UptimeBlocks:    "0",
		Status:          types.ServiceDeploying,
		DeployedAtBlock: currentBlock,
	}

	m.Keeper.SetService(ctx, service)

	// Update project service tracking
	project.ServiceIds = append(project.ServiceIds, serviceId)
	m.Keeper.SetProject(ctx, project)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_deploy_service",
		sdk.NewAttribute("service_id", serviceId),
		sdk.NewAttribute("project_id", msg.ProjectId),
	))

	return &types.MsgDeployServiceResponse{ServiceId: serviceId}, nil
}

// PauseService pauses an active service.
func (m msgServer) PauseService(goCtx context.Context, msg *types.MsgPauseService) (*types.MsgPauseServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	service, found := m.Keeper.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, types.ErrServiceNotFound
	}

	// Verify sender is project founder or contributor
	project, found := m.Keeper.GetProject(ctx, service.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Sender) {
		return nil, types.ErrNotContributor
	}

	if service.Status != types.ServiceActive && service.Status != types.ServiceDeploying && service.Status != types.ServiceDegraded {
		return nil, types.ErrInvalidServiceState
	}

	service.Status = types.ServicePaused
	m.Keeper.SetService(ctx, service)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_pause_service",
		sdk.NewAttribute("service_id", msg.ServiceId),
	))

	return &types.MsgPauseServiceResponse{}, nil
}

// ResumeService resumes a paused service.
func (m msgServer) ResumeService(goCtx context.Context, msg *types.MsgResumeService) (*types.MsgResumeServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	service, found := m.Keeper.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, types.ErrServiceNotFound
	}

	project, found := m.Keeper.GetProject(ctx, service.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Sender) {
		return nil, types.ErrNotContributor
	}

	if service.Status != types.ServicePaused {
		return nil, types.ErrInvalidServiceState
	}

	service.Status = types.ServiceActive
	m.Keeper.SetService(ctx, service)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_resume_service",
		sdk.NewAttribute("service_id", msg.ServiceId),
	))

	return &types.MsgResumeServiceResponse{}, nil
}

// RetireService permanently retires a service.
func (m msgServer) RetireService(goCtx context.Context, msg *types.MsgRetireService) (*types.MsgRetireServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	service, found := m.Keeper.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, types.ErrServiceNotFound
	}

	project, found := m.Keeper.GetProject(ctx, service.ProjectId)
	if !found {
		return nil, types.ErrProjectNotFound
	}
	if !IsFounderOrContributor(project, msg.Sender) {
		return nil, types.ErrNotContributor
	}

	if service.Status == types.ServiceRetired {
		return nil, types.ErrInvalidServiceState
	}

	service.Status = types.ServiceRetired
	m.Keeper.SetService(ctx, service)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"tree_retire_service",
		sdk.NewAttribute("service_id", msg.ServiceId),
	))

	return &types.MsgRetireServiceResponse{}, nil
}
