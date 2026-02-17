package types

import "cosmossdk.io/errors"

var (
	ErrProviderNotFound       = errors.Register(ModuleName, 2, "compute provider not found")
	ErrProviderExists         = errors.Register(ModuleName, 3, "compute provider already registered")
	ErrProviderNotActive      = errors.Register(ModuleName, 4, "compute provider is not active")
	ErrInsufficientStake      = errors.Register(ModuleName, 5, "insufficient provider stake")
	ErrInsufficientCredits    = errors.Register(ModuleName, 6, "insufficient compute credits")
	ErrTicketNotFound         = errors.Register(ModuleName, 7, "redemption ticket not found")
	ErrTicketExpired          = errors.Register(ModuleName, 8, "redemption ticket has expired")
	ErrTicketAlreadyConfirmed = errors.Register(ModuleName, 9, "redemption ticket already confirmed")
	ErrTicketAlreadyDisputed  = errors.Register(ModuleName, 10, "redemption ticket already disputed")
	ErrPriceExceedsCeiling    = errors.Register(ModuleName, 11, "provider price exceeds ceiling")
	ErrProviderCapExceeded    = errors.Register(ModuleName, 12, "provider epoch cap exceeded")
	ErrCreditExpired          = errors.Register(ModuleName, 13, "compute credits have expired")
	ErrMaxBalanceExceeded     = errors.Register(ModuleName, 14, "credit balance would exceed maximum")
	ErrInvalidServiceType     = errors.Register(ModuleName, 15, "invalid compute service type")
	ErrHeartbeatTooEarly      = errors.Register(ModuleName, 16, "heartbeat submitted too early")
	ErrProviderSuspended      = errors.Register(ModuleName, 17, "provider is suspended due to missed heartbeat")
	ErrUnauthorized           = errors.Register(ModuleName, 18, "unauthorized")
	ErrInvalidAmount          = errors.Register(ModuleName, 19, "invalid amount")
	ErrProviderUnbonding      = errors.Register(ModuleName, 20, "provider is currently unbonding")
	ErrInvalidValidator       = errors.Register(ModuleName, 21, "invalid or unknown validator")
	ErrPoolInsufficient       = errors.Register(ModuleName, 22, "compute pool has insufficient funds")
	ErrSelfRedemption         = errors.Register(ModuleName, 23, "validator cannot redeem to own provider")
)
