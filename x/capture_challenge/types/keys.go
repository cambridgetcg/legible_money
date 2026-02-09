package types

const (
	// ModuleName is the name of the capture challenge module.
	ModuleName = "capture_challenge"

	// StoreKey is the KV store key for the capture challenge module.
	StoreKey = ModuleName

	// RouterKey is the message routing key for the capture challenge module.
	RouterKey = ModuleName
)

// Store key prefixes for sub-stores.
var (
	ChallengeKeyPrefix  = []byte{0x01}
	BountyPoolKeyPrefix = []byte{0x02}
	ParamsKeyPrefix     = []byte{0x03}
)

// ChallengeKey returns the store key for a challenge by ID.
func ChallengeKey(challengeId string) []byte {
	return append(ChallengeKeyPrefix, []byte(challengeId)...)
}

// BountyPoolKey returns the store key for a domain's bounty pool.
func BountyPoolKey(domain string) []byte {
	return append(BountyPoolKeyPrefix, []byte(domain)...)
}
