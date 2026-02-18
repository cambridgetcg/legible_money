package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- String-based channel status constants (used by keeper for state storage) ---
// These complement the protobuf ChannelStatus enum with string representations.
// Note: The ChannelStatus type name is now reserved by the protobuf-generated enum (int32).

const (
	ChannelStatusOpen     = "open"
	ChannelStatusClosing  = "closing"  // unilateral close challenge period
	ChannelStatusDisputed = "disputed" // dispute challenge period
	ChannelStatusClosed   = "closed"   // terminal state
)

// --- Params Validation ---

func (p *Params) Validate() error {
	if p.MaxChannelDuration == 0 {
		return fmt.Errorf("max channel duration must be positive")
	}
	if p.DisputePeriodBlocks == 0 {
		return fmt.Errorf("dispute period must be positive")
	}
	if p.MaxChannelsPerPayer == 0 {
		return fmt.Errorf("max channels per payer must be positive")
	}
	return nil
}

// --- ValidateBasic methods for protobuf message types ---

func (m *MsgOpenChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Receiver); err != nil {
		return fmt.Errorf("invalid receiver address: %w", err)
	}
	if m.Sender == m.Receiver {
		return fmt.Errorf("sender and receiver cannot be the same address")
	}
	if m.Deposit == "" || m.Deposit == "0" {
		return fmt.Errorf("deposit must be positive")
	}
	if m.TimeoutHeight == 0 {
		return fmt.Errorf("timeout height must be positive")
	}
	return nil
}

func (m *MsgCloseChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Closer); err != nil {
		return fmt.Errorf("invalid closer address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

func (m *MsgSettleChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

func (m *MsgDisputeChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Disputer); err != nil {
		return fmt.Errorf("invalid disputer address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

func (m *MsgUpdateChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

// --- ValidateBasic / GetSigners methods for proto-generated message types ---
// Struct definitions are in tx.pb.go (proto-generated). Only behavioral methods here.

func (m *MsgDepositChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Depositor); err != nil {
		return fmt.Errorf("invalid depositor address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	if m.Amount == "" || m.Amount == "0" {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (m *MsgUpdateState) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

func (m *MsgClaimExpired) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Claimer); err != nil {
		return fmt.Errorf("invalid claimer address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
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
