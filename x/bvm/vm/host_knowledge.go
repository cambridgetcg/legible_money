package vm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/big"
)

// KnowledgeKeeper is the interface for the knowledge module keeper.
// Matches x/bvm/types/expected_keepers.go KnowledgeKeeper.
type KnowledgeKeeper interface {
	GetFact(ctx context.Context, factId string) (fact interface{}, found bool)
	GetFactsByDomain(ctx context.Context, domain string) []interface{}
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
	// Extract confidence and domain from fact (type assertion)
	// The actual structure depends on x/knowledge types.
	// For now, return a default confidence of 0.8 scaled to uint64 (0.8 * 1e18)
	return true, 800000000000000000, []byte("general")
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
	for i := range facts {
		// Hash the index as placeholder ID
		h := sha256.Sum256([]byte(fmt.Sprintf("%s/%d", domainStr, i)))
		ids[i] = h[:]
	}
	return uint64(len(facts)), ids
}

func (h *KnowledgeHost) SubmitClaim(callerDID string, domain, dataHash, evidenceHash []byte, blockNumber uint64) []byte {
	// Claims submitted through BVM need to go through the normal claim lifecycle.
	// For now, return a deterministic claim ID.
	input := fmt.Sprintf("claim/%s/%x/%x/%d", callerDID, domain, dataHash, blockNumber)
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}

func (h *KnowledgeHost) CommitVote(callerDID string, claimId, voteHash []byte, blockNumber uint64) bool {
	// Vote commitment through BVM
	return callerDID != "" && len(claimId) > 0 && len(voteHash) > 0
}

func (h *KnowledgeHost) RevealVote(callerDID string, claimId []byte, vote, salt *big.Int, blockNumber uint64) bool {
	// Vote reveal through BVM
	return callerDID != "" && len(claimId) > 0 && vote != nil && salt != nil
}

func (h *KnowledgeHost) InitiateChallenge(callerDID string, factId, evidenceHash []byte, stake *big.Int, blockNumber uint64) []byte {
	// Challenge through BVM
	if callerDID == "" || len(factId) == 0 || stake == nil || stake.Sign() == 0 {
		return nil
	}
	input := fmt.Sprintf("challenge/%s/%x/%d", callerDID, factId, blockNumber)
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}
