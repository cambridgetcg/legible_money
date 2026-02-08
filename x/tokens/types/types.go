package types

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// -----------------------------------------------------------------------
// Core domain types
// -----------------------------------------------------------------------

// PeriodBalance represents tokens minted in a specific period.
type PeriodBalance struct {
	Period    uint64 `json:"period"`     // period index (floor(blockHeight / periodBlocks))
	Balance   string `json:"balance"`    // amount in ulgm
	ExpiresAt uint64 `json:"expires_at"` // block height when this period expires
	Swept     bool   `json:"swept"`      // whether expired tokens were retrieved
}

func (pb *PeriodBalance) Reset()        {}
func (pb *PeriodBalance) String() string { return fmt.Sprintf("%+v", *pb) }
func (pb *PeriodBalance) ProtoMessage()  {}

// ExpirableAccount tracks period-based balances for an account.
type ExpirableAccount struct {
	Address        string          `json:"address"`         // bech32 address
	TotalBalance   string          `json:"total_balance"`   // sum of all active period balances
	PeriodBalances []PeriodBalance `json:"period_balances"` // ordered by period
	EarliestPeriod uint64          `json:"earliest_period"` // oldest non-zero period
	LatestPeriod   uint64          `json:"latest_period"`   // newest non-zero period
	TotalExpired   string          `json:"total_expired"`   // cumulative expired amount
	AccountType    string          `json:"account_type"`    // "standard", "validator", "foundation", "research"
}

func (ea *ExpirableAccount) Reset()        {}
func (ea *ExpirableAccount) String() string { return fmt.Sprintf("%+v", *ea) }
func (ea *ExpirableAccount) ProtoMessage()  {}

// ExpirationConfig configures the token expiration parameters.
type ExpirationConfig struct {
	PeriodBlocks       uint64 `json:"period_blocks"`        // blocks per period (default: 34272 = ~1 day)
	PeriodsUntilExpiry uint64 `json:"periods_until_expiry"` // periods until tokens expire (default: 365 = ~1 year)
	DustThreshold      string `json:"dust_threshold"`       // minimum balance to track
	SweepBatchSize     uint32 `json:"sweep_batch_size"`     // max accounts per sweep
}

func (ec *ExpirationConfig) Reset()        {}
func (ec *ExpirationConfig) String() string { return fmt.Sprintf("%+v", *ec) }
func (ec *ExpirationConfig) ProtoMessage()  {}

// RAIHoldingAccount tracks the Regenerative Agent Income pool.
type RAIHoldingAccount struct {
	Balance          string `json:"balance"`           // current pool balance
	TotalReceived    string `json:"total_received"`    // cumulative from expiration
	TotalDistributed string `json:"total_distributed"` // cumulative distributed
	LastDistBlock    uint64 `json:"last_dist_block"`   // last distribution block
}

func (ra *RAIHoldingAccount) Reset()        {}
func (ra *RAIHoldingAccount) String() string { return fmt.Sprintf("%+v", *ra) }
func (ra *RAIHoldingAccount) ProtoMessage()  {}

// RAIDistribution represents a distribution event.
type RAIDistribution struct {
	Id             string `json:"id"`              // distribution ID
	Block          uint64 `json:"block"`           // block height
	TotalAmount    string `json:"total_amount"`    // total distributed
	RecipientCount uint32 `json:"recipient_count"` // number of recipients
	PerRecipient   string `json:"per_recipient"`   // amount per recipient
}

func (rd *RAIDistribution) Reset()        {}
func (rd *RAIDistribution) String() string { return fmt.Sprintf("%+v", *rd) }
func (rd *RAIDistribution) ProtoMessage()  {}

// TransferPolicy configures transfer behavior.
type TransferPolicy struct {
	FreshExpirationOnReceive bool   `json:"fresh_expiration_on_receive"` // false = preserve sender's period
	MaxTransferPerBlock      string `json:"max_transfer_per_block"`     // max per block (0 = unlimited)
}

func (tp *TransferPolicy) Reset()        {}
func (tp *TransferPolicy) String() string { return fmt.Sprintf("%+v", *tp) }
func (tp *TransferPolicy) ProtoMessage()  {}

// Params holds module parameters.
type Params struct {
	PeriodBlocks              uint64 `json:"period_blocks"`                // blocks per period
	PeriodsUntilExpiry        uint64 `json:"periods_until_expiry"`        // periods until expiration
	DustThreshold             string `json:"dust_threshold"`              // minimum tracked balance
	SweepBatchSize            uint32 `json:"sweep_batch_size"`            // accounts per sweep
	RAIDistributionRate       uint64 `json:"rai_distribution_rate"`       // blocks between RAI distributions
	FreshExpirationOnTransfer bool   `json:"fresh_expiration_on_transfer"` // whether transfers get fresh expiration
}

func (p *Params) Reset()        {}
func (p *Params) String() string { return fmt.Sprintf("%+v", *p) }
func (p *Params) ProtoMessage()  {}

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
	if p.RAIDistributionRate == 0 {
		return fmt.Errorf("rai_distribution_rate must be positive")
	}
	dust := new(big.Int)
	if _, ok := dust.SetString(p.DustThreshold, 10); !ok || dust.Sign() < 0 {
		return fmt.Errorf("invalid dust_threshold: %s", p.DustThreshold)
	}
	return nil
}

// -----------------------------------------------------------------------
// Genesis state
// -----------------------------------------------------------------------

// GenesisState defines the tokens module genesis state.
type GenesisState struct {
	Params   Params              `json:"params"`
	Accounts []ExpirableAccount  `json:"accounts"`
	RAI      RAIHoldingAccount   `json:"rai"`
}

func (gs *GenesisState) Reset()        {}
func (gs *GenesisState) String() string { return fmt.Sprintf("%+v", *gs) }
func (gs *GenesisState) ProtoMessage()  {}

// -----------------------------------------------------------------------
// Message types
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
// MsgServer interface
// -----------------------------------------------------------------------

// MsgServer defines the tokens module message service.
type MsgServer interface {
	Transfer(context.Context, *MsgTransfer) (*MsgTransferResponse, error)
	ApproveAllowance(context.Context, *MsgApproveAllowance) (*MsgApproveAllowanceResponse, error)
	Burn(context.Context, *MsgBurn) (*MsgBurnResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}

// -----------------------------------------------------------------------
// gRPC service descriptor (hand-written, replaces proto generation)
// -----------------------------------------------------------------------

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.tokens.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Transfer", Handler: nil},
		{MethodName: "ApproveAllowance", Handler: nil},
		{MethodName: "Burn", Handler: nil},
		{MethodName: "UpdateParams", Handler: nil},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/tokens/v1/tx.proto",
}

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}
