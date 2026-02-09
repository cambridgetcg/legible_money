package types

const (
	// ModuleName defines the module name.
	ModuleName = "claiming_pot"

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
	PotKeyPrefix          = []byte{0x01}
	ContributionKeyPrefix = []byte{0x02}
	DistributionKeyPrefix = []byte{0x03}
	MembershipKeyPrefix   = []byte{0x04}
	ParamsKey             = []byte{0x05}
)

// PotKey returns the store key for a specific pot.
func PotKey(potId string) []byte {
	return append(PotKeyPrefix, []byte(potId)...)
}

// ClaimRecordKey returns the store key for a claim record.
func ClaimRecordKey(potId, claimer string) []byte {
	return append(ContributionKeyPrefix, []byte(potId+"/"+claimer)...)
}

// ClaimRecordPrefixKey returns the prefix for all claim records of a pot.
func ClaimRecordPrefixKey(potId string) []byte {
	return append(ContributionKeyPrefix, []byte(potId+"/")...)
}

// CreatorPotCountKey returns the store key for a creator's pot count.
func CreatorPotCountKey(creator string) []byte {
	return append(MembershipKeyPrefix, []byte(creator)...)
}
