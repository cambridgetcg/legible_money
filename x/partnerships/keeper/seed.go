package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/partnerships/types"
)

// Seed partnership constants (mirror TS SEED_TIER).
const (
	SeedContribution = "22000000000000000"  // 0.022 LGM
	SeedBond         = "11000000000000000"  // 0.011 LGM
	SeedExpiryBlocks = 22222
	SeedMaxPerDID    = 2
	SeedPotCap       = "222000000000000000" // 0.222 LGM
)

// ---------- SeedPartnership CRUD ----------

// SetSeedPartnership stores a seed partnership.
func (k Keeper) SetSeedPartnership(ctx sdk.Context, sp types.SeedPartnership) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&sp)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal seed partnership: %v", err))
	}
	_ = kvStore.Set(seedPartnershipKey(sp.Id), bz)

	// Bidirectional DID indexes for seed count enforcement
	_ = kvStore.Set(byDIDSeedKey(sp.HumanAddr, sp.Id), []byte(sp.Id))
	_ = kvStore.Set(byDIDSeedKey(sp.AgentAddr, sp.Id), []byte(sp.Id))
}

// GetSeedPartnership retrieves a seed partnership by ID.
func (k Keeper) GetSeedPartnership(ctx sdk.Context, id string) (types.SeedPartnership, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(seedPartnershipKey(id))
	if err != nil || bz == nil {
		return types.SeedPartnership{}, false
	}
	var sp types.SeedPartnership
	if err := proto.Unmarshal(bz, &sp); err != nil {
		return types.SeedPartnership{}, false
	}
	return sp, true
}

// DeleteSeedPartnership removes a seed partnership and its indexes.
func (k Keeper) DeleteSeedPartnership(ctx sdk.Context, sp types.SeedPartnership) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(seedPartnershipKey(sp.Id))
	_ = kvStore.Delete(byDIDSeedKey(sp.HumanAddr, sp.Id))
	_ = kvStore.Delete(byDIDSeedKey(sp.AgentAddr, sp.Id))
}

// CountActiveSeedsByDID returns the number of active seed partnerships for a DID.
func (k Keeper) CountActiveSeedsByDID(ctx sdk.Context, addr string) int {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := byDIDSeedPrefix(addr)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return 0
	}
	defer iter.Close()

	count := 0
	for ; iter.Valid(); iter.Next() {
		id := string(iter.Value())
		if sp, found := k.GetSeedPartnership(ctx, id); found && sp.Status == "active" {
			count++
		}
	}
	return count
}

// ---------- Key helpers ----------

func seedPartnershipKey(id string) []byte {
	return append(types.SeedPartnershipKeyPrefix, []byte(id)...)
}

func byDIDSeedPrefix(addr string) []byte {
	return append(types.ByDIDSeedIndexPrefix, []byte(addr+"/")...)
}

func byDIDSeedKey(addr, seedId string) []byte {
	return append(types.ByDIDSeedIndexPrefix, []byte(addr+"/"+seedId)...)
}
