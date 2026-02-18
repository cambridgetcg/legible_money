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

	// UnbondingSeqKey stores the monotonic unbonding sequence counter
	UnbondingSeqKey = []byte{0x07}

	// RedelegationCooldownPrefix tracks last redelegation height per delegator
	RedelegationCooldownPrefix = []byte{0x08}

	// ValidatorDelegationIndexPrefix is the reverse index: validator → delegators.
	// P1-1 fix: DelegationsByValidatorPrefix was broken (used DelegationKeyPrefix
	// which is keyed by delegator, not validator).
	ValidatorDelegationIndexPrefix = []byte{0x09}
)

// RedelegationCooldownBlocks is the minimum gap between redelegations for a
// single delegator. Prevents rapid stake-shifting attacks that could
// manipulate VRF selection probabilities within a single epoch.
// 1,111 blocks ≈ 46 minutes at 2.521s/block.
const RedelegationCooldownBlocks uint64 = 1111

// RedelegationCooldownKey returns the store key for a delegator's last redelegation height
func RedelegationCooldownKey(delegatorAddr string) []byte {
	return append(RedelegationCooldownPrefix, []byte(delegatorAddr)...)
}

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

// DelegationsByValidatorPrefix returns the prefix for all delegations to a validator.
// P1-1 fix: Uses the reverse index (ValidatorDelegationIndexPrefix) instead of
// DelegationKeyPrefix, which is keyed by delegator address.
func DelegationsByValidatorPrefix(validatorAddr string) []byte {
	return append(ValidatorDelegationIndexPrefix, append([]byte(validatorAddr), 0x00)...)
}

// ValidatorDelegationIndexKey returns the store key for a validator→delegator index entry.
func ValidatorDelegationIndexKey(validatorAddr, delegatorAddr string) []byte {
	key := append(ValidatorDelegationIndexPrefix, []byte(validatorAddr)...)
	key = append(key, 0x00) // separator
	return append(key, []byte(delegatorAddr)...)
}

// UnbondingKey returns the store key for an unbonding entry
func UnbondingKey(id string) []byte {
	return append(UnbondingKeyPrefix, []byte(id)...)
}

// TierConfigKey returns the store key for a tier config
func TierConfigKey(tier ValidatorTier) []byte {
	return append(TierConfigKeyPrefix, byte(tier))
}
