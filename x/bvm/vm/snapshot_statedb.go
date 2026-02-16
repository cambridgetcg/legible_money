package vm

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

// SnapshotStateDB is a copy-on-write wrapper around a base StateDB.
// Writes are captured in local caches. Commit() flushes to the base.
// If discarded (not committed), the base is untouched — this is the REVERT mechanism.
//
// Used by CALL/CREATE opcodes: create a snapshot before sub-call execution,
// commit on success, discard on failure/revert.
type SnapshotStateDB struct {
	base StateDB

	// Overlay caches — nil means no override, read through to base.
	storage map[string]map[string][]byte // addrHex -> keyHex -> value
	code    map[string][]byte            // addrHex -> bytecode
	nonces  map[string]uint64            // addrHex -> nonce (only if base supports NonceDB)
	exists  map[string]bool              // addrHex -> true/false override

	// Track which nonces have been written so Commit knows to flush them.
	noncesDirty map[string]bool
}

// NewSnapshot creates a copy-on-write overlay on top of base.
func NewSnapshot(base StateDB) *SnapshotStateDB {
	return &SnapshotStateDB{
		base:        base,
		storage:     make(map[string]map[string][]byte),
		code:        make(map[string][]byte),
		nonces:      make(map[string]uint64),
		exists:      make(map[string]bool),
		noncesDirty: make(map[string]bool),
	}
}

func snapAddrHex(addr []byte) string {
	return hex.EncodeToString(addr)
}

// --- StateDB interface ---

func (s *SnapshotStateDB) GetStorage(contract []byte, key []byte) []byte {
	a := snapAddrHex(contract)
	k := hex.EncodeToString(key)

	if m, ok := s.storage[a]; ok {
		if v, ok2 := m[k]; ok2 {
			return v
		}
	}
	// Read through to base
	return s.base.GetStorage(contract, key)
}

func (s *SnapshotStateDB) SetStorage(contract []byte, key []byte, value []byte) {
	a := snapAddrHex(contract)
	k := hex.EncodeToString(key)

	m, ok := s.storage[a]
	if !ok {
		m = make(map[string][]byte)
		s.storage[a] = m
	}
	// Copy value to avoid aliasing
	v := make([]byte, len(value))
	copy(v, value)
	m[k] = v
}

func (s *SnapshotStateDB) GetBalance(addr []byte) *big.Int {
	// Balance always reads through to base (bank keeper, not state).
	return s.base.GetBalance(addr)
}

func (s *SnapshotStateDB) GetCode(addr []byte) []byte {
	a := snapAddrHex(addr)
	if c, ok := s.code[a]; ok {
		return c
	}
	return s.base.GetCode(addr)
}

func (s *SnapshotStateDB) GetCodeSize(addr []byte) int {
	return len(s.GetCode(addr))
}

func (s *SnapshotStateDB) GetCodeHash(addr []byte) []byte {
	code := s.GetCode(addr)
	if code == nil {
		return nil
	}
	h := sha256.Sum256(code)
	return h[:]
}

func (s *SnapshotStateDB) Exists(addr []byte) bool {
	a := snapAddrHex(addr)
	if e, ok := s.exists[a]; ok {
		return e
	}
	// Check overlay caches
	if _, ok := s.code[a]; ok {
		return true
	}
	if _, ok := s.storage[a]; ok {
		return true
	}
	return s.base.Exists(addr)
}

// --- NonceDB interface ---

func (s *SnapshotStateDB) GetNonce(addr []byte) uint64 {
	a := snapAddrHex(addr)
	if _, ok := s.noncesDirty[a]; ok {
		return s.nonces[a]
	}
	// Read through to base if it supports nonces
	if nb, ok := s.base.(NonceDB); ok {
		return nb.GetNonce(addr)
	}
	return 0
}

func (s *SnapshotStateDB) SetNonce(addr []byte, nonce uint64) {
	a := snapAddrHex(addr)
	s.nonces[a] = nonce
	s.noncesDirty[a] = true
}

// --- CodeWriter interface ---

func (s *SnapshotStateDB) SetCode(addr []byte, bytecode []byte) {
	a := snapAddrHex(addr)
	c := make([]byte, len(bytecode))
	copy(c, bytecode)
	s.code[a] = c
	s.exists[a] = true
}

// --- StorageCounter interface ---

func (s *SnapshotStateDB) GetStorageSlotCount(contract []byte) uint64 {
	a := snapAddrHex(contract)

	// Start with base count
	var baseCount uint64
	if sc, ok := s.base.(StorageCounter); ok {
		baseCount = sc.GetStorageSlotCount(contract)
	}

	// Add overlay slots not in base (approximate — may overcount)
	if m, ok := s.storage[a]; ok {
		for k := range m {
			keyBytes, _ := hex.DecodeString(k)
			if s.base.GetStorage(contract, keyBytes) == nil {
				baseCount++
			}
		}
	}
	return baseCount
}

// --- Commit / Discard ---

// Commit flushes all cached writes to the base StateDB.
// After Commit, the snapshot should not be used further.
func (s *SnapshotStateDB) Commit() {
	// Flush storage
	for addrHex, slots := range s.storage {
		addr, _ := hex.DecodeString(addrHex)
		for keyHex, value := range slots {
			key, _ := hex.DecodeString(keyHex)
			s.base.SetStorage(addr, key, value)
		}
	}

	// Flush code
	if cw, ok := s.base.(CodeWriter); ok {
		for addrHex, bytecode := range s.code {
			addr, _ := hex.DecodeString(addrHex)
			cw.SetCode(addr, bytecode)
		}
	}

	// Flush nonces
	if nb, ok := s.base.(NonceDB); ok {
		for addrHex := range s.noncesDirty {
			addr, _ := hex.DecodeString(addrHex)
			nb.SetNonce(addr, s.nonces[addrHex])
		}
	}
}
