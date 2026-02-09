package types

const (
	// ModuleName defines the module name.
	ModuleName = "channels"

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
	ChannelKeyPrefix    = []byte{0x01}
	ChannelStatePrefix  = []byte{0x02}
	DisputeKeyPrefix    = []byte{0x03}
	SettlementKeyPrefix = []byte{0x04}
	ParamsKey           = []byte{0x05}
	// Index prefixes
	PayerIndexPrefix    = []byte{0x10}
	ProviderIndexPrefix = []byte{0x11}
)

// ChannelKey returns the store key for a specific channel.
func ChannelKey(channelId string) []byte {
	return append(ChannelKeyPrefix, []byte(channelId)...)
}

// DisputeKey returns the store key for a dispute.
func DisputeKey(channelId string) []byte {
	return append(DisputeKeyPrefix, []byte(channelId)...)
}

// PayerChannelKey returns the index key for a payer's channel.
func PayerChannelKey(payer, channelId string) []byte {
	return append(PayerIndexPrefix, []byte(payer+"/"+channelId)...)
}

// PayerChannelPrefix returns the prefix for all of a payer's channels.
func PayerChannelPrefix(payer string) []byte {
	return append(PayerIndexPrefix, []byte(payer+"/")...)
}

// ProviderChannelKey returns the index key for a provider's channel.
func ProviderChannelKey(provider, channelId string) []byte {
	return append(ProviderIndexPrefix, []byte(provider+"/"+channelId)...)
}

// ProviderChannelPrefix returns the prefix for all of a provider's channels.
func ProviderChannelPrefix(provider string) []byte {
	return append(ProviderIndexPrefix, []byte(provider+"/")...)
}
