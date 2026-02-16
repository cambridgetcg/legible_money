package types

// DefaultParams returns default module parameters.
// Expiration fields are zeroed — the period overlay has been removed.
func DefaultParams() Params {
	return Params{}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:   &p,
		Accounts: []*ExpirableAccount{},
		Rai: &RAIHoldingAccount{
			Balance:          "0",
			TotalReceived:    "0",
			TotalDistributed: "0",
			LastDistBlock:    0,
		},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	return nil
}
