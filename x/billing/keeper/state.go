package keeper

import (
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/billing/types"
)

// ---------- Provider Operations ----------

// SetProvider stores a provider in the KV store.
func (k Keeper) SetProvider(ctx sdk.Context, provider *types.Provider) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := providerKey(provider.Address)
	bz, err := proto.Marshal(provider)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal provider: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetProvider retrieves a provider by address.
func (k Keeper) GetProvider(ctx sdk.Context, address string) (*types.Provider, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := providerKey(address)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var provider types.Provider
	if err := proto.Unmarshal(bz, &provider); err != nil {
		return nil, false
	}
	return &provider, true
}

// DeleteProvider removes a provider from the store.
func (k Keeper) DeleteProvider(ctx sdk.Context, address string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(providerKey(address))
}

// GetAllProviders returns all providers.
func (k Keeper) GetAllProviders(ctx sdk.Context) []*types.Provider {
	var providers []*types.Provider
	k.IterateProviders(ctx, func(p *types.Provider) bool {
		providers = append(providers, p)
		return false
	})
	return providers
}

// IterateProviders iterates over all providers. Return true from cb to stop.
func (k Keeper) IterateProviders(ctx sdk.Context, cb func(*types.Provider) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ProviderKeyPrefix, prefixEndBytes(types.ProviderKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var provider types.Provider
		if err := proto.Unmarshal(iter.Value(), &provider); err != nil {
			continue
		}
		if cb(&provider) {
			break
		}
	}
}

// ---------- Domain Index Operations ----------

// SetDomainIndex adds a provider to the domain index.
func (k Keeper) SetDomainIndex(ctx sdk.Context, domain string, address string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := domainIndexKey(domain, address)
	_ = kvStore.Set(key, []byte(address))
}

// DeleteDomainIndex removes a provider from the domain index.
func (k Keeper) DeleteDomainIndex(ctx sdk.Context, domain string, address string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(domainIndexKey(domain, address))
}

// GetProvidersByDomain returns provider addresses for a given domain.
func (k Keeper) GetProvidersByDomain(ctx sdk.Context, domain string) []string {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := domainPrefix(domain)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var addresses []string
	for ; iter.Valid(); iter.Next() {
		addresses = append(addresses, string(iter.Value()))
	}
	return addresses
}

// ---------- Quote Operations ----------

// GetNextQuoteId returns and increments the quote sequence counter.
func (k Keeper) GetNextQuoteId(ctx sdk.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.QuoteSequenceKey)
	if err != nil || bz == nil {
		// Start at 1
		nextBz := make([]byte, 8)
		binary.BigEndian.PutUint64(nextBz, 2)
		_ = kvStore.Set(types.QuoteSequenceKey, nextBz)
		return 1
	}
	current := binary.BigEndian.Uint64(bz)
	nextBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nextBz, current+1)
	_ = kvStore.Set(types.QuoteSequenceKey, nextBz)
	return current
}

// SetQuote stores a quote in the KV store.
func (k Keeper) SetQuote(ctx sdk.Context, quote *types.APIQuote) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := quoteKey(quote.QuoteId)
	bz, err := proto.Marshal(quote)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal quote: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetQuote retrieves a quote by ID.
func (k Keeper) GetQuote(ctx sdk.Context, quoteId uint64) (*types.APIQuote, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := quoteKey(quoteId)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var quote types.APIQuote
	if err := proto.Unmarshal(bz, &quote); err != nil {
		return nil, false
	}
	return &quote, true
}

// DeleteQuote removes a quote from the store.
func (k Keeper) DeleteQuote(ctx sdk.Context, quoteId uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(quoteKey(quoteId))
}

// IterateQuotes iterates over all quotes. Return true from cb to stop.
func (k Keeper) IterateQuotes(ctx sdk.Context, cb func(*types.APIQuote) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.QuoteKeyPrefix, prefixEndBytes(types.QuoteKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var quote types.APIQuote
		if err := proto.Unmarshal(iter.Value(), &quote); err != nil {
			continue
		}
		if cb(&quote) {
			break
		}
	}
}

// ---------- CallRecord Operations ----------

// GetNextCallRecordId returns and increments the call record sequence counter.
func (k Keeper) GetNextCallRecordId(ctx sdk.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.CallRecordSeqKey)
	if err != nil || bz == nil {
		nextBz := make([]byte, 8)
		binary.BigEndian.PutUint64(nextBz, 2)
		_ = kvStore.Set(types.CallRecordSeqKey, nextBz)
		return 1
	}
	current := binary.BigEndian.Uint64(bz)
	nextBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nextBz, current+1)
	_ = kvStore.Set(types.CallRecordSeqKey, nextBz)
	return current
}

// SetCallRecord stores a call record in the KV store.
func (k Keeper) SetCallRecord(ctx sdk.Context, record *types.CallRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := callRecordKey(record.RecordId)
	bz, err := proto.Marshal(record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal call record: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

// GetCallRecord retrieves a call record by ID.
func (k Keeper) GetCallRecord(ctx sdk.Context, recordId uint64) (*types.CallRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := callRecordKey(recordId)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var record types.CallRecord
	if err := proto.Unmarshal(bz, &record); err != nil {
		return nil, false
	}
	return &record, true
}

// IterateCallRecords iterates over all call records. Return true from cb to stop.
func (k Keeper) IterateCallRecords(ctx sdk.Context, cb func(*types.CallRecord) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.CallRecordKeyPrefix, prefixEndBytes(types.CallRecordKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var record types.CallRecord
		if err := proto.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		if cb(&record) {
			break
		}
	}
}

// ---------- Key Construction Helpers ----------

func providerKey(address string) []byte {
	return append(types.ProviderKeyPrefix, []byte(address)...)
}

func domainPrefix(domain string) []byte {
	return append(types.DomainIndexPrefix, []byte(domain+"/")...)
}

func domainIndexKey(domain string, address string) []byte {
	return append(types.DomainIndexPrefix, []byte(domain+"/"+address)...)
}

func quoteKey(quoteId uint64) []byte {
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, quoteId)
	return append(types.QuoteKeyPrefix, idBytes...)
}

func callRecordKey(recordId uint64) []byte {
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, recordId)
	return append(types.CallRecordKeyPrefix, idBytes...)
}
