package types

import "cosmossdk.io/errors"

var (
	ErrNotRegistered         = errors.Register(ModuleName, 2, "interchain account not registered")
	ErrMaxAccountsReached    = errors.Register(ModuleName, 3, "max remote accounts per DID reached")
	ErrUnauthorizedKeyType   = errors.Register(ModuleName, 4, "ICA operations require operational key (not session)")
	ErrMsgTypeNotAllowed     = errors.Register(ModuleName, 5, "message type not allowed for this ICA")
	ErrNoLegibleAccount      = errors.Register(ModuleName, 6, "no Legible account found for address")
	ErrRegistrationCooldown  = errors.Register(ModuleName, 7, "ICA registration cooldown not elapsed")
	ErrAlreadyRegistered     = errors.Register(ModuleName, 8, "ICA already registered for this connection")
)
