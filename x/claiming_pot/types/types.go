package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Enums ---

// PotStatus represents the lifecycle state of a claiming pot.
type PotStatus string

const (
	PotStatusActive    PotStatus = "active"
	PotStatusExhausted PotStatus = "exhausted"
	PotStatusExpired   PotStatus = "expired"
	PotStatusClosed    PotStatus = "closed"
)

// --- Params Validation ---

func (p *Params) Validate() error {
	if p.MaxPotsPerCreator == 0 {
		return fmt.Errorf("max pots per creator must be positive")
	}
	if p.MaxAllowlistSize == 0 {
		return fmt.Errorf("max allowlist size must be positive")
	}
	return nil
}

// --- ValidateBasic methods for proto message types ---

func (m *MsgCreatePot) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.InitialFunding == "" || m.InitialFunding == "0" {
		return fmt.Errorf("initial funding must be positive")
	}
	return nil
}

func (m *MsgFundPot) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Funder); err != nil {
		return fmt.Errorf("invalid funder address: %w", err)
	}
	if m.PotId == "" {
		return fmt.Errorf("pot id cannot be empty")
	}
	if m.Amount == "" || m.Amount == "0" {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (m *MsgClaimFromPot) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Claimer); err != nil {
		return fmt.Errorf("invalid claimer address: %w", err)
	}
	if m.PotId == "" {
		return fmt.Errorf("pot id cannot be empty")
	}
	return nil
}

func (m *MsgClosePot) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if m.PotId == "" {
		return fmt.Errorf("pot id cannot be empty")
	}
	return nil
}


// MsgUpdateParams is the governance message to update module params.
type MsgUpdateParams struct {
	Authority string  `json:"authority"`
	Params    *Params `json:"params"`
}

// MsgUpdateParamsResponse is the response type for MsgUpdateParams.
type MsgUpdateParamsResponse struct{}

func (*MsgUpdateParams) ProtoMessage()             {}
func (*MsgUpdateParams) Reset()                     {}
func (*MsgUpdateParams) String() string             { return "MsgUpdateParams" }
func (*MsgUpdateParamsResponse) ProtoMessage()      {}
func (*MsgUpdateParamsResponse) Reset()             {}
func (*MsgUpdateParamsResponse) String() string     { return "MsgUpdateParamsResponse" }

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
