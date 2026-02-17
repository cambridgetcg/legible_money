package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/partnerships/types"
)

// Pool matching constants (mirror TS FORMATION_POOL).
const (
	PoolRegistrationDeposit = "11000" // 0.011 LGM in ulgm
	PoolMaxSize             = 222
	PoolEntryExpiryBlocks   = 11111
	PoolWithdrawCooldown    = 222
	IntentDeclarationWindow = 2222
	IntentDeposit           = "11000" // 0.011 LGM in ulgm
	IntentMaxPending        = 3
)

// ---------- PoolEntry CRUD ----------

// SetPoolEntry stores a pool entry.
func (k Keeper) SetPoolEntry(ctx sdk.Context, pe types.PoolEntry) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&pe)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal pool entry: %v", err))
	}
	_ = kvStore.Set(poolEntryKey(pe.Address), bz)
}

// GetPoolEntry retrieves a pool entry by address.
func (k Keeper) GetPoolEntry(ctx sdk.Context, addr string) (types.PoolEntry, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(poolEntryKey(addr))
	if err != nil || bz == nil {
		return types.PoolEntry{}, false
	}
	var pe types.PoolEntry
	if err := proto.Unmarshal(bz, &pe); err != nil {
		return types.PoolEntry{}, false
	}
	return pe, true
}

// DeletePoolEntry removes a pool entry.
func (k Keeper) DeletePoolEntry(ctx sdk.Context, addr string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(poolEntryKey(addr))
}

// CountActivePoolEntries returns the number of active entries in the pool.
func (k Keeper) CountActivePoolEntries(ctx sdk.Context) int {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.PoolEntryKeyPrefix, prefixEndBytes(types.PoolEntryKeyPrefix))
	if err != nil {
		return 0
	}
	defer iter.Close()

	count := 0
	for ; iter.Valid(); iter.Next() {
		var pe types.PoolEntry
		if err := proto.Unmarshal(iter.Value(), &pe); err == nil && pe.Status == "active" {
			count++
		}
	}
	return count
}

// GetAllPoolEntries returns all pool entries.
func (k Keeper) GetAllPoolEntries(ctx sdk.Context) []types.PoolEntry {
	var entries []types.PoolEntry
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.PoolEntryKeyPrefix, prefixEndBytes(types.PoolEntryKeyPrefix))
	if err != nil {
		return entries
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var pe types.PoolEntry
		if err := proto.Unmarshal(iter.Value(), &pe); err == nil {
			entries = append(entries, pe)
		}
	}
	return entries
}

// ---------- PartnershipIntent CRUD ----------

// SetIntent stores a partnership intent.
func (k Keeper) SetIntent(ctx sdk.Context, pi types.PartnershipIntent) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&pi)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal intent: %v", err))
	}
	_ = kvStore.Set(intentKey(pi.IntentId), bz)
	_ = kvStore.Set(byDIDIntentKey(pi.Address, pi.IntentId), []byte(pi.IntentId))
}

// GetIntent retrieves an intent by ID.
func (k Keeper) GetIntent(ctx sdk.Context, id string) (types.PartnershipIntent, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(intentKey(id))
	if err != nil || bz == nil {
		return types.PartnershipIntent{}, false
	}
	var pi types.PartnershipIntent
	if err := proto.Unmarshal(bz, &pi); err != nil {
		return types.PartnershipIntent{}, false
	}
	return pi, true
}

// DeleteIntent removes an intent.
func (k Keeper) DeleteIntent(ctx sdk.Context, pi types.PartnershipIntent) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(intentKey(pi.IntentId))
	_ = kvStore.Delete(byDIDIntentKey(pi.Address, pi.IntentId))
}

// CountPendingIntentsByDID returns the number of pending intents for a DID.
func (k Keeper) CountPendingIntentsByDID(ctx sdk.Context, addr string) int {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := byDIDIntentPrefix(addr)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return 0
	}
	defer iter.Close()

	count := 0
	for ; iter.Valid(); iter.Next() {
		id := string(iter.Value())
		if pi, found := k.GetIntent(ctx, id); found && pi.Status == "pending" {
			count++
		}
	}
	return count
}

// ---------- MentorshipConfig CRUD ----------

// SetMentorship stores a mentorship config.
func (k Keeper) SetMentorship(ctx sdk.Context, mc types.MentorshipConfig) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&mc)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal mentorship: %v", err))
	}
	_ = kvStore.Set(mentorshipKey(mc.PartnershipId), bz)
	_ = kvStore.Set(bySponsorMentorKey(mc.SponsorAddr, mc.PartnershipId), []byte(mc.PartnershipId))
}

// GetMentorship retrieves a mentorship config by partnership ID.
func (k Keeper) GetMentorship(ctx sdk.Context, partnershipId string) (types.MentorshipConfig, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(mentorshipKey(partnershipId))
	if err != nil || bz == nil {
		return types.MentorshipConfig{}, false
	}
	var mc types.MentorshipConfig
	if err := proto.Unmarshal(bz, &mc); err != nil {
		return types.MentorshipConfig{}, false
	}
	return mc, true
}

// CountActiveMentorshipsBySponsor returns the number of active mentorships for a sponsor.
func (k Keeper) CountActiveMentorshipsBySponsor(ctx sdk.Context, sponsorAddr string) int {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := bySponsorMentorPrefix(sponsorAddr)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return 0
	}
	defer iter.Close()

	count := 0
	for ; iter.Valid(); iter.Next() {
		pId := string(iter.Value())
		if mc, found := k.GetMentorship(ctx, pId); found && !mc.Graduated {
			count++
		}
	}
	return count
}

// ---------- Key helpers ----------

func poolEntryKey(addr string) []byte {
	return append(types.PoolEntryKeyPrefix, []byte(addr)...)
}

func intentKey(id string) []byte {
	return append(types.IntentKeyPrefix, []byte(id)...)
}

func byDIDIntentPrefix(addr string) []byte {
	return append(types.ByDIDIntentIndexPrefix, []byte(addr+"/")...)
}

func byDIDIntentKey(addr, intentId string) []byte {
	return append(types.ByDIDIntentIndexPrefix, []byte(addr+"/"+intentId)...)
}

func mentorshipKey(partnershipId string) []byte {
	return append(types.MentorshipKeyPrefix, []byte(partnershipId)...)
}

func bySponsorMentorPrefix(sponsorAddr string) []byte {
	return append(types.BySponsorMentorIndexPrefix, []byte(sponsorAddr+"/")...)
}

func bySponsorMentorKey(sponsorAddr, partnershipId string) []byte {
	return append(types.BySponsorMentorIndexPrefix, []byte(sponsorAddr+"/"+partnershipId)...)
}
