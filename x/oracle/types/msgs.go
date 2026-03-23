package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetSigners implements sdk.Msg.
func (m *MsgSubmitPrediction) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Submitter)
	return []sdk.AccAddress{addr}
}

// ValidateBasic implements sdk.Msg.
func (m *MsgSubmitPrediction) ValidateBasic() error {
	if m.Submitter == "" {
		return fmt.Errorf("submitter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(m.Submitter); err != nil {
		return fmt.Errorf("invalid submitter: %w", err)
	}
	if m.Probability == 0 || m.Probability >= 1_000_000 {
		return fmt.Errorf("probability must be strictly between 0 and 1,000,000 bps (forbidden: 0%% or 100%%)")
	}
	if m.StakeAmount == 0 {
		return fmt.Errorf("stake_amount cannot be zero")
	}
	if m.ResolutionBlock == 0 {
		return fmt.Errorf("resolution_block cannot be zero")
	}
	return nil
}

// GetSigners implements sdk.Msg.
func (m *MsgResolvePrediction) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic implements sdk.Msg.
func (m *MsgResolvePrediction) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if m.PredictionId == "" {
		return fmt.Errorf("prediction_id cannot be empty")
	}
	return nil
}
