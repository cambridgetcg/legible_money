package types

import "cosmossdk.io/errors"

var (
	ErrProposalNotFound      = errors.Register(ModuleName, 2, "governance proposal not found")
	ErrVotingPeriodEnded     = errors.Register(ModuleName, 3, "voting period has ended")
	ErrAlreadyVoted          = errors.Register(ModuleName, 4, "sender has already voted on this proposal")
	ErrInsufficientVotePower = errors.Register(ModuleName, 5, "insufficient voting power for quadratic vote weight")
	ErrDisbursementExceeded  = errors.Register(ModuleName, 6, "disbursement exceeds approved budget")
	ErrLIPNotFound           = errors.Register(ModuleName, 7, "LIP not found")
	ErrDisbursementNotFound  = errors.Register(ModuleName, 8, "disbursement not found")
	ErrInvalidLIPCategory    = errors.Register(ModuleName, 9, "invalid LIP category")
	ErrInvalidVoteDirection  = errors.Register(ModuleName, 10, "invalid vote direction")
	ErrInsufficientStake     = errors.Register(ModuleName, 11, "insufficient stake amount")
	ErrVotingNotActive       = errors.Register(ModuleName, 12, "voting is not active for this proposal")
	ErrTimelockNotExpired    = errors.Register(ModuleName, 13, "timelock period has not expired")
	ErrNotProposer           = errors.Register(ModuleName, 14, "only the proposer can perform this action")
	ErrInvalidLIPStage       = errors.Register(ModuleName, 15, "LIP is not in the correct stage for this action")
	ErrInvalidDisbStage      = errors.Register(ModuleName, 16, "disbursement is not in the correct stage for this action")
	ErrMaxVotesExceeded      = errors.Register(ModuleName, 17, "maximum votes per voter exceeded")
	ErrTerminalStage         = errors.Register(ModuleName, 18, "proposal is in a terminal stage")
	ErrUnauthorized          = errors.Register(ModuleName, 19, "unauthorized")
)
