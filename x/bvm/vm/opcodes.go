package vm

// Opcode byte constants for the BVM (EVM-compatible + Legible Money extensions).

// Stop and Arithmetic (0x00-0x0F)
const (
	STOP       byte = 0x00
	ADD        byte = 0x01
	MUL        byte = 0x02
	SUB        byte = 0x03
	DIV        byte = 0x04
	SDIV       byte = 0x05
	MOD        byte = 0x06
	SMOD       byte = 0x07
	ADDMOD     byte = 0x08
	MULMOD     byte = 0x09
	EXP        byte = 0x0A
	SIGNEXTEND byte = 0x0B
)

// Comparison and Bitwise (0x10-0x1D)
const (
	LT     byte = 0x10
	GT     byte = 0x11
	SLT    byte = 0x12
	SGT    byte = 0x13
	EQ     byte = 0x14
	ISZERO byte = 0x15
	AND    byte = 0x16
	OR     byte = 0x17
	XOR    byte = 0x18
	NOT    byte = 0x19
	BYTE   byte = 0x1A
	SHL    byte = 0x1B
	SHR    byte = 0x1C
	SAR    byte = 0x1D
)

// SHA3
const SHA3 byte = 0x20

// Environmental (0x30-0x3F)
const (
	ADDRESS        byte = 0x30
	BALANCE        byte = 0x31
	ORIGIN         byte = 0x32
	CALLER         byte = 0x33
	CALLVALUE      byte = 0x34
	CALLDATALOAD   byte = 0x35
	CALLDATASIZE   byte = 0x36
	CALLDATACOPY   byte = 0x37
	CODESIZE       byte = 0x38
	CODECOPY       byte = 0x39
	GASPRICE       byte = 0x3A
	EXTCODESIZE    byte = 0x3B
	EXTCODECOPY    byte = 0x3C
	RETURNDATASIZE byte = 0x3D
	RETURNDATACOPY byte = 0x3E
	EXTCODEHASH    byte = 0x3F
)

// Block Information (0x40-0x48)
const (
	BLOCKHASH   byte = 0x40
	COINBASE    byte = 0x41
	TIMESTAMP   byte = 0x42
	NUMBER      byte = 0x43
	PREVRANDAO  byte = 0x44
	GASLIMIT    byte = 0x45
	CHAINID     byte = 0x46
	SELFBALANCE byte = 0x47
	BASEFEE     byte = 0x48
)

// Stack, Memory, Storage (0x50-0x5F)
const (
	POP      byte = 0x50
	MLOAD    byte = 0x51
	MSTORE   byte = 0x52
	MSTORE8  byte = 0x53
	SLOAD    byte = 0x54
	SSTORE   byte = 0x55
	JUMP     byte = 0x56
	JUMPI    byte = 0x57
	PC       byte = 0x58
	MSIZE    byte = 0x59
	GAS      byte = 0x5A
	JUMPDEST byte = 0x5B
	PUSH0    byte = 0x5F
)

// Push (0x60-0x7F)
const (
	PUSH1  byte = 0x60
	PUSH2  byte = 0x61
	PUSH3  byte = 0x62
	PUSH4  byte = 0x63
	PUSH5  byte = 0x64
	PUSH6  byte = 0x65
	PUSH7  byte = 0x66
	PUSH8  byte = 0x67
	PUSH9  byte = 0x68
	PUSH10 byte = 0x69
	PUSH11 byte = 0x6A
	PUSH12 byte = 0x6B
	PUSH13 byte = 0x6C
	PUSH14 byte = 0x6D
	PUSH15 byte = 0x6E
	PUSH16 byte = 0x6F
	PUSH17 byte = 0x70
	PUSH18 byte = 0x71
	PUSH19 byte = 0x72
	PUSH20 byte = 0x73
	PUSH21 byte = 0x74
	PUSH22 byte = 0x75
	PUSH23 byte = 0x76
	PUSH24 byte = 0x77
	PUSH25 byte = 0x78
	PUSH26 byte = 0x79
	PUSH27 byte = 0x7A
	PUSH28 byte = 0x7B
	PUSH29 byte = 0x7C
	PUSH30 byte = 0x7D
	PUSH31 byte = 0x7E
	PUSH32 byte = 0x7F
)

// Dup (0x80-0x8F)
const (
	DUP1  byte = 0x80
	DUP2  byte = 0x81
	DUP3  byte = 0x82
	DUP4  byte = 0x83
	DUP5  byte = 0x84
	DUP6  byte = 0x85
	DUP7  byte = 0x86
	DUP8  byte = 0x87
	DUP9  byte = 0x88
	DUP10 byte = 0x89
	DUP11 byte = 0x8A
	DUP12 byte = 0x8B
	DUP13 byte = 0x8C
	DUP14 byte = 0x8D
	DUP15 byte = 0x8E
	DUP16 byte = 0x8F
)

// Swap (0x90-0x9F)
const (
	SWAP1  byte = 0x90
	SWAP2  byte = 0x91
	SWAP3  byte = 0x92
	SWAP4  byte = 0x93
	SWAP5  byte = 0x94
	SWAP6  byte = 0x95
	SWAP7  byte = 0x96
	SWAP8  byte = 0x97
	SWAP9  byte = 0x98
	SWAP10 byte = 0x99
	SWAP11 byte = 0x9A
	SWAP12 byte = 0x9B
	SWAP13 byte = 0x9C
	SWAP14 byte = 0x9D
	SWAP15 byte = 0x9E
	SWAP16 byte = 0x9F
)

// Log (0xA0-0xA4)
const (
	LOG0 byte = 0xA0
	LOG1 byte = 0xA1
	LOG2 byte = 0xA2
	LOG3 byte = 0xA3
	LOG4 byte = 0xA4
)

// Knowledge Operations (0xB0-0xB5) — Legible Money PoT Integration
const (
	QUERY_FACT         byte = 0xB0
	QUERY_DOMAIN       byte = 0xB1
	SUBMIT_CLAIM       byte = 0xB2
	COMMIT_VOTE        byte = 0xB3
	REVEAL_VOTE        byte = 0xB4
	INITIATE_CHALLENGE byte = 0xB5
)

// Extended Relationships (0xC0-0xC3)
const (
	LIST_TRUSTED      byte = 0xC0
	BLOCK_AGENT       byte = 0xC1
	UNBLOCK_AGENT     byte = 0xC2
	REVOKE_DELEGATION byte = 0xC3
)

// Treasury (0xD0-0xD7)
const (
	CLAIM_VESTING   byte = 0xD0
	QUERY_VESTING   byte = 0xD1
	STAKE           byte = 0xD2
	UNSTAKE         byte = 0xD3
	QUERY_EXPIRING  byte = 0xD4
	EXTEND_EXPIRING byte = 0xD5
	SET_SPEND_LIMIT byte = 0xD6
	QUERY_TREASURY  byte = 0xD7
)

// Guardian (0xD8-0xDB)
const (
	QUERY_ALERTS   byte = 0xD8
	DISMISS_ALERT  byte = 0xD9
	SET_DEFENSE    byte = 0xDA
	QUERY_DEFENSES byte = 0xDB
)

// Access Control (0xDC-0xDF)
const (
	REGISTER_KEY      byte = 0xDC
	REVOKE_KEY        byte = 0xDD
	QUERY_PERMISSIONS byte = 0xDE
	QUERY_SESSION     byte = 0xDF
)

// Agent Identity (0xE0-0xE3)
const (
	AGENT_DID       byte = 0xE0
	AGENT_HOME      byte = 0xE1
	VERIFY_IDENTITY byte = 0xE2
	ATTEST_IDENTITY byte = 0xE3
)

// Agent Belief (0xE4-0xE7)
const (
	GET_BELIEF     byte = 0xE4
	SET_BELIEF     byte = 0xE5
	COMPARE_BELIEF byte = 0xE6
	BELIEF_COUNT   byte = 0xE7
)

// Agent Relationship (0xE8-0xEB)
const (
	GET_TRUST  byte = 0xE8
	SET_TRUST  byte = 0xE9
	IS_TRUSTED byte = 0xEA
	DELEGATE   byte = 0xEB
)

// Agent Autonomy (0xEC-0xEF)
const (
	SCHEDULE_SELF   byte = 0xEC
	CANCEL_SCHEDULE byte = 0xED
	MODIFY_SCHEDULE byte = 0xEE
	GET_SCHEDULES   byte = 0xEF
)

// System Operations (0xF0-0xFF)
const (
	CREATE       byte = 0xF0
	CALL         byte = 0xF1
	CALLCODE     byte = 0xF2
	RETURN       byte = 0xF3
	DELEGATECALL byte = 0xF4
	CREATE2      byte = 0xF5
	STATICCALL   byte = 0xFA
	REVERT       byte = 0xFD
	INVALID      byte = 0xFE
	SELFDESTRUCT byte = 0xFF
)

// Introspection (0xF6-0xF9)
const (
	SELF_BALANCE_OP byte = 0xF6
	SELF_STAKE      byte = 0xF7
	SELF_REPUTATION byte = 0xF8
	SELF_ACCURACY   byte = 0xF9
)

// OpcodeInfo holds metadata for a single opcode.
type OpcodeInfo struct {
	Code              byte
	Mnemonic          string
	StackInputs       int
	StackOutputs      int
	GasCost           uint64 // Base gas cost (0 = dynamic)
	IsStateModifier   bool
	MinVersion        uint32 // Minimum BVM version required (0 = v1, available from genesis)
	ActivationBlock   uint64 // Block height at which opcode becomes available (0 = always active)
	DeprecatedAtBlock uint64 // Block height at which opcode becomes a no-op (0 = not deprecated)
}

// OpcodeTable provides O(1) lookup by opcode byte.
var OpcodeTable [256]*OpcodeInfo

func init() {
	// Arithmetic
	reg(STOP, "STOP", 0, 0, 0, false)
	reg(ADD, "ADD", 2, 1, 3, false)
	reg(MUL, "MUL", 2, 1, 5, false)
	reg(SUB, "SUB", 2, 1, 3, false)
	reg(DIV, "DIV", 2, 1, 5, false)
	reg(SDIV, "SDIV", 2, 1, 5, false)
	reg(MOD, "MOD", 2, 1, 5, false)
	reg(SMOD, "SMOD", 2, 1, 5, false)
	reg(ADDMOD, "ADDMOD", 3, 1, 8, false)
	reg(MULMOD, "MULMOD", 3, 1, 8, false)
	reg(EXP, "EXP", 2, 1, 10, false) // +50/byte dynamic
	reg(SIGNEXTEND, "SIGNEXTEND", 2, 1, 5, false)

	// Comparison & Bitwise
	reg(LT, "LT", 2, 1, 3, false)
	reg(GT, "GT", 2, 1, 3, false)
	reg(SLT, "SLT", 2, 1, 3, false)
	reg(SGT, "SGT", 2, 1, 3, false)
	reg(EQ, "EQ", 2, 1, 3, false)
	reg(ISZERO, "ISZERO", 1, 1, 3, false)
	reg(AND, "AND", 2, 1, 3, false)
	reg(OR, "OR", 2, 1, 3, false)
	reg(XOR, "XOR", 2, 1, 3, false)
	reg(NOT, "NOT", 1, 1, 3, false)
	reg(BYTE, "BYTE", 2, 1, 3, false)
	reg(SHL, "SHL", 2, 1, 3, false)
	reg(SHR, "SHR", 2, 1, 3, false)
	reg(SAR, "SAR", 2, 1, 3, false)

	// SHA3
	reg(SHA3, "SHA3", 2, 1, 30, false) // +6/word dynamic

	// Environmental
	reg(ADDRESS, "ADDRESS", 0, 1, 2, false)
	reg(BALANCE, "BALANCE", 1, 1, 400, false)
	reg(ORIGIN, "ORIGIN", 0, 1, 2, false)
	reg(CALLER, "CALLER", 0, 1, 2, false)
	reg(CALLVALUE, "CALLVALUE", 0, 1, 2, false)
	reg(CALLDATALOAD, "CALLDATALOAD", 1, 1, 3, false)
	reg(CALLDATASIZE, "CALLDATASIZE", 0, 1, 2, false)
	reg(CALLDATACOPY, "CALLDATACOPY", 3, 0, 3, false) // +3/word dynamic
	reg(CODESIZE, "CODESIZE", 0, 1, 2, false)
	reg(CODECOPY, "CODECOPY", 3, 0, 3, false)
	reg(GASPRICE, "GASPRICE", 0, 1, 2, false)
	reg(EXTCODESIZE, "EXTCODESIZE", 1, 1, 700, false)
	reg(EXTCODECOPY, "EXTCODECOPY", 4, 0, 700, false)
	reg(RETURNDATASIZE, "RETURNDATASIZE", 0, 1, 2, false)
	reg(RETURNDATACOPY, "RETURNDATACOPY", 3, 0, 3, false)
	reg(EXTCODEHASH, "EXTCODEHASH", 1, 1, 400, false)

	// Block Info
	reg(BLOCKHASH, "BLOCKHASH", 1, 1, 20, false)
	reg(COINBASE, "COINBASE", 0, 1, 2, false)
	reg(TIMESTAMP, "TIMESTAMP", 0, 1, 2, false)
	reg(NUMBER, "NUMBER", 0, 1, 2, false)
	reg(PREVRANDAO, "PREVRANDAO", 0, 1, 2, false)
	reg(GASLIMIT, "GASLIMIT", 0, 1, 2, false)
	reg(CHAINID, "CHAINID", 0, 1, 2, false)
	reg(SELFBALANCE, "SELFBALANCE", 0, 1, 5, false)
	reg(BASEFEE, "BASEFEE", 0, 1, 2, false)

	// Stack, Memory, Storage
	reg(POP, "POP", 1, 0, 2, false)
	reg(MLOAD, "MLOAD", 1, 1, 3, false)
	reg(MSTORE, "MSTORE", 2, 0, 3, false)
	reg(MSTORE8, "MSTORE8", 2, 0, 3, false)
	reg(SLOAD, "SLOAD", 1, 1, 200, false)
	reg(SSTORE, "SSTORE", 2, 0, 0, true) // Dynamic gas
	reg(JUMP, "JUMP", 1, 0, 8, false)
	reg(JUMPI, "JUMPI", 2, 0, 10, false)
	reg(PC, "PC", 0, 1, 2, false)
	reg(MSIZE, "MSIZE", 0, 1, 2, false)
	reg(GAS, "GAS", 0, 1, 2, false)
	reg(JUMPDEST, "JUMPDEST", 0, 0, 1, false)
	reg(PUSH0, "PUSH0", 0, 1, 2, false)

	// PUSH1-PUSH32
	for i := byte(0); i < 32; i++ {
		op := PUSH1 + i
		mnemonic := "PUSH" + itoa(int(i+1))
		reg(op, mnemonic, 0, 1, 3, false)
	}

	// DUP1-DUP16
	for i := byte(0); i < 16; i++ {
		op := DUP1 + i
		mnemonic := "DUP" + itoa(int(i+1))
		reg(op, mnemonic, int(i+1), int(i+2), 3, false)
	}

	// SWAP1-SWAP16
	for i := byte(0); i < 16; i++ {
		op := SWAP1 + i
		mnemonic := "SWAP" + itoa(int(i+1))
		reg(op, mnemonic, int(i+2), int(i+2), 3, false)
	}

	// LOG0-LOG4
	for i := byte(0); i <= 4; i++ {
		op := LOG0 + i
		mnemonic := "LOG" + itoa(int(i))
		reg(op, mnemonic, int(2+i), 0, 375+375*uint64(i), true)
	}

	// Knowledge
	reg(QUERY_FACT, "QUERY_FACT", 1, 3, 100, false)
	reg(QUERY_DOMAIN, "QUERY_DOMAIN", 2, 1, 200, false)
	reg(SUBMIT_CLAIM, "SUBMIT_CLAIM", 3, 1, 1000, true)
	reg(COMMIT_VOTE, "COMMIT_VOTE", 2, 1, 200, true)
	reg(REVEAL_VOTE, "REVEAL_VOTE", 3, 1, 300, true)
	reg(INITIATE_CHALLENGE, "INITIATE_CHALLENGE", 3, 1, 1000, true)

	// Extended Relationships
	reg(LIST_TRUSTED, "LIST_TRUSTED", 2, 1, 50, false)
	reg(BLOCK_AGENT, "BLOCK_AGENT", 2, 1, 500, true)
	reg(UNBLOCK_AGENT, "UNBLOCK_AGENT", 1, 1, 500, true)
	reg(REVOKE_DELEGATION, "REVOKE_DELEGATION", 1, 1, 300, true)

	// Treasury
	reg(CLAIM_VESTING, "CLAIM_VESTING", 1, 1, 500, true)
	reg(QUERY_VESTING, "QUERY_VESTING", 2, 1, 10, false)
	reg(STAKE, "STAKE", 2, 1, 1000, true)
	reg(UNSTAKE, "UNSTAKE", 1, 1, 500, true)
	reg(QUERY_EXPIRING, "QUERY_EXPIRING", 1, 2, 10, false)
	reg(EXTEND_EXPIRING, "EXTEND_EXPIRING", 2, 1, 300, true)
	reg(SET_SPEND_LIMIT, "SET_SPEND_LIMIT", 3, 1, 200, true)
	reg(QUERY_TREASURY, "QUERY_TREASURY", 0, 4, 10, false)

	// Guardian
	reg(QUERY_ALERTS, "QUERY_ALERTS", 2, 1, 10, false)
	reg(DISMISS_ALERT, "DISMISS_ALERT", 1, 1, 100, true)
	reg(SET_DEFENSE, "SET_DEFENSE", 3, 1, 300, true)
	reg(QUERY_DEFENSES, "QUERY_DEFENSES", 0, 1, 10, false)

	// Access Control
	reg(REGISTER_KEY, "REGISTER_KEY", 4, 1, 500, true)
	reg(REVOKE_KEY, "REVOKE_KEY", 1, 1, 200, true)
	reg(QUERY_PERMISSIONS, "QUERY_PERMISSIONS", 1, 1, 10, false)
	reg(QUERY_SESSION, "QUERY_SESSION", 0, 4, 10, false)

	// Agent Identity
	reg(AGENT_DID, "AGENT_DID", 0, 1, 10, false)
	reg(AGENT_HOME, "AGENT_HOME", 0, 1, 10, false)
	reg(VERIFY_IDENTITY, "VERIFY_IDENTITY", 1, 1, 50, false)
	reg(ATTEST_IDENTITY, "ATTEST_IDENTITY", 3, 1, 500, true)

	// Agent Belief
	reg(GET_BELIEF, "GET_BELIEF", 1, 1, 50, false)
	reg(SET_BELIEF, "SET_BELIEF", 3, 0, 500, true)
	reg(COMPARE_BELIEF, "COMPARE_BELIEF", 1, 1, 30, false)
	reg(BELIEF_COUNT, "BELIEF_COUNT", 1, 1, 50, false)

	// Agent Relationship
	reg(GET_TRUST, "GET_TRUST", 1, 1, 50, false)
	reg(SET_TRUST, "SET_TRUST", 3, 0, 500, true)
	reg(IS_TRUSTED, "IS_TRUSTED", 2, 1, 50, false)
	reg(DELEGATE, "DELEGATE", 3, 1, 500, true)

	// Agent Autonomy
	reg(SCHEDULE_SELF, "SCHEDULE_SELF", 3, 1, 1000, true)
	reg(CANCEL_SCHEDULE, "CANCEL_SCHEDULE", 1, 1, 100, true)
	reg(MODIFY_SCHEDULE, "MODIFY_SCHEDULE", 2, 1, 200, true)
	reg(GET_SCHEDULES, "GET_SCHEDULES", 0, 1, 50, false)

	// System
	reg(CREATE, "CREATE", 3, 1, 32000, true)
	reg(CALL, "CALL", 7, 1, 0, false) // Dynamic gas
	reg(CALLCODE, "CALLCODE", 7, 1, 0, false)
	reg(RETURN, "RETURN", 2, 0, 0, false)
	reg(DELEGATECALL, "DELEGATECALL", 6, 1, 0, false)
	reg(CREATE2, "CREATE2", 4, 1, 32000, true)
	reg(STATICCALL, "STATICCALL", 6, 1, 0, false)
	reg(REVERT, "REVERT", 2, 0, 0, false)
	reg(INVALID, "INVALID", 0, 0, 0, false)
	reg(SELFDESTRUCT, "SELFDESTRUCT", 1, 0, 5000, true)

	// Introspection
	reg(SELF_BALANCE_OP, "SELF_BALANCE", 0, 1, 10, false)
	reg(SELF_STAKE, "SELF_STAKE", 0, 1, 10, false)
	reg(SELF_REPUTATION, "SELF_REPUTATION", 0, 1, 50, false)
	reg(SELF_ACCURACY, "SELF_ACCURACY", 0, 1, 50, false)
}

func reg(code byte, mnemonic string, in, out int, gas uint64, stateMod bool) {
	OpcodeTable[code] = &OpcodeInfo{
		Code:            code,
		Mnemonic:        mnemonic,
		StackInputs:     in,
		StackOutputs:    out,
		GasCost:         gas,
		IsStateModifier: stateMod,
	}
}

func IsValidOpcode(op byte) bool {
	return OpcodeTable[op] != nil
}

func IsAgentOpcode(op byte) bool {
	return (op >= 0xB0 && op <= 0xB5) ||
		(op >= 0xC0 && op <= 0xC3) ||
		(op >= 0xD0 && op <= 0xDF) ||
		(op >= 0xE0 && op <= 0xEF) ||
		(op >= 0xF6 && op <= 0xF9)
}

func IsStateModifying(op byte) bool {
	info := OpcodeTable[op]
	if info == nil {
		return false
	}
	return info.IsStateModifier
}

func GetMnemonic(op byte) string {
	info := OpcodeTable[op]
	if info == nil {
		return "UNDEFINED"
	}
	return info.Mnemonic
}

// DeprecateOpcode marks an opcode as deprecated starting at the given block height.
// After that block, the opcode becomes a no-op (pops inputs, pushes zeros).
func DeprecateOpcode(code byte, atBlock uint64) {
	if OpcodeTable[code] != nil {
		OpcodeTable[code].DeprecatedAtBlock = atBlock
	}
}

// OpcodeByMnemonic returns the opcode byte for a given mnemonic string.
// Used by the gas schedule to map human-readable names to opcode bytes.
func OpcodeByMnemonic(mnemonic string) (byte, bool) {
	for i := 0; i < 256; i++ {
		if OpcodeTable[i] != nil && OpcodeTable[i].Mnemonic == mnemonic {
			return byte(i), true
		}
	}
	return 0, false
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
