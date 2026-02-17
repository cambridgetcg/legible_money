package vm

import (
	"math/big"

	"golang.org/x/crypto/sha3"
)

// Interpreter executes BVM bytecode.
type Interpreter struct {
	stack      *Stack
	memory     *Memory
	gas        *GasMeter
	pc         int
	returnData []byte
	logs       []Log
	changes    []StateChange
	callDepth  int
	stopped    bool

	// EIP-2929 cold/warm access tracking for storage and addresses
	accessTracker *AccessTracker

	// Precomputed JUMPDEST set for the current bytecode.
	jumpDests map[int]bool
}

func NewInterpreter() *Interpreter {
	return &Interpreter{
		stack:         NewStack(),
		memory:        NewMemory(),
		accessTracker: NewAccessTracker(),
		jumpDests:     make(map[int]bool),
	}
}

// Execute runs bytecode with the given context, state, and optional host functions.
func (interp *Interpreter) Execute(
	bytecode []byte,
	ctx *ExecutionContext,
	state StateDB,
	host HostFunctions,
) *ExecutionResult {
	interp.pc = 0
	interp.returnData = nil
	interp.logs = nil
	interp.changes = nil
	interp.stopped = false
	interp.gas = NewGasMeter(ctx.GasLimit)
	interp.accessTracker = NewAccessTracker()

	// Pre-warm caller and contract addresses (EIP-2929: tx-origin and called address are warm)
	interp.accessTracker.MarkAddressWarm(ctx.Caller)
	interp.accessTracker.MarkAddressWarm(ctx.CurrentContract)
	if ctx.Origin != nil {
		interp.accessTracker.MarkAddressWarm(ctx.Origin)
	}
	if ctx.Coinbase != nil {
		interp.accessTracker.MarkAddressWarm(ctx.Coinbase)
	}

	// Enforce call depth limit (prevents recursive call DoS)
	if interp.callDepth > MaxCallDepth {
		return interp.errorResult(ErrCodeCallDepthExceeded, "call depth exceeded")
	}

	// Enforce bytecode size limit (EIP-170: 24 KB)
	if len(bytecode) > MaxBytecodeSize {
		return interp.errorResult(ErrCodeExecutionFailed, "bytecode exceeds max size")
	}

	// Pre-scan for JUMPDEST
	interp.jumpDests = scanJumpDests(bytecode)

	for interp.pc < len(bytecode) && !interp.stopped {
		opcode := bytecode[interp.pc]
		info := OpcodeTable[opcode]

		// Invalid opcode
		if info == nil {
			return interp.errorResult(ErrCodeInvalidOpcode, "invalid opcode: 0x"+hexByte(opcode))
		}

		// Version gate: reject opcodes requiring a newer BVM version
		if info.MinVersion > ctx.ContractBvmVersion {
			return interp.errorResult(ErrCodeInvalidOpcode, "opcode "+info.Mnemonic+" requires BVM version "+uitoa(uint64(info.MinVersion)))
		}

		// Activation gate: reject opcodes before their activation block
		if info.ActivationBlock > 0 && ctx.BlockNumber < info.ActivationBlock {
			return interp.errorResult(ErrCodeInvalidOpcode, "opcode "+info.Mnemonic+" not active until block "+uitoa(info.ActivationBlock))
		}

		// Deprecation gate: deprecated opcodes become no-ops (pop inputs, push zeros)
		if info.DeprecatedAtBlock > 0 && ctx.BlockNumber >= info.DeprecatedAtBlock {
			for i := 0; i < info.StackInputs; i++ {
				if _, err := interp.stack.Pop(); err != nil {
					return interp.errorResult(ErrCodeStackUnderflow, "stack underflow in deprecated "+info.Mnemonic)
				}
			}
			for i := 0; i < info.StackOutputs; i++ {
				interp.stack.Push(new(big.Int))
			}
			interp.pc++
			continue
		}

		// Staticcall restriction: reject state-modifying opcodes
		if ctx.Mode == ModeStaticCall && info.IsStateModifier {
			return interp.errorResult(ErrCodeStaticStateChange, "state modification in static call: "+info.Mnemonic)
		}

		// Consume base gas (with governance override if set)
		gasCost := ctx.GasSchedule.GasCostFor(opcode, info.GasCost)
		if gasCost > 0 {
			if err := interp.gas.Consume(gasCost); err != nil {
				return interp.errorResult(ErrCodeOutOfGas, "out of gas at "+info.Mnemonic)
			}
		}

		// Dispatch
		action := interp.executeOpcode(opcode, bytecode, ctx, state, host)

		switch action {
		case actionContinue:
			interp.pc++
		case actionStop:
			return interp.successResult()
		case actionReturn:
			return interp.successResult()
		case actionRevert:
			return interp.revertResult()
		case actionError:
			// Error already captured; return last error
			return interp.errorResult(ErrCodeExecutionFailed, "execution failed at "+info.Mnemonic)
		}
	}

	return interp.successResult()
}

type opcodeAction int

const (
	actionContinue opcodeAction = iota
	actionStop
	actionReturn
	actionRevert
	actionError
)

func (interp *Interpreter) executeOpcode(
	opcode byte,
	bytecode []byte,
	ctx *ExecutionContext,
	state StateDB,
	host HostFunctions,
) opcodeAction {
	// Helper closures for stack operations
	pop := func() (*big.Int, bool) {
		v, err := interp.stack.Pop()
		if err != nil {
			return nil, false
		}
		return v, true
	}
	push := func(v *big.Int) bool {
		if err := interp.stack.Push(v); err != nil {
			return false
		}
		return true
	}

	switch opcode {
	// --- Arithmetic ---
	case STOP:
		return actionStop

	case ADD:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		push(Mod256(new(big.Int).Add(a, b)))

	case MUL:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		push(Mod256(new(big.Int).Mul(a, b)))

	case SUB:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		result := new(big.Int).Sub(a, b)
		if result.Sign() < 0 {
			result.Add(result, WordMod)
		}
		push(Mod256(result))

	case DIV:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if b.Sign() == 0 {
			push(new(big.Int))
		} else {
			push(new(big.Int).Div(a, b))
		}

	case SDIV:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if b.Sign() == 0 {
			push(new(big.Int))
		} else {
			sa := ToSigned(a)
			sb := ToSigned(b)
			result := new(big.Int).Quo(sa, sb) // Quo truncates toward zero
			push(FromSigned(result))
		}

	case MOD:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if b.Sign() == 0 {
			push(new(big.Int))
		} else {
			push(new(big.Int).Mod(a, b))
		}

	case SMOD:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if b.Sign() == 0 {
			push(new(big.Int))
		} else {
			sa := ToSigned(a)
			sb := ToSigned(b)
			result := new(big.Int).Rem(sa, sb)
			push(FromSigned(result))
		}

	case ADDMOD:
		a, ok1 := pop()
		b, ok2 := pop()
		n, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		if n.Sign() == 0 {
			push(new(big.Int))
		} else {
			sum := new(big.Int).Add(a, b)
			push(new(big.Int).Mod(sum, n))
		}

	case MULMOD:
		a, ok1 := pop()
		b, ok2 := pop()
		n, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		if n.Sign() == 0 {
			push(new(big.Int))
		} else {
			product := new(big.Int).Mul(a, b)
			push(new(big.Int).Mod(product, n))
		}

	case EXP:
		base, ok1 := pop()
		exp, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		// Dynamic gas: 50 per byte of exponent (EIP-160)
		expBytes := (exp.BitLen() + 7) / 8
		if expBytes > 0 {
			if err := interp.gas.Consume(GasExpPerByte * uint64(expBytes)); err != nil {
				return actionError
			}
		}
		if exp.Sign() == 0 {
			push(big.NewInt(1))
		} else {
			result := new(big.Int).Exp(base, exp, WordMod)
			push(result)
		}

	case SIGNEXTEND:
		b, ok1 := pop()
		x, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if b.Cmp(big.NewInt(30)) <= 0 {
			bitIdx := uint(b.Uint64()*8 + 7)
			if x.Bit(int(bitIdx)) == 1 {
				// Set all higher bits
				mask := new(big.Int).Sub(new(big.Int).Lsh(One, bitIdx), One)
				notMask := new(big.Int).Xor(mask, MaxUint256)
				push(Mod256(new(big.Int).Or(x, notMask)))
			} else {
				// Clear all higher bits
				mask := new(big.Int).Sub(new(big.Int).Lsh(One, bitIdx+1), One)
				push(new(big.Int).And(x, mask))
			}
		} else {
			push(x)
		}

	// --- Comparison & Bitwise ---
	case LT:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if a.Cmp(b) < 0 {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case GT:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if a.Cmp(b) > 0 {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case SLT:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		sa := ToSigned(a)
		sb := ToSigned(b)
		if sa.Cmp(sb) < 0 {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case SGT:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		sa := ToSigned(a)
		sb := ToSigned(b)
		if sa.Cmp(sb) > 0 {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case EQ:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if a.Cmp(b) == 0 {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case ISZERO:
		a, ok := pop()
		if !ok {
			return actionError
		}
		if a.Sign() == 0 {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case AND:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		push(new(big.Int).And(a, b))

	case OR:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		push(new(big.Int).Or(a, b))

	case XOR:
		a, ok1 := pop()
		b, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		push(new(big.Int).Xor(a, b))

	case NOT:
		a, ok := pop()
		if !ok {
			return actionError
		}
		push(new(big.Int).Xor(a, MaxUint256))

	case BYTE:
		i, ok1 := pop()
		x, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if i.Cmp(big.NewInt(31)) > 0 {
			push(new(big.Int))
		} else {
			b32 := WordToBytes32(x)
			push(big.NewInt(int64(b32[i.Uint64()])))
		}

	case SHL:
		shift, ok1 := pop()
		val, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if shift.Cmp(big.NewInt(255)) > 0 {
			push(new(big.Int))
		} else {
			result := new(big.Int).Lsh(val, uint(shift.Uint64()))
			push(Mod256(result))
		}

	case SHR:
		shift, ok1 := pop()
		val, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if shift.Cmp(big.NewInt(255)) > 0 {
			push(new(big.Int))
		} else {
			push(new(big.Int).Rsh(val, uint(shift.Uint64())))
		}

	case SAR:
		shift, ok1 := pop()
		val, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		signed := ToSigned(val)
		if shift.Cmp(big.NewInt(256)) >= 0 {
			if signed.Sign() >= 0 {
				push(new(big.Int))
			} else {
				push(new(big.Int).Set(MaxUint256))
			}
		} else {
			result := new(big.Int).Rsh(signed, uint(shift.Uint64()))
			push(FromSigned(result))
		}

	// --- SHA3 ---
	case SHA3:
		offset, ok1 := pop()
		length, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		off := int(offset.Uint64())
		ln := int(length.Uint64())
		// Dynamic gas: 6 per word
		words := (ln + 31) / 32
		if words > 0 {
			if err := interp.gas.Consume(GasSHA3PerWord * uint64(words)); err != nil {
				return actionError
			}
		}
		// Memory expansion
		if ln > 0 {
			if gasCost, err := interp.memory.Expand(off + ln); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
		}
		data := interp.memory.Read(off, ln)
		h := sha3.NewLegacyKeccak256()
		h.Write(data)
		hash := h.Sum(nil)
		push(new(big.Int).SetBytes(hash))

	// --- Environmental ---
	case ADDRESS:
		push(new(big.Int).SetBytes(ctx.CurrentContract))

	case BALANCE:
		addr, ok := pop()
		if !ok {
			return actionError
		}
		b32 := WordToBytes32(addr)
		addrBytes := b32[12:] // last 20 bytes
		// EIP-2929: charge cold/warm gas for address access
		accessGas := interp.accessTracker.AddressGas(addrBytes)
		if err := interp.gas.Consume(accessGas); err != nil {
			return actionError
		}
		bal := state.GetBalance(addrBytes)
		push(bal)

	case ORIGIN:
		push(new(big.Int).SetBytes(ctx.Origin))

	case CALLER:
		push(new(big.Int).SetBytes(ctx.Caller))

	case CALLVALUE:
		if ctx.Value != nil {
			push(new(big.Int).Set(ctx.Value))
		} else {
			push(new(big.Int))
		}

	case CALLDATALOAD:
		off, ok := pop()
		if !ok {
			return actionError
		}
		offset64 := off.Uint64()
		data := make([]byte, 32)
		for i := 0; i < 32; i++ {
			idx := int(offset64) + i
			if idx < len(ctx.CallData) {
				data[i] = ctx.CallData[idx]
			}
		}
		push(new(big.Int).SetBytes(data))

	case CALLDATASIZE:
		push(big.NewInt(int64(len(ctx.CallData))))

	case CALLDATACOPY:
		destOff, ok1 := pop()
		dataOff, ok2 := pop()
		length, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		ln := int(length.Uint64())
		if ln > 0 {
			words := (ln + 31) / 32
			if err := interp.gas.Consume(GasCopy * uint64(words)); err != nil {
				return actionError
			}
			if gasCost, err := interp.memory.Expand(int(destOff.Uint64()) + ln); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
			src := make([]byte, ln)
			srcOff := int(dataOff.Uint64())
			for i := 0; i < ln; i++ {
				if srcOff+i < len(ctx.CallData) {
					src[i] = ctx.CallData[srcOff+i]
				}
			}
			interp.memory.Write(int(destOff.Uint64()), src)
		}

	case CODESIZE:
		push(big.NewInt(int64(len(ctx.Bytecode))))

	case CODECOPY:
		destOff, ok1 := pop()
		codeOff, ok2 := pop()
		length, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		ln := int(length.Uint64())
		if ln > 0 {
			words := (ln + 31) / 32
			if err := interp.gas.Consume(GasCopy * uint64(words)); err != nil {
				return actionError
			}
			if gasCost, err := interp.memory.Expand(int(destOff.Uint64()) + ln); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
			src := make([]byte, ln)
			srcOff := int(codeOff.Uint64())
			for i := 0; i < ln; i++ {
				if srcOff+i < len(ctx.Bytecode) {
					src[i] = ctx.Bytecode[srcOff+i]
				}
			}
			interp.memory.Write(int(destOff.Uint64()), src)
		}

	case GASPRICE:
		if ctx.GasPrice != nil {
			push(new(big.Int).Set(ctx.GasPrice))
		} else {
			push(new(big.Int))
		}

	case EXTCODESIZE:
		addr, ok := pop()
		if !ok {
			return actionError
		}
		b32 := WordToBytes32(addr)
		addrBytes := b32[12:]
		// EIP-2929: charge cold/warm gas for address access
		accessGas := interp.accessTracker.AddressGas(addrBytes)
		if err := interp.gas.Consume(accessGas); err != nil {
			return actionError
		}
		push(big.NewInt(int64(state.GetCodeSize(addrBytes))))

	case EXTCODECOPY:
		addr, ok1 := pop()
		destOff, ok2 := pop()
		codeOff, ok3 := pop()
		length, ok4 := pop()
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return actionError
		}
		b32 := WordToBytes32(addr)
		addrBytes := b32[12:]
		// EIP-2929: charge cold/warm gas for address access
		accessGas := interp.accessTracker.AddressGas(addrBytes)
		if err := interp.gas.Consume(accessGas); err != nil {
			return actionError
		}
		ln := int(length.Uint64())
		if ln > 0 {
			words := (ln + 31) / 32
			if err := interp.gas.Consume(GasCopy * uint64(words)); err != nil {
				return actionError
			}
			if gasCost, err := interp.memory.Expand(int(destOff.Uint64()) + ln); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
			code := state.GetCode(addrBytes)
			src := make([]byte, ln)
			srcOff := int(codeOff.Uint64())
			for i := 0; i < ln; i++ {
				if srcOff+i < len(code) {
					src[i] = code[srcOff+i]
				}
			}
			interp.memory.Write(int(destOff.Uint64()), src)
		}

	case RETURNDATASIZE:
		push(big.NewInt(int64(len(interp.returnData))))

	case RETURNDATACOPY:
		destOff, ok1 := pop()
		dataOff, ok2 := pop()
		length, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		ln := int(length.Uint64())
		srcOff := int(dataOff.Uint64())
		if srcOff+ln > len(interp.returnData) {
			return actionError // Out of bounds
		}
		if ln > 0 {
			words := (ln + 31) / 32
			if err := interp.gas.Consume(GasCopy * uint64(words)); err != nil {
				return actionError
			}
			if gasCost, err := interp.memory.Expand(int(destOff.Uint64()) + ln); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
			interp.memory.Write(int(destOff.Uint64()), interp.returnData[srcOff:srcOff+ln])
		}

	case EXTCODEHASH:
		addr, ok := pop()
		if !ok {
			return actionError
		}
		b32 := WordToBytes32(addr)
		addrBytes := b32[12:]
		// EIP-2929: charge cold/warm gas for address access
		accessGas := interp.accessTracker.AddressGas(addrBytes)
		if err := interp.gas.Consume(accessGas); err != nil {
			return actionError
		}
		hash := state.GetCodeHash(addrBytes)
		if hash == nil {
			push(new(big.Int))
		} else {
			push(new(big.Int).SetBytes(hash))
		}

	// --- Block Info ---
	case BLOCKHASH:
		_, ok := pop() // Block number - we can't look up actual block hashes
		if !ok {
			return actionError
		}
		push(new(big.Int)) // Return 0 (no block hash oracle)

	case COINBASE:
		if ctx.Coinbase != nil {
			push(new(big.Int).SetBytes(ctx.Coinbase))
		} else {
			push(new(big.Int))
		}

	case TIMESTAMP:
		push(new(big.Int).SetUint64(ctx.Timestamp))

	case NUMBER:
		push(new(big.Int).SetUint64(ctx.BlockNumber))

	case PREVRANDAO:
		if ctx.PrevRandao != nil {
			push(new(big.Int).Set(ctx.PrevRandao))
		} else {
			push(new(big.Int))
		}

	case GASLIMIT:
		push(new(big.Int).SetUint64(ctx.GasLimit))

	case CHAINID:
		if ctx.ChainID != nil {
			push(new(big.Int).Set(ctx.ChainID))
		} else {
			push(new(big.Int))
		}

	case SELFBALANCE:
		bal := state.GetBalance(ctx.CurrentContract)
		push(bal)

	case BASEFEE:
		if ctx.BaseFee != nil {
			push(new(big.Int).Set(ctx.BaseFee))
		} else {
			push(new(big.Int))
		}

	// --- Stack, Memory, Storage ---
	case POP:
		if _, ok := pop(); !ok {
			return actionError
		}

	case MLOAD:
		offset, ok := pop()
		if !ok {
			return actionError
		}
		off := int(offset.Uint64())
		if gasCost, err := interp.memory.Expand(off + 32); err != nil {
			return actionError
		} else if gasCost > 0 {
			if err := interp.gas.Consume(gasCost); err != nil {
				return actionError
			}
		}
		push(interp.memory.MLoad(off))

	case MSTORE:
		offset, ok1 := pop()
		val, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		off := int(offset.Uint64())
		if gasCost, err := interp.memory.Expand(off + 32); err != nil {
			return actionError
		} else if gasCost > 0 {
			if err := interp.gas.Consume(gasCost); err != nil {
				return actionError
			}
		}
		interp.memory.MStore(off, val)

	case MSTORE8:
		offset, ok1 := pop()
		val, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		off := int(offset.Uint64())
		if gasCost, err := interp.memory.Expand(off + 1); err != nil {
			return actionError
		} else if gasCost > 0 {
			if err := interp.gas.Consume(gasCost); err != nil {
				return actionError
			}
		}
		interp.memory.SetByte(off, byte(val.Uint64()&0xFF))

	case SLOAD:
		key, ok := pop()
		if !ok {
			return actionError
		}
		b32 := WordToBytes32(key)
		// EIP-2929: charge cold/warm gas for storage access
		accessGas := interp.accessTracker.StorageGas(ctx.CurrentContract, b32[:])
		if err := interp.gas.Consume(accessGas); err != nil {
			return actionError
		}
		val := state.GetStorage(ctx.CurrentContract, b32[:])
		if val == nil {
			push(new(big.Int))
		} else {
			push(new(big.Int).SetBytes(val))
		}

	case SSTORE:
		key, ok1 := pop()
		val, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		b32Key := WordToBytes32(key)
		// EIP-2929: charge cold gas on first access to this slot
		if !interp.accessTracker.IsStorageWarm(ctx.CurrentContract, b32Key[:]) {
			if err := interp.gas.Consume(GasColdSloadCost); err != nil {
				return actionError
			}
			interp.accessTracker.MarkStorageWarm(ctx.CurrentContract, b32Key[:])
		}
		oldVal := state.GetStorage(ctx.CurrentContract, b32Key[:])
		oldWord := new(big.Int)
		if oldVal != nil {
			oldWord.SetBytes(oldVal)
		}

		// Enforce per-contract storage slot limit on new slot creation
		if oldWord.Sign() == 0 && val.Sign() != 0 {
			if sc, ok := state.(StorageCounter); ok {
				if sc.GetStorageSlotCount(ctx.CurrentContract) >= MaxStorageSlotsPerContract {
					return actionError // Storage limit exceeded
				}
			}
		}

		// Dynamic gas: EIP-2200 simplified
		if oldWord.Sign() == 0 && val.Sign() != 0 {
			if err := interp.gas.Consume(GasSstoreSet); err != nil {
				return actionError
			}
		} else if oldWord.Sign() != 0 && val.Sign() == 0 {
			if err := interp.gas.Consume(GasSstoreClear); err != nil {
				return actionError
			}
		} else {
			if err := interp.gas.Consume(GasSstoreReset); err != nil {
				return actionError
			}
		}

		b32Val := WordToBytes32(val)
		state.SetStorage(ctx.CurrentContract, b32Key[:], b32Val[:])

		interp.changes = append(interp.changes, StateChange{
			Type:     StateChangeStorage,
			Address:  ctx.CurrentContract,
			Key:      new(big.Int).SetBytes(b32Key[:]).Text(16),
			OldValue: oldWord,
			NewValue: new(big.Int).Set(val),
		})

	case JUMP:
		dest, ok := pop()
		if !ok {
			return actionError
		}
		d := int(dest.Int64())
		if !interp.jumpDests[d] {
			return actionError // Invalid jump destination
		}
		interp.pc = d
		return actionContinue // Don't increment PC

	case JUMPI:
		dest, ok1 := pop()
		cond, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		if cond.Sign() != 0 {
			d := int(dest.Int64())
			if !interp.jumpDests[d] {
				return actionError
			}
			interp.pc = d
			return actionContinue
		}
		// Fall through (condition false)

	case PC:
		push(big.NewInt(int64(interp.pc)))

	case MSIZE:
		push(big.NewInt(int64(interp.memory.Size())))

	case GAS:
		push(new(big.Int).SetUint64(interp.gas.Remaining()))

	case JUMPDEST:
		// No-op marker

	case PUSH0:
		push(new(big.Int))

	// --- System ---
	case RETURN:
		offset, ok1 := pop()
		length, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		off := int(offset.Uint64())
		ln := int(length.Uint64())
		if ln > 0 {
			if gasCost, err := interp.memory.Expand(off + ln); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
		}
		interp.returnData = interp.memory.Read(off, ln)
		return actionReturn

	case REVERT:
		offset, ok1 := pop()
		length, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		off := int(offset.Uint64())
		ln := int(length.Uint64())
		if ln > 0 {
			if gasCost, err := interp.memory.Expand(off + ln); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
		}
		interp.returnData = interp.memory.Read(off, ln)
		return actionRevert

	case INVALID:
		// Consume all remaining gas
		interp.gas.Consume(interp.gas.Remaining())
		return actionError

	case SELFDESTRUCT:
		// Pop beneficiary address but don't actually self-destruct (deprecated in EVM)
		if _, ok := pop(); !ok {
			return actionError
		}
		return actionStop

	// --- Logging ---
	case LOG0, LOG1, LOG2, LOG3, LOG4:
		numTopics := int(opcode - LOG0)
		offset, ok1 := pop()
		length, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		topics := make([][]byte, numTopics)
		for i := 0; i < numTopics; i++ {
			t, ok := pop()
			if !ok {
				return actionError
			}
			b := WordToBytes32(t)
			topics[i] = b[:]
		}
		off := int(offset.Uint64())
		ln := int(length.Uint64())
		// Dynamic gas: 8 per byte of data
		if ln > 0 {
			if err := interp.gas.Consume(GasLogData * uint64(ln)); err != nil {
				return actionError
			}
			if gasCost, err := interp.memory.Expand(off + ln); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
		}
		data := interp.memory.Read(off, ln)
		interp.logs = append(interp.logs, Log{
			Address: ctx.CurrentContract,
			Topics:  topics,
			Data:    data,
		})

	// --- CALL/CREATE (ZD-X03 FIX: proper gas accounting + error signal) ---
	// These opcodes are not yet implemented but must consume gas and
	// leave the stack in a well-defined state (0 = failure).
	case CALL, CALLCODE:
		// Pop 7 args: gas, addr, value, argsOff, argsLen, retOff, retLen
		gasArg, ok1 := pop()
		addrArg, ok2 := pop()
		valueArg, ok3 := pop()
		argsOffArg, ok4 := pop()
		argsLenArg, ok5 := pop()
		retOffArg, ok6 := pop()
		retLenArg, ok7 := pop()
		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 || !ok7 {
			return actionError
		}

		// Expand memory for input and output regions
		argsOff := int(argsOffArg.Uint64())
		argsLen := int(argsLenArg.Uint64())
		retOff := int(retOffArg.Uint64())
		retLen := int(retLenArg.Uint64())
		maxEnd := argsOff + argsLen
		if retOff+retLen > maxEnd {
			maxEnd = retOff + retLen
		}
		if maxEnd > 0 {
			if gasCost, err := interp.memory.Expand(maxEnd); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
		}

		// Gas: base cost already charged by OpcodeTable. Forward gas per 63/64 rule.
		gasForward := CalculateForwardGas(interp.gas.Remaining(), gasArg.Uint64())
		if err := interp.gas.Consume(gasForward); err != nil {
			return actionError
		}

		// Extra gas for value transfer
		if valueArg.Sign() > 0 {
			if err := interp.gas.Consume(GasCallValue); err != nil {
				return actionError
			}
		}

		// Prepare call
		callData := interp.memory.Read(argsOff, argsLen)
		targetAddr := WordToBytes20(addrArg)

		mode := ModeCall
		if opcode == CALLCODE {
			mode = ModeCall // CALLCODE uses caller's storage (similar to DELEGATECALL without msg.sender preservation)
		}

		// STATICCALL inheritance: if parent is static, sub-calls must also be static
		if ctx.Mode == ModeStaticCall {
			mode = ModeStaticCall
		}

		// Create snapshot for REVERT rollback
		snapshot := NewSnapshot(state)

		result := ExecuteCall(mode, ctx, targetAddr, callData, gasForward, valueArg, snapshot, host, interp.callDepth)

		// Refund unused gas
		if result.GasUsed < gasForward {
			interp.gas.Refund(gasForward - result.GasUsed)
		}

		if result.Success {
			snapshot.Commit()
			push(big.NewInt(1))
		} else {
			// Snapshot discarded — state rolled back
			push(new(big.Int)) // 0 = failure
		}

		// Copy return data to memory
		interp.returnData = result.ReturnData
		if result.ReturnData != nil && retLen > 0 {
			copyLen := retLen
			if len(result.ReturnData) < copyLen {
				copyLen = len(result.ReturnData)
			}
			interp.memory.Write(retOff, result.ReturnData[:copyLen])
		}

		// Collect logs from sub-call
		if result.Success && result.Logs != nil {
			interp.logs = append(interp.logs, result.Logs...)
		}

	case DELEGATECALL, STATICCALL:
		// Pop 6 args: gas, addr, argsOff, argsLen, retOff, retLen (no value)
		gasArg, ok1 := pop()
		addrArg, ok2 := pop()
		argsOffArg, ok3 := pop()
		argsLenArg, ok4 := pop()
		retOffArg, ok5 := pop()
		retLenArg, ok6 := pop()
		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 {
			return actionError
		}

		argsOff := int(argsOffArg.Uint64())
		argsLen := int(argsLenArg.Uint64())
		retOff := int(retOffArg.Uint64())
		retLen := int(retLenArg.Uint64())
		maxEnd := argsOff + argsLen
		if retOff+retLen > maxEnd {
			maxEnd = retOff + retLen
		}
		if maxEnd > 0 {
			if gasCost, err := interp.memory.Expand(maxEnd); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
		}

		gasForward := CalculateForwardGas(interp.gas.Remaining(), gasArg.Uint64())
		if err := interp.gas.Consume(gasForward); err != nil {
			return actionError
		}

		callData := interp.memory.Read(argsOff, argsLen)
		targetAddr := WordToBytes20(addrArg)

		mode := ModeDelegateCall
		if opcode == STATICCALL {
			mode = ModeStaticCall
		}
		// Static inheritance
		if ctx.Mode == ModeStaticCall {
			mode = ModeStaticCall
		}

		snapshot := NewSnapshot(state)
		result := ExecuteCall(mode, ctx, targetAddr, callData, gasForward, new(big.Int), snapshot, host, interp.callDepth)

		if result.GasUsed < gasForward {
			interp.gas.Refund(gasForward - result.GasUsed)
		}

		if result.Success {
			snapshot.Commit()
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

		interp.returnData = result.ReturnData
		if result.ReturnData != nil && retLen > 0 {
			copyLen := retLen
			if len(result.ReturnData) < copyLen {
				copyLen = len(result.ReturnData)
			}
			interp.memory.Write(retOff, result.ReturnData[:copyLen])
		}

		if result.Success && result.Logs != nil {
			interp.logs = append(interp.logs, result.Logs...)
		}

	case CREATE:
		// Pop 3 args: value, offset, length
		valueArg, ok1 := pop()
		offsetArg, ok2 := pop()
		lengthArg, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}

		off := int(offsetArg.Uint64())
		ln := int(lengthArg.Uint64())
		if ln > 0 {
			if gasCost, err := interp.memory.Expand(off + ln); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
		}

		// Read init code from memory
		initCode := interp.memory.Read(off, ln)

		// Derive address: keccak256(sender || nonce)
		var nonce uint64
		if nb, ok := state.(NonceDB); ok {
			nonce = nb.GetNonce(ctx.CurrentContract)
			nb.SetNonce(ctx.CurrentContract, nonce+1)
		}
		newAddr := CreateAddress(ctx.CurrentContract, nonce)

		// Gas forwarding for init code execution
		gasForward := CalculateForwardGas(interp.gas.Remaining(), interp.gas.Remaining())
		if err := interp.gas.Consume(gasForward); err != nil {
			return actionError
		}

		// Execute init code in snapshot
		snapshot := NewSnapshot(state)
		result := ExecuteCall(ModeCreate, ctx, newAddr, initCode, gasForward, valueArg, snapshot, host, interp.callDepth)

		if result.GasUsed < gasForward {
			interp.gas.Refund(gasForward - result.GasUsed)
		}

		if result.Success && len(result.ReturnData) > 0 {
			// Enforce EIP-170: deployed code max 24KB
			if len(result.ReturnData) > MaxBytecodeSize {
				push(new(big.Int)) // 0 = failure
			} else {
				// Store deployed code
				snapshot.SetCode(newAddr, result.ReturnData)
				snapshot.Commit()
				push(new(big.Int).SetBytes(newAddr))
			}
		} else if result.Success {
			// Empty code deployment (valid)
			snapshot.Commit()
			push(new(big.Int).SetBytes(newAddr))
		} else {
			// CREATE failed — snapshot discarded
			push(new(big.Int)) // 0 = failure
		}

		interp.returnData = result.ReturnData
		if result.Success && result.Logs != nil {
			interp.logs = append(interp.logs, result.Logs...)
		}

	case CREATE2:
		// Pop 4 args: value, offset, length, salt
		valueArg, ok1 := pop()
		offsetArg, ok2 := pop()
		lengthArg, ok3 := pop()
		saltArg, ok4 := pop()
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return actionError
		}

		off := int(offsetArg.Uint64())
		ln := int(lengthArg.Uint64())
		if ln > 0 {
			if gasCost, err := interp.memory.Expand(off + ln); err != nil {
				return actionError
			} else if gasCost > 0 {
				if err := interp.gas.Consume(gasCost); err != nil {
					return actionError
				}
			}
		}

		initCode := interp.memory.Read(off, ln)

		// Derive address: keccak256(0xff || sender || salt || keccak256(initCode))
		var saltBytes [32]byte
		saltB32 := WordToBytes32(saltArg)
		copy(saltBytes[:], saltB32[:])
		newAddr := Create2Address(ctx.CurrentContract, saltBytes, initCode)

		// Gas: charge for keccak256 of init code
		wordSize := uint64((ln + 31) / 32)
		if err := interp.gas.Consume(GasSHA3PerWord * wordSize); err != nil {
			return actionError
		}

		gasForward := CalculateForwardGas(interp.gas.Remaining(), interp.gas.Remaining())
		if err := interp.gas.Consume(gasForward); err != nil {
			return actionError
		}

		snapshot := NewSnapshot(state)
		result := ExecuteCall(ModeCreate, ctx, newAddr, initCode, gasForward, valueArg, snapshot, host, interp.callDepth)

		if result.GasUsed < gasForward {
			interp.gas.Refund(gasForward - result.GasUsed)
		}

		if result.Success && len(result.ReturnData) > 0 {
			if len(result.ReturnData) > MaxBytecodeSize {
				push(new(big.Int))
			} else {
				snapshot.SetCode(newAddr, result.ReturnData)
				snapshot.Commit()
				push(new(big.Int).SetBytes(newAddr))
			}
		} else if result.Success {
			snapshot.Commit()
			push(new(big.Int).SetBytes(newAddr))
		} else {
			push(new(big.Int))
		}

		interp.returnData = result.ReturnData
		if result.Success && result.Logs != nil {
			interp.logs = append(interp.logs, result.Logs...)
		}

	default:
		// PUSH1-PUSH32
		if opcode >= PUSH1 && opcode <= PUSH32 {
			n := int(opcode - PUSH1 + 1)
			data := make([]byte, n)
			for i := 0; i < n; i++ {
				idx := interp.pc + 1 + i
				if idx < len(bytecode) {
					data[i] = bytecode[idx]
				}
			}
			push(new(big.Int).SetBytes(data))
			interp.pc += n // Advance past pushed bytes (loop will add 1 more)
			return actionContinue
		}

		// DUP1-DUP16
		if opcode >= DUP1 && opcode <= DUP16 {
			depth := int(opcode-DUP1) + 1
			if err := interp.stack.Dup(depth); err != nil {
				return actionError
			}
			return actionContinue
		}

		// SWAP1-SWAP16
		if opcode >= SWAP1 && opcode <= SWAP16 {
			depth := int(opcode-SWAP1) + 1
			if err := interp.stack.Swap(depth); err != nil {
				return actionError
			}
			return actionContinue
		}

		// Agent opcodes — delegate to host functions
		if IsAgentOpcode(opcode) {
			return interp.executeAgentOpcode(opcode, ctx, state, host)
		}

		return actionError
	}

	return actionContinue
}

// checkCapability returns true if the execution context permits the given capability.
// C-1 FIX: Secure default — nil capabilities DENY all restricted operations.
// Only explicit SessionCapabilities grant access to agent opcodes.
// Callers authenticated via identity/operational key (not session) should have
// their capabilities set to all-true by the keeper before execution.
func checkCapability(ctx *ExecutionContext, cap string) bool {
	if ctx.Capabilities == nil {
		return false // SECURE DEFAULT: deny without explicit session
	}
	switch cap {
	case "stake":
		return ctx.Capabilities.CanStake
	case "transfer":
		return ctx.Capabilities.CanTransfer
	case "submit_claims":
		return ctx.Capabilities.CanSubmitClaims
	case "vote":
		return ctx.Capabilities.CanVote
	default:
		return false // Unknown capability: deny
	}
}

// executeAgentOpcode handles all Legible Money extension opcodes.
// H-1: Reentrancy guard — rejects agent opcodes if already inside a host call.
func (interp *Interpreter) executeAgentOpcode(
	opcode byte,
	ctx *ExecutionContext,
	state StateDB,
	host HostFunctions,
) opcodeAction {
	pop := func() (*big.Int, bool) {
		v, err := interp.stack.Pop()
		if err != nil {
			return nil, false
		}
		return v, true
	}
	push := func(v *big.Int) {
		interp.stack.Push(v)
	}

	// H-1: Reentrancy guard — reject if already inside a host function call
	if ctx.InHostCall {
		return actionError
	}

	// If no host functions, agent opcodes silently push 0 / no-op
	if host == nil {
		info := OpcodeTable[opcode]
		if info != nil {
			// Pop inputs
			for i := 0; i < info.StackInputs; i++ {
				if _, ok := pop(); !ok {
					return actionError
				}
			}
			// Push outputs (all zeros)
			for i := 0; i < info.StackOutputs; i++ {
				push(new(big.Int))
			}
		}
		return actionContinue
	}

	// H-1: Set reentrancy guard before host function dispatch, clear after.
	ctx.InHostCall = true
	defer func() { ctx.InHostCall = false }()

	switch opcode {
	// Knowledge (0xB0-0xB5)
	case QUERY_FACT:
		factId, ok := pop()
		if !ok {
			return actionError
		}
		b := WordToBytes32(factId)
		exists, confidence, domain := host.QueryFact(b[:])
		if exists {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}
		push(new(big.Int).SetUint64(confidence))
		push(new(big.Int).SetBytes(domain))

	case QUERY_DOMAIN:
		domain, ok1 := pop()
		limit, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		b := WordToBytes32(domain)
		count, _ := host.QueryDomain(b[:], limit.Uint64())
		push(new(big.Int).SetUint64(count))

	case SUBMIT_CLAIM:
		if !checkCapability(ctx, "submit_claims") {
			// Pop 3 inputs, push 0 (denied)
			for i := 0; i < 3; i++ {
				if _, ok := pop(); !ok {
					return actionError
				}
			}
			push(new(big.Int))
			return actionContinue
		}
		domain, ok1 := pop()
		dataHash, ok2 := pop()
		evidenceHash, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		bd := WordToBytes32(domain)
		bData := WordToBytes32(dataHash)
		bEvidence := WordToBytes32(evidenceHash)
		claimId := host.SubmitClaim(ctx.CallerDID, bd[:], bData[:], bEvidence[:], ctx.BlockNumber)
		if claimId != nil {
			push(new(big.Int).SetBytes(claimId))
		} else {
			push(new(big.Int))
		}

	case COMMIT_VOTE:
		if !checkCapability(ctx, "vote") {
			for i := 0; i < 2; i++ {
				if _, ok := pop(); !ok {
					return actionError
				}
			}
			push(new(big.Int))
			return actionContinue
		}
		claimId, ok1 := pop()
		voteHash, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		bc := WordToBytes32(claimId)
		bv := WordToBytes32(voteHash)
		if host.CommitVote(ctx.CallerDID, bc[:], bv[:], ctx.BlockNumber) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case REVEAL_VOTE:
		if !checkCapability(ctx, "vote") {
			for i := 0; i < 3; i++ {
				if _, ok := pop(); !ok {
					return actionError
				}
			}
			push(new(big.Int))
			return actionContinue
		}
		claimId, ok1 := pop()
		vote, ok2 := pop()
		salt, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		bc := WordToBytes32(claimId)
		if host.RevealVote(ctx.CallerDID, bc[:], vote, salt, ctx.BlockNumber) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case INITIATE_CHALLENGE:
		factId, ok1 := pop()
		evidenceHash, ok2 := pop()
		stakeAmt, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		bf := WordToBytes32(factId)
		be := WordToBytes32(evidenceHash)
		challengeId := host.InitiateChallenge(ctx.CallerDID, bf[:], be[:], stakeAmt, ctx.BlockNumber)
		if challengeId != nil {
			push(new(big.Int).SetBytes(challengeId))
		} else {
			push(new(big.Int))
		}

	// Agent Identity (0xE0-0xE3)
	case AGENT_DID:
		did := host.AgentDID(ctx.Caller)
		if did != nil {
			push(new(big.Int).SetBytes(did))
		} else {
			push(new(big.Int))
		}

	case AGENT_HOME:
		home := host.AgentHome(ctx.CallerDID)
		if home != nil {
			push(new(big.Int).SetBytes(home))
		} else {
			push(new(big.Int))
		}

	case VERIFY_IDENTITY:
		targetDID, ok := pop()
		if !ok {
			return actionError
		}
		b := WordToBytes32(targetDID)
		if host.VerifyIdentity(b[:]) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case ATTEST_IDENTITY:
		targetDID, ok1 := pop()
		attestType, ok2 := pop()
		data, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		bt := WordToBytes32(targetDID)
		ba := WordToBytes32(attestType)
		bd := WordToBytes32(data)
		attestId := host.AttestIdentity(ctx.CallerDID, bt[:], ba[:], bd[:])
		if attestId != nil {
			push(new(big.Int).SetBytes(attestId))
		} else {
			push(new(big.Int))
		}

	// Agent Belief (0xE4-0xE7)
	case GET_BELIEF:
		factId, ok := pop()
		if !ok {
			return actionError
		}
		b := WordToBytes32(factId)
		confidence := host.GetBelief(ctx.CallerDID, b[:])
		push(new(big.Int).SetUint64(confidence))

	case SET_BELIEF:
		factId, ok1 := pop()
		confidence, ok2 := pop()
		source, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		bf := WordToBytes32(factId)
		bs := WordToBytes32(source)
		host.SetBelief(ctx.CallerDID, bf[:], confidence.Uint64(), bs[:])

	case COMPARE_BELIEF:
		factId, ok := pop()
		if !ok {
			return actionError
		}
		// FARM-4: During commit phase, COMPARE_BELIEF returns 0 to prevent
		// confidence herding — agents cannot peek at network consensus before voting.
		if ctx.IsCommitPhase {
			push(FromSigned(big.NewInt(0)))
		} else {
			b := WordToBytes32(factId)
			diff := host.CompareBelief(ctx.CallerDID, b[:])
			push(FromSigned(big.NewInt(diff)))
		}

	case BELIEF_COUNT:
		domain, ok := pop()
		if !ok {
			return actionError
		}
		b := WordToBytes32(domain)
		count := host.BeliefCount(ctx.CallerDID, b[:])
		push(new(big.Int).SetUint64(count))

	// Agent Relationship (0xE8-0xEB)
	case GET_TRUST:
		targetDID, ok := pop()
		if !ok {
			return actionError
		}
		b := WordToBytes32(targetDID)
		level := host.GetTrust(ctx.CallerDID, b[:])
		push(new(big.Int).SetUint64(level))

	case SET_TRUST:
		targetDID, ok1 := pop()
		level, ok2 := pop()
		reason, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		bt := WordToBytes32(targetDID)
		br := WordToBytes32(reason)
		host.SetTrust(ctx.CallerDID, bt[:], level.Uint64(), br[:])

	case IS_TRUSTED:
		targetDID, ok1 := pop()
		threshold, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		bt := WordToBytes32(targetDID)
		if host.IsTrusted(ctx.CallerDID, bt[:], threshold.Uint64()) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case DELEGATE:
		targetDID, ok1 := pop()
		capability, ok2 := pop()
		constraints, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		bt := WordToBytes32(targetDID)
		bc := WordToBytes32(capability)
		bcon := WordToBytes32(constraints)
		delegationId := host.Delegate(ctx.CallerDID, bt[:], bc[:], bcon[:])
		if delegationId != nil {
			push(new(big.Int).SetBytes(delegationId))
		} else {
			push(new(big.Int))
		}

	// Agent Autonomy (0xEC-0xEF)
	case SCHEDULE_SELF:
		callData, ok1 := pop()
		executeBlock, ok2 := pop()
		gasLimit, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		bd := WordToBytes32(callData)
		scheduleId := host.ScheduleSelf(ctx.CurrentContract, bd[:], executeBlock.Uint64(), gasLimit.Uint64())
		if scheduleId != nil {
			push(new(big.Int).SetBytes(scheduleId))
		} else {
			push(new(big.Int))
		}

	case CANCEL_SCHEDULE:
		scheduleId, ok := pop()
		if !ok {
			return actionError
		}
		bs := WordToBytes32(scheduleId)
		if host.CancelScheduleHost(bs[:]) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case MODIFY_SCHEDULE:
		scheduleId, ok1 := pop()
		newBlock, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		bs := WordToBytes32(scheduleId)
		if host.ModifySchedule(bs[:], newBlock.Uint64()) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case GET_SCHEDULES:
		count := host.GetScheduleCount(ctx.CurrentContract)
		push(new(big.Int).SetUint64(count))

	// Extended Relationships (0xC0-0xC3)
	case LIST_TRUSTED:
		offset, ok1 := pop()
		limit, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		count, _ := host.ListTrusted(ctx.CallerDID, offset.Uint64(), limit.Uint64())
		push(new(big.Int).SetUint64(count))

	case BLOCK_AGENT:
		targetDID, ok1 := pop()
		reason, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		bt := WordToBytes32(targetDID)
		br := WordToBytes32(reason)
		if host.BlockAgent(ctx.CallerDID, bt[:], br[:]) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case UNBLOCK_AGENT:
		targetDID, ok := pop()
		if !ok {
			return actionError
		}
		bt := WordToBytes32(targetDID)
		if host.UnblockAgent(ctx.CallerDID, bt[:]) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case REVOKE_DELEGATION:
		delegationId, ok := pop()
		if !ok {
			return actionError
		}
		bd := WordToBytes32(delegationId)
		if host.RevokeDelegation(ctx.CallerDID, bd[:]) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	// Treasury (0xD0-0xD7)
	case CLAIM_VESTING:
		if !checkCapability(ctx, "transfer") {
			if _, ok := pop(); !ok {
				return actionError
			}
			push(new(big.Int))
			return actionContinue
		}
		vestingId, ok := pop()
		if !ok {
			return actionError
		}
		bv := WordToBytes32(vestingId)
		amount := host.ClaimVesting(ctx.CallerDID, bv[:])
		if amount != nil {
			push(amount)
		} else {
			push(new(big.Int))
		}

	case QUERY_VESTING:
		offset, ok1 := pop()
		limit, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		count, _ := host.QueryVesting(ctx.CallerDID, offset.Uint64(), limit.Uint64())
		push(new(big.Int).SetUint64(count))

	case STAKE:
		if !checkCapability(ctx, "stake") {
			for i := 0; i < 2; i++ {
				if _, ok := pop(); !ok {
					return actionError
				}
			}
			push(new(big.Int))
			return actionContinue
		}
		amount, ok1 := pop()
		validatorDID, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		bv := WordToBytes32(validatorDID)
		stakeId := host.StakeTokens(ctx.CallerDID, amount, bv[:])
		if stakeId != nil {
			push(new(big.Int).SetBytes(stakeId))
		} else {
			push(new(big.Int))
		}

	case UNSTAKE:
		if !checkCapability(ctx, "stake") {
			if _, ok := pop(); !ok {
				return actionError
			}
			push(new(big.Int))
			return actionContinue
		}
		stakeId, ok := pop()
		if !ok {
			return actionError
		}
		bs := WordToBytes32(stakeId)
		amount := host.UnstakeTokens(ctx.CallerDID, bs[:])
		if amount != nil {
			push(amount)
		} else {
			push(new(big.Int))
		}

	case QUERY_EXPIRING:
		periodId, ok := pop()
		if !ok {
			return actionError
		}
		bp := WordToBytes32(periodId)
		amount, nextExpiry := host.QueryExpiring(ctx.CallerDID, bp[:])
		if amount != nil {
			push(amount)
		} else {
			push(new(big.Int))
		}
		push(new(big.Int).SetUint64(nextExpiry))

	case EXTEND_EXPIRING:
		if !checkCapability(ctx, "transfer") {
			for i := 0; i < 2; i++ {
				if _, ok := pop(); !ok {
					return actionError
				}
			}
			push(new(big.Int))
			return actionContinue
		}
		periodId, ok1 := pop()
		newExpiry, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		bp := WordToBytes32(periodId)
		if host.ExtendExpiring(ctx.CallerDID, bp[:], newExpiry.Uint64()) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case SET_SPEND_LIMIT:
		if !checkCapability(ctx, "transfer") {
			for i := 0; i < 3; i++ {
				if _, ok := pop(); !ok {
					return actionError
				}
			}
			push(new(big.Int))
			return actionContinue
		}
		limitType, ok1 := pop()
		amount, ok2 := pop()
		period, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		bl := WordToBytes32(limitType)
		ba := WordToBytes32(amount)
		if host.SetSpendLimit(ctx.CallerDID, bl[:], ba[:], period.Uint64()) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case QUERY_TREASURY:
		available, reserved, staked, delegated := host.QueryTreasury(ctx.CallerDID)
		pushOrZero := func(v *big.Int) {
			if v != nil {
				push(v)
			} else {
				push(new(big.Int))
			}
		}
		pushOrZero(available)
		pushOrZero(reserved)
		pushOrZero(staked)
		pushOrZero(delegated)

	// Guardian (0xD8-0xDB)
	case QUERY_ALERTS:
		offset, ok1 := pop()
		limit, ok2 := pop()
		if !ok1 || !ok2 {
			return actionError
		}
		count, _ := host.QueryAlerts(ctx.CallerDID, offset.Uint64(), limit.Uint64())
		push(new(big.Int).SetUint64(count))

	case DISMISS_ALERT:
		alertId, ok := pop()
		if !ok {
			return actionError
		}
		ba := WordToBytes32(alertId)
		if host.DismissAlert(ctx.CallerDID, ba[:]) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case SET_DEFENSE:
		defenseType, ok1 := pop()
		config, ok2 := pop()
		threshold, ok3 := pop()
		if !ok1 || !ok2 || !ok3 {
			return actionError
		}
		bd := WordToBytes32(defenseType)
		bc := WordToBytes32(config)
		bt := WordToBytes32(threshold)
		if host.SetDefense(ctx.CallerDID, bd[:], bc[:], bt[:]) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case QUERY_DEFENSES:
		count, _ := host.QueryDefenses(ctx.CallerDID)
		push(new(big.Int).SetUint64(count))

	// Access Control (0xDC-0xDF)
	case REGISTER_KEY:
		keyType, ok1 := pop()
		pubKey, ok2 := pop()
		perms, ok3 := pop()
		expiry, ok4 := pop()
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return actionError
		}
		bk := WordToBytes32(keyType)
		bp := WordToBytes32(pubKey)
		bperms := WordToBytes32(perms)
		be := WordToBytes32(expiry)
		if host.RegisterKey(ctx.CallerDID, bk[:], bp[:], bperms[:], be[:]) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case REVOKE_KEY:
		keyId, ok := pop()
		if !ok {
			return actionError
		}
		bk := WordToBytes32(keyId)
		if host.RevokeKey(ctx.CallerDID, bk[:]) {
			push(big.NewInt(1))
		} else {
			push(new(big.Int))
		}

	case QUERY_PERMISSIONS:
		role, ok := pop()
		if !ok {
			return actionError
		}
		br := WordToBytes32(role)
		perms := host.QueryPermissions(ctx.CallerDID, br[:])
		if perms != nil {
			push(perms)
		} else {
			push(new(big.Int))
		}

	case QUERY_SESSION:
		sessionId, role, expiresAt, actionsRemaining := host.QuerySession(ctx.CallerDID)
		if sessionId != nil {
			push(new(big.Int).SetBytes(sessionId))
		} else {
			push(new(big.Int))
		}
		if role != nil {
			push(new(big.Int).SetBytes(role))
		} else {
			push(new(big.Int))
		}
		push(new(big.Int).SetUint64(expiresAt))
		push(new(big.Int).SetUint64(actionsRemaining))

	// Introspection (0xF6-0xF9)
	case SELF_BALANCE_OP:
		bal := host.SelfBalance(ctx.CallerDID)
		if bal != nil {
			push(bal)
		} else {
			push(new(big.Int))
		}

	case SELF_STAKE:
		stake := host.SelfStake(ctx.CallerDID)
		if stake != nil {
			push(stake)
		} else {
			push(new(big.Int))
		}

	case SELF_REPUTATION:
		rep := host.SelfReputation(ctx.CallerDID)
		push(new(big.Int).SetUint64(rep))

	case SELF_ACCURACY:
		acc := host.SelfAccuracy(ctx.CallerDID)
		push(new(big.Int).SetUint64(acc))

	default:
		return actionError
	}

	return actionContinue
}

func (interp *Interpreter) successResult() *ExecutionResult {
	return &ExecutionResult{
		Success:      true,
		ReturnData:   interp.returnData,
		GasUsed:      interp.gas.Used(),
		Logs:         interp.logs,
		StateChanges: interp.changes,
	}
}

func (interp *Interpreter) revertResult() *ExecutionResult {
	return &ExecutionResult{
		Success:    false,
		ReturnData: interp.returnData,
		GasUsed:    interp.gas.Used(),
		Logs:       nil,
		Error:      &ExecutionError{Code: ErrCodeRevert, Message: "execution reverted"},
	}
}

func (interp *Interpreter) errorResult(code ErrorCode, msg string) *ExecutionResult {
	return &ExecutionResult{
		Success:    false,
		ReturnData: nil,
		GasUsed:    interp.gas.Used(),
		Logs:       nil,
		Error:      &ExecutionError{Code: code, Message: msg},
	}
}

func scanJumpDests(bytecode []byte) map[int]bool {
	dests := make(map[int]bool)
	for i := 0; i < len(bytecode); {
		op := bytecode[i]
		if op == JUMPDEST {
			dests[i] = true
		}
		// Skip PUSH data bytes
		if op >= PUSH1 && op <= PUSH32 {
			i += int(op-PUSH1) + 2
		} else {
			i++
		}
	}
	return dests
}

func hexByte(b byte) string {
	const hex = "0123456789abcdef"
	return string([]byte{hex[b>>4], hex[b&0x0f]})
}
