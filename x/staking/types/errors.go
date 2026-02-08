package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrValidatorNotFound         = errors.Register(ModuleName, 1, "validator not found")
	ErrValidatorAlreadyExists    = errors.Register(ModuleName, 2, "validator already exists")
	ErrDelegationNotFound        = errors.Register(ModuleName, 3, "delegation not found")
	ErrUnbondingNotFound         = errors.Register(ModuleName, 4, "unbonding entry not found")
	ErrInsufficientStake         = errors.Register(ModuleName, 5, "insufficient stake")
	ErrInvalidTier               = errors.Register(ModuleName, 6, "invalid validator tier")
	ErrTierRequirementsNotMet    = errors.Register(ModuleName, 7, "tier requirements not met")
	ErrMaxApprenticesReached     = errors.Register(ModuleName, 8, "maximum apprentice validators reached")
	ErrMaxValidatorsReached      = errors.Register(ModuleName, 9, "maximum validators reached")
	ErrInvalidCommission         = errors.Register(ModuleName, 10, "invalid commission rate")
	ErrInvalidPubkey             = errors.Register(ModuleName, 11, "invalid consensus pubkey")
	ErrInvalidDID                = errors.Register(ModuleName, 12, "invalid DID")
	ErrCategoryNotAllowed        = errors.Register(ModuleName, 13, "category not allowed for validator tier")
	ErrValidatorInactive         = errors.Register(ModuleName, 14, "validator is inactive")
	ErrInsufficientSelfDelegation = errors.Register(ModuleName, 15, "insufficient self-delegation")
	ErrUnbondingNotMature        = errors.Register(ModuleName, 16, "unbonding period not completed")
	ErrSlashLimitExceeded        = errors.Register(ModuleName, 17, "validator slash limit exceeded")
	ErrDuplicatePubkey           = errors.Register(ModuleName, 18, "consensus pubkey already in use")
	ErrDuplicateDID              = errors.Register(ModuleName, 19, "DID already registered")
)
