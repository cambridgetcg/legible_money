package types

const (
	// ModuleName defines the module name.
	ModuleName = "channels"

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
	ChannelKeyPrefix       = []byte{0x01}
	ChannelStatePrefix     = []byte{0x02}
	DisputeKeyPrefix       = []byte{0x03}
	SettlementKeyPrefix    = []byte{0x04}
	ParamsKey              = []byte{0x05}
)
