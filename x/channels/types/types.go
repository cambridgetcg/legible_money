package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Enums ---

// ChannelStatus represents the lifecycle state of a payment channel.
type ChannelStatus string

const (
	ChannelStatusOpen     ChannelStatus = "open"
	ChannelStatusClosing  ChannelStatus = "closing"
	ChannelStatusDisputed ChannelStatus = "disputed"
	ChannelStatusSettled  ChannelStatus = "settled"
	ChannelStatusClosed   ChannelStatus = "closed"
)

// --- Channel ---

// PaymentChannel represents a unidirectional payment channel.
type PaymentChannel struct {
	ChannelId           string        `json:"channel_id"`
	Payer               string        `json:"payer"`               // bech32
	Provider            string        `json:"provider"`            // bech32
	Deposited           string        `json:"deposited"`           // ulgm
	Spent               string        `json:"spent"`               // ulgm
	Available           string        `json:"available"`           // ulgm
	Status              ChannelStatus `json:"status"`
	OpenedAtBlock       uint64        `json:"opened_at_block"`
	ExpiresAtBlock      uint64        `json:"expires_at_block"`
	Nonce               uint64        `json:"nonce"`
	LastStateHash       string        `json:"last_state_hash"`
	SettlementFrequency uint64        `json:"settlement_frequency"` // blocks between auto-settlements
	LastSettlementBlock uint64        `json:"last_settlement_block"`
	DisputeDeadline     uint64        `json:"dispute_deadline,omitempty"`
	DisputeNonce        uint64        `json:"dispute_nonce,omitempty"`
}

func (c *PaymentChannel) Reset()         { *c = PaymentChannel{} }
func (c *PaymentChannel) String() string { return fmt.Sprintf("PaymentChannel{%s}", c.ChannelId) }
func (c *PaymentChannel) ProtoMessage()  {}

// --- Channel State Update ---

// ChannelStateUpdate represents an off-chain state update for a channel.
type ChannelStateUpdate struct {
	ChannelId         string `json:"channel_id"`
	Nonce             uint64 `json:"nonce"`
	Spent             string `json:"spent"`              // ulgm cumulative
	StateHash         string `json:"state_hash"`
	PayerSignature    string `json:"payer_signature"`
	ProviderSignature string `json:"provider_signature"`
}

// --- Dispute ---

// ChannelDispute represents a dispute on a payment channel.
type ChannelDispute struct {
	ChannelId        string `json:"channel_id"`
	Disputer         string `json:"disputer"`          // Who opened the dispute
	DisputedNonce    uint64 `json:"disputed_nonce"`
	DisputedSpent    string `json:"disputed_spent"`
	DisputeStateHash string `json:"dispute_state_hash"`
	DeadlineBlock    uint64 `json:"deadline_block"`
	Resolved         bool   `json:"resolved"`
	Resolution       string `json:"resolution,omitempty"` // "payer_wins" | "provider_wins" | "split"
}

func (d *ChannelDispute) Reset()         { *d = ChannelDispute{} }
func (d *ChannelDispute) String() string { return fmt.Sprintf("ChannelDispute{%s}", d.ChannelId) }
func (d *ChannelDispute) ProtoMessage()  {}

// --- Params ---

// Params defines the channels module parameters.
type Params struct {
	MinDeposit            string `json:"min_deposit"`              // ulgm
	MaxChannelDuration    uint64 `json:"max_channel_duration"`     // blocks
	DefaultSettlementFreq uint64 `json:"default_settlement_freq"`  // blocks
	DisputePeriodBlocks   uint64 `json:"dispute_period_blocks"`
	ExpiryGracePeriod     uint64 `json:"expiry_grace_period"`      // blocks
	FraudPenaltyBPS       uint64 `json:"fraud_penalty_bps"`        // basis points
	MaxChannelsPerPayer   uint64 `json:"max_channels_per_payer"`
}

func (p *Params) Reset()         { *p = Params{} }
func (p *Params) String() string { return fmt.Sprintf("Params{dispute:%d}", p.DisputePeriodBlocks) }
func (p *Params) ProtoMessage()  {}

func DefaultParams() Params {
	return Params{
		MinDeposit:            "1000000",  // 1 LGM in ulgm
		MaxChannelDuration:    1036800,    // ~30 days at 2.521s blocks
		DefaultSettlementFreq: 100,
		DisputePeriodBlocks:   1000,
		ExpiryGracePeriod:     500,
		FraudPenaltyBPS:       500,        // 5%
		MaxChannelsPerPayer:   50,
	}
}

func (p Params) Validate() error {
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

// --- Genesis ---

// GenesisState defines the channels module genesis state.
type GenesisState struct {
	Params   Params           `json:"params"`
	Channels []PaymentChannel `json:"channels"`
	Disputes []ChannelDispute `json:"disputes"`
}

func (gs *GenesisState) Reset()         { *gs = GenesisState{} }
func (gs *GenesisState) String() string { return fmt.Sprintf("GenesisState{channels:%d}", len(gs.Channels)) }
func (gs *GenesisState) ProtoMessage()  {}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		Channels: []PaymentChannel{},
		Disputes: []ChannelDispute{},
	}
}

func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

// --- Message Types ---

// MsgOpenChannel opens a new payment channel.
type MsgOpenChannel struct {
	Payer               string `json:"payer"`
	Provider            string `json:"provider"`
	Deposit             string `json:"deposit"`  // ulgm
	Duration            uint64 `json:"duration"` // blocks
	SettlementFrequency uint64 `json:"settlement_frequency,omitempty"`
}

func (m *MsgOpenChannel) Reset()         { *m = MsgOpenChannel{} }
func (m *MsgOpenChannel) String() string { return fmt.Sprintf("MsgOpenChannel{%s->%s}", m.Payer, m.Provider) }
func (m *MsgOpenChannel) ProtoMessage()  {}

func (m *MsgOpenChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Payer); err != nil {
		return fmt.Errorf("invalid payer address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}
	if m.Payer == m.Provider {
		return fmt.Errorf("payer and provider cannot be the same address")
	}
	if m.Deposit == "" || m.Deposit == "0" {
		return fmt.Errorf("deposit must be positive")
	}
	if m.Duration == 0 {
		return fmt.Errorf("duration must be positive")
	}
	return nil
}

type MsgOpenChannelResponse struct {
	ChannelId string `json:"channel_id"`
}

func (m *MsgOpenChannelResponse) Reset()         { *m = MsgOpenChannelResponse{} }
func (m *MsgOpenChannelResponse) String() string { return fmt.Sprintf("MsgOpenChannelResponse{%s}", m.ChannelId) }
func (m *MsgOpenChannelResponse) ProtoMessage()  {}

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

// MsgDisputeChannel opens a dispute on a channel.
type MsgDisputeChannel struct {
	Disputer  string `json:"disputer"` // payer or provider
	ChannelId string `json:"channel_id"`
	Nonce     uint64 `json:"nonce"`
	Spent     string `json:"spent"`
	StateHash string `json:"state_hash"`
	Signature string `json:"signature"`
}

func (m *MsgDisputeChannel) Reset()         { *m = MsgDisputeChannel{} }
func (m *MsgDisputeChannel) String() string { return fmt.Sprintf("MsgDisputeChannel{%s}", m.ChannelId) }
func (m *MsgDisputeChannel) ProtoMessage()  {}

func (m *MsgDisputeChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Disputer); err != nil {
		return fmt.Errorf("invalid disputer address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

type MsgDisputeChannelResponse struct {
	DeadlineBlock uint64 `json:"deadline_block"`
}

func (m *MsgDisputeChannelResponse) Reset()         { *m = MsgDisputeChannelResponse{} }
func (m *MsgDisputeChannelResponse) String() string { return fmt.Sprintf("MsgDisputeChannelResponse{deadline:%d}", m.DeadlineBlock) }
func (m *MsgDisputeChannelResponse) ProtoMessage()  {}

// MsgSettleChannel settles a channel (distributes funds).
type MsgSettleChannel struct {
	Sender    string `json:"sender"` // either party
	ChannelId string `json:"channel_id"`
}

func (m *MsgSettleChannel) Reset()         { *m = MsgSettleChannel{} }
func (m *MsgSettleChannel) String() string { return fmt.Sprintf("MsgSettleChannel{%s}", m.ChannelId) }
func (m *MsgSettleChannel) ProtoMessage()  {}

func (m *MsgSettleChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

type MsgSettleChannelResponse struct {
	PayerRefund    string `json:"payer_refund"`
	ProviderPayout string `json:"provider_payout"`
}

func (m *MsgSettleChannelResponse) Reset()         { *m = MsgSettleChannelResponse{} }
func (m *MsgSettleChannelResponse) String() string { return fmt.Sprintf("MsgSettleChannelResponse{refund:%s, payout:%s}", m.PayerRefund, m.ProviderPayout) }
func (m *MsgSettleChannelResponse) ProtoMessage()  {}

// MsgCloseChannel cooperatively closes a channel.
type MsgCloseChannel struct {
	Sender            string `json:"sender"` // either party
	ChannelId         string `json:"channel_id"`
	FinalNonce        uint64 `json:"final_nonce"`
	FinalSpent        string `json:"final_spent"`
	PayerSignature    string `json:"payer_signature"`
	ProviderSignature string `json:"provider_signature"`
}

func (m *MsgCloseChannel) Reset()         { *m = MsgCloseChannel{} }
func (m *MsgCloseChannel) String() string { return fmt.Sprintf("MsgCloseChannel{%s}", m.ChannelId) }
func (m *MsgCloseChannel) ProtoMessage()  {}

func (m *MsgCloseChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

type MsgCloseChannelResponse struct {
	PayerRefund    string `json:"payer_refund"`
	ProviderPayout string `json:"provider_payout"`
}

func (m *MsgCloseChannelResponse) Reset()         { *m = MsgCloseChannelResponse{} }
func (m *MsgCloseChannelResponse) String() string { return fmt.Sprintf("MsgCloseChannelResponse{refund:%s, payout:%s}", m.PayerRefund, m.ProviderPayout) }
func (m *MsgCloseChannelResponse) ProtoMessage()  {}

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

// --- Service Interfaces ---

// MsgServer defines the channels message handler interface.
type MsgServer interface {
	OpenChannel(context.Context, *MsgOpenChannel) (*MsgOpenChannelResponse, error)
	DepositChannel(context.Context, *MsgDepositChannel) (*MsgDepositChannelResponse, error)
	UpdateState(context.Context, *MsgUpdateState) (*MsgUpdateStateResponse, error)
	DisputeChannel(context.Context, *MsgDisputeChannel) (*MsgDisputeChannelResponse, error)
	SettleChannel(context.Context, *MsgSettleChannel) (*MsgSettleChannelResponse, error)
	CloseChannel(context.Context, *MsgCloseChannel) (*MsgCloseChannelResponse, error)
	ClaimExpired(context.Context, *MsgClaimExpired) (*MsgClaimExpiredResponse, error)
}
