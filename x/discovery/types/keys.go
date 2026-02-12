package types

const (
	// ModuleName defines the module name.
	ModuleName = "discovery"

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
	AgentProfileKeyPrefix   = []byte{0x01} // {address} -> AgentProfile
	ByDomainIndexPrefix     = []byte{0x02} // {domain}/{address} -> nil
	ByCapabilityIndexPrefix = []byte{0x03} // {capability_type}/{address} -> nil
	ParamsKey               = []byte{0x04}
)
