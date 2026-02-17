package types

// DefaultParams returns default home module parameters.
func DefaultParams() Params {
	return Params{
		MaxHomesPerAccount:   3,
		MinCreationStake:     "1000000", // 1 LGM in ulgm
		SessionMaxDuration:   34272,     // 1 day in blocks
		MaxKeysPerHome:       20,
		MaxAlertsPerHome:     100,
		MaxSessionsPerHome:   5,
		DeadmanMinThreshold:  34272,  // 1 day in blocks
		AlertRetentionBlocks: 342720, // 10 days in blocks
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	params := DefaultParams()
	return &GenesisState{
		Params: &params,
		Homes:  []*AgentHome{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Params != nil {
		return gs.Params.Validate()
	}
	return nil
}
