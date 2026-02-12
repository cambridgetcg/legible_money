package types

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		MaxActivePerAccount:    100,
		MaxGasPerBlock:         2000000,
		MinIntervalBlocks:      1,
		MaxFutureBlocks:        12500000, // ~1 year at 2521ms blocks
		MinPrepaidExecutions:   10,
		ExpiredRetentionBlocks: 1000,
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Processes: []ScheduleProcess{},
		Sequence:  0,
	}
}

// Validate performs genesis state validation.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
