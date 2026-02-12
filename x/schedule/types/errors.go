package types

import "cosmossdk.io/errors"

var (
	ErrProcessNotFound      = errors.Register(ModuleName, 2, "schedule process not found")
	ErrInvalidCondition     = errors.Register(ModuleName, 3, "invalid schedule condition")
	ErrMaxSchedulesExceeded = errors.Register(ModuleName, 4, "max active schedules per account exceeded")
	ErrInsufficientFee      = errors.Register(ModuleName, 5, "insufficient prepaid fee")
	ErrProcessNotActive     = errors.Register(ModuleName, 6, "schedule process is not active")
	ErrProcessNotPaused     = errors.Register(ModuleName, 7, "schedule process is not paused")
	ErrUnauthorized         = errors.Register(ModuleName, 8, "unauthorized: not process creator")
	ErrInvalidInterval      = errors.Register(ModuleName, 9, "interval below minimum")
	ErrInvalidFutureBlock   = errors.Register(ModuleName, 10, "execution block too far in the future")
	ErrInvalidExecuteAt     = errors.Register(ModuleName, 11, "execution block must be in the future")
	ErrProcessCompleted     = errors.Register(ModuleName, 12, "schedule process already completed")
	ErrProcessCancelled     = errors.Register(ModuleName, 13, "schedule process already cancelled")
	ErrCompoundDepthExceeded = errors.Register(ModuleName, 14, "compound condition depth exceeded (max 3)")
	ErrInvalidCallData      = errors.Register(ModuleName, 15, "invalid call data")
)
