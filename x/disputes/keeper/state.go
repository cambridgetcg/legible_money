package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/disputes/types"
)

// ---------- Dispute CRUD ----------

// SetDispute stores a dispute in the KV store.
func (k Keeper) SetDispute(ctx sdk.Context, dispute types.Dispute) {
	store := k.storeService.OpenKVStore(ctx)
	key := disputeKey(dispute.Id)
	bz, err := json.Marshal(dispute)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal dispute: %v", err))
	}
	_ = store.Set(key, bz)

	// Maintain fact→dispute index
	indexKey := disputeByFactKey(dispute.FactId, dispute.Id)
	_ = store.Set(indexKey, []byte(dispute.Id))

	// Maintain active dispute index
	if dispute.Status != string(types.DisputeStatusResolved) && dispute.Status != string(types.DisputeStatusExpired) {
		activeKey := activeDisputeKey(dispute.Id)
		_ = store.Set(activeKey, []byte(dispute.Id))
	} else {
		// Remove from active index if resolved/expired
		activeKey := activeDisputeKey(dispute.Id)
		_ = store.Delete(activeKey)
	}
}

// GetDispute retrieves a dispute by ID.
func (k Keeper) GetDispute(ctx sdk.Context, id string) (types.Dispute, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := disputeKey(id)
	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.Dispute{}, false
	}
	var dispute types.Dispute
	if err := json.Unmarshal(bz, &dispute); err != nil {
		return types.Dispute{}, false
	}
	return dispute, true
}

// DeleteDispute removes a dispute from the store.
func (k Keeper) DeleteDispute(ctx sdk.Context, id string) {
	dispute, found := k.GetDispute(ctx, id)
	if !found {
		return
	}
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(disputeKey(id))
	_ = store.Delete(disputeByFactKey(dispute.FactId, id))
	_ = store.Delete(activeDisputeKey(id))
}

// IterateDisputes iterates over all disputes. Return true from cb to stop.
func (k Keeper) IterateDisputes(ctx sdk.Context, cb func(types.Dispute) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.DisputeKeyPrefix, prefixEndBytes(types.DisputeKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var dispute types.Dispute
		if err := json.Unmarshal(iter.Value(), &dispute); err != nil {
			continue
		}
		if cb(dispute) {
			break
		}
	}
}

// GetDisputesByFact returns all disputes for a given fact.
func (k Keeper) GetDisputesByFact(ctx sdk.Context, factId string) []types.Dispute {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.DisputeByFactPrefix, []byte(factId+":")...)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var disputes []types.Dispute
	for ; iter.Valid(); iter.Next() {
		disputeId := string(iter.Value())
		dispute, found := k.GetDispute(ctx, disputeId)
		if found {
			disputes = append(disputes, dispute)
		}
	}
	return disputes
}

// GetActiveDisputes returns all non-resolved, non-expired disputes.
func (k Keeper) GetActiveDisputes(ctx sdk.Context) []types.Dispute {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ActiveDisputePrefix, prefixEndBytes(types.ActiveDisputePrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var disputes []types.Dispute
	for ; iter.Valid(); iter.Next() {
		disputeId := string(iter.Value())
		dispute, found := k.GetDispute(ctx, disputeId)
		if found {
			disputes = append(disputes, dispute)
		}
	}
	return disputes
}

// GetActiveDisputeCountForFact returns the number of active disputes for a fact.
func (k Keeper) GetActiveDisputeCountForFact(ctx sdk.Context, factId string) uint32 {
	disputes := k.GetDisputesByFact(ctx, factId)
	var count uint32
	for _, d := range disputes {
		if d.Status != string(types.DisputeStatusResolved) && d.Status != string(types.DisputeStatusExpired) {
			count++
		}
	}
	return count
}

// ---------- Evidence CRUD ----------

// SetEvidence stores dispute evidence in the KV store.
func (k Keeper) SetEvidence(ctx sdk.Context, evidence types.DisputeEvidence) {
	store := k.storeService.OpenKVStore(ctx)
	key := evidenceKey(evidence.DisputeId, evidence.Id)
	bz, err := json.Marshal(evidence)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal evidence: %v", err))
	}
	_ = store.Set(key, bz)
}

// GetEvidence retrieves evidence by dispute ID and evidence ID.
func (k Keeper) GetEvidence(ctx sdk.Context, disputeId, evidenceId string) (types.DisputeEvidence, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := evidenceKey(disputeId, evidenceId)
	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.DisputeEvidence{}, false
	}
	var evidence types.DisputeEvidence
	if err := json.Unmarshal(bz, &evidence); err != nil {
		return types.DisputeEvidence{}, false
	}
	return evidence, true
}

// GetEvidenceByDispute returns all evidence for a dispute.
func (k Keeper) GetEvidenceByDispute(ctx sdk.Context, disputeId string) []types.DisputeEvidence {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.EvidenceKeyPrefix, []byte(disputeId+":")...)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var evidences []types.DisputeEvidence
	for ; iter.Valid(); iter.Next() {
		var evidence types.DisputeEvidence
		if err := json.Unmarshal(iter.Value(), &evidence); err != nil {
			continue
		}
		evidences = append(evidences, evidence)
	}
	return evidences
}

// ---------- Vote CRUD ----------

// SetVote stores a dispute vote in the KV store.
func (k Keeper) SetVote(ctx sdk.Context, vote types.DisputeVote) {
	store := k.storeService.OpenKVStore(ctx)
	key := voteKey(vote.DisputeId, vote.Arbiter)
	bz, err := json.Marshal(vote)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal vote: %v", err))
	}
	_ = store.Set(key, bz)
}

// GetVote retrieves a vote by dispute ID and arbiter address.
func (k Keeper) GetVote(ctx sdk.Context, disputeId, arbiter string) (types.DisputeVote, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := voteKey(disputeId, arbiter)
	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.DisputeVote{}, false
	}
	var vote types.DisputeVote
	if err := json.Unmarshal(bz, &vote); err != nil {
		return types.DisputeVote{}, false
	}
	return vote, true
}

// GetVotes returns all votes for a dispute.
func (k Keeper) GetVotes(ctx sdk.Context, disputeId string) []types.DisputeVote {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.VoteKeyPrefix, []byte(disputeId+":")...)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var votes []types.DisputeVote
	for ; iter.Valid(); iter.Next() {
		var vote types.DisputeVote
		if err := json.Unmarshal(iter.Value(), &vote); err != nil {
			continue
		}
		votes = append(votes, vote)
	}
	return votes
}

// ---------- Key Construction Helpers ----------

func disputeKey(id string) []byte {
	return append(types.DisputeKeyPrefix, []byte(id)...)
}

func disputeByFactKey(factId, disputeId string) []byte {
	return append(types.DisputeByFactPrefix, []byte(factId+":"+disputeId)...)
}

func activeDisputeKey(disputeId string) []byte {
	return append(types.ActiveDisputePrefix, []byte(disputeId)...)
}

func evidenceKey(disputeId, evidenceId string) []byte {
	return append(types.EvidenceKeyPrefix, []byte(disputeId+":"+evidenceId)...)
}

func voteKey(disputeId, arbiter string) []byte {
	return append(types.VoteKeyPrefix, []byte(disputeId+":"+arbiter)...)
}

func commitmentKey(disputeId, submitter string) []byte {
	return append(types.EvidenceCommitmentPrefix, []byte(disputeId+":"+submitter)...)
}

// ---------- Evidence Commitment CRUD ----------

// SetEvidenceCommitment stores an evidence commitment.
func (k Keeper) SetEvidenceCommitment(ctx sdk.Context, commitment types.EvidenceCommitment) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(commitment)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal evidence commitment: %v", err))
	}
	_ = store.Set(commitmentKey(commitment.DisputeId, commitment.Submitter), bz)
}

// GetEvidenceCommitment retrieves an evidence commitment.
func (k Keeper) GetEvidenceCommitment(ctx sdk.Context, disputeId, submitter string) (types.EvidenceCommitment, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(commitmentKey(disputeId, submitter))
	if err != nil || bz == nil {
		return types.EvidenceCommitment{}, false
	}
	var commitment types.EvidenceCommitment
	if err := json.Unmarshal(bz, &commitment); err != nil {
		return types.EvidenceCommitment{}, false
	}
	return commitment, true
}

// GetUnrevealedCommitments returns all unrevealed commitments for a dispute.
func (k Keeper) GetUnrevealedCommitments(ctx sdk.Context, disputeId string) []types.EvidenceCommitment {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.EvidenceCommitmentPrefix, []byte(disputeId+":")...)

	var commitments []types.EvidenceCommitment
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var commitment types.EvidenceCommitment
		if err := json.Unmarshal(iter.Value(), &commitment); err != nil {
			continue
		}
		if !commitment.Revealed {
			commitments = append(commitments, commitment)
		}
	}
	return commitments
}
