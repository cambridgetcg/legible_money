package types

const (
	// ModuleName defines the module name
	ModuleName = "legible_staking"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_" + ModuleName
)

var (
	// ValidatorKeyPrefix is the prefix for validator store
	ValidatorKeyPrefix = []byte{0x01}

	// DelegationKeyPrefix is the prefix for delegation store
	DelegationKeyPrefix = []byte{0x02}

	// UnbondingKeyPrefix is the prefix for unbonding entries
	UnbondingKeyPrefix = []byte{0x03}

	// TierConfigKeyPrefix is the prefix for tier configs
	TierConfigKeyPrefix = []byte{0x04}

	// ParamsKey is the key for module params
	ParamsKey = []byte{0x05}

	// ValidatorByDIDPrefix is the prefix for DID -> validator address index
	ValidatorByDIDPrefix = []byte{0x06}
)

// ValidatorKey returns the store key for a validator by operator address
func ValidatorKey(operatorAddr string) []byte {
	return append(ValidatorKeyPrefix, []byte(operatorAddr)...)
}

// ValidatorByDIDKey returns the store key for DID -> validator address index
func ValidatorByDIDKey(did string) []byte {
	return append(ValidatorByDIDPrefix, []byte(did)...)
}

// DelegationKey returns the store key for a delegation
func DelegationKey(delegatorAddr, validatorAddr string) []byte {
	key := append(DelegationKeyPrefix, []byte(delegatorAddr)...)
	key = append(key, 0x00) // separator
	return append(key, []byte(validatorAddr)...)
}

// DelegationsByValidatorPrefix returns the prefix for all delegations to a validator
func DelegationsByValidatorPrefix(validatorAddr string) []byte {
	return append(DelegationKeyPrefix, []byte(validatorAddr)...)
}

// UnbondingKey returns the store key for an unbonding entry
func UnbondingKey(id string) []byte {
	return append(UnbondingKeyPrefix, []byte(id)...)
}

// TierConfigKey returns the store key for a tier config
func TierConfigKey(tier ValidatorTier) []byte {
	return append(TierConfigKeyPrefix, byte(tier))
}
