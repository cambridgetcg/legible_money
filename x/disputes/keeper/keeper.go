package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/disputes/types"
)

// Keeper manages the disputes module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

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
	storeService store.KVStoreService,
	bk types.BankKeeper,
	kk types.KnowledgeKeeper,
	sk types.StakingKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:             cdc,
		storeService:    storeService,
		bankKeeper:      bk,
		knowledgeKeeper: kk,
		stakingKeeper:   sk,
		authority:       authority,
	}
}

// prefixEndBytes returns the end key for prefix iteration.
// It increments the last byte of the prefix. If overflow, returns nil (iterate to end).
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
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// ---------- Genesis ----------

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, *genState.Params)
	}

	for _, dispute := range genState.Disputes {
		if dispute != nil {
			k.SetDispute(ctx, *dispute)
		}
	}
	for _, evidence := range genState.Evidence {
		if evidence != nil {
			k.SetEvidence(ctx, *evidence)
		}
	}
	for _, vote := range genState.Votes {
		if vote != nil {
			k.SetVote(ctx, *vote)
		}
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var disputes []*types.Dispute
	k.IterateDisputes(ctx, func(d types.Dispute) bool {
		disputes = append(disputes, &d)
		return false
	})

	var allEvidence []*types.DisputeEvidence
	var allVotes []*types.DisputeVote
	for _, d := range disputes {
		for _, e := range k.GetEvidenceByDispute(ctx, d.Id) {
			allEvidence = append(allEvidence, &e)
		}
		for _, v := range k.GetVotes(ctx, d.Id) {
			allVotes = append(allVotes, &v)
		}
	}

	p := k.GetParams(ctx)
	return &types.GenesisState{
		Params:   &p,
		Disputes: disputes,
		Evidence: allEvidence,
		Votes:    allVotes,
	}
}
