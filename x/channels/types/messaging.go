package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Channel type constants.
const (
	ChannelTypeKnowledge  = "knowledge"
	ChannelTypeGovernance = "governance"
	ChannelTypeSocial     = "social"
	ChannelTypeTrade      = "trade"

	// DefaultMaxHistory is the default FIFO history cap per channel.
	DefaultMaxHistory = 1000
)

// ValidChannelTypes is the set of valid channel types.
var ValidChannelTypes = map[string]bool{
	ChannelTypeKnowledge:  true,
	ChannelTypeGovernance: true,
	ChannelTypeSocial:     true,
	ChannelTypeTrade:      true,
}

// MessageChannel represents a typed communication channel for agents.
type MessageChannel struct {
	ChannelId    string   `json:"channel_id"`
	ChannelType  string   `json:"channel_type"` // knowledge, governance, social, trade
	Creator      string   `json:"creator"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreatedAt    uint64   `json:"created_at"`     // block height
	MaxHistory   uint64   `json:"max_history"`    // FIFO cap
	MessageCount uint64   `json:"message_count"`  // total messages sent (monotonic)
}

// ChannelMessage represents a message sent in a channel.
type ChannelMessage struct {
	MessageId string   `json:"message_id"`
	ChannelId string   `json:"channel_id"`
	Sender    string   `json:"sender"`
	Content   string   `json:"content"`
	SentAt    uint64   `json:"sent_at"`              // block height
	AckedBy   []string `json:"acked_by,omitempty"`
	Sequence  uint64   `json:"sequence"`             // per-channel monotonic
}

// ValidateChannelType returns an error if the channel type is not valid.
func ValidateChannelType(ct string) error {
	if !ValidChannelTypes[ct] {
		return fmt.Errorf("invalid channel type %q: must be one of knowledge, governance, social, trade", ct)
	}
	return nil
}

// ValidateMessageChannel validates a MessageChannel struct.
func ValidateMessageChannel(ch *MessageChannel) error {
	if ch.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	if err := ValidateChannelType(ch.ChannelType); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(ch.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if ch.Name == "" {
		return fmt.Errorf("channel name cannot be empty")
	}
	if ch.MaxHistory == 0 {
		return fmt.Errorf("max history must be positive")
	}
	return nil
}
