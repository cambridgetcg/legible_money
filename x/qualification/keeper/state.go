package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/qualification/types"
)

// --- Qualifications ---

// SetQualification stores a domain qualification and maintains indexes.
func (k Keeper) SetQualification(ctx sdk.Context, q types.DomainQualification) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(q)
	if err != nil {
		panic("failed to marshal domain qualification: " + err.Error())
	}
	store.Set(types.QualificationKey(q.ValidatorAddr, q.Domain), bz)

	// Maintain domain→validator index
	store.Set(types.DomainValidatorIndexKey(q.Domain, q.ValidatorAddr), []byte{})
}

// GetQualification retrieves a domain qualification.
func (k Keeper) GetQualification(ctx sdk.Context, validatorAddr, domain string) (types.DomainQualification, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.QualificationKey(validatorAddr, domain))
	if bz == nil {
		return types.DomainQualification{}, false
	}
	var q types.DomainQualification
	if err := json.Unmarshal(bz, &q); err != nil {
		return types.DomainQualification{}, false
	}
	return q, true
}

// DeleteQualification removes a domain qualification and its indexes.
func (k Keeper) DeleteQualification(ctx sdk.Context, validatorAddr, domain string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.QualificationKey(validatorAddr, domain))
	store.Delete(types.DomainValidatorIndexKey(domain, validatorAddr))
}

// GetQualificationsByValidator returns all qualifications for a validator.
func (k Keeper) GetQualificationsByValidator(ctx sdk.Context, validatorAddr string) []types.DomainQualification {
	store := ctx.KVStore(k.storeKey)
	prefix := types.ValidatorQualificationPrefix(validatorAddr)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var qualifications []types.DomainQualification
	for ; iter.Valid(); iter.Next() {
		var q types.DomainQualification
		if err := json.Unmarshal(iter.Value(), &q); err != nil {
			continue
		}
		qualifications = append(qualifications, q)
	}
	return qualifications
}

// --- Endorsements ---

// SetEndorsement stores an endorsement and maintains indexes.
func (k Keeper) SetEndorsement(ctx sdk.Context, e types.QualificationEndorsement) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(e)
	if err != nil {
		panic("failed to marshal endorsement: " + err.Error())
	}
	store.Set(types.EndorsementKey(e.Id), bz)

	// Maintain indexes
	store.Set(types.EndorserIndexKey(e.EndorserAddr, e.Id), []byte{})
	store.Set(types.TargetIndexKey(e.TargetValidator, e.Id), []byte{})
}

// GetEndorsement retrieves an endorsement by ID.
func (k Keeper) GetEndorsement(ctx sdk.Context, id string) (types.QualificationEndorsement, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.EndorsementKey(id))
	if bz == nil {
		return types.QualificationEndorsement{}, false
	}
	var e types.QualificationEndorsement
	if err := json.Unmarshal(bz, &e); err != nil {
		return types.QualificationEndorsement{}, false
	}
	return e, true
}

// DeleteEndorsement removes an endorsement and its indexes.
func (k Keeper) DeleteEndorsement(ctx sdk.Context, e types.QualificationEndorsement) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.EndorsementKey(e.Id))
	store.Delete(types.EndorserIndexKey(e.EndorserAddr, e.Id))
	store.Delete(types.TargetIndexKey(e.TargetValidator, e.Id))
}

// GetEndorsementsForValidator returns all endorsements targeting a validator.
func (k Keeper) GetEndorsementsForValidator(ctx sdk.Context, targetAddr string) []types.QualificationEndorsement {
	store := ctx.KVStore(k.storeKey)
	prefix := types.TargetValidatorPrefix(targetAddr)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var endorsements []types.QualificationEndorsement
	for ; iter.Valid(); iter.Next() {
		// Value is empty (index entry). Extract endorsement ID from key.
		key := iter.Key()
		endorsementId := string(key[len(prefix):])
		e, found := k.GetEndorsement(ctx, endorsementId)
		if found {
			endorsements = append(endorsements, e)
		}
	}
	return endorsements
}

// GetEndorsementsForValidatorInDomain returns endorsements for a validator in a specific domain.
func (k Keeper) GetEndorsementsForValidatorInDomain(ctx sdk.Context, targetAddr, domain string) []types.QualificationEndorsement {
	all := k.GetEndorsementsForValidator(ctx, targetAddr)
	currentBlock := uint64(ctx.BlockHeight())

	var filtered []types.QualificationEndorsement
	for _, e := range all {
		if e.Domain == domain && (e.ExpiresAtBlock == 0 || e.ExpiresAtBlock > currentBlock) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// --- Endorsement Counter ---

// GetNextEndorsementId increments and returns the next endorsement ID.
func (k Keeper) GetNextEndorsementId(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.EndorsementCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(types.EndorsementCounterKey, newBz)
	return fmt.Sprintf("endorse_%d", counter)
}

// --- Params ---

// GetParams returns the qualification module parameters.
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

// SetParams stores the qualification module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic("failed to marshal qualification params: " + err.Error())
	}
	store.Set(types.ParamsKey, bz)
}
