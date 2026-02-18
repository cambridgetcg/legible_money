package types

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		MaxRemoteAccountsPerDid: 10,
		AllowedHostMsgTypes: []string{
			"/cosmos.bank.v1beta1.MsgSend",
			"/cosmos.staking.v1beta1.MsgDelegate",
			"/cosmos.staking.v1beta1.MsgUndelegate",
			"/cosmos.gov.v1.MsgVote",
			// MsgTransfer intentionally excluded: ICA-initiated IBC transfers
			// bypass rate limiting middleware, creating a bridge drain vector (P0-6).
		},
		RegistrationCooldown: 100, // ~4 minutes at 2521ms blocks
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:  &p,
		Records: []*InterchainAccountRecord{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Params == nil {
		return nil
	}
	return gs.Params.Validate()
}
