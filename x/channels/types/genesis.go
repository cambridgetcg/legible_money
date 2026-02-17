package types

import "fmt"

// DefaultParams returns default channels module parameters.
func DefaultParams() *Params {
	return &Params{
		MinDeposit:            "1000000", // 1 LGM in ulgm
		MaxChannelDuration:    1036800,   // ~30 days at 2.521s blocks
		DefaultSettlementFreq: 100,
		DisputePeriodBlocks:   1000,
		ExpiryGracePeriod:     500,
		FraudPenaltyBps:       500, // 5%
		MaxChannelsPerPayer:   50,
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		Channels: []*PaymentChannel{},
		Disputes: []*ChannelDispute{},
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}
