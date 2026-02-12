package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

// ---------- Evidence KV Operations ----------

// SetEvidence stores an evidence record in the KV store.
func (k Keeper) SetEvidence(ctx sdk.Context, evidence types.Evidence) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := evidenceKey(evidence.Id)
	bz, err := json.Marshal(evidence)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal evidence: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set evidence: %v", err))
	}

	// Maintain claim→evidence index
	indexKey := evidenceByClaimKey(evidence.ClaimId, evidence.Id)
	if err := kvStore.Set(indexKey, []byte(evidence.Id)); err != nil {
		panic(fmt.Sprintf("failed to set evidence index: %v", err))
	}
}

// GetEvidence retrieves an evidence record by ID.
func (k Keeper) GetEvidence(ctx sdk.Context, id string) (types.Evidence, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(evidenceKey(id))
	if err != nil || bz == nil {
		return types.Evidence{}, false
	}
	var evidence types.Evidence
	if err := json.Unmarshal(bz, &evidence); err != nil {
		return types.Evidence{}, false
	}
	return evidence, true
}

// DeleteEvidence removes an evidence record from the store.
func (k Keeper) DeleteEvidence(ctx sdk.Context, id string) {
	evidence, found := k.GetEvidence(ctx, id)
	if !found {
		return
	}
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(evidenceKey(id))
	_ = kvStore.Delete(evidenceByClaimKey(evidence.ClaimId, id))
}

// IterateEvidence iterates over all evidence. Return true from cb to stop.
func (k Keeper) IterateEvidence(ctx sdk.Context, cb func(types.Evidence) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.EvidenceKeyPrefix, prefixEndBytes(types.EvidenceKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var evidence types.Evidence
		if err := json.Unmarshal(iter.Value(), &evidence); err != nil {
			continue
		}
		if cb(evidence) {
			break
		}
	}
}

// GetEvidenceByClaimId returns all evidence for a given claim.
func (k Keeper) GetEvidenceByClaimId(ctx sdk.Context, claimId string) []types.Evidence {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(types.EvidenceByClaimPrefix, []byte(claimId+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var evidences []types.Evidence
	for ; iter.Valid(); iter.Next() {
		evidenceId := string(iter.Value())
		evidence, found := k.GetEvidence(ctx, evidenceId)
		if found {
			evidences = append(evidences, evidence)
		}
	}
	return evidences
}

// GetEvidenceCountForClaim returns the number of evidence records for a claim.
func (k Keeper) GetEvidenceCountForClaim(ctx sdk.Context, claimId string) uint32 {
	return uint32(len(k.GetEvidenceByClaimId(ctx, claimId)))
}

// ---------- Audit KV Operations ----------

// SetAudit stores an audit record in the KV store.
func (k Keeper) SetAudit(ctx sdk.Context, audit types.EvidenceAudit) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := auditKey(audit.Id)
	bz, err := json.Marshal(audit)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal audit: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set audit: %v", err))
	}

	// Maintain evidence→audit index
	indexKey := auditByEvidenceKey(audit.EvidenceId, audit.Id)
	if err := kvStore.Set(indexKey, []byte(audit.Id)); err != nil {
		panic(fmt.Sprintf("failed to set audit index: %v", err))
	}
}

// GetAudit retrieves an audit record by ID.
func (k Keeper) GetAudit(ctx sdk.Context, id string) (types.EvidenceAudit, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(auditKey(id))
	if err != nil || bz == nil {
		return types.EvidenceAudit{}, false
	}
	var audit types.EvidenceAudit
	if err := json.Unmarshal(bz, &audit); err != nil {
		return types.EvidenceAudit{}, false
	}
	return audit, true
}

// DeleteAudit removes an audit record from the store.
func (k Keeper) DeleteAudit(ctx sdk.Context, id string) {
	audit, found := k.GetAudit(ctx, id)
	if !found {
		return
	}
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(auditKey(id))
	_ = kvStore.Delete(auditByEvidenceKey(audit.EvidenceId, id))
}

// IterateAudits iterates over all audits. Return true from cb to stop.
func (k Keeper) IterateAudits(ctx sdk.Context, cb func(types.EvidenceAudit) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.AuditKeyPrefix, prefixEndBytes(types.AuditKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var audit types.EvidenceAudit
		if err := json.Unmarshal(iter.Value(), &audit); err != nil {
			continue
		}
		if cb(audit) {
			break
		}
	}
}

// GetAuditsForEvidence returns all audits for a given evidence ID.
func (k Keeper) GetAuditsForEvidence(ctx sdk.Context, evidenceId string) []types.EvidenceAudit {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(types.AuditByEvidencePrefix, []byte(evidenceId+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var audits []types.EvidenceAudit
	for ; iter.Valid(); iter.Next() {
		auditId := string(iter.Value())
		audit, found := k.GetAudit(ctx, auditId)
		if found {
			audits = append(audits, audit)
		}
	}
	return audits
}

// ---------- Consensus KV Operations ----------

// IterateConsensus iterates over all audit consensus records. Return true from cb to stop.
func (k Keeper) IterateConsensus(ctx sdk.Context, cb func(types.AuditConsensus) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ConsensusKeyPrefix, prefixEndBytes(types.ConsensusKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var consensus types.AuditConsensus
		if err := json.Unmarshal(iter.Value(), &consensus); err != nil {
			continue
		}
		if cb(consensus) {
			break
		}
	}
}

// SetConsensus stores an audit consensus record.
func (k Keeper) SetConsensus(ctx sdk.Context, consensus types.AuditConsensus) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := consensusKey(consensus.EvidenceId)
	bz, err := json.Marshal(consensus)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal consensus: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set consensus: %v", err))
	}
}

// GetConsensus retrieves the audit consensus for an evidence ID.
func (k Keeper) GetConsensus(ctx sdk.Context, evidenceId string) (types.AuditConsensus, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(consensusKey(evidenceId))
	if err != nil || bz == nil {
		return types.AuditConsensus{}, false
	}
	var consensus types.AuditConsensus
	if err := json.Unmarshal(bz, &consensus); err != nil {
		return types.AuditConsensus{}, false
	}
	return consensus, true
}

// ---------- Expiry Index Operations ----------

// SetExpiryIndex writes a secondary index entry for time-based evidence expiry scanning.
func (k Keeper) SetExpiryIndex(ctx sdk.Context, expiryHeight uint64, evidenceId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := expiryIndexKey(expiryHeight, evidenceId)
	if err := kvStore.Set(key, []byte(evidenceId)); err != nil {
		panic(fmt.Sprintf("failed to set expiry index: %v", err))
	}
}

// RemoveExpiryIndex removes the expiry index entry for an evidence record.
func (k Keeper) RemoveExpiryIndex(ctx sdk.Context, expiryHeight uint64, evidenceId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(expiryIndexKey(expiryHeight, evidenceId))
}

// IterateExpiredEvidence iterates over evidence IDs whose expiry height <= currentHeight.
// Because keys are big-endian encoded, this scans in ascending height order and stops early.
func (k Keeper) IterateExpiredEvidence(ctx sdk.Context, currentHeight uint64, cb func(evidenceId string) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	// Scan from prefix start up to (currentHeight+1) exclusive
	endHeight := currentHeight + 1
	endKey := make([]byte, 8)
	binary.BigEndian.PutUint64(endKey, endHeight)
	end := append(types.ExpiryIndexPrefix, endKey...)

	iter, err := kvStore.Iterator(types.ExpiryIndexPrefix, end)
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		evidenceId := string(iter.Value())
		if cb(evidenceId) {
			break
		}
	}
}

// ---------- Oracle KV Operations ----------

// SetOracle stores an oracle record in the KV store.
func (k Keeper) SetOracle(ctx sdk.Context, oracle types.Oracle) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := oracleKey(oracle.Id)
	bz, err := json.Marshal(oracle)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal oracle: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set oracle: %v", err))
	}
}

// GetOracle retrieves an oracle by ID.
func (k Keeper) GetOracle(ctx sdk.Context, id string) (types.Oracle, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(oracleKey(id))
	if err != nil || bz == nil {
		return types.Oracle{}, false
	}
	var oracle types.Oracle
	if err := json.Unmarshal(bz, &oracle); err != nil {
		return types.Oracle{}, false
	}
	return oracle, true
}

// DeleteOracle removes an oracle from the store.
func (k Keeper) DeleteOracle(ctx sdk.Context, id string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(oracleKey(id))
	_ = kvStore.Delete(activeOracleKey(id))
}

// IterateOracles iterates over all oracles. Return true from cb to stop.
func (k Keeper) IterateOracles(ctx sdk.Context, cb func(types.Oracle) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.OracleKeyPrefix, prefixEndBytes(types.OracleKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var oracle types.Oracle
		if err := json.Unmarshal(iter.Value(), &oracle); err != nil {
			continue
		}
		if cb(oracle) {
			break
		}
	}
}

// SetActiveOracle marks an oracle as active in the active oracle index.
func (k Keeper) SetActiveOracle(ctx sdk.Context, oracleId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	if err := kvStore.Set(activeOracleKey(oracleId), []byte(oracleId)); err != nil {
		panic(fmt.Sprintf("failed to set active oracle: %v", err))
	}
}

// RemoveActiveOracle removes an oracle from the active index.
func (k Keeper) RemoveActiveOracle(ctx sdk.Context, oracleId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(activeOracleKey(oracleId))
}

// GetActiveOracles returns all active oracles.
func (k Keeper) GetActiveOracles(ctx sdk.Context) []types.Oracle {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ActiveOraclePrefix, prefixEndBytes(types.ActiveOraclePrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var oracles []types.Oracle
	for ; iter.Valid(); iter.Next() {
		oracleId := string(iter.Value())
		oracle, found := k.GetOracle(ctx, oracleId)
		if found && oracle.IsActive {
			oracles = append(oracles, oracle)
		}
	}
	return oracles
}

// GetOracleByOperator finds an oracle by its operator address.
func (k Keeper) GetOracleByOperator(ctx sdk.Context, operator string) (types.Oracle, bool) {
	var found types.Oracle
	var exists bool
	k.IterateOracles(ctx, func(o types.Oracle) bool {
		if o.Operator == operator {
			found = o
			exists = true
			return true
		}
		return false
	})
	return found, exists
}

// ---------- ID Generation ----------

// GenerateEvidenceID generates a deterministic evidence ID.
func GenerateEvidenceID(submitter, claimId, contentHash string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.evidence.id.v1:"))
	h.Write([]byte(submitter))
	h.Write([]byte(":"))
	h.Write([]byte(claimId))
	h.Write([]byte(":"))
	h.Write([]byte(contentHash))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}

// GenerateAuditID generates a deterministic audit ID.
func GenerateAuditID(auditor, evidenceId string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.audit.id.v1:"))
	h.Write([]byte(auditor))
	h.Write([]byte(":"))
	h.Write([]byte(evidenceId))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}

// GenerateOracleID generates a deterministic oracle ID.
func GenerateOracleID(operator, name string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.oracle.id.v1:"))
	h.Write([]byte(operator))
	h.Write([]byte(":"))
	h.Write([]byte(name))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}

// ---------- Key Construction Helpers ----------

func evidenceKey(id string) []byte {
	return append(types.EvidenceKeyPrefix, []byte(id)...)
}

func auditKey(id string) []byte {
	return append(types.AuditKeyPrefix, []byte(id)...)
}

func oracleKey(id string) []byte {
	return append(types.OracleKeyPrefix, []byte(id)...)
}

func consensusKey(evidenceId string) []byte {
	return append(types.ConsensusKeyPrefix, []byte(evidenceId)...)
}

func evidenceByClaimKey(claimId, evidenceId string) []byte {
	return append(types.EvidenceByClaimPrefix, []byte(claimId+":"+evidenceId)...)
}

func auditByEvidenceKey(evidenceId, auditId string) []byte {
	return append(types.AuditByEvidencePrefix, []byte(evidenceId+":"+auditId)...)
}

func activeOracleKey(oracleId string) []byte {
	return append(types.ActiveOraclePrefix, []byte(oracleId)...)
}

func expiryIndexKey(expiryHeight uint64, evidenceId string) []byte {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, expiryHeight)
	key := append(types.ExpiryIndexPrefix, heightBytes...)
	return append(key, []byte(evidenceId)...)
}
