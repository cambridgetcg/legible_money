package app

import "sync"

// LocalCommitment holds a validator's local commitment data (salt and verdict)
// that must be retained between the commit and reveal phases.
//
// This is in-memory only for Phase 2. If the validator restarts between commit
// and reveal phases, they forfeit pending rewards. File-based persistence
// will be added in Phase 3.
type LocalCommitment struct {
	RoundID    string
	Verdict    string // "accept", "reject", "abstain"
	Confidence uint64 // 0-1000000
	Salt       string // random salt used in commitment hash
	Height     uint64 // block height when committed
}

// LocalCommitmentStore manages validator-local commitment data.
// Thread-safe for concurrent access from ExtendVote calls.
type LocalCommitmentStore struct {
	mu          sync.RWMutex
	commitments map[string]LocalCommitment // roundID → commitment
}

// NewLocalCommitmentStore creates a new in-memory commitment store.
func NewLocalCommitmentStore() *LocalCommitmentStore {
	return &LocalCommitmentStore{
		commitments: make(map[string]LocalCommitment),
	}
}

// Store saves a local commitment for a round.
func (s *LocalCommitmentStore) Store(commitment LocalCommitment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commitments[commitment.RoundID] = commitment
}

// Get retrieves a local commitment for a round.
func (s *LocalCommitmentStore) Get(roundID string) (LocalCommitment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.commitments[roundID]
	return c, ok
}

// Delete removes a local commitment (after successful reveal or round completion).
func (s *LocalCommitmentStore) Delete(roundID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.commitments, roundID)
}

// CleanupExpired removes commitments for rounds older than the given height.
func (s *LocalCommitmentStore) CleanupExpired(currentHeight uint64, maxAge uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if currentHeight < maxAge {
		return
	}
	cutoff := currentHeight - maxAge

	for id, c := range s.commitments {
		if c.Height < cutoff {
			delete(s.commitments, id)
		}
	}
}

// Count returns the number of stored commitments.
func (s *LocalCommitmentStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.commitments)
}
