package types

import "cosmossdk.io/errors"

var (
	ErrHomeNotFound           = errors.Register(ModuleName, 2, "home not found for DID")
	ErrHomeAlreadyExists      = errors.Register(ModuleName, 3, "home already exists for DID")
	ErrNotHomeOwner           = errors.Register(ModuleName, 4, "signer is not home owner")
	ErrInvalidStatus          = errors.Register(ModuleName, 5, "invalid home status transition")
	ErrSessionNotFound        = errors.Register(ModuleName, 6, "session not found")
	ErrSessionExpired         = errors.Register(ModuleName, 7, "session has expired")
	ErrKeyNotFound            = errors.Register(ModuleName, 8, "key registration not found")
	ErrKeyAlreadyRegistered   = errors.Register(ModuleName, 9, "key hash already registered")
	ErrKeyRevoked             = errors.Register(ModuleName, 10, "key has been revoked")
	ErrInsufficientPermission = errors.Register(ModuleName, 11, "key lacks required permission")
	ErrSpendingLimitExceeded  = errors.Register(ModuleName, 12, "spending limit exceeded")
	ErrMaxKeysReached         = errors.Register(ModuleName, 13, "max keys per home reached")
	ErrMaxHomesReached        = errors.Register(ModuleName, 14, "max homes per account reached")
	ErrInvalidDeadmanConfig   = errors.Register(ModuleName, 15, "invalid deadman switch config")
	ErrMaxAlertsReached       = errors.Register(ModuleName, 16, "max pending alerts reached")
	ErrAlertNotFound          = errors.Register(ModuleName, 17, "alert not found")
	ErrMaxSessionsReached     = errors.Register(ModuleName, 18, "max concurrent sessions reached")
)
