package types

import "cosmossdk.io/errors"

var (
	ErrProjectNotFound        = errors.Register(ModuleName, 2, "project not found")
	ErrServiceNotFound        = errors.Register(ModuleName, 3, "service not found")
	ErrTaskNotFound           = errors.Register(ModuleName, 4, "task not found")
	ErrSeedAlreadyPlanted     = errors.Register(ModuleName, 5, "opportunity seed already planted")
	ErrInvalidTaskState       = errors.Register(ModuleName, 6, "invalid task state transition")
	ErrInvalidPhaseTransition = errors.Register(ModuleName, 7, "invalid project phase transition")
	ErrNotFounder             = errors.Register(ModuleName, 8, "sender is not the project founder")
	ErrMaxTasksReached        = errors.Register(ModuleName, 9, "max tasks per project reached")
	ErrNotAssignee            = errors.Register(ModuleName, 10, "sender is not the task assignee")
	ErrMaxContributorsReached = errors.Register(ModuleName, 11, "max contributors reached")
	ErrMaxApplicationsReached = errors.Register(ModuleName, 12, "max applications reached")
	ErrAlreadyApplied         = errors.Register(ModuleName, 13, "already applied to this project")
	ErrApplicationNotFound    = errors.Register(ModuleName, 14, "application not found")
	ErrSeedNotFound           = errors.Register(ModuleName, 15, "opportunity seed not found")
	ErrSeedExpired            = errors.Register(ModuleName, 16, "opportunity seed has expired")
	ErrInvalidServiceState    = errors.Register(ModuleName, 17, "invalid service state transition")
	ErrMaxRejectionsReached   = errors.Register(ModuleName, 18, "max rejections reached for task")
	ErrNotContributor         = errors.Register(ModuleName, 19, "sender is not a project contributor")
	ErrInsufficientBudget     = errors.Register(ModuleName, 20, "insufficient project budget")
)
