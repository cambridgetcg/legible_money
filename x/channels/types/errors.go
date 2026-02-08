package types

import "cosmossdk.io/errors"

var (
	ErrChannelNotFound       = errors.Register(ModuleName, 2, "payment channel not found")
	ErrChannelAlreadyClosed  = errors.Register(ModuleName, 3, "payment channel already closed")
	ErrInvalidStateUpdate    = errors.Register(ModuleName, 4, "invalid channel state update")
	ErrDisputePeriodActive   = errors.Register(ModuleName, 5, "channel dispute period still active")
	ErrInsufficientDeposit   = errors.Register(ModuleName, 6, "insufficient channel deposit")
)
