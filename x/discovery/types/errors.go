package types

import "cosmossdk.io/errors"

var (
	ErrAgentNotFound      = errors.Register(ModuleName, 2, "agent profile not found")
	ErrAgentExists        = errors.Register(ModuleName, 3, "agent already registered")
	ErrInsufficientStake  = errors.Register(ModuleName, 4, "insufficient registration stake")
	ErrMaxCapabilities    = errors.Register(ModuleName, 5, "max capabilities per agent exceeded")
	ErrCapabilityNotFound = errors.Register(ModuleName, 6, "capability not found")
	ErrCapabilityExists   = errors.Register(ModuleName, 7, "capability already registered")
	ErrAgentInactive      = errors.Register(ModuleName, 8, "agent profile is not active")
	ErrUnauthorized       = errors.Register(ModuleName, 9, "unauthorized: not profile owner")
)
