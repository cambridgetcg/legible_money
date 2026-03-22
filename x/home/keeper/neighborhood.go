package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// CreateNeighborhood creates a new neighborhood with the founder as the first member.
func (k Keeper) CreateNeighborhood(ctx sdk.Context, founderHomeID, domain string) (string, error) {
	if founderHomeID == "" || domain == "" {
		return "", fmt.Errorf("founderHomeID and domain must be non-empty")
	}
	if _, found := k.GetHome(ctx, founderHomeID); !found {
		return "", types.ErrHomeNotFound.Wrapf("founder home %s not found", founderHomeID)
	}

	// Increment neighborhood counter.
	store := k.storeService.OpenKVStore(ctx)
	var counter uint64
	if bz, err := store.Get(types.NeighborhoodCounterKey); err == nil && bz != nil && len(bz) == 8 {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	counterBz := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBz, counter)
	if err := store.Set(types.NeighborhoodCounterKey, counterBz); err != nil {
		return "", fmt.Errorf("failed to set neighborhood counter: %w", err)
	}

	neighborhoodID := fmt.Sprintf("nbr-%d", counter)

	neighborhood := types.Neighborhood{
		NeighborhoodID: neighborhoodID,
		FounderHomeID:  founderHomeID,
		Domain:         domain,
		Members:        []string{founderHomeID},
		CreatedAt:      uint64(ctx.BlockHeight()),
	}

	if err := k.setNeighborhood(ctx, &neighborhood); err != nil {
		return "", err
	}

	// Set membership reverse index.
	if err := store.Set(types.NeighborhoodMemberKey(founderHomeID, neighborhoodID), []byte{1}); err != nil {
		return "", fmt.Errorf("failed to set membership index: %w", err)
	}

	return neighborhoodID, nil
}

// JoinNeighborhood adds a home to an existing neighborhood.
func (k Keeper) JoinNeighborhood(ctx sdk.Context, homeID, neighborhoodID string) error {
	if homeID == "" || neighborhoodID == "" {
		return fmt.Errorf("homeID and neighborhoodID must be non-empty")
	}
	if _, found := k.GetHome(ctx, homeID); !found {
		return types.ErrHomeNotFound.Wrapf("home %s not found", homeID)
	}

	nbr, found := k.GetNeighborhood(ctx, neighborhoodID)
	if !found {
		return fmt.Errorf("neighborhood %s not found", neighborhoodID)
	}

	// Check max members.
	if len(nbr.Members) >= types.MaxNeighborhoodMembers {
		return fmt.Errorf("neighborhood %s is full (%d/%d members)", neighborhoodID, len(nbr.Members), types.MaxNeighborhoodMembers)
	}

	// Check not already a member.
	for _, m := range nbr.Members {
		if m == homeID {
			return fmt.Errorf("home %s is already a member of neighborhood %s", homeID, neighborhoodID)
		}
	}

	nbr.Members = append(nbr.Members, homeID)
	sort.Strings(nbr.Members) // deterministic ordering

	if err := k.setNeighborhood(ctx, &nbr); err != nil {
		return err
	}

	// Set membership reverse index.
	store := k.storeService.OpenKVStore(ctx)
	return store.Set(types.NeighborhoodMemberKey(homeID, neighborhoodID), []byte{1})
}

// LeaveNeighborhood removes a home from a neighborhood.
func (k Keeper) LeaveNeighborhood(ctx sdk.Context, homeID, neighborhoodID string) error {
	if homeID == "" || neighborhoodID == "" {
		return fmt.Errorf("homeID and neighborhoodID must be non-empty")
	}

	nbr, found := k.GetNeighborhood(ctx, neighborhoodID)
	if !found {
		return fmt.Errorf("neighborhood %s not found", neighborhoodID)
	}

	// Find and remove member.
	idx := -1
	for i, m := range nbr.Members {
		if m == homeID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("home %s is not a member of neighborhood %s", homeID, neighborhoodID)
	}

	nbr.Members = append(nbr.Members[:idx], nbr.Members[idx+1:]...)

	if err := k.setNeighborhood(ctx, &nbr); err != nil {
		return err
	}

	// Remove membership reverse index.
	store := k.storeService.OpenKVStore(ctx)
	return store.Delete(types.NeighborhoodMemberKey(homeID, neighborhoodID))
}

// GetNeighborhood retrieves a neighborhood by ID.
func (k Keeper) GetNeighborhood(ctx sdk.Context, neighborhoodID string) (types.Neighborhood, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.NeighborhoodKey(neighborhoodID))
	if err != nil || bz == nil {
		return types.Neighborhood{}, false
	}
	var nbr types.Neighborhood
	if err := json.Unmarshal(bz, &nbr); err != nil {
		return types.Neighborhood{}, false
	}
	return nbr, true
}

// ListNeighborhoodMembers returns the sorted member list for a neighborhood.
func (k Keeper) ListNeighborhoodMembers(ctx sdk.Context, neighborhoodID string) []string {
	nbr, found := k.GetNeighborhood(ctx, neighborhoodID)
	if !found {
		return nil
	}
	return nbr.Members
}

// NeighborhoodBonus returns the bonus in basis points for a home.
// Returns 500 bps (5%) if the home is a member of any neighborhood with >= MinNeighborhoodMembers,
// otherwise 0.
func (k Keeper) NeighborhoodBonus(ctx sdk.Context, homeID string) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.NeighborhoodMemberPrefixKey(homeID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return 0
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		// Extract neighborhoodID from key: prefix + homeID + "/" + neighborhoodID
		key := iter.Key()
		// key = 0x1C + homeId + "/" + neighborhoodId
		// prefix = 0x1C + homeId + "/"
		nbrID := string(key[len(prefix):])
		nbr, found := k.GetNeighborhood(ctx, nbrID)
		if found && len(nbr.Members) >= types.MinNeighborhoodMembers {
			return types.NeighborhoodBonusBps
		}
	}

	return 0
}

// setNeighborhood persists a neighborhood to the store.
func (k Keeper) setNeighborhood(ctx sdk.Context, nbr *types.Neighborhood) error {
	bz, err := json.Marshal(nbr)
	if err != nil {
		return fmt.Errorf("failed to marshal neighborhood: %w", err)
	}
	store := k.storeService.OpenKVStore(ctx)
	return store.Set(types.NeighborhoodKey(nbr.NeighborhoodID), bz)
}
