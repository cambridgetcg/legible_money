package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/research/types"
)

// Keeper manages the research module's state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey

	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new research module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	bk types.BankKeeper,
	sk types.StakingKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		bankKeeper:    bk,
		stakingKeeper: sk,
		authority:     authority,
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
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	k.SetTreasuryBalance(ctx, genState.TreasuryBalance)

	for _, sub := range genState.Submissions {
		k.SetResearch(ctx, sub)
	}
	for _, bounty := range genState.Bounties {
		k.SetBounty(ctx, bounty)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var submissions []types.Research
	k.IterateResearches(ctx, func(r types.Research) bool {
		submissions = append(submissions, r)
		return false
	})

	var bounties []types.Bounty
	k.IterateBounties(ctx, func(b types.Bounty) bool {
		bounties = append(bounties, b)
		return false
	})

	return &types.GenesisState{
		Params:          k.GetParams(ctx),
		Submissions:     submissions,
		Bounties:        bounties,
		TreasuryBalance: k.GetTreasuryBalance(ctx),
	}
}

// Ensure unused imports are referenced.
var _ = json.Marshal
