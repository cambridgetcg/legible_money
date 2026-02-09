package types

import "cosmossdk.io/errors"

var (
	ErrDisputeNotFound        = errors.Register(ModuleName, 2, "dispute not found")
	ErrDisputeAlreadyResolved = errors.Register(ModuleName, 3, "dispute already resolved")
	ErrNotArbiter             = errors.Register(ModuleName, 4, "sender is not an assigned arbiter")
	ErrAlreadyVoted           = errors.Register(ModuleName, 5, "arbiter has already voted on this dispute")
	ErrInsufficientBond       = errors.Register(ModuleName, 6, "insufficient bond amount for dispute tier")
	ErrDisputeExpired         = errors.Register(ModuleName, 7, "dispute has expired")
	ErrNotInVotingPhase       = errors.Register(ModuleName, 8, "dispute is not in voting phase")
	ErrNotInEvidencePhase     = errors.Register(ModuleName, 9, "dispute is not in evidence phase")
	ErrMaxEscalation          = errors.Register(ModuleName, 10, "dispute already at maximum escalation tier")
	ErrFactNotFound           = errors.Register(ModuleName, 11, "fact not found in knowledge module")
	ErrFactNotDisputable      = errors.Register(ModuleName, 12, "fact is not in a disputable state")
	ErrMaxActiveDisputes      = errors.Register(ModuleName, 13, "maximum active disputes reached for this fact")
	ErrEscalationTooEarly     = errors.Register(ModuleName, 14, "escalation delay period has not elapsed")
	ErrInvalidBond            = errors.Register(ModuleName, 15, "invalid bond amount")
	ErrInvalidSide            = errors.Register(ModuleName, 16, "invalid evidence side")
	ErrEvidenceDeadlinePassed = errors.Register(ModuleName, 17, "evidence submission deadline has passed")
	ErrVotingDeadlinePassed   = errors.Register(ModuleName, 18, "voting deadline has passed")
	ErrNoArbitersAvailable    = errors.Register(ModuleName, 19, "no eligible arbiters available")
	ErrTierConfigNotFound     = errors.Register(ModuleName, 20, "tier configuration not found")
	ErrEvidenceHashMismatch   = errors.Register(ModuleName, 21, "evidence content does not match committed hash")
	ErrRefusalToReveal        = errors.Register(ModuleName, 22, "committed evidence was not revealed before deadline")
)
