package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Enums (not in proto; used as semantic aliases) ---

type ProjectPhase string

const (
	PhaseSeed     ProjectPhase = "seed"
	PhaseSprout   ProjectPhase = "sprout"
	PhaseGrowing  ProjectPhase = "growing"
	PhaseMature   ProjectPhase = "mature"
	PhaseFruiting ProjectPhase = "fruiting"
	PhaseSeeding  ProjectPhase = "seeding"
	PhaseDormant  ProjectPhase = "dormant"
	PhaseWithered ProjectPhase = "withered"
)

type TaskStatus string

const (
	TaskOpen       TaskStatus = "open"
	TaskAssigned   TaskStatus = "assigned"
	TaskInProgress TaskStatus = "in_progress"
	TaskReview     TaskStatus = "review"
	TaskCompleted  TaskStatus = "completed"
	TaskRejected   TaskStatus = "rejected"
	TaskDisputed   TaskStatus = "disputed"
)

type TaskType string

const (
	TaskDesign     TaskType = "design"
	TaskImplement  TaskType = "implement"
	TaskTest       TaskType = "test"
	TaskReviewType TaskType = "review"
	TaskDeploy     TaskType = "deploy"
	TaskDocument   TaskType = "document"
	TaskVerify     TaskType = "verify"
	TaskResearch   TaskType = "research"
	TaskGovernance TaskType = "governance"
	TaskMaintain   TaskType = "maintain"
)

type ServiceStatus string

const (
	ServiceDeploying ServiceStatus = "deploying"
	ServiceActive    ServiceStatus = "active"
	ServiceDegraded  ServiceStatus = "degraded"
	ServicePaused    ServiceStatus = "paused"
	ServiceRetired   ServiceStatus = "retired"
)

type ServiceType string

const (
	ServiceAPI          ServiceType = "api"
	ServiceDataFeed     ServiceType = "data_feed"
	ServiceComputation  ServiceType = "computation"
	ServiceVerification ServiceType = "verification"
	ServiceAggregation  ServiceType = "aggregation"
	ServiceAnalytics    ServiceType = "analytics"
)

type SeedStatus string

const (
	SeedDetected SeedStatus = "detected"
	SeedProposed SeedStatus = "proposed"
	SeedClaimed  SeedStatus = "claimed"
	SeedExpired  SeedStatus = "expired"
)

type ContributorRole string

const (
	RoleFounder    ContributorRole = "founder"
	RoleArchitect  ContributorRole = "architect"
	RoleDeveloper  ContributorRole = "developer"
	RoleReviewer   ContributorRole = "reviewer"
	RoleTester     ContributorRole = "tester"
	RoleMaintainer ContributorRole = "maintainer"
)

type ApplicationStatus string

const (
	AppPending  ApplicationStatus = "pending"
	AppApproved ApplicationStatus = "approved"
	AppRejected ApplicationStatus = "rejected"
)

type FundingSourceType string

const (
	FundingResearch  FundingSourceType = "research_fund"
	FundingBounty    FundingSourceType = "bounty_pool"
	FundingAgent     FundingSourceType = "agent_funded"
	FundingCommunity FundingSourceType = "community"
)

// --- Params Validation ---
// Params struct is proto-generated in genesis.pb.go.

func (p Params) Validate() error {
	if p.MaxTasksPerProject == 0 {
		return fmt.Errorf("max tasks per project must be positive")
	}
	if p.MaxRejections == 0 {
		return fmt.Errorf("max rejections must be positive")
	}
	bpSum := p.EvidenceTaxBp + p.ContributorsBp + p.ResearchFundBp + p.ProtocolTreasuryBp
	if bpSum != 0 && bpSum != 1000000 {
		return fmt.Errorf("revenue basis points must sum to 1000000, got %d", bpSum)
	}
	return nil
}

// --- ValidateBasic methods for proto-generated Msg types ---
// Proto field names: MsgCreateProject has Creator, Title (not Name), Domain (not KnowledgeDomain).
// MsgStartDevelopment, MsgCompleteProject, MsgPauseProject, MsgResumeProject, MsgAbandonProject, MsgReopenTask use Authority (not Sender).
// MsgSpawnChildProject uses Creator (not Sender).
// MsgAssignTask uses Assigner (not Sender).
// MsgAddTask uses Creator (not Sender), Bounty (not BountyAmount).
// MsgStartWork uses Worker (not Assignee).
// MsgSubmitDeliverable uses Worker (not Assignee), DeliverableHash/DeliverableUri (not Deliverable).
// MsgApproveDeliverable uses Approver (not Reviewer).
// MsgRejectDeliverable uses Rejector (not Reviewer).
// MsgDeployService has Deployer (same), but no ProjectId, no ServiceType, no ContractAddress.
// MsgPauseService/ResumeService/RetireService use Owner (not Sender).
// MsgApplyToProject uses Role (not ProposedRole), Pitch (not Message).
// MsgReviewApplication uses ApplicationId (not ProjectId+ApplicantDid), Accepted (not Approve).
// MsgCallService uses Caller, ServiceId (same), Payload (not Input), MaxFee (not in old).

func (m *MsgCreateProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	return nil
}

func (m *MsgProposeProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

func (m *MsgStartDevelopment) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

func (m *MsgCompleteProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

func (m *MsgPauseProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

func (m *MsgResumeProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

func (m *MsgAbandonProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

func (m *MsgSpawnChildProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator: %w", err)
	}
	if m.ParentProjectId == "" {
		return fmt.Errorf("parent project id cannot be empty")
	}
	if m.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	return nil
}

func (m *MsgAddTask) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	if m.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	return nil
}

func (m *MsgAssignTask) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Assigner); err != nil {
		return fmt.Errorf("invalid assigner: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Assignee); err != nil {
		return fmt.Errorf("invalid assignee: %w", err)
	}
	return nil
}

func (m *MsgStartWork) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Worker); err != nil {
		return fmt.Errorf("invalid worker: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	return nil
}

func (m *MsgSubmitDeliverable) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Worker); err != nil {
		return fmt.Errorf("invalid worker: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	return nil
}

func (m *MsgApproveDeliverable) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Approver); err != nil {
		return fmt.Errorf("invalid approver: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	return nil
}

func (m *MsgRejectDeliverable) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Rejector); err != nil {
		return fmt.Errorf("invalid rejector: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	return nil
}

func (m *MsgReopenTask) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	return nil
}

func (m *MsgApplyToProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Applicant); err != nil {
		return fmt.Errorf("invalid applicant: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

func (m *MsgReviewApplication) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Reviewer); err != nil {
		return fmt.Errorf("invalid reviewer: %w", err)
	}
	if m.ApplicationId == "" {
		return fmt.Errorf("application id cannot be empty")
	}
	return nil
}

func (m *MsgSetAvailability) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Agent); err != nil {
		return fmt.Errorf("invalid agent: %w", err)
	}
	return nil
}

func (m *MsgDeployService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Deployer); err != nil {
		return fmt.Errorf("invalid deployer: %w", err)
	}
	if m.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}

func (m *MsgPauseService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner: %w", err)
	}
	if m.ServiceId == "" {
		return fmt.Errorf("service id cannot be empty")
	}
	return nil
}

func (m *MsgResumeService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner: %w", err)
	}
	if m.ServiceId == "" {
		return fmt.Errorf("service id cannot be empty")
	}
	return nil
}

func (m *MsgRetireService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner: %w", err)
	}
	if m.ServiceId == "" {
		return fmt.Errorf("service id cannot be empty")
	}
	return nil
}

func (m *MsgCallService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Caller); err != nil {
		return fmt.Errorf("invalid caller address: %w", err)
	}
	if m.ServiceId == "" {
		return fmt.Errorf("service id cannot be empty")
	}
	return nil
}
