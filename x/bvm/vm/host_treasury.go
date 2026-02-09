package vm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// TreasuryHost bridges BVM treasury/guardian/access control opcodes to contract state.
// Treasury positions, alerts, defenses, and access keys are stored under reserved prefixes.
type TreasuryHost struct {
	stateDB StateDB
}

func NewTreasuryHost(stateDB StateDB) *TreasuryHost {
	return &TreasuryHost{stateDB: stateDB}
}

// --- Treasury (0xD0-0xD7) ---

func (h *TreasuryHost) ClaimVesting(callerDID string, vestingId []byte) *big.Int {
	if callerDID == "" || len(vestingId) == 0 {
		return nil
	}
	// Check if vesting position exists and is claimable
	key := sha256.Sum256([]byte("vesting/" + callerDID + "/" + hex.EncodeToString(vestingId)))
	val := h.stateDB.GetStorage([]byte("agent_treasury"), key[:])
	if val == nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(val)
}

func (h *TreasuryHost) QueryVesting(callerDID string, offset, limit uint64) (uint64, [][]byte) {
	// In a full implementation, iterate vesting positions.
	// Contract state doesn't support efficient iteration yet.
	return 0, nil
}

func (h *TreasuryHost) StakeTokens(callerDID string, amount *big.Int, validatorDID []byte) []byte {
	if callerDID == "" || amount == nil || amount.Sign() <= 0 {
		return nil
	}
	input := fmt.Sprintf("stake/%s/%s/%x", callerDID, amount.String(), validatorDID)
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}

func (h *TreasuryHost) UnstakeTokens(callerDID string, stakeId []byte) *big.Int {
	if callerDID == "" || len(stakeId) == 0 {
		return nil
	}
	key := sha256.Sum256([]byte("stake/" + callerDID + "/" + hex.EncodeToString(stakeId)))
	val := h.stateDB.GetStorage([]byte("agent_treasury"), key[:])
	if val == nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(val)
}

func (h *TreasuryHost) QueryExpiring(callerDID string, periodId []byte) (*big.Int, uint64) {
	if callerDID == "" {
		return nil, 0
	}
	key := sha256.Sum256([]byte("expiring/" + callerDID + "/" + hex.EncodeToString(periodId)))
	val := h.stateDB.GetStorage([]byte("agent_treasury"), key[:])
	if val == nil {
		return big.NewInt(0), 0
	}
	amount := new(big.Int).SetBytes(val)
	// Expiration block stored in a companion key
	expiryKey := sha256.Sum256([]byte("expiry/" + callerDID + "/" + hex.EncodeToString(periodId)))
	expiryVal := h.stateDB.GetStorage([]byte("agent_treasury"), expiryKey[:])
	var nextExpiry uint64
	if expiryVal != nil {
		nextExpiry = new(big.Int).SetBytes(expiryVal).Uint64()
	}
	return amount, nextExpiry
}

func (h *TreasuryHost) ExtendExpiring(callerDID string, periodId []byte, newExpiry uint64) bool {
	if callerDID == "" || len(periodId) == 0 || newExpiry == 0 {
		return false
	}
	expiryKey := sha256.Sum256([]byte("expiry/" + callerDID + "/" + hex.EncodeToString(periodId)))
	val := new(big.Int).SetUint64(newExpiry)
	b32 := WordToBytes32(val)
	h.stateDB.SetStorage([]byte("agent_treasury"), expiryKey[:], b32[:])
	return true
}

func (h *TreasuryHost) SetSpendLimit(callerDID string, limitType, amount []byte, period uint64) bool {
	if callerDID == "" || len(limitType) == 0 {
		return false
	}
	key := sha256.Sum256([]byte("spend_limit/" + callerDID + "/" + hex.EncodeToString(limitType)))
	b32 := WordToBytes32(new(big.Int).SetBytes(amount))
	h.stateDB.SetStorage([]byte("agent_treasury"), key[:], b32[:])
	return true
}

func (h *TreasuryHost) QueryTreasury(callerDID string) (*big.Int, *big.Int, *big.Int, *big.Int) {
	if callerDID == "" {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)
	}
	load := func(prefix string) *big.Int {
		key := sha256.Sum256([]byte(prefix + "/" + callerDID))
		val := h.stateDB.GetStorage([]byte("agent_treasury"), key[:])
		if val == nil {
			return big.NewInt(0)
		}
		return new(big.Int).SetBytes(val)
	}
	return load("available"), load("reserved"), load("staked"), load("delegated")
}

// --- Guardian (0xD8-0xDB) ---

func (h *TreasuryHost) QueryAlerts(callerDID string, offset, limit uint64) (uint64, [][]byte) {
	// Full implementation requires iteration over alert keys.
	return 0, nil
}

func (h *TreasuryHost) DismissAlert(callerDID string, alertId []byte) bool {
	if callerDID == "" || len(alertId) == 0 {
		return false
	}
	key := sha256.Sum256([]byte("alert/" + callerDID + "/" + hex.EncodeToString(alertId)))
	h.stateDB.SetStorage([]byte("agent_guardian"), key[:], nil)
	return true
}

func (h *TreasuryHost) SetDefense(callerDID string, defenseType, config, threshold []byte) bool {
	if callerDID == "" || len(defenseType) == 0 {
		return false
	}
	key := sha256.Sum256([]byte("defense/" + callerDID + "/" + hex.EncodeToString(defenseType)))
	b32 := WordToBytes32(new(big.Int).SetBytes(config))
	h.stateDB.SetStorage([]byte("agent_guardian"), key[:], b32[:])
	return true
}

func (h *TreasuryHost) QueryDefenses(callerDID string) (uint64, [][]byte) {
	// Full implementation requires iteration.
	return 0, nil
}

// --- Access Control (0xDC-0xDF) ---

func (h *TreasuryHost) RegisterKey(callerDID string, keyType, publicKey, permissions, expiry []byte) bool {
	if callerDID == "" || len(publicKey) == 0 {
		return false
	}
	keyHash := sha256.Sum256(publicKey)
	storeKey := sha256.Sum256([]byte("key/" + callerDID + "/" + hex.EncodeToString(keyHash[:])))
	b32 := WordToBytes32(new(big.Int).SetBytes(permissions))
	h.stateDB.SetStorage([]byte("agent_access"), storeKey[:], b32[:])
	return true
}

func (h *TreasuryHost) RevokeKey(callerDID string, keyId []byte) bool {
	if callerDID == "" || len(keyId) == 0 {
		return false
	}
	key := sha256.Sum256([]byte("key/" + callerDID + "/" + hex.EncodeToString(keyId)))
	h.stateDB.SetStorage([]byte("agent_access"), key[:], nil)
	return true
}

func (h *TreasuryHost) QueryPermissions(callerDID string, role []byte) *big.Int {
	if callerDID == "" {
		return big.NewInt(0)
	}
	key := sha256.Sum256([]byte("perm/" + callerDID + "/" + hex.EncodeToString(role)))
	val := h.stateDB.GetStorage([]byte("agent_access"), key[:])
	if val == nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(val)
}

func (h *TreasuryHost) QuerySession(callerDID string) ([]byte, []byte, uint64, uint64) {
	if callerDID == "" {
		return nil, nil, 0, 0
	}
	key := sha256.Sum256([]byte("session/" + callerDID))
	val := h.stateDB.GetStorage([]byte("agent_access"), key[:])
	if val == nil {
		return nil, nil, 0, 0
	}
	// Session data packed: first 32 bytes = sessionId hash, role = "default"
	return val, []byte("default"), 0, 0
}

// --- Introspection (0xF6-0xF9) ---

func (h *TreasuryHost) SelfBalance(callerDID string) *big.Int {
	if callerDID == "" {
		return big.NewInt(0)
	}
	key := sha256.Sum256([]byte("available/" + callerDID))
	val := h.stateDB.GetStorage([]byte("agent_treasury"), key[:])
	if val == nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(val)
}

func (h *TreasuryHost) SelfStake(callerDID string) *big.Int {
	if callerDID == "" {
		return big.NewInt(0)
	}
	key := sha256.Sum256([]byte("staked/" + callerDID))
	val := h.stateDB.GetStorage([]byte("agent_treasury"), key[:])
	if val == nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(val)
}

func (h *TreasuryHost) SelfReputation(callerDID string) uint64 {
	if callerDID == "" {
		return 0
	}
	key := sha256.Sum256([]byte("reputation/" + callerDID))
	val := h.stateDB.GetStorage([]byte("agent_treasury"), key[:])
	if val == nil {
		return 0
	}
	return new(big.Int).SetBytes(val).Uint64()
}

func (h *TreasuryHost) SelfAccuracy(callerDID string) uint64 {
	if callerDID == "" {
		return 0
	}
	key := sha256.Sum256([]byte("accuracy/" + callerDID))
	val := h.stateDB.GetStorage([]byte("agent_treasury"), key[:])
	if val == nil {
		return 0
	}
	return new(big.Int).SetBytes(val).Uint64()
}
