package types

import "fmt"

// DefaultParams returns the default capture defense parameters.
func DefaultParams() *Params {
	return &Params{
		StakeLockPeriod:            StakeLockPeriod,
		MinVerificationsForQual:    MinVerificationsForQual,
		MinAccuracyForQual:         MinAccuracyForQual,
		MinEndorsersForQual:        MinEndorsersForQual,
		InactivityExpirationBlocks: InactivityExpirationBlocks,
		ProbationaryPeriodBlocks:   ProbationaryPeriodBlocks,
		ReputationHalfLifeBlocks:   ReputationHalfLifeBlocks,
		DetectionIntervalBlocks:    DetectionIntervalBlocks,
		DecayIntervalBlocks:        DecayIntervalBlocks,
		HerfindahlTriggerThreshold: HerfindahlTriggerThreshold,
		MaxExpirationsPerBlock:     MaxExpirationsPerBlock,
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		Qualifications: []*DomainQualification{},
		Reputations:    []*IsolatedReputation{},
		Metrics:        []*CaptureMetrics{},
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}
