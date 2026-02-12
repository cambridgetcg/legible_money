package types

const (
	// ModuleName is the module name for the icaauth module.
	ModuleName = "icaauth"

	// StoreKey is the KVStore key for the icaauth module.
	StoreKey = ModuleName

	// RouterKey is the message router key for the icaauth module.
	RouterKey = ModuleName
)

// Store key prefixes.
var (
	// InterchainAccountPrefix stores DID → []RemoteAccount mappings.
	// Key format: 0x01 | DID bytes
	InterchainAccountPrefix = []byte{0x01}

	// RemoteAccountPrefix stores connectionId+portId → RemoteAccount for reverse lookup.
	// Key format: 0x02 | connectionId | 0x00 | portId
	RemoteAccountPrefix = []byte{0x02}

	// ParamsKey stores module parameters.
	ParamsKey = []byte{0x03}
)

// InterchainAccountKey returns the store key for a DID's interchain accounts.
func InterchainAccountKey(did string) []byte {
	return append(InterchainAccountPrefix, []byte(did)...)
}

// RemoteAccountKey returns the store key for a reverse lookup by connection+port.
func RemoteAccountKey(connectionId, portId string) []byte {
	key := append(RemoteAccountPrefix, []byte(connectionId)...)
	key = append(key, 0x00)
	key = append(key, []byte(portId)...)
	return key
}
