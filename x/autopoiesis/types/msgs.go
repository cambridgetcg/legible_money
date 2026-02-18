package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ============ MsgUpdateParams ============

// MsgUpdateParams is the governance message to update autopoiesis params.
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (msg *MsgUpdateParams) Reset()         {}
func (msg *MsgUpdateParams) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgUpdateParams) ProtoMessage()  {}

type MsgUpdateParamsResponse struct{}

func (msg *MsgUpdateParamsResponse) Reset()         {}
func (msg *MsgUpdateParamsResponse) String() string { return "MsgUpdateParamsResponse" }
func (msg *MsgUpdateParamsResponse) ProtoMessage()  {}

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
	return msg.Params.Validate()
}

// ============ MsgActivateAutopoiesis ============

// MsgActivateAutopoiesis enables or disables the autopoiesis engine via governance.
type MsgActivateAutopoiesis struct {
	Authority string `json:"authority"`
	Active    bool   `json:"active"`
}

func (msg *MsgActivateAutopoiesis) Reset()         {}
func (msg *MsgActivateAutopoiesis) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgActivateAutopoiesis) ProtoMessage()  {}

type MsgActivateAutopoiesisResponse struct{}

func (msg *MsgActivateAutopoiesisResponse) Reset()         {}
func (msg *MsgActivateAutopoiesisResponse) String() string { return "MsgActivateAutopoiesisResponse" }
func (msg *MsgActivateAutopoiesisResponse) ProtoMessage()  {}

func (msg *MsgActivateAutopoiesis) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgActivateAutopoiesis) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return nil
}

// ============ MsgOverrideMultiplier ============

// MsgOverrideMultiplier allows governance to manually set a specific multiplier value.
// Used for emergency corrections or initial calibration.
type MsgOverrideMultiplier struct {
	Authority string `json:"authority"`
	Path      string `json:"path"`  // Multiplier path (e.g., "slash.system")
	Value     uint64 `json:"value"` // New multiplier value in BPS (1M = 1.0x)
}

func (msg *MsgOverrideMultiplier) Reset()         {}
func (msg *MsgOverrideMultiplier) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgOverrideMultiplier) ProtoMessage()  {}

type MsgOverrideMultiplierResponse struct{}

func (msg *MsgOverrideMultiplierResponse) Reset()         {}
func (msg *MsgOverrideMultiplierResponse) String() string { return "MsgOverrideMultiplierResponse" }
func (msg *MsgOverrideMultiplierResponse) ProtoMessage()  {}

func (msg *MsgOverrideMultiplier) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgOverrideMultiplier) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if msg.Value == 0 {
		return fmt.Errorf("value must be positive")
	}
	return nil
}

// ============ MsgFreezeMultiplier ============

// MsgFreezeMultiplier freezes a specific multiplier at its current value,
// preventing the autopoiesis engine from modifying it until unfrozen.
type MsgFreezeMultiplier struct {
	Authority string `json:"authority"`
	Path      string `json:"path"`
	Frozen    bool   `json:"frozen"` // true = freeze, false = unfreeze
}

func (msg *MsgFreezeMultiplier) Reset()         {}
func (msg *MsgFreezeMultiplier) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgFreezeMultiplier) ProtoMessage()  {}

type MsgFreezeMultiplierResponse struct{}

func (msg *MsgFreezeMultiplierResponse) Reset()         {}
func (msg *MsgFreezeMultiplierResponse) String() string { return "MsgFreezeMultiplierResponse" }
func (msg *MsgFreezeMultiplierResponse) ProtoMessage()  {}

func (msg *MsgFreezeMultiplier) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgFreezeMultiplier) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	return nil
}
