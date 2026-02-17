package types

import "cosmossdk.io/errors"

var (
	ErrNotQualified               = errors.Register(ModuleName, 2, "validator not qualified for domain")
	ErrSuspended                  = errors.Register(ModuleName, 3, "qualification is suspended")
	ErrExpired                    = errors.Register(ModuleName, 4, "qualification has expired")
	ErrInsufficientStake          = errors.Register(ModuleName, 5, "insufficient stake for qualification")
	ErrDomainNotFound             = errors.Register(ModuleName, 6, "domain not found")
	ErrAlreadyQualified           = errors.Register(ModuleName, 7, "validator already qualified for domain")
	ErrInsufficientEndorsers      = errors.Register(ModuleName, 8, "insufficient endorsers for qualification")
	ErrInvalidPathway             = errors.Register(ModuleName, 9, "invalid qualification pathway")
	ErrInsufficientTrackRecord    = errors.Register(ModuleName, 10, "insufficient track record for qualification")
	ErrNoInheritableQualification = errors.Register(ModuleName, 11, "no inheritable qualification found")
	ErrEndorserNotQualified       = errors.Register(ModuleName, 12, "endorser not qualified for domain")
	ErrSelfEndorsement            = errors.Register(ModuleName, 13, "cannot endorse own qualification")
	ErrMaxDepthExceeded           = errors.Register(ModuleName, 14, "maximum qualification depth exceeded")
)
