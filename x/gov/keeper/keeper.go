package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/gov/types"
)

// Keeper manages the gov module's state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey

	// Module authority (typically governance module address)
	authority string

	// Cross-module keepers (nil-safe — tokens overlay is optional)
	stakingKeeper types.StakingKeeper
	bankKeeper    types.BankKeeper
}

// NewKeeper creates a new gov module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	authority string,
	sk types.StakingKeeper,
	bk types.BankKeeper,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		authority:     authority,
		stakingKeeper: sk,
		bankKeeper:    bk,
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
	k.SetNextLIPNumber(ctx, gs.NextLIPNumber)
	k.SetNextDisbNumber(ctx, gs.NextDisbNumber)

	for _, lip := range gs.LIPs {
		k.SetLIP(ctx, lip)
	}
	for _, vote := range gs.Votes {
		k.SetVote(ctx, vote)
		k.SetVoteDedupe(ctx, vote.ProposalId, vote.Voter)
	}
	for _, disb := range gs.Disbursements {
		k.SetDisbursement(ctx, disb)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:         k.GetParams(ctx),
		LIPs:           k.GetAllLIPs(ctx),
		Votes:          k.GetAllVotes(ctx),
		Disbursements:  k.GetAllDisbursements(ctx),
		NextLIPNumber:  k.GetNextLIPNumber(ctx),
		NextDisbNumber: k.GetNextDisbNumber(ctx),
	}
}

// ExportGenesisJSON exports genesis as raw JSON bytes.
func (k Keeper) ExportGenesisJSON(ctx sdk.Context) json.RawMessage {
	gs := k.ExportGenesis(ctx)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic("failed to marshal gov genesis: " + err.Error())
	}
	return bz
}
