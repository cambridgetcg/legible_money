package types

// Note: GenesisState struct is now defined in genesis.pb.go (proto-generated).

// DefaultGenesis returns the default research module genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:          &p,
		Researches:      []*Research{},
		Bounties:        []*Bounty{},
		PeerReviews:     []*PeerReview{},
		TreasuryBalance: &TreasuryBalance{Balance: "0"},
	}
}

// Validate performs basic genesis state validation.
func (gs *GenesisState) Validate() error {
	return nil
}
