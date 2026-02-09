package types

const (
	// ModuleName defines the module name.
	ModuleName = "home"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName
)

// Store key prefixes.
var (
	HomeKeyPrefix         = []byte{0x01} // 0x01 + DID string
	SessionKeyPrefix      = []byte{0x02} // 0x02 + homeId/sessionId
	KeyRegistrationPrefix = []byte{0x03} // 0x03 + homeId/keyHash
	AlertKeyPrefix        = []byte{0x04} // 0x04 + homeId/alertId
	SpendingLimitPrefix   = []byte{0x05} // 0x05 + homeId/keyType
	HomeByOwnerPrefix     = []byte{0x06} // 0x06 + ownerAddr (index)
	ParamsKey             = []byte{0x07}
	HomeCounterKey        = []byte{0x08}
)

// HomeKey returns the store key for a home by DID.
func HomeKey(did string) []byte {
	return append(HomeKeyPrefix, []byte(did)...)
}

// SessionKey returns the store key for a session.
func SessionKey(homeId, sessionId string) []byte {
	return append(SessionKeyPrefix, []byte(homeId+"/"+sessionId)...)
}

// SessionPrefixKey returns the prefix for all sessions of a home.
func SessionPrefixKey(homeId string) []byte {
	return append(SessionKeyPrefix, []byte(homeId+"/")...)
}

// KeyRegKey returns the store key for a key registration.
func KeyRegKey(homeId, keyHash string) []byte {
	return append(KeyRegistrationPrefix, []byte(homeId+"/"+keyHash)...)
}

// KeyRegPrefixKey returns the prefix for all key registrations of a home.
func KeyRegPrefixKey(homeId string) []byte {
	return append(KeyRegistrationPrefix, []byte(homeId+"/")...)
}

// AlertKey returns the store key for an alert.
func AlertKey(homeId, alertId string) []byte {
	return append(AlertKeyPrefix, []byte(homeId+"/"+alertId)...)
}

// AlertPrefixKey returns the prefix for all alerts of a home.
func AlertPrefixKey(homeId string) []byte {
	return append(AlertKeyPrefix, []byte(homeId+"/")...)
}

// SpendLimitKey returns the store key for a spending limit.
func SpendLimitKey(homeId, keyType string) []byte {
	return append(SpendingLimitPrefix, []byte(homeId+"/"+keyType)...)
}

// HomeByOwnerKey returns the store key for a home-by-owner index entry.
func HomeByOwnerKey(addr string) []byte {
	return append(HomeByOwnerPrefix, []byte(addr)...)
}
