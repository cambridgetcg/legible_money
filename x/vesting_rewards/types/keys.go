package types

const (
	// ModuleName defines the module name.
	ModuleName = "vesting_rewards"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key.
	MemStoreKey = "mem_" + ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName

	// MaxSupplyUlgm is the hard supply cap: 222,222,222 LGM in ulgm.
	MaxSupplyUlgm = "222222222000000"
)

// Store key prefixes.
var (
	VestingScheduleKeyPrefix    = []byte{0x01} // {vestingId} -> VestingSchedule
	ClaimRecordKeyPrefix        = []byte{0x02} // {claimId} -> vestingId
	FalsificationKeyPrefix      = []byte{0x03} // {claimId} -> ClawbackRecord
	TruthLinkKeyPrefix          = []byte{0x04} // {factId} -> []vestingId
	ParamsKey                   = []byte{0x05} // -> Params
	CategoryConfigKeyPrefix     = []byte{0x06} // {category} -> CategoryConfig
	BlockRewardKeyPrefix        = []byte{0x07} // {blockHeight} -> BlockRewardDistribution
	VestingByRecipientPrefix    = []byte{0x08} // {address}/{vestingId} -> nil (index)
	ActiveVestingPrefix         = []byte{0x09} // {vestingId} -> nil (index for active schedules)
	BlockRewardFundBalanceKey   = []byte{0x0A} // -> string (remaining fund balance in ulgm) [DEPRECATED: use TotalMintedKey]
	TotalMintedKey             = []byte{0x0B} // -> string (total LGM minted so far in ulgm)
)
