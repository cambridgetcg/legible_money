package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/compute_pool/types"
)

// Keeper manages the compute_pool module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	bankKeeper      types.BankKeeper
	stakingKeeper   types.StakingKeeper
	knowledgeKeeper types.KnowledgeKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new compute_pool module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bk types.BankKeeper,
	sk types.StakingKeeper,
	kk types.KnowledgeKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:             cdc,
		storeService:    storeService,
		bankKeeper:      bk,
		stakingKeeper:   sk,
		knowledgeKeeper: kk,
		authority:       authority,
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

// ---------- Adjustment Factor ----------

// GetAdjustmentFactor returns the current credit adjustment factor (bps).
// Returns DefaultAdjustmentFactorBps (1000000 = 1.0x) if not yet set.
func (k Keeper) GetAdjustmentFactor(ctx sdk.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.AdjustmentFactorKey)
	if err != nil || bz == nil {
		return types.DefaultAdjustmentFactorBps
	}
	return binary.BigEndian.Uint64(bz)
}

// SetAdjustmentFactor stores the current credit adjustment factor (bps).
func (k Keeper) SetAdjustmentFactor(ctx sdk.Context, factorBps uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, factorBps)
	_ = kvStore.Set(types.AdjustmentFactorKey, bz)
}

// ---------- Params ----------

// SetParams stores module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	_ = kvStore.Set(types.ParamsKey, bz)
}

// GetParams returns the current module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// ---------- Genesis ----------

// InitGenesis initializes the module genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	for _, provider := range genState.Providers {
		k.SetProvider(ctx, provider)
	}
	for _, credit := range genState.Credits {
		k.SetCredit(ctx, credit)
	}
}

// ExportGenesis exports the module genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	providers := k.GetAllProviders(ctx)
	credits := k.GetAllCredits(ctx)
	return &types.GenesisState{
		Params:    params,
		Providers: providers,
		Credits:   credits,
	}
}

// InitGenesisExtended initializes tasks and usage records from extended genesis.
func (k Keeper) InitGenesisExtended(ctx sdk.Context, tasks []*types.ComputeTask, usages []*types.AgentUsageRecord) {
	for _, task := range tasks {
		k.SetTask(ctx, task)
		k.SetTaskRequesterIndex(ctx, task.RequesterDID, task.Id)
		if task.Provider != "" {
			k.SetTaskProviderIndex(ctx, task.Provider, task.Id)
		}
	}
	for _, usage := range usages {
		k.SetAgentUsage(ctx, usage)
	}
}

// ExportGenesisExtended exports tasks and usage records.
func (k Keeper) ExportGenesisExtended(ctx sdk.Context) ([]*types.ComputeTask, []*types.AgentUsageRecord) {
	return k.GetAllTasks(ctx), k.GetAllAgentUsages(ctx)
}
