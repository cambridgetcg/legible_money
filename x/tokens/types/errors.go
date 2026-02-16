package types

import "cosmossdk.io/errors"

var (
	ErrTokenNotFound       = errors.Register(ModuleName, 2, "token not found")
	ErrInsufficientBalance = errors.Register(ModuleName, 3, "insufficient token balance")
	ErrTokenAlreadyExists  = errors.Register(ModuleName, 4, "token with this denom already exists")
	ErrUnauthorizedMint    = errors.Register(ModuleName, 5, "sender not authorized to mint this token")
	ErrAllowanceExceeded   = errors.Register(ModuleName, 6, "transfer amount exceeds allowance")
	ErrInvalidTransfer     = errors.Register(ModuleName, 11, "invalid transfer")
	ErrInvalidAmount       = errors.Register(ModuleName, 13, "invalid amount")
)
