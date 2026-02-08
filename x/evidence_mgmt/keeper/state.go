package keeper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

// ---------- Evidence KV Operations ----------

// SetEvidence stores an evidence record in the KV store.
func (k Keeper) SetEvidence(ctx sdk.Context, evidence types.Evidence) {
	store := ctx.KVStore(k.storeKey)
	key := evidenceKey(evidence.Id)
	bz, err := json.Marshal(evidence)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal evidence: %v", err))
	}
	store.Set(key, bz)

	// Maintain claim→evidence index
	indexKey := evidenceByClaimKey(evidence.ClaimId, evidence.Id)
	store.Set(indexKey, []byte(evidence.Id))
}

// GetEvidence retrieves an evidence record by ID.
func (k Keeper) GetEvidence(ctx sdk.Context, id string) (types.Evidence, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(evidenceKey(id))
	if bz == nil {
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
	store := ctx.KVStore(k.storeKey)
	store.Delete(evidenceKey(id))
	store.Delete(evidenceByClaimKey(evidence.ClaimId, id))
}

// IterateEvidence iterates over all evidence. Return true from cb to stop.
func (k Keeper) IterateEvidence(ctx sdk.Context, cb func(types.Evidence) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.EvidenceKeyPrefix)
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
	store := ctx.KVStore(k.storeKey)
	prefix := append(types.EvidenceByClaimPrefix, []byte(claimId+":")...)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
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
	store := ctx.KVStore(k.storeKey)
	key := auditKey(audit.Id)
	bz, err := json.Marshal(audit)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal audit: %v", err))
	}
	store.Set(key, bz)

	// Maintain evidence→audit index
	indexKey := auditByEvidenceKey(audit.EvidenceId, audit.Id)
	store.Set(indexKey, []byte(audit.Id))
}

// GetAudit retrieves an audit record by ID.
func (k Keeper) GetAudit(ctx sdk.Context, id string) (types.EvidenceAudit, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(auditKey(id))
	if bz == nil {
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
	store := ctx.KVStore(k.storeKey)
	store.Delete(auditKey(id))
	store.Delete(auditByEvidenceKey(audit.EvidenceId, id))
}

// IterateAudits iterates over all audits. Return true from cb to stop.
func (k Keeper) IterateAudits(ctx sdk.Context, cb func(types.EvidenceAudit) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.AuditKeyPrefix)
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
	store := ctx.KVStore(k.storeKey)
	prefix := append(types.AuditByEvidencePrefix, []byte(evidenceId+":")...)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
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

// SetConsensus stores an audit consensus record.
func (k Keeper) SetConsensus(ctx sdk.Context, consensus types.AuditConsensus) {
	store := ctx.KVStore(k.storeKey)
	key := consensusKey(consensus.EvidenceId)
	bz, err := json.Marshal(consensus)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal consensus: %v", err))
	}
	store.Set(key, bz)
}

// GetConsensus retrieves the audit consensus for an evidence ID.
func (k Keeper) GetConsensus(ctx sdk.Context, evidenceId string) (types.AuditConsensus, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(consensusKey(evidenceId))
	if bz == nil {
		return types.AuditConsensus{}, false
	}
	var consensus types.AuditConsensus
	if err := json.Unmarshal(bz, &consensus); err != nil {
		return types.AuditConsensus{}, false
	}
	return consensus, true
}

// ---------- Oracle KV Operations ----------

// SetOracle stores an oracle record in the KV store.
func (k Keeper) SetOracle(ctx sdk.Context, oracle types.Oracle) {
	store := ctx.KVStore(k.storeKey)
	key := oracleKey(oracle.Id)
	bz, err := json.Marshal(oracle)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal oracle: %v", err))
	}
	store.Set(key, bz)
}

// GetOracle retrieves an oracle by ID.
func (k Keeper) GetOracle(ctx sdk.Context, id string) (types.Oracle, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(oracleKey(id))
	if bz == nil {
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
	store := ctx.KVStore(k.storeKey)
	store.Delete(oracleKey(id))
	store.Delete(activeOracleKey(id))
}

// IterateOracles iterates over all oracles. Return true from cb to stop.
func (k Keeper) IterateOracles(ctx sdk.Context, cb func(types.Oracle) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.OracleKeyPrefix)
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
	store := ctx.KVStore(k.storeKey)
	store.Set(activeOracleKey(oracleId), []byte(oracleId))
}

// RemoveActiveOracle removes an oracle from the active index.
func (k Keeper) RemoveActiveOracle(ctx sdk.Context, oracleId string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(activeOracleKey(oracleId))
}

// GetActiveOracles returns all active oracles.
func (k Keeper) GetActiveOracles(ctx sdk.Context) []types.Oracle {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ActiveOraclePrefix)
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
