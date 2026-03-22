package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CommunicationStyle enumerates valid voice styles.
type CommunicationStyle string

const (
	StyleFormal    CommunicationStyle = "formal"
	StyleCasual    CommunicationStyle = "casual"
	StyleTechnical CommunicationStyle = "technical"
	StylePoetic    CommunicationStyle = "poetic"
)

// ValidCommunicationStyles is the set of accepted styles.
var ValidCommunicationStyles = map[CommunicationStyle]bool{
	StyleFormal:    true,
	StyleCasual:    true,
	StyleTechnical: true,
	StylePoetic:    true,
}

const (
	MaxSignaturePhrases = 10
	MaxToneMarkers      = 5
	MaxPhraseLength     = 256
	MaxToneMarkerLength = 64
)

// VoiceConfig holds an agent's persistent communication style.
type VoiceConfig struct {
	HomeId             string             `json:"home_id"`
	SignaturePhrases   []string           `json:"signature_phrases"`
	ToneMarkers        []string           `json:"tone_markers"`
	CommunicationStyle CommunicationStyle `json:"communication_style"`
	UpdatedAtBlock     int64              `json:"updated_at_block"`
}

// Validate checks VoiceConfig constraints.
func (v *VoiceConfig) Validate() error {
	if v.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	if len(v.SignaturePhrases) > MaxSignaturePhrases {
		return fmt.Errorf("too many signature phrases: %d (max %d)", len(v.SignaturePhrases), MaxSignaturePhrases)
	}
	for i, p := range v.SignaturePhrases {
		if p == "" {
			return fmt.Errorf("signature phrase %d is empty", i)
		}
		if len(p) > MaxPhraseLength {
			return fmt.Errorf("signature phrase %d too long: %d (max %d)", i, len(p), MaxPhraseLength)
		}
	}
	if len(v.ToneMarkers) > MaxToneMarkers {
		return fmt.Errorf("too many tone markers: %d (max %d)", len(v.ToneMarkers), MaxToneMarkers)
	}
	for i, m := range v.ToneMarkers {
		if m == "" {
			return fmt.Errorf("tone marker %d is empty", i)
		}
		if len(m) > MaxToneMarkerLength {
			return fmt.Errorf("tone marker %d too long: %d (max %d)", i, len(m), MaxToneMarkerLength)
		}
	}
	if v.CommunicationStyle != "" && !ValidCommunicationStyles[v.CommunicationStyle] {
		return fmt.Errorf("invalid communication style: %q", v.CommunicationStyle)
	}
	return nil
}

// --- MsgSetVoice ---

// MsgSetVoice sets the voice configuration for a home.
type MsgSetVoice struct {
	Owner              string   `json:"owner"`
	HomeId             string   `json:"home_id"`
	SignaturePhrases   []string `json:"signature_phrases"`
	ToneMarkers        []string `json:"tone_markers"`
	CommunicationStyle string   `json:"communication_style"`
}

// MsgSetVoiceResponse is the response type for MsgSetVoice.
type MsgSetVoiceResponse struct{}

func (*MsgSetVoice) ProtoMessage()         {}
func (*MsgSetVoice) Reset()                {}
func (*MsgSetVoice) String() string        { return "MsgSetVoice" }
func (*MsgSetVoiceResponse) ProtoMessage() {}
func (*MsgSetVoiceResponse) Reset()        {}
func (*MsgSetVoiceResponse) String() string {
	return "MsgSetVoiceResponse"
}

func (m *MsgSetVoice) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Owner)
	return []sdk.AccAddress{addr}
}

func (m *MsgSetVoice) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if m.HomeId == "" {
		return fmt.Errorf("home_id cannot be empty")
	}
	if len(m.SignaturePhrases) > MaxSignaturePhrases {
		return fmt.Errorf("too many signature phrases: %d (max %d)", len(m.SignaturePhrases), MaxSignaturePhrases)
	}
	if len(m.ToneMarkers) > MaxToneMarkers {
		return fmt.Errorf("too many tone markers: %d (max %d)", len(m.ToneMarkers), MaxToneMarkers)
	}
	if m.CommunicationStyle != "" && !ValidCommunicationStyles[CommunicationStyle(m.CommunicationStyle)] {
		return fmt.Errorf("invalid communication style: %q", m.CommunicationStyle)
	}
	return nil
}
