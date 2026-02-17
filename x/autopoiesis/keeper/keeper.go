package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

// Keeper manages autopoiesis state.
type Keeper struct {
	storeService    store.KVStoreService
	logger          log.Logger
	stakingKeeper   types.StakingKeeper
	knowledgeKeeper types.KnowledgeKeeper
	emergencyKeeper types.EmergencyKeeper
	govKeeper       types.GovKeeper
	authority       string
}

// NewKeeper creates a new autopoiesis keeper.
func NewKeeper(
	storeService store.KVStoreService,
	logger log.Logger,
	sk types.StakingKeeper,
	kk types.KnowledgeKeeper,
	ek types.EmergencyKeeper,
	authority string,
) Keeper {
	return Keeper{
		storeService:    storeService,
		logger:          logger,
		stakingKeeper:   sk,
		knowledgeKeeper: kk,
		emergencyKeeper: ek,
		authority:       authority,
	}
}

// SetGovKeeper sets the governance keeper (called after init to avoid circular deps).
func (k *Keeper) SetGovKeeper(gk types.GovKeeper) {
	k.govKeeper = gk
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

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal autopoiesis params: %v", err))
	}
	if err := kvStore.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set autopoiesis params: %v", err))
	}
}

// GetParams returns module parameters.
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

// SetState sets module activation state.
func (k Keeper) SetState(ctx sdk.Context, state types.AutopoiesisState) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(state)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal autopoiesis state: %v", err))
	}
	if err := kvStore.Set(types.StateKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set autopoiesis state: %v", err))
	}
}

// GetState returns module activation state.
func (k Keeper) GetState(ctx sdk.Context) types.AutopoiesisState {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.StateKey)
	if err != nil || bz == nil {
		return types.AutopoiesisState{Active: false}
	}
	var state types.AutopoiesisState
	if err := json.Unmarshal(bz, &state); err != nil {
		return types.AutopoiesisState{Active: false}
	}
	return state
}

// IsActive returns whether autopoiesis is active and not emergency-halted.
func (k Keeper) IsActive(ctx sdk.Context) bool {
	state := k.GetState(ctx)
	if !state.Active {
		return false
	}
	params := k.GetParams(ctx)
	if !params.Enabled {
		return false
	}
	// Emergency freeze
	if k.emergencyKeeper != nil && k.emergencyKeeper.IsHalted(ctx) {
		return false
	}
	return true
}

// --- Multipliers ---

func multiplierKey(path string) []byte {
	return append(types.MultiplierKeyPrefix, []byte(path)...)
}

// SetMultiplier stores a multiplier entry.
func (k Keeper) SetMultiplier(ctx sdk.Context, entry types.MultiplierEntry) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = kvStore.Set(multiplierKey(entry.Path), bz)
}

// GetMultiplier returns the multiplier for a path. Returns DefaultMultiplier (1.0x) if not set.
// This is the consumer-facing API — safe to call from any module.
func (k Keeper) GetMultiplier(ctx sdk.Context, path string) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(multiplierKey(path))
	if err != nil || bz == nil {
		return types.DefaultMultiplier
	}
	var entry types.MultiplierEntry
	if err := json.Unmarshal(bz, &entry); err != nil {
		return types.DefaultMultiplier
	}
	return entry.Value
}

// GetMultiplierEntry returns the full multiplier entry, or nil if not set.
func (k Keeper) GetMultiplierEntry(ctx sdk.Context, path string) *types.MultiplierEntry {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(multiplierKey(path))
	if err != nil || bz == nil {
		return nil
	}
	var entry types.MultiplierEntry
	if err := json.Unmarshal(bz, &entry); err != nil {
		return nil
	}
	return &entry
}

// --- Observations ---

func observationKey(epoch uint64) []byte {
	return append(types.ObservationKeyPrefix, []byte(fmt.Sprintf("%016d", epoch))...)
}

// SetObservation stores an epoch observation.
func (k Keeper) SetObservation(ctx sdk.Context, obs types.EpochObservation) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(obs)
	if err != nil {
		return
	}
	_ = kvStore.Set(observationKey(obs.Epoch), bz)
}

// GetObservation returns an observation for an epoch, or nil.
func (k Keeper) GetObservation(ctx sdk.Context, epoch uint64) *types.EpochObservation {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(observationKey(epoch))
	if err != nil || bz == nil {
		return nil
	}
	var obs types.EpochObservation
	if err := json.Unmarshal(bz, &obs); err != nil {
		return nil
	}
	return &obs
}

// --- Hormone Values ---

func hormoneKey(epoch uint64) []byte {
	return append(types.HormoneKeyPrefix, []byte(fmt.Sprintf("%016d", epoch))...)
}

// SetHormoneValues stores all computed hormone values for an epoch.
func (k Keeper) SetHormoneValues(ctx sdk.Context, hv types.HormoneValues) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(hv)
	if err != nil {
		return
	}
	_ = kvStore.Set(hormoneKey(hv.Epoch), bz)
}

// GetHormoneValues returns the hormone values for an epoch, or nil.
func (k Keeper) GetHormoneValues(ctx sdk.Context, epoch uint64) *types.HormoneValues {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(hormoneKey(epoch))
	if err != nil || bz == nil {
		return nil
	}
	var hv types.HormoneValues
	if err := json.Unmarshal(bz, &hv); err != nil {
		return nil
	}
	return &hv
}

// SetSSI stores an SSI value for an epoch (backward-compatible wrapper).
func (k Keeper) SetSSI(ctx sdk.Context, ssi types.SSIValue) {
	hv := k.GetHormoneValues(ctx, ssi.Epoch)
	if hv == nil {
		hv = &types.HormoneValues{Epoch: ssi.Epoch}
	}
	hv.SSI = ssi.Value
	k.SetHormoneValues(ctx, *hv)
}

// GetSSI returns the SSI for an epoch, or nil (backward-compatible wrapper).
func (k Keeper) GetSSI(ctx sdk.Context, epoch uint64) *types.SSIValue {
	hv := k.GetHormoneValues(ctx, epoch)
	if hv == nil {
		return nil
	}
	return &types.SSIValue{Epoch: epoch, Value: hv.SSI}
}

// --- DVI Tracking (consecutive negative epochs per domain) ---

func dviTrackingKey(domain string) []byte {
	return append(types.DVITrackingKeyPrefix, []byte(domain)...)
}

// SetDVITracking stores DVI consecutive negative epoch tracking for a domain.
func (k Keeper) SetDVITracking(ctx sdk.Context, dvi types.DVIValue) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(dvi)
	if err != nil {
		return
	}
	_ = kvStore.Set(dviTrackingKey(dvi.Domain), bz)
}

// GetDVITracking returns DVI tracking for a domain, or nil.
func (k Keeper) GetDVITracking(ctx sdk.Context, domain string) *types.DVIValue {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(dviTrackingKey(domain))
	if err != nil || bz == nil {
		return nil
	}
	var dvi types.DVIValue
	if err := json.Unmarshal(bz, &dvi); err != nil {
		return nil
	}
	return &dvi
}

// --- Adaptation History ---

func adaptationKey(epoch uint64) []byte {
	return append(types.AdaptationKeyPrefix, []byte(fmt.Sprintf("%016d", epoch))...)
}

// SetAdaptations stores adaptation records for an epoch.
func (k Keeper) SetAdaptations(ctx sdk.Context, epoch uint64, adaptations []types.Adaptation) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(adaptations)
	if err != nil {
		return
	}
	_ = kvStore.Set(adaptationKey(epoch), bz)
}

// --- Genesis ---

// InitGenesis initializes module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	k.SetParams(ctx, gs.Params)
	k.SetState(ctx, types.AutopoiesisState{
		Active:             gs.Params.Enabled,
		CurrentEpoch:       0,
		LastObservedHeight: 0,
	})
}

// ExportGenesis exports current module state as genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	return types.GenesisState{
		Params: k.GetParams(ctx),
	}
}

// --- Epoch Cycle ---

// CollectAndAdapt performs the full OBSERVE → COMPUTE → ADAPT cycle.
// Called from EndBlocker at epoch boundaries.
//
// Hormone priority (conflict resolution):
//
//	SSI (priority 4) > CRI (priority 3) > DVI/VCF (priority 2) > GRM (priority 1)
//
// When multiple hormones target the same multiplier path, the higher-priority
// hormone's adaptation wins. DVI and VCF have equal priority — their deltas
// are averaged when they conflict on the same path.
func (k Keeper) CollectAndAdapt(ctx sdk.Context) {
	if !k.IsActive(ctx) {
		return
	}

	state := k.GetState(ctx)
	params := k.GetParams(ctx)
	blockHeight := uint64(ctx.BlockHeight())

	// Check if we've reached an epoch boundary
	epoch := blockHeight / params.EpochLengthBlocks
	if epoch <= state.CurrentEpoch && state.CurrentEpoch > 0 {
		return // not a new epoch
	}

	// Calculate observation range
	epochStart := state.LastObservedHeight
	if epochStart == 0 && epoch > 0 {
		epochStart = (epoch - 1) * params.EpochLengthBlocks
	}

	// ============ OBSERVE ============
	obs := k.collectMetrics(ctx, epoch, blockHeight, epochStart)
	k.SetObservation(ctx, obs)

	// ============ COMPUTE ============
	// Load previous DVI tracking for consecutive negative epoch detection
	var previousDVI []types.DVIValue
	for _, dom := range obs.Domains {
		if prev := k.GetDVITracking(ctx, dom.Domain); prev != nil {
			previousDVI = append(previousDVI, *prev)
		}
	}

	ssi := types.ComputeSSI(obs)
	cri := types.ComputeCRI(obs.Retentions)
	dvi := types.ComputeDVI(obs.Domains, previousDVI)
	vcf := types.ComputeVCF(obs.Domains)
	grm := types.ComputeGRM(obs)
	lifeForce := types.ComputeLifeForce(obs, cri, ssi)

	hormones := types.HormoneValues{
		Epoch:     epoch,
		SSI:       ssi,
		CRI:       cri,
		DVI:       dvi,
		VCF:       vcf,
		GRM:       grm,
		LifeForce: lifeForce,
	}
	k.SetHormoneValues(ctx, hormones)

	// ============ ADAPT ============
	// Priority-based conflict resolution: higher-priority hormones claim paths first.
	// claimedPaths tracks which hormone owns each path.
	claimedPaths := map[string]string{} // path → hormone name
	var allAdaptations []types.Adaptation

	// --- Phase 1: SSI (priority 4 — survival) ---
	ssiAdaptations := k.computeSSIAdaptations(ctx, ssi, epoch, params)
	for _, a := range ssiAdaptations {
		claimedPaths[a.Path] = "SSI"
		allAdaptations = append(allAdaptations, a)
	}

	// --- Phase 2: CRI (priority 3 — retention) ---
	for _, c := range cri {
		criAdaptations := k.computeCRIAdaptations(ctx, c, epoch, params)
		for _, a := range criAdaptations {
			if _, claimed := claimedPaths[a.Path]; !claimed {
				claimedPaths[a.Path] = "CRI"
				allAdaptations = append(allAdaptations, a)
			}
		}
	}

	// --- Phase 3: DVI + VCF (priority 2 — growth direction) ---
	for _, d := range dvi {
		dviAdaptations := k.computeDVIAdaptations(ctx, d, epoch, params)
		for _, a := range dviAdaptations {
			if _, claimed := claimedPaths[a.Path]; !claimed {
				claimedPaths[a.Path] = "DVI"
				allAdaptations = append(allAdaptations, a)
			}
		}
	}
	for _, v := range vcf {
		vcfAdaptations := k.computeVCFAdaptations(ctx, v, epoch, params)
		for _, a := range vcfAdaptations {
			existing, claimed := claimedPaths[a.Path]
			if !claimed {
				claimedPaths[a.Path] = "VCF"
				allAdaptations = append(allAdaptations, a)
			} else if existing == "DVI" {
				// DVI and VCF have equal priority — average their deltas
				for i := range allAdaptations {
					if allAdaptations[i].Path == a.Path {
						avgDelta := (allAdaptations[i].Delta + a.Delta) / 2
						allAdaptations[i].Delta = avgDelta
						if avgDelta >= 0 {
							allAdaptations[i].NewValue = allAdaptations[i].OldValue + uint64(avgDelta)
						} else {
							sub := uint64(-avgDelta)
							if sub > allAdaptations[i].OldValue {
								allAdaptations[i].NewValue = 0
							} else {
								allAdaptations[i].NewValue = allAdaptations[i].OldValue - sub
							}
						}
						allAdaptations[i].Hormone = "DVI+VCF"
						break
					}
				}
			}
		}
	}

	// --- Phase 4: GRM (priority 1 — pace) ---
	currentMaxTx := k.GetMultiplier(ctx, types.PathMaxTxSystem)
	currentGas := k.GetMultiplier(ctx, types.PathGasSystem)
	grmAdaptations := types.ComputeGRMAdaptation(grm, currentMaxTx, currentGas, params)
	for _, a := range grmAdaptations {
		if _, claimed := claimedPaths[a.Path]; !claimed {
			a.Epoch = epoch
			allAdaptations = append(allAdaptations, a)
		}
	}

	// === Write all adaptations ===
	for _, a := range allAdaptations {
		if a.Delta != 0 {
			k.SetMultiplier(ctx, types.MultiplierEntry{
				Path:    a.Path,
				Value:   a.NewValue,
				Hormone: a.Hormone,
				Epoch:   epoch,
			})
		}
	}
	k.SetAdaptations(ctx, epoch, allAdaptations)

	// === Update DVI tracking for consecutive negative epochs ===
	for _, d := range dvi {
		k.SetDVITracking(ctx, d)
	}

	// === Meta-Learning (Loop 2): evaluate adaptation effectiveness ===
	// Runs every LearningIntervalEpochs (10) after minimum data threshold.
	if epoch >= types.MinLearningEpochs && epoch%uint64(types.LearningIntervalEpochs) == 0 {
		k.RunMetaLearning(ctx, epoch, hormones)
	}

	// === Auto-Governance Proposals (Loop 4): check for structural issues ===
	k.CheckProposalTriggers(ctx, epoch, hormones)

	// === Logging ===
	changed := 0
	for _, a := range allAdaptations {
		if a.Delta != 0 {
			changed++
		}
	}
	k.Logger(ctx).Info("autopoiesis epoch",
		"epoch", epoch,
		"ssi", ssi,
		"grm", grm,
		"cri_count", len(cri),
		"dvi_count", len(dvi),
		"vcf_count", len(vcf),
		"life_force", lifeForce.Status,
		"adaptations", changed,
	)

	// === Emit event ===
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("autopoiesis_epoch",
			sdk.NewAttribute("epoch", fmt.Sprintf("%d", epoch)),
			sdk.NewAttribute("ssi", fmt.Sprintf("%d", ssi)),
			sdk.NewAttribute("grm", fmt.Sprintf("%d", grm)),
			sdk.NewAttribute("life_force", fmt.Sprintf("%d", lifeForce.Value)),
			sdk.NewAttribute("life_force_status", lifeForce.Status),
			sdk.NewAttribute("adaptations", fmt.Sprintf("%d", changed)),
		),
	)

	// Update state
	state.CurrentEpoch = epoch
	state.LastObservedHeight = blockHeight
	k.SetState(ctx, state)
}

// ============ Per-Hormone Adaptation Computation ============

// computeSSIAdaptations computes SSI-driven parameter changes.
// SSI targets: slash.system, blockreward.system, minverifiers.system
func (k Keeper) computeSSIAdaptations(ctx sdk.Context, ssi uint64, epoch uint64, params types.Params) []types.Adaptation {
	var results []types.Adaptation

	currentSlash := k.GetMultiplier(ctx, types.PathSlashSystem)
	newSlash, slashDelta := types.ComputeSSIAdaptation(ssi, currentSlash, params)
	if slashDelta != 0 {
		results = append(results, types.Adaptation{
			Path:     types.PathSlashSystem,
			OldValue: currentSlash,
			NewValue: newSlash,
			Delta:    slashDelta,
			Hormone:  "SSI",
			Epoch:    epoch,
		})
	}

	// SSI also targets block reward and min verifiers at moderate+ stress
	if ssi > params.SSIModerateThreshold {
		currentReward := k.GetMultiplier(ctx, types.PathBlockRewardSystem)
		rawDelta := -int64(types.MaxChangePerEpochBps / 2) // reduce rewards under stress
		rawDelta = k.applyLearnedRate(ctx, types.PathBlockRewardSystem, rawDelta)
		effectiveDelta := types.ApplyDampedDelta(rawDelta, currentReward,
			types.BoundsBlockRewardMultiplier, params.MaxChangePerEpochBps)
		if effectiveDelta != 0 {
			newReward := applySignedDelta(currentReward, effectiveDelta)
			results = append(results, types.Adaptation{
				Path:     types.PathBlockRewardSystem,
				OldValue: currentReward,
				NewValue: newReward,
				Delta:    effectiveDelta,
				Hormone:  "SSI",
				Epoch:    epoch,
			})
		}
	} else {
		// Healthy: relax block reward toward 1.0x
		currentReward := k.GetMultiplier(ctx, types.PathBlockRewardSystem)
		if currentReward < types.DefaultMultiplier {
			rawDelta := int64(types.MaxChangePerEpochBps / 4)
			rawDelta = k.applyLearnedRate(ctx, types.PathBlockRewardSystem, rawDelta)
			effectiveDelta := types.ApplyDampedDelta(rawDelta, currentReward,
				types.BoundsBlockRewardMultiplier, params.MaxChangePerEpochBps)
			if effectiveDelta != 0 {
				newReward := applySignedDelta(currentReward, effectiveDelta)
				results = append(results, types.Adaptation{
					Path:     types.PathBlockRewardSystem,
					OldValue: currentReward,
					NewValue: newReward,
					Delta:    effectiveDelta,
					Hormone:  "SSI",
					Epoch:    epoch,
				})
			}
		}
	}

	return results
}

// computeCRIAdaptations computes CRI-driven parameter changes.
// CRI targets: decay.{category}.{domain}, revbonus.{category}.{domain}
func (k Keeper) computeCRIAdaptations(ctx sdk.Context, cri types.CRIValue, epoch uint64, params types.Params) []types.Adaptation {
	var results []types.Adaptation

	decayPath := fmt.Sprintf("decay.%s.%s", cri.Category, cri.Domain)
	revBonusPath := fmt.Sprintf("revbonus.%s.%s", cri.Category, cri.Domain)

	if cri.Value > types.CRIIncreaseThreshold {
		// High retention → slow the decay (increase decay multiplier = longer half-life)
		currentDecay := k.GetMultiplier(ctx, decayPath)
		rawDelta := int64(types.MaxChangePerEpochBps)
		rawDelta = k.applyLearnedRate(ctx, decayPath, rawDelta)
		effectiveDelta := types.ApplyDampedDelta(rawDelta, currentDecay,
			types.BoundsDecayMultiplier, params.MaxChangePerEpochBps)
		if effectiveDelta != 0 {
			results = append(results, types.Adaptation{
				Path:     decayPath,
				OldValue: currentDecay,
				NewValue: applySignedDelta(currentDecay, effectiveDelta),
				Delta:    effectiveDelta,
				Hormone:  "CRI",
				Epoch:    epoch,
			})
		}
	} else if cri.Value < types.CRIDecreaseThreshold {
		// Low retention → accelerate decay + incentivize reverification
		currentDecay := k.GetMultiplier(ctx, decayPath)
		rawDelta := -int64(types.MaxChangePerEpochBps)
		rawDelta = k.applyLearnedRate(ctx, decayPath, rawDelta)
		effectiveDelta := types.ApplyDampedDelta(rawDelta, currentDecay,
			types.BoundsDecayMultiplier, params.MaxChangePerEpochBps)
		if effectiveDelta != 0 {
			results = append(results, types.Adaptation{
				Path:     decayPath,
				OldValue: currentDecay,
				NewValue: applySignedDelta(currentDecay, effectiveDelta),
				Delta:    effectiveDelta,
				Hormone:  "CRI",
				Epoch:    epoch,
			})
		}

		currentRevBonus := k.GetMultiplier(ctx, revBonusPath)
		revDelta := int64(types.MaxChangePerEpochBps * 3 / 5) // +3%
		revDelta = k.applyLearnedRate(ctx, revBonusPath, revDelta)
		effectiveRev := types.ApplyDampedDelta(revDelta, currentRevBonus,
			types.BoundsRevBonusMultiplier, params.MaxChangePerEpochBps)
		if effectiveRev != 0 {
			results = append(results, types.Adaptation{
				Path:     revBonusPath,
				OldValue: currentRevBonus,
				NewValue: applySignedDelta(currentRevBonus, effectiveRev),
				Delta:    effectiveRev,
				Hormone:  "CRI",
				Epoch:    epoch,
			})
		}
	}

	return results
}

// computeDVIAdaptations computes DVI-driven parameter changes.
// DVI targets: reward.{domain}, claimstake.{domain}
func (k Keeper) computeDVIAdaptations(ctx sdk.Context, dvi types.DVIValue, epoch uint64, params types.Params) []types.Adaptation {
	var results []types.Adaptation

	rewardPath := fmt.Sprintf("reward.%s", dvi.Domain)
	stakePath := fmt.Sprintf("claimstake.%s", dvi.Domain)

	if dvi.Value > int64(types.DVIGrowingThreshold) {
		// Growing domain: increase reward, lower claim stake
		currentReward := k.GetMultiplier(ctx, rewardPath)
		rawDelta := int64(types.MaxChangePerEpochBps * 3 / 5) // +3%
		rawDelta = k.applyLearnedRate(ctx, rewardPath, rawDelta)
		effectiveDelta := types.ApplyDampedDelta(rawDelta, currentReward,
			types.BoundsDomainRewardWeight, params.MaxChangePerEpochBps)
		if effectiveDelta != 0 {
			results = append(results, types.Adaptation{
				Path:     rewardPath,
				OldValue: currentReward,
				NewValue: applySignedDelta(currentReward, effectiveDelta),
				Delta:    effectiveDelta,
				Hormone:  "DVI",
				Epoch:    epoch,
			})
		}

		currentStake := k.GetMultiplier(ctx, stakePath)
		stakeDelta := -int64(types.MaxChangePerEpochBps) // -5%
		stakeDelta = k.applyLearnedRate(ctx, stakePath, stakeDelta)
		effectiveStake := types.ApplyDampedDelta(stakeDelta, currentStake,
			types.BoundsClaimStakeMultiplier, params.MaxChangePerEpochBps)
		if effectiveStake != 0 {
			results = append(results, types.Adaptation{
				Path:     stakePath,
				OldValue: currentStake,
				NewValue: applySignedDelta(currentStake, effectiveStake),
				Delta:    effectiveStake,
				Hormone:  "DVI",
				Epoch:    epoch,
			})
		}
	} else if dvi.Value < -int64(types.DVIGrowingThreshold) {
		// Shrinking domain: decrease reward
		currentReward := k.GetMultiplier(ctx, rewardPath)
		rawDelta := -int64(types.MaxChangePerEpochBps * 3 / 5) // -3%
		rawDelta = k.applyLearnedRate(ctx, rewardPath, rawDelta)
		effectiveDelta := types.ApplyDampedDelta(rawDelta, currentReward,
			types.BoundsDomainRewardWeight, params.MaxChangePerEpochBps)
		if effectiveDelta != 0 {
			results = append(results, types.Adaptation{
				Path:     rewardPath,
				OldValue: currentReward,
				NewValue: applySignedDelta(currentReward, effectiveDelta),
				Delta:    effectiveDelta,
				Hormone:  "DVI",
				Epoch:    epoch,
			})
		}

		// Sustained negative → stress: incentivize reverification
		if dvi.ConsecutiveNegativeEpochs >= types.DVIStressEpochs {
			revPath := fmt.Sprintf("revbonus.empirical.%s", dvi.Domain)
			currentRev := k.GetMultiplier(ctx, revPath)
			revDelta := int64(types.MaxChangePerEpochBps * 2) // +10% (urgency)
			effectiveRev := types.ApplyDampedDelta(revDelta, currentRev,
				types.BoundsRevBonusMultiplier, params.MaxChangePerEpochBps*2) // allow double rate for stress
			if effectiveRev != 0 {
				results = append(results, types.Adaptation{
					Path:     revPath,
					OldValue: currentRev,
					NewValue: applySignedDelta(currentRev, effectiveRev),
					Delta:    effectiveRev,
					Hormone:  "DVI",
					Epoch:    epoch,
				})
			}
		}
	}

	return results
}

// computeVCFAdaptations computes VCF-driven parameter changes.
// VCF targets: citation.{domain}, domainprior.{domain}
func (k Keeper) computeVCFAdaptations(ctx sdk.Context, vcf types.VCFValue, epoch uint64, params types.Params) []types.Adaptation {
	var results []types.Adaptation

	citationPath := fmt.Sprintf("citation.%s", vcf.Domain)
	priorPath := fmt.Sprintf("domainprior.%s", vcf.Domain)

	if vcf.Value > types.VCFHighCascade {
		// High cascade: increase citation weight + confidence prior
		currentCitation := k.GetMultiplier(ctx, citationPath)
		rawDelta := int64(types.MaxChangePerEpochBps * 2) // +10%
		rawDelta = k.applyLearnedRate(ctx, citationPath, rawDelta)
		effectiveDelta := types.ApplyDampedDelta(rawDelta, currentCitation,
			types.BoundsCitationEvidenceWeight, params.MaxChangePerEpochBps)
		if effectiveDelta != 0 {
			results = append(results, types.Adaptation{
				Path:     citationPath,
				OldValue: currentCitation,
				NewValue: applySignedDelta(currentCitation, effectiveDelta),
				Delta:    effectiveDelta,
				Hormone:  "VCF",
				Epoch:    epoch,
			})
		}

		currentPrior := k.GetMultiplier(ctx, priorPath)
		priorDelta := int64(types.MaxChangePerEpochBps / 5) // +1%
		priorDelta = k.applyLearnedRate(ctx, priorPath, priorDelta)
		effectivePrior := types.ApplyDampedDelta(priorDelta, currentPrior,
			types.BoundsDomainConfidencePrior, params.MaxChangePerEpochBps)
		if effectivePrior != 0 {
			results = append(results, types.Adaptation{
				Path:     priorPath,
				OldValue: currentPrior,
				NewValue: applySignedDelta(currentPrior, effectivePrior),
				Delta:    effectivePrior,
				Hormone:  "VCF",
				Epoch:    epoch,
			})
		}
	} else if vcf.Value < types.VCFLowCascade {
		// Low cascade: decrease citation weight
		currentCitation := k.GetMultiplier(ctx, citationPath)
		rawDelta := -int64(types.MaxChangePerEpochBps) // -5%
		rawDelta = k.applyLearnedRate(ctx, citationPath, rawDelta)
		effectiveDelta := types.ApplyDampedDelta(rawDelta, currentCitation,
			types.BoundsCitationEvidenceWeight, params.MaxChangePerEpochBps)
		if effectiveDelta != 0 {
			results = append(results, types.Adaptation{
				Path:     citationPath,
				OldValue: currentCitation,
				NewValue: applySignedDelta(currentCitation, effectiveDelta),
				Delta:    effectiveDelta,
				Hormone:  "VCF",
				Epoch:    epoch,
			})
		}
	}

	return results
}

// applyLearnedRate scales a raw delta by the meta-learning rate multiplier for the path.
// If no learning record exists, the rate is 1.0x (no change).
func (k Keeper) applyLearnedRate(ctx sdk.Context, path string, rawDelta int64) int64 {
	rate := k.GetAdaptationRate(ctx, path)
	if rate == types.DefaultRateMultiplier {
		return rawDelta
	}
	// Scale: rawDelta * rate / BasisPoints
	return (rawDelta * int64(rate)) / int64(types.BasisPoints)
}

// applySignedDelta applies a signed BPS delta to a uint64 value, clamping at 0.
func applySignedDelta(value uint64, delta int64) uint64 {
	if delta >= 0 {
		return value + uint64(delta)
	}
	sub := uint64(-delta)
	if sub > value {
		return 0
	}
	return value - sub
}

// ============ Metric Collection ============

// collectMetrics gathers chain metrics from dependent keepers.
func (k Keeper) collectMetrics(ctx sdk.Context, epoch, blockHeight, epochStart uint64) types.EpochObservation {
	obs := types.EpochObservation{
		Epoch:       epoch,
		BlockHeight: blockHeight,
	}

	if k.stakingKeeper != nil {
		obs.ActiveValidators = k.stakingKeeper.GetActiveValidatorCount(ctx)
		obs.TotalValidators = k.stakingKeeper.GetTotalValidatorCount(ctx)
		obs.ValidatorAccuracyBps = k.stakingKeeper.GetAverageValidatorAccuracy(ctx)
		obs.SlashEventsThisEpoch = k.stakingKeeper.GetSlashEventsInRange(ctx, epochStart, blockHeight)
	}

	if k.knowledgeKeeper != nil {
		obs.TotalActiveFacts = k.knowledgeKeeper.GetActiveFactCount(ctx)
		obs.FactsCreated = k.knowledgeKeeper.GetFactsCreatedInRange(ctx, epochStart, blockHeight)
		obs.FactsExpired = k.knowledgeKeeper.GetFactsExpiredInRange(ctx, epochStart, blockHeight)
		obs.ChallengeSuccessRateBps = k.knowledgeKeeper.GetChallengeSuccessRate(ctx)

		// Domain-level metrics for CRI/DVI/VCF
		domains := k.knowledgeKeeper.GetActiveDomains(ctx)
		for _, domain := range domains {
			domObs := k.knowledgeKeeper.GetDomainObservation(ctx, domain, epochStart, blockHeight)
			obs.Domains = append(obs.Domains, domObs)
		}
		obs.Retentions = k.knowledgeKeeper.GetCategoryRetentions(ctx, epochStart, blockHeight)
	}

	return obs
}

// ============ Alignment Integration ============

// GetAllMultiplierEntries returns all stored multiplier entries.
// Used by the alignment module's Harmony sensor to measure variance, saturation, etc.
func (k Keeper) GetAllMultiplierEntries(ctx sdk.Context) []types.MultiplierEntry {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.MultiplierKeyPrefix, append(types.MultiplierKeyPrefix, 0xFF))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var entries []types.MultiplierEntry
	for ; iter.Valid(); iter.Next() {
		var entry types.MultiplierEntry
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

// GetAdaptationCount returns the number of adaptations recorded for an epoch.
func (k Keeper) GetAdaptationCount(ctx sdk.Context, epoch uint64) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(adaptationKey(epoch))
	if err != nil || bz == nil {
		return 0
	}
	var adaptations []types.Adaptation
	if err := json.Unmarshal(bz, &adaptations); err != nil {
		return 0
	}
	return uint64(len(adaptations))
}

// GetConflictCount returns the number of path conflicts for an epoch.
// A conflict occurs when a hormone targets a path already claimed by a higher-priority hormone.
func (k Keeper) GetConflictCount(ctx sdk.Context, epoch uint64) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(adaptationKey(epoch))
	if err != nil || bz == nil {
		return 0
	}
	var adaptations []types.Adaptation
	if err := json.Unmarshal(bz, &adaptations); err != nil {
		return 0
	}
	// Count entries with "DVI+VCF" as they represent resolved conflicts
	var conflicts uint64
	for _, a := range adaptations {
		if a.Hormone == "DVI+VCF" {
			conflicts++
		}
	}
	return conflicts
}

// GetDirectionChanges returns the number of multiplier direction reversals
// over the last N epochs. A direction change is when a multiplier goes from
// increasing to decreasing or vice versa.
func (k Keeper) GetDirectionChanges(ctx sdk.Context, lastNEpochs uint64) uint64 {
	state := k.GetState(ctx)
	if state.CurrentEpoch == 0 {
		return 0
	}

	startEpoch := uint64(0)
	if state.CurrentEpoch > lastNEpochs {
		startEpoch = state.CurrentEpoch - lastNEpochs
	}

	// Track direction per path across epochs
	type direction struct {
		path      string
		lastDelta int64
	}
	dirs := make(map[string]int64) // path → last seen delta sign
	var changes uint64

	for e := startEpoch; e <= state.CurrentEpoch; e++ {
		kvStore := k.storeService.OpenKVStore(ctx)
		bz, err := kvStore.Get(adaptationKey(e))
		if err != nil || bz == nil {
			continue
		}
		var adaptations []types.Adaptation
		if err := json.Unmarshal(bz, &adaptations); err != nil {
			continue
		}
		for _, a := range adaptations {
			if a.Delta == 0 {
				continue
			}
			prevSign, exists := dirs[a.Path]
			currentSign := int64(1)
			if a.Delta < 0 {
				currentSign = -1
			}
			if exists && prevSign != currentSign {
				changes++
			}
			dirs[a.Path] = currentSign
		}
	}

	return changes
}
