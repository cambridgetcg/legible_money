package types

// GenesisState defines the research module's genesis state.
type GenesisState struct {
	Params          Params     `json:"params"`
	Submissions     []Research `json:"submissions"`
	Bounties        []Bounty   `json:"bounties"`
	TreasuryBalance string     `json:"treasury_balance"`
}

// DefaultGenesis returns the default research module genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:          DefaultParams(),
		Submissions:     []Research{},
		Bounties:        []Bounty{},
		TreasuryBalance: "0",
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	return nil
}
