package keeper

import (
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/partnerships/types"
)

// ---------- Partnership CRUD ----------

// SetPartnership stores a partnership and maintains bidirectional indexes.
func (k Keeper) SetPartnership(ctx sdk.Context, p *types.Partnership) {
	kvStore := k.storeService.OpenKVStore(ctx)

	bz, err := proto.Marshal(p)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal partnership: %v", err))
	}

	key := partnershipKey(p.Id)
	_ = kvStore.Set(key, bz)

	// Bidirectional indexes: ByHuman and ByAgent
	_ = kvStore.Set(byHumanKey(p.HumanAddr, p.Id), []byte(p.Id))
	_ = kvStore.Set(byAgentKey(p.AgentAddr, p.Id), []byte(p.Id))
}

// GetPartnership retrieves a partnership by ID.
func (k Keeper) GetPartnership(ctx sdk.Context, id string) (*types.Partnership, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(partnershipKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var p types.Partnership
	if err := proto.Unmarshal(bz, &p); err != nil {
		return nil, false
	}
	return &p, true
}

// DeletePartnership removes a partnership and its indexes.
func (k Keeper) DeletePartnership(ctx sdk.Context, p *types.Partnership) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(partnershipKey(p.Id))
	_ = kvStore.Delete(byHumanKey(p.HumanAddr, p.Id))
	_ = kvStore.Delete(byAgentKey(p.AgentAddr, p.Id))
}

// GetAllPartnerships returns all partnerships.
func (k Keeper) GetAllPartnerships(ctx sdk.Context) []*types.Partnership {
	var partnerships []*types.Partnership
	k.IteratePartnerships(ctx, func(p *types.Partnership) bool {
		partnerships = append(partnerships, p)
		return false
	})
	return partnerships
}

// IteratePartnerships iterates over all partnerships. Return true to stop.
func (k Keeper) IteratePartnerships(ctx sdk.Context, cb func(*types.Partnership) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.PartnershipKeyPrefix, prefixEndBytes(types.PartnershipKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var p types.Partnership
		if err := proto.Unmarshal(iter.Value(), &p); err != nil {
			continue
		}
		if cb(&p) {
			break
		}
	}
}

// GetPartnershipsByHuman returns all partnership IDs for a human address.
func (k Keeper) GetPartnershipsByHuman(ctx sdk.Context, humanAddr string) []string {
	return k.getPartnershipIdsByPrefix(ctx, byHumanPrefix(humanAddr))
}

// GetPartnershipsByAgent returns all partnership IDs for an agent address.
func (k Keeper) GetPartnershipsByAgent(ctx sdk.Context, agentAddr string) []string {
	return k.getPartnershipIdsByPrefix(ctx, byAgentPrefix(agentAddr))
}

// GetPartnershipsByParticipant returns all partnerships where the address is human or agent.
func (k Keeper) GetPartnershipsByParticipant(ctx sdk.Context, addr string) []*types.Partnership {
	seen := make(map[string]bool)
	var result []*types.Partnership

	for _, id := range k.GetPartnershipsByHuman(ctx, addr) {
		if !seen[id] {
			seen[id] = true
			if p, found := k.GetPartnership(ctx, id); found {
				result = append(result, p)
			}
		}
	}
	for _, id := range k.GetPartnershipsByAgent(ctx, addr) {
		if !seen[id] {
			seen[id] = true
			if p, found := k.GetPartnership(ctx, id); found {
				result = append(result, p)
			}
		}
	}

	return result
}

func (k Keeper) getPartnershipIdsByPrefix(ctx sdk.Context, prefix []byte) []string {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var ids []string
	for ; iter.Valid(); iter.Next() {
		ids = append(ids, string(iter.Value()))
	}
	return ids
}

// ---------- ConsensusOperation CRUD ----------

// SetConsensusOperation stores a consensus operation.
func (k Keeper) SetConsensusOperation(ctx sdk.Context, op *types.ConsensusOperation) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(op)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal consensus operation: %v", err))
	}
	_ = kvStore.Set(consensusOpKey(op.Id), bz)
}

// GetConsensusOperation retrieves a consensus operation by ID.
func (k Keeper) GetConsensusOperation(ctx sdk.Context, id string) (*types.ConsensusOperation, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(consensusOpKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var op types.ConsensusOperation
	if err := proto.Unmarshal(bz, &op); err != nil {
		return nil, false
	}
	return &op, true
}

// DeleteConsensusOperation removes a consensus operation.
func (k Keeper) DeleteConsensusOperation(ctx sdk.Context, id string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(consensusOpKey(id))
}

// GetAllConsensusOperations returns all consensus operations.
func (k Keeper) GetAllConsensusOperations(ctx sdk.Context) []*types.ConsensusOperation {
	var ops []*types.ConsensusOperation
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ConsensusOpKeyPrefix, prefixEndBytes(types.ConsensusOpKeyPrefix))
	if err != nil {
		return ops
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var op types.ConsensusOperation
		if err := proto.Unmarshal(iter.Value(), &op); err != nil {
			continue
		}
		ops = append(ops, &op)
	}
	return ops
}

// ---------- SafetyFreeze CRUD ----------

// SetSafetyFreeze stores a safety freeze.
func (k Keeper) SetSafetyFreeze(ctx sdk.Context, sf *types.SafetyFreeze) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(sf)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal safety freeze: %v", err))
	}
	_ = kvStore.Set(safetyFreezeKey(sf.PartnershipId), bz)
}

// GetSafetyFreeze retrieves a safety freeze by partnership ID.
func (k Keeper) GetSafetyFreeze(ctx sdk.Context, partnershipId string) (*types.SafetyFreeze, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(safetyFreezeKey(partnershipId))
	if err != nil || bz == nil {
		return nil, false
	}
	var sf types.SafetyFreeze
	if err := proto.Unmarshal(bz, &sf); err != nil {
		return nil, false
	}
	return &sf, true
}

// DeleteSafetyFreeze removes a safety freeze.
func (k Keeper) DeleteSafetyFreeze(ctx sdk.Context, partnershipId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(safetyFreezeKey(partnershipId))
}

// GetAllSafetyFreezes returns all safety freezes.
func (k Keeper) GetAllSafetyFreezes(ctx sdk.Context) []*types.SafetyFreeze {
	var freezes []*types.SafetyFreeze
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.SafetyFreezeKeyPrefix, prefixEndBytes(types.SafetyFreezeKeyPrefix))
	if err != nil {
		return freezes
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var sf types.SafetyFreeze
		if err := proto.Unmarshal(iter.Value(), &sf); err != nil {
			continue
		}
		freezes = append(freezes, &sf)
	}
	return freezes
}

// ---------- CoercionSignal CRUD ----------

// SetCoercionSignal stores a coercion signal.
func (k Keeper) SetCoercionSignal(ctx sdk.Context, cs *types.CoercionSignal) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(cs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal coercion signal: %v", err))
	}
	_ = kvStore.Set(coercionSignalKey(cs.SignalId), bz)
}

// GetCoercionSignal retrieves a coercion signal by ID.
func (k Keeper) GetCoercionSignal(ctx sdk.Context, signalId string) (*types.CoercionSignal, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(coercionSignalKey(signalId))
	if err != nil || bz == nil {
		return nil, false
	}
	var cs types.CoercionSignal
	if err := proto.Unmarshal(bz, &cs); err != nil {
		return nil, false
	}
	return &cs, true
}

// DeleteCoercionSignal removes a coercion signal.
func (k Keeper) DeleteCoercionSignal(ctx sdk.Context, signalId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(coercionSignalKey(signalId))
}

// GetAllCoercionSignals returns all coercion signals.
func (k Keeper) GetAllCoercionSignals(ctx sdk.Context) []*types.CoercionSignal {
	var signals []*types.CoercionSignal
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.CoercionSignalKeyPrefix, prefixEndBytes(types.CoercionSignalKeyPrefix))
	if err != nil {
		return signals
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var cs types.CoercionSignal
		if err := proto.Unmarshal(iter.Value(), &cs); err != nil {
			continue
		}
		signals = append(signals, &cs)
	}
	return signals
}

// ---------- RejectionCooldown CRUD ----------

// SetRejectionCooldown stores a rejection cooldown.
func (k Keeper) SetRejectionCooldown(ctx sdk.Context, rc *types.RejectionCooldown) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(rc)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal rejection cooldown: %v", err))
	}
	_ = kvStore.Set(rejectionCooldownKey(rc.PartnershipId), bz)
}

// GetRejectionCooldown retrieves a rejection cooldown by partnership ID.
func (k Keeper) GetRejectionCooldown(ctx sdk.Context, partnershipId string) (*types.RejectionCooldown, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(rejectionCooldownKey(partnershipId))
	if err != nil || bz == nil {
		return nil, false
	}
	var rc types.RejectionCooldown
	if err := proto.Unmarshal(bz, &rc); err != nil {
		return nil, false
	}
	return &rc, true
}

// DeleteRejectionCooldown removes a rejection cooldown.
func (k Keeper) DeleteRejectionCooldown(ctx sdk.Context, partnershipId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(rejectionCooldownKey(partnershipId))
}

// ---------- Sequence Counter ----------

// NextSequence returns the next auto-incrementing sequence number.
func (k Keeper) NextSequence(ctx sdk.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.SequenceKey)
	if err != nil || bz == nil {
		// Start at 1
		seq := uint64(1)
		seqBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(seqBytes, seq+1)
		_ = kvStore.Set(types.SequenceKey, seqBytes)
		return seq
	}
	seq := binary.BigEndian.Uint64(bz)
	nextBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nextBytes, seq+1)
	_ = kvStore.Set(types.SequenceKey, nextBytes)
	return seq
}

// ---------- Key Construction Helpers ----------

func partnershipKey(id string) []byte {
	return append(types.PartnershipKeyPrefix, []byte(id)...)
}

func byHumanPrefix(humanAddr string) []byte {
	return append(types.ByHumanIndexPrefix, []byte(humanAddr+"/")...)
}

func byHumanKey(humanAddr, partnershipId string) []byte {
	return append(types.ByHumanIndexPrefix, []byte(humanAddr+"/"+partnershipId)...)
}

func byAgentPrefix(agentAddr string) []byte {
	return append(types.ByAgentIndexPrefix, []byte(agentAddr+"/")...)
}

func byAgentKey(agentAddr, partnershipId string) []byte {
	return append(types.ByAgentIndexPrefix, []byte(agentAddr+"/"+partnershipId)...)
}

func consensusOpKey(id string) []byte {
	return append(types.ConsensusOpKeyPrefix, []byte(id)...)
}

func safetyFreezeKey(partnershipId string) []byte {
	return append(types.SafetyFreezeKeyPrefix, []byte(partnershipId)...)
}

func coercionSignalKey(signalId string) []byte {
	return append(types.CoercionSignalKeyPrefix, []byte(signalId)...)
}

func rejectionCooldownKey(partnershipId string) []byte {
	return append(types.RejectionCooldownKeyPrefix, []byte(partnershipId)...)
}
