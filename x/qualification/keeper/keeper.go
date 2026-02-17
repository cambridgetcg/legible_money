package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/qualification/types"
)

// Keeper manages the qualification module's state.
type Keeper struct {
	cdc            codec.Codec
	storeService   store.KVStoreService
	ontologyKeeper types.OntologyKeeper
	stakingKeeper  types.StakingKeeper
	bankKeeper     types.BankKeeper
	authority      string
}

// NewKeeper creates a new qualification module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	ok types.OntologyKeeper,
	sk types.StakingKeeper,
	bk types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:            cdc,
		storeService:   storeService,
		ontologyKeeper: ok,
		stakingKeeper:  sk,
		bankKeeper:     bk,
		authority:      authority,
	}
}

// prefixEndBytes returns the end key for prefix iteration.
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
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, *genState.Params)
	} else {
		k.SetParams(ctx, types.DefaultParams())
	}
	for _, q := range genState.Qualifications {
		if q != nil {
			k.SetQualification(ctx, *q)
		}
	}
	for _, e := range genState.Endorsements {
		if e != nil {
			k.SetEndorsement(ctx, *e)
		}
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params:         &params,
		Qualifications: toQualificationPointers(k.GetAllQualifications(ctx)),
		Endorsements:   toEndorsementPointers(k.GetAllEndorsements(ctx)),
	}
}

// Helper functions to convert value slices to pointer slices for proto compatibility.
func toQualificationPointers(qualifications []types.DomainQualification) []*types.DomainQualification {
	result := make([]*types.DomainQualification, len(qualifications))
	for i := range qualifications {
		result[i] = &qualifications[i]
	}
	return result
}

func toEndorsementPointers(endorsements []types.QualificationEndorsement) []*types.QualificationEndorsement {
	result := make([]*types.QualificationEndorsement, len(endorsements))
	for i := range endorsements {
		result[i] = &endorsements[i]
	}
	return result
}

// --- Cross-module interface (consumed by x/knowledge) ---

// IsQualified returns true if the validator has an active or probationary qualification in the domain.
func (k Keeper) IsQualified(ctx sdk.Context, validatorAddr, domain string) bool {
	q, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		return false
	}
	if q.Status != string(types.StatusActive) && q.Status != string(types.StatusProbationary) {
		return false
	}
	if q.ExpiresAtBlock > 0 && uint64(ctx.BlockHeight()) > q.ExpiresAtBlock {
		return false
	}
	return true
}

// GetQualificationWeight returns the effective weight (basis points) for a validator in a domain.
// Returns 0 if not qualified.
func (k Keeper) GetQualificationWeight(ctx sdk.Context, validatorAddr, domain string) uint32 {
	q, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		return 0
	}
	if q.Status != string(types.StatusActive) && q.Status != string(types.StatusProbationary) {
		return 0
	}
	if q.ExpiresAtBlock > 0 && uint64(ctx.BlockHeight()) > q.ExpiresAtBlock {
		return 0
	}
	return k.calculateWeight(q)
}

// GetQualifiedValidators returns all validator addresses qualified in a domain.
func (k Keeper) GetQualifiedValidators(ctx sdk.Context, domain string) []string {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.DomainValidatorPrefix(domain)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var validators []string
	for ; iter.Valid(); iter.Next() {
		// Key is: prefix | domain | "/" | validatorAddr
		// The value is empty (index entry); extract validatorAddr from key suffix
		key := iter.Key()
		// The full key starts with the prefix bytes, so validatorAddr starts after prefix
		validatorAddr := string(key[len(prefix):])

		if k.IsQualified(ctx, validatorAddr, domain) {
			validators = append(validators, validatorAddr)
		}
	}
	return validators
}

// RecordVerificationOutcome updates a validator's domain metrics after a verification.
func (k Keeper) RecordVerificationOutcome(ctx sdk.Context, validatorAddr, domain string, correct bool) error {
	q, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		return nil // no qualification to update — not an error
	}

	// Ensure Metrics is initialized
	if q.Metrics == nil {
		q.Metrics = &types.QualificationMetrics{}
	}

	q.Metrics.TotalVerifications++
	q.Metrics.LastVerificationBlock = uint64(ctx.BlockHeight())

	if correct {
		q.Metrics.CorrectVerifications++
		q.Metrics.ConsecutiveCorrect++
		q.Metrics.ConsecutiveIncorrect = 0
	} else {
		q.Metrics.ConsecutiveIncorrect++
		q.Metrics.ConsecutiveCorrect = 0
	}

	// Recalculate accuracy
	if q.Metrics.TotalVerifications > 0 {
		q.Metrics.AccuracyBps = uint32(uint64(q.Metrics.CorrectVerifications) * 10000 / uint64(q.Metrics.TotalVerifications))
	}

	// Check for suspension triggers
	params := k.GetParams(ctx)
	if q.Status == string(types.StatusActive) || q.Status == string(types.StatusProbationary) {
		if q.Metrics.AccuracyBps < params.SuspensionAccuracyBps && q.Metrics.TotalVerifications >= params.MinVerifications {
			q.Status = string(types.StatusSuspended)
		}
		if q.Metrics.ConsecutiveIncorrect >= params.SuspensionConsecutive {
			q.Status = string(types.StatusSuspended)
		}
		if q.Metrics.ChallengesLost >= params.RevocationChallenges {
			q.Status = string(types.StatusRevoked)
		}
	}

	// Recalculate weight
	q.Metrics.EffectiveWeightBps = k.calculateWeight(q)

	k.SetQualification(ctx, q)
	return nil
}

// calculateWeight computes the effective weight for a qualification.
func (k Keeper) calculateWeight(q types.DomainQualification) uint32 {
	weight := uint32(10000) // 100% base

	// Apply inheritance discount
	if q.InheritanceDiscountBps > 0 && q.InheritanceDiscountBps < 10000 {
		weight = weight * q.InheritanceDiscountBps / 10000
	}

	// Apply probationary discount
	if q.Status == string(types.StatusProbationary) {
		params, _ := k.getParamsFromQualification(q)
		weight = weight * params.ProbationaryWeightBps / 10000
	}

	// Cap at 12000 bps (120%) for accuracy bonus
	if q.Metrics != nil && q.Metrics.AccuracyBps > 7700 && q.Metrics.TotalVerifications > 0 {
		bonus := (q.Metrics.AccuracyBps - 7700) / 2 // up to ~1150 bonus bps
		if bonus > 2000 {
			bonus = 2000
		}
		weight = weight + weight*bonus/10000
	}

	return weight
}

// getParamsFromQualification is a helper to get params without requiring ctx.
// Falls back to defaults when we don't have context.
func (k Keeper) getParamsFromQualification(_ types.DomainQualification) (types.Params, bool) {
	return types.DefaultParams(), true
}

// --- Helper: calculate required stake based on domain stratum ---

func (k Keeper) requiredStakeForDomain(ctx sdk.Context, domain string) string {
	params := k.GetParams(ctx)

	if k.ontologyKeeper == nil {
		return params.StakeLeaf // fallback
	}

	d, found := k.ontologyKeeper.GetDomain(ctx, domain)
	if !found {
		return params.StakeLeaf
	}

	// Map stratum to stake tier: 0-1 = top, 2-3 = mid, 4+ = leaf
	switch {
	case d.Stratum <= 1:
		return params.StakeTop
	case d.Stratum <= 3:
		return params.StakeMid
	default:
		return params.StakeLeaf
	}
}

// GetAllQualifications returns all qualifications in the store.
func (k Keeper) GetAllQualifications(ctx sdk.Context) []types.DomainQualification {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.QualificationKeyPrefix, prefixEndBytes(types.QualificationKeyPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var qualifications []types.DomainQualification
	for ; iter.Valid(); iter.Next() {
		var q types.DomainQualification
		if err := proto.Unmarshal(iter.Value(), &q); err != nil {
			continue
		}
		qualifications = append(qualifications, q)
	}
	return qualifications
}

// GetAllEndorsements returns all endorsements in the store.
func (k Keeper) GetAllEndorsements(ctx sdk.Context) []types.QualificationEndorsement {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.EndorsementKeyPrefix, prefixEndBytes(types.EndorsementKeyPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var endorsements []types.QualificationEndorsement
	for ; iter.Valid(); iter.Next() {
		var e types.QualificationEndorsement
		if err := proto.Unmarshal(iter.Value(), &e); err != nil {
			continue
		}
		endorsements = append(endorsements, e)
	}
	return endorsements
}
