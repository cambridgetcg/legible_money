package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/alignment/types"
)

// Keeper manages alignment state.
type Keeper struct {
	storeService       store.KVStoreService
	logger             log.Logger
	authority          string

	// Cross-module keepers (nil-safe)
	knowledgeKeeper    types.KnowledgeKeeper
	stakingKeeper      types.StakingKeeper
	ontologyKeeper     types.OntologyKeeper
	autopoiesisKeeper  types.AutopoiesisKeeper
	emergencyKeeper    types.EmergencyKeeper
	vestingKeeper      types.VestingRewardsKeeper
}

// NewKeeper creates a new alignment keeper.
func NewKeeper(
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
) Keeper {
	return Keeper{
		storeService: storeService,
		logger:       logger,
		authority:    authority,
	}
}

// SetKnowledgeKeeper sets the knowledge keeper (post-init wiring).
func (k *Keeper) SetKnowledgeKeeper(kk types.KnowledgeKeeper) {
	k.knowledgeKeeper = kk
}

// SetStakingKeeper sets the staking keeper.
func (k *Keeper) SetStakingKeeper(sk types.StakingKeeper) {
	k.stakingKeeper = sk
}

// SetOntologyKeeper sets the ontology keeper.
func (k *Keeper) SetOntologyKeeper(ok types.OntologyKeeper) {
	k.ontologyKeeper = ok
}

// SetAutopoiesisKeeper sets the autopoiesis keeper.
func (k *Keeper) SetAutopoiesisKeeper(ak types.AutopoiesisKeeper) {
	k.autopoiesisKeeper = ak
}

// SetEmergencyKeeper sets the emergency keeper.
func (k *Keeper) SetEmergencyKeeper(ek types.EmergencyKeeper) {
	k.emergencyKeeper = ek
}

// SetVestingKeeper sets the vesting rewards keeper.
func (k *Keeper) SetVestingKeeper(vk types.VestingRewardsKeeper) {
	k.vestingKeeper = vk
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return k.logger.With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the governance authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// --- Params ---

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal alignment params: %v", err))
	}
	if err := kvStore.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set alignment params: %v", err))
	}
}

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// --- Module State ---

func (k Keeper) SetState(ctx sdk.Context, state types.AlignmentState) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(state)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal alignment state: %v", err))
	}
	if err := kvStore.Set(types.StateKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set alignment state: %v", err))
	}
}

func (k Keeper) GetState(ctx sdk.Context) types.AlignmentState {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.StateKey)
	if err != nil || bz == nil {
		return types.AlignmentState{
			Active:            false,
			SmoothingWindow:   5,
			CorrectionDamping: types.BasisPoints, // 1.0 = no damping
		}
	}
	var state types.AlignmentState
	if err := json.Unmarshal(bz, &state); err != nil {
		return types.AlignmentState{Active: false, SmoothingWindow: 5, CorrectionDamping: types.BasisPoints}
	}
	return state
}

// IsActive returns whether the alignment module is active.
func (k Keeper) IsActive(ctx sdk.Context) bool {
	state := k.GetState(ctx)
	if !state.Active {
		return false
	}
	params := k.GetParams(ctx)
	if !params.Enabled {
		return false
	}
	if k.emergencyKeeper != nil && k.emergencyKeeper.IsHalted(ctx) {
		return false
	}
	return true
}

// --- Observations ---

func observationKey(epoch uint64) []byte {
	return append(types.ObservationKeyPrefix, []byte(fmt.Sprintf("%016d", epoch))...)
}

func (k Keeper) SetObservation(ctx sdk.Context, obs types.AlignmentObservation) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(obs)
	if err != nil {
		return
	}
	_ = kvStore.Set(observationKey(obs.Epoch), bz)
}

func (k Keeper) GetObservation(ctx sdk.Context, epoch uint64) *types.AlignmentObservation {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(observationKey(epoch))
	if err != nil || bz == nil {
		return nil
	}
	var obs types.AlignmentObservation
	if err := json.Unmarshal(bz, &obs); err != nil {
		return nil
	}
	return &obs
}

// --- Dimension Scores ---

func scoreKey(epoch uint64) []byte {
	return append(types.ScoreKeyPrefix, []byte(fmt.Sprintf("%016d", epoch))...)
}

func (k Keeper) SetScores(ctx sdk.Context, scores types.DimensionScores) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(scores)
	if err != nil {
		return
	}
	_ = kvStore.Set(scoreKey(scores.Epoch), bz)
}

func (k Keeper) GetScores(ctx sdk.Context, epoch uint64) *types.DimensionScores {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(scoreKey(epoch))
	if err != nil || bz == nil {
		return nil
	}
	var scores types.DimensionScores
	if err := json.Unmarshal(bz, &scores); err != nil {
		return nil
	}
	return &scores
}

// --- Corrections ---

func correctionKey(epoch uint64) []byte {
	return append(types.CorrectionKeyPrefix, []byte(fmt.Sprintf("%016d", epoch))...)
}

func (k Keeper) SetCorrections(ctx sdk.Context, epoch uint64, corrections []types.CorrectionRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(corrections)
	if err != nil {
		return
	}
	_ = kvStore.Set(correctionKey(epoch), bz)
}

func (k Keeper) GetCorrections(ctx sdk.Context, epoch uint64) []types.CorrectionRecord {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(correctionKey(epoch))
	if err != nil || bz == nil {
		return nil
	}
	var corrections []types.CorrectionRecord
	if err := json.Unmarshal(bz, &corrections); err != nil {
		return nil
	}
	return corrections
}

// --- Meta-Alignment ---

func metaKey(epoch uint64) []byte {
	return append(types.MetaKeyPrefix, []byte(fmt.Sprintf("%016d", epoch))...)
}

func (k Keeper) SetMetaHealth(ctx sdk.Context, ahi types.AlignmentHealthIndex) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(ahi)
	if err != nil {
		return
	}
	_ = kvStore.Set(metaKey(ahi.Epoch), bz)
}

func (k Keeper) GetMetaHealth(ctx sdk.Context, epoch uint64) *types.AlignmentHealthIndex {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(metaKey(epoch))
	if err != nil || bz == nil {
		return nil
	}
	var ahi types.AlignmentHealthIndex
	if err := json.Unmarshal(bz, &ahi); err != nil {
		return nil
	}
	return &ahi
}

// --- Correction History (for meta-loop) ---

func correctionHistKey(epoch uint64) []byte {
	return append(types.CorrectionHistPrefix, []byte(fmt.Sprintf("%016d", epoch))...)
}

func (k Keeper) SetCorrectionOutcomes(ctx sdk.Context, epoch uint64, outcomes []types.CorrectionOutcome) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(outcomes)
	if err != nil {
		return
	}
	_ = kvStore.Set(correctionHistKey(epoch), bz)
}

func (k Keeper) GetCorrectionOutcomes(ctx sdk.Context, epoch uint64) []types.CorrectionOutcome {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(correctionHistKey(epoch))
	if err != nil || bz == nil {
		return nil
	}
	var outcomes []types.CorrectionOutcome
	if err := json.Unmarshal(bz, &outcomes); err != nil {
		return nil
	}
	return outcomes
}

// --- Genesis ---

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	k.SetParams(ctx, gs.Params)
	k.SetState(ctx, types.AlignmentState{
		Active:            gs.Params.Enabled,
		CurrentEpoch:      0,
		SmoothingWindow:   5,
		CorrectionDamping: types.BasisPoints,
	})
}

func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	return types.GenesisState{
		Params: k.GetParams(ctx),
	}
}

// --- Epoch Cycle ---

// SenseJudgeCorrectReflect performs the full alignment cycle:
// SENSE → JUDGE → CORRECT → REFLECT (meta-loop every N epochs).
// Called from EndBlocker at epoch boundaries (after autopoiesis).
func (k Keeper) SenseJudgeCorrectReflect(ctx sdk.Context) {
	if !k.IsActive(ctx) {
		return
	}

	state := k.GetState(ctx)
	params := k.GetParams(ctx)
	blockHeight := uint64(ctx.BlockHeight())

	// Check epoch boundary
	epoch := blockHeight / params.EpochLengthBlocks
	if epoch <= state.CurrentEpoch && state.CurrentEpoch > 0 {
		return
	}

	epochStart := state.LastObservedHeight
	if epochStart == 0 && epoch > 0 {
		epochStart = (epoch - 1) * params.EpochLengthBlocks
	}

	// ============ SENSE ============
	obs := k.collectSignals(ctx, epoch, blockHeight, epochStart)
	k.SetObservation(ctx, obs)

	// ============ JUDGE ============
	scores := k.computeScores(obs, state)
	scores.Epoch = epoch
	k.SetScores(ctx, scores)

	// ============ CORRECT ============
	corrections := k.computeCorrections(ctx, scores, state, params)
	k.applyCorrections(ctx, corrections, epoch)
	k.SetCorrections(ctx, epoch, corrections)

	// ============ REFLECT (meta-loop every N epochs) ============
	if epoch >= types.MinEpochsForMeta && epoch%uint64(types.MetaLoopInterval) == 0 {
		k.runMetaLoop(ctx, epoch, &state)
	}

	// Logging
	changed := 0
	for _, c := range corrections {
		if c.Delta != 0 {
			changed++
		}
	}
	k.Logger(ctx).Info("alignment epoch",
		"epoch", epoch,
		"tdi", scores.TDI,
		"dci", scores.DCI,
		"hi", scores.HI,
		"fi", scores.FI,
		"ni", scores.NI,
		"corrections", changed,
	)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("alignment_epoch",
			sdk.NewAttribute("epoch", fmt.Sprintf("%d", epoch)),
			sdk.NewAttribute("tdi", fmt.Sprintf("%d", scores.TDI)),
			sdk.NewAttribute("dci", fmt.Sprintf("%d", scores.DCI)),
			sdk.NewAttribute("hi", fmt.Sprintf("%d", scores.HI)),
			sdk.NewAttribute("fi", fmt.Sprintf("%d", scores.FI)),
			sdk.NewAttribute("ni", fmt.Sprintf("%d", scores.NI)),
			sdk.NewAttribute("corrections", fmt.Sprintf("%d", changed)),
		),
	)

	// Update state
	state.CurrentEpoch = epoch
	state.LastObservedHeight = blockHeight
	k.SetState(ctx, state)
}
