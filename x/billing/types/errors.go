package types

import "cosmossdk.io/errors"

var (
	ErrProviderNotFound    = errors.Register(ModuleName, 2, "API provider not found")
	ErrProviderExists      = errors.Register(ModuleName, 3, "API provider already registered")
	ErrQuoteNotFound       = errors.Register(ModuleName, 4, "quote not found")
	ErrQuoteExpired        = errors.Register(ModuleName, 5, "quote has expired")
	ErrInsufficientStake   = errors.Register(ModuleName, 6, "insufficient provider stake")
	ErrInsufficientFunds   = errors.Register(ModuleName, 7, "insufficient funds for payment")
	ErrProviderInactive    = errors.Register(ModuleName, 8, "provider is not active")
	ErrProviderNotActive   = errors.Register(ModuleName, 9, "provider is not active")
	ErrInvalidPricingModel = errors.Register(ModuleName, 10, "invalid pricing model")
	ErrNoFactsRequested    = errors.Register(ModuleName, 11, "no facts requested in query")
	ErrFactNotFound        = errors.Register(ModuleName, 12, "referenced fact not found")
	ErrInvalidDistribution = errors.Register(ModuleName, 13, "payment distribution does not sum correctly")
	ErrZeroPayment         = errors.Register(ModuleName, 14, "payment amount must be positive")
	ErrProviderSelfQuery   = errors.Register(ModuleName, 15, "provider cannot query own endpoint")
	ErrInvalidBasisPoints  = errors.Register(ModuleName, 16, "basis points must not exceed 10000")
	ErrUnauthorized        = errors.Register(ModuleName, 17, "unauthorized")
	ErrQuoteAlreadyUsed    = errors.Register(ModuleName, 18, "quote already executed")
	ErrInvalidAmount       = errors.Register(ModuleName, 19, "invalid amount")
)
