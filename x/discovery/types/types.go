package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Params Validation ---
// Methods below attach to protobuf-generated types from state.pb.go and genesis.pb.go.

// Validate performs parameter validation.
func (p *Params) Validate() error {
	if p.MaxCapabilitiesPerAgent == 0 {
		return fmt.Errorf("max_capabilities_per_agent must be positive")
	}
	if p.ProfileExpiryBlocks == 0 {
		return fmt.Errorf("profile_expiry_blocks must be positive")
	}
	return nil
}

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		MinRegistrationStake:    "100000000", // 100 LGM
		MaxCapabilitiesPerAgent: 50,
		ProfileExpiryBlocks:     342720, // ~10 days at 2521ms blocks
	}
}

// --- Genesis state validation ---

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:   &p,
		Profiles: []*AgentProfile{},
	}
}

// Validate performs genesis state validation.
func (gs *GenesisState) Validate() error {
	if gs.Params != nil {
		return gs.Params.Validate()
	}
	return nil
}

// -----------------------------------------------------------------------
// GetSigners + ValidateBasic for proto-generated Msg types
// -----------------------------------------------------------------------

func (msg *MsgRegisterAgent) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterAgent) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	if msg.DisplayName == "" {
		return fmt.Errorf("display name cannot be empty")
	}
	return nil
}

func (msg *MsgUpdateProfile) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateProfile) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	return nil
}

func (msg *MsgAddCapability) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAddCapability) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	if msg.Capability == nil || msg.Capability.CapabilityType == "" {
		return fmt.Errorf("capability type cannot be empty")
	}
	return nil
}

func (msg *MsgRemoveCapability) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRemoveCapability) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	if msg.CapabilityType == "" {
		return fmt.Errorf("capability type cannot be empty")
	}
	return nil
}

func (msg *MsgDeregisterAgent) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgDeregisterAgent) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	return nil
}

func (msg *MsgHeartbeat) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

func (msg *MsgHeartbeat) ValidateBasic() error {
	if msg.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	return nil
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return msg.Params.Validate()
}

// Query types are now proto-generated in query.pb.go
// QueryServer and RegisterQueryServer are in query_grpc.pb.go
