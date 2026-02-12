package types

import "fmt"

// DefaultParams returns the default bvm module parameters.
func DefaultParams() Params {
	return Params{
		MaxBytecodeSize:         1_048_576, // 1 MB
		MaxStateEntries:         100_000,
		MaxSchedulesPerContract: 100,
		DeployGasCost:           200_000,
		CallBaseGasCost:         21_000,
		StorageWriteGasCost:     5_000,
		StorageReadGasCost:      2_000,
	}
}

// DefaultGenesisState returns the default genesis state for the bvm module.
func DefaultGenesisState() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:    &p,
		Contracts: []*DeployedContract{},
		Code:      []*ContractCode{},
		Schedules: []*ContractSchedule{},
		State:     []*ContractStateEntry{},
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	for i, c := range gs.Contracts {
		if c.Address == "" {
			return fmt.Errorf("contract %d: empty address", i)
		}
		if c.CodeHash == "" {
			return fmt.Errorf("contract %d: empty code hash", i)
		}
		if c.Creator == "" {
			return fmt.Errorf("contract %d: empty creator", i)
		}
	}
	for i, code := range gs.Code {
		if code.CodeHash == "" {
			return fmt.Errorf("code %d: empty code hash", i)
		}
	}
	return nil
}
