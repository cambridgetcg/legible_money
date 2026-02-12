package types

import "cosmossdk.io/errors"

var (
	ErrInsufficientChallengeStake = errors.Register(ModuleName, 2, "insufficient challenge stake")
	ErrInsufficientEvidence       = errors.Register(ModuleName, 3, "insufficient evidence")
	ErrDeadlinePassed             = errors.Register(ModuleName, 4, "evidence deadline passed")
	ErrChallengeNotFound          = errors.Register(ModuleName, 5, "challenge not found")
	ErrInvalidChallengeStatus     = errors.Register(ModuleName, 6, "invalid challenge status for operation")
	ErrUnauthorizedResolver       = errors.Register(ModuleName, 7, "unauthorized resolver")
	ErrDomainNotFound             = errors.Register(ModuleName, 8, "domain not found")
	ErrMaxEvidenceExceeded        = errors.Register(ModuleName, 9, "maximum evidence pieces exceeded")
)
