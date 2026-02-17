package vm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// KnowledgeKeeper is the interface for the knowledge module keeper.
// Matches x/bvm/types/expected_keepers.go KnowledgeKeeper.
type KnowledgeKeeper interface {
	// Read methods
	GetFact(ctx context.Context, factId string) (fact interface{}, found bool)
	GetFactsByDomain(ctx context.Context, domain string) []interface{}

	// Write methods for BVM opcode integration
	SubmitClaimFromBVM(ctx context.Context, submitterDID, domain, contentHash, evidenceHash, logicZone string, stakeAmount uint64) (claimId string, err error)
	CommitVoteFromBVM(ctx context.Context, validatorDID, roundId, commitmentHash string) error
	RevealVoteFromBVM(ctx context.Context, validatorDID, roundId, verdict string, confidence uint64, salt string) error
	ChallengeFactFromBVM(ctx context.Context, challengerDID, factId, evidenceHash string, stakeAmount uint64) (roundId string, err error)
}

// FactData is a duck-typing interface for extracting fields from a Fact.
// This avoids a direct import of x/knowledge/types which would create circular deps.
// The x/knowledge Fact type (proto-generated) already has these getter methods.
type FactData interface {
	GetConfidence() uint64
	GetDomain() string
}

// KnowledgeHost bridges BVM knowledge opcodes (0xB0-0xB5) to the x/knowledge keeper.
type KnowledgeHost struct {
	keeper KnowledgeKeeper
	ctx    context.Context
}

func NewKnowledgeHost(keeper KnowledgeKeeper, ctx context.Context) *KnowledgeHost {
	return &KnowledgeHost{keeper: keeper, ctx: ctx}
}

func (h *KnowledgeHost) QueryFact(factId []byte) (exists bool, confidence uint64, domain []byte) {
	if h.keeper == nil {
		return false, 0, nil
	}
	factIdHex := fmt.Sprintf("%x", factId)
	fact, found := h.keeper.GetFact(h.ctx, factIdHex)
	if !found || fact == nil {
		return false, 0, nil
	}
	// Extract confidence and domain from the fact via the FactData interface.
	// The adapter returns the actual x/knowledge Fact which satisfies FactData.
	if fd, ok := fact.(FactData); ok {
		// Confidence is 0-1,000,000 on chain; scale to uint64 (multiply by 1e12)
		// so BVM contracts can work with 1e18 precision.
		conf := fd.GetConfidence() * 1_000_000_000_000
		return true, conf, []byte(fd.GetDomain())
	}
	// Fallback: fact exists but type is unknown — return exists=true with zero values.
	return true, 0, nil
}

// FactIdentifiable is a duck-typing interface for extracting the ID from a Fact.
type FactIdentifiable interface {
	GetId() string
}

func (h *KnowledgeHost) QueryDomain(domain []byte, limit uint64) (count uint64, factIds [][]byte) {
	if h.keeper == nil {
		return 0, nil
	}
	domainStr := string(domain)
	facts := h.keeper.GetFactsByDomain(h.ctx, domainStr)
	if uint64(len(facts)) > limit && limit > 0 {
		facts = facts[:limit]
	}
	ids := make([][]byte, len(facts))
	for i, fact := range facts {
		if fi, ok := fact.(FactIdentifiable); ok {
			// Use actual fact ID hashed to 32 bytes for BVM consumption.
			idHash := sha256.Sum256([]byte(fi.GetId()))
			ids[i] = idHash[:]
		} else {
			// Fallback: deterministic hash from domain + index.
			idHash := sha256.Sum256([]byte(fmt.Sprintf("%s/%d", domainStr, i)))
			ids[i] = idHash[:]
		}
	}
	return uint64(len(facts)), ids
}

func (h *KnowledgeHost) SubmitClaim(callerDID string, domain, dataHash, evidenceHash []byte, blockNumber uint64) []byte {
	if h.keeper == nil || callerDID == "" {
		return nil
	}
	domainStr := string(trimNullBytes(domain))
	contentHashHex := hex.EncodeToString(dataHash)
	evidenceHashHex := hex.EncodeToString(evidenceHash)

	claimId, err := h.keeper.SubmitClaimFromBVM(
		h.ctx, callerDID, domainStr, contentHashHex, evidenceHashHex,
		"empirical", // default logic zone for BVM-submitted claims
		0,           // stake handled by caller's escrow, 0 = use minimum
	)
	if err != nil {
		// On failure, return deterministic fallback ID so caller knows it failed
		return nil
	}
	// Return claim ID as bytes
	hash := sha256.Sum256([]byte(claimId))
	return hash[:]
}

func (h *KnowledgeHost) CommitVote(callerDID string, claimId, voteHash []byte, blockNumber uint64) bool {
	if h.keeper == nil || callerDID == "" || len(claimId) == 0 || len(voteHash) == 0 {
		return false
	}
	roundIdHex := hex.EncodeToString(claimId)
	commitHashHex := hex.EncodeToString(voteHash)

	err := h.keeper.CommitVoteFromBVM(h.ctx, callerDID, roundIdHex, commitHashHex)
	return err == nil
}

func (h *KnowledgeHost) RevealVote(callerDID string, claimId []byte, vote, salt *big.Int, blockNumber uint64) bool {
	if h.keeper == nil || callerDID == "" || len(claimId) == 0 || vote == nil || salt == nil {
		return false
	}
	roundIdHex := hex.EncodeToString(claimId)
	// Interpret vote: lower 20 bits = verdict (0=reject, >0=accept),
	// upper bits = confidence (0-1,000,000 scale).
	verdict := "reject"
	if vote.Sign() > 0 {
		verdict = "accept"
	}
	// Extract confidence: if vote > 1, it encodes confidence directly.
	// Otherwise default to 800,000 (0.8 on 1M scale).
	confidence := uint64(800_000)
	if vote.Uint64() > 1 {
		confidence = vote.Uint64()
		if confidence > 1_000_000 {
			confidence = 1_000_000
		}
	}
	saltHex := fmt.Sprintf("%x", salt)

	err := h.keeper.RevealVoteFromBVM(h.ctx, callerDID, roundIdHex, verdict, confidence, saltHex)
	return err == nil
}

func (h *KnowledgeHost) InitiateChallenge(callerDID string, factId, evidenceHash []byte, stake *big.Int, blockNumber uint64) []byte {
	if h.keeper == nil || callerDID == "" || len(factId) == 0 || stake == nil || stake.Sign() == 0 {
		return nil
	}
	factIdHex := hex.EncodeToString(factId)
	evidenceHashHex := hex.EncodeToString(evidenceHash)

	roundId, err := h.keeper.ChallengeFactFromBVM(
		h.ctx, callerDID, factIdHex, evidenceHashHex, stake.Uint64(),
	)
	if err != nil {
		return nil
	}
	hash := sha256.Sum256([]byte(roundId))
	return hash[:]
}

// trimNullBytes removes trailing null bytes from a byte slice.
func trimNullBytes(b []byte) []byte {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			return b[:i+1]
		}
	}
	return nil
}
