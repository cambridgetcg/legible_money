package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tokens/types"
)

// ---------- ExpirableAccount Operations ----------

// SetExpirableAccount stores an expirable account in the KV store.
func (k Keeper) SetExpirableAccount(ctx sdk.Context, acct types.ExpirableAccount) {
	store := ctx.KVStore(k.storeKey)
	key := accountKey(acct.Address)
	bz, err := json.Marshal(acct)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal expirable account: %v", err))
	}
	store.Set(key, bz)
}

// GetExpirableAccount retrieves an expirable account by address.
func (k Keeper) GetExpirableAccount(ctx sdk.Context, address string) (types.ExpirableAccount, bool) {
	store := ctx.KVStore(k.storeKey)
	key := accountKey(address)
	bz := store.Get(key)
	if bz == nil {
		return types.ExpirableAccount{}, false
	}
	var acct types.ExpirableAccount
	if err := json.Unmarshal(bz, &acct); err != nil {
		return types.ExpirableAccount{}, false
	}
	return acct, true
}

// DeleteExpirableAccount removes an expirable account from the store.
func (k Keeper) DeleteExpirableAccount(ctx sdk.Context, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(accountKey(address))
}

// GetAllExpirableAccounts returns all expirable accounts.
func (k Keeper) GetAllExpirableAccounts(ctx sdk.Context) []types.ExpirableAccount {
	var accounts []types.ExpirableAccount
	k.IterateExpirableAccounts(ctx, func(acct types.ExpirableAccount) bool {
		accounts = append(accounts, acct)
		return false
	})
	return accounts
}

// IterateExpirableAccounts iterates over all expirable accounts.
// Return true from cb to stop.
func (k Keeper) IterateExpirableAccounts(ctx sdk.Context, cb func(types.ExpirableAccount) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.BalanceKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var acct types.ExpirableAccount
		if err := json.Unmarshal(iter.Value(), &acct); err != nil {
			continue
		}
		if cb(acct) {
			break
		}
	}
}

// ---------- Allowance Operations ----------

// SetAllowance sets a spending allowance for (owner, spender).
func (k Keeper) SetAllowance(ctx sdk.Context, owner, spender, amount string) {
	store := ctx.KVStore(k.storeKey)
	key := allowanceKey(owner, spender)
	store.Set(key, []byte(amount))
}

// GetAllowance returns the spending allowance for (owner, spender).
// Returns "0" if no allowance is set.
func (k Keeper) GetAllowance(ctx sdk.Context, owner, spender string) string {
	store := ctx.KVStore(k.storeKey)
	key := allowanceKey(owner, spender)
	bz := store.Get(key)
	if bz == nil {
		return "0"
	}
	return string(bz)
}

// DeleteAllowance removes a spending allowance.
func (k Keeper) DeleteAllowance(ctx sdk.Context, owner, spender string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(allowanceKey(owner, spender))
}

// ---------- RAI Account Operations ----------

// SetRAIAccount stores the RAI holding account.
func (k Keeper) SetRAIAccount(ctx sdk.Context, rai types.RAIHoldingAccount) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(rai)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal RAI account: %v", err))
	}
	store.Set(types.RAIAccountKey, bz)
}

// GetRAIAccount returns the RAI holding account.
func (k Keeper) GetRAIAccount(ctx sdk.Context) types.RAIHoldingAccount {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.RAIAccountKey)
	if bz == nil {
		return types.RAIHoldingAccount{
			Balance:          "0",
			TotalReceived:    "0",
			TotalDistributed: "0",
			LastDistBlock:    0,
		}
	}
	var rai types.RAIHoldingAccount
	if err := json.Unmarshal(bz, &rai); err != nil {
		return types.RAIHoldingAccount{
			Balance:          "0",
			TotalReceived:    "0",
			TotalDistributed: "0",
			LastDistBlock:    0,
		}
	}
	return rai
}

// ---------- Sweep Index Operations ----------

// AddToSweepIndex adds an address to the sweep index for a given period.
func (k Keeper) AddToSweepIndex(ctx sdk.Context, period uint64, address string) {
	store := ctx.KVStore(k.storeKey)
	key := sweepIndexKey(period, address)
	store.Set(key, []byte(address))
}

// GetSweepAddresses returns addresses that need sweeping for a given period.
func (k Keeper) GetSweepAddresses(ctx sdk.Context, period uint64) []string {
	store := ctx.KVStore(k.storeKey)
	prefix := sweepIndexPrefixForPeriod(period)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var addresses []string
	for ; iter.Valid(); iter.Next() {
		addresses = append(addresses, string(iter.Value()))
	}
	return addresses
}

// RemoveFromSweepIndex removes an address from the sweep index for a period.
func (k Keeper) RemoveFromSweepIndex(ctx sdk.Context, period uint64, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(sweepIndexKey(period, address))
}

// IterateExpiredPeriods iterates sweep index entries for periods that have expired
// relative to the current period. Return true from cb to stop.
func (k Keeper) IterateExpiredPeriods(ctx sdk.Context, currentPeriod uint64, periodsUntilExpiry uint64, cb func(period uint64, address string) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.SweepIndexPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		// key format: SweepIndexPrefix + 8-byte period + address
		if len(key) < len(types.SweepIndexPrefix)+8 {
			continue
		}
		periodBytes := key[len(types.SweepIndexPrefix) : len(types.SweepIndexPrefix)+8]
		period := binary.BigEndian.Uint64(periodBytes)

		if !k.IsExpired(period, currentPeriod, periodsUntilExpiry) {
			continue
		}

		address := string(iter.Value())
		if cb(period, address) {
			break
		}
	}
}

// ---------- Key Construction Helpers ----------

func accountKey(address string) []byte {
	return append(types.BalanceKeyPrefix, []byte(address)...)
}

func allowanceKey(owner, spender string) []byte {
	return append(types.AllowanceKeyPrefix, []byte(owner+"/"+spender)...)
}

func sweepIndexPrefixForPeriod(period uint64) []byte {
	periodBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(periodBytes, period)
	return append(types.SweepIndexPrefix, periodBytes...)
}

func sweepIndexKey(period uint64, address string) []byte {
	prefix := sweepIndexPrefixForPeriod(period)
	return append(prefix, []byte(address)...)
}
