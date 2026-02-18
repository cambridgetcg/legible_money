package types

import (
	"fmt"
)

// --- Params Validation ---
// Methods below attach to protobuf-generated types from state.pb.go and genesis.pb.go.

// DefaultParams returns the default rate limit parameters.
func DefaultParams() Params {
	return Params{
		EpochLengthBlocks:  34272,
		MaxOutflowPerEpoch: "10000000000", // 10,000 LGM
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	}
}

// Validate performs basic validation of params.
func (p *Params) Validate() error {
	if p.EpochLengthBlocks == 0 {
		return fmt.Errorf("epoch length blocks must be positive")
	}
	if p.TrackedDenom == "" {
		return fmt.Errorf("tracked denom cannot be empty")
	}
	return nil
}

// --- Genesis state validation ---

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() GenesisState {
	p := DefaultParams()
	return GenesisState{
		Params: &p,
		Flows:  []*ChannelFlow{},
	}
}

// Validate performs genesis state validation.
func (gs *GenesisState) Validate() error {
	if gs.Params != nil {
		return gs.Params.Validate()
	}
	return nil
}

// --- Message Validation (extends proto-generated types in tx.pb.go) ---

func (m *MsgUpdateParams) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if m.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return m.Params.Validate()
}
