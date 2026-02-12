package types

import "cosmossdk.io/errors"

var (
	ErrOutflowExceeded = errors.Register(ModuleName, 2, "IBC outflow exceeds epoch rate limit")
	ErrInflowExceeded  = errors.Register(ModuleName, 3, "IBC inflow exceeds epoch rate limit")
	ErrFlowNotFound    = errors.Register(ModuleName, 4, "channel flow record not found")
)
