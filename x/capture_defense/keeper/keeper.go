package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_defense/types"
)

// Keeper manages the capture defense module's state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey

	stakingKeeper  types.StakingKeeper
	ontologyKeeper types.OntologyKeeper
	bankKeeper     types.BankKeeper

	authority string
}

// NewKeeper creates a new capture defense Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	sk types.StakingKeeper,
	ok types.OntologyKeeper,
	bk types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:            cdc,
		storeKey:       storeKey,
		stakingKeeper:  sk,
		ontologyKeeper: ok,
		bankKeeper:     bk,
		authority:      authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// InitGenesis initializes the module from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	for _, qual := range genState.Qualifications {
		k.SetQualification(ctx, qual)
	}
	for _, rep := range genState.Reputations {
		k.SetIsolatedReputation(ctx, rep)
	}
	for _, m := range genState.Metrics {
		k.SetCaptureMetrics(ctx, m)
	}
}

// ExportGenesis exports the module's genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	var quals []types.DomainQualification
	k.IterateQualifications(ctx, func(q types.DomainQualification) bool {
		quals = append(quals, q)
		return false
	})

	var reps []types.IsolatedReputation
	k.IterateReputations(ctx, func(r types.IsolatedReputation) bool {
		reps = append(reps, r)
		return false
	})

	return types.GenesisState{
		Params:         k.GetParams(ctx),
		Qualifications: quals,
		Reputations:    reps,
	}
}

// --- Qualification Processing ---

// ProcessQualificationRequest evaluates a qualification request through
// one of four pathways: stake_commitment, track_record, cross_reference,
// or stratum_inheritance.
func (k Keeper) ProcessQualificationRequest(
	ctx sdk.Context,
	validatorAddr, domain string,
	pathway types.QualificationPathway,
	stake string,
	endorsers []string,
	inheritFrom string,
) (types.DomainQualification, error) {
	currentBlock := uint64(ctx.BlockHeight())

	// Check domain exists via ontology keeper
	if k.ontologyKeeper != nil && !k.ontologyKeeper.DomainExists(ctx, domain) {
		return types.DomainQualification{}, types.ErrDomainNotFound
	}

	// Check not already qualified
	if existing, found := k.GetQualification(ctx, validatorAddr, domain); found {
		if existing.Status == types.StatusActive || existing.Status == types.StatusProbationary {
			return types.DomainQualification{}, types.ErrAlreadyQualified
		}
	}

	var qual types.DomainQualification
	var err error

	switch pathway {
	case types.PathwayStakeCommitment:
		qual, err = k.qualifyByStake(ctx, validatorAddr, domain, stake, currentBlock)
	case types.PathwayTrackRecord:
		qual, err = k.qualifyByTrackRecord(ctx, validatorAddr, domain, currentBlock)
	case types.PathwayCrossReference:
		qual, err = k.qualifyByCrossReference(ctx, validatorAddr, domain, endorsers, currentBlock)
	case types.PathwayStratumInheritance:
		qual, err = k.qualifyByInheritance(ctx, validatorAddr, domain, inheritFrom, currentBlock)
	default:
		return types.DomainQualification{}, types.ErrInvalidPathway
	}

	if err != nil {
		return types.DomainQualification{}, err
	}

	k.SetQualification(ctx, qual)
	return qual, nil
}

// qualifyByStake creates a qualification via locked stake commitment.
func (k Keeper) qualifyByStake(
	ctx sdk.Context,
	validatorAddr, domain, stake string,
	currentBlock uint64,
) (types.DomainQualification, error) {
	requiredStake := k.getRequiredStakeForDomain(ctx, domain)

	stakeInt := new(big.Int)
	stakeInt.SetString(stake, 10)
	reqInt := new(big.Int)
	reqInt.SetString(requiredStake, 10)

	if stakeInt.Cmp(reqInt) < 0 {
		return types.DomainQualification{}, types.ErrInsufficientStake
	}

	params := k.GetParams(ctx)
	stratum := k.getDomainStratum(ctx, domain)

	return types.DomainQualification{
		ValidatorAddr:       validatorAddr,
		Domain:              domain,
		Stratum:             stratum,
		Pathway:             types.PathwayStakeCommitment,
		Status:              types.StatusProbationary,
		EarnedAtBlock:       currentBlock,
		ExpiresAtBlock:      0, // No expiry for stake commitment
		Metrics:             types.QualificationMetrics{EffectiveWeight: types.ProbationaryWeight},
		LockedStake:         stake,
		StakeUnlocksAtBlock: currentBlock + params.StakeLockPeriod,
		InheritanceDiscount: 1_000_000, // 100% = no discount
	}, nil
}

// qualifyByTrackRecord creates a qualification based on verification history.
func (k Keeper) qualifyByTrackRecord(
	ctx sdk.Context,
	validatorAddr, domain string,
	currentBlock uint64,
) (types.DomainQualification, error) {
	params := k.GetParams(ctx)
	rep := k.GetOrCreateReputation(ctx, validatorAddr, currentBlock)

	domainRep, hasDomainRep := rep.DomainReputations[domain]
	if !hasDomainRep || domainRep.TotalVerifications < params.MinVerificationsForQual {
		return types.DomainQualification{}, types.ErrInsufficientTrackRecord
	}

	if domainRep.Accuracy < params.MinAccuracyForQual {
		return types.DomainQualification{}, types.ErrInsufficientTrackRecord
	}

	stratum := k.getDomainStratum(ctx, domain)

	return types.DomainQualification{
		ValidatorAddr:  validatorAddr,
		Domain:         domain,
		Stratum:        stratum,
		Pathway:        types.PathwayTrackRecord,
		Status:         types.StatusActive,
		EarnedAtBlock:  currentBlock,
		ExpiresAtBlock: currentBlock + params.InactivityExpirationBlocks,
		Metrics: types.QualificationMetrics{
			TotalVerifications:   domainRep.TotalVerifications,
			CorrectVerifications: domainRep.CorrectVerifications,
			Accuracy:             domainRep.Accuracy,
			EffectiveWeight:      1_000_000, // 100%
		},
		InheritanceDiscount: 1_000_000, // 100% = no discount
	}, nil
}

// qualifyByCrossReference creates a qualification via endorsements.
func (k Keeper) qualifyByCrossReference(
	ctx sdk.Context,
	validatorAddr, domain string,
	endorsers []string,
	currentBlock uint64,
) (types.DomainQualification, error) {
	params := k.GetParams(ctx)

	if uint64(len(endorsers)) < params.MinEndorsersForQual {
		return types.DomainQualification{}, types.ErrInsufficientEndorsers
	}

	// Verify each endorser is qualified for the domain
	validEndorsers := []string{}
	for _, endorser := range endorsers {
		if endorser == validatorAddr {
			continue // Skip self-endorsement
		}
		eQual, found := k.GetQualification(ctx, endorser, domain)
		if !found || (eQual.Status != types.StatusActive && eQual.Status != types.StatusProbationary) {
			continue
		}
		validEndorsers = append(validEndorsers, endorser)
	}

	if uint64(len(validEndorsers)) < params.MinEndorsersForQual {
		return types.DomainQualification{}, types.ErrInsufficientEndorsers
	}

	stratum := k.getDomainStratum(ctx, domain)

	return types.DomainQualification{
		ValidatorAddr:  validatorAddr,
		Domain:         domain,
		Stratum:        stratum,
		Pathway:        types.PathwayCrossReference,
		Status:         types.StatusProbationary,
		EarnedAtBlock:  currentBlock,
		ExpiresAtBlock: currentBlock + types.EndorsementDecayBlocks,
		Metrics:        types.QualificationMetrics{EffectiveWeight: types.ProbationaryWeight},
		Endorsers:      validEndorsers,
		InheritanceDiscount: 1_000_000, // 100% = no discount
	}, nil
}

// qualifyByInheritance creates a qualification from a lower stratum qualification.
func (k Keeper) qualifyByInheritance(
	ctx sdk.Context,
	validatorAddr, domain, inheritFrom string,
	currentBlock uint64,
) (types.DomainQualification, error) {
	if inheritFrom == "" {
		// Try to find inheritable qualification automatically
		inheritFrom = k.findInheritableQualification(ctx, validatorAddr, domain)
		if inheritFrom == "" {
			return types.DomainQualification{}, types.ErrNoInheritableQualification
		}
	}

	parentQual, found := k.GetQualification(ctx, validatorAddr, inheritFrom)
	if !found || parentQual.Status != types.StatusActive {
		return types.DomainQualification{}, types.ErrNoInheritableQualification
	}

	stratum := k.getDomainStratum(ctx, domain)

	return types.DomainQualification{
		ValidatorAddr:       validatorAddr,
		Domain:              domain,
		Stratum:             stratum,
		Pathway:             types.PathwayStratumInheritance,
		Status:              types.StatusProbationary,
		EarnedAtBlock:       currentBlock,
		ExpiresAtBlock:      currentBlock + types.InactivityExpirationBlocks,
		Metrics:             types.QualificationMetrics{EffectiveWeight: types.StratumInheritanceDiscount},
		InheritedFrom:       inheritFrom,
		InheritanceDiscount: types.StratumInheritanceDiscount,
	}, nil
}

// findInheritableQualification looks for an active qualification in a related domain.
func (k Keeper) findInheritableQualification(ctx sdk.Context, validatorAddr, targetDomain string) string {
	if k.ontologyKeeper == nil {
		return ""
	}
	parent, found := k.ontologyKeeper.GetParentDomain(ctx, targetDomain)
	if !found {
		return ""
	}
	if qual, ok := k.GetQualification(ctx, validatorAddr, parent); ok && qual.Status == types.StatusActive {
		return parent
	}
	return ""
}

// --- Helper Methods ---

// getRequiredStakeForDomain returns the required stake based on domain depth.
func (k Keeper) getRequiredStakeForDomain(ctx sdk.Context, domain string) string {
	if k.ontologyKeeper == nil {
		return types.StakeLeafDomain
	}
	depth, found := k.ontologyKeeper.GetDomainDepth(ctx, domain)
	if !found {
		return types.StakeLeafDomain
	}
	if depth <= 1 {
		return types.StakeTopDomain
	}
	if depth == 2 {
		return types.StakeMidDomain
	}
	return types.StakeLeafDomain
}

// getDomainStratum returns the stratum for a domain.
func (k Keeper) getDomainStratum(ctx sdk.Context, domain string) uint32 {
	if k.ontologyKeeper == nil {
		return 0
	}
	stratum, found := k.ontologyKeeper.GetDomainStratum(ctx, domain)
	if !found {
		return 0
	}
	return stratum
}

// IsQualifiedForDomain checks if a validator is qualified for a domain.
func (k Keeper) IsQualifiedForDomain(ctx sdk.Context, validatorAddr, domain string, currentBlock uint64) bool {
	qual, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		return false
	}
	if qual.Status != types.StatusActive && qual.Status != types.StatusProbationary {
		return false
	}
	if qual.ExpiresAtBlock > 0 && currentBlock > qual.ExpiresAtBlock {
		return false
	}
	return true
}

// GetQualificationWeight returns the effective weight of a validator's qualification.
func (k Keeper) GetQualificationWeight(ctx sdk.Context, validatorAddr, domain string) uint64 {
	qual, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		return 0
	}
	if qual.Status != types.StatusActive && qual.Status != types.StatusProbationary {
		return 0
	}
	return qual.Metrics.EffectiveWeight
}

// UpdateQualificationMetrics updates a qualification's metrics after a verification.
func (k Keeper) UpdateQualificationMetrics(
	ctx sdk.Context,
	validatorAddr, domain string,
	wasCorrect bool,
	currentBlock uint64,
) {
	qual, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		return
	}

	qual.Metrics.TotalVerifications++
	qual.Metrics.LastVerificationBlock = currentBlock

	if wasCorrect {
		qual.Metrics.CorrectVerifications++
		qual.Metrics.ConsecutiveCorrect++
		qual.Metrics.ConsecutiveIncorrect = 0
	} else {
		qual.Metrics.ConsecutiveIncorrect++
		qual.Metrics.ConsecutiveCorrect = 0
	}

	if qual.Metrics.TotalVerifications > 0 {
		qual.Metrics.Accuracy = (qual.Metrics.CorrectVerifications * 1_000_000) / qual.Metrics.TotalVerifications
	}

	// Transition from probationary to active after sufficient verifications
	if qual.Status == types.StatusProbationary {
		params := k.GetParams(ctx)
		if qual.Metrics.TotalVerifications >= params.MinVerificationsForQual &&
			qual.Metrics.Accuracy >= params.MinAccuracyForQual {
			qual.Status = types.StatusActive
			qual.Metrics.EffectiveWeight = qual.InheritanceDiscount // Use full weight (or inheritance discount)
		}
	}

	// Check for suspension
	if qual.Metrics.ConsecutiveIncorrect >= types.ConsecutiveIncorrectSuspend {
		qual.Status = types.StatusSuspended
		qual.Metrics.EffectiveWeight = 0
	}
	if qual.Metrics.Accuracy < types.AccuracyThresholdSuspension && qual.Metrics.TotalVerifications >= types.MinVerificationsForQual {
		qual.Status = types.StatusSuspended
		qual.Metrics.EffectiveWeight = 0
	}

	// Renew expiration on activity
	if qual.ExpiresAtBlock > 0 {
		params := k.GetParams(ctx)
		qual.ExpiresAtBlock = currentBlock + params.InactivityExpirationBlocks
	}

	k.SetQualification(ctx, qual)
}

// UpdateAfterVerification updates both qualification and reputation after a verification round.
func (k Keeper) UpdateAfterVerification(
	ctx sdk.Context,
	validatorAddr, domain string,
	stratum uint32,
	wasCorrect bool,
	verdictConf, finalConf uint64,
	currentBlock uint64,
) error {
	// Update qualification metrics
	k.UpdateQualificationMetrics(ctx, validatorAddr, domain, wasCorrect, currentBlock)

	// Update reputation
	k.UpdateDomainReputation(ctx, validatorAddr, domain, stratum, wasCorrect, verdictConf, finalConf, currentBlock)

	return nil
}

// RecordVerificationHistory records a verification round entry for capture analysis.
func (k Keeper) RecordVerificationHistory(ctx sdk.Context, domain string, entry types.VerificationHistoryEntry) error {
	k.AppendVerificationHistoryEntry(ctx, domain, entry)
	return nil
}
