package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// --- Beliefs ---

// SetBelief stores or updates a belief for a home. If the home already has
// MaxBeliefsPerHome beliefs, the oldest (lowest CreatedAt) is evicted.
func (k Keeper) SetBelief(ctx sdk.Context, homeID, factID string, confidence uint64, source string) error {
	if homeID == "" || factID == "" {
		return fmt.Errorf("homeID and factID must be non-empty")
	}
	if _, found := k.GetHome(ctx, homeID); !found {
		return types.ErrHomeNotFound.Wrapf("home %s not found", homeID)
	}

	store := k.storeService.OpenKVStore(ctx)
	key := types.BeliefKey(homeID, factID)

	// Check if this is an update (no eviction needed for updates).
	existing, _ := store.Get(key)
	isNew := existing == nil

	if isNew {
		// Count current beliefs and evict oldest if at capacity.
		beliefs := k.ListBeliefs(ctx, homeID)
		if len(beliefs) >= types.MaxBeliefsPerHome {
			// Find oldest by CreatedAt, with FactID tiebreaker for determinism.
			oldest := beliefs[0]
			for _, b := range beliefs[1:] {
				if b.CreatedAt < oldest.CreatedAt || (b.CreatedAt == oldest.CreatedAt && b.FactID < oldest.FactID) {
					oldest = b
				}
			}
			if err := store.Delete(types.BeliefKey(homeID, oldest.FactID)); err != nil {
				return fmt.Errorf("failed to evict oldest belief: %w", err)
			}
		}
	}

	entry := types.BeliefEntry{
		HomeID:     homeID,
		FactID:     factID,
		Confidence: confidence,
		Source:     source,
		CreatedAt:  uint64(ctx.BlockHeight()),
	}

	bz, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal belief entry: %w", err)
	}
	return store.Set(key, bz)
}

// GetBelief returns the confidence, source, and existence flag for a belief.
func (k Keeper) GetBelief(ctx sdk.Context, homeID, factID string) (confidence uint64, source string, exists bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.BeliefKey(homeID, factID))
	if err != nil || bz == nil {
		return 0, "", false
	}
	var entry types.BeliefEntry
	if err := json.Unmarshal(bz, &entry); err != nil {
		return 0, "", false
	}
	return entry.Confidence, entry.Source, true
}

// ListBeliefs returns all belief entries for a home.
func (k Keeper) ListBeliefs(ctx sdk.Context, homeID string) []types.BeliefEntry {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.BeliefPrefixKey(homeID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var entries []types.BeliefEntry
	for ; iter.Valid(); iter.Next() {
		var entry types.BeliefEntry
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

// --- Relationships ---

// SetRelationship stores or updates a relationship between two homes.
// If the home already has MaxRelationshipsPerHome relationships, the oldest is evicted.
func (k Keeper) SetRelationship(ctx sdk.Context, homeID, targetID, relType string, strength uint64) error {
	if homeID == "" || targetID == "" {
		return fmt.Errorf("homeID and targetID must be non-empty")
	}
	if !types.ValidRelationshipTypes[relType] {
		return fmt.Errorf("invalid relationship type %q: must be ally|neutral|adversary|mentor|student", relType)
	}
	if strength > 100 {
		return fmt.Errorf("strength must be 0-100, got %d", strength)
	}
	if _, found := k.GetHome(ctx, homeID); !found {
		return types.ErrHomeNotFound.Wrapf("home %s not found", homeID)
	}

	store := k.storeService.OpenKVStore(ctx)
	key := types.RelationshipKey(homeID, targetID)

	existing, _ := store.Get(key)
	isNew := existing == nil

	if isNew {
		rels := k.ListRelationships(ctx, homeID)
		if len(rels) >= types.MaxRelationshipsPerHome {
			oldest := rels[0]
			for _, r := range rels[1:] {
				if r.UpdatedAt < oldest.UpdatedAt || (r.UpdatedAt == oldest.UpdatedAt && r.TargetID < oldest.TargetID) {
					oldest = r
				}
			}
			if err := store.Delete(types.RelationshipKey(homeID, oldest.TargetID)); err != nil {
				return fmt.Errorf("failed to evict oldest relationship: %w", err)
			}
		}
	}

	entry := types.RelationshipEntry{
		HomeID:    homeID,
		TargetID:  targetID,
		RelType:   relType,
		Strength:  strength,
		UpdatedAt: uint64(ctx.BlockHeight()),
	}

	bz, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal relationship entry: %w", err)
	}
	return store.Set(key, bz)
}

// GetRelationship returns the type, strength, and existence flag for a relationship.
func (k Keeper) GetRelationship(ctx sdk.Context, homeID, targetID string) (relType string, strength uint64, exists bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.RelationshipKey(homeID, targetID))
	if err != nil || bz == nil {
		return "", 0, false
	}
	var entry types.RelationshipEntry
	if err := json.Unmarshal(bz, &entry); err != nil {
		return "", 0, false
	}
	return entry.RelType, entry.Strength, true
}

// ListRelationships returns all relationship entries for a home.
func (k Keeper) ListRelationships(ctx sdk.Context, homeID string) []types.RelationshipEntry {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.RelationshipPrefixKey(homeID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var entries []types.RelationshipEntry
	for ; iter.Valid(); iter.Next() {
		var entry types.RelationshipEntry
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

// --- Milestones ---

// AddMilestone records a milestone for a home. If the home already has
// MaxMilestonesPerHome milestones, the oldest (lowest BlockHeight) is removed.
// Returns the new milestone ID.
func (k Keeper) AddMilestone(ctx sdk.Context, homeID, category, description string, blockHeight uint64) (string, error) {
	if homeID == "" || category == "" {
		return "", fmt.Errorf("homeID and category must be non-empty")
	}
	if _, found := k.GetHome(ctx, homeID); !found {
		return "", types.ErrHomeNotFound.Wrapf("home %s not found", homeID)
	}

	// Get next milestone ID via per-home counter.
	store := k.storeService.OpenKVStore(ctx)
	counterKey := types.MilestoneCounterKey(homeID)
	var counter uint64
	if bz, err := store.Get(counterKey); err == nil && bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	counterBz := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBz, counter)
	if err := store.Set(counterKey, counterBz); err != nil {
		return "", fmt.Errorf("failed to set milestone counter: %w", err)
	}

	milestoneID := fmt.Sprintf("ms-%d", counter)

	// Evict oldest if at capacity.
	milestones := k.ListMilestones(ctx, homeID)
	if len(milestones) >= types.MaxMilestonesPerHome {
		// ListMilestones returns newest first; oldest is last.
		oldest := milestones[len(milestones)-1]
		if err := store.Delete(types.MilestoneKey(homeID, oldest.ID)); err != nil {
			return "", fmt.Errorf("failed to evict oldest milestone: %w", err)
		}
	}

	entry := types.MilestoneEntry{
		HomeID:      homeID,
		ID:          milestoneID,
		Category:    category,
		Description: description,
		BlockHeight: blockHeight,
	}

	bz, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("failed to marshal milestone entry: %w", err)
	}
	if err := store.Set(types.MilestoneKey(homeID, milestoneID), bz); err != nil {
		return "", fmt.Errorf("failed to store milestone: %w", err)
	}

	return milestoneID, nil
}

// GetMilestone retrieves a milestone by ID.
func (k Keeper) GetMilestone(ctx sdk.Context, homeID, milestoneID string) (types.MilestoneEntry, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.MilestoneKey(homeID, milestoneID))
	if err != nil || bz == nil {
		return types.MilestoneEntry{}, false
	}
	var entry types.MilestoneEntry
	if err := json.Unmarshal(bz, &entry); err != nil {
		return types.MilestoneEntry{}, false
	}
	return entry, true
}

// ListMilestones returns all milestones for a home, newest first (descending BlockHeight).
func (k Keeper) ListMilestones(ctx sdk.Context, homeID string) []types.MilestoneEntry {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.MilestonePrefixKey(homeID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var entries []types.MilestoneEntry
	for ; iter.Valid(); iter.Next() {
		var entry types.MilestoneEntry
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	// Sort newest first; deterministic tiebreaker on ID.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].BlockHeight != entries[j].BlockHeight {
			return entries[i].BlockHeight > entries[j].BlockHeight
		}
		return entries[i].ID > entries[j].ID
	})

	return entries
}
