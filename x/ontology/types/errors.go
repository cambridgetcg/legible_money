package types

import "cosmossdk.io/errors"

var (
	ErrDomainNotFound     = errors.Register(ModuleName, 2, "domain not found")
	ErrDomainExists       = errors.Register(ModuleName, 3, "domain already exists")
	ErrInvalidStratum     = errors.Register(ModuleName, 4, "invalid stratum")
	ErrProposalNotFound   = errors.Register(ModuleName, 5, "proposal not found")
	ErrProposalExpired    = errors.Register(ModuleName, 6, "proposal expired")
	ErrInsufficientStake  = errors.Register(ModuleName, 7, "insufficient stake for proposal")
	ErrDomainInactive     = errors.Register(ModuleName, 8, "domain is not active")
	ErrMaxDomainsReached  = errors.Register(ModuleName, 9, "max domains per stratum reached")
	ErrAlreadyVoted       = errors.Register(ModuleName, 10, "already voted on this proposal")
	ErrGoedelLimitReached = errors.Register(ModuleName, 11, "claim exceeds Goedel completeness boundary")
	ErrInvalidHierarchy   = errors.Register(ModuleName, 12, "invalid domain hierarchy")
	ErrProposalNotActive  = errors.Register(ModuleName, 13, "proposal is not active")
	ErrLinkExists         = errors.Register(ModuleName, 14, "cross-stratum link already exists")
	ErrLinkNotFound       = errors.Register(ModuleName, 15, "cross-stratum link not found")
)
