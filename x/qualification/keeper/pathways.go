package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/qualification/types"
)

// QualifyViaStakeCommitment qualifies a validator by locking stake in the domain.
func (k Keeper) QualifyViaStakeCommitment(ctx sdk.Context, validatorAddr, domain, stakeStr string) (*types.DomainQualification, error) {
	// Verify domain exists
	stratum, err := k.getDomainStratum(ctx, domain)
	if err != nil {
		return nil, err
	}

	// Check not already qualified
	if _, found := k.GetQualification(ctx, validatorAddr, domain); found {
		return nil, types.ErrAlreadyQualified
	}

	// Verify validator exists
	if _, found := k.stakingKeeper.GetValidator(ctx, validatorAddr); !found {
		return nil, types.ErrValidatorNotFound
	}

	// Check stake is sufficient
	requiredStake := k.requiredStakeForDomain(ctx, domain)
	required, ok := new(big.Int).SetString(requiredStake, 10)
	if !ok {
		return nil, fmt.Errorf("invalid required stake: %s", requiredStake)
	}
	provided, ok := new(big.Int).SetString(stakeStr, 10)
	if !ok {
		return nil, fmt.Errorf("invalid provided stake: %s", stakeStr)
	}
	if provided.Cmp(required) < 0 {
		return nil, fmt.Errorf("%w: need %s, have %s", types.ErrInsufficientStake, requiredStake, stakeStr)
	}

	// Lock stake via bank keeper
	if k.bankKeeper != nil {
		addr, addrErr := sdk.AccAddressFromBech32(validatorAddr)
		if addrErr != nil {
			return nil, addrErr
		}
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(provided)))
		if sendErr := k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins); sendErr != nil {
			return nil, fmt.Errorf("%w: %v", types.ErrInsufficientStake, sendErr)
		}
	}

	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	q := types.DomainQualification{
		ValidatorAddr:          validatorAddr,
		Domain:                 domain,
		Stratum:                stratum,
		Pathway:                types.PathwayStakeCommitment,
		Status:                 types.StatusProbationary,
		EarnedAtBlock:          currentBlock,
		ExpiresAtBlock:         0, // no expiry for stake commitment
		Metrics:                emptyMetrics(),
		LockedStake:            stakeStr,
		StakeUnlocksAt:         currentBlock + params.StakeLockPeriod,
		InheritanceDiscountBps: 10000, // no discount
	}

	k.SetQualification(ctx, q)
	return &q, nil
}

// QualifyViaTrackRecord qualifies a validator based on their verification history in a parent/adjacent domain.
func (k Keeper) QualifyViaTrackRecord(ctx sdk.Context, validatorAddr, domain string) (*types.DomainQualification, error) {
	stratum, err := k.getDomainStratum(ctx, domain)
	if err != nil {
		return nil, err
	}

	if _, found := k.GetQualification(ctx, validatorAddr, domain); found {
		return nil, types.ErrAlreadyQualified
	}

	validator, found := k.stakingKeeper.GetValidator(ctx, validatorAddr)
	if !found {
		return nil, types.ErrValidatorNotFound
	}

	params := k.GetParams(ctx)

	// Check validator's overall verification stats meet the threshold
	if validator.VerificationsCompleted < uint64(params.MinVerifications) {
		return nil, fmt.Errorf("%w: need %d verifications, have %d",
			types.ErrInsufficientTrackRecord, params.MinVerifications, validator.VerificationsCompleted)
	}

	// Check accuracy
	if validator.VerificationsCompleted > 0 {
		accuracyBps := uint32(validator.VerificationsCorrect * 10000 / validator.VerificationsCompleted)
		if accuracyBps < params.MinAccuracyBps {
			return nil, fmt.Errorf("%w: need %d bps accuracy, have %d bps",
				types.ErrInsufficientTrackRecord, params.MinAccuracyBps, accuracyBps)
		}
	}

	currentBlock := uint64(ctx.BlockHeight())

	q := types.DomainQualification{
		ValidatorAddr:          validatorAddr,
		Domain:                 domain,
		Stratum:                stratum,
		Pathway:                types.PathwayTrackRecord,
		Status:                 types.StatusActive, // track record = immediate active
		EarnedAtBlock:          currentBlock,
		ExpiresAtBlock:         currentBlock + params.InactivityExpiration,
		Metrics:                emptyMetrics(),
		InheritanceDiscountBps: 10000, // no discount
	}

	k.SetQualification(ctx, q)
	return &q, nil
}

// QualifyViaCrossReference qualifies a validator based on endorsements from qualified validators.
func (k Keeper) QualifyViaCrossReference(ctx sdk.Context, validatorAddr, domain string) (*types.DomainQualification, error) {
	stratum, err := k.getDomainStratum(ctx, domain)
	if err != nil {
		return nil, err
	}

	if _, found := k.GetQualification(ctx, validatorAddr, domain); found {
		return nil, types.ErrAlreadyQualified
	}

	if _, found := k.stakingKeeper.GetValidator(ctx, validatorAddr); !found {
		return nil, types.ErrValidatorNotFound
	}

	params := k.GetParams(ctx)

	// Count valid endorsements
	endorsements := k.GetEndorsementsForValidatorInDomain(ctx, validatorAddr, domain)

	// Filter: endorser must be qualified in the same domain
	var validEndorsers []string
	for _, e := range endorsements {
		if k.IsQualified(ctx, e.EndorserAddr, domain) {
			validEndorsers = append(validEndorsers, e.EndorserAddr)
		}
	}

	if uint32(len(validEndorsers)) < params.MinEndorsers {
		return nil, fmt.Errorf("%w: need %d qualified endorsers, have %d",
			types.ErrInsufficientEndorsements, params.MinEndorsers, len(validEndorsers))
	}

	currentBlock := uint64(ctx.BlockHeight())

	q := types.DomainQualification{
		ValidatorAddr:          validatorAddr,
		Domain:                 domain,
		Stratum:                stratum,
		Pathway:                types.PathwayCrossReference,
		Status:                 types.StatusProbationary,
		EarnedAtBlock:          currentBlock,
		ExpiresAtBlock:         currentBlock + params.InactivityExpiration,
		Metrics:                emptyMetrics(),
		Endorsers:              validEndorsers,
		InheritanceDiscountBps: 10000, // no discount
	}

	k.SetQualification(ctx, q)
	return &q, nil
}

// QualifyViaStratumInheritance qualifies a validator in a domain based on qualification
// in a lower (more verifiable) stratum domain.
func (k Keeper) QualifyViaStratumInheritance(ctx sdk.Context, validatorAddr, domain, inheritFrom string) (*types.DomainQualification, error) {
	stratum, err := k.getDomainStratum(ctx, domain)
	if err != nil {
		return nil, err
	}

	if _, found := k.GetQualification(ctx, validatorAddr, domain); found {
		return nil, types.ErrAlreadyQualified
	}

	if _, found := k.stakingKeeper.GetValidator(ctx, validatorAddr); !found {
		return nil, types.ErrValidatorNotFound
	}

	// Verify source qualification exists and is active
	sourceQ, found := k.GetQualification(ctx, validatorAddr, inheritFrom)
	if !found || (sourceQ.Status != types.StatusActive && sourceQ.Status != types.StatusProbationary) {
		return nil, fmt.Errorf("%w: not qualified in source domain %s", types.ErrNoInheritanceSource, inheritFrom)
	}

	// Verify source domain is in a lower (more verifiable) stratum
	if k.ontologyKeeper != nil {
		sourceDomain, sourceFound := k.ontologyKeeper.GetDomain(ctx, inheritFrom)
		if sourceFound && uint32(sourceDomain.Stratum) >= stratum {
			return nil, fmt.Errorf("%w: source domain stratum %d must be lower than target stratum %d",
				types.ErrNoInheritanceSource, sourceDomain.Stratum, stratum)
		}
	}

	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	q := types.DomainQualification{
		ValidatorAddr:          validatorAddr,
		Domain:                 domain,
		Stratum:                stratum,
		Pathway:                types.PathwayStratumInheritance,
		Status:                 types.StatusProbationary,
		EarnedAtBlock:          currentBlock,
		ExpiresAtBlock:         currentBlock + params.InactivityExpiration,
		Metrics:                emptyMetrics(),
		InheritedFrom:          inheritFrom,
		InheritanceDiscountBps: params.StratumDiscountBps,
	}

	k.SetQualification(ctx, q)
	return &q, nil
}

// --- Helpers ---

// getDomainStratum returns the stratum for a domain, or an error if the domain doesn't exist.
func (k Keeper) getDomainStratum(ctx sdk.Context, domain string) (uint32, error) {
	if k.ontologyKeeper == nil {
		return 0, nil // no ontology = allow all domains with stratum 0
	}
	d, found := k.ontologyKeeper.GetDomain(ctx, domain)
	if !found {
		return 0, types.ErrDomainNotFound
	}
	return uint32(d.Stratum), nil
}

func emptyMetrics() types.QualificationMetrics {
	return types.QualificationMetrics{
		EffectiveWeightBps: 10000, // 100% default
	}
}
