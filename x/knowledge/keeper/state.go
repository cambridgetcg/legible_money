package keeper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// ---------- Fact Operations ----------

// SetFact stores a fact in the KV store.
func (k Keeper) SetFact(ctx sdk.Context, fact types.Fact) {
	store := ctx.KVStore(k.storeKey)
	key := factKey(fact.Id)
	bz, err := json.Marshal(fact)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal fact: %v", err))
	}
	store.Set(key, bz)

	// Maintain domain→fact index
	indexKey := domainFactIndexKey(fact.Domain, fact.Id)
	store.Set(indexKey, []byte(fact.Id))
}

// GetFact retrieves a fact by ID.
func (k Keeper) GetFact(ctx sdk.Context, factID string) (types.Fact, bool) {
	store := ctx.KVStore(k.storeKey)
	key := factKey(factID)
	bz := store.Get(key)
	if bz == nil {
		return types.Fact{}, false
	}
	var fact types.Fact
	if err := json.Unmarshal(bz, &fact); err != nil {
		return types.Fact{}, false
	}
	return fact, true
}

// DeleteFact removes a fact from the store.
func (k Keeper) DeleteFact(ctx sdk.Context, factID string) {
	fact, found := k.GetFact(ctx, factID)
	if !found {
		return
	}
	store := ctx.KVStore(k.storeKey)
	store.Delete(factKey(factID))
	store.Delete(domainFactIndexKey(fact.Domain, factID))
}

// IterateFacts iterates over all facts. Return true from cb to stop.
func (k Keeper) IterateFacts(ctx sdk.Context, cb func(types.Fact) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.FactKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var fact types.Fact
		if err := json.Unmarshal(iter.Value(), &fact); err != nil {
			continue
		}
		if cb(fact) {
			break
		}
	}
}

// GetFactsByDomain returns all facts in a domain.
func (k Keeper) GetFactsByDomain(ctx sdk.Context, domain string) []types.Fact {
	store := ctx.KVStore(k.storeKey)
	prefix := append(types.DomainFactIndexPrefix, []byte(domain+":")...)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var facts []types.Fact
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
	k.IterateFacts(ctx, func(_ types.Fact) bool {
		count++
		return false
	})
	return count
}

// ---------- Claim Operations ----------

// SetClaim stores a claim.
func (k Keeper) SetClaim(ctx sdk.Context, claim types.Claim) {
	store := ctx.KVStore(k.storeKey)
	key := claimKey(claim.Id)
	bz, err := json.Marshal(claim)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal claim: %v", err))
	}
	store.Set(key, bz)
}

// GetClaim retrieves a claim by ID.
func (k Keeper) GetClaim(ctx sdk.Context, claimID string) (types.Claim, bool) {
	store := ctx.KVStore(k.storeKey)
	key := claimKey(claimID)
	bz := store.Get(key)
	if bz == nil {
		return types.Claim{}, false
	}
	var claim types.Claim
	if err := json.Unmarshal(bz, &claim); err != nil {
		return types.Claim{}, false
	}
	return claim, true
}

// GetClaimByContentHash finds a claim by its content hash (for duplicate detection).
func (k Keeper) GetClaimByContentHash(ctx sdk.Context, contentHash string) (types.Claim, bool) {
	var found types.Claim
	var exists bool
	k.IterateClaims(ctx, func(c types.Claim) bool {
		if c.ContentHash == contentHash {
			found = c
			exists = true
			return true
		}
		return false
	})
	return found, exists
}

// DeleteClaim removes a claim.
func (k Keeper) DeleteClaim(ctx sdk.Context, claimID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(claimKey(claimID))
}

// IterateClaims iterates over all claims. Return true from cb to stop.
func (k Keeper) IterateClaims(ctx sdk.Context, cb func(types.Claim) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ClaimKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var claim types.Claim
		if err := json.Unmarshal(iter.Value(), &claim); err != nil {
			continue
		}
		if cb(claim) {
			break
		}
	}
}

// GetPendingClaims returns claims awaiting verification.
func (k Keeper) GetPendingClaims(ctx sdk.Context) []types.Claim {
	var claims []types.Claim
	k.IterateClaims(ctx, func(c types.Claim) bool {
		if c.Status == "pending" {
			claims = append(claims, c)
		}
		return false
	})
	return claims
}

// ---------- Verification Round Operations ----------

// SetVerificationRound stores a verification round.
func (k Keeper) SetVerificationRound(ctx sdk.Context, round types.VerificationRound) {
	store := ctx.KVStore(k.storeKey)
	key := roundKey(round.Id)
	bz, err := json.Marshal(round)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal round: %v", err))
	}
	store.Set(key, bz)
}

// GetVerificationRound retrieves a round by ID.
func (k Keeper) GetVerificationRound(ctx sdk.Context, roundID string) (types.VerificationRound, bool) {
	store := ctx.KVStore(k.storeKey)
	key := roundKey(roundID)
	bz := store.Get(key)
	if bz == nil {
		return types.VerificationRound{}, false
	}
	var round types.VerificationRound
	if err := json.Unmarshal(bz, &round); err != nil {
		return types.VerificationRound{}, false
	}
	return round, true
}

// GetVerificationRoundForClaim finds the active round for a claim.
func (k Keeper) GetVerificationRoundForClaim(ctx sdk.Context, claimID string) (types.VerificationRound, bool) {
	var found types.VerificationRound
	var exists bool
	k.IterateVerificationRounds(ctx, func(r types.VerificationRound) bool {
		if r.ClaimId == claimID && r.Phase != "complete" {
			found = r
			exists = true
			return true
		}
		return false
	})
	return found, exists
}

// DeleteVerificationRound removes a round.
func (k Keeper) DeleteVerificationRound(ctx sdk.Context, roundID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(roundKey(roundID))
}

// IterateVerificationRounds iterates over all rounds.
func (k Keeper) IterateVerificationRounds(ctx sdk.Context, cb func(types.VerificationRound) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.VerificationRoundKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var round types.VerificationRound
		if err := json.Unmarshal(iter.Value(), &round); err != nil {
			continue
		}
		if cb(round) {
			break
		}
	}
}

// GetActiveRounds returns all non-complete rounds.
func (k Keeper) GetActiveRounds(ctx sdk.Context) []types.VerificationRound {
	var rounds []types.VerificationRound
	k.IterateVerificationRounds(ctx, func(r types.VerificationRound) bool {
		if r.Phase != "complete" {
			rounds = append(rounds, r)
		}
		return false
	})
	return rounds
}

// GetActiveRoundsForValidator returns rounds where a validator is selected.
func (k Keeper) GetActiveRoundsForValidator(ctx sdk.Context, validatorAddr string) []types.VerificationRound {
	var rounds []types.VerificationRound
	k.IterateVerificationRounds(ctx, func(r types.VerificationRound) bool {
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

// GetExpiredRounds returns rounds that have passed their deadlines.
func (k Keeper) GetExpiredRounds(ctx sdk.Context, currentHeight uint64) []types.VerificationRound {
	var rounds []types.VerificationRound
	k.IterateVerificationRounds(ctx, func(r types.VerificationRound) bool {
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
func (k Keeper) SetFactReference(ctx sdk.Context, ref types.FactReference) {
	store := ctx.KVStore(k.storeKey)
	key := factRefKey(ref.FromFactId, ref.ToFactId)
	bz, err := json.Marshal(ref)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal fact reference: %v", err))
	}
	store.Set(key, bz)

	// Maintain incoming reference index (toFact → fromFact)
	incomingKey := incomingRefKey(ref.ToFactId, ref.FromFactId)
	store.Set(incomingKey, bz)
}

// GetFactReferences returns all outgoing references from a fact.
func (k Keeper) GetFactReferences(ctx sdk.Context, fromFactID string) []types.FactReference {
	store := ctx.KVStore(k.storeKey)
	prefix := append(types.FactReferenceKeyPrefix, []byte(fromFactID+":")...)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var refs []types.FactReference
	for ; iter.Valid(); iter.Next() {
		var ref types.FactReference
		if err := json.Unmarshal(iter.Value(), &ref); err != nil {
			continue
		}
		refs = append(refs, ref)
	}
	return refs
}

// GetIncomingReferences returns all references pointing TO a fact.
func (k Keeper) GetIncomingReferences(ctx sdk.Context, toFactID string) []types.FactReference {
	store := ctx.KVStore(k.storeKey)
	prefix := append(types.IncomingRefIndexPrefix, []byte(toFactID+":")...)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var refs []types.FactReference
	for ; iter.Valid(); iter.Next() {
		var ref types.FactReference
		if err := json.Unmarshal(iter.Value(), &ref); err != nil {
			continue
		}
		refs = append(refs, ref)
	}
	return refs
}

// GetSupportingFacts returns facts that support the given fact.
func (k Keeper) GetSupportingFacts(ctx sdk.Context, factID string) []types.FactReference {
	incoming := k.GetIncomingReferences(ctx, factID)
	var supporting []types.FactReference
	for _, ref := range incoming {
		if ref.ReferenceType == "supports" {
			supporting = append(supporting, ref)
		}
	}
	return supporting
}

// GetContradictingFacts returns facts that contradict the given fact.
func (k Keeper) GetContradictingFacts(ctx sdk.Context, factID string) []types.FactReference {
	incoming := k.GetIncomingReferences(ctx, factID)
	var contradicting []types.FactReference
	for _, ref := range incoming {
		if ref.ReferenceType == "contradicts" {
			contradicting = append(contradicting, ref)
		}
	}
	return contradicting
}

// ---------- Domain Operations ----------

// SetDomain stores an ontological domain (replacing the stub).
func (k Keeper) SetDomain(ctx sdk.Context, domain types.OntologicalDomain) {
	store := ctx.KVStore(k.storeKey)
	key := domainKey(domain.Name)
	bz, err := json.Marshal(domain)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal domain: %v", err))
	}
	store.Set(key, bz)
}

// GetDomain retrieves a domain by name.
func (k Keeper) GetDomain(ctx sdk.Context, name string) (types.OntologicalDomain, bool) {
	store := ctx.KVStore(k.storeKey)
	key := domainKey(name)
	bz := store.Get(key)
	if bz == nil {
		return types.OntologicalDomain{}, false
	}
	var domain types.OntologicalDomain
	if err := json.Unmarshal(bz, &domain); err != nil {
		return types.OntologicalDomain{}, false
	}
	return domain, true
}

// IterateDomains iterates over all domains.
func (k Keeper) IterateDomains(ctx sdk.Context, cb func(types.OntologicalDomain) bool) {
	store := ctx.KVStore(k.storeKey)
	// Domain keys use a separate prefix (0x07)
	prefix := []byte{0x07}
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var domain types.OntologicalDomain
		if err := json.Unmarshal(iter.Value(), &domain); err != nil {
			continue
		}
		if cb(domain) {
			break
		}
	}
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

// GenerateClaimID generates a deterministic claim ID.
func GenerateClaimID(submitter, contentHash string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.claim.id.v1:"))
	h.Write([]byte(submitter))
	h.Write([]byte(":"))
	h.Write([]byte(contentHash))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}

// GenerateFactID generates a deterministic fact ID from a verified claim.
func GenerateFactID(claimID, roundID string) string {
	h := sha256.New()
	h.Write([]byte("LGM.fact.id.v1:"))
	h.Write([]byte(claimID))
	h.Write([]byte(":"))
	h.Write([]byte(roundID))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
}

// GenerateRoundID generates a deterministic round ID.
func GenerateRoundID(claimID string, height uint64) string {
	h := sha256.New()
	h.Write([]byte("LGM.round.id.v1:"))
	h.Write([]byte(claimID))
	h.Write([]byte(fmt.Sprintf(":%d", height)))
	return fmt.Sprintf("%x", h.Sum(nil))[:32]
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
