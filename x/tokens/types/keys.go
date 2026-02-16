package types

const (
	// ModuleName defines the module name.
	ModuleName = "tokens"

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
	ParamsKey         = []byte{0x00}
	TokenDefKeyPrefix = []byte{0x01} // {denom} -> TokenDefinition (LGM-20)
	LPPoolKeyPrefix   = []byte{0x04} // {pool_id} -> LPPool (LGM-20)
)
