package vm

import "math/big"

// 256-bit word type for EVM-compatible stack machine.
// Backed by *big.Int for correctness; optimize with holiman/uint256 if profiling shows need.

var (
	MaxUint256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	SignBit    = new(big.Int).Lsh(big.NewInt(1), 255)
	WordMod    = new(big.Int).Lsh(big.NewInt(1), 256)
	Zero       = big.NewInt(0)
	One        = big.NewInt(1)
)

func NewWord(v int64) *big.Int {
	return big.NewInt(v)
}

func WordFromUint64(v uint64) *big.Int {
	return new(big.Int).SetUint64(v)
}

func WordFromBytes(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}

// WordToBytes32 returns a 32-byte big-endian representation, left-padded with zeros.
func WordToBytes32(w *big.Int) [32]byte {
	var result [32]byte
	b := w.Bytes()
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(result[32-len(b):], b)
	return result
}

// WordToBytes20 returns the low 20 bytes (EVM address) from a 256-bit word.
func WordToBytes20(w *big.Int) []byte {
	b32 := WordToBytes32(w)
	result := make([]byte, 20)
	copy(result, b32[12:]) // last 20 bytes
	return result
}

// ToSigned interprets a 256-bit unsigned value as two's complement signed.
func ToSigned(w *big.Int) *big.Int {
	if w.Bit(255) == 0 {
		return new(big.Int).Set(w)
	}
	// Negative: result = w - 2^256
	return new(big.Int).Sub(w, WordMod)
}

// FromSigned converts a signed value back to 256-bit unsigned (mod 2^256).
func FromSigned(w *big.Int) *big.Int {
	if w.Sign() >= 0 {
		return new(big.Int).And(w, MaxUint256)
	}
	// w + 2^256
	return new(big.Int).Add(w, WordMod)
}

// Mod256 reduces w modulo 2^256.
func Mod256(w *big.Int) *big.Int {
	return new(big.Int).And(w, MaxUint256)
}
