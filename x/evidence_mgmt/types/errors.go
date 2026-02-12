package types

import "cosmossdk.io/errors"

var (
	ErrEvidenceNotFound        = errors.Register(ModuleName, 2, "evidence not found")
	ErrInvalidEvidence         = errors.Register(ModuleName, 3, "invalid evidence submission")
	// Error codes 4-5 reserved (previously ErrOracleNotRegistered, ErrEvidenceExpired — unused, removed)
	ErrDuplicateEvidence       = errors.Register(ModuleName, 6, "duplicate evidence hash")
	ErrAuditNotFound           = errors.Register(ModuleName, 7, "audit not found")
	ErrOracleNotFound          = errors.Register(ModuleName, 8, "oracle not found")
	ErrInsufficientAuditStake  = errors.Register(ModuleName, 9, "insufficient audit stake")
	ErrAuditWindowClosed       = errors.Register(ModuleName, 10, "audit window has closed")
	ErrMaxEvidenceReached      = errors.Register(ModuleName, 11, "maximum evidence per claim reached")
	ErrOracleInactive          = errors.Register(ModuleName, 12, "oracle is inactive")
	ErrDuplicateAudit          = errors.Register(ModuleName, 13, "duplicate audit for this evidence")
	ErrInvalidScore            = errors.Register(ModuleName, 14, "invalid score value")
	ErrOracleAlreadyRegistered = errors.Register(ModuleName, 15, "oracle already registered")
	ErrInsufficientOracleStake = errors.Register(ModuleName, 16, "insufficient oracle stake")
	ErrInvalidParams           = errors.Register(ModuleName, 17, "invalid module parameters")
	ErrSelfAudit               = errors.Register(ModuleName, 18, "submitter cannot audit own evidence")
	ErrUnauthorizedOracle      = errors.Register(ModuleName, 19, "sender is not the oracle operator")
	ErrClaimNotFound           = errors.Register(ModuleName, 20, "claim not found")
	ErrInsufficientRewardFunds = errors.Register(ModuleName, 21, "insufficient module funds for rewards")
)
