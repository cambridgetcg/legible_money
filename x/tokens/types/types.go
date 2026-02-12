package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// -----------------------------------------------------------------------
// Core domain types
// -----------------------------------------------------------------------

// Note: PeriodBalance, ExpirableAccount, ExpirationConfig, RAIHoldingAccount,
// RAIDistribution, and TransferPolicy structs are now defined in state.pb.go
// (proto-generated).

// Note: Params struct is now defined in genesis.pb.go (proto-generated).

// Validate performs parameter validation.
func (p Params) Validate() error {
	if p.PeriodBlocks == 0 {
		return fmt.Errorf("period_blocks must be positive")
	}
	if p.PeriodsUntilExpiry == 0 {
		return fmt.Errorf("periods_until_expiry must be positive")
	}
	if p.SweepBatchSize == 0 {
		return fmt.Errorf("sweep_batch_size must be positive")
	}
	if p.RaiDistributionRate == 0 {
		return fmt.Errorf("rai_distribution_rate must be positive")
	}
	dust := new(big.Int)
	if _, ok := dust.SetString(p.DustThreshold, 10); !ok || dust.Sign() < 0 {
		return fmt.Errorf("invalid dust_threshold: %s", p.DustThreshold)
	}
	return nil
}

// Note: GenesisState struct is now defined in genesis.pb.go (proto-generated).

// -----------------------------------------------------------------------
// Message types — hand-written (not in proto)
// -----------------------------------------------------------------------

// MsgTransfer transfers tokens with period preservation.
type MsgTransfer struct {
	Sender   string `json:"sender"`   // bech32 address
	Receiver string `json:"receiver"` // bech32 address
	Amount   string `json:"amount"`   // bigint as string (ulgm)
}

func (msg *MsgTransfer) Reset()        {}
func (msg *MsgTransfer) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgTransfer) ProtoMessage()  {}

func (msg *MsgTransfer) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgTransfer) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.Receiver == "" {
		return fmt.Errorf("receiver cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Receiver); err != nil {
		return fmt.Errorf("invalid receiver address: %w", err)
	}
	if msg.Sender == msg.Receiver {
		return fmt.Errorf("sender and receiver cannot be the same")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}
	return nil
}

// MsgTransferResponse is the response for MsgTransfer.
type MsgTransferResponse struct{}

func (msg *MsgTransferResponse) Reset()        {}
func (msg *MsgTransferResponse) String() string { return "" }
func (msg *MsgTransferResponse) ProtoMessage()  {}

// MsgApproveAllowance approves a spending allowance.
type MsgApproveAllowance struct {
	Owner   string `json:"owner"`   // bech32 address
	Spender string `json:"spender"` // bech32 address
	Amount  string `json:"amount"`  // bigint as string (ulgm), "0" to revoke
}

func (msg *MsgApproveAllowance) Reset()        {}
func (msg *MsgApproveAllowance) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgApproveAllowance) ProtoMessage()  {}

func (msg *MsgApproveAllowance) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{addr}
}

func (msg *MsgApproveAllowance) ValidateBasic() error {
	if msg.Owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.Spender == "" {
		return fmt.Errorf("spender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Spender); err != nil {
		return fmt.Errorf("invalid spender address: %w", err)
	}
	if msg.Owner == msg.Spender {
		return fmt.Errorf("owner and spender cannot be the same")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() < 0 {
		return fmt.Errorf("amount must be a non-negative integer")
	}
	return nil
}

// MsgApproveAllowanceResponse is the response for MsgApproveAllowance.
type MsgApproveAllowanceResponse struct{}

func (msg *MsgApproveAllowanceResponse) Reset()        {}
func (msg *MsgApproveAllowanceResponse) String() string { return "" }
func (msg *MsgApproveAllowanceResponse) ProtoMessage()  {}

// MsgBurn destroys tokens.
type MsgBurn struct {
	Sender string `json:"sender"` // bech32 address
	Amount string `json:"amount"` // bigint as string (ulgm)
}

func (msg *MsgBurn) Reset()        {}
func (msg *MsgBurn) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgBurn) ProtoMessage()  {}

func (msg *MsgBurn) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgBurn) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}
	return nil
}

// MsgBurnResponse is the response for MsgBurn.
type MsgBurnResponse struct{}

func (msg *MsgBurnResponse) Reset()        {}
func (msg *MsgBurnResponse) String() string { return "" }
func (msg *MsgBurnResponse) ProtoMessage()  {}

// MsgUpdateParams updates module parameters (governance-gated).
type MsgUpdateParams struct {
	Authority string `json:"authority"` // bech32 address (gov module)
	Params    Params `json:"params"`
}

func (msg *MsgUpdateParams) Reset()        {}
func (msg *MsgUpdateParams) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgUpdateParams) ProtoMessage()  {}

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

// MsgUpdateParamsResponse is the response for MsgUpdateParams.
type MsgUpdateParamsResponse struct{}

func (msg *MsgUpdateParamsResponse) Reset()        {}
func (msg *MsgUpdateParamsResponse) String() string { return "" }
func (msg *MsgUpdateParamsResponse) ProtoMessage()  {}

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

// Note: MsgServer interface, RegisterMsgServer, and all gRPC handler functions
// are now defined in tx_grpc.pb.go (proto-generated).
