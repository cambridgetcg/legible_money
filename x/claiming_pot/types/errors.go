package types

import "cosmossdk.io/errors"

var (
	ErrPotNotFound          = errors.Register(ModuleName, 2, "claiming pot not found")
	ErrNotPotMember         = errors.Register(ModuleName, 3, "sender is not a member of this pot")
	ErrPotAlreadyFinalized  = errors.Register(ModuleName, 4, "claiming pot already finalized")
	ErrInsufficientPotFunds = errors.Register(ModuleName, 5, "insufficient funds in claiming pot")
	ErrDuplicateContribution = errors.Register(ModuleName, 6, "duplicate contribution to this pot")
)
