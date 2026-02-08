package types

const (
	// ModuleName defines the module name.
	ModuleName = "knowledge"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key.
	MemStoreKey = "mem_" + ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName

	// PortID is the IBC port ID for the knowledge oracle.
	PortID = "knowledge"

	// Version is the IBC module version for knowledge oracle.
	Version = "lgm-knowledge-1"
)

// Store key prefixes.
var (
	FactKeyPrefix              = []byte{0x01}
	ClaimKeyPrefix             = []byte{0x02}
	VerificationRoundKeyPrefix = []byte{0x03}
	FactReferenceKeyPrefix     = []byte{0x04}
	DomainFactIndexPrefix      = []byte{0x05}
	ParamsKey                  = []byte{0x06}
)
