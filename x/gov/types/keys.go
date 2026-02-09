package types

const (
	// ModuleName defines the module name.
	ModuleName = "lgm_gov"

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
	LIPKeyPrefix          = []byte{0x01}
	VoteKeyPrefix         = []byte{0x02}
	ReviewKeyPrefix       = []byte{0x03}
	DisbursementKeyPrefix = []byte{0x04}
	ParamsKey             = []byte{0x05}
	LIPCounterKey         = []byte{0x06}
	DisbCounterKey        = []byte{0x07}
	VoteDedupeKeyPrefix   = []byte{0x08}
)
