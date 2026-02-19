package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// Secondary index key prefixes for query performance (B8-6).
// These extend the knowledge module's key space beyond 0x1e.
var (
	idxClaimStatus  = []byte{0x1f} // status:claimId → sentinel
	idxActiveRound  = []byte{0x20} // roundId → sentinel
	idxClaimDomain  = []byte{0x21} // domain:claimId → sentinel
	idxClaimSubmit  = []byte{0x22} // submitter:claimId → sentinel
	idxClaimCat     = []byte{0x23} // category:claimId → sentinel
	idxFactSubmit   = []byte{0x24} // submitter:factId → sentinel
	idxFactCat      = []byte{0x25} // category:factId → sentinel
	idxFactStatus   = []byte{0x26} // status:factId → sentinel
	idxSentinel     = []byte{1}
)

// ---------- Secondary Index Key Construction ----------

func claimStatusIndexKey(status, claimId string) []byte {
	return append(idxClaimStatus, []byte(status+":"+claimId)...)
}

func activeRoundIndexKey(roundId string) []byte {
	return append(idxActiveRound, []byte(roundId)...)
}

func claimDomainIndexKey(domain, claimId string) []byte {
	return append(idxClaimDomain, []byte(domain+":"+claimId)...)
}

func claimSubmitterIndexKey(submitter, claimId string) []byte {
	return append(idxClaimSubmit, []byte(submitter+":"+claimId)...)
}

func claimCategoryIndexKey(category, claimId string) []byte {
	return append(idxClaimCat, []byte(category+":"+claimId)...)
}

func factSubmitterIndexKey(submitter, factId string) []byte {
	return append(idxFactSubmit, []byte(submitter+":"+factId)...)
}

func factCategoryIndexKey(category, factId string) []byte {
	return append(idxFactCat, []byte(category+":"+factId)...)
}

func factStatusIndexKey(status, factId string) []byte {
	return append(idxFactStatus, []byte(status+":"+factId)...)
}

// ---------- Index Maintenance (called from Set/Delete in state.go) ----------

// maintainClaimIndexes updates secondary indexes when a claim is set.
// oldClaim may be nil if this is a new claim.
func (k Keeper) maintainClaimIndexes(ctx sdk.Context, claim *types.Claim, oldClaim *types.Claim) {
	kvStore := k.storeService.OpenKVStore(ctx)

	// Remove stale index entries if fields changed
	if oldClaim != nil {
		if oldClaim.Status != claim.Status {
			_ = kvStore.Delete(claimStatusIndexKey(oldClaim.Status, oldClaim.Id))
		}
		if oldClaim.Domain != claim.Domain {
			_ = kvStore.Delete(claimDomainIndexKey(oldClaim.Domain, oldClaim.Id))
		}
		if oldClaim.Submitter != claim.Submitter {
			_ = kvStore.Delete(claimSubmitterIndexKey(oldClaim.Submitter, oldClaim.Id))
		}
		if oldClaim.EpistemicCategory != claim.EpistemicCategory {
			_ = kvStore.Delete(claimCategoryIndexKey(oldClaim.EpistemicCategory, oldClaim.Id))
		}
	}

	// Write current index entries
	if claim.Status != "" {
		_ = kvStore.Set(claimStatusIndexKey(claim.Status, claim.Id), idxSentinel)
	}
	if claim.Domain != "" {
		_ = kvStore.Set(claimDomainIndexKey(claim.Domain, claim.Id), idxSentinel)
	}
	if claim.Submitter != "" {
		_ = kvStore.Set(claimSubmitterIndexKey(claim.Submitter, claim.Id), idxSentinel)
	}
	if claim.EpistemicCategory != "" {
		_ = kvStore.Set(claimCategoryIndexKey(claim.EpistemicCategory, claim.Id), idxSentinel)
	}
}

// removeClaimIndexes removes all secondary indexes for a claim.
func (k Keeper) removeClaimIndexes(ctx sdk.Context, claim *types.Claim) {
	kvStore := k.storeService.OpenKVStore(ctx)
	if claim.Status != "" {
		_ = kvStore.Delete(claimStatusIndexKey(claim.Status, claim.Id))
	}
	if claim.Domain != "" {
		_ = kvStore.Delete(claimDomainIndexKey(claim.Domain, claim.Id))
	}
	if claim.Submitter != "" {
		_ = kvStore.Delete(claimSubmitterIndexKey(claim.Submitter, claim.Id))
	}
	if claim.EpistemicCategory != "" {
		_ = kvStore.Delete(claimCategoryIndexKey(claim.EpistemicCategory, claim.Id))
	}
}

// maintainFactIndexes updates secondary indexes when a fact is set.
func (k Keeper) maintainFactIndexes(ctx sdk.Context, fact *types.Fact, oldFact *types.Fact) {
	kvStore := k.storeService.OpenKVStore(ctx)

	if oldFact != nil {
		if oldFact.Submitter != fact.Submitter {
			_ = kvStore.Delete(factSubmitterIndexKey(oldFact.Submitter, oldFact.Id))
		}
		if oldFact.EpistemicCategory != fact.EpistemicCategory {
			_ = kvStore.Delete(factCategoryIndexKey(oldFact.EpistemicCategory, oldFact.Id))
		}
		if oldFact.Status != fact.Status {
			_ = kvStore.Delete(factStatusIndexKey(oldFact.Status, oldFact.Id))
		}
	}

	if fact.Submitter != "" {
		_ = kvStore.Set(factSubmitterIndexKey(fact.Submitter, fact.Id), idxSentinel)
	}
	if fact.EpistemicCategory != "" {
		_ = kvStore.Set(factCategoryIndexKey(fact.EpistemicCategory, fact.Id), idxSentinel)
	}
	if fact.Status != "" {
		_ = kvStore.Set(factStatusIndexKey(fact.Status, fact.Id), idxSentinel)
	}
}

// removeFactIndexes removes all secondary indexes for a fact.
func (k Keeper) removeFactIndexes(ctx sdk.Context, fact *types.Fact) {
	kvStore := k.storeService.OpenKVStore(ctx)
	if fact.Submitter != "" {
		_ = kvStore.Delete(factSubmitterIndexKey(fact.Submitter, fact.Id))
	}
	if fact.EpistemicCategory != "" {
		_ = kvStore.Delete(factCategoryIndexKey(fact.EpistemicCategory, fact.Id))
	}
	if fact.Status != "" {
		_ = kvStore.Delete(factStatusIndexKey(fact.Status, fact.Id))
	}
}

// maintainActiveRoundIndex adds or removes a round from the active round index.
func (k Keeper) maintainActiveRoundIndex(ctx sdk.Context, round *types.VerificationRound) {
	kvStore := k.storeService.OpenKVStore(ctx)
	if round.Phase != "complete" {
		_ = kvStore.Set(activeRoundIndexKey(round.Id), idxSentinel)
	} else {
		_ = kvStore.Delete(activeRoundIndexKey(round.Id))
	}
}

// ---------- Indexed Query Methods ----------

// GetClaimsByStatus returns claims with the given status using the secondary index.
func (k Keeper) GetClaimsByStatus(ctx sdk.Context, status string) []*types.Claim {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(idxClaimStatus, []byte(status+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var claims []*types.Claim
	for ; iter.Valid(); iter.Next() {
		claimId := string(iter.Key()[len(prefix):])
		claim, found := k.GetClaim(ctx, claimId)
		if found {
			claims = append(claims, claim)
		}
	}
	return claims
}

// GetClaimsByDomainIndexed returns claims in a domain using the secondary index.
func (k Keeper) GetClaimsByDomainIndexed(ctx sdk.Context, domain string) []*types.Claim {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(idxClaimDomain, []byte(domain+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var claims []*types.Claim
	for ; iter.Valid(); iter.Next() {
		claimId := string(iter.Key()[len(prefix):])
		claim, found := k.GetClaim(ctx, claimId)
		if found {
			claims = append(claims, claim)
		}
	}
	return claims
}

// GetClaimsBySubmitter returns claims from a submitter using the secondary index.
func (k Keeper) GetClaimsBySubmitter(ctx sdk.Context, submitter string) []*types.Claim {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(idxClaimSubmit, []byte(submitter+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var claims []*types.Claim
	for ; iter.Valid(); iter.Next() {
		claimId := string(iter.Key()[len(prefix):])
		claim, found := k.GetClaim(ctx, claimId)
		if found {
			claims = append(claims, claim)
		}
	}
	return claims
}

// GetClaimsByCategory returns claims by epistemic category using the secondary index.
func (k Keeper) GetClaimsByCategory(ctx sdk.Context, category string) []*types.Claim {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(idxClaimCat, []byte(category+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var claims []*types.Claim
	for ; iter.Valid(); iter.Next() {
		claimId := string(iter.Key()[len(prefix):])
		claim, found := k.GetClaim(ctx, claimId)
		if found {
			claims = append(claims, claim)
		}
	}
	return claims
}

// GetActiveRoundsIndexed returns all non-complete rounds using the active round index.
func (k Keeper) GetActiveRoundsIndexed(ctx sdk.Context) []*types.VerificationRound {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(idxActiveRound, prefixEndBytes(idxActiveRound))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var rounds []*types.VerificationRound
	for ; iter.Valid(); iter.Next() {
		roundId := string(iter.Key()[len(idxActiveRound):])
		round, found := k.GetVerificationRound(ctx, roundId)
		if found {
			rounds = append(rounds, round)
		}
	}
	return rounds
}

// GetFactsBySubmitter returns facts from a submitter using the secondary index.
func (k Keeper) GetFactsBySubmitter(ctx sdk.Context, submitter string) []*types.Fact {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(idxFactSubmit, []byte(submitter+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var facts []*types.Fact
	for ; iter.Valid(); iter.Next() {
		factId := string(iter.Key()[len(prefix):])
		fact, found := k.GetFact(ctx, factId)
		if found {
			facts = append(facts, fact)
		}
	}
	return facts
}

// GetFactsByCategory returns facts by epistemic category using the secondary index.
func (k Keeper) GetFactsByCategory(ctx sdk.Context, category string) []*types.Fact {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(idxFactCat, []byte(category+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var facts []*types.Fact
	for ; iter.Valid(); iter.Next() {
		factId := string(iter.Key()[len(prefix):])
		fact, found := k.GetFact(ctx, factId)
		if found {
			facts = append(facts, fact)
		}
	}
	return facts
}

// GetFactsByStatusIndexed returns facts with the given status using the secondary index.
func (k Keeper) GetFactsByStatusIndexed(ctx sdk.Context, status string) []*types.Fact {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := append(idxFactStatus, []byte(status+":")...)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var facts []*types.Fact
	for ; iter.Valid(); iter.Next() {
		factId := string(iter.Key()[len(prefix):])
		fact, found := k.GetFact(ctx, factId)
		if found {
			facts = append(facts, fact)
		}
	}
	return facts
}

// ---------- Migration / Genesis Index Population ----------

// PopulateSecondaryIndexes builds all secondary indexes from primary data.
// Call this during InitGenesis or state migration.
func (k Keeper) PopulateSecondaryIndexes(ctx sdk.Context) {
	// Index all claims
	k.IterateClaims(ctx, func(claim *types.Claim) bool {
		k.maintainClaimIndexes(ctx, claim, nil)
		return false
	})

	// Index all facts
	k.IterateFacts(ctx, func(fact *types.Fact) bool {
		k.maintainFactIndexes(ctx, fact, nil)
		return false
	})

	// Index all active verification rounds
	k.IterateVerificationRounds(ctx, func(round *types.VerificationRound) bool {
		k.maintainActiveRoundIndex(ctx, round)
		return false
	})
}

// PopulatePendingEvalClaimIndex builds the PendingEvalClaimIndex for claims
// in pending_evaluation status. Uses the existing prefix 0x0e declared in keys.go
// but never wired.
func (k Keeper) PopulatePendingEvalClaimIndex(ctx sdk.Context) {
	kvStore := k.storeService.OpenKVStore(ctx)
	k.IterateClaims(ctx, func(claim *types.Claim) bool {
		if claim.Status == types.ClaimStatusPendingEvaluation {
			key := pendingEvalIndexKey(uint64(ctx.BlockHeight()), claim.Id)
			_ = kvStore.Set(key, idxSentinel)
		}
		return false
	})
}

func pendingEvalIndexKey(height uint64, claimId string) []byte {
	hBytes := fmt.Sprintf("%020d", height)
	return append(types.PendingEvalClaimIndexPrefix, []byte(hBytes+":"+claimId)...)
}
