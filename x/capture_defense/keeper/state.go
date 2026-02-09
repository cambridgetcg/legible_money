package keeper

import (
	"encoding/json"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_defense/types"
)

// --- Qualification CRUD ---

// GetQualification returns a validator's qualification for a domain.
func (k Keeper) GetQualification(ctx sdk.Context, validatorAddr, domain string) (types.DomainQualification, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.QualificationKey(validatorAddr, domain))
	if bz == nil {
		return types.DomainQualification{}, false
	}
	var qual types.DomainQualification
	if err := json.Unmarshal(bz, &qual); err != nil {
		return types.DomainQualification{}, false
	}
	return qual, true
}

// SetQualification stores a validator's qualification for a domain.
func (k Keeper) SetQualification(ctx sdk.Context, qual types.DomainQualification) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(qual)
	if err != nil {
		panic("failed to marshal qualification: " + err.Error())
	}
	store.Set(types.QualificationKey(qual.ValidatorAddr, qual.Domain), bz)
}

// DeleteQualification removes a validator's qualification for a domain.
func (k Keeper) DeleteQualification(ctx sdk.Context, validatorAddr, domain string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.QualificationKey(validatorAddr, domain))
}

// GetValidatorQualifications returns all qualifications for a validator.
func (k Keeper) GetValidatorQualifications(ctx sdk.Context, validatorAddr string) []types.DomainQualification {
	store := ctx.KVStore(k.storeKey)
	prefix := types.QualificationsByValidatorPrefix(validatorAddr)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var quals []types.DomainQualification
	for ; iter.Valid(); iter.Next() {
		var qual types.DomainQualification
		if err := json.Unmarshal(iter.Value(), &qual); err != nil {
			continue
		}
		quals = append(quals, qual)
	}
	return quals
}

// IterateQualifications iterates over all qualifications.
func (k Keeper) IterateQualifications(ctx sdk.Context, cb func(types.DomainQualification) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.QualificationKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var qual types.DomainQualification
		if err := json.Unmarshal(iter.Value(), &qual); err != nil {
			continue
		}
		if cb(qual) {
			break
		}
	}
}

// --- Reputation CRUD ---

// GetIsolatedReputation returns a validator's isolated reputation.
func (k Keeper) GetIsolatedReputation(ctx sdk.Context, validatorAddr string) (types.IsolatedReputation, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GlobalReputationKey(validatorAddr))
	if bz == nil {
		return types.IsolatedReputation{}, false
	}
	var rep types.IsolatedReputation
	if err := json.Unmarshal(bz, &rep); err != nil {
		return types.IsolatedReputation{}, false
	}
	return rep, true
}

// SetIsolatedReputation stores a validator's isolated reputation.
func (k Keeper) SetIsolatedReputation(ctx sdk.Context, rep types.IsolatedReputation) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(rep)
	if err != nil {
		panic("failed to marshal reputation: " + err.Error())
	}
	store.Set(types.GlobalReputationKey(rep.ValidatorAddr), bz)
}

// GetOrCreateReputation returns existing reputation or creates a new one.
func (k Keeper) GetOrCreateReputation(ctx sdk.Context, validatorAddr string, currentBlock uint64) types.IsolatedReputation {
	rep, found := k.GetIsolatedReputation(ctx, validatorAddr)
	if found {
		return rep
	}
	return types.IsolatedReputation{
		ValidatorAddr:     validatorAddr,
		GlobalReputation:  types.GlobalReputation{Score: types.BaseScore},
		StratumReputations: make(map[uint32]types.StratumReputation),
		DomainReputations: make(map[string]types.DomainReputation),
		CreatedAtBlock:    currentBlock,
		LastUpdatedBlock:  currentBlock,
	}
}

// IterateReputations iterates over all reputations.
func (k Keeper) IterateReputations(ctx sdk.Context, cb func(types.IsolatedReputation) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.GlobalReputationKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var rep types.IsolatedReputation
		if err := json.Unmarshal(iter.Value(), &rep); err != nil {
			continue
		}
		if cb(rep) {
			break
		}
	}
}

// --- Verification History CRUD ---

// GetVerificationHistory returns the verification history for a domain.
func (k Keeper) GetVerificationHistory(ctx sdk.Context, domain string) types.VerificationHistory {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.VerificationHistoryKey(domain))
	if bz == nil {
		return types.VerificationHistory{Domain: domain, Entries: []types.VerificationHistoryEntry{}}
	}
	var history types.VerificationHistory
	if err := json.Unmarshal(bz, &history); err != nil {
		return types.VerificationHistory{Domain: domain, Entries: []types.VerificationHistoryEntry{}}
	}
	return history
}

// SetVerificationHistory stores the verification history for a domain.
func (k Keeper) SetVerificationHistory(ctx sdk.Context, history types.VerificationHistory) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(history)
	if err != nil {
		panic("failed to marshal verification history: " + err.Error())
	}
	store.Set(types.VerificationHistoryKey(history.Domain), bz)
}

// AppendVerificationHistoryEntry adds an entry and maintains a max of 100 entries per domain.
func (k Keeper) AppendVerificationHistoryEntry(ctx sdk.Context, domain string, entry types.VerificationHistoryEntry) {
	history := k.GetVerificationHistory(ctx, domain)
	history.Entries = append(history.Entries, entry)
	if len(history.Entries) > 100 {
		history.Entries = history.Entries[len(history.Entries)-100:]
	}
	k.SetVerificationHistory(ctx, history)
}

// --- Capture Metrics CRUD ---

// GetCaptureMetrics returns the capture metrics for a domain.
func (k Keeper) GetCaptureMetrics(ctx sdk.Context, domain string) (types.CaptureMetrics, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CaptureMetricsKey(domain))
	if bz == nil {
		return types.CaptureMetrics{}, false
	}
	var metrics types.CaptureMetrics
	if err := json.Unmarshal(bz, &metrics); err != nil {
		return types.CaptureMetrics{}, false
	}
	return metrics, true
}

// SetCaptureMetrics stores the capture metrics for a domain.
func (k Keeper) SetCaptureMetrics(ctx sdk.Context, metrics types.CaptureMetrics) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(metrics)
	if err != nil {
		panic("failed to marshal capture metrics: " + err.Error())
	}
	store.Set(types.CaptureMetricsKey(metrics.Domain), bz)
}

// --- Params CRUD ---

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKeyPrefix)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams stores the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic("failed to marshal params: " + err.Error())
	}
	store.Set(types.ParamsKeyPrefix, bz)
}
