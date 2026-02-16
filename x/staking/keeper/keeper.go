package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/staking/types"
)

type Keeper struct {
	cdc               codec.Codec
	storeService      store.KVStoreService
	accountKeeper     types.AccountKeeper
	bankKeeper        types.BankKeeper
	authKeeper        types.LegibleAuthKeeper
	autopoiesisKeeper types.AutopoiesisKeeper
	authority         string // module authority (gov module)
}

func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	authKeeper types.LegibleAuthKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
		authKeeper:    authKeeper,
		authority:     authority,
	}
}

// SetAutopoiesisKeeper sets the autopoiesis keeper after init (avoids circular dep).
func (k *Keeper) SetAutopoiesisKeeper(ak types.AutopoiesisKeeper) {
	k.autopoiesisKeeper = ak
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// GetAuthority returns the module's governance authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
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

// --- Validator Operations ---

// SetValidator stores a validator
func (k Keeper) SetValidator(ctx sdk.Context, validator types.LegibleValidator) error {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&validator)
	if err != nil {
		return err
	}

	key := types.ValidatorKey(validator.OperatorAddress)
	if err := store.Set(key, bz); err != nil {
		return err
	}

	// Maintain DID -> address index
	didKey := types.ValidatorByDIDKey(validator.Did)
	return store.Set(didKey, []byte(validator.OperatorAddress))
}

// GetValidator retrieves a validator by operator address
func (k Keeper) GetValidator(ctx sdk.Context, operatorAddr string) (types.LegibleValidator, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.ValidatorKey(operatorAddr)

	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.LegibleValidator{}, false
	}

	var validator types.LegibleValidator
	if err := proto.Unmarshal(bz, &validator); err != nil {
		return types.LegibleValidator{}, false
	}

	return validator, true
}

// GetValidatorByDID retrieves a validator by DID
func (k Keeper) GetValidatorByDID(ctx sdk.Context, did string) (types.LegibleValidator, bool) {
	store := k.storeService.OpenKVStore(ctx)
	didKey := types.ValidatorByDIDKey(did)

	operatorAddrBz, err := store.Get(didKey)
	if err != nil || operatorAddrBz == nil {
		return types.LegibleValidator{}, false
	}

	return k.GetValidator(ctx, string(operatorAddrBz))
}

// DeleteValidator removes a validator (use cautiously)
func (k Keeper) DeleteValidator(ctx sdk.Context, operatorAddr string) error {
	validator, found := k.GetValidator(ctx, operatorAddr)
	if !found {
		return types.ErrValidatorNotFound
	}

	store := k.storeService.OpenKVStore(ctx)

	// Remove from primary store
	if err := store.Delete(types.ValidatorKey(operatorAddr)); err != nil {
		return err
	}

	// Remove DID index
	return store.Delete(types.ValidatorByDIDKey(validator.Did))
}

// IterateValidators iterates over all validators
func (k Keeper) IterateValidators(ctx sdk.Context, cb func(types.LegibleValidator) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iterator, err := store.Iterator(types.ValidatorKeyPrefix, prefixEndBytes(types.ValidatorKeyPrefix))
	if err != nil {
		k.Logger(ctx).Error("failed to create validator iterator", "error", err)
		return
	}
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var validator types.LegibleValidator
		if err := proto.Unmarshal(iterator.Value(), &validator); err != nil {
			continue
		}
		if cb(validator) {
			break
		}
	}
}

// GetActiveValidatorSet returns all active validators
func (k Keeper) GetActiveValidatorSet(ctx sdk.Context) []types.LegibleValidator {
	var validators []types.LegibleValidator
	k.IterateValidators(ctx, func(v types.LegibleValidator) bool {
		if v.IsActive {
			validators = append(validators, v)
		}
		return false
	})
	return validators
}

// CountBlockProducers counts tier 2+ (bonded/guardian) validators eligible
// for CometBFT block production. MaxValidators cap applies only to these.
func (k Keeper) CountBlockProducers(ctx sdk.Context) uint32 {
	var count uint32
	k.IterateValidators(ctx, func(v types.LegibleValidator) bool {
		tier := types.ValidatorTierFromString(v.Tier)
		if tier >= types.TierBonded {
			count++
		}
		return false
	})
	return count
}

// --- Delegation Operations ---

// SetDelegation stores a delegation
func (k Keeper) SetDelegation(ctx sdk.Context, delegation types.Delegation) error {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&delegation)
	if err != nil {
		return err
	}

	key := types.DelegationKey(delegation.DelegatorAddress, delegation.ValidatorAddress)
	return store.Set(key, bz)
}

// GetDelegation retrieves a delegation
func (k Keeper) GetDelegation(ctx sdk.Context, delegatorAddr, validatorAddr string) (types.Delegation, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.DelegationKey(delegatorAddr, validatorAddr)

	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.Delegation{}, false
	}

	var delegation types.Delegation
	if err := proto.Unmarshal(bz, &delegation); err != nil {
		return types.Delegation{}, false
	}

	return delegation, true
}

// GetDelegationsForValidator returns all delegations to a validator
func (k Keeper) GetDelegationsForValidator(ctx sdk.Context, validatorAddr string) []types.Delegation {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.DelegationsByValidatorPrefix(validatorAddr)
	iterator, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		k.Logger(ctx).Error("failed to create delegation iterator", "validator", validatorAddr, "error", err)
		return nil
	}
	defer iterator.Close()

	var delegations []types.Delegation
	for ; iterator.Valid(); iterator.Next() {
		var delegation types.Delegation
		if err := proto.Unmarshal(iterator.Value(), &delegation); err != nil {
			continue
		}
		delegations = append(delegations, delegation)
	}

	return delegations
}

// DeleteDelegation removes a delegation
func (k Keeper) DeleteDelegation(ctx sdk.Context, delegatorAddr, validatorAddr string) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.DelegationKey(delegatorAddr, validatorAddr)
	return store.Delete(key)
}

// --- Unbonding Operations ---

// SetUnbonding stores an unbonding entry
func (k Keeper) SetUnbonding(ctx sdk.Context, unbonding types.UnbondingEntry) error {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&unbonding)
	if err != nil {
		return err
	}

	key := types.UnbondingKey(unbonding.Id)
	return store.Set(key, bz)
}

// GetUnbonding retrieves an unbonding entry
func (k Keeper) GetUnbonding(ctx sdk.Context, id string) (types.UnbondingEntry, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.UnbondingKey(id)

	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.UnbondingEntry{}, false
	}

	var unbonding types.UnbondingEntry
	if err := proto.Unmarshal(bz, &unbonding); err != nil {
		return types.UnbondingEntry{}, false
	}

	return unbonding, true
}

// GetMatureUnbondings returns unbondings that have completed
func (k Keeper) GetMatureUnbondings(ctx sdk.Context, currentHeight uint64) []types.UnbondingEntry {
	store := k.storeService.OpenKVStore(ctx)
	iterator, err := store.Iterator(types.UnbondingKeyPrefix, prefixEndBytes(types.UnbondingKeyPrefix))
	if err != nil {
		k.Logger(ctx).Error("failed to create unbonding iterator", "error", err)
		return nil
	}
	defer iterator.Close()

	var mature []types.UnbondingEntry
	for ; iterator.Valid(); iterator.Next() {
		var unbonding types.UnbondingEntry
		if err := proto.Unmarshal(iterator.Value(), &unbonding); err != nil {
			continue
		}
		if unbonding.Status == "pending" && unbonding.CompletesAtHeight <= currentHeight {
			mature = append(mature, unbonding)
		}
	}

	return mature
}

// DeleteUnbonding removes an unbonding entry
func (k Keeper) DeleteUnbonding(ctx sdk.Context, id string) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.UnbondingKey(id)
	return store.Delete(key)
}

// GetUnbondingsForDelegator returns all unbonding entries for a delegator
func (k Keeper) GetUnbondingsForDelegator(ctx sdk.Context, delegatorAddr string) []types.UnbondingEntry {
	var entries []types.UnbondingEntry
	k.IterateUnbondings(ctx, func(u types.UnbondingEntry) bool {
		if u.DelegatorAddress == delegatorAddr {
			entries = append(entries, u)
		}
		return false
	})
	return entries
}

// IterateUnbondings iterates over all unbonding entries
func (k Keeper) IterateUnbondings(ctx sdk.Context, cb func(types.UnbondingEntry) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iterator, err := store.Iterator(types.UnbondingKeyPrefix, prefixEndBytes(types.UnbondingKeyPrefix))
	if err != nil {
		k.Logger(ctx).Error("failed to create unbonding iterator", "error", err)
		return
	}
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var unbonding types.UnbondingEntry
		if err := proto.Unmarshal(iterator.Value(), &unbonding); err != nil {
			continue
		}
		if cb(unbonding) {
			break
		}
	}
}

// --- Unbonding Sequence Counter ---

// NextUnbondingSeq returns the next unbonding sequence number and increments the counter.
func (k Keeper) NextUnbondingSeq(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.UnbondingSeqKey)
	var seq uint64
	if err == nil && bz != nil {
		seq = binary.BigEndian.Uint64(bz)
	}
	seq++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, seq)
	if err := store.Set(types.UnbondingSeqKey, newBz); err != nil {
		k.Logger(ctx).Error("failed to set unbonding sequence", "error", err)
	}
	return seq
}

// GetUnbondingSeq returns the current unbonding sequence number without incrementing.
func (k Keeper) GetUnbondingSeq(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.UnbondingSeqKey)
	if err != nil || bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// SetUnbondingSeq sets the unbonding sequence counter (used in genesis import).
func (k Keeper) SetUnbondingSeq(ctx sdk.Context, seq uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, seq)
	if err := store.Set(types.UnbondingSeqKey, bz); err != nil {
		k.Logger(ctx).Error("failed to set unbonding sequence", "error", err)
	}
}

// IterateDelegations iterates over all delegations
func (k Keeper) IterateDelegations(ctx sdk.Context, cb func(types.Delegation) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iterator, err := store.Iterator(types.DelegationKeyPrefix, prefixEndBytes(types.DelegationKeyPrefix))
	if err != nil {
		k.Logger(ctx).Error("failed to create delegation iterator", "error", err)
		return
	}
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var delegation types.Delegation
		if err := proto.Unmarshal(iterator.Value(), &delegation); err != nil {
			continue
		}
		if cb(delegation) {
			break
		}
	}
}

// --- Tier Configuration ---

// SetTierConfig stores a tier configuration
func (k Keeper) SetTierConfig(ctx sdk.Context, config types.TierConfig) error {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&config)
	if err != nil {
		return err
	}

	key := types.TierConfigKey(types.ValidatorTierFromString(config.Tier))
	return store.Set(key, bz)
}

// GetTierConfig retrieves a tier configuration
func (k Keeper) GetTierConfig(ctx sdk.Context, tier types.ValidatorTier) (types.TierConfig, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.TierConfigKey(tier)

	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.TierConfig{}, false
	}

	var config types.TierConfig
	if err := proto.Unmarshal(bz, &config); err != nil {
		return types.TierConfig{}, false
	}

	return config, true
}

// GetAllTierConfigs returns all tier configurations
func (k Keeper) GetAllTierConfigs(ctx sdk.Context) []types.TierConfig {
	configs := make([]types.TierConfig, 0, 4)
	for tier := types.TierApprentice; tier <= types.TierGuardian; tier++ {
		if config, found := k.GetTierConfig(ctx, tier); found {
			configs = append(configs, config)
		}
	}
	return configs
}

// --- Tier Computation (delegated to tiers.go) ---

// ComputeValidatorTierSimple calculates the tier with basic params (backward-compat helper)
func (k Keeper) ComputeValidatorTierSimple(ctx sdk.Context, stake string, verifications, correctVerifications, slashCount uint64) types.ValidatorTier {
	return ComputeValidatorTier(ctx, k, stake, verifications, correctVerifications, slashCount, 0, 0)
}

// CheckTierTransition checks if validator should transition tiers
func (k Keeper) CheckTierTransition(ctx sdk.Context, validator types.LegibleValidator) (types.ValidatorTier, bool) {
	newTier := ComputeValidatorTier(ctx, k, validator.TotalStake, validator.VerificationsCompleted, validator.VerificationsCorrect, validator.SlashCount, validator.ContestedVerificationsCorrect, validator.LastSlashedHeight)
	return newTier, types.ValidatorTierString(newTier) != validator.Tier
}

// IsTierEligibleForCategory checks if tier can verify a category
func (k Keeper) IsTierEligibleForCategory(ctx sdk.Context, tier types.ValidatorTier, category string) bool {
	return IsTierEligibleForCategory(ctx, k, tier, category)
}

// GetEffectiveSelectionStake returns virtual/real stake for VRF selection
func (k Keeper) GetEffectiveSelectionStake(ctx sdk.Context, validator types.LegibleValidator) string {
	params := k.GetParams(ctx)
	return GetEffectiveSelectionStake(ctx, k, validator, params)
}

// CalculateTierReward applies tier multiplier to base reward
func (k Keeper) CalculateTierReward(ctx sdk.Context, tier types.ValidatorTier, baseReward string) string {
	return CalculateTierReward(ctx, k, tier, baseReward)
}

// CalculateTierSlash applies tier multiplier to slash amount
func (k Keeper) CalculateTierSlash(ctx sdk.Context, tier types.ValidatorTier, baseSlash string) string {
	return CalculateTierSlash(ctx, k, tier, baseSlash)
}

// --- Performance Tracking ---

// RecordVerification updates validator verification stats
func (k Keeper) RecordVerification(ctx sdk.Context, validatorAddr string, correct bool) error {
	validator, found := k.GetValidator(ctx, validatorAddr)
	if !found {
		return types.ErrValidatorNotFound
	}

	validator.VerificationsCompleted++
	if correct {
		validator.VerificationsCorrect++
		// Increase reputation slightly
		if validator.Reputation < 1000000 {
			validator.Reputation += 100 // +0.01%
		}
	} else {
		// Decrease reputation
		if validator.Reputation > 100 {
			validator.Reputation -= 200 // -0.02%
		}
	}

	// Check for tier transition
	if newTier, changed := k.CheckTierTransition(ctx, validator); changed {
		oldTier := validator.Tier
		validator.Tier = types.ValidatorTierString(newTier)
		k.Logger(ctx).Info("validator tier changed",
			"validator", validatorAddr,
			"old_tier", oldTier,
			"new_tier", types.ValidatorTierString(newTier),
		)
	}

	return k.SetValidator(ctx, validator)
}

// SlashValidator reduces stake and potentially deactivates validator.
// R2: Per-epoch slash cap — if maxSlashesPerEpoch reached, skip slash.
// R4: Slash floor — 0-amount slashes don't increment slashCount.
func (k Keeper) SlashValidator(ctx sdk.Context, validatorAddr string, amount string, reason string) error {
	validator, found := k.GetValidator(ctx, validatorAddr)
	if !found {
		return types.ErrValidatorNotFound
	}

	params := k.GetParams(ctx)

	// R2: Per-epoch slash cap — skip if cap reached
	maxPerEpoch := params.GetMaxSlashesPerEpoch()
	if validator.SlashesThisEpoch >= uint64(maxPerEpoch) {
		k.Logger(ctx).Info("slash capped for epoch",
			"validator", validatorAddr,
			"slashes_this_epoch", validator.SlashesThisEpoch,
			"max_per_epoch", maxPerEpoch,
			"reason", reason,
		)
		return nil // skip slash but don't error
	}

	// Use the caller-provided amount directly. The caller (e.g., knowledge module)
	// already computed a stake-proportional slash using effective stake, which
	// incorporates tier differentiation via virtual/real stake. Applying
	// CalculateTierSlash here would double-count tier-based scaling.
	adjustedAmount := amount

	// Progressive escalation: adjustedAmount * (1M + slashCount * escalationBps) / 1M
	if validator.SlashCount > 0 {
		escalationBps := params.GetSlashEscalationBps()
		adjInt := new(big.Int)
		adjInt.SetString(adjustedAmount, 10)
		multiplier := new(big.Int).SetUint64(1000000 + validator.SlashCount*escalationBps)
		adjInt.Mul(adjInt, multiplier)
		adjInt.Div(adjInt, big.NewInt(1000000))
		adjustedAmount = adjInt.String()
	}

	// Apply autopoiesis SSI multiplier (1.0x default, up to 3.0x under stress)
	if k.autopoiesisKeeper != nil {
		ssiMult := k.autopoiesisKeeper.GetMultiplier(ctx, "system.slash")
		if ssiMult != 0 && ssiMult != 1_000_000 { // skip if default 1.0x
			adjInt := new(big.Int)
			adjInt.SetString(adjustedAmount, 10)
			adjInt.Mul(adjInt, new(big.Int).SetUint64(ssiMult))
			adjInt.Div(adjInt, big.NewInt(1_000_000))
			adjustedAmount = adjInt.String()
		}
	}

	// Parse stake fields
	ownStake := new(big.Int)
	ownStake.SetString(validator.Stake, 10)
	delegatedStake := new(big.Int)
	delegatedStake.SetString(validator.DelegatedStake, 10)
	totalStake := new(big.Int).Add(new(big.Int).Set(ownStake), new(big.Int).Set(delegatedStake))

	slashInt := new(big.Int)
	slashInt.SetString(adjustedAmount, 10)

	// Cap slash at total available stake
	if slashInt.Cmp(totalStake) > 0 {
		slashInt.Set(totalStake)
	}

	// Slash own stake first, then delegated for remainder
	slashFromOwn := new(big.Int).Set(slashInt)
	if slashFromOwn.Cmp(ownStake) > 0 {
		slashFromOwn.Set(ownStake)
	}
	remainingSlash := new(big.Int).Sub(slashInt, slashFromOwn)

	// Cap delegated slash at available delegated stake
	slashFromDelegated := new(big.Int).Set(remainingSlash)
	if slashFromDelegated.Cmp(delegatedStake) > 0 {
		slashFromDelegated.Set(delegatedStake)
	}

	validator.Stake = new(big.Int).Sub(ownStake, slashFromOwn).String()
	validator.DelegatedStake = new(big.Int).Sub(delegatedStake, slashFromDelegated).String()
	newStake := new(big.Int).Sub(totalStake, slashInt)
	validator.TotalStake = newStake.String()

	// Explicitly burn slashed tokens from the staking module account.
	// If burn fails, abort the slash entirely to prevent state/supply discrepancy.
	if slashInt.Sign() > 0 {
		burnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(slashInt)))
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
			return fmt.Errorf("slash aborted: failed to burn %s ulgm from staking module: %w", slashInt.String(), err)
		}
	}

	// R4: Slash floor — only increment slashCount if actual slashed amount > 0
	if slashInt.Sign() > 0 {
		validator.SlashCount++
		validator.SlashesThisEpoch++
		validator.LastSlashedHeight = uint64(ctx.BlockHeight())
	}

	// Decrease reputation significantly
	if validator.Reputation > 10000 {
		validator.Reputation -= 10000 // -1%
	} else {
		validator.Reputation = 0
	}

	// Check if validator should be deactivated (configurable, default 3)
	maxSlashCount := uint64(params.GetMaxSlashCountDeactivate())
	if validator.SlashCount >= maxSlashCount {
		validator.IsActive = false
		k.Logger(ctx).Info("validator deactivated due to slashes",
			"validator", validatorAddr,
			"slash_count", validator.SlashCount,
			"reason", reason,
		)
	}

	// Check for tier demotion
	if newTier, changed := k.CheckTierTransition(ctx, validator); changed {
		oldTier := validator.Tier
		validator.Tier = types.ValidatorTierString(newTier)
		k.Logger(ctx).Info("validator tier demoted after slash",
			"validator", validatorAddr,
			"old_tier", oldTier,
			"new_tier", types.ValidatorTierString(newTier),
		)
	}

	return k.SetValidator(ctx, validator)
}

// --- Redelegation Cooldown ---

// SetLastRedelegationHeight records the height at which a delegator last redelegated.
func (k Keeper) SetLastRedelegationHeight(ctx sdk.Context, delegatorAddr string, height uint64) error {
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, height)
	return store.Set(types.RedelegationCooldownKey(delegatorAddr), bz)
}

// GetLastRedelegationHeight returns the height at which a delegator last redelegated.
func (k Keeper) GetLastRedelegationHeight(ctx sdk.Context, delegatorAddr string) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.RedelegationCooldownKey(delegatorAddr))
	if err != nil || bz == nil || len(bz) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// --- Params ---

// SetParams stores module parameters
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&params)
	if err != nil {
		return err
	}
	return store.Set(types.ParamsKey, bz)
}

// GetParams retrieves module parameters
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

// --- Genesis ---

// InitGenesis initializes the module from genesis state
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) error {
	// Set params
	if gs.Params != nil {
		if err := k.SetParams(ctx, *gs.Params); err != nil {
			return err
		}
	}

	// Set tier configs
	for _, config := range gs.TierConfigs {
		if config != nil {
			if err := k.SetTierConfig(ctx, *config); err != nil {
				return err
			}
		}
	}

	// Set validators
	for _, validator := range gs.Validators {
		if validator != nil {
			if err := k.SetValidator(ctx, *validator); err != nil {
				return err
			}
		}
	}

	// Set delegations
	for _, delegation := range gs.Delegations {
		if delegation != nil {
			if err := k.SetDelegation(ctx, *delegation); err != nil {
				return err
			}
		}
	}

	// Set unbonding entries
	for _, unbonding := range gs.UnbondingEntries {
		if unbonding != nil {
			if err := k.SetUnbonding(ctx, *unbonding); err != nil {
				return err
			}
		}
	}

	// Restore unbonding sequence counter
	if gs.UnbondingSeq > 0 {
		k.SetUnbondingSeq(ctx, gs.UnbondingSeq)
	}

	return nil
}

// ExportGenesis exports the module state
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	tierConfigs := k.GetAllTierConfigs(ctx)

	var validators []*types.LegibleValidator
	k.IterateValidators(ctx, func(v types.LegibleValidator) bool {
		validators = append(validators, &v)
		return false
	})

	var delegations []*types.Delegation
	k.IterateDelegations(ctx, func(d types.Delegation) bool {
		delegations = append(delegations, &d)
		return false
	})

	var unbondings []*types.UnbondingEntry
	k.IterateUnbondings(ctx, func(u types.UnbondingEntry) bool {
		unbondings = append(unbondings, &u)
		return false
	})

	// Convert []TierConfig to []*TierConfig
	tierConfigPtrs := make([]*types.TierConfig, len(tierConfigs))
	for i := range tierConfigs {
		tierConfigPtrs[i] = &tierConfigs[i]
	}

	return &types.GenesisState{
		Params:           &params,
		Validators:       validators,
		Delegations:      delegations,
		UnbondingEntries: unbondings,
		TierConfigs:      tierConfigPtrs,
		UnbondingSeq:     k.GetUnbondingSeq(ctx),
	}
}

// --- Msg Server Implementation ---

// msgServer implements both the proto-generated MsgServer (RegisterValidator)
// and the extended FullMsgServer (Delegate, Undelegate, Redelegate, UpdateValidatorStake).
type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

var _ types.MsgServer = &msgServer{}

// NewMsgServerImpl returns an implementation of the staking MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (k msgServer) RegisterValidator(goCtx context.Context, msg *types.MsgRegisterValidator) (*types.MsgRegisterValidatorResponse, error) {
	return k.Keeper.RegisterValidator(sdk.UnwrapSDKContext(goCtx), msg)
}

func (k msgServer) UpdateValidatorStake(ctx sdk.Context, msg *types.MsgUpdateValidatorStake) (*types.MsgUpdateValidatorStakeResponse, error) {
	return k.Keeper.UpdateValidatorStake(ctx, msg)
}

func (k msgServer) Delegate(ctx sdk.Context, msg *types.MsgDelegate) (*types.MsgDelegateResponse, error) {
	return k.Keeper.Delegate(ctx, msg)
}

func (k msgServer) Undelegate(ctx sdk.Context, msg *types.MsgUndelegate) (*types.MsgUndelegateResponse, error) {
	return k.Keeper.Undelegate(ctx, msg)
}

func (k msgServer) Redelegate(ctx sdk.Context, msg *types.MsgRedelegate) (*types.MsgRedelegateResponse, error) {
	return k.Keeper.Redelegate(ctx, msg)
}

func (k msgServer) UpdateParams(ctx sdk.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	return k.Keeper.UpdateParams(ctx, msg)
}

// --- Block Processing ---

// BeginBlocker is called at the beginning of every block
func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	currentHeight := uint64(ctx.BlockHeight())
	params := k.GetParams(ctx)

	// R1: Slash count decay + R2: Epoch reset of per-epoch slash counter
	// Check if we're at an epoch boundary (every SlashDecayPeriodBlocks blocks)
	decayPeriod := params.GetSlashDecayPeriodBlocks()
	if decayPeriod > 0 && currentHeight > 0 && currentHeight%decayPeriod == 0 {
		var updatedValidators []types.LegibleValidator

		k.IterateValidators(ctx, func(v types.LegibleValidator) bool {
			changed := false

			// R1: Decay slashCount by 1 if no slash occurred this epoch
			if v.SlashCount > 0 && v.SlashesThisEpoch == 0 {
				v.SlashCount--
				changed = true
				k.Logger(ctx).Info("slash count decayed",
					"validator", v.OperatorAddress,
					"new_slash_count", v.SlashCount,
				)
			}

			// R2: Reset per-epoch slash counter
			if v.SlashesThisEpoch > 0 {
				v.SlashesThisEpoch = 0
				changed = true
			}

			if changed {
				updatedValidators = append(updatedValidators, v)
			}
			return false
		})

		for _, v := range updatedValidators {
			if err := k.SetValidator(ctx, v); err != nil {
				k.Logger(ctx).Error("failed to save validator after slash decay",
					"validator", v.OperatorAddress, "error", err)
			}
		}
	}

	// Process mature unbondings
	matureUnbondings := k.GetMatureUnbondings(ctx, currentHeight)

	for _, unbonding := range matureUnbondings {
		// Transfer tokens back to delegator
		addr, err := sdk.AccAddressFromBech32(unbonding.DelegatorAddress)
		if err != nil {
			k.Logger(ctx).Error("invalid delegator address in unbonding",
				"id", unbonding.Id,
				"address", unbonding.DelegatorAddress,
				"error", err,
			)
			continue
		}

		amountInt := new(big.Int)
		amountInt.SetString(unbonding.Amount, 10)
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amountInt)))

		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins); err != nil {
			k.Logger(ctx).Error("failed to complete unbonding",
				"id", unbonding.Id,
				"error", err,
			)
			continue
		}

		// Mark as completed
		unbonding.Status = "completed"
		if err := k.SetUnbonding(ctx, unbonding); err != nil {
			k.Logger(ctx).Error("failed to update unbonding status", "id", unbonding.Id, "error", err)
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"unbonding_completed",
				sdk.NewAttribute("id", unbonding.Id),
				sdk.NewAttribute("delegator", unbonding.DelegatorAddress),
				sdk.NewAttribute("validator", unbonding.ValidatorAddress),
				sdk.NewAttribute("amount", unbonding.Amount),
			),
		)
	}

	return nil
}

// EndBlocker is called at the end of every block
func (k Keeper) EndBlocker(ctx sdk.Context) error {
	// Check for tier transitions due to verification activity
	// (most transitions happen immediately on RecordVerification, but this is a safety check)
	var transitionedValidators []types.LegibleValidator

	k.IterateValidators(ctx, func(v types.LegibleValidator) bool {
		if newTier, changed := k.CheckTierTransition(ctx, v); changed {
			oldTier := v.Tier
			v.Tier = types.ValidatorTierString(newTier)
			transitionedValidators = append(transitionedValidators, v)

			k.Logger(ctx).Info("validator tier transition at end block",
				"validator", v.OperatorAddress,
				"old_tier", oldTier,
				"new_tier", types.ValidatorTierString(newTier),
			)
		}
		return false
	})

	// Save transitioned validators
	for _, v := range transitionedValidators {
		if err := k.SetValidator(ctx, v); err != nil {
			k.Logger(ctx).Error("failed to save validator tier transition",
				"validator", v.OperatorAddress,
				"error", err,
			)
		}
	}

	return nil
}
