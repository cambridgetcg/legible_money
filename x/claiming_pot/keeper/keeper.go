package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/claiming_pot/types"
)

// Keeper manages the claiming_pot module's state.
type Keeper struct {
	cdc        codec.Codec
	storeKey   storetypes.StoreKey
	bankKeeper types.BankKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new claiming_pot module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bankKeeper,
		authority:  authority,
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

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	k.SetParams(ctx, gs.Params)

	for _, pot := range gs.Pots {
		k.SetPot(ctx, pot)
	}

	for _, claim := range gs.Claims {
		k.SetClaimRecord(ctx, claim)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var pots []types.ClaimingPot
	k.IteratePots(ctx, func(pot types.ClaimingPot) bool {
		pots = append(pots, pot)
		return false
	})

	var claims []types.ClaimRecord
	// Iterate all claim records across all pots
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ContributionKeyPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var record types.ClaimRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		claims = append(claims, record)
	}

	return &types.GenesisState{
		Params: k.GetParams(ctx),
		Pots:   pots,
		Claims: claims,
	}
}
