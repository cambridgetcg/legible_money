package types

const (
	// ModuleName defines the module name.
	ModuleName = "billing"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key.
	MemStoreKey = "mem_" + ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName
)

// Store key prefixes.
var (
	ParamsKey           = []byte{0x00}
	ProviderKeyPrefix   = []byte{0x01} // {address} -> Provider
	QuoteKeyPrefix      = []byte{0x02} // {quoteId BE64} -> APIQuote
	CallRecordKeyPrefix = []byte{0x03} // {recordId BE64} -> CallRecord
	DomainIndexPrefix   = []byte{0x04} // {domain}/{address} -> provider address
	QuoteSequenceKey    = []byte{0x05} // -> uint64 (next quote ID)
	CallRecordSeqKey    = []byte{0x06} // -> uint64 (next call record ID)
)
