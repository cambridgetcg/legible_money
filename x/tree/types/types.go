package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Enums ---

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

// --- Supporting Structs ---

type ContributorRecord struct {
	Did            string          `json:"did"`
	Role           ContributorRole `json:"role"`
	JoinedAtBlock  uint64          `json:"joined_at_block"`
	TasksCompleted uint32          `json:"tasks_completed"`
	TotalEarned    string          `json:"total_earned"` // ulgm
}

type ProductSpec struct {
	ServiceType    ServiceType `json:"service_type"`
	Requirements   []string    `json:"requirements"`
	Capabilities   []string    `json:"capabilities"`
	EstimatedTasks uint32      `json:"estimated_tasks"`
	MvpCriteria    []string    `json:"mvp_criteria"`
}

type FundingSource struct {
	Type       FundingSourceType `json:"type"`
	Amount     string            `json:"amount,omitempty"`
	BountyId   string            `json:"bounty_id,omitempty"`
	Funder     string            `json:"funder,omitempty"`
	ProposalId string            `json:"proposal_id,omitempty"`
}

type DemandSignal struct {
	Type     string            `json:"type"`
	Evidence map[string]string `json:"evidence"`
	Strength string            `json:"strength"` // "0.0"-"1.0"
}

type ProjectApplication struct {
	Did            string            `json:"did"`
	ProposedRole   ContributorRole   `json:"proposed_role"`
	Message        string            `json:"message"`
	Capabilities   []string          `json:"capabilities"`
	AppliedAtBlock uint64            `json:"applied_at_block"`
	Status         ApplicationStatus `json:"status"`
}

type TaskDeliverable struct {
	Type             string   `json:"type"` // code, config, data, proof, document
	Hash             string   `json:"hash"`
	SubmittedAtBlock uint64   `json:"submitted_at_block"`
	VerifiedBy       []string `json:"verified_by"`
}

// --- Top-Level Entities ---

// ProductProject represents a project in the Tree of Life.
type ProductProject struct {
	Id                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	Phase               ProjectPhase        `json:"phase"`
	PreviousPhase       ProjectPhase        `json:"previous_phase,omitempty"`
	PauseReason         string              `json:"pause_reason,omitempty"`
	CreatedAtBlock      uint64              `json:"created_at_block"`
	PhaseChangedAtBlock uint64              `json:"phase_changed_at_block"`
	KnowledgeDomain     string              `json:"knowledge_domain"`
	DemandSignal        DemandSignal        `json:"demand_signal"`
	ParentProjectId     string              `json:"parent_project_id,omitempty"`
	Founder             string              `json:"founder"` // bech32
	Contributors        []ContributorRecord `json:"contributors"`
	PartnershipId       string              `json:"partnership_id,omitempty"`
	ProductSpec         ProductSpec         `json:"product_spec"`
	TaskIds             []string            `json:"task_ids"`
	CompletedTasks      uint32              `json:"completed_tasks"`
	TotalTasks          uint32              `json:"total_tasks"`
	FundingSource       FundingSource       `json:"funding_source"`
	Budget              string              `json:"budget"` // ulgm
	Spent               string              `json:"spent"`  // ulgm
	RevenueGenerated    string              `json:"revenue_generated"`
	ContractAddress     string              `json:"contract_address,omitempty"`
	ServiceIds          []string            `json:"service_ids"`
	Applications        []ProjectApplication `json:"applications,omitempty"`
}

func (p *ProductProject) Reset()         { *p = ProductProject{} }
func (p *ProductProject) String() string { return fmt.Sprintf("ProductProject{%s:%s}", p.Id, p.Phase) }
func (p *ProductProject) ProtoMessage()  {}

// ProjectTask represents a task within a project.
type ProjectTask struct {
	Id                   string           `json:"id"`
	ProjectId            string           `json:"project_id"`
	Title                string           `json:"title"`
	Description          string           `json:"description"`
	TaskType             TaskType         `json:"task_type"`
	RequiredCapabilities []string         `json:"required_capabilities"`
	Assignee             string           `json:"assignee,omitempty"` // bech32
	Reviewers            []string         `json:"reviewers"`
	Status               TaskStatus       `json:"status"`
	CreatedAtBlock       uint64           `json:"created_at_block"`
	DeadlineBlock        uint64           `json:"deadline_block,omitempty"`
	CompletedAtBlock     uint64           `json:"completed_at_block,omitempty"`
	Deliverable          *TaskDeliverable `json:"deliverable,omitempty"`
	BountyAmount         string           `json:"bounty_amount"` // ulgm
	BountyId             string           `json:"bounty_id,omitempty"`
	RejectionCount       uint32           `json:"rejection_count"`
}

func (t *ProjectTask) Reset()         { *t = ProjectTask{} }
func (t *ProjectTask) String() string { return fmt.Sprintf("ProjectTask{%s:%s}", t.Id, t.Status) }
func (t *ProjectTask) ProtoMessage()  {}

// ServiceLeaf represents a deployed service from a project.
type ServiceLeaf struct {
	Id               string        `json:"id"`
	ProjectId        string        `json:"project_id"`
	Name             string        `json:"name"`
	Description      string        `json:"description"`
	ServiceType      ServiceType   `json:"service_type"`
	ContractAddress  string        `json:"contract_address"`
	TotalCalls       string        `json:"total_calls"`
	TotalRevenue     string        `json:"total_revenue"`
	UptimeBlocks     string        `json:"uptime_blocks"`
	LastCalledBlock  uint64        `json:"last_called_block,omitempty"`
	AverageLatencyMs uint32        `json:"average_latency_ms"`
	Status           ServiceStatus `json:"status"`
	DeployedAtBlock  uint64        `json:"deployed_at_block"`
}

func (s *ServiceLeaf) Reset()         { *s = ServiceLeaf{} }
func (s *ServiceLeaf) String() string { return fmt.Sprintf("ServiceLeaf{%s:%s}", s.Id, s.Status) }
func (s *ServiceLeaf) ProtoMessage()  {}

// OpportunitySeed represents a detected opportunity.
type OpportunitySeed struct {
	Id              string       `json:"id"`
	DetectedAtBlock uint64       `json:"detected_at_block"`
	KnowledgeDomain string       `json:"knowledge_domain"`
	Signal          DemandSignal `json:"signal"`
	Confidence      string       `json:"confidence"`
	EstimatedDemand string       `json:"estimated_demand"`
	EstimatedEffort uint32       `json:"estimated_effort"`
	EstimatedROI    string       `json:"estimated_roi"`
	Status          SeedStatus   `json:"status"`
	ClaimedBy       string       `json:"claimed_by,omitempty"`
	ProjectId       string       `json:"project_id,omitempty"`
	ExpiresAtBlock  uint64       `json:"expires_at_block"`
}

func (s *OpportunitySeed) Reset()         { *s = OpportunitySeed{} }
func (s *OpportunitySeed) String() string { return fmt.Sprintf("OpportunitySeed{%s:%s}", s.Id, s.Status) }
func (s *OpportunitySeed) ProtoMessage()  {}

// --- Params ---

type Params struct {
	MinBudget              string `json:"min_budget"`               // ulgm
	MaxTasksPerProject     uint32 `json:"max_tasks_per_project"`
	MaxContributors        uint32 `json:"max_contributors"`
	MaxApplications        uint32 `json:"max_applications"`
	TaskDeadlineMinBlocks  uint64 `json:"task_deadline_min_blocks"`
	TaskDeadlineMaxBlocks  uint64 `json:"task_deadline_max_blocks"`
	MaxRejections          uint32 `json:"max_rejections"`           // before auto-dispute
	SeedExpiryBlocks       uint64 `json:"seed_expiry_blocks"`
	MinContributorsToStart uint32 `json:"min_contributors_to_start"` // to move sprout->growing
}

func (p *Params) Reset()         { *p = Params{} }
func (p *Params) String() string { return fmt.Sprintf("Params{maxTasks:%d}", p.MaxTasksPerProject) }
func (p *Params) ProtoMessage()  {}

func DefaultParams() Params {
	return Params{
		MinBudget:              "1000000",   // 1 LGM
		MaxTasksPerProject:     200,
		MaxContributors:        50,
		MaxApplications:        100,
		TaskDeadlineMinBlocks:  100,
		TaskDeadlineMaxBlocks:  1036800,     // ~30 days
		MaxRejections:          3,
		SeedExpiryBlocks:       172800,      // ~5 days
		MinContributorsToStart: 1,
	}
}

func (p Params) Validate() error {
	if p.MaxTasksPerProject == 0 {
		return fmt.Errorf("max tasks per project must be positive")
	}
	if p.MaxRejections == 0 {
		return fmt.Errorf("max rejections must be positive")
	}
	return nil
}

// --- Genesis ---

type GenesisState struct {
	Params   Params            `json:"params"`
	Projects []ProductProject  `json:"projects"`
	Tasks    []ProjectTask     `json:"tasks"`
	Services []ServiceLeaf     `json:"services"`
	Seeds    []OpportunitySeed `json:"seeds"`
}

func (gs *GenesisState) Reset()         { *gs = GenesisState{} }
func (gs *GenesisState) String() string { return fmt.Sprintf("GenesisState{projects:%d}", len(gs.Projects)) }
func (gs *GenesisState) ProtoMessage()  {}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		Projects: []ProductProject{},
		Tasks:    []ProjectTask{},
		Services: []ServiceLeaf{},
		Seeds:    []OpportunitySeed{},
	}
}

func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

// ======================== Message Types ========================

// --- Project Messages (8) ---

type MsgCreateProject struct {
	Creator         string        `json:"creator"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	KnowledgeDomain string        `json:"knowledge_domain"`
	ProductSpec     ProductSpec   `json:"product_spec"`
	FundingSource   FundingSource `json:"funding_source"`
	Budget          string        `json:"budget"`
}

func (m *MsgCreateProject) Reset()         { *m = MsgCreateProject{} }
func (m *MsgCreateProject) String() string { return fmt.Sprintf("MsgCreateProject{%s}", m.Name) }
func (m *MsgCreateProject) ProtoMessage()  {}
func (m *MsgCreateProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}

type MsgCreateProjectResponse struct {
	ProjectId string `json:"project_id"`
}

func (m *MsgCreateProjectResponse) Reset()         { *m = MsgCreateProjectResponse{} }
func (m *MsgCreateProjectResponse) String() string { return fmt.Sprintf("MsgCreateProjectResponse{%s}", m.ProjectId) }
func (m *MsgCreateProjectResponse) ProtoMessage()  {}

type MsgProposeProject struct {
	Proposer  string `json:"proposer"`
	ProjectId string `json:"project_id"`
}

func (m *MsgProposeProject) Reset()         { *m = MsgProposeProject{} }
func (m *MsgProposeProject) String() string { return "MsgProposeProject" }
func (m *MsgProposeProject) ProtoMessage()  {}
func (m *MsgProposeProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

type MsgProposeProjectResponse struct{}

func (m *MsgProposeProjectResponse) Reset()         { *m = MsgProposeProjectResponse{} }
func (m *MsgProposeProjectResponse) String() string { return "MsgProposeProjectResponse" }
func (m *MsgProposeProjectResponse) ProtoMessage()  {}

type MsgStartDevelopment struct {
	Sender    string `json:"sender"`
	ProjectId string `json:"project_id"`
}

func (m *MsgStartDevelopment) Reset()         { *m = MsgStartDevelopment{} }
func (m *MsgStartDevelopment) String() string { return "MsgStartDevelopment" }
func (m *MsgStartDevelopment) ProtoMessage()  {}
func (m *MsgStartDevelopment) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

type MsgStartDevelopmentResponse struct{}

func (m *MsgStartDevelopmentResponse) Reset()         { *m = MsgStartDevelopmentResponse{} }
func (m *MsgStartDevelopmentResponse) String() string { return "MsgStartDevelopmentResponse" }
func (m *MsgStartDevelopmentResponse) ProtoMessage()  {}

type MsgCompleteProject struct {
	Sender    string `json:"sender"`
	ProjectId string `json:"project_id"`
}

func (m *MsgCompleteProject) Reset()         { *m = MsgCompleteProject{} }
func (m *MsgCompleteProject) String() string { return "MsgCompleteProject" }
func (m *MsgCompleteProject) ProtoMessage()  {}
func (m *MsgCompleteProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

type MsgCompleteProjectResponse struct{}

func (m *MsgCompleteProjectResponse) Reset()         { *m = MsgCompleteProjectResponse{} }
func (m *MsgCompleteProjectResponse) String() string { return "MsgCompleteProjectResponse" }
func (m *MsgCompleteProjectResponse) ProtoMessage()  {}

type MsgPauseProject struct {
	Sender    string `json:"sender"`
	ProjectId string `json:"project_id"`
	Reason    string `json:"reason"`
}

func (m *MsgPauseProject) Reset()         { *m = MsgPauseProject{} }
func (m *MsgPauseProject) String() string { return "MsgPauseProject" }
func (m *MsgPauseProject) ProtoMessage()  {}
func (m *MsgPauseProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

type MsgPauseProjectResponse struct{}

func (m *MsgPauseProjectResponse) Reset()         { *m = MsgPauseProjectResponse{} }
func (m *MsgPauseProjectResponse) String() string { return "MsgPauseProjectResponse" }
func (m *MsgPauseProjectResponse) ProtoMessage()  {}

type MsgResumeProject struct {
	Sender    string `json:"sender"`
	ProjectId string `json:"project_id"`
}

func (m *MsgResumeProject) Reset()         { *m = MsgResumeProject{} }
func (m *MsgResumeProject) String() string { return "MsgResumeProject" }
func (m *MsgResumeProject) ProtoMessage()  {}
func (m *MsgResumeProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

type MsgResumeProjectResponse struct{}

func (m *MsgResumeProjectResponse) Reset()         { *m = MsgResumeProjectResponse{} }
func (m *MsgResumeProjectResponse) String() string { return "MsgResumeProjectResponse" }
func (m *MsgResumeProjectResponse) ProtoMessage()  {}

type MsgAbandonProject struct {
	Sender    string `json:"sender"`
	ProjectId string `json:"project_id"`
}

func (m *MsgAbandonProject) Reset()         { *m = MsgAbandonProject{} }
func (m *MsgAbandonProject) String() string { return "MsgAbandonProject" }
func (m *MsgAbandonProject) ProtoMessage()  {}
func (m *MsgAbandonProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

type MsgAbandonProjectResponse struct{}

func (m *MsgAbandonProjectResponse) Reset()         { *m = MsgAbandonProjectResponse{} }
func (m *MsgAbandonProjectResponse) String() string { return "MsgAbandonProjectResponse" }
func (m *MsgAbandonProjectResponse) ProtoMessage()  {}

type MsgSpawnChildProject struct {
	Sender          string      `json:"sender"`
	ParentProjectId string      `json:"parent_project_id"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	ProductSpec     ProductSpec `json:"product_spec"`
}

func (m *MsgSpawnChildProject) Reset()         { *m = MsgSpawnChildProject{} }
func (m *MsgSpawnChildProject) String() string { return "MsgSpawnChildProject" }
func (m *MsgSpawnChildProject) ProtoMessage()  {}
func (m *MsgSpawnChildProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.ParentProjectId == "" {
		return fmt.Errorf("parent project id cannot be empty")
	}
	if m.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}

type MsgSpawnChildProjectResponse struct {
	ProjectId string `json:"project_id"`
}

func (m *MsgSpawnChildProjectResponse) Reset()         { *m = MsgSpawnChildProjectResponse{} }
func (m *MsgSpawnChildProjectResponse) String() string { return "MsgSpawnChildProjectResponse" }
func (m *MsgSpawnChildProjectResponse) ProtoMessage()  {}

// --- Task Messages (7) ---

type MsgAddTask struct {
	Sender               string   `json:"sender"`
	ProjectId            string   `json:"project_id"`
	Title                string   `json:"title"`
	Description          string   `json:"description"`
	TaskType             TaskType `json:"task_type"`
	RequiredCapabilities []string `json:"required_capabilities"`
	BountyAmount         string   `json:"bounty_amount"`
	DeadlineBlock        uint64   `json:"deadline_block,omitempty"`
}

func (m *MsgAddTask) Reset()         { *m = MsgAddTask{} }
func (m *MsgAddTask) String() string { return fmt.Sprintf("MsgAddTask{%s}", m.Title) }
func (m *MsgAddTask) ProtoMessage()  {}
func (m *MsgAddTask) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	if m.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	return nil
}

type MsgAddTaskResponse struct {
	TaskId string `json:"task_id"`
}

func (m *MsgAddTaskResponse) Reset()         { *m = MsgAddTaskResponse{} }
func (m *MsgAddTaskResponse) String() string { return "MsgAddTaskResponse" }
func (m *MsgAddTaskResponse) ProtoMessage()  {}

type MsgAssignTask struct {
	Sender   string `json:"sender"`
	TaskId   string `json:"task_id"`
	Assignee string `json:"assignee"`
}

func (m *MsgAssignTask) Reset()         { *m = MsgAssignTask{} }
func (m *MsgAssignTask) String() string { return "MsgAssignTask" }
func (m *MsgAssignTask) ProtoMessage()  {}
func (m *MsgAssignTask) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Assignee); err != nil {
		return fmt.Errorf("invalid assignee: %w", err)
	}
	return nil
}

type MsgAssignTaskResponse struct{}

func (m *MsgAssignTaskResponse) Reset()         { *m = MsgAssignTaskResponse{} }
func (m *MsgAssignTaskResponse) String() string { return "MsgAssignTaskResponse" }
func (m *MsgAssignTaskResponse) ProtoMessage()  {}

type MsgStartWork struct {
	Assignee string `json:"assignee"`
	TaskId   string `json:"task_id"`
}

func (m *MsgStartWork) Reset()         { *m = MsgStartWork{} }
func (m *MsgStartWork) String() string { return "MsgStartWork" }
func (m *MsgStartWork) ProtoMessage()  {}
func (m *MsgStartWork) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Assignee); err != nil {
		return fmt.Errorf("invalid assignee: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	return nil
}

type MsgStartWorkResponse struct{}

func (m *MsgStartWorkResponse) Reset()         { *m = MsgStartWorkResponse{} }
func (m *MsgStartWorkResponse) String() string { return "MsgStartWorkResponse" }
func (m *MsgStartWorkResponse) ProtoMessage()  {}

type MsgSubmitDeliverable struct {
	Assignee    string          `json:"assignee"`
	TaskId      string          `json:"task_id"`
	Deliverable TaskDeliverable `json:"deliverable"`
}

func (m *MsgSubmitDeliverable) Reset()         { *m = MsgSubmitDeliverable{} }
func (m *MsgSubmitDeliverable) String() string { return "MsgSubmitDeliverable" }
func (m *MsgSubmitDeliverable) ProtoMessage()  {}
func (m *MsgSubmitDeliverable) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Assignee); err != nil {
		return fmt.Errorf("invalid assignee: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	return nil
}

type MsgSubmitDeliverableResponse struct{}

func (m *MsgSubmitDeliverableResponse) Reset()         { *m = MsgSubmitDeliverableResponse{} }
func (m *MsgSubmitDeliverableResponse) String() string { return "MsgSubmitDeliverableResponse" }
func (m *MsgSubmitDeliverableResponse) ProtoMessage()  {}

type MsgApproveDeliverable struct {
	Reviewer string `json:"reviewer"`
	TaskId   string `json:"task_id"`
}

func (m *MsgApproveDeliverable) Reset()         { *m = MsgApproveDeliverable{} }
func (m *MsgApproveDeliverable) String() string { return "MsgApproveDeliverable" }
func (m *MsgApproveDeliverable) ProtoMessage()  {}
func (m *MsgApproveDeliverable) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Reviewer); err != nil {
		return fmt.Errorf("invalid reviewer: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	return nil
}

type MsgApproveDeliverableResponse struct{}

func (m *MsgApproveDeliverableResponse) Reset()         { *m = MsgApproveDeliverableResponse{} }
func (m *MsgApproveDeliverableResponse) String() string { return "MsgApproveDeliverableResponse" }
func (m *MsgApproveDeliverableResponse) ProtoMessage()  {}

type MsgRejectDeliverable struct {
	Reviewer string `json:"reviewer"`
	TaskId   string `json:"task_id"`
	Reason   string `json:"reason"`
}

func (m *MsgRejectDeliverable) Reset()         { *m = MsgRejectDeliverable{} }
func (m *MsgRejectDeliverable) String() string { return "MsgRejectDeliverable" }
func (m *MsgRejectDeliverable) ProtoMessage()  {}
func (m *MsgRejectDeliverable) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Reviewer); err != nil {
		return fmt.Errorf("invalid reviewer: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	return nil
}

type MsgRejectDeliverableResponse struct{}

func (m *MsgRejectDeliverableResponse) Reset()         { *m = MsgRejectDeliverableResponse{} }
func (m *MsgRejectDeliverableResponse) String() string { return "MsgRejectDeliverableResponse" }
func (m *MsgRejectDeliverableResponse) ProtoMessage()  {}

type MsgReopenTask struct {
	Sender string `json:"sender"`
	TaskId string `json:"task_id"`
}

func (m *MsgReopenTask) Reset()         { *m = MsgReopenTask{} }
func (m *MsgReopenTask) String() string { return "MsgReopenTask" }
func (m *MsgReopenTask) ProtoMessage()  {}
func (m *MsgReopenTask) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.TaskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	return nil
}

type MsgReopenTaskResponse struct{}

func (m *MsgReopenTaskResponse) Reset()         { *m = MsgReopenTaskResponse{} }
func (m *MsgReopenTaskResponse) String() string { return "MsgReopenTaskResponse" }
func (m *MsgReopenTaskResponse) ProtoMessage()  {}

// --- Application Messages (3) ---

type MsgApplyToProject struct {
	Applicant    string          `json:"applicant"`
	ProjectId    string          `json:"project_id"`
	ProposedRole ContributorRole `json:"proposed_role"`
	Message      string          `json:"message"`
	Capabilities []string        `json:"capabilities"`
}

func (m *MsgApplyToProject) Reset()         { *m = MsgApplyToProject{} }
func (m *MsgApplyToProject) String() string { return "MsgApplyToProject" }
func (m *MsgApplyToProject) ProtoMessage()  {}
func (m *MsgApplyToProject) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Applicant); err != nil {
		return fmt.Errorf("invalid applicant: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

type MsgApplyToProjectResponse struct{}

func (m *MsgApplyToProjectResponse) Reset()         { *m = MsgApplyToProjectResponse{} }
func (m *MsgApplyToProjectResponse) String() string { return "MsgApplyToProjectResponse" }
func (m *MsgApplyToProjectResponse) ProtoMessage()  {}

type MsgReviewApplication struct {
	Reviewer     string `json:"reviewer"`
	ProjectId    string `json:"project_id"`
	ApplicantDid string `json:"applicant_did"`
	Approve      bool   `json:"approve"`
}

func (m *MsgReviewApplication) Reset()         { *m = MsgReviewApplication{} }
func (m *MsgReviewApplication) String() string { return "MsgReviewApplication" }
func (m *MsgReviewApplication) ProtoMessage()  {}
func (m *MsgReviewApplication) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Reviewer); err != nil {
		return fmt.Errorf("invalid reviewer: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	return nil
}

type MsgReviewApplicationResponse struct{}

func (m *MsgReviewApplicationResponse) Reset()         { *m = MsgReviewApplicationResponse{} }
func (m *MsgReviewApplicationResponse) String() string { return "MsgReviewApplicationResponse" }
func (m *MsgReviewApplicationResponse) ProtoMessage()  {}

type MsgSetAvailability struct {
	Agent        string   `json:"agent"`
	Available    bool     `json:"available"`
	Capabilities []string `json:"capabilities"`
}

func (m *MsgSetAvailability) Reset()         { *m = MsgSetAvailability{} }
func (m *MsgSetAvailability) String() string { return "MsgSetAvailability" }
func (m *MsgSetAvailability) ProtoMessage()  {}
func (m *MsgSetAvailability) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Agent); err != nil {
		return fmt.Errorf("invalid agent: %w", err)
	}
	return nil
}

type MsgSetAvailabilityResponse struct{}

func (m *MsgSetAvailabilityResponse) Reset()         { *m = MsgSetAvailabilityResponse{} }
func (m *MsgSetAvailabilityResponse) String() string { return "MsgSetAvailabilityResponse" }
func (m *MsgSetAvailabilityResponse) ProtoMessage()  {}

// --- Service Messages (4) ---

type MsgDeployService struct {
	Deployer        string      `json:"deployer"`
	ProjectId       string      `json:"project_id"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	ServiceType     ServiceType `json:"service_type"`
	ContractAddress string      `json:"contract_address"`
}

func (m *MsgDeployService) Reset()         { *m = MsgDeployService{} }
func (m *MsgDeployService) String() string { return "MsgDeployService" }
func (m *MsgDeployService) ProtoMessage()  {}
func (m *MsgDeployService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Deployer); err != nil {
		return fmt.Errorf("invalid deployer: %w", err)
	}
	if m.ProjectId == "" {
		return fmt.Errorf("project id cannot be empty")
	}
	if m.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}

type MsgDeployServiceResponse struct {
	ServiceId string `json:"service_id"`
}

func (m *MsgDeployServiceResponse) Reset()         { *m = MsgDeployServiceResponse{} }
func (m *MsgDeployServiceResponse) String() string { return "MsgDeployServiceResponse" }
func (m *MsgDeployServiceResponse) ProtoMessage()  {}

type MsgPauseService struct {
	Sender    string `json:"sender"`
	ServiceId string `json:"service_id"`
}

func (m *MsgPauseService) Reset()         { *m = MsgPauseService{} }
func (m *MsgPauseService) String() string { return "MsgPauseService" }
func (m *MsgPauseService) ProtoMessage()  {}
func (m *MsgPauseService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.ServiceId == "" {
		return fmt.Errorf("service id cannot be empty")
	}
	return nil
}

type MsgPauseServiceResponse struct{}

func (m *MsgPauseServiceResponse) Reset()         { *m = MsgPauseServiceResponse{} }
func (m *MsgPauseServiceResponse) String() string { return "MsgPauseServiceResponse" }
func (m *MsgPauseServiceResponse) ProtoMessage()  {}

type MsgResumeService struct {
	Sender    string `json:"sender"`
	ServiceId string `json:"service_id"`
}

func (m *MsgResumeService) Reset()         { *m = MsgResumeService{} }
func (m *MsgResumeService) String() string { return "MsgResumeService" }
func (m *MsgResumeService) ProtoMessage()  {}
func (m *MsgResumeService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.ServiceId == "" {
		return fmt.Errorf("service id cannot be empty")
	}
	return nil
}

type MsgResumeServiceResponse struct{}

func (m *MsgResumeServiceResponse) Reset()         { *m = MsgResumeServiceResponse{} }
func (m *MsgResumeServiceResponse) String() string { return "MsgResumeServiceResponse" }
func (m *MsgResumeServiceResponse) ProtoMessage()  {}

type MsgRetireService struct {
	Sender    string `json:"sender"`
	ServiceId string `json:"service_id"`
}

func (m *MsgRetireService) Reset()         { *m = MsgRetireService{} }
func (m *MsgRetireService) String() string { return "MsgRetireService" }
func (m *MsgRetireService) ProtoMessage()  {}
func (m *MsgRetireService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender: %w", err)
	}
	if m.ServiceId == "" {
		return fmt.Errorf("service id cannot be empty")
	}
	return nil
}

type MsgRetireServiceResponse struct{}

func (m *MsgRetireServiceResponse) Reset()         { *m = MsgRetireServiceResponse{} }
func (m *MsgRetireServiceResponse) String() string { return "MsgRetireServiceResponse" }
func (m *MsgRetireServiceResponse) ProtoMessage()  {}

// --- Service Interfaces ---

type MsgServer interface {
	// Projects
	CreateProject(context.Context, *MsgCreateProject) (*MsgCreateProjectResponse, error)
	ProposeProject(context.Context, *MsgProposeProject) (*MsgProposeProjectResponse, error)
	StartDevelopment(context.Context, *MsgStartDevelopment) (*MsgStartDevelopmentResponse, error)
	CompleteProject(context.Context, *MsgCompleteProject) (*MsgCompleteProjectResponse, error)
	PauseProject(context.Context, *MsgPauseProject) (*MsgPauseProjectResponse, error)
	ResumeProject(context.Context, *MsgResumeProject) (*MsgResumeProjectResponse, error)
	AbandonProject(context.Context, *MsgAbandonProject) (*MsgAbandonProjectResponse, error)
	SpawnChildProject(context.Context, *MsgSpawnChildProject) (*MsgSpawnChildProjectResponse, error)
	// Tasks
	AddTask(context.Context, *MsgAddTask) (*MsgAddTaskResponse, error)
	AssignTask(context.Context, *MsgAssignTask) (*MsgAssignTaskResponse, error)
	StartWork(context.Context, *MsgStartWork) (*MsgStartWorkResponse, error)
	SubmitDeliverable(context.Context, *MsgSubmitDeliverable) (*MsgSubmitDeliverableResponse, error)
	ApproveDeliverable(context.Context, *MsgApproveDeliverable) (*MsgApproveDeliverableResponse, error)
	RejectDeliverable(context.Context, *MsgRejectDeliverable) (*MsgRejectDeliverableResponse, error)
	ReopenTask(context.Context, *MsgReopenTask) (*MsgReopenTaskResponse, error)
	// Applications
	ApplyToProject(context.Context, *MsgApplyToProject) (*MsgApplyToProjectResponse, error)
	ReviewApplication(context.Context, *MsgReviewApplication) (*MsgReviewApplicationResponse, error)
	SetAvailability(context.Context, *MsgSetAvailability) (*MsgSetAvailabilityResponse, error)
	// Services
	DeployService(context.Context, *MsgDeployService) (*MsgDeployServiceResponse, error)
	PauseService(context.Context, *MsgPauseService) (*MsgPauseServiceResponse, error)
	ResumeService(context.Context, *MsgResumeService) (*MsgResumeServiceResponse, error)
	RetireService(context.Context, *MsgRetireService) (*MsgRetireServiceResponse, error)
}
