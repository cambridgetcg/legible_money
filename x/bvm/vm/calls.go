package vm

import (
	"crypto/sha256"
	"math/big"

	"golang.org/x/crypto/sha3"
)

// ExecuteCall performs a sub-call with a new interpreter instance.
// Used by CALL, CALLCODE, DELEGATECALL, STATICCALL, CREATE, CREATE2.
func ExecuteCall(
	mode ExecutionMode,
	callerCtx *ExecutionContext,
	targetAddr []byte,
	callData []byte,
	gasForward uint64,
	value *big.Int,
	state StateDB,
	host HostFunctions,
	callDepth int,
) *ExecutionResult {
	if callDepth >= MaxCallDepth {
		return &ExecutionResult{
			Success: false,
			Error:   &ExecutionError{Code: ErrCodeCallDepthExceeded, Message: "call depth exceeded"},
		}
	}

	// Resolve bytecode
	var bytecode []byte
	var currentContract []byte
	var caller []byte

	switch mode {
	case ModeCall:
		bytecode = state.GetCode(targetAddr)
		currentContract = targetAddr
		caller = callerCtx.CurrentContract

	case ModeStaticCall:
		bytecode = state.GetCode(targetAddr)
		currentContract = targetAddr
		caller = callerCtx.CurrentContract

	case ModeDelegateCall:
		// Execute target's code in caller's storage context
		bytecode = state.GetCode(targetAddr)
		currentContract = callerCtx.CurrentContract // Caller's storage
		caller = callerCtx.Caller                   // Preserve original caller

	case ModeCallCode:
		// CALLCODE: execute target's code in caller's storage context.
		// Unlike DELEGATECALL, msg.sender = the calling contract (not the original caller).
		bytecode = state.GetCode(targetAddr)
		currentContract = callerCtx.CurrentContract // Caller's storage
		caller = callerCtx.CurrentContract           // msg.sender = calling contract

	case ModeCreate:
		bytecode = callData // Init code IS the calldata
		currentContract = targetAddr
		caller = callerCtx.CurrentContract
		callData = nil
	}

	if len(bytecode) == 0 && mode != ModeCreate {
		// Calling a non-existent contract or EOA — success with empty return
		return &ExecutionResult{
			Success:    true,
			ReturnData: nil,
			GasUsed:    0,
		}
	}

	subCtx := &ExecutionContext{
		Caller:             caller,
		CallerDID:          callerCtx.CallerDID,
		Origin:             callerCtx.Origin,
		OriginDID:          callerCtx.OriginDID,
		CurrentContract:    currentContract,
		BlockNumber:        callerCtx.BlockNumber,
		Timestamp:          callerCtx.Timestamp,
		GasPrice:           callerCtx.GasPrice,
		GasLimit:           gasForward,
		Value:              value,
		CallData:           callData,
		Mode:               mode,
		Bytecode:           bytecode,
		ChainID:            callerCtx.ChainID,
		Coinbase:           callerCtx.Coinbase,
		PrevRandao:         callerCtx.PrevRandao,
		BaseFee:            callerCtx.BaseFee,
		// P0-3: propagate execution context fields to sub-calls so agent opcodes,
		// governance gas overrides, and version gates work correctly in call chains.
		Capabilities:       callerCtx.Capabilities,
		GasSchedule:        callerCtx.GasSchedule,
		IsCommitPhase:      callerCtx.IsCommitPhase,
		ContractBvmVersion: callerCtx.ContractBvmVersion,
	}

	subInterp := NewInterpreter()
	subInterp.callDepth = callDepth + 1

	return subInterp.Execute(bytecode, subCtx, state, host)
}

// CalculateForwardGas computes gas to forward per 63/64 rule (EIP-150).
func CalculateForwardGas(available uint64, requested uint64) uint64 {
	maxForward := available - available/64
	if requested == 0 || requested > maxForward {
		return maxForward
	}
	return requested
}

// CreateAddress computes a deterministic CREATE address: keccak256(sender || nonce).
func CreateAddress(sender []byte, nonce uint64) []byte {
	data := make([]byte, len(sender)+8)
	copy(data, sender)
	for i := 7; i >= 0; i-- {
		data[len(sender)+i] = byte(nonce)
		nonce >>= 8
	}
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	hash := h.Sum(nil)
	return hash[12:] // 20-byte address
}

// Create2Address computes a deterministic CREATE2 address: keccak256(0xff || sender || salt || keccak256(initCode)).
func Create2Address(sender []byte, salt [32]byte, initCode []byte) []byte {
	codeHash := sha3.NewLegacyKeccak256()
	codeHash.Write(initCode)
	codeDigest := codeHash.Sum(nil)

	h := sha3.NewLegacyKeccak256()
	h.Write([]byte{0xFF})
	// Pad sender to 20 bytes
	padded := make([]byte, 20)
	if len(sender) >= 20 {
		copy(padded, sender[len(sender)-20:])
	} else {
		copy(padded[20-len(sender):], sender)
	}
	h.Write(padded)
	h.Write(salt[:])
	h.Write(codeDigest)
	hash := h.Sum(nil)
	return hash[12:]
}

// ContractAddressSHA256 generates a deterministic address using SHA256 (matching existing keeper).
func ContractAddressSHA256(creator string, blockHeight, nonce uint64) []byte {
	input := creator + "/" + uitoa(blockHeight) + "/" + uitoa(nonce)
	hash := sha256.Sum256([]byte(input))
	return hash[:20]
}

func uitoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	// Reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
