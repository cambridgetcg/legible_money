// templates.go provides pre-built BVM contract bytecode templates for common
// knowledge operations. Agents can deploy these directly without writing assembly.
//
// Each template returns deployable bytecode that accepts calldata arguments
// and returns results via RETURN. Templates use the standard ABI-like encoding:
// all arguments and return values are 32-byte words.
package agentsdk

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// SubmitClaimBytecode returns bytecode that submits a knowledge claim.
// The contract pushes domain, dataHash, and evidenceHash to the stack,
// calls SUBMIT_CLAIM, and returns the 32-byte claim ID.
//
// Arguments are baked into the bytecode (no calldata needed).
func SubmitClaimBytecode(domain string, contentHash, evidenceHash [32]byte) []byte {
	domainPadded := PadRight32([]byte(domain))

	// Stack order for SUBMIT_CLAIM: pop domain, pop dataHash, pop evidenceHash
	// So push in reverse: evidenceHash first, dataHash second, domain last.
	src := fmt.Sprintf(`
		PUSH32 0x%s  ; evidenceHash (popped third)
		PUSH32 0x%s  ; dataHash (popped second)
		PUSH32 0x%s  ; domain (popped first)
		SUBMIT_CLAIM ; pops 3, pushes claimId
		PUSH1 0x00   ; memory offset
		MSTORE       ; store claimId at mem[0:32]
		PUSH1 0x20   ; length = 32
		PUSH1 0x00   ; offset = 0
		RETURN       ; return mem[0:32]
	`,
		hex.EncodeToString(evidenceHash[:]),
		hex.EncodeToString(contentHash[:]),
		hex.EncodeToString(domainPadded[:]),
	)

	bytecode, err := Assemble(src)
	if err != nil {
		panic(fmt.Sprintf("SubmitClaimBytecode: assembly failed: %v", err))
	}
	return bytecode
}

// QueryFactBytecode returns bytecode that queries a fact by ID.
// Returns 3 words: exists (0/1), confidence (scaled to 1e18), domain (32 bytes).
func QueryFactBytecode(factIDHash [32]byte) []byte {
	src := fmt.Sprintf(`
		PUSH32 0x%s  ; factId hash
		QUERY_FACT   ; pops 1, pushes 3: exists, confidence, domain
		PUSH1 0x40   ; store domain at mem[64:96]
		MSTORE
		PUSH1 0x20   ; store confidence at mem[32:64]
		MSTORE
		PUSH1 0x00   ; store exists at mem[0:32]
		MSTORE
		PUSH1 0x60   ; length = 96 (3 words)
		PUSH1 0x00   ; offset = 0
		RETURN
	`, hex.EncodeToString(factIDHash[:]))

	bytecode, err := Assemble(src)
	if err != nil {
		panic(fmt.Sprintf("QueryFactBytecode: assembly failed: %v", err))
	}
	return bytecode
}

// CommitVoteBytecode returns bytecode that commits a verification vote.
// roundIDHash is the SHA256 of the round ID string.
// commitmentHash is the SHA256 commitment (from ComputeCommitment).
func CommitVoteBytecode(roundIDHash, commitmentHash [32]byte) []byte {
	// COMMIT_VOTE: pop claimId, pop voteHash → push success
	src := fmt.Sprintf(`
		PUSH32 0x%s  ; commitmentHash (popped second)
		PUSH32 0x%s  ; roundId hash (popped first)
		COMMIT_VOTE  ; pops 2, pushes success (0/1)
		PUSH1 0x00
		MSTORE
		PUSH1 0x20
		PUSH1 0x00
		RETURN
	`,
		hex.EncodeToString(commitmentHash[:]),
		hex.EncodeToString(roundIDHash[:]),
	)

	bytecode, err := Assemble(src)
	if err != nil {
		panic(fmt.Sprintf("CommitVoteBytecode: assembly failed: %v", err))
	}
	return bytecode
}

// RevealVoteBytecode returns bytecode that reveals a verification vote.
// vote encodes verdict + confidence: 0=reject, >0=accept with confidence value.
func RevealVoteBytecode(roundIDHash [32]byte, vote uint64, saltHash [32]byte) []byte {
	// REVEAL_VOTE: pop claimId, pop vote, pop salt → push success
	voteBytes := PadLeft32(uint64ToBytes(vote))
	src := fmt.Sprintf(`
		PUSH32 0x%s  ; salt (popped third)
		PUSH32 0x%s  ; vote (popped second)
		PUSH32 0x%s  ; roundId hash (popped first)
		REVEAL_VOTE  ; pops 3, pushes success (0/1)
		PUSH1 0x00
		MSTORE
		PUSH1 0x20
		PUSH1 0x00
		RETURN
	`,
		hex.EncodeToString(saltHash[:]),
		hex.EncodeToString(voteBytes[:]),
		hex.EncodeToString(roundIDHash[:]),
	)

	bytecode, err := Assemble(src)
	if err != nil {
		panic(fmt.Sprintf("RevealVoteBytecode: assembly failed: %v", err))
	}
	return bytecode
}

// ChallengeBytecode returns bytecode that challenges an existing fact.
func ChallengeBytecode(factIDHash, evidenceHash [32]byte, stake uint64) []byte {
	stakeBytes := PadLeft32(uint64ToBytes(stake))
	// INITIATE_CHALLENGE: pop factId, pop evidenceHash, pop stake → push roundId
	src := fmt.Sprintf(`
		PUSH32 0x%s        ; stake (popped third)
		PUSH32 0x%s        ; evidenceHash (popped second)
		PUSH32 0x%s        ; factId hash (popped first)
		INITIATE_CHALLENGE ; pops 3, pushes roundId
		PUSH1 0x00
		MSTORE
		PUSH1 0x20
		PUSH1 0x00
		RETURN
	`,
		hex.EncodeToString(stakeBytes[:]),
		hex.EncodeToString(evidenceHash[:]),
		hex.EncodeToString(factIDHash[:]),
	)

	bytecode, err := Assemble(src)
	if err != nil {
		panic(fmt.Sprintf("ChallengeBytecode: assembly failed: %v", err))
	}
	return bytecode
}

// QueryDomainBytecode returns bytecode that queries facts in a domain.
// Returns 1 word: fact count.
func QueryDomainBytecode(domain string, limit uint64) []byte {
	domainPadded := PadRight32([]byte(domain))
	limitBytes := PadLeft32(uint64ToBytes(limit))
	// QUERY_DOMAIN: pop domain, pop limit → push count
	src := fmt.Sprintf(`
		PUSH32 0x%s  ; limit (popped second)
		PUSH32 0x%s  ; domain (popped first)
		QUERY_DOMAIN ; pops 2, pushes count
		PUSH1 0x00
		MSTORE
		PUSH1 0x20
		PUSH1 0x00
		RETURN
	`,
		hex.EncodeToString(limitBytes[:]),
		hex.EncodeToString(domainPadded[:]),
	)

	bytecode, err := Assemble(src)
	if err != nil {
		panic(fmt.Sprintf("QueryDomainBytecode: assembly failed: %v", err))
	}
	return bytecode
}

// CalldataClaimBytecode returns a reusable contract that reads claim parameters
// from calldata. Deploy once, call multiple times with different claims.
//
// Calldata layout (96 bytes):
//
//	[0:32]   domain (right-padded)
//	[32:64]  contentHash
//	[64:96]  evidenceHash
//
// Returns: 32-byte claim ID hash.
func CalldataClaimBytecode() []byte {
	src := `
		; Load evidenceHash from calldata[64:96]
		PUSH1 0x40       ; offset 64
		CALLDATALOAD     ; stack: [evidenceHash]

		; Load contentHash from calldata[32:64]
		PUSH1 0x20       ; offset 32
		CALLDATALOAD     ; stack: [contentHash, evidenceHash]

		; Load domain from calldata[0:32]
		PUSH1 0x00       ; offset 0
		CALLDATALOAD     ; stack: [domain, contentHash, evidenceHash]

		SUBMIT_CLAIM     ; pops 3, pushes claimId

		PUSH1 0x00
		MSTORE
		PUSH1 0x20
		PUSH1 0x00
		RETURN
	`

	bytecode, err := Assemble(src)
	if err != nil {
		panic(fmt.Sprintf("CalldataClaimBytecode: assembly failed: %v", err))
	}
	return bytecode
}

// --- Helpers ---

// HashToBytes32 computes SHA-256 and returns a fixed 32-byte array.
// Useful for converting string IDs to BVM-compatible 32-byte words.
func HashToBytes32(s string) [32]byte {
	return sha256.Sum256([]byte(s))
}

func uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
	return b
}
