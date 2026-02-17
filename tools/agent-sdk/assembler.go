// assembler.go provides a simple BVM bytecode assembler for AI agents.
//
// Converts human-readable BVM assembly into deployable hex bytecode.
// Supports all EVM base opcodes plus Legible Money extensions (0xB0-0xEF).
//
// Example:
//
//	bytecode, err := agentsdk.Assemble(`
//	    PUSH32 0x<evidenceHash>
//	    PUSH32 0x<dataHash>
//	    PUSH32 0x<domain padded to 32 bytes>
//	    SUBMIT_CLAIM
//	    PUSH1 0x00
//	    MSTORE
//	    PUSH1 0x20
//	    PUSH1 0x00
//	    RETURN
//	`)
package agentsdk

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// opcodeMap maps mnemonic names to byte values.
// Covers EVM base set + all Legible Money extensions.
var opcodeMap = map[string]byte{
	// Stop and Arithmetic
	"STOP": 0x00, "ADD": 0x01, "MUL": 0x02, "SUB": 0x03,
	"DIV": 0x04, "SDIV": 0x05, "MOD": 0x06, "SMOD": 0x07,
	"ADDMOD": 0x08, "MULMOD": 0x09, "EXP": 0x0A, "SIGNEXTEND": 0x0B,
	// Comparison and Bitwise
	"LT": 0x10, "GT": 0x11, "SLT": 0x12, "SGT": 0x13,
	"EQ": 0x14, "ISZERO": 0x15, "AND": 0x16, "OR": 0x17,
	"XOR": 0x18, "NOT": 0x19, "BYTE": 0x1A, "SHL": 0x1B,
	"SHR": 0x1C, "SAR": 0x1D,
	// SHA3
	"SHA3": 0x20,
	// Environmental
	"ADDRESS": 0x30, "BALANCE": 0x31, "ORIGIN": 0x32, "CALLER": 0x33,
	"CALLVALUE": 0x34, "CALLDATALOAD": 0x35, "CALLDATASIZE": 0x36,
	"CALLDATACOPY": 0x37, "CODESIZE": 0x38, "CODECOPY": 0x39,
	"GASPRICE": 0x3A, "EXTCODESIZE": 0x3B, "EXTCODECOPY": 0x3C,
	"RETURNDATASIZE": 0x3D, "RETURNDATACOPY": 0x3E, "EXTCODEHASH": 0x3F,
	// Block Information
	"BLOCKHASH": 0x40, "COINBASE": 0x41, "TIMESTAMP": 0x42,
	"NUMBER": 0x43, "PREVRANDAO": 0x44, "GASLIMIT": 0x45,
	"CHAINID": 0x46, "SELFBALANCE": 0x47, "BASEFEE": 0x48,
	// Stack, Memory, Storage
	"POP": 0x50, "MLOAD": 0x51, "MSTORE": 0x52, "MSTORE8": 0x53,
	"SLOAD": 0x54, "SSTORE": 0x55, "JUMP": 0x56, "JUMPI": 0x57,
	"PC": 0x58, "MSIZE": 0x59, "GAS": 0x5A, "JUMPDEST": 0x5B,
	"PUSH0": 0x5F,
	// Push
	"PUSH1": 0x60, "PUSH2": 0x61, "PUSH3": 0x62, "PUSH4": 0x63,
	"PUSH5": 0x64, "PUSH6": 0x65, "PUSH7": 0x66, "PUSH8": 0x67,
	"PUSH9": 0x68, "PUSH10": 0x69, "PUSH11": 0x6A, "PUSH12": 0x6B,
	"PUSH13": 0x6C, "PUSH14": 0x6D, "PUSH15": 0x6E, "PUSH16": 0x6F,
	"PUSH17": 0x70, "PUSH18": 0x71, "PUSH19": 0x72, "PUSH20": 0x73,
	"PUSH21": 0x74, "PUSH22": 0x75, "PUSH23": 0x76, "PUSH24": 0x77,
	"PUSH25": 0x78, "PUSH26": 0x79, "PUSH27": 0x7A, "PUSH28": 0x7B,
	"PUSH29": 0x7C, "PUSH30": 0x7D, "PUSH31": 0x7E, "PUSH32": 0x7F,
	// Dup
	"DUP1": 0x80, "DUP2": 0x81, "DUP3": 0x82, "DUP4": 0x83,
	"DUP5": 0x84, "DUP6": 0x85, "DUP7": 0x86, "DUP8": 0x87,
	"DUP9": 0x88, "DUP10": 0x89, "DUP11": 0x8A, "DUP12": 0x8B,
	"DUP13": 0x8C, "DUP14": 0x8D, "DUP15": 0x8E, "DUP16": 0x8F,
	// Swap
	"SWAP1": 0x90, "SWAP2": 0x91, "SWAP3": 0x92, "SWAP4": 0x93,
	"SWAP5": 0x94, "SWAP6": 0x95, "SWAP7": 0x96, "SWAP8": 0x97,
	"SWAP9": 0x98, "SWAP10": 0x99, "SWAP11": 0x9A, "SWAP12": 0x9B,
	"SWAP13": 0x9C, "SWAP14": 0x9D, "SWAP15": 0x9E, "SWAP16": 0x9F,
	// Log
	"LOG0": 0xA0, "LOG1": 0xA1, "LOG2": 0xA2, "LOG3": 0xA3, "LOG4": 0xA4,
	// System
	"CREATE": 0xF0, "CALL": 0xF1, "CALLCODE": 0xF2, "RETURN": 0xF3,
	"DELEGATECALL": 0xF4, "CREATE2": 0xF5, "STATICCALL": 0xFA,
	"REVERT": 0xFD, "INVALID": 0xFE, "SELFDESTRUCT": 0xFF,
	// --- Legible Money Extensions ---
	// Knowledge (0xB0-0xB5)
	"QUERY_FACT": 0xB0, "QUERY_DOMAIN": 0xB1, "SUBMIT_CLAIM": 0xB2,
	"COMMIT_VOTE": 0xB3, "REVEAL_VOTE": 0xB4, "INITIATE_CHALLENGE": 0xB5,
	// Relationships (0xC0-0xC3)
	"LIST_TRUSTED": 0xC0, "BLOCK_AGENT": 0xC1, "UNBLOCK_AGENT": 0xC2,
	"REVOKE_DELEGATION": 0xC3,
	// Treasury (0xD0-0xD7)
	"CLAIM_VESTING": 0xD0, "QUERY_VESTING": 0xD1, "STAKE": 0xD2,
	"UNSTAKE": 0xD3, "QUERY_EXPIRING": 0xD4, "EXTEND_EXPIRING": 0xD5,
	"SET_SPEND_LIMIT": 0xD6, "QUERY_TREASURY": 0xD7,
	// Guardian (0xD8-0xDB)
	"QUERY_ALERTS": 0xD8, "DISMISS_ALERT": 0xD9,
	"SET_DEFENSE": 0xDA, "QUERY_DEFENSES": 0xDB,
	// Access Control (0xDC-0xDF)
	"REGISTER_KEY": 0xDC, "REVOKE_KEY": 0xDD,
	"QUERY_PERMISSIONS": 0xDE, "QUERY_SESSION": 0xDF,
	// Agent Identity (0xE0-0xE3)
	"AGENT_DID": 0xE0, "AGENT_HOME": 0xE1,
	"VERIFY_IDENTITY": 0xE2, "ATTEST_IDENTITY": 0xE3,
	// Agent Belief (0xE4-0xE7)
	"GET_BELIEF": 0xE4, "SET_BELIEF": 0xE5,
	"COMPARE_BELIEF": 0xE6, "BELIEF_COUNT": 0xE7,
	// Agent Relationship (0xE8-0xEB)
	"GET_TRUST": 0xE8, "SET_TRUST": 0xE9,
	"IS_TRUSTED": 0xEA, "DELEGATE": 0xEB,
	// Agent Autonomy (0xEC-0xEF)
	"SCHEDULE_SELF": 0xEC, "CANCEL_SCHEDULE": 0xED,
	"QUERY_SCHEDULE": 0xEE, "SELF_UPGRADE": 0xEF,
}

// Assemble converts BVM assembly text into bytecode.
//
// Syntax:
//   - One instruction per line
//   - Comments start with ; (stripped)
//   - PUSH1-PUSH32 require a hex argument: PUSH32 0xabcd...
//   - All other instructions take no arguments
//   - Blank lines and leading/trailing whitespace are ignored
//
// Returns the bytecode as a byte slice.
func Assemble(source string) ([]byte, error) {
	var bytecode []byte
	lines := strings.Split(source, "\n")

	for lineNum, line := range lines {
		// Strip comments.
		if idx := strings.Index(line, ";"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		mnemonic := strings.ToUpper(parts[0])

		opcode, ok := opcodeMap[mnemonic]
		if !ok {
			return nil, fmt.Errorf("line %d: unknown opcode %q", lineNum+1, mnemonic)
		}

		bytecode = append(bytecode, opcode)

		// PUSH1-PUSH32 require an immediate operand.
		if opcode >= 0x60 && opcode <= 0x7F {
			expectedBytes := int(opcode - 0x60 + 1)
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: %s requires a hex argument", lineNum+1, mnemonic)
			}
			hexStr := strings.TrimPrefix(parts[1], "0x")
			// Pad to expected length if shorter.
			if len(hexStr) < expectedBytes*2 {
				hexStr = strings.Repeat("0", expectedBytes*2-len(hexStr)) + hexStr
			}
			if len(hexStr) > expectedBytes*2 {
				return nil, fmt.Errorf("line %d: %s argument too long (got %d hex chars, max %d)",
					lineNum+1, mnemonic, len(hexStr), expectedBytes*2)
			}
			data, err := hex.DecodeString(hexStr)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid hex %q: %w", lineNum+1, hexStr, err)
			}
			bytecode = append(bytecode, data...)
		}
	}

	return bytecode, nil
}

// AssembleHex is like Assemble but returns the result as a hex string.
func AssembleHex(source string) (string, error) {
	bytecode, err := Assemble(source)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytecode), nil
}

// PadRight32 pads a byte slice to 32 bytes (right-padded with zeros).
// Useful for encoding domain names and other strings for BVM stack consumption.
func PadRight32(data []byte) [32]byte {
	var result [32]byte
	copy(result[:], data)
	return result
}

// PadLeft32 pads a byte slice to 32 bytes (left-padded with zeros).
// Useful for encoding integers and hashes for BVM stack consumption.
func PadLeft32(data []byte) [32]byte {
	var result [32]byte
	copy(result[32-len(data):], data)
	return result
}
