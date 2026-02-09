package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tree/types"
)

// Keeper manages the tree module's state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey

	bankKeeper types.BankKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new tree module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	bk types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bk,
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
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	k.SetParams(ctx, gs.Params)
	for _, p := range gs.Projects {
		p := p
		k.SetProject(ctx, p)
	}
	for _, t := range gs.Tasks {
		t := t
		k.SetTask(ctx, t)
	}
	for _, s := range gs.Services {
		s := s
		k.SetService(ctx, s)
	}
	for _, s := range gs.Seeds {
		s := s
		k.SetSeed(ctx, s)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	gs := &types.GenesisState{
		Params:   k.GetParams(ctx),
		Projects: []types.ProductProject{},
		Tasks:    []types.ProjectTask{},
		Services: []types.ServiceLeaf{},
		Seeds:    []types.OpportunitySeed{},
	}

	k.IterateProjects(ctx, func(p types.ProductProject) bool {
		gs.Projects = append(gs.Projects, p)
		return false
	})
	k.IterateTasks(ctx, func(t types.ProjectTask) bool {
		gs.Tasks = append(gs.Tasks, t)
		return false
	})
	k.IterateServices(ctx, func(s types.ServiceLeaf) bool {
		gs.Services = append(gs.Services, s)
		return false
	})
	k.IterateSeeds(ctx, func(s types.OpportunitySeed) bool {
		gs.Seeds = append(gs.Seeds, s)
		return false
	})

	return gs
}

// ExportGenesisJSON exports genesis as JSON for the module interface.
func (k Keeper) ExportGenesisJSON(ctx sdk.Context) json.RawMessage {
	gs := k.ExportGenesis(ctx)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal tree genesis: %v", err))
	}
	return bz
}
