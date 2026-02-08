package types

import "cosmossdk.io/errors"

var (
	ErrContractNotFound   = errors.Register(ModuleName, 2, "contract not found")
	ErrExecutionFailed    = errors.Register(ModuleName, 3, "contract execution failed")
	ErrOutOfGas           = errors.Register(ModuleName, 4, "contract execution out of gas")
	ErrInvalidBytecode    = errors.Register(ModuleName, 5, "invalid contract bytecode")
	ErrStaticCallViolation = errors.Register(ModuleName, 6, "state modification in static call context")
)
