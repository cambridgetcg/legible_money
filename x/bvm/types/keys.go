package types

import "encoding/binary"

const (
	// ModuleName defines the module name.
	ModuleName = "bvm"

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
	ContractKeyPrefix = []byte{0x01}
	CodeKeyPrefix     = []byte{0x02}
	StateKeyPrefix    = []byte{0x03}
	ScheduleKeyPrefix = []byte{0x04}
	ParamsKey         = []byte{0x05}
	GasScheduleKey    = []byte{0x06}

	// Index prefixes
	ContractCreatorIndexPrefix   = []byte{0x10}
	ScheduleBlockIndexPrefix     = []byte{0x11}
	ScheduleCapabilityKeyPrefix  = []byte{0x12}

	// Counter keys
	ContractCounterKey = []byte{0x20}
	ScheduleCounterKey = []byte{0x21}
)

// ContractKey returns the store key for a contract by address.
func ContractKey(address string) []byte {
	return append(ContractKeyPrefix, []byte(address)...)
}

// CodeKey returns the store key for contract code by hash.
func CodeKey(codeHash string) []byte {
	return append(CodeKeyPrefix, []byte(codeHash)...)
}

// ContractStateKey returns the store key for a contract state entry.
// Uses length-prefixed encoding to prevent delimiter collision.
func ContractStateKey(contractAddress, key string) []byte {
	return lengthPrefixedKey(StateKeyPrefix, contractAddress, key)
}

// ContractStatePrefix returns the key prefix for all state of a contract.
func ContractStatePrefix(contractAddress string) []byte {
	return lengthPrefixedPrefix(StateKeyPrefix, contractAddress)
}

// ScheduleKey returns the store key for a schedule by ID.
func ScheduleKey(scheduleId string) []byte {
	return append(ScheduleKeyPrefix, []byte(scheduleId)...)
}

// ContractCreatorIndexKey returns the index key for contracts by creator.
// Uses length-prefixed encoding to prevent delimiter collision.
func ContractCreatorIndexKey(creator, address string) []byte {
	return lengthPrefixedKey(ContractCreatorIndexPrefix, creator, address)
}

// ContractCreatorPrefix returns the key prefix for a creator's contracts.
func ContractCreatorPrefix(creator string) []byte {
	return lengthPrefixedPrefix(ContractCreatorIndexPrefix, creator)
}

// ScheduleBlockIndexKey returns the index key for schedules by block.
// Block number is fixed-width 8 bytes (BigEndian), no delimiter needed.
func ScheduleBlockIndexKey(block uint64, scheduleId string) []byte {
	key := make([]byte, 0, len(ScheduleBlockIndexPrefix)+8+len(scheduleId))
	key = append(key, ScheduleBlockIndexPrefix...)
	key = append(key, Uint64ToBytes(block)...)
	key = append(key, []byte(scheduleId)...)
	return key
}

// ScheduleBlockPrefix returns the key prefix for schedules at a given block.
func ScheduleBlockPrefix(block uint64) []byte {
	key := make([]byte, 0, len(ScheduleBlockIndexPrefix)+8)
	key = append(key, ScheduleBlockIndexPrefix...)
	key = append(key, Uint64ToBytes(block)...)
	return key
}

// ScheduleCapabilityKey returns the store key for a schedule's stored capabilities.
func ScheduleCapabilityKey(scheduleId string) []byte {
	return append(ScheduleCapabilityKeyPrefix, []byte(scheduleId)...)
}

// Uint64ToBytes converts a uint64 to big-endian bytes.
func Uint64ToBytes(v uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, v)
	return bz
}

// BytesToUint64 converts big-endian bytes to a uint64.
func BytesToUint64(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

// lengthPrefixedKey builds a composite key: prefix + len(a) + a + b
// where len(a) is a single byte. a must be < 256 bytes.
func lengthPrefixedKey(prefix []byte, a, b string) []byte {
	aBytes := []byte(a)
	key := make([]byte, 0, len(prefix)+1+len(aBytes)+len(b))
	key = append(key, prefix...)
	key = append(key, byte(len(aBytes)))
	key = append(key, aBytes...)
	key = append(key, []byte(b)...)
	return key
}

// lengthPrefixedPrefix builds a prefix: prefix + len(a) + a
func lengthPrefixedPrefix(prefix []byte, a string) []byte {
	aBytes := []byte(a)
	key := make([]byte, 0, len(prefix)+1+len(aBytes))
	key = append(key, prefix...)
	key = append(key, byte(len(aBytes)))
	key = append(key, aBytes...)
	return key
}
