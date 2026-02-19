package types

// DefaultParams returns default tree module parameters.
func DefaultParams() *Params {
	return &Params{
		MinBudget:              "1000000", // 1 LGM
		MaxTasksPerProject:     200,
		MaxContributors:        50,
		MaxApplications:        100,
		TaskDeadlineMinBlocks:  100,
		TaskDeadlineMaxBlocks:  1036800, // ~30 days
		MaxRejections:          3,
		SeedExpiryBlocks:       172800, // ~5 days
		MinContributorsToStart: 1,
		EvidenceTaxBp:          220000,
		ContributorsBp:         390000,
		// TODO: Tree service revenue research split (23.4%) is independent from
		// the 3.33% block reward research tax. These are different economic
		// contexts (service payments vs minting). Evaluate unification in v0.3.
		ResearchFundBp:         234000,
		ProtocolTreasuryBp:     156000,
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		Projects: []*ProductProject{},
		Tasks:    []*ProjectTask{},
		Services: []*ServiceLeaf{},
		Seeds:    []*OpportunitySeed{},
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return nil
	}
	return gs.Params.Validate()
}
