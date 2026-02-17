package types

const (
	// ModuleName defines the module name.
	ModuleName = "disputes"

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
	ParamsKey                = []byte{0x00}
	DisputeKeyPrefix         = []byte{0x01}
	EvidenceKeyPrefix        = []byte{0x02}
	VoteKeyPrefix            = []byte{0x03}
	DisputeByFactPrefix      = []byte{0x04}
	ActiveDisputePrefix      = []byte{0x05}
	EvidenceCommitmentPrefix = []byte{0x06}
)
