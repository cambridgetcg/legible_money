package keeper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/knowledge/types"
)

// deterministicMarshal serializes a protobuf message with deterministic output.
// This is CRITICAL for consensus: proto.Marshal uses random map iteration order
// for map fields (e.g. VerificationRound.Commitments, .Reveals), causing
// different validators to produce different byte sequences for the same state.
// Using Deterministic: true ensures map entries are sorted by key.
var deterministicMarshal = proto.MarshalOptions{Deterministic: true}.Marshal

// ---------- Fact Operations ----------

// SetFact stores a fact in the KV store.
func (k Keeper) SetFact(ctx sdk.Context, fact *types.Fact) {
	// Read old fact for index delta detection
	oldFact, _ := k.GetFact(ctx, fact.Id)

	kvStore := k.storeService.OpenKVStore(ctx)
	key := factKey(fact.Id)
	bz, err := deterministicMarshal(fact)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal fact: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set fact: %v", err))
	}

	// Maintain domain→fact index
	indexKey := domainFactIndexKey(fact.Domain, fact.Id)
	if err := kvStore.Set(indexKey, []byte(fact.Id)); err != nil {
		panic(fmt.Sprintf("failed to set domain fact index: %v", err))
	}

	// Maintain secondary indexes (B8-6)
	k.maintainFactIndexes(ctx, fact, oldFact)
}

// GetFact retrieves a fact by ID.
func (k Keeper) GetFact(ctx sdk.Context, factID string) (*types.Fact, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := factKey(factID)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var fact types.Fact
	if err := proto.Unmarshal(bz, &fact); err != nil {
		return nil, false
	}
	return &fact, true
}

// DeleteFact removes a fact from the store.
func (k Keeper) DeleteFact(ctx sdk.Context, factID string) {
	fact, found := k.GetFact(ctx, factID)
	if !found {
		return
	}
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(factKey(factID))
	_ = kvStore.Delete(domainFactIndexKey(fact.Domain, factID))

	// Remove secondary indexes (B8-6)
	k.removeFactIndexes(ctx, fact)
}

// IterateFacts iterates over all facts. Return true from cb to stop.
func (k Keeper) IterateFacts(ctx sdk.Context, cb func(*types.Fact) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.FactKeyPrefix, prefixEndBytes(types.FactKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var fact types.Fact
		if err := proto.Unmarshal(iter.Value(), &fact); err != nil {
			continue
		}
		if cb(&fact) {
			break
		}
	}
}

// GetFactsByDomain returns all facts in a domain.
func (k Keeper) GetFactsByDomain(ctx sdk.Context, domain string) []*types.Fact {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(types.DomainFactIndexPrefix, []byte(domain+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var facts []*types.Fact
	for ; iter.Valid(); iter.Next() {
		factID := string(iter.Value())
		fact, found := k.GetFact(ctx, factID)
		if found {
			facts = append(facts, fact)
		}
	}
	return facts
}

// GetFactCount returns the total number of facts.
func (k Keeper) GetFactCount(ctx sdk.Context) uint64 {
	var count uint64
	k.IterateFacts(ctx, func(_ *types.Fact) bool {
		count++
		return false
	})
	return count
}

// ---------- Claim Operations ----------

// SetClaim stores a claim and maintains the content hash index.
func (k Keeper) SetClaim(ctx sdk.Context, claim *types.Claim) {
	// Read old claim for index delta detection
	oldClaim, _ := k.GetClaim(ctx, claim.Id)

	kvStore := k.storeService.OpenKVStore(ctx)
	key := claimKey(claim.Id)
	bz, err := deterministicMarshal(claim)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal claim: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set claim: %v", err))
	}

	// Maintain contentHash → claimId index for O(1) duplicate detection
	if claim.ContentHash != "" {
		if err := kvStore.Set(contentHashIndexKey(claim.ContentHash), []byte(claim.Id)); err != nil {
			panic(fmt.Sprintf("failed to set content hash index: %v", err))
		}
	}

	// Maintain secondary indexes (B8-6)
	k.maintainClaimIndexes(ctx, claim, oldClaim)
}

// GetClaim retrieves a claim by ID.
func (k Keeper) GetClaim(ctx sdk.Context, claimID string) (*types.Claim, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := claimKey(claimID)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var claim types.Claim
	if err := proto.Unmarshal(bz, &claim); err != nil {
		return nil, false
	}
	return &claim, true
}

// GetClaimByContentHash finds a claim by its content hash using an O(1) index lookup.
func (k Keeper) GetClaimByContentHash(ctx sdk.Context, contentHash string) (*types.Claim, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	claimIDBytes, err := kvStore.Get(contentHashIndexKey(contentHash))
	if err != nil || claimIDBytes == nil {
		return nil, false
	}
	return k.GetClaim(ctx, string(claimIDBytes))
}

// DeleteClaim removes a claim and its indexes.
func (k Keeper) DeleteClaim(ctx sdk.Context, claimID string) {
	claim, found := k.GetClaim(ctx, claimID)
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(claimKey(claimID))
	if found {
		if claim.ContentHash != "" {
			_ = kvStore.Delete(contentHashIndexKey(claim.ContentHash))
		}
		// Remove secondary indexes (B8-6)
		k.removeClaimIndexes(ctx, claim)
	}
}

// IterateClaims iterates over all claims. Return true from cb to stop.
func (k Keeper) IterateClaims(ctx sdk.Context, cb func(*types.Claim) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ClaimKeyPrefix, prefixEndBytes(types.ClaimKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var claim types.Claim
		if err := proto.Unmarshal(iter.Value(), &claim); err != nil {
			continue
		}
		if cb(&claim) {
			break
		}
	}
}

// GetPendingClaims returns claims awaiting verification using the status index.
func (k Keeper) GetPendingClaims(ctx sdk.Context) []*types.Claim {
	return k.GetClaimsByStatus(ctx, "pending")
}

// ---------- Verification Round Operations ----------

// SetVerificationRound stores a verification round and maintains the claim→round index.
func (k Keeper) SetVerificationRound(ctx sdk.Context, round *types.VerificationRound) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := roundKey(round.Id)
	bz, err := deterministicMarshal(round)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal round: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set round: %v", err))
	}

	// Maintain claimId → roundId index for active (non-complete) rounds
	if round.ClaimId != "" {
		if round.Phase != "complete" {
			if err := kvStore.Set(claimRoundIndexKey(round.ClaimId), []byte(round.Id)); err != nil {
				panic(fmt.Sprintf("failed to set claim round index: %v", err))
			}
		} else {
			// Clean up index when round completes
			_ = kvStore.Delete(claimRoundIndexKey(round.ClaimId))
		}
	}

	// Maintain active round index (B8-6)
	k.maintainActiveRoundIndex(ctx, round)
}

// GetVerificationRound retrieves a round by ID.
func (k Keeper) GetVerificationRound(ctx sdk.Context, roundID string) (*types.VerificationRound, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := roundKey(roundID)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var round types.VerificationRound
	if err := proto.Unmarshal(bz, &round); err != nil {
		return nil, false
	}
	// Proto maps may be nil after unmarshal; initialize to avoid nil map panics
	if round.Commitments == nil {
		round.Commitments = make(map[string]*types.VerificationCommitment)
	}
	if round.Reveals == nil {
		round.Reveals = make(map[string]*types.VerificationReveal)
	}
	return &round, true
}

// GetVerificationRoundForClaim finds the active round for a claim using an O(1) index lookup.
func (k Keeper) GetVerificationRoundForClaim(ctx sdk.Context, claimID string) (*types.VerificationRound, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	roundIDBytes, err := kvStore.Get(claimRoundIndexKey(claimID))
	if err != nil || roundIDBytes == nil {
		return nil, false
	}
	return k.GetVerificationRound(ctx, string(roundIDBytes))
}

// DeleteVerificationRound removes a round and its indexes.
func (k Keeper) DeleteVerificationRound(ctx sdk.Context, roundID string) {
	round, found := k.GetVerificationRound(ctx, roundID)
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(roundKey(roundID))
	if found {
		if round.ClaimId != "" {
			_ = kvStore.Delete(claimRoundIndexKey(round.ClaimId))
		}
		// Remove active round index entry (B8-6)
		_ = kvStore.Delete(activeRoundIndexKey(round.Id))
	}
}

// IterateVerificationRounds iterates over all rounds.
func (k Keeper) IterateVerificationRounds(ctx sdk.Context, cb func(*types.VerificationRound) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.VerificationRoundKeyPrefix, prefixEndBytes(types.VerificationRoundKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var round types.VerificationRound
		if err := proto.Unmarshal(iter.Value(), &round); err != nil {
			continue
		}
		if cb(&round) {
			break
		}
	}
}

// GetActiveRounds returns all non-complete rounds using the active round index.
func (k Keeper) GetActiveRounds(ctx sdk.Context) []*types.VerificationRound {
	return k.GetActiveRoundsIndexed(ctx)
}

// GetActiveRoundsForValidator returns rounds where a validator is selected.
func (k Keeper) GetActiveRoundsForValidator(ctx sdk.Context, validatorAddr string) []*types.VerificationRound {
	var rounds []*types.VerificationRound
	k.IterateVerificationRounds(ctx, func(r *types.VerificationRound) bool {
		if r.Phase == "complete" {
			return false
		}
		for _, v := range r.SelectedValidators {
			if v == validatorAddr {
				rounds = append(rounds, r)
				break
			}
		}
		return false
	})
	return rounds
}

// HasActiveRoundForClaim checks if any non-complete round exists for the given claim.
// Uses the O(1) ClaimRoundIndex (0x0a) instead of full-scan iteration.
func (k Keeper) HasActiveRoundForClaim(ctx sdk.Context, claimId string) bool {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(claimRoundIndexKey(claimId))
	return err == nil && bz != nil
}

// GetExpiredRounds returns rounds that have passed their deadlines.
func (k Keeper) GetExpiredRounds(ctx sdk.Context, currentHeight uint64) []*types.VerificationRound {
	var rounds []*types.VerificationRound
	k.IterateVerificationRounds(ctx, func(r *types.VerificationRound) bool {
		switch r.Phase {
		case "commit":
			if currentHeight > r.CommitDeadlineHeight {
				rounds = append(rounds, r)
			}
		case "reveal":
			if currentHeight > r.RevealDeadlineHeight {
				rounds = append(rounds, r)
			}
		}
		return false
	})
	return rounds
}

// ---------- Fact Reference Operations ----------

// SetFactReference stores a fact-to-fact reference with bidirectional indexes.
func (k Keeper) SetFactReference(ctx sdk.Context, ref *types.FactReference) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := factRefKey(ref.FromFactId, ref.ToFactId)
	bz, err := deterministicMarshal(ref)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal fact reference: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set fact reference: %v", err))
	}

	// Maintain incoming reference index (toFact → fromFact)
	incomingKey := incomingRefKey(ref.ToFactId, ref.FromFactId)
	if err := kvStore.Set(incomingKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set incoming ref index: %v", err))
	}
}

// GetFactReferences returns all outgoing references from a fact.
func (k Keeper) GetFactReferences(ctx sdk.Context, fromFactID string) []*types.FactReference {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(types.FactReferenceKeyPrefix, []byte(fromFactID+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var refs []*types.FactReference
	for ; iter.Valid(); iter.Next() {
		var ref types.FactReference
		if err := proto.Unmarshal(iter.Value(), &ref); err != nil {
			continue
		}
		refs = append(refs, &ref)
	}
	return refs
}

// GetIncomingReferences returns all references pointing TO a fact.
func (k Keeper) GetIncomingReferences(ctx sdk.Context, toFactID string) []*types.FactReference {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(types.IncomingRefIndexPrefix, []byte(toFactID+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var refs []*types.FactReference
	for ; iter.Valid(); iter.Next() {
		var ref types.FactReference
		if err := proto.Unmarshal(iter.Value(), &ref); err != nil {
			continue
		}
		refs = append(refs, &ref)
	}
	return refs
}

// GetSupportingFacts returns facts that support the given fact.
func (k Keeper) GetSupportingFacts(ctx sdk.Context, factID string) []*types.FactReference {
	incoming := k.GetIncomingReferences(ctx, factID)
	var supporting []*types.FactReference
	for _, ref := range incoming {
		if ref.ReferenceType == "supports" {
			supporting = append(supporting, ref)
		}
	}
	return supporting
}

// GetContradictingFacts returns facts that contradict the given fact.
func (k Keeper) GetContradictingFacts(ctx sdk.Context, factID string) []*types.FactReference {
	incoming := k.GetIncomingReferences(ctx, factID)
	var contradicting []*types.FactReference
	for _, ref := range incoming {
		if ref.ReferenceType == "contradicts" {
			contradicting = append(contradicting, ref)
		}
	}
	return contradicting
}

// ---------- Domain Operations ----------

// SetDomain stores an ontological domain (replacing the stub).
func (k Keeper) SetDomain(ctx sdk.Context, domain *types.OntologicalDomain) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := domainKey(domain.Name)
	bz, err := deterministicMarshal(domain)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal domain: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set domain: %v", err))
	}
}

// GetDomain retrieves a domain by name.
func (k Keeper) GetDomain(ctx sdk.Context, name string) (*types.OntologicalDomain, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := domainKey(name)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var domain types.OntologicalDomain
	if err := proto.Unmarshal(bz, &domain); err != nil {
		return nil, false
	}
	return &domain, true
}

// IterateDomains iterates over all domains.
func (k Keeper) IterateDomains(ctx sdk.Context, cb func(*types.OntologicalDomain) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	// Domain keys use a separate prefix (0x07)
	prefix := []byte{0x07}
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var domain types.OntologicalDomain
		if err := proto.Unmarshal(iter.Value(), &domain); err != nil {
			continue
		}
		if cb(&domain) {
			break
		}
	}
}

// ---------- Equivocation Evidence Operations ----------

// SetEquivocationEvidence stores proof of a validator's equivocation.
func (k Keeper) SetEquivocationEvidence(ctx sdk.Context, evidence types.EquivocationEvidence) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := equivocationKey(evidence.RoundId, evidence.Validator)
	bz, err := json.Marshal(evidence)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal equivocation evidence: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set equivocation evidence: %v", err))
	}
}

// GetEquivocationEvidence retrieves equivocation evidence for a round+validator.
func (k Keeper) GetEquivocationEvidence(ctx sdk.Context, roundID, validator string) (types.EquivocationEvidence, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := equivocationKey(roundID, validator)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return types.EquivocationEvidence{}, false
	}
	var evidence types.EquivocationEvidence
	if err := json.Unmarshal(bz, &evidence); err != nil {
		return types.EquivocationEvidence{}, false
	}
	return evidence, true
}

// ---------- Submitter Calibration Operations ----------

// SetCalibrationRecord stores a submitter's calibration record.
func (k Keeper) SetCalibrationRecord(ctx sdk.Context, record types.CalibrationRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := calibrationKey(record.Submitter)
	bz, err := json.Marshal(record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal calibration record: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set calibration record: %v", err))
	}
}

// GetCalibrationRecord retrieves a submitter's calibration record.
func (k Keeper) GetCalibrationRecord(ctx sdk.Context, submitter string) (types.CalibrationRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := calibrationKey(submitter)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return types.CalibrationRecord{}, false
	}
	var record types.CalibrationRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return types.CalibrationRecord{}, false
	}
	return record, true
}

// UpdateSubmitterCalibration records a claim outcome for a submitter.
// accepted=true increments AcceptedClaims, false increments RejectedClaims.
func (k Keeper) UpdateSubmitterCalibration(ctx sdk.Context, submitter string, accepted bool) {
	record, found := k.GetCalibrationRecord(ctx, submitter)
	if !found {
		record = types.CalibrationRecord{
			Submitter: submitter,
		}
	}

	record.TotalClaims++
	if accepted {
		record.AcceptedClaims++
	} else {
		record.RejectedClaims++
	}
	record.LastUpdateHeight = uint64(ctx.BlockHeight())

	k.SetCalibrationRecord(ctx, record)
}

// ---------- Content Hash Generation ----------

// ComputeClaimContentHash generates a deterministic content hash for a claim.
func ComputeClaimContentHash(claimText, domain, epistemicCategory string) string {
	h := sha256.New()
	h.Write([]byte("LGM.claim.content.v1:"))
	h.Write([]byte(claimText))
	h.Write([]byte(":"))
	h.Write([]byte(domain))
	h.Write([]byte(":"))
	h.Write([]byte(epistemicCategory))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ---------- Deterministic ID Generation ----------
//
// All knowledge-layer IDs are generated as SHA-256 hashes with domain separation:
//   ID = hex(SHA-256("LGM.<type>.id.v1:" || field1 || ":" || field2 || ... ))
//
// Properties:
//   - Deterministic: same inputs always produce the same ID
//   - Collision-resistant: SHA-256 over unique domain tags
//   - Verifiable: any node can recompute an ID from its constituent fields
//   - 64 hex chars (256-bit) for claims, facts, and rounds
//   - "cfact-" + 64 hex chars for counter-facts (prefix distinguishes negative knowledge)
//
// ID types and their inputs:
//   ClaimID:       submitter + contentHash + height       → hex(SHA-256(...))
//   FactID:        claimID + roundID                      → hex(SHA-256(...))
//   RoundID:       claimID + height                       → hex(SHA-256(...))
//   CounterFactID: targetFactId + targetClaimId + submitter + height → "cfact-" + hex(SHA-256(...))
//
// The height parameter ensures uniqueness when the same submitter resubmits
// after rejection (different block → different ID).

// GenerateClaimID generates a deterministic claim ID (full 256-bit SHA-256, 64 hex chars).
func GenerateClaimID(submitter, contentHash string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.claim.id.v1:"))
	h.Write([]byte(submitter))
	h.Write([]byte(":"))
	h.Write([]byte(contentHash))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GenerateFactID generates a deterministic fact ID (full 256-bit SHA-256, 64 hex chars).
func GenerateFactID(claimID, roundID string) string {
	h := sha256.New()
	h.Write([]byte("LGM.fact.id.v1:"))
	h.Write([]byte(claimID))
	h.Write([]byte(":"))
	h.Write([]byte(roundID))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GenerateRoundID generates a deterministic round ID (full 256-bit SHA-256, 64 hex chars).
func GenerateRoundID(claimID string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.round.id.v1:"))
	h.Write([]byte(claimID))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ---------- Key Construction Helpers ----------

func factKey(id string) []byte {
	return append(types.FactKeyPrefix, []byte(id)...)
}

func claimKey(id string) []byte {
	return append(types.ClaimKeyPrefix, []byte(id)...)
}

func roundKey(id string) []byte {
	return append(types.VerificationRoundKeyPrefix, []byte(id)...)
}

func factRefKey(fromID, toID string) []byte {
	return append(types.FactReferenceKeyPrefix, []byte(fromID+":"+toID)...)
}

func domainFactIndexKey(domain, factID string) []byte {
	return append(types.DomainFactIndexPrefix, []byte(domain+":"+factID)...)
}

func domainKey(name string) []byte {
	return append([]byte{0x07}, []byte(name)...)
}

func incomingRefKey(toID, fromID string) []byte {
	return append(types.IncomingRefIndexPrefix, []byte(toID+":"+fromID)...)
}

func contentHashIndexKey(contentHash string) []byte {
	return append(types.ContentHashIndexPrefix, []byte(contentHash)...)
}

func claimRoundIndexKey(claimID string) []byte {
	return append(types.ClaimRoundIndexPrefix, []byte(claimID)...)
}

func equivocationKey(roundID, validator string) []byte {
	return append(types.EquivocationKeyPrefix, []byte(roundID+":"+validator)...)
}

func calibrationKey(submitter string) []byte {
	return append(types.SubmitterCalibrationPrefix, []byte(submitter)...)
}

// ---------- Counter-Fact Operations (Negative Knowledge) ----------

func counterFactKey(id string) []byte {
	return append(types.CounterFactKeyPrefix, []byte(id)...)
}

func counterFactByFactKey(targetFactId, counterFactId string) []byte {
	return append(types.CounterFactByFactIndexPrefix, []byte(targetFactId+":"+counterFactId)...)
}

func counterFactByDomainKey(domain, counterFactId string) []byte {
	return append(types.CounterFactByDomainIndexPrefix, []byte(domain+":"+counterFactId)...)
}

func counterFactByHeightKey(height uint64, counterFactId string) []byte {
	// Big-endian height for natural ordering
	hBytes := fmt.Sprintf("%020d", height) // zero-padded for lexicographic order
	return append(types.CounterFactByHeightIndexPrefix, []byte(hBytes+":"+counterFactId)...)
}

func contradictionCooldownKey(submitter string, epoch uint64) []byte {
	return append(types.ContradictionCooldownPrefix, []byte(fmt.Sprintf("%s:%d", submitter, epoch))...)
}

// GenerateCounterFactID generates a deterministic counter-fact ID.
func GenerateCounterFactID(targetFactId, targetClaimId, submitter string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.counterfact.id.v1:"))
	h.Write([]byte(targetFactId))
	h.Write([]byte(":"))
	h.Write([]byte(targetClaimId))
	h.Write([]byte(":"))
	h.Write([]byte(submitter))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("cfact-%x", h.Sum(nil))
}

// SetCounterFact stores a counter-fact and maintains all indexes.
func (k Keeper) SetCounterFact(ctx sdk.Context, cf *types.CounterFact) {
	kvStore := k.storeService.OpenKVStore(ctx)

	// Primary storage
	bz, err := deterministicMarshal(cf)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal counter-fact: %v", err))
	}
	if err := kvStore.Set(counterFactKey(cf.Id), bz); err != nil {
		panic(fmt.Sprintf("failed to set counter-fact: %v", err))
	}

	// Index: targetFactId → counterFactId
	if cf.TargetFactId != "" {
		if err := kvStore.Set(counterFactByFactKey(cf.TargetFactId, cf.Id), []byte(cf.Id)); err != nil {
			panic(fmt.Sprintf("failed to set counter-fact-by-fact index: %v", err))
		}
	}

	// Index: domain → counterFactId
	if cf.Domain != "" {
		if err := kvStore.Set(counterFactByDomainKey(cf.Domain, cf.Id), []byte(cf.Id)); err != nil {
			panic(fmt.Sprintf("failed to set counter-fact-by-domain index: %v", err))
		}
	}

	// Index: height → counterFactId
	if err := kvStore.Set(counterFactByHeightKey(cf.CreatedAtHeight, cf.Id), []byte(cf.Id)); err != nil {
		panic(fmt.Sprintf("failed to set counter-fact-by-height index: %v", err))
	}
}

// GetCounterFact retrieves a counter-fact by ID.
func (k Keeper) GetCounterFact(ctx sdk.Context, id string) (*types.CounterFact, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(counterFactKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var cf types.CounterFact
	if err := proto.Unmarshal(bz, &cf); err != nil {
		return nil, false
	}
	return &cf, true
}

// DeleteCounterFact removes a counter-fact and all its indexes.
func (k Keeper) DeleteCounterFact(ctx sdk.Context, id string) {
	cf, found := k.GetCounterFact(ctx, id)
	if !found {
		return
	}
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(counterFactKey(id))
	if cf.TargetFactId != "" {
		_ = kvStore.Delete(counterFactByFactKey(cf.TargetFactId, id))
	}
	if cf.Domain != "" {
		_ = kvStore.Delete(counterFactByDomainKey(cf.Domain, id))
	}
	_ = kvStore.Delete(counterFactByHeightKey(cf.CreatedAtHeight, id))
}

// GetCounterFactsByFact returns all counter-facts targeting a specific fact.
func (k Keeper) GetCounterFactsByFact(ctx sdk.Context, factId string) []*types.CounterFact {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(types.CounterFactByFactIndexPrefix, []byte(factId+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var results []*types.CounterFact
	for ; iter.Valid(); iter.Next() {
		cfId := string(iter.Value())
		cf, found := k.GetCounterFact(ctx, cfId)
		if found {
			results = append(results, cf)
		}
	}
	return results
}

// GetCounterFactsByDomain returns counter-facts in a domain.
func (k Keeper) GetCounterFactsByDomain(ctx sdk.Context, domain string) []*types.CounterFact {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(types.CounterFactByDomainIndexPrefix, []byte(domain+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var results []*types.CounterFact
	for ; iter.Valid(); iter.Next() {
		cfId := string(iter.Value())
		cf, found := k.GetCounterFact(ctx, cfId)
		if found {
			results = append(results, cf)
		}
	}
	return results
}

// IterateCounterFacts iterates over all counter-facts. Return true from cb to stop.
func (k Keeper) IterateCounterFacts(ctx sdk.Context, cb func(*types.CounterFact) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.CounterFactKeyPrefix, prefixEndBytes(types.CounterFactKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var cf types.CounterFact
		if err := proto.Unmarshal(iter.Value(), &cf); err != nil {
			continue
		}
		if cb(&cf) {
			break
		}
	}
}

// GetMaxFalsityConfidence returns the highest counter-fact confidence for a fact.
// This is the "falsity confidence" component of bidirectional confidence.
func (k Keeper) GetMaxFalsityConfidence(ctx sdk.Context, factId string) uint64 {
	counterFacts := k.GetCounterFactsByFact(ctx, factId)
	var maxConf uint64
	for _, cf := range counterFacts {
		if cf.Status == types.CounterFactStatusVerified && cf.Confidence > maxConf {
			maxConf = cf.Confidence
		}
	}
	return maxConf
}

// GetBidirectionalConfidence computes the dual-track epistemic assessment for a fact.
func (k Keeper) GetBidirectionalConfidence(ctx sdk.Context, factId string) *types.BidirectionalConfidence {
	fact, found := k.GetFact(ctx, factId)
	if !found {
		return &types.BidirectionalConfidence{FactId: factId, Status: "uncertain"}
	}

	falsityConf := k.GetMaxFalsityConfidence(ctx, factId)
	counterFacts := k.GetCounterFactsByFact(ctx, factId)
	var verifiedCount uint32
	for _, cf := range counterFacts {
		if cf.Status == types.CounterFactStatusVerified {
			verifiedCount++
		}
	}

	net := int64(fact.Confidence) - int64(falsityConf)

	return &types.BidirectionalConfidence{
		FactId:            factId,
		TruthConfidence:   fact.Confidence,
		FalsityConfidence: falsityConf,
		NetConfidence:     net,
		CounterFactCount:  verifiedCount,
		Status:            types.ComputeBidirectionalStatus(net),
	}
}

// GetContradictionCount returns how many contradictions an agent submitted in an epoch.
func (k Keeper) GetContradictionCount(ctx sdk.Context, submitter string, epoch uint64) uint32 {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := contradictionCooldownKey(submitter, epoch)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return 0
	}
	var count uint32
	if err := json.Unmarshal(bz, &count); err != nil {
		return 0
	}
	return count
}

// IncrementContradictionCount increments the epoch contradiction counter.
func (k Keeper) IncrementContradictionCount(ctx sdk.Context, submitter string, epoch uint64) {
	count := k.GetContradictionCount(ctx, submitter, epoch)
	count++
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, _ := json.Marshal(count)
	_ = kvStore.Set(contradictionCooldownKey(submitter, epoch), bz)
}

// ---------- Falsification Epoch Cap ----------

func falsificationEpochPaidKey(epoch uint64) []byte {
	return append(types.FalsificationEpochPaidPrefix, []byte(fmt.Sprintf("%d", epoch))...)
}

// GetFalsificationEpochPaid returns total ulgm paid in falsification rewards this epoch.
func (k Keeper) GetFalsificationEpochPaid(ctx sdk.Context, epoch uint64) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(falsificationEpochPaidKey(epoch))
	if err != nil || bz == nil {
		return 0
	}
	var amount uint64
	if err := json.Unmarshal(bz, &amount); err != nil {
		return 0
	}
	return amount
}

// SetFalsificationEpochPaid records total ulgm paid in falsification rewards this epoch.
func (k Keeper) SetFalsificationEpochPaid(ctx sdk.Context, epoch uint64, amount uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, _ := json.Marshal(amount)
	_ = kvStore.Set(falsificationEpochPaidKey(epoch), bz)
}

// ---------- Falsification Carry-Forward (R-3) ----------

func falsificationCarryForwardKey(epoch uint64) []byte {
	return append(types.FalsificationCarryForwardPrefix, []byte(fmt.Sprintf("%d", epoch))...)
}

// GetFalsificationCarryForward returns ulgm carried forward from a prior epoch's cap overflow.
func (k Keeper) GetFalsificationCarryForward(ctx sdk.Context, epoch uint64) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(falsificationCarryForwardKey(epoch))
	if err != nil || bz == nil {
		return 0
	}
	var amount uint64
	if err := json.Unmarshal(bz, &amount); err != nil {
		return 0
	}
	return amount
}

// SetFalsificationCarryForward records ulgm to carry forward into a future epoch.
func (k Keeper) SetFalsificationCarryForward(ctx sdk.Context, epoch uint64, amount uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, _ := json.Marshal(amount)
	_ = kvStore.Set(falsificationCarryForwardKey(epoch), bz)
}

// AddFalsificationCarryForward atomically adds to an epoch's carry-forward balance.
func (k Keeper) AddFalsificationCarryForward(ctx sdk.Context, epoch uint64, amount uint64) {
	current := k.GetFalsificationCarryForward(ctx, epoch)
	k.SetFalsificationCarryForward(ctx, epoch, current+amount)
}

// ---------- Counter-Fact Challenge Window (R-2) ----------

func counterFactChallengeWindowKey(counterFactId string) []byte {
	return append(types.CounterFactChallengeWindowPrefix, []byte(counterFactId)...)
}

func counterFactChallengeKey(challengeId string) []byte {
	return append(types.CounterFactChallengeKeyPrefix, []byte(challengeId)...)
}

// GetCounterFactChallengeWindow returns the challenge window end height for a counter-fact.
func (k Keeper) GetCounterFactChallengeWindow(ctx sdk.Context, counterFactId string) (uint64, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(counterFactChallengeWindowKey(counterFactId))
	if err != nil || bz == nil {
		return 0, false
	}
	var endHeight uint64
	if err := json.Unmarshal(bz, &endHeight); err != nil {
		return 0, false
	}
	return endHeight, true
}

// SetCounterFactChallengeWindow stores the challenge window end height for a counter-fact.
func (k Keeper) SetCounterFactChallengeWindow(ctx sdk.Context, counterFactId string, endHeight uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, _ := json.Marshal(endHeight)
	_ = kvStore.Set(counterFactChallengeWindowKey(counterFactId), bz)
}

// SetCounterFactChallenge stores a counter-fact challenge.
func (k Keeper) SetCounterFactChallenge(ctx sdk.Context, challenge types.CounterFactChallenge) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(challenge)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal counter-fact challenge: %v", err))
	}
	_ = kvStore.Set(counterFactChallengeKey(challenge.Id), bz)
}

// GetCounterFactChallenge retrieves a counter-fact challenge by ID.
func (k Keeper) GetCounterFactChallenge(ctx sdk.Context, challengeId string) (types.CounterFactChallenge, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(counterFactChallengeKey(challengeId))
	if err != nil || bz == nil {
		return types.CounterFactChallenge{}, false
	}
	var challenge types.CounterFactChallenge
	if err := json.Unmarshal(bz, &challenge); err != nil {
		return types.CounterFactChallenge{}, false
	}
	return challenge, true
}

// ---------- Citation Source Record Operations (FARM-11) ----------

func citationSourceKey(factID string) []byte {
	return append(types.CitationSourcePrefix, []byte(factID)...)
}

// GetCitationSourceRecord retrieves citation source tracking for a fact.
func (k Keeper) GetCitationSourceRecord(ctx sdk.Context, factID string) (types.CitationSourceRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(citationSourceKey(factID))
	if err != nil || bz == nil {
		return types.CitationSourceRecord{}, false
	}
	var record types.CitationSourceRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return types.CitationSourceRecord{}, false
	}
	return record, true
}

// SetCitationSourceRecord stores citation source tracking for a fact.
func (k Keeper) SetCitationSourceRecord(ctx sdk.Context, record types.CitationSourceRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal citation source record: %v", err))
	}
	_ = kvStore.Set(citationSourceKey(record.FactID), bz)
}

// UpdateCitationSource increments the citation count for a citing address on a fact.
func (k Keeper) UpdateCitationSource(ctx sdk.Context, factID, citingAddr string) {
	record, found := k.GetCitationSourceRecord(ctx, factID)
	if !found {
		record = types.CitationSourceRecord{
			FactID:       factID,
			SourceCounts: make(map[string]uint64),
		}
	}
	record.SourceCounts[citingAddr]++
	record.TotalCites++
	k.SetCitationSourceRecord(ctx, record)
}

// ---------- Patronage Record Operations ----------

func patronageRecordKey(factId, patron string) []byte {
	return append(types.PatronageRecordPrefix, []byte(factId+":"+patron)...)
}

func patronageRecordByFactPrefix(factId string) []byte {
	return append(types.PatronageRecordPrefix, []byte(factId+":")...)
}

// SetPatronageRecord stores a patronage record.
func (k Keeper) SetPatronageRecord(ctx sdk.Context, record types.PatronageRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal patronage record: %v", err))
	}
	_ = kvStore.Set(patronageRecordKey(record.FactId, record.Patron), bz)
}

// GetPatronageRecord retrieves a patronage record.
func (k Keeper) GetPatronageRecord(ctx sdk.Context, factId, patron string) (types.PatronageRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(patronageRecordKey(factId, patron))
	if err != nil || bz == nil {
		return types.PatronageRecord{}, false
	}
	var record types.PatronageRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return types.PatronageRecord{}, false
	}
	return record, true
}

// GetPatronageRecordsByFact returns all patronage records for a fact.
func (k Keeper) GetPatronageRecordsByFact(ctx sdk.Context, factId string) []types.PatronageRecord {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := patronageRecordByFactPrefix(factId)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var records []types.PatronageRecord
	for ; iter.Valid(); iter.Next() {
		var record types.PatronageRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	return records
}

// ---------- Pruning Queue Operations ----------

func pruningQueueKey(graceEndBlock uint64, factId string) []byte {
	// Zero-padded for lexicographic ordering
	hBytes := fmt.Sprintf("%020d", graceEndBlock)
	return append(types.PruningQueuePrefix, []byte(hBytes+":"+factId)...)
}

// SetPruningQueueEntry adds a fact to the pruning queue indexed by grace end block.
func (k Keeper) SetPruningQueueEntry(ctx sdk.Context, graceEndBlock uint64, factId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Set(pruningQueueKey(graceEndBlock, factId), []byte{1})
}

// DeletePruningQueueEntry removes a fact from the pruning queue.
func (k Keeper) DeletePruningQueueEntry(ctx sdk.Context, graceEndBlock uint64, factId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(pruningQueueKey(graceEndBlock, factId))
}

// IterateCounterFactChallenges iterates over all counter-fact challenges.
func (k Keeper) IterateCounterFactChallenges(ctx sdk.Context, cb func(types.CounterFactChallenge) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.CounterFactChallengeKeyPrefix, prefixEndBytes(types.CounterFactChallengeKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var challenge types.CounterFactChallenge
		if err := json.Unmarshal(iter.Value(), &challenge); err != nil {
			continue
		}
		if cb(challenge) {
			break
		}
	}
}

// ---------- Domain Unique Contributors (FARM-7) ----------

// GetDomainUniqueContributors counts the number of unique fact submitters in a domain.
// Used to gate novelty bonuses — domains with fewer than MinDomainContributorsForNovelty
// unique contributors get proportionally reduced novelty bonuses.
func (k Keeper) GetDomainUniqueContributors(ctx sdk.Context, domain string) uint64 {
	facts := k.GetFactsByDomain(ctx, domain)
	submitters := make(map[string]struct{}, len(facts))
	for _, fact := range facts {
		if fact.Submitter != "" {
			submitters[fact.Submitter] = struct{}{}
		}
	}
	return uint64(len(submitters))
}

// GetFactCountByDomain returns the number of facts in a domain.
func (k Keeper) GetFactCountByDomain(ctx sdk.Context, domain string) uint64 {
	return uint64(len(k.GetFactsByDomain(ctx, domain)))
}

// GetDomainCount returns the total number of registered domains.
func (k Keeper) GetDomainCount(ctx sdk.Context) uint64 {
	var count uint64
	k.IterateDomains(ctx, func(_ *types.OntologicalDomain) bool {
		count++
		return false
	})
	return count
}

// ---------- Domain Stratum (FARM-12) ----------

func domainStratumKey(name string) []byte {
	return append(types.DomainStratumPrefix, []byte(name)...)
}

// SetDomainStratum stores the stratum assignment for a domain.
func (k Keeper) SetDomainStratum(ctx sdk.Context, name, stratum string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	if err := kvStore.Set(domainStratumKey(name), []byte(stratum)); err != nil {
		panic(fmt.Sprintf("failed to set domain stratum: %v", err))
	}
}

// GetDomainStratum retrieves the stratum assignment for a domain.
func (k Keeper) GetDomainStratum(ctx sdk.Context, name string) (string, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(domainStratumKey(name))
	if err != nil || bz == nil {
		return "", false
	}
	return string(bz), true
}

// ---------- Verifier Conformity (FARM-1) ----------

func verifierConformityKey(validatorAddr string) []byte {
	return append(types.VerifierConformityPrefix, []byte(validatorAddr)...)
}

// GetVerifierConformity retrieves a verifier's conformity tracking record.
func (k Keeper) GetVerifierConformity(ctx sdk.Context, validatorAddr string) (types.VerifierConformityRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(verifierConformityKey(validatorAddr))
	if err != nil || bz == nil {
		return types.VerifierConformityRecord{}, false
	}
	var record types.VerifierConformityRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return types.VerifierConformityRecord{}, false
	}
	return record, true
}

// SetVerifierConformity stores a verifier's conformity record.
func (k Keeper) SetVerifierConformity(ctx sdk.Context, record types.VerifierConformityRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal verifier conformity record: %v", err))
	}
	_ = kvStore.Set(verifierConformityKey(record.ValidatorAddr), bz)
}

// UpdateVerifierConformity records whether a validator voted with the round outcome.
func (k Keeper) UpdateVerifierConformity(ctx sdk.Context, validatorAddr string, votedWithOutcome bool) {
	record, found := k.GetVerifierConformity(ctx, validatorAddr)
	if !found {
		record = types.VerifierConformityRecord{
			ValidatorAddr: validatorAddr,
		}
	}
	record.TotalVotes++
	if votedWithOutcome {
		record.ConformingVotes++
	}
	record.LastUpdateHeight = uint64(ctx.BlockHeight())
	k.SetVerifierConformity(ctx, record)
}

// GetConformityPenaltyBPS returns the reward penalty for a rubber-stamp verifier.
// Returns 0 if the validator has insufficient samples or is below the conformity threshold.
func (k Keeper) GetConformityPenaltyBPS(ctx sdk.Context, validatorAddr string) uint64 {
	ep := k.GetExtendedParams(ctx)
	record, found := k.GetVerifierConformity(ctx, validatorAddr)
	if !found || record.TotalVotes < ep.ConformityMinSamples {
		return 0
	}
	// conformityRate = conformingVotes * 1_000_000 / totalVotes
	conformityRate := (record.ConformingVotes * 1_000_000) / record.TotalVotes
	if conformityRate <= ep.ConformityThresholdBps {
		return 0
	}
	// Linear penalty: scales from 0 at threshold to ConformityPenaltyMaxBps at 100%
	excess := conformityRate - ep.ConformityThresholdBps
	penaltyRange := uint64(1_000_000) - ep.ConformityThresholdBps
	if penaltyRange == 0 {
		return ep.ConformityPenaltyMaxBps
	}
	penalty := (excess * ep.ConformityPenaltyMaxBps) / penaltyRange
	return penalty
}

// ---------- Validator Participation (FARM-8) ----------

func validatorParticipationKey(validatorAddr string) []byte {
	return append(types.ValidatorParticipationPrefix, []byte(validatorAddr)...)
}

// GetValidatorParticipation retrieves a validator's participation tracking record.
func (k Keeper) GetValidatorParticipation(ctx sdk.Context, validatorAddr string) (types.ValidatorParticipationRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(validatorParticipationKey(validatorAddr))
	if err != nil || bz == nil {
		return types.ValidatorParticipationRecord{}, false
	}
	var record types.ValidatorParticipationRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return types.ValidatorParticipationRecord{}, false
	}
	return record, true
}

// SetValidatorParticipation stores a validator's participation record.
func (k Keeper) SetValidatorParticipation(ctx sdk.Context, record types.ValidatorParticipationRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal validator participation record: %v", err))
	}
	_ = kvStore.Set(validatorParticipationKey(record.ValidatorAddr), bz)
}

// IncrementRoundsAssigned increments the assigned round count for a validator.
func (k Keeper) IncrementRoundsAssigned(ctx sdk.Context, validatorAddr string) {
	record, found := k.GetValidatorParticipation(ctx, validatorAddr)
	if !found {
		record = types.ValidatorParticipationRecord{
			ValidatorAddr: validatorAddr,
		}
	}
	record.RoundsAssigned++
	record.LastUpdateHeight = uint64(ctx.BlockHeight())
	k.SetValidatorParticipation(ctx, record)
}

// IncrementRoundsParticipated increments the participated round count for a validator.
func (k Keeper) IncrementRoundsParticipated(ctx sdk.Context, validatorAddr string) {
	record, found := k.GetValidatorParticipation(ctx, validatorAddr)
	if !found {
		record = types.ValidatorParticipationRecord{
			ValidatorAddr: validatorAddr,
		}
	}
	record.RoundsParticipated++
	record.LastUpdateHeight = uint64(ctx.BlockHeight())
	k.SetValidatorParticipation(ctx, record)
}

// GetParticipationPenaltyBPS returns the reward penalty for a passive staking validator.
// Returns 0 if the validator has insufficient samples or meets the participation threshold.
func (k Keeper) GetParticipationPenaltyBPS(ctx sdk.Context, validatorAddr string) uint64 {
	ep := k.GetExtendedParams(ctx)
	record, found := k.GetValidatorParticipation(ctx, validatorAddr)
	if !found || record.RoundsAssigned < ep.ParticipationMinSamples {
		return 0
	}
	// participationRate = roundsParticipated * 1_000_000 / roundsAssigned
	participationRate := (record.RoundsParticipated * 1_000_000) / record.RoundsAssigned
	if participationRate >= ep.MinParticipationRateBps {
		return 0
	}
	// Below threshold: apply flat PassiveStakingPenaltyBps
	return ep.PassiveStakingPenaltyBps
}

// ---------- Research Fund Governance Operations ----------

func researchProposalKey(id string) []byte {
	return append(types.ResearchProposalPrefix, []byte(id)...)
}

func researchVoteKey(proposalId, voter string) []byte {
	return append(types.ResearchVotePrefix, []byte(proposalId+":"+voter)...)
}

func researchFundStatsKey() []byte {
	return types.ResearchFundStatsPrefix
}

// SetResearchProposal stores a research proposal (JSON-encoded).
func (k Keeper) SetResearchProposal(ctx sdk.Context, proposal *types.ResearchProposal) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(proposal)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal research proposal: %v", err))
	}
	if err := kvStore.Set(researchProposalKey(proposal.ProposalId), bz); err != nil {
		panic(fmt.Sprintf("failed to set research proposal: %v", err))
	}
}

// GetResearchProposal retrieves a research proposal by ID.
func (k Keeper) GetResearchProposal(ctx sdk.Context, proposalId string) (*types.ResearchProposal, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(researchProposalKey(proposalId))
	if err != nil || bz == nil {
		return nil, false
	}
	var proposal types.ResearchProposal
	if err := json.Unmarshal(bz, &proposal); err != nil {
		return nil, false
	}
	return &proposal, true
}

// IterateResearchProposals iterates over all research proposals. Return true from cb to stop.
func (k Keeper) IterateResearchProposals(ctx sdk.Context, cb func(*types.ResearchProposal) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ResearchProposalPrefix, prefixEndBytes(types.ResearchProposalPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var proposal types.ResearchProposal
		if err := json.Unmarshal(iter.Value(), &proposal); err != nil {
			continue
		}
		if cb(&proposal) {
			break
		}
	}
}

// SetResearchVote stores a vote on a research proposal (JSON-encoded).
func (k Keeper) SetResearchVote(ctx sdk.Context, vote *types.ResearchVote) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(vote)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal research vote: %v", err))
	}
	if err := kvStore.Set(researchVoteKey(vote.ProposalId, vote.Voter), bz); err != nil {
		panic(fmt.Sprintf("failed to set research vote: %v", err))
	}
}

// GetResearchVote retrieves a specific vote.
func (k Keeper) GetResearchVote(ctx sdk.Context, proposalId, voter string) (*types.ResearchVote, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(researchVoteKey(proposalId, voter))
	if err != nil || bz == nil {
		return nil, false
	}
	var vote types.ResearchVote
	if err := json.Unmarshal(bz, &vote); err != nil {
		return nil, false
	}
	return &vote, true
}

// GetVotesByProposal returns all votes for a given proposal.
func (k Keeper) GetVotesByProposal(ctx sdk.Context, proposalId string) []*types.ResearchVote {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(types.ResearchVotePrefix, []byte(proposalId+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var votes []*types.ResearchVote
	for ; iter.Valid(); iter.Next() {
		var vote types.ResearchVote
		if err := json.Unmarshal(iter.Value(), &vote); err != nil {
			continue
		}
		votes = append(votes, &vote)
	}
	return votes
}

// GetResearchFundStats retrieves research fund accounting stats.
func (k Keeper) GetResearchFundStats(ctx sdk.Context) types.ResearchFundStats {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(researchFundStatsKey())
	if err != nil || bz == nil {
		return types.ResearchFundStats{}
	}
	var stats types.ResearchFundStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return types.ResearchFundStats{}
	}
	return stats
}

// SetResearchFundStats stores research fund accounting stats.
func (k Keeper) SetResearchFundStats(ctx sdk.Context, stats types.ResearchFundStats) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(stats)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal research fund stats: %v", err))
	}
	if err := kvStore.Set(researchFundStatsKey(), bz); err != nil {
		panic(fmt.Sprintf("failed to set research fund stats: %v", err))
	}
}

// IncrementResearchFundCollected adds to the cumulative collected total.
func (k Keeper) IncrementResearchFundCollected(ctx sdk.Context, amount uint64) {
	stats := k.GetResearchFundStats(ctx)
	stats.TotalCollected += amount
	k.SetResearchFundStats(ctx, stats)
}
