package types

import "cosmossdk.io/errors"

var (
	ErrScheduleNotFound      = errors.Register(ModuleName, 2, "vesting schedule not found")
	ErrNothingToClaim        = errors.Register(ModuleName, 3, "no vested tokens available to claim")
	ErrScheduleAlreadyExists = errors.Register(ModuleName, 4, "vesting schedule already exists for account")
	ErrTruthLinkBroken       = errors.Register(ModuleName, 5, "linked fact has been falsified")
	ErrCliffNotReached       = errors.Register(ModuleName, 6, "vesting cliff period not yet reached")
	ErrInvalidCategory       = errors.Register(ModuleName, 7, "invalid vesting category")
	ErrAlreadyFalsified      = errors.Register(ModuleName, 8, "claim already falsified")
	ErrSchedulePaused        = errors.Register(ModuleName, 9, "vesting schedule is paused")
	ErrInvalidRewardAmount   = errors.Register(ModuleName, 10, "invalid reward amount")
	ErrUnauthorized          = errors.Register(ModuleName, 11, "unauthorized")
)
