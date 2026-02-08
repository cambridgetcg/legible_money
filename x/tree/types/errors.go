package types

import "cosmossdk.io/errors"

var (
	ErrProjectNotFound    = errors.Register(ModuleName, 2, "project not found")
	ErrServiceNotFound    = errors.Register(ModuleName, 3, "service not found")
	ErrTaskNotFound       = errors.Register(ModuleName, 4, "task not found")
	ErrSeedAlreadyPlanted = errors.Register(ModuleName, 5, "opportunity seed already planted")
	ErrInvalidTaskState   = errors.Register(ModuleName, 6, "invalid task state transition")
)
