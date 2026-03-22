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

	// Messaging prefixes
	MsgChannelKeyPrefix    = []byte{0x30} // message channel by ID
	MsgChannelTypePrefix   = []byte{0x31} // index: type → channel IDs
	MsgChannelMemberPrefix = []byte{0x32} // index: member → channel IDs
	ChannelMsgKeyPrefix    = []byte{0x33} // messages within a channel
	MsgChannelCounterKey   = []byte{0x34} // monotonic channel ID counter
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

// --- Messaging Keys ---

// MsgChannelKey returns the store key for a message channel.
func MsgChannelKey(channelId string) []byte {
	return append(MsgChannelKeyPrefix, []byte(channelId)...)
}

// MsgChannelTypeKey returns the index key for a channel under its type.
func MsgChannelTypeKey(channelType, channelId string) []byte {
	return append(MsgChannelTypePrefix, []byte(channelType+"/"+channelId)...)
}

// MsgChannelTypeKeyPrefix returns the prefix for all channels of a type.
func MsgChannelTypeKeyPrefix(channelType string) []byte {
	return append(MsgChannelTypePrefix, []byte(channelType+"/")...)
}

// MsgChannelMemberKey returns the index key for a member's channel.
func MsgChannelMemberKey(member, channelId string) []byte {
	return append(MsgChannelMemberPrefix, []byte(member+"/"+channelId)...)
}

// MsgChannelMemberKeyPrefix returns the prefix for all channels a member belongs to.
func MsgChannelMemberKeyPrefix(member string) []byte {
	return append(MsgChannelMemberPrefix, []byte(member+"/")...)
}

// ChannelMsgKey returns the store key for a specific message in a channel.
// Messages are keyed by channel ID + 8-byte big-endian sequence for ordered iteration.
func ChannelMsgKey(channelId string, sequence uint64) []byte {
	prefix := append(ChannelMsgKeyPrefix, []byte(channelId+"/")...)
	seqBytes := make([]byte, 8)
	seqBytes[0] = byte(sequence >> 56)
	seqBytes[1] = byte(sequence >> 48)
	seqBytes[2] = byte(sequence >> 40)
	seqBytes[3] = byte(sequence >> 32)
	seqBytes[4] = byte(sequence >> 24)
	seqBytes[5] = byte(sequence >> 16)
	seqBytes[6] = byte(sequence >> 8)
	seqBytes[7] = byte(sequence)
	return append(prefix, seqBytes...)
}

// ChannelMsgPrefix returns the prefix for all messages in a channel.
func ChannelMsgPrefix(channelId string) []byte {
	return append(ChannelMsgKeyPrefix, []byte(channelId+"/")...)
}
