package types

import "cosmossdk.io/errors"

var (
	ErrChannelNotFound         = errors.Register(ModuleName, 2, "payment channel not found")
	ErrChannelAlreadyClosed    = errors.Register(ModuleName, 3, "payment channel already closed")
	ErrInvalidStateUpdate      = errors.Register(ModuleName, 4, "invalid channel state update")
	ErrDisputePeriodActive     = errors.Register(ModuleName, 5, "channel dispute period still active")
	ErrInsufficientDeposit     = errors.Register(ModuleName, 6, "insufficient channel deposit")
	ErrChannelExpired          = errors.Register(ModuleName, 7, "channel has expired")
	ErrNotChannelParty         = errors.Register(ModuleName, 8, "sender is not payer or provider")
	ErrInvalidNonce            = errors.Register(ModuleName, 9, "nonce must be higher than current")
	ErrDisputeAlreadyActive    = errors.Register(ModuleName, 10, "dispute already active for this channel")
	ErrNoActiveDispute         = errors.Register(ModuleName, 11, "no active dispute to resolve")
	ErrChannelDurationExceeded = errors.Register(ModuleName, 12, "channel duration exceeds maximum")
	ErrMaxChannelsExceeded     = errors.Register(ModuleName, 13, "payer has reached max open channels")
	ErrInvalidSignature        = errors.Register(ModuleName, 14, "invalid state update signature")
	ErrSpentExceedsDeposit     = errors.Register(ModuleName, 15, "spent amount exceeds deposited amount")
	ErrChannelNotOpen          = errors.Register(ModuleName, 16, "channel is not in open status")
	ErrNotExpired              = errors.Register(ModuleName, 17, "channel has not expired yet")
)
