package types

import "cosmossdk.io/errors"

var (
	ErrPotNotFound           = errors.Register(ModuleName, 2, "claiming pot not found")
	ErrNotPotMember          = errors.Register(ModuleName, 3, "sender is not a member of this pot")
	ErrPotAlreadyFinalized   = errors.Register(ModuleName, 4, "claiming pot already finalized")
	ErrInsufficientPotFunds  = errors.Register(ModuleName, 5, "insufficient funds in claiming pot")
	ErrDuplicateContribution = errors.Register(ModuleName, 6, "duplicate contribution to this pot")
	ErrPotNotActive          = errors.Register(ModuleName, 7, "pot is not in active status")
	ErrPotExpired            = errors.Register(ModuleName, 8, "pot has expired")
	ErrMaxClaimsReached      = errors.Register(ModuleName, 9, "maximum claims reached for this pot")
	ErrAlreadyClaimed        = errors.Register(ModuleName, 10, "account has reached max claims per account")
	ErrIneligible            = errors.Register(ModuleName, 11, "account does not meet eligibility requirements")
	ErrNotPotCreator         = errors.Register(ModuleName, 12, "sender is not the pot creator")
	ErrCreatorCannotClose    = errors.Register(ModuleName, 13, "pot config does not allow creator close")
	ErrInvalidConfig         = errors.Register(ModuleName, 14, "invalid pot configuration")
	ErrInsufficientDeposit   = errors.Register(ModuleName, 15, "deposit below minimum")
	ErrMaxPotsExceeded       = errors.Register(ModuleName, 16, "creator has reached max pots limit")
)
