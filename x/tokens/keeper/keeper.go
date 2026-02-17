package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/tokens/types"
)

// Keeper manages the tokens module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	bankKeeper types.BankKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new tokens module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bk types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bk,
		authority:    authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// ---------- Params ----------

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	_ = kvStore.Set(types.ParamsKey, bz)
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// ---------- Genesis ----------

// genesisJSON is the combined genesis structure for JSON marshal/unmarshal.
// It includes both the proto GenesisState fields and the hand-written token entries.
type genesisJSON struct {
	Params       *types.Params                `json:"params,omitempty"`
	Accounts     []*types.ExpirableAccount    `json:"accounts,omitempty"`
	Rai          *types.RAIHoldingAccount     `json:"rai,omitempty"`
	TokenEntries []types.TokenGenesisEntry    `json:"token_entries,omitempty"`
	PoolEntries  []types.PoolGenesisEntry     `json:"pool_entries,omitempty"`
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, *genState.Params)
	}

	// The proto GenesisState doesn't have token_entries, so we need to
	// re-unmarshal from the raw JSON to get our hand-written fields.
	// This is called from module.go which passes the already-unmarshaled genState,
	// so we handle token entries via a separate InitGenesisTokens call.
}

// InitGenesisTokens initializes token state from raw genesis JSON.
func (k Keeper) InitGenesisTokens(ctx sdk.Context, data json.RawMessage) {
	if data == nil {
		return
	}
	var g genesisJSON
	if err := json.Unmarshal(data, &g); err != nil {
		return
	}
	for i := range g.TokenEntries {
		entry := &g.TokenEntries[i]
		k.SetToken(ctx, entry.Token)

		for ownerAddr, amtStr := range entry.Balances {
			bal := new(big.Int)
			if _, ok := bal.SetString(amtStr, 10); ok && bal.Sign() > 0 {
				k.SetBalance(ctx, entry.Token.Id, ownerAddr, bal)
			}
		}

		for key, amtStr := range entry.Allowances {
			parts := strings.SplitN(key, "/", 2)
			if len(parts) != 2 {
				continue
			}
			al := new(big.Int)
			if _, ok := al.SetString(amtStr, 10); ok && al.Sign() > 0 {
				k.SetAllowance(ctx, entry.Token.Id, parts[0], parts[1], al)
			}
		}
	}
}

// InitGenesisPools initializes pool and LP position state from raw genesis JSON.
func (k Keeper) InitGenesisPools(ctx sdk.Context, data json.RawMessage) {
	if data == nil {
		return
	}
	var g genesisJSON
	if err := json.Unmarshal(data, &g); err != nil {
		return
	}
	for i := range g.PoolEntries {
		entry := &g.PoolEntries[i]
		k.SetPool(ctx, entry.Pool)

		for providerAddr, amtStr := range entry.Positions {
			lp := new(big.Int)
			if _, ok := lp.SetString(amtStr, 10); ok && lp.Sign() > 0 {
				k.SetLPPosition(ctx, entry.Pool.Id, providerAddr, lp)
			}
		}
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	p := k.GetParams(ctx)
	return &types.GenesisState{
		Params:   &p,
		Accounts: []*types.ExpirableAccount{},
		Rai: &types.RAIHoldingAccount{
			Balance:          "0",
			TotalReceived:    "0",
			TotalDistributed: "0",
			LastDistBlock:    0,
		},
	}
}

// ExportGenesisJSON exports the full genesis state including token entries as raw JSON.
func (k Keeper) ExportGenesisJSON(ctx sdk.Context) json.RawMessage {
	p := k.GetParams(ctx)

	var tokenEntries []types.TokenGenesisEntry
	k.IterateTokens(ctx, func(token *types.TokenDefinition) bool {
		entry := types.TokenGenesisEntry{
			Token:      token,
			Balances:   make(map[string]string),
			Allowances: make(map[string]string),
		}

		k.IterateBalancesByToken(ctx, token.Id, func(ownerAddr string, balance *big.Int) bool {
			entry.Balances[ownerAddr] = balance.String()
			return false
		})

		// Iterate all allowances for this token by scanning the full allowance prefix
		kvStore := k.storeService.OpenKVStore(ctx)
		tokenPrefix := append(types.AllowanceKeyPrefix, []byte(token.Id+"/")...)
		iter, err := kvStore.Iterator(tokenPrefix, prefixEndBytes(tokenPrefix))
		if err == nil {
			defer iter.Close()
			for ; iter.Valid(); iter.Next() {
				// Key format: AllowanceKeyPrefix + tokenId + "/" + ownerAddr + "/" + spenderAddr
				suffix := string(iter.Key()[len(tokenPrefix):])
				al := new(big.Int)
				if _, ok := al.SetString(string(iter.Value()), 10); ok && al.Sign() > 0 {
					entry.Allowances[suffix] = al.String()
				}
			}
		}

		tokenEntries = append(tokenEntries, entry)
		return false
	})

	var poolEntries []types.PoolGenesisEntry
	k.IteratePools(ctx, func(pool *types.LiquidityPool) bool {
		entry := types.PoolGenesisEntry{
			Pool:      pool,
			Positions: make(map[string]string),
		}

		k.IterateLPPositionsByPool(ctx, pool.Id, func(providerAddr string, lpTokens *big.Int) bool {
			entry.Positions[providerAddr] = lpTokens.String()
			return false
		})

		poolEntries = append(poolEntries, entry)
		return false
	})

	g := genesisJSON{
		Params: &p,
		Accounts: []*types.ExpirableAccount{},
		Rai: &types.RAIHoldingAccount{
			Balance:          "0",
			TotalReceived:    "0",
			TotalDistributed: "0",
			LastDistBlock:    0,
		},
		TokenEntries: tokenEntries,
		PoolEntries:  poolEntries,
	}

	bz, err := json.Marshal(g)
	if err != nil {
		panic("failed to marshal genesis: " + err.Error())
	}
	return bz
}
