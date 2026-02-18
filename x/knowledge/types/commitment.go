package types

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// ComputeCommitmentHash generates a commitment hash for a verification verdict.
// Uses domain-separated, length-prefixed encoding to prevent second-preimage attacks.
//
//	hash = SHA-256("LGM.commitment.v1:" || u32BE(len(roundID)) || roundID
//	              || u32BE(len(verdict)) || verdict || u64BE(confidence)
//	              || u32BE(len(salt)) || salt)
func ComputeCommitmentHash(roundID, verdict string, confidence uint64, salt string) string {
	h := sha256.New()
	// Domain separation
	h.Write([]byte("LGM.commitment.v1:"))
	// Length-prefixed roundID
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(roundID)))
	h.Write(lenBuf)
	h.Write([]byte(roundID))
	// Length-prefixed verdict
	binary.BigEndian.PutUint32(lenBuf, uint32(len(verdict)))
	h.Write(lenBuf)
	h.Write([]byte(verdict))
	// Fixed-size confidence
	confBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(confBytes, confidence)
	h.Write(confBytes)
	// Length-prefixed salt
	binary.BigEndian.PutUint32(lenBuf, uint32(len(salt)))
	h.Write(lenBuf)
	h.Write([]byte(salt))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// VerifyCommitmentHash checks that a reveal matches its prior commitment hash.
func VerifyCommitmentHash(commitmentHash, roundID, verdict string, confidence uint64, salt string) bool {
	computed := ComputeCommitmentHash(roundID, verdict, confidence, salt)
	return computed == commitmentHash
}
