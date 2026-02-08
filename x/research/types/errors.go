package types

import "cosmossdk.io/errors"

var (
	ErrSubmissionNotFound   = errors.Register(ModuleName, 2, "research submission not found")
	ErrInvalidSubmission    = errors.Register(ModuleName, 3, "invalid research submission")
	ErrBountyNotFound       = errors.Register(ModuleName, 4, "bounty not found")
	ErrBountyAlreadyClaimed = errors.Register(ModuleName, 5, "bounty already claimed")
	ErrInsufficientTreasury = errors.Register(ModuleName, 6, "insufficient research treasury balance")
)
