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
)
