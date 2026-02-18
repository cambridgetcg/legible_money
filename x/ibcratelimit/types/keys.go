package types

const (
	// ModuleName defines the module name.
	ModuleName = "ibcratelimit"

	// StoreKey defines the primary module store key.
	// Uses "ratelimit" instead of ModuleName to avoid prefix collision
	// with the IBC core store key ("ibc" is a prefix of "ibcratelimit").
	StoreKey = "ratelimit"

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName
)

// Store key prefixes.
var (
	FlowPrefix         = []byte{0x01} // channel_id -> ChannelFlow (current epoch)
	ParamsKey          = []byte{0x02}
	PreviousFlowPrefix = []byte{0x03} // channel_id -> ChannelFlow (previous epoch, for sliding window)
)

// FlowKey returns the store key for a specific channel's flow record.
func FlowKey(channelID string) []byte {
	return append(FlowPrefix, []byte(channelID)...)
}

// PreviousFlowKey returns the store key for a channel's previous epoch flow record.
func PreviousFlowKey(channelID string) []byte {
	return append(PreviousFlowPrefix, []byte(channelID)...)
}
