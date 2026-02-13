package types

import (
	"cosmossdk.io/errors"
)

var (
	// ErrAccountNotFound is returned when an account is not found
	ErrAccountNotFound = errors.Register(ModuleName, 2, "account not found")

	// ErrDIDNotFound is returned when a DID mapping is not found
	ErrDIDNotFound = errors.Register(ModuleName, 3, "DID not found")

	// ErrSessionExpired is returned when a session key has expired
	ErrSessionExpired = errors.Register(ModuleName, 4, "session key expired")

	// ErrMaxSessionKeys is returned when max session keys limit is reached
	ErrMaxSessionKeys = errors.Register(ModuleName, 5, "max session keys reached")

	// ErrUnauthorizedKey is returned when unauthorized key operation is attempted
	ErrUnauthorizedKey = errors.Register(ModuleName, 6, "unauthorized key operation")

	// ErrKeyRotationCooldown is returned when key rotation cooldown is active
	ErrKeyRotationCooldown = errors.Register(ModuleName, 7, "key rotation cooldown active")

	// ErrAccountFrozen is returned when account is frozen
	ErrAccountFrozen = errors.Register(ModuleName, 8, "account is frozen")

	// ErrInvalidDID is returned when DID format is invalid
	ErrInvalidDID = errors.Register(ModuleName, 9, "invalid DID format")

	// ErrDuplicateDID is returned when DID is already registered
	ErrDuplicateDID = errors.Register(ModuleName, 10, "DID already registered")

	// ErrSessionNotFound is returned when session key is not found
	ErrSessionNotFound = errors.Register(ModuleName, 11, "session key not found")

	// ErrInvalidKeyType is returned when key type is invalid
	ErrInvalidKeyType = errors.Register(ModuleName, 12, "invalid key type")

	// ErrSessionDurationExceeded is returned when session duration exceeds max
	ErrSessionDurationExceeded = errors.Register(ModuleName, 13, "session duration exceeds maximum")

	// ErrAccountAlreadyExists is returned when account already exists
	ErrAccountAlreadyExists = errors.Register(ModuleName, 14, "account already exists")

	// ErrInvalidPublicKey is returned when public key format is invalid
	ErrInvalidPublicKey = errors.Register(ModuleName, 15, "invalid public key")

	// ErrSessionCapabilityDenied is returned when session key lacks capability
	ErrSessionCapabilityDenied = errors.Register(ModuleName, 16, "session key capability denied")

	// ErrDIDResolutionFailed is returned when DID cannot be resolved
	ErrDIDResolutionFailed = errors.Register(ModuleName, 17, "DID resolution failed")

	// ErrUnauthorized is returned for unauthorized operations
	ErrUnauthorized = errors.Register(ModuleName, 18, "unauthorized")

	// ErrRecoveryNotFound is returned when no recovery request exists
	ErrRecoveryNotFound = errors.Register(ModuleName, 19, "recovery request not found")

	// ErrRecoveryAlreadyActive is returned when recovery is already in progress
	ErrRecoveryAlreadyActive = errors.Register(ModuleName, 20, "recovery already active")

	// ErrRecoveryNotExecutable is returned when recovery cannot be executed yet
	ErrRecoveryNotExecutable = errors.Register(ModuleName, 21, "recovery not executable")

	// ErrRecoveryConfigNotFound is returned when recovery config is not set
	ErrRecoveryConfigNotFound = errors.Register(ModuleName, 22, "recovery config not found")

	// ErrInvalidShard is returned when recovery shard is invalid
	ErrInvalidShard = errors.Register(ModuleName, 23, "invalid recovery shard")

	// ErrRecoveryChallenged is returned when recovery was successfully challenged
	ErrRecoveryChallenged = errors.Register(ModuleName, 24, "recovery has been challenged")

	// ErrAccountNotFrozen is returned when trying to unfreeze a non-frozen account
	ErrAccountNotFrozen = errors.Register(ModuleName, 25, "account is not frozen")

	// ErrBootstrapAlreadyClaimed is returned when bootstrap fund already claimed
	ErrBootstrapAlreadyClaimed = errors.Register(ModuleName, 26, "bootstrap fund already claimed")

	// ErrBootstrapDisabled is returned when bootstrap fund is disabled
	ErrBootstrapDisabled = errors.Register(ModuleName, 27, "bootstrap fund is disabled")

	// ErrBootstrapInsufficientFunds is returned when module account has insufficient funds
	ErrBootstrapInsufficientFunds = errors.Register(ModuleName, 28, "bootstrap fund has insufficient balance")

	// ErrDIDDerivationMismatch is returned when DID does not derive from the provided public key
	ErrDIDDerivationMismatch = errors.Register(ModuleName, 29, "DID does not derive from public key")
)
