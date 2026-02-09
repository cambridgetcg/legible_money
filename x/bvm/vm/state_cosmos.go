package vm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"

	storetypes "cosmossdk.io/store/types"
)

// CosmosStateDB adapts a Cosmos KV store to the StateDB interface.
// Key layout matches x/bvm/keeper/state.go and x/bvm/types/keys.go exactly.
type CosmosStateDB struct {
	store storetypes.KVStore

	// Prefix bytes matching x/bvm/types/keys.go
	statePrefix    []byte // 0x03
	codePrefix     []byte // 0x02
	contractPrefix []byte // 0x01
}

// NewCosmosStateDB creates a StateDB backed by a Cosmos KV store.
func NewCosmosStateDB(store storetypes.KVStore) *CosmosStateDB {
	return &CosmosStateDB{
		store:          store,
		statePrefix:    []byte{0x03},
		codePrefix:     []byte{0x02},
		contractPrefix: []byte{0x01},
	}
}

// contractStateKey builds a key matching types.ContractStateKey:
// 0x03 + len(addr) + addr + hexKey (length-prefixed format)
func (s *CosmosStateDB) contractStateKey(contract []byte, key []byte) []byte {
	// contract is []byte of the address string (e.g., []byte("lgm1contract..."))
	addr := contract
	hexKey := hex.EncodeToString(key)
	k := make([]byte, 0, len(s.statePrefix)+1+len(addr)+len(hexKey))
	k = append(k, s.statePrefix...)
	k = append(k, byte(len(addr)))
	k = append(k, addr...)
	k = append(k, []byte(hexKey)...)
	return k
}

func (s *CosmosStateDB) GetStorage(contract []byte, key []byte) []byte {
	k := s.contractStateKey(contract, key)
	bz := s.store.Get(k)
	if bz == nil {
		return nil
	}
	// Value is stored as hex string by the keeper's SetContractState
	decoded, err := hex.DecodeString(string(bz))
	if err == nil && decoded != nil {
		return decoded
	}
	// Backward compat: try JSON format from old SetStorage
	var entry struct {
		Value string `json:"value"`
	}
	if jsonErr := json.Unmarshal(bz, &entry); jsonErr == nil {
		if d, _ := hex.DecodeString(entry.Value); d != nil {
			return d
		}
	}
	return bz
}

func (s *CosmosStateDB) SetStorage(contract []byte, key []byte, value []byte) {
	k := s.contractStateKey(contract, key)
	// Store as hex string matching keeper's SetContractState format
	hexVal := hex.EncodeToString(value)
	s.store.Set(k, []byte(hexVal))
}

func (s *CosmosStateDB) GetBalance(addr []byte) *big.Int {
	// Balance queries go through the bank keeper, not the KV store.
	// This is a fallback that returns 0.
	return new(big.Int)
}

func (s *CosmosStateDB) GetCode(addr []byte) []byte {
	// Look up code via the contract record → code hash → bytecode
	// addr is []byte of the address string — use directly (no hex encoding)
	contractKey := append([]byte{}, s.contractPrefix...)
	contractKey = append(contractKey, addr...)
	bz := s.store.Get(contractKey)
	if bz == nil {
		return nil
	}
	var contract struct {
		CodeHash string `json:"code_hash"`
	}
	if err := json.Unmarshal(bz, &contract); err != nil {
		return nil
	}
	codeKey := append([]byte{}, s.codePrefix...)
	codeKey = append(codeKey, []byte(contract.CodeHash)...)
	codeBz := s.store.Get(codeKey)
	if codeBz == nil {
		return nil
	}
	var code struct {
		Bytecode []byte `json:"bytecode"`
	}
	if err := json.Unmarshal(codeBz, &code); err != nil {
		return nil
	}
	return code.Bytecode
}

func (s *CosmosStateDB) GetCodeSize(addr []byte) int {
	return len(s.GetCode(addr))
}

func (s *CosmosStateDB) GetCodeHash(addr []byte) []byte {
	code := s.GetCode(addr)
	if code == nil {
		return nil
	}
	h := sha256.Sum256(code)
	return h[:]
}

func (s *CosmosStateDB) Exists(addr []byte) bool {
	// addr is []byte of the address string — use directly
	contractKey := append([]byte{}, s.contractPrefix...)
	contractKey = append(contractKey, addr...)
	return s.store.Has(contractKey)
}
