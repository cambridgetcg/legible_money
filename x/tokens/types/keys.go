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
	ParamsKey           = []byte{0x00}
	TokenDefKeyPrefix   = []byte{0x01}
	BalanceKeyPrefix    = []byte{0x02} // {addr} -> ExpirableAccount
	AllowanceKeyPrefix  = []byte{0x03} // {owner}/{spender} -> amount
	LPPoolKeyPrefix     = []byte{0x04}
	PeriodBalancePrefix = []byte{0x05} // {addr}/{period} -> PeriodBalance
	RAIAccountKey       = []byte{0x06}
	SweepIndexPrefix    = []byte{0x07} // {period} -> []address to sweep
)
