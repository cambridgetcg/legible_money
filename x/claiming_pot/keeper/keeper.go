package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/claiming_pot/types"
)

// Keeper manages the claiming_pot module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService
	bankKeeper   types.BankKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new claiming_pot module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bankKeeper,
		authority:    authority,
	}
}

// prefixEndBytes returns the end key for a prefix scan (exclusive upper bound).
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

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
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
	var pots []*types.ClaimingPot
	k.IteratePots(ctx, func(pot *types.ClaimingPot) bool {
		pots = append(pots, pot)
		return false
	})

	var claims []*types.ClaimRecord
	// Iterate all claim records across all pots
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ContributionKeyPrefix, prefixEndBytes(types.ContributionKeyPrefix))
	if err == nil {
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			var record types.ClaimRecord
			if err := proto.Unmarshal(iter.Value(), &record); err != nil {
				continue
			}
			claims = append(claims, &record)
		}
	}

	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params: params,
		Pots:   pots,
		Claims: claims,
	}
}
