package vm

import "errors"

var ErrOutOfGas = errors.New("out of gas")

// GasMeter tracks gas consumption during execution.
type GasMeter struct {
	limit uint64
	used  uint64
}

func NewGasMeter(limit uint64) *GasMeter {
	return &GasMeter{limit: limit}
}

// Consume deducts gas. Returns ErrOutOfGas if insufficient.
func (g *GasMeter) Consume(amount uint64) error {
	if g.used+amount > g.limit {
		g.used = g.limit
		return ErrOutOfGas
	}
	g.used += amount
	return nil
}

func (g *GasMeter) Remaining() uint64 {
	if g.used >= g.limit {
		return 0
	}
	return g.limit - g.used
}

func (g *GasMeter) Used() uint64 {
	return g.used
}

// Refund returns unused gas from a sub-call back to the parent meter.
func (g *GasMeter) Refund(amount uint64) {
	if amount > g.used {
		g.used = 0
		return
	}
	g.used -= amount
}

func (g *GasMeter) Limit() uint64 {
	return g.limit
}

// Gas cost constants matching the TS DEFAULT_GAS_COSTS + EIP conventions.
const (
	GasZero         uint64 = 0
	GasBase         uint64 = 2
	GasVeryLow      uint64 = 3
	GasLow          uint64 = 5
	GasMid          uint64 = 8
	GasHigh         uint64 = 10
	GasExpBase      uint64 = 10
	GasExpPerByte   uint64 = 50 // EIP-160
	GasSHA3Base     uint64 = 30
	GasSHA3PerWord  uint64 = 6
	GasSloadCost    uint64 = 200 // Legacy flat cost (used as base table entry)
	GasSstoreSet    uint64 = 20000
	GasSstoreReset  uint64 = 5000
	GasSstoreClear  uint64 = 5000
	GasMemoryBase   uint64 = 3
	GasLogBase      uint64 = 375
	GasLogTopic     uint64 = 375
	GasLogData      uint64 = 8
	GasCallBase     uint64 = 700
	GasCallValue    uint64 = 9000
	GasCallNew      uint64 = 25000
	GasCallStipend  uint64 = 2300
	GasBalance      uint64 = 400 // Legacy flat cost (used as base table entry)
	GasExtCode      uint64 = 700
	GasCreate       uint64 = 32000
	GasSelfDestruct uint64 = 5000
	GasCopy         uint64 = 3 // Per word
	GasJumpDest     uint64 = 1

	// EIP-2929 cold/warm access costs
	GasColdAccountAccess uint64 = 2600 // First access to an address in a tx
	GasColdSloadCost     uint64 = 2100 // First access to a storage slot in a tx
	GasWarmAccess        uint64 = 100  // Subsequent accesses within same tx
)

// GasSchedule provides governance-adjustable gas cost overrides for opcodes.
// When set on an ExecutionContext, any opcode with an override will use
// the override cost instead of its hardcoded OpcodeTable.GasCost.
type GasSchedule struct {
	Overrides map[byte]uint64
}

// GasCostFor returns the gas cost for an opcode, using the governance override
// if one exists, otherwise falling back to the default cost from OpcodeTable.
func (gs *GasSchedule) GasCostFor(opcode byte, defaultCost uint64) uint64 {
	if gs != nil && gs.Overrides != nil {
		if override, ok := gs.Overrides[opcode]; ok {
			return override
		}
	}
	return defaultCost
}

// AccessTracker implements EIP-2929 cold/warm storage access tracking.
// First access to an address or storage slot is "cold" (expensive);
// subsequent accesses within the same execution are "warm" (cheap).
type AccessTracker struct {
	addresses   map[string]bool            // warm addresses
	storageKeys map[string]map[string]bool // contract addr → slot → warm
}

// NewAccessTracker creates a fresh access tracker for a new execution.
func NewAccessTracker() *AccessTracker {
	return &AccessTracker{
		addresses:   make(map[string]bool),
		storageKeys: make(map[string]map[string]bool),
	}
}

// IsAddressWarm checks if an address has been accessed in this execution.
func (at *AccessTracker) IsAddressWarm(addr []byte) bool {
	return at.addresses[string(addr)]
}

// MarkAddressWarm marks an address as warm (accessed).
func (at *AccessTracker) MarkAddressWarm(addr []byte) {
	at.addresses[string(addr)] = true
}

// IsStorageWarm checks if a storage slot has been accessed in this execution.
func (at *AccessTracker) IsStorageWarm(contract, key []byte) bool {
	slots, ok := at.storageKeys[string(contract)]
	if !ok {
		return false
	}
	return slots[string(key)]
}

// MarkStorageWarm marks a storage slot as warm (accessed).
func (at *AccessTracker) MarkStorageWarm(contract, key []byte) {
	cKey := string(contract)
	if at.storageKeys[cKey] == nil {
		at.storageKeys[cKey] = make(map[string]bool)
	}
	at.storageKeys[cKey][string(key)] = true
}

// StorageGas returns the gas cost for an SLOAD, charging cold on first access.
func (at *AccessTracker) StorageGas(contract, key []byte) uint64 {
	if at.IsStorageWarm(contract, key) {
		return GasWarmAccess
	}
	at.MarkStorageWarm(contract, key)
	return GasColdSloadCost
}

// AddressGas returns the gas cost for a BALANCE/EXTCODE access.
func (at *AccessTracker) AddressGas(addr []byte) uint64 {
	if at.IsAddressWarm(addr) {
		return GasWarmAccess
	}
	at.MarkAddressWarm(addr)
	return GasColdAccountAccess
}
