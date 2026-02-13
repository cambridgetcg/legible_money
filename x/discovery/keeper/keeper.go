package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/discovery/types"
)

// Keeper manages the discovery module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	bankKeeper types.BankKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new discovery module Keeper.
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

// prefixEndBytes returns the end key for prefix iteration.
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
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, *genState.Params)
	}
	for _, profile := range genState.Profiles {
		k.SetProfile(ctx, *profile)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	var profiles []*types.AgentProfile
	k.IterateProfiles(ctx, func(p types.AgentProfile) bool {
		cp := p // copy to avoid aliasing
		profiles = append(profiles, &cp)
		return false
	})

	return &types.GenesisState{
		Params:   &params,
		Profiles: profiles,
	}
}

// ExpireStaleProfiles marks profiles as expired if they haven't sent a heartbeat.
// Returns the number of expired profiles.
func (k Keeper) ExpireStaleProfiles(ctx sdk.Context) int {
	params := k.GetParams(ctx)
	currentHeight := uint64(ctx.BlockHeight())
	expired := 0

	k.IterateProfiles(ctx, func(p types.AgentProfile) bool {
		if p.Status != "active" {
			return false
		}
		if currentHeight > p.LastActiveBlock+params.ProfileExpiryBlocks {
			p.Status = "expired"
			k.SetProfile(ctx, p)
			expired++

			ctx.EventManager().EmitEvents(sdk.Events{
				sdk.NewEvent("agent_expired",
					sdk.NewAttribute("address", p.Address),
					sdk.NewAttribute("last_active", fmt.Sprintf("%d", p.LastActiveBlock)),
				),
			})
		}
		return false
	})

	return expired
}
