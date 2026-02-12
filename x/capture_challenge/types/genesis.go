package types

// DefaultParams returns the default capture challenge parameters.
func DefaultParams() *Params {
	return &Params{
		EvidencePeriodBlocks:      DefaultEvidencePeriodBlocks,
		ReviewPeriodBlocks:        DefaultReviewPeriodBlocks,
		DomainPauseDurationBlocks: DefaultDomainPauseDurationBlocks,
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		Challenges:  []*CaptureChallenge{},
		BountyPools: []*DomainBountyPool{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Params != nil {
		return gs.Params.Validate()
	}
	return nil
}
