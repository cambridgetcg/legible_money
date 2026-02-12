package types

const (
	// ModuleName defines the module name.
	ModuleName = "evidence_mgmt"

	// StoreKey defines the primary module store key.
	// Renamed from ModuleName to avoid prefix collision with the standard SDK "evidence" module.
	StoreKey = "evid_mgmt"

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key.
	MemStoreKey = "mem_" + ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName
)

// Store key prefixes.
var (
	ParamsKey             = []byte{0x00}
	EvidenceKeyPrefix     = []byte{0x01}
	AuditKeyPrefix        = []byte{0x02}
	OracleKeyPrefix       = []byte{0x03}
	ConsensusKeyPrefix    = []byte{0x04}
	EvidenceByClaimPrefix = []byte{0x05}
	AuditByEvidencePrefix = []byte{0x06}
	ActiveOraclePrefix    = []byte{0x07}
	ExpiryIndexPrefix     = []byte{0x08} // expiryHeight(8 bytes big-endian) + evidenceId → evidenceId
)
