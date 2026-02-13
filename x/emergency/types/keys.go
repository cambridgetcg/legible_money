package types

const (
	// ModuleName defines the module name.
	ModuleName = "emergency"

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
	ProposalKeyPrefix           = []byte{0x01}
	VoteKeyPrefix               = []byte{0x02}
	HaltStatusKey               = []byte{0x03}
	AuditLogKeyPrefix           = []byte{0x04}
	ParamsKey                   = []byte{0x05}
	CeremonyKeyPrefix           = []byte{0x06}
	GuardianProposalCountPrefix = []byte{0x07}
	EpochProposalCountKey       = []byte{0x08}
	LastProposalBlockKey        = []byte{0x09}
	ActiveHaltCeremonyIdKey     = []byte{0x0A}
	HaltStartBlockKey           = []byte{0x0B}
)

// CeremonyKey returns the store key for a ceremony by ID.
func CeremonyKey(id string) []byte {
	return append(CeremonyKeyPrefix, []byte(id)...)
}

// GuardianProposalCountKey returns the store key for a guardian's proposal count.
func GuardianProposalCountKey(addr string) []byte {
	return append(GuardianProposalCountPrefix, []byte(addr)...)
}

// AuditLogKey returns the store key for an audit entry by height and index.
func AuditLogKey(height uint64, index uint32) []byte {
	key := make([]byte, len(AuditLogKeyPrefix)+12)
	copy(key, AuditLogKeyPrefix)
	// Big-endian height for ordering
	key[len(AuditLogKeyPrefix)] = byte(height >> 56)
	key[len(AuditLogKeyPrefix)+1] = byte(height >> 48)
	key[len(AuditLogKeyPrefix)+2] = byte(height >> 40)
	key[len(AuditLogKeyPrefix)+3] = byte(height >> 32)
	key[len(AuditLogKeyPrefix)+4] = byte(height >> 24)
	key[len(AuditLogKeyPrefix)+5] = byte(height >> 16)
	key[len(AuditLogKeyPrefix)+6] = byte(height >> 8)
	key[len(AuditLogKeyPrefix)+7] = byte(height)
	// Big-endian index
	key[len(AuditLogKeyPrefix)+8] = byte(index >> 24)
	key[len(AuditLogKeyPrefix)+9] = byte(index >> 16)
	key[len(AuditLogKeyPrefix)+10] = byte(index >> 8)
	key[len(AuditLogKeyPrefix)+11] = byte(index)
	return key
}
