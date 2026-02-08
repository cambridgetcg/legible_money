package types

import "cosmossdk.io/errors"

var (
	ErrTokenNotFound       = errors.Register(ModuleName, 2, "token not found")
	ErrInsufficientBalance = errors.Register(ModuleName, 3, "insufficient token balance")
	ErrTokenAlreadyExists  = errors.Register(ModuleName, 4, "token with this denom already exists")
	ErrUnauthorizedMint    = errors.Register(ModuleName, 5, "sender not authorized to mint this token")
	ErrAllowanceExceeded   = errors.Register(ModuleName, 6, "transfer amount exceeds allowance")
	ErrExpiredTokens       = errors.Register(ModuleName, 7, "tokens have expired")
	ErrInvalidPeriod       = errors.Register(ModuleName, 8, "invalid period")
	ErrRAINotEligible      = errors.Register(ModuleName, 9, "not eligible for RAI")
	ErrSweepInProgress     = errors.Register(ModuleName, 10, "sweep already in progress")
	ErrInvalidTransfer     = errors.Register(ModuleName, 11, "invalid transfer")
	ErrAccountNotFound     = errors.Register(ModuleName, 12, "expirable account not found")
	ErrInvalidAmount       = errors.Register(ModuleName, 13, "invalid amount")
	ErrDustAmount          = errors.Register(ModuleName, 14, "amount below dust threshold")
)
