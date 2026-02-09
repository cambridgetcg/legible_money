package types

import "cosmossdk.io/errors"

var (
	ErrDomainNotFound           = errors.Register(ModuleName, 1, "domain not found")
	ErrAlreadyQualified         = errors.Register(ModuleName, 2, "already qualified in domain")
	ErrInsufficientStake        = errors.Register(ModuleName, 3, "insufficient stake for qualification")
	ErrNotQualified             = errors.Register(ModuleName, 4, "not qualified in domain")
	ErrValidatorNotFound        = errors.Register(ModuleName, 5, "validator not found")
	ErrInsufficientTrackRecord  = errors.Register(ModuleName, 6, "insufficient track record")
	ErrInsufficientEndorsements = errors.Register(ModuleName, 7, "insufficient endorsements")
	ErrNoInheritanceSource      = errors.Register(ModuleName, 8, "no qualification in parent stratum")
	ErrQualificationExpired     = errors.Register(ModuleName, 9, "qualification has expired")
	ErrQualificationSuspended   = errors.Register(ModuleName, 10, "qualification is suspended")
	ErrSelfEndorsement          = errors.Register(ModuleName, 11, "cannot endorse yourself")
	ErrEndorsementExpired       = errors.Register(ModuleName, 12, "endorsement has expired")
)
