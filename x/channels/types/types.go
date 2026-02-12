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
	ChannelStatusClosing  = "closing"
	ChannelStatusDisputed = "disputed"
	ChannelStatusSettled  = "settled"
	ChannelStatusClosed   = "closed"
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

// --- Extra message types not in protobuf definitions ---

// MsgDepositChannel adds funds to an existing channel.
type MsgDepositChannel struct {
	Depositor string `json:"depositor"` // must be payer
	ChannelId string `json:"channel_id"`
	Amount    string `json:"amount"` // ulgm
}

func (m *MsgDepositChannel) Reset()         { *m = MsgDepositChannel{} }
func (m *MsgDepositChannel) String() string { return fmt.Sprintf("MsgDepositChannel{%s, %s}", m.ChannelId, m.Amount) }
func (m *MsgDepositChannel) ProtoMessage()  {}

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

type MsgDepositChannelResponse struct{}

func (m *MsgDepositChannelResponse) Reset()         { *m = MsgDepositChannelResponse{} }
func (m *MsgDepositChannelResponse) String() string { return "MsgDepositChannelResponse{}" }
func (m *MsgDepositChannelResponse) ProtoMessage()  {}

// MsgUpdateState submits an off-chain state update.
type MsgUpdateState struct {
	Sender    string             `json:"sender"` // payer or provider
	ChannelId string             `json:"channel_id"`
	Update    ChannelStateUpdate `json:"update"`
}

func (m *MsgUpdateState) Reset()         { *m = MsgUpdateState{} }
func (m *MsgUpdateState) String() string { return fmt.Sprintf("MsgUpdateState{%s, nonce:%d}", m.ChannelId, m.Update.Nonce) }
func (m *MsgUpdateState) ProtoMessage()  {}

func (m *MsgUpdateState) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

type MsgUpdateStateResponse struct{}

func (m *MsgUpdateStateResponse) Reset()         { *m = MsgUpdateStateResponse{} }
func (m *MsgUpdateStateResponse) String() string { return "MsgUpdateStateResponse{}" }
func (m *MsgUpdateStateResponse) ProtoMessage()  {}

// MsgClaimExpired claims refund from an expired channel.
type MsgClaimExpired struct {
	Claimer   string `json:"claimer"` // payer
	ChannelId string `json:"channel_id"`
}

func (m *MsgClaimExpired) Reset()         { *m = MsgClaimExpired{} }
func (m *MsgClaimExpired) String() string { return fmt.Sprintf("MsgClaimExpired{%s}", m.ChannelId) }
func (m *MsgClaimExpired) ProtoMessage()  {}

func (m *MsgClaimExpired) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Claimer); err != nil {
		return fmt.Errorf("invalid claimer address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

type MsgClaimExpiredResponse struct {
	RefundedAmount string `json:"refunded_amount"` // ulgm
}

func (m *MsgClaimExpiredResponse) Reset()         { *m = MsgClaimExpiredResponse{} }
func (m *MsgClaimExpiredResponse) String() string { return fmt.Sprintf("MsgClaimExpiredResponse{%s}", m.RefundedAmount) }
func (m *MsgClaimExpiredResponse) ProtoMessage()  {}
