package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Note: Proto-generated types (TokenDefinition, MsgCreateToken, MsgTransferFrom, etc.)
// are defined in tx.pb.go and state.pb.go.

// Note: Params struct is defined in genesis.pb.go (proto-generated).
// The expiration fields (PeriodBlocks, PeriodsUntilExpiry, etc.) remain in the
// proto schema but are unused — the period overlay has been removed.

// Validate performs parameter validation. No-op since expiration params are unused.
func (p Params) Validate() error {
	return nil
}

// -----------------------------------------------------------------------
// ValidateBasic / GetSigners for proto-generated MsgTransferFrom
// (struct defined in tx.pb.go; fields: Spender, TokenId, From, To, Amount)
// -----------------------------------------------------------------------

func (msg *MsgTransferFrom) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Spender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgTransferFrom) ValidateBasic() error {
	if msg.Spender == "" {
		return fmt.Errorf("spender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Spender); err != nil {
		return fmt.Errorf("invalid spender address: %w", err)
	}
	if msg.From == "" {
		return fmt.Errorf("from cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.From); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	if msg.To == "" {
		return fmt.Errorf("to cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.To); err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	if msg.From == msg.To {
		return fmt.Errorf("from and to cannot be the same")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
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
