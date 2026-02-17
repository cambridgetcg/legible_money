package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/gov/types"
)

// Keeper manages the gov module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	// Module authority (typically governance module address)
	authority string

	// Cross-module keepers (nil-safe — all optional)
	stakingKeeper types.StakingKeeper
	bankKeeper    types.BankKeeper
	upgradeKeeper types.UpgradeKeeper
}

// NewKeeper creates a new gov module Keeper.
// The upgradeKeeper parameter is optional (nil-safe). When set, accepted
// consensus-category LIPs with upgrade plans will call ScheduleUpgrade.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	authority string,
	sk types.StakingKeeper,
	bk types.BankKeeper,
	uk types.UpgradeKeeper,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		authority:     authority,
		stakingKeeper: sk,
		bankKeeper:    bk,
		upgradeKeeper: uk,
	}
}

// GetUpgradeKeeper returns the upgrade keeper (may be nil).
func (k Keeper) GetUpgradeKeeper() types.UpgradeKeeper {
	return k.upgradeKeeper
}

// prefixEndBytes returns the end key for a prefix scan (exclusive upper bound).
// It increments the last byte of the prefix; if it overflows, it tries the
// previous byte, and so on. Returns nil if all bytes overflow (prefix is all 0xff).
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
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	k.SetParams(ctx, *gs.Params)
	k.SetNextLIPNumber(ctx, gs.NextLipNumber)
	k.SetNextDisbNumber(ctx, gs.NextDisbNumber)

	for _, lip := range gs.Lips {
		k.SetLIP(ctx, *lip)
	}
	for _, vote := range gs.Votes {
		k.SetVote(ctx, *vote)
		k.SetVoteDedupe(ctx, vote.ProposalId, vote.Voter)
	}
	for _, disb := range gs.Disbursements {
		k.SetDisbursement(ctx, *disb)
	}

	// Restore sybil detection state (supplementary, hand-written JSON)
	k.InitSybilGenesis(ctx, k.loadSybilGenesisFromStore(ctx))
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	p := k.GetParams(ctx)
	lips := k.GetAllLIPs(ctx)
	lipPtrs := make([]*types.LIP, len(lips))
	for i := range lips {
		lipPtrs[i] = &lips[i]
	}
	votes := k.GetAllVotes(ctx)
	votePtrs := make([]*types.QuadraticVote, len(votes))
	for i := range votes {
		votePtrs[i] = &votes[i]
	}
	disbs := k.GetAllDisbursements(ctx)
	disbPtrs := make([]*types.DisbursementProposal, len(disbs))
	for i := range disbs {
		disbPtrs[i] = &disbs[i]
	}
	return &types.GenesisState{
		Params:         &p,
		Lips:           lipPtrs,
		Votes:          votePtrs,
		Disbursements:  disbPtrs,
		NextLipNumber:  k.GetNextLIPNumber(ctx),
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

// SaveSybilGenesisToStore persists sybil genesis JSON into the module store.
// Called during ExportGenesis so that InitGenesis can restore it.
func (k Keeper) SaveSybilGenesisToStore(ctx sdk.Context) {
	bz := k.ExportSybilGenesis(ctx)
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(sybilGenesisStoreKey(), bz); err != nil {
		panic("failed to save sybil genesis: " + err.Error())
	}
}

// loadSybilGenesisFromStore loads sybil genesis JSON from the module store.
func (k Keeper) loadSybilGenesisFromStore(ctx sdk.Context) json.RawMessage {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(sybilGenesisStoreKey())
	if err != nil || bz == nil {
		return nil
	}
	return bz
}

// sybilGenesisStoreKey returns a store key for the sybil genesis blob.
// Uses SybilParamsKey with a "genesis" suffix to avoid collision with the params key.
func sybilGenesisStoreKey() []byte {
	return append(types.SybilParamsKey, []byte("/genesis")...)
}
