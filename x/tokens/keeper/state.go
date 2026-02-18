package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/tokens/types"
)

var symbolRegex = regexp.MustCompile(`^[A-Z0-9]{1,16}$`)

// isValidSymbol checks that a symbol is 1-16 uppercase alphanumeric characters.
func isValidSymbol(symbol string) bool {
	return symbolRegex.MatchString(symbol)
}

// GenerateTokenID produces a deterministic token ID from creator, symbol, and block height.
// Formula: hex(SHA256("LGM.token.id.v1:" + creator + ":" + symbol + ":" + blockHeight))
func GenerateTokenID(creator, symbol string, blockHeight int64) string {
	data := fmt.Sprintf("LGM.token.id.v1:%s:%s:%d", creator, symbol, blockHeight)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ---------- Token CRUD ----------

// SetToken stores a token definition and maintains the symbol index.
func (k Keeper) SetToken(ctx sdk.Context, token *types.TokenDefinition) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(token)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal token: %v", err))
	}
	_ = kvStore.Set(types.TokenDefKey(token.Id), bz)
	// Maintain symbol index
	_ = kvStore.Set(types.SymbolIndexKey(token.Symbol), []byte(token.Id))
}

// GetToken retrieves a token definition by ID. Returns nil if not found.
func (k Keeper) GetToken(ctx sdk.Context, tokenId string) *types.TokenDefinition {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.TokenDefKey(tokenId))
	if err != nil || bz == nil {
		return nil
	}
	var token types.TokenDefinition
	if err := proto.Unmarshal(bz, &token); err != nil {
		return nil
	}
	return &token
}

// GetTokenBySymbol retrieves a token definition by symbol using the symbol index.
func (k Keeper) GetTokenBySymbol(ctx sdk.Context, symbol string) *types.TokenDefinition {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.SymbolIndexKey(symbol))
	if err != nil || bz == nil {
		return nil
	}
	return k.GetToken(ctx, string(bz))
}

// DeleteToken removes a token definition and its symbol index entry.
func (k Keeper) DeleteToken(ctx sdk.Context, token *types.TokenDefinition) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(types.TokenDefKey(token.Id))
	_ = kvStore.Delete(types.SymbolIndexKey(token.Symbol))
}

// IterateTokens iterates over all token definitions. Return true from cb to stop.
func (k Keeper) IterateTokens(ctx sdk.Context, cb func(token *types.TokenDefinition) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.TokenDefKeyPrefix, prefixEndBytes(types.TokenDefKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var token types.TokenDefinition
		if err := proto.Unmarshal(iter.Value(), &token); err != nil {
			continue
		}
		if cb(&token) {
			break
		}
	}
}

// ---------- Balance CRUD ----------

// GetBalance returns the balance of an owner for a token. Returns zero if not found.
func (k Keeper) GetBalance(ctx sdk.Context, tokenId, ownerAddr string) *big.Int {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.BalanceKey(tokenId, ownerAddr))
	if err != nil || bz == nil {
		return new(big.Int)
	}
	bal := new(big.Int)
	if _, ok := bal.SetString(string(bz), 10); !ok {
		return new(big.Int)
	}
	return bal
}

// SetBalance sets the balance of an owner for a token.
// If the balance is zero, deletes the entry and the owner-token index.
func (k Keeper) SetBalance(ctx sdk.Context, tokenId, ownerAddr string, balance *big.Int) {
	kvStore := k.storeService.OpenKVStore(ctx)
	if balance.Sign() == 0 {
		_ = kvStore.Delete(types.BalanceKey(tokenId, ownerAddr))
		_ = kvStore.Delete(types.OwnerTokenIndexKey(ownerAddr, tokenId))
		return
	}
	_ = kvStore.Set(types.BalanceKey(tokenId, ownerAddr), []byte(balance.String()))
	_ = kvStore.Set(types.OwnerTokenIndexKey(ownerAddr, tokenId), []byte{1})
}

// IterateBalancesByToken iterates over all balances for a given token.
// Callback receives (ownerAddr, balance). Return true to stop.
func (k Keeper) IterateBalancesByToken(ctx sdk.Context, tokenId string, cb func(ownerAddr string, balance *big.Int) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.BalancesByTokenPrefix(tokenId)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()

	prefixLen := len(prefix)
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		ownerAddr := string(key[prefixLen:])
		bal := new(big.Int)
		if _, ok := bal.SetString(string(iter.Value()), 10); !ok {
			continue
		}
		if cb(ownerAddr, bal) {
			break
		}
	}
}

// ---------- Allowance CRUD ----------

// GetAllowance returns the allowance granted by owner to spender for a token. Returns zero if not found.
func (k Keeper) GetAllowance(ctx sdk.Context, tokenId, ownerAddr, spenderAddr string) *big.Int {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.AllowanceKey(tokenId, ownerAddr, spenderAddr))
	if err != nil || bz == nil {
		return new(big.Int)
	}
	al := new(big.Int)
	if _, ok := al.SetString(string(bz), 10); !ok {
		return new(big.Int)
	}
	return al
}

// SetAllowance sets the allowance for a spender. Zero deletes the entry.
func (k Keeper) SetAllowance(ctx sdk.Context, tokenId, ownerAddr, spenderAddr string, allowance *big.Int) {
	kvStore := k.storeService.OpenKVStore(ctx)
	if allowance.Sign() == 0 {
		_ = kvStore.Delete(types.AllowanceKey(tokenId, ownerAddr, spenderAddr))
		return
	}
	_ = kvStore.Set(types.AllowanceKey(tokenId, ownerAddr, spenderAddr), []byte(allowance.String()))
}

// IterateAllowancesByOwner iterates over all allowances granted by an owner for a token.
// Callback receives (spenderAddr, allowance). Return true to stop.
func (k Keeper) IterateAllowancesByOwner(ctx sdk.Context, tokenId, ownerAddr string, cb func(spenderAddr string, allowance *big.Int) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.AllowancesByOwnerPrefix(tokenId, ownerAddr)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()

	prefixLen := len(prefix)
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		spenderAddr := string(key[prefixLen:])
		al := new(big.Int)
		if _, ok := al.SetString(string(iter.Value()), 10); !ok {
			continue
		}
		if cb(spenderAddr, al) {
			break
		}
	}
}

// ---------- Pool CRUD ----------

// GeneratePoolID produces a deterministic pool ID from a sorted token pair.
// Formula: hex(SHA256("LGM.pool.v1:" + sorted(A,B)[0] + ":" + sorted(A,B)[1]))
// One pool per token pair, regardless of submission order.
func GeneratePoolID(tokenAId, tokenBId string) string {
	a, b := tokenAId, tokenBId
	if a > b {
		a, b = b, a
	}
	data := fmt.Sprintf("LGM.pool.v1:%s:%s", a, b)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// SetPool stores a liquidity pool and maintains the token-pair index.
func (k Keeper) SetPool(ctx sdk.Context, pool *types.LiquidityPool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(pool)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal pool: %v", err))
	}
	_ = kvStore.Set(types.LPPoolKey(pool.Id), bz)
	_ = kvStore.Set(types.PoolPairIndexKey(pool.TokenAId, pool.TokenBId), []byte(pool.Id))
}

// GetPool retrieves a liquidity pool by ID. Returns nil if not found.
func (k Keeper) GetPool(ctx sdk.Context, poolId string) *types.LiquidityPool {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.LPPoolKey(poolId))
	if err != nil || bz == nil {
		return nil
	}
	var pool types.LiquidityPool
	if err := proto.Unmarshal(bz, &pool); err != nil {
		return nil
	}
	return &pool
}

// GetPoolByTokenPair retrieves a pool by its token pair (order-independent).
func (k Keeper) GetPoolByTokenPair(ctx sdk.Context, tokenAId, tokenBId string) *types.LiquidityPool {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.PoolPairIndexKey(tokenAId, tokenBId))
	if err != nil || bz == nil {
		return nil
	}
	return k.GetPool(ctx, string(bz))
}

// ---------- LP Position CRUD ----------

// GetLPPosition returns a provider's LP token balance for a pool. Returns zero if not found.
func (k Keeper) GetLPPosition(ctx sdk.Context, poolId, providerAddr string) *big.Int {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.LPPositionKey(poolId, providerAddr))
	if err != nil || bz == nil {
		return new(big.Int)
	}
	pos := new(big.Int)
	if _, ok := pos.SetString(string(bz), 10); !ok {
		return new(big.Int)
	}
	return pos
}

// SetLPPosition sets a provider's LP token balance for a pool. Zero deletes the entry.
func (k Keeper) SetLPPosition(ctx sdk.Context, poolId, providerAddr string, amount *big.Int) {
	kvStore := k.storeService.OpenKVStore(ctx)
	if amount.Sign() == 0 {
		_ = kvStore.Delete(types.LPPositionKey(poolId, providerAddr))
		return
	}
	_ = kvStore.Set(types.LPPositionKey(poolId, providerAddr), []byte(amount.String()))
}

// IteratePools iterates over all liquidity pools. Return true from cb to stop.
func (k Keeper) IteratePools(ctx sdk.Context, cb func(pool *types.LiquidityPool) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.LPPoolKeyPrefix, prefixEndBytes(types.LPPoolKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var pool types.LiquidityPool
		if err := proto.Unmarshal(iter.Value(), &pool); err != nil {
			continue
		}
		if cb(&pool) {
			break
		}
	}
}

// IterateLPPositionsByPool iterates over all LP positions for a pool.
// Callback receives (providerAddr, lpTokens). Return true to stop.
func (k Keeper) IterateLPPositionsByPool(ctx sdk.Context, poolId string, cb func(providerAddr string, lpTokens *big.Int) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.LPPositionsByPoolPrefix(poolId)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()

	prefixLen := len(prefix)
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		providerAddr := string(key[prefixLen:])
		lp := new(big.Int)
		if _, ok := lp.SetString(string(iter.Value()), 10); !ok {
			continue
		}
		if cb(providerAddr, lp) {
			break
		}
	}
}

// ---------- Helpers ----------

// prefixEndBytes returns the end key for prefix iteration (prefix with last byte incremented).
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil // overflow — all 0xFF bytes
}
