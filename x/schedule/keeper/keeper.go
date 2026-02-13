package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/schedule/types"
)

// Keeper manages the schedule module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	bankKeeper types.BankKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new schedule module Keeper.
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
	k.SetSequence(ctx, genState.Sequence)

	for _, process := range genState.Processes {
		if process != nil {
			k.SetProcess(ctx, *process)
		}
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var processes []*types.ScheduleProcess
	k.IterateProcesses(ctx, func(p types.ScheduleProcess) bool {
		pCopy := p
		processes = append(processes, &pCopy)
		return false
	})

	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params:    &params,
		Processes: processes,
		Sequence:  k.GetSequence(ctx),
	}
}
