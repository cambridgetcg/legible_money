package types

import "fmt"

// DefaultParams returns default claiming_pot module parameters.
func DefaultParams() *Params {
	return &Params{
		MaxPotsPerCreator: 100,
		MinInitialDeposit: "1000000", // 1 LGM in ulgm
		MaxAllowlistSize:  10000,
		MinAmountPerClaim: "1000", // 0.001 LGM in ulgm
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Pots:   []*ClaimingPot{},
		Claims: []*ClaimRecord{},
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}
