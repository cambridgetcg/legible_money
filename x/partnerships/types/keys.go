package types

const (
	// ModuleName defines the module name.
	ModuleName = "partnerships"

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
	PartnershipKeyPrefix       = []byte{0x01}
	FormationKeyPrefix         = []byte{0x02}
	ByHumanIndexPrefix         = []byte{0x03}
	ByAgentIndexPrefix         = []byte{0x04}
	CommonPotKeyPrefix         = []byte{0x05}
	ConsensusOpKeyPrefix       = []byte{0x06}
	CoercionSignalKeyPrefix    = []byte{0x07}
	SafetyFreezeKeyPrefix      = []byte{0x08}
	RejectionCooldownKeyPrefix = []byte{0x09}
	ParamsKey                  = []byte{0x0a}
	SequenceKey                = []byte{0x0b}
	SeedPartnershipKeyPrefix   = []byte{0x10}
	PoolEntryKeyPrefix         = []byte{0x11}
	IntentKeyPrefix            = []byte{0x12}
	MentorshipKeyPrefix        = []byte{0x13}
	ByDIDSeedIndexPrefix       = []byte{0x14}
	ByDIDIntentIndexPrefix     = []byte{0x15}
	BySponsorMentorIndexPrefix = []byte{0x16}
)
