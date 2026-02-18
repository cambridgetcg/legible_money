package keeper

import (
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/qualification/types"
)

// --- Qualifications ---

// SetQualification stores a domain qualification and maintains indexes.
func (k Keeper) SetQualification(ctx sdk.Context, q *types.DomainQualification) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(q)
	if err != nil {
		panic("failed to marshal domain qualification: " + err.Error())
	}
	if err := store.Set(types.QualificationKey(q.ValidatorAddr, q.Domain), bz); err != nil {
		panic("failed to set qualification: " + err.Error())
	}

	// Maintain domain→validator index
	if err := store.Set(types.DomainValidatorIndexKey(q.Domain, q.ValidatorAddr), []byte{}); err != nil {
		panic("failed to set domain-validator index: " + err.Error())
	}
}

// GetQualification retrieves a domain qualification.
func (k Keeper) GetQualification(ctx sdk.Context, validatorAddr, domain string) (*types.DomainQualification, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.QualificationKey(validatorAddr, domain))
	if err != nil || bz == nil {
		return nil, false
	}
	var q types.DomainQualification
	if err := proto.Unmarshal(bz, &q); err != nil {
		return nil, false
	}
	return &q, true
}

// DeleteQualification removes a domain qualification and its indexes.
func (k Keeper) DeleteQualification(ctx sdk.Context, validatorAddr, domain string) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.QualificationKey(validatorAddr, domain))
	_ = store.Delete(types.DomainValidatorIndexKey(domain, validatorAddr))
}

// GetQualificationsByValidator returns all qualifications for a validator.
func (k Keeper) GetQualificationsByValidator(ctx sdk.Context, validatorAddr string) []*types.DomainQualification {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.ValidatorQualificationPrefix(validatorAddr)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var qualifications []*types.DomainQualification
	for ; iter.Valid(); iter.Next() {
		var q types.DomainQualification
		if err := proto.Unmarshal(iter.Value(), &q); err != nil {
			continue
		}
		qualifications = append(qualifications, &q)
	}
	return qualifications
}

// --- Endorsements ---

// SetEndorsement stores an endorsement and maintains indexes.
func (k Keeper) SetEndorsement(ctx sdk.Context, e *types.QualificationEndorsement) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(e)
	if err != nil {
		panic("failed to marshal endorsement: " + err.Error())
	}
	if err := store.Set(types.EndorsementKey(e.Id), bz); err != nil {
		panic("failed to set endorsement: " + err.Error())
	}

	// Maintain indexes
	if err := store.Set(types.EndorserIndexKey(e.EndorserAddr, e.Id), []byte{}); err != nil {
		panic("failed to set endorser index: " + err.Error())
	}
	if err := store.Set(types.TargetIndexKey(e.TargetValidator, e.Id), []byte{}); err != nil {
		panic("failed to set target index: " + err.Error())
	}
}

// GetEndorsement retrieves an endorsement by ID.
func (k Keeper) GetEndorsement(ctx sdk.Context, id string) (*types.QualificationEndorsement, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.EndorsementKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var e types.QualificationEndorsement
	if err := proto.Unmarshal(bz, &e); err != nil {
		return nil, false
	}
	return &e, true
}

// DeleteEndorsement removes an endorsement and its indexes.
func (k Keeper) DeleteEndorsement(ctx sdk.Context, e *types.QualificationEndorsement) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(types.EndorsementKey(e.Id))
	_ = store.Delete(types.EndorserIndexKey(e.EndorserAddr, e.Id))
	_ = store.Delete(types.TargetIndexKey(e.TargetValidator, e.Id))
}

// GetEndorsementsForValidator returns all endorsements targeting a validator.
func (k Keeper) GetEndorsementsForValidator(ctx sdk.Context, targetAddr string) []*types.QualificationEndorsement {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.TargetValidatorPrefix(targetAddr)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var endorsements []*types.QualificationEndorsement
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
func (k Keeper) GetEndorsementsForValidatorInDomain(ctx sdk.Context, targetAddr, domain string) []*types.QualificationEndorsement {
	all := k.GetEndorsementsForValidator(ctx, targetAddr)
	currentBlock := uint64(ctx.BlockHeight())

	var filtered []*types.QualificationEndorsement
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
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.EndorsementCounterKey)
	var counter uint64
	if err == nil && bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	if err := store.Set(types.EndorsementCounterKey, newBz); err != nil {
		panic("failed to set endorsement counter: " + err.Error())
	}
	return fmt.Sprintf("endorse_%d", counter)
}

// --- Params ---

// GetParams returns the qualification module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// SetParams stores the qualification module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic("failed to marshal qualification params: " + err.Error())
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic("failed to set qualification params: " + err.Error())
	}
}
