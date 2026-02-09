package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/disputes/types"
)

// Keeper manages the disputes module's state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey

	// Dependency keepers
	bankKeeper      types.BankKeeper
	knowledgeKeeper types.KnowledgeKeeper
	stakingKeeper   types.StakingKeeper
	vestingKeeper   types.VestingKeeper // nil-safe: vesting integration is optional

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new disputes module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	bk types.BankKeeper,
	kk types.KnowledgeKeeper,
	sk types.StakingKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:             cdc,
		storeKey:        storeKey,
		bankKeeper:      bk,
		knowledgeKeeper: kk,
		stakingKeeper:   sk,
		authority:       authority,
	}
}

// SetVestingKeeper sets the vesting keeper after initialization (avoids circular deps).
func (k *Keeper) SetVestingKeeper(vk types.VestingKeeper) {
	k.vestingKeeper = vk
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
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	store.Set(types.ParamsKey, bz)
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// ---------- Genesis ----------

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)

	for _, dispute := range genState.Disputes {
		k.SetDispute(ctx, dispute)
	}
	for _, evidence := range genState.Evidence {
		k.SetEvidence(ctx, evidence)
	}
	for _, vote := range genState.Votes {
		k.SetVote(ctx, vote)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var disputes []types.Dispute
	k.IterateDisputes(ctx, func(d types.Dispute) bool {
		disputes = append(disputes, d)
		return false
	})

	var allEvidence []types.DisputeEvidence
	var allVotes []types.DisputeVote
	for _, d := range disputes {
		allEvidence = append(allEvidence, k.GetEvidenceByDispute(ctx, d.Id)...)
		allVotes = append(allVotes, k.GetVotes(ctx, d.Id)...)
	}

	return &types.GenesisState{
		Params:   k.GetParams(ctx),
		Disputes: disputes,
		Evidence: allEvidence,
		Votes:    allVotes,
	}
}
