package types

const (
	// ModuleName defines the module name.
	ModuleName = "tokens"

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
	ParamsKey              = []byte{0x00}
	TokenDefKeyPrefix      = []byte{0x01} // {tokenId} -> TokenDefinition (proto)
	BalanceKeyPrefix       = []byte{0x02} // {tokenId}/{ownerAddr} -> big.Int string
	AllowanceKeyPrefix     = []byte{0x03} // {tokenId}/{ownerAddr}/{spenderAddr} -> big.Int string
	LPPoolKeyPrefix        = []byte{0x04} // {pool_id} -> LPPool (LGM-20)
	OwnerTokenIndexPrefix  = []byte{0x05} // {ownerAddr}/{tokenId} -> []byte{1} (exists flag)
	SymbolIndexPrefix      = []byte{0x06} // {symbol} -> tokenId string
	LPPositionKeyPrefix    = []byte{0x07} // {pool_id}/{provider_addr} -> big.Int string (LP tokens held)
	PoolPairIndexPrefix    = []byte{0x08} // {sorted_token_pair} -> pool_id string
)

// TokenDefKey returns the store key for a token definition.
func TokenDefKey(tokenId string) []byte {
	return append(TokenDefKeyPrefix, []byte(tokenId)...)
}

// BalanceKey returns the store key for a specific balance.
func BalanceKey(tokenId, ownerAddr string) []byte {
	return append(BalanceKeyPrefix, []byte(tokenId+"/"+ownerAddr)...)
}

// BalancesByTokenPrefix returns the prefix for iterating all balances of a token.
func BalancesByTokenPrefix(tokenId string) []byte {
	return append(BalanceKeyPrefix, []byte(tokenId+"/")...)
}

// AllowanceKey returns the store key for a specific allowance.
func AllowanceKey(tokenId, ownerAddr, spenderAddr string) []byte {
	return append(AllowanceKeyPrefix, []byte(tokenId+"/"+ownerAddr+"/"+spenderAddr)...)
}

// AllowancesByOwnerPrefix returns the prefix for iterating all allowances granted by an owner for a token.
func AllowancesByOwnerPrefix(tokenId, ownerAddr string) []byte {
	return append(AllowanceKeyPrefix, []byte(tokenId+"/"+ownerAddr+"/")...)
}

// OwnerTokenIndexKey returns the store key for the owner-token index entry.
func OwnerTokenIndexKey(ownerAddr, tokenId string) []byte {
	return append(OwnerTokenIndexPrefix, []byte(ownerAddr+"/"+tokenId)...)
}

// OwnerTokensPrefix returns the prefix for iterating all tokens owned by an address.
func OwnerTokensPrefix(ownerAddr string) []byte {
	return append(OwnerTokenIndexPrefix, []byte(ownerAddr+"/")...)
}

// SymbolIndexKey returns the store key for the symbol-to-tokenId index.
func SymbolIndexKey(symbol string) []byte {
	return append(SymbolIndexPrefix, []byte(symbol)...)
}

// LPPoolKey returns the store key for a liquidity pool.
func LPPoolKey(poolId string) []byte {
	return append(LPPoolKeyPrefix, []byte(poolId)...)
}

// LPPositionKey returns the store key for a provider's LP position in a pool.
func LPPositionKey(poolId, providerAddr string) []byte {
	return append(LPPositionKeyPrefix, []byte(poolId+"/"+providerAddr)...)
}

// LPPositionsByPoolPrefix returns the prefix for iterating all positions in a pool.
func LPPositionsByPoolPrefix(poolId string) []byte {
	return append(LPPositionKeyPrefix, []byte(poolId+"/")...)
}

// PoolPairIndexKey returns the store key for the token-pair-to-poolId index.
// tokenA and tokenB are sorted lexicographically to ensure one pool per pair.
func PoolPairIndexKey(tokenAId, tokenBId string) []byte {
	a, b := tokenAId, tokenBId
	if a > b {
		a, b = b, a
	}
	return append(PoolPairIndexPrefix, []byte(a+"/"+b)...)
}
