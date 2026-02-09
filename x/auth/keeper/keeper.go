package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/auth/types"
)

// Keeper manages Legible Money account state with 4-layer key architecture
type Keeper struct {
	cdc           codec.Codec
	storeKey      storetypes.StoreKey
	accountKeeper types.CosmosAccountKeeper
	bankKeeper    types.BankKeeper
	authority     string // module authority (gov)
}

// NewKeeper creates a new Keeper instance
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	accountKeeper types.CosmosAccountKeeper,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
		authority:     authority,
	}
}

// Logger returns a module-specific logger
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// SetAccount stores a Legible account
func (k Keeper) SetAccount(ctx sdk.Context, account types.LegibleAccount) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(account)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal account: %v", err))
	}
	store.Set(types.AccountKey(account.Address), bz)
}

// GetAccount retrieves a Legible account by bech32 address
func (k Keeper) GetAccount(ctx sdk.Context, address string) (types.LegibleAccount, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AccountKey(address))
	if bz == nil {
		return types.LegibleAccount{}, false
	}

	var account types.LegibleAccount
	if err := json.Unmarshal(bz, &account); err != nil {
		return types.LegibleAccount{}, false
	}
	return account, true
}

// GetAccountByDID retrieves a Legible account by DID
func (k Keeper) GetAccountByDID(ctx sdk.Context, did string) (types.LegibleAccount, bool) {
	address, found := k.GetAddressForDID(ctx, did)
	if !found {
		return types.LegibleAccount{}, false
	}
	return k.GetAccount(ctx, address)
}

// SetDIDMapping stores a DID -> bech32 mapping
func (k Keeper) SetDIDMapping(ctx sdk.Context, mapping types.DIDMapping) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(mapping)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal DID mapping: %v", err))
	}
	store.Set(types.DIDMappingKey(mapping.Did), bz)
}

// GetDIDMapping retrieves a DID mapping
func (k Keeper) GetDIDMapping(ctx sdk.Context, did string) (types.DIDMapping, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.DIDMappingKey(did))
	if bz == nil {
		return types.DIDMapping{}, false
	}

	var mapping types.DIDMapping
	if err := json.Unmarshal(bz, &mapping); err != nil {
		return types.DIDMapping{}, false
	}
	return mapping, true
}

// GetAddressForDID returns the bech32 address for a DID
func (k Keeper) GetAddressForDID(ctx sdk.Context, did string) (string, bool) {
	mapping, found := k.GetDIDMapping(ctx, did)
	if !found {
		return "", false
	}
	return mapping.Bech32, true
}

// SetSessionKey stores a session key
func (k Keeper) SetSessionKey(ctx sdk.Context, session types.SessionKey) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(session)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal session key: %v", err))
	}
	store.Set(types.SessionKeyKey(session.Owner, session.KeyHash), bz)
}

// GetSessionKey retrieves a session key
func (k Keeper) GetSessionKey(ctx sdk.Context, owner, keyHash string) (types.SessionKey, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SessionKeyKey(owner, keyHash))
	if bz == nil {
		return types.SessionKey{}, false
	}

	var session types.SessionKey
	if err := json.Unmarshal(bz, &session); err != nil {
		return types.SessionKey{}, false
	}
	return session, true
}

// DeleteSessionKey removes a session key
func (k Keeper) DeleteSessionKey(ctx sdk.Context, owner, keyHash string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.SessionKeyKey(owner, keyHash))
}

// GetSessionKeysForOwner returns all session keys for an owner
func (k Keeper) GetSessionKeysForOwner(ctx sdk.Context, owner string) []types.SessionKey {
	store := ctx.KVStore(k.storeKey)
	prefix := types.SessionKeyOwnerPrefix(owner)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var sessions []types.SessionKey
	for ; iterator.Valid(); iterator.Next() {
		var session types.SessionKey
		if err := json.Unmarshal(iterator.Value(), &session); err != nil {
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions
}

// IsSessionValid checks if a session key is valid at the given height
func (k Keeper) IsSessionValid(ctx sdk.Context, owner, keyHash string, height uint64) bool {
	session, found := k.GetSessionKey(ctx, owner, keyHash)
	if !found {
		return false
	}
	return height < session.ExpiresAtBlock
}

// CountSessionKeys returns the number of active session keys for an owner
func (k Keeper) CountSessionKeys(ctx sdk.Context, owner string) uint32 {
	sessions := k.GetSessionKeysForOwner(ctx, owner)
	height := uint64(ctx.BlockHeight())

	count := uint32(0)
	for _, session := range sessions {
		if session.ExpiresAtBlock > height {
			count++
		}
	}
	return count
}

// SetParams sets module parameters
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}
	store.Set(types.ParamsKey, bz)
	return nil
}

// GetParams retrieves module parameters
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}

	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetLastRotation stores the block height of last key rotation
func (k Keeper) SetLastRotation(ctx sdk.Context, address string, height uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.LastRotationKey(address), types.Uint64ToBytes(height))
}

// GetLastRotation retrieves the block height of last key rotation
func (k Keeper) GetLastRotation(ctx sdk.Context, address string) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LastRotationKey(address))
	if bz == nil {
		return 0
	}
	return types.BytesToUint64(bz)
}

// SetRecoveryConfig stores recovery configuration for an account
func (k Keeper) SetRecoveryConfig(ctx sdk.Context, address string, config types.RecoveryConfig) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(config)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal recovery config: %v", err))
	}
	store.Set(types.RecoveryConfigKey(address), bz)
}

// GetRecoveryConfig retrieves recovery configuration for an account
func (k Keeper) GetRecoveryConfig(ctx sdk.Context, address string) (types.RecoveryConfig, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.RecoveryConfigKey(address))
	if bz == nil {
		return types.RecoveryConfig{}, false
	}
	var config types.RecoveryConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return types.RecoveryConfig{}, false
	}
	return config, true
}

// SetRecoveryRequest stores a recovery request
func (k Keeper) SetRecoveryRequest(ctx sdk.Context, req types.RecoveryRequest) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(req)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal recovery request: %v", err))
	}
	store.Set(types.RecoveryRequestKey(req.AccountAddress), bz)
}

// GetRecoveryRequest retrieves a recovery request by account address
func (k Keeper) GetRecoveryRequest(ctx sdk.Context, address string) (types.RecoveryRequest, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.RecoveryRequestKey(address))
	if bz == nil {
		return types.RecoveryRequest{}, false
	}
	var req types.RecoveryRequest
	if err := json.Unmarshal(bz, &req); err != nil {
		return types.RecoveryRequest{}, false
	}
	return req, true
}

// DeleteRecoveryRequest removes a recovery request
func (k Keeper) DeleteRecoveryRequest(ctx sdk.Context, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.RecoveryRequestKey(address))
}

// IterateRecoveryRequests iterates over all recovery requests
func (k Keeper) IterateRecoveryRequests(ctx sdk.Context, cb func(types.RecoveryRequest) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.RecoveryRequestPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var req types.RecoveryRequest
		if err := json.Unmarshal(iterator.Value(), &req); err != nil {
			continue
		}
		if cb(req) {
			break
		}
	}
}

// CleanupExpiredSessions removes expired session keys and decrements account counters.
// Called from BeginBlocker.
func (k Keeper) CleanupExpiredSessions(ctx sdk.Context) {
	currentHeight := uint64(ctx.BlockHeight())
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.SessionKeyPrefix)
	defer iterator.Close()

	type deletion struct {
		key   []byte
		owner string
	}
	var toDelete []deletion

	for ; iterator.Valid(); iterator.Next() {
		var session types.SessionKey
		if err := json.Unmarshal(iterator.Value(), &session); err != nil {
			continue
		}
		if session.ExpiresAtBlock <= currentHeight {
			keyCopy := make([]byte, len(iterator.Key()))
			copy(keyCopy, iterator.Key())
			toDelete = append(toDelete, deletion{key: keyCopy, owner: session.Owner})
		}
	}

	// Apply deletions outside of iteration
	for _, d := range toDelete {
		store.Delete(d.key)
		// Decrement session key count on account
		if account, found := k.GetAccount(ctx, d.owner); found {
			if account.SessionKeyCount > 0 {
				account.SessionKeyCount--
				k.SetAccount(ctx, account)
			}
		}
	}
}

// ProcessRecoveryTimeouts advances recovery request statuses based on block height.
// Called from BeginBlocker.
func (k Keeper) ProcessRecoveryTimeouts(ctx sdk.Context) {
	currentHeight := uint64(ctx.BlockHeight())

	k.IterateRecoveryRequests(ctx, func(req types.RecoveryRequest) bool {
		changed := false

		switch req.Status {
		case "delayed":
			if currentHeight >= req.DelayExpiresAt {
				req.Status = "challengeable"
				changed = true
			}
		case "challengeable":
			if currentHeight >= req.ChallengeExpiresAt {
				req.Status = "executable"
				changed = true
			}
		}

		if changed {
			k.SetRecoveryRequest(ctx, req)
		}
		return false
	})
}

// GetAuthority returns the module authority address
func (k Keeper) GetAuthority() string {
	return k.authority
}

// IterateAccounts iterates over all accounts
func (k Keeper) IterateAccounts(ctx sdk.Context, cb func(types.LegibleAccount) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.AccountKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var account types.LegibleAccount
		if err := json.Unmarshal(iterator.Value(), &account); err != nil {
			continue
		}
		if cb(account) {
			break
		}
	}
}

// InitGenesis initializes the module state from genesis
func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) error {
	// Set params
	if err := k.SetParams(ctx, data.Params); err != nil {
		return fmt.Errorf("failed to set params: %w", err)
	}

	// Set accounts
	for _, account := range data.Accounts {
		k.SetAccount(ctx, account)
	}

	// Set DID mappings
	for _, mapping := range data.DIDMappings {
		k.SetDIDMapping(ctx, mapping)
	}

	// Set session keys
	for _, session := range data.SessionKeys {
		k.SetSessionKey(ctx, session)
	}

	return nil
}

// ExportGenesis exports the module state for genesis
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)

	// Collect all accounts
	var accounts []types.LegibleAccount
	k.IterateAccounts(ctx, func(account types.LegibleAccount) bool {
		accounts = append(accounts, account)
		return false
	})

	// Collect all DID mappings
	var mappings []types.DIDMapping
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.DIDMappingPrefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var mapping types.DIDMapping
		if err := json.Unmarshal(iterator.Value(), &mapping); err != nil {
			continue
		}
		mappings = append(mappings, mapping)
	}

	// Collect all session keys
	var sessions []types.SessionKey
	sessionIterator := storetypes.KVStorePrefixIterator(store, types.SessionKeyPrefix)
	defer sessionIterator.Close()
	for ; sessionIterator.Valid(); sessionIterator.Next() {
		var session types.SessionKey
		if err := json.Unmarshal(sessionIterator.Value(), &session); err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	return &types.GenesisState{
		Params:      params,
		Accounts:    accounts,
		DIDMappings: mappings,
		SessionKeys: sessions,
	}
}
