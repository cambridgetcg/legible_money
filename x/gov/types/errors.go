package types

import "cosmossdk.io/errors"

var (
	ErrProposalNotFound       = errors.Register(ModuleName, 2, "governance proposal not found")
	ErrVotingPeriodEnded      = errors.Register(ModuleName, 3, "voting period has ended")
	ErrAlreadyVoted           = errors.Register(ModuleName, 4, "sender has already voted on this proposal")
	ErrInsufficientVotePower  = errors.Register(ModuleName, 5, "insufficient voting power for quadratic vote weight")
	ErrDisbursementExceeded   = errors.Register(ModuleName, 6, "disbursement exceeds approved budget")
)
