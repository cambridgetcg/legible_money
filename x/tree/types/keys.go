package types

const (
	// ModuleName defines the module name.
	ModuleName = "tree"

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
	ProjectKeyPrefix    = []byte{0x01}
	ServiceKeyPrefix    = []byte{0x02}
	TaskKeyPrefix       = []byte{0x03}
	SeedKeyPrefix       = []byte{0x04}
	ParamsKey           = []byte{0x05}
)
