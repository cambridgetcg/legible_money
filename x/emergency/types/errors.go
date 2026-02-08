package types

import "cosmossdk.io/errors"

var (
	ErrNotGuardian            = errors.Register(ModuleName, 2, "sender is not a Guardian-tier validator")
	ErrChainAlreadyHalted     = errors.Register(ModuleName, 3, "chain is already in halted state")
	ErrQuorumNotReached       = errors.Register(ModuleName, 4, "guardian quorum threshold not reached")
	ErrCooldownActive         = errors.Register(ModuleName, 5, "emergency cooldown period still active")
	ErrProposalLimitExceeded  = errors.Register(ModuleName, 6, "emergency proposal limit per epoch exceeded")
	ErrInvalidPhase           = errors.Register(ModuleName, 7, "ceremony is not in the expected phase")
	ErrNoCeremony             = errors.Register(ModuleName, 8, "no active ceremony found")
	ErrDuplicateVote          = errors.Register(ModuleName, 9, "duplicate vote from this guardian")
	ErrPrevoteRequired        = errors.Register(ModuleName, 10, "must prevote yes before precommit")
	ErrCeremonyTimedOut       = errors.Register(ModuleName, 11, "ceremony has timed out")
	ErrRevertDepthExceeded    = errors.Register(ModuleName, 12, "revert depth exceeds maximum allowed")
	ErrHaltRequired           = errors.Register(ModuleName, 13, "chain must be halted for this operation")
	ErrInsufficientGuardians  = errors.Register(ModuleName, 14, "insufficient total guardian stake")
	ErrStatusConflict         = errors.Register(ModuleName, 15, "operation conflicts with current emergency status")
	ErrCeremonyActive         = errors.Register(ModuleName, 16, "another ceremony is already active")
	ErrInvalidHaltReference   = errors.Register(ModuleName, 17, "halt ceremony ID does not match active halt")
)
