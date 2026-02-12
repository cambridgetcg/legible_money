package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/partnerships/types"
)

// Keeper manages the partnerships module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new partnerships module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bk types.BankKeeper,
	sk types.StakingKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		bankKeeper:    bk,
		stakingKeeper: sk,
		authority:     authority,
	}
}

// prefixEndBytes returns the end key for prefix iteration (exclusive).
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
	return nil
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
	bz, err := json.Marshal(params)
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
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// ---------- Formation Expiry ----------

// ExpireFormations dissolves partnerships whose formation window has passed.
func (k Keeper) ExpireFormations(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())
	kvStore := k.storeService.OpenKVStore(ctx)

	iter, err := kvStore.Iterator(types.FormationKeyPrefix, prefixEndBytes(types.FormationKeyPrefix))
	if err != nil {
		return
	}

	var toDelete [][]byte
	for ; iter.Valid(); iter.Next() {
		var expiry uint64
		if _, err := fmt.Sscanf(string(iter.Value()), "%d", &expiry); err != nil {
			continue
		}
		if currentBlock > expiry {
			key := iter.Key()
			partnershipId := string(key[len(types.FormationKeyPrefix):])
			if p, found := k.GetPartnership(ctx, partnershipId); found {
				if p.Status == types.StatusForming {
					p.Status = types.StatusDissolved
					k.SetPartnership(ctx, p)
				}
			}
			toDelete = append(toDelete, append([]byte{}, key...))
		}
	}
	iter.Close()
	for _, key := range toDelete {
		_ = kvStore.Delete(key)
	}
}

// ---------- Genesis ----------

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)

	for _, p := range genState.Partnerships {
		k.SetPartnership(ctx, p)
	}
	for _, op := range genState.ConsensusOperations {
		k.SetConsensusOperation(ctx, op)
	}
	for _, sf := range genState.SafetyFreezes {
		k.SetSafetyFreeze(ctx, sf)
	}
	for _, cs := range genState.CoercionSignals {
		k.SetCoercionSignal(ctx, cs)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	partnerships := k.GetAllPartnerships(ctx)
	ops := k.GetAllConsensusOperations(ctx)
	freezes := k.GetAllSafetyFreezes(ctx)
	signals := k.GetAllCoercionSignals(ctx)

	return &types.GenesisState{
		Params:              k.GetParams(ctx),
		Partnerships:        partnerships,
		ConsensusOperations: ops,
		SafetyFreezes:       freezes,
		CoercionSignals:     signals,
	}
}
