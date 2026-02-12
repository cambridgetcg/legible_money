package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_defense/types"
)

// Keeper manages the capture defense module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	stakingKeeper   types.StakingKeeper
	ontologyKeeper  types.OntologyKeeper
	bankKeeper      types.BankKeeper
	emergencyKeeper types.EmergencyKeeper

	authority string
}

// NewKeeper creates a new capture defense Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	sk types.StakingKeeper,
	ok types.OntologyKeeper,
	bk types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:            cdc,
		storeService:   storeService,
		stakingKeeper:  sk,
		ontologyKeeper: ok,
		bankKeeper:     bk,
		authority:      authority,
	}
}

// SetEmergencyKeeper sets the emergency keeper (set after keeper creation to avoid circular deps).
func (k *Keeper) SetEmergencyKeeper(ek types.EmergencyKeeper) {
	k.emergencyKeeper = ek
}

// prefixEndBytes returns the end key for a prefix scan (exclusive upper bound).
// It increments the last byte of the prefix; if it overflows, it tries the
// previous byte, and so on. Returns nil if all bytes overflow (prefix is all 0xff).
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

// InitGenesis initializes the module from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, *genState.Params)
	}
	for _, qual := range genState.Qualifications {
		if qual == nil {
			continue
		}
		k.SetQualification(ctx, *qual)
	}
	for _, rep := range genState.Reputations {
		if rep == nil {
			continue
		}
		k.SetIsolatedReputation(ctx, *rep)
	}
	for _, m := range genState.Metrics {
		if m == nil {
			continue
		}
		k.SetCaptureMetrics(ctx, *m)
	}
}

// ExportGenesis exports the module's genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	var qualPtrs []*types.DomainQualification
	k.IterateQualifications(ctx, func(q types.DomainQualification) bool {
		qCopy := q
		qualPtrs = append(qualPtrs, &qCopy)
		return false
	})

	var repPtrs []*types.IsolatedReputation
	k.IterateReputations(ctx, func(r types.IsolatedReputation) bool {
		rCopy := r
		repPtrs = append(repPtrs, &rCopy)
		return false
	})

	var metricPtrs []*types.CaptureMetrics
	k.IterateCaptureMetrics(ctx, func(m types.CaptureMetrics) bool {
		mCopy := m
		metricPtrs = append(metricPtrs, &mCopy)
		return false
	})

	params := k.GetParams(ctx)
	return types.GenesisState{
		Params:         &params,
		Qualifications: qualPtrs,
		Reputations:    repPtrs,
		Metrics:        metricPtrs,
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
		if existing.Status == string(types.StatusActive) || existing.Status == string(types.StatusProbationary) {
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
		Pathway:             string(types.PathwayStakeCommitment),
		Status:              string(types.StatusProbationary),
		EarnedAtBlock:       currentBlock,
		ExpiresAtBlock:      0, // No expiry for stake commitment
		Metrics:             &types.QualificationMetrics{EffectiveWeight: types.ProbationaryWeight},
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

	domainRep := findDomainReputation(rep.DomainReputations, domain)
	if domainRep == nil || domainRep.TotalVerifications < params.MinVerificationsForQual {
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
		Pathway:        string(types.PathwayTrackRecord),
		Status:         string(types.StatusActive),
		EarnedAtBlock:  currentBlock,
		ExpiresAtBlock: currentBlock + params.InactivityExpirationBlocks,
		Metrics: &types.QualificationMetrics{
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
		if !found || (eQual.Status != string(types.StatusActive) && eQual.Status != string(types.StatusProbationary)) {
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
		Pathway:        string(types.PathwayCrossReference),
		Status:         string(types.StatusProbationary),
		EarnedAtBlock:  currentBlock,
		ExpiresAtBlock: currentBlock + types.EndorsementDecayBlocks,
		Metrics:        &types.QualificationMetrics{EffectiveWeight: types.ProbationaryWeight},
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
	if !found || parentQual.Status != string(types.StatusActive) {
		return types.DomainQualification{}, types.ErrNoInheritableQualification
	}

	stratum := k.getDomainStratum(ctx, domain)

	return types.DomainQualification{
		ValidatorAddr:       validatorAddr,
		Domain:              domain,
		Stratum:             stratum,
		Pathway:             string(types.PathwayStratumInheritance),
		Status:              string(types.StatusProbationary),
		EarnedAtBlock:       currentBlock,
		ExpiresAtBlock:      currentBlock + types.InactivityExpirationBlocks,
		Metrics:             &types.QualificationMetrics{EffectiveWeight: types.StratumInheritanceDiscount},
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
	if qual, ok := k.GetQualification(ctx, validatorAddr, parent); ok && qual.Status == string(types.StatusActive) {
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
	if qual.Status != string(types.StatusActive) && qual.Status != string(types.StatusProbationary) {
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
	if qual.Status != string(types.StatusActive) && qual.Status != string(types.StatusProbationary) {
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
	if qual.Status == string(types.StatusProbationary) {
		params := k.GetParams(ctx)
		if qual.Metrics.TotalVerifications >= params.MinVerificationsForQual &&
			qual.Metrics.Accuracy >= params.MinAccuracyForQual {
			qual.Status = string(types.StatusActive)
			qual.Metrics.EffectiveWeight = qual.InheritanceDiscount // Use full weight (or inheritance discount)
		}
	}

	// Check for suspension
	if qual.Metrics.ConsecutiveIncorrect >= types.ConsecutiveIncorrectSuspend {
		qual.Status = string(types.StatusSuspended)
		qual.Metrics.EffectiveWeight = 0
	}
	if qual.Metrics.Accuracy < types.AccuracyThresholdSuspension && qual.Metrics.TotalVerifications >= types.MinVerificationsForQual {
		qual.Status = string(types.StatusSuspended)
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

// findDomainReputation looks up a DomainReputation by domain key in a proto entry slice.
func findDomainReputation(entries []*types.DomainReputationEntry, domain string) *types.DomainReputation {
	for _, e := range entries {
		if e != nil && e.Key == domain {
			return e.Value
		}
	}
	return nil
}

// findStratumReputation looks up a StratumReputation by stratum key in a proto entry slice.
func findStratumReputation(entries []*types.StratumReputationEntry, stratum uint32) *types.StratumReputation {
	for _, e := range entries {
		if e != nil && e.Key == stratum {
			return e.Value
		}
	}
	return nil
}
