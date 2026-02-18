package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/compute_pool/types"
)

// -----------------------------------------------------------------------
// Provider CRUD
// -----------------------------------------------------------------------

// SetProvider stores a compute provider record.
func (k Keeper) SetProvider(ctx sdk.Context, provider types.ComputeProvider) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := providerKey(provider.Address)
	bz, err := json.Marshal(&provider)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal provider: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetProvider retrieves a compute provider by address.
func (k Keeper) GetProvider(ctx sdk.Context, address string) (types.ComputeProvider, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := providerKey(address)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return types.ComputeProvider{}, false
	}
	var provider types.ComputeProvider
	if err := json.Unmarshal(bz, &provider); err != nil {
		return types.ComputeProvider{}, false
	}
	return provider, true
}

// DeleteProvider removes a compute provider.
func (k Keeper) DeleteProvider(ctx sdk.Context, address string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(providerKey(address))
}

// GetAllProviders returns all compute providers.
func (k Keeper) GetAllProviders(ctx sdk.Context) []types.ComputeProvider {
	var providers []types.ComputeProvider
	k.IterateProviders(ctx, func(p types.ComputeProvider) bool {
		providers = append(providers, p)
		return false
	})
	return providers
}

// IterateProviders iterates over all providers. Return true from cb to stop.
func (k Keeper) IterateProviders(ctx sdk.Context, cb func(types.ComputeProvider) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ProviderKeyPrefix, prefixEndBytes(types.ProviderKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var provider types.ComputeProvider
		if err := json.Unmarshal(iter.Value(), &provider); err != nil {
			continue
		}
		if cb(provider) {
			break
		}
	}
}

// -----------------------------------------------------------------------
// Credit CRUD
// -----------------------------------------------------------------------

// SetCredit stores a validator's compute credit record.
func (k Keeper) SetCredit(ctx sdk.Context, credit types.ComputeCredit) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := creditKey(credit.ValidatorAddr)
	bz, err := json.Marshal(&credit)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal credit: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetCredit retrieves a validator's compute credit balance.
func (k Keeper) GetCredit(ctx sdk.Context, validatorAddr string) (types.ComputeCredit, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := creditKey(validatorAddr)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return types.ComputeCredit{}, false
	}
	var credit types.ComputeCredit
	if err := json.Unmarshal(bz, &credit); err != nil {
		return types.ComputeCredit{}, false
	}
	return credit, true
}

// GetAllCredits returns all credit records.
func (k Keeper) GetAllCredits(ctx sdk.Context) []types.ComputeCredit {
	var credits []types.ComputeCredit
	k.IterateCredits(ctx, func(c types.ComputeCredit) bool {
		credits = append(credits, c)
		return false
	})
	return credits
}

// IterateCredits iterates over all credit records. Return true from cb to stop.
func (k Keeper) IterateCredits(ctx sdk.Context, cb func(types.ComputeCredit) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.CreditBalancePrefix, prefixEndBytes(types.CreditBalancePrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var credit types.ComputeCredit
		if err := json.Unmarshal(iter.Value(), &credit); err != nil {
			continue
		}
		if cb(credit) {
			break
		}
	}
}

// -----------------------------------------------------------------------
// Redemption Ticket CRUD
// -----------------------------------------------------------------------

// SetTicket stores a redemption ticket.
func (k Keeper) SetTicket(ctx sdk.Context, ticket types.RedemptionTicket) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := ticketKey(ticket.Id)
	bz, err := json.Marshal(&ticket)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal ticket: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetTicket retrieves a redemption ticket by ID.
func (k Keeper) GetTicket(ctx sdk.Context, ticketId uint64) (types.RedemptionTicket, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := ticketKey(ticketId)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return types.RedemptionTicket{}, false
	}
	var ticket types.RedemptionTicket
	if err := json.Unmarshal(bz, &ticket); err != nil {
		return types.RedemptionTicket{}, false
	}
	return ticket, true
}

// DeleteTicket removes a redemption ticket.
func (k Keeper) DeleteTicket(ctx sdk.Context, ticketId uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(ticketKey(ticketId))
}

// IterateTickets iterates over all redemption tickets. Return true from cb to stop.
func (k Keeper) IterateTickets(ctx sdk.Context, cb func(types.RedemptionTicket) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.RedemptionTicketPrefix, prefixEndBytes(types.RedemptionTicketPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var ticket types.RedemptionTicket
		if err := json.Unmarshal(iter.Value(), &ticket); err != nil {
			continue
		}
		if cb(ticket) {
			break
		}
	}
}

// GetNextTicketId returns the next ticket ID and increments the counter.
func (k Keeper) GetNextTicketId(ctx sdk.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.TicketSequenceKey)
	if err != nil || bz == nil {
		// Start at 1
		nextBz := make([]byte, 8)
		binary.BigEndian.PutUint64(nextBz, 2)
		_ = kvStore.Set(types.TicketSequenceKey, nextBz)
		return 1
	}
	current := binary.BigEndian.Uint64(bz)
	nextBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nextBz, current+1)
	_ = kvStore.Set(types.TicketSequenceKey, nextBz)
	return current
}

// -----------------------------------------------------------------------
// Usage Report CRUD
// -----------------------------------------------------------------------

// SetUsageReport stores a confirmed usage report.
func (k Keeper) SetUsageReport(ctx sdk.Context, report types.UsageReport) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := usageReportKey(report.TicketId)
	bz, err := json.Marshal(&report)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal usage report: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetUsageReport retrieves a usage report by ticket ID.
func (k Keeper) GetUsageReport(ctx sdk.Context, ticketId uint64) (types.UsageReport, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := usageReportKey(ticketId)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return types.UsageReport{}, false
	}
	var report types.UsageReport
	if err := json.Unmarshal(bz, &report); err != nil {
		return types.UsageReport{}, false
	}
	return report, true
}

// -----------------------------------------------------------------------
// Epoch Stats CRUD
// -----------------------------------------------------------------------

// SetEpochStats stores epoch statistics.
func (k Keeper) SetEpochStats(ctx sdk.Context, stats types.EpochStats) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := epochStatsKey(stats.Epoch)
	bz, err := json.Marshal(&stats)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal epoch stats: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetEpochStats retrieves epoch statistics.
func (k Keeper) GetEpochStats(ctx sdk.Context, epoch uint64) (types.EpochStats, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := epochStatsKey(epoch)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return types.EpochStats{}, false
	}
	var stats types.EpochStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return types.EpochStats{}, false
	}
	return stats, true
}

// GetOrCreateEpochStats returns existing stats or creates a new entry.
func (k Keeper) GetOrCreateEpochStats(ctx sdk.Context, epoch uint64) types.EpochStats {
	stats, found := k.GetEpochStats(ctx, epoch)
	if found {
		return stats
	}
	return types.EpochStats{
		Epoch:          epoch,
		PoolInflow:     "0",
		LgmDistributed: "0",
	}
}

// -----------------------------------------------------------------------
// Provider Epoch Flow CRUD
// -----------------------------------------------------------------------

// SetProviderEpochFlow stores per-provider per-epoch flow data.
func (k Keeper) SetProviderEpochFlow(ctx sdk.Context, flow types.ProviderEpochFlow) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := providerEpochKey(flow.Epoch, flow.Provider)
	bz, err := json.Marshal(&flow)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal provider epoch flow: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetProviderEpochFlow retrieves per-provider per-epoch flow data.
func (k Keeper) GetProviderEpochFlow(ctx sdk.Context, epoch uint64, provider string) (types.ProviderEpochFlow, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := providerEpochKey(epoch, provider)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return types.ProviderEpochFlow{}, false
	}
	var flow types.ProviderEpochFlow
	if err := json.Unmarshal(bz, &flow); err != nil {
		return types.ProviderEpochFlow{}, false
	}
	return flow, true
}

// GetOrCreateProviderEpochFlow returns existing flow or creates a new entry.
func (k Keeper) GetOrCreateProviderEpochFlow(ctx sdk.Context, epoch uint64, provider string) types.ProviderEpochFlow {
	flow, found := k.GetProviderEpochFlow(ctx, epoch, provider)
	if found {
		return flow
	}
	return types.ProviderEpochFlow{
		Epoch:       epoch,
		Provider:    provider,
		LgmReceived: "0",
	}
}

// -----------------------------------------------------------------------
// Key construction helpers
// -----------------------------------------------------------------------

func providerKey(address string) []byte {
	return append(types.ProviderKeyPrefix, []byte(address)...)
}

func creditKey(validatorAddr string) []byte {
	return append(types.CreditBalancePrefix, []byte(validatorAddr)...)
}

func ticketKey(ticketId uint64) []byte {
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, ticketId)
	return append(types.RedemptionTicketPrefix, idBytes...)
}

func usageReportKey(ticketId uint64) []byte {
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, ticketId)
	return append(types.UsageReportPrefix, idBytes...)
}

func epochStatsKey(epoch uint64) []byte {
	epochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epochBytes, epoch)
	return append(types.EpochStatsPrefix, epochBytes...)
}

func providerEpochKey(epoch uint64, provider string) []byte {
	epochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epochBytes, epoch)
	key := append(types.ProviderEpochPrefix, epochBytes...)
	key = append(key, '/')
	return append(key, []byte(provider)...)
}
