package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/research/types"
)

// Keeper manages the research module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new research module Keeper.
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

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, *genState.Params)
	}
	if genState.TreasuryBalance != nil {
		k.SetTreasuryBalance(ctx, genState.TreasuryBalance.Balance)
	}

	for _, sub := range genState.Researches {
		if sub != nil {
			k.SetResearch(ctx, *sub)
		}
	}
	for _, bounty := range genState.Bounties {
		if bounty != nil {
			k.SetBounty(ctx, *bounty)
		}
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var researches []*types.Research
	k.IterateResearches(ctx, func(r types.Research) bool {
		rCopy := r
		researches = append(researches, &rCopy)
		return false
	})

	var bounties []*types.Bounty
	k.IterateBounties(ctx, func(b types.Bounty) bool {
		bCopy := b
		bounties = append(bounties, &bCopy)
		return false
	})

	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params:          &params,
		Researches:      researches,
		Bounties:        bounties,
		TreasuryBalance: &types.TreasuryBalance{Balance: k.GetTreasuryBalance(ctx)},
	}
}

// Ensure unused imports are referenced.
var _ = json.Marshal
