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
	GasZero          uint64 = 0
	GasBase          uint64 = 2
	GasVeryLow       uint64 = 3
	GasLow           uint64 = 5
	GasMid           uint64 = 8
	GasHigh          uint64 = 10
	GasExpBase       uint64 = 10
	GasExpPerByte    uint64 = 50 // EIP-160
	GasSHA3Base      uint64 = 30
	GasSHA3PerWord   uint64 = 6
	GasSloadCost     uint64 = 200
	GasSstoreSet     uint64 = 20000
	GasSstoreReset   uint64 = 5000
	GasSstoreClear   uint64 = 5000
	GasMemoryBase    uint64 = 3
	GasLogBase       uint64 = 375
	GasLogTopic      uint64 = 375
	GasLogData       uint64 = 8
	GasCallBase      uint64 = 700
	GasCallValue     uint64 = 9000
	GasCallNew       uint64 = 25000
	GasCallStipend   uint64 = 2300
	GasBalance       uint64 = 400
	GasExtCode       uint64 = 700
	GasCreate        uint64 = 32000
	GasSelfDestruct  uint64 = 5000
	GasCopy          uint64 = 3 // Per word
	GasJumpDest      uint64 = 1
)
