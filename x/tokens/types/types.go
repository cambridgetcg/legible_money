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
func (p *Params) Validate() error {
	return nil
}

// -----------------------------------------------------------------------
// ValidateBasic / GetSigners for proto-generated messages
// -----------------------------------------------------------------------

// --- MsgCreateToken ---

func (msg *MsgCreateToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCreateToken) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if msg.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if msg.InitialSupply != "" {
		amt := new(big.Int)
		if _, ok := amt.SetString(msg.InitialSupply, 10); !ok || amt.Sign() < 0 {
			return fmt.Errorf("initial_supply must be a non-negative integer")
		}
	}
	if msg.MaxSupply != "" {
		amt := new(big.Int)
		if _, ok := amt.SetString(msg.MaxSupply, 10); !ok || amt.Sign() < 0 {
			return fmt.Errorf("max_supply must be a non-negative integer")
		}
	}
	return nil
}

// --- MsgMintToken ---

func (msg *MsgMintToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgMintToken) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.TokenId == "" {
		return fmt.Errorf("token_id cannot be empty")
	}
	if msg.To == "" {
		return fmt.Errorf("to cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.To); err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}
	return nil
}

// --- MsgBurnToken ---

func (msg *MsgBurnToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Burner)
	return []sdk.AccAddress{addr}
}

func (msg *MsgBurnToken) ValidateBasic() error {
	if msg.Burner == "" {
		return fmt.Errorf("burner cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Burner); err != nil {
		return fmt.Errorf("invalid burner address: %w", err)
	}
	if msg.TokenId == "" {
		return fmt.Errorf("token_id cannot be empty")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}
	return nil
}

// --- MsgTransferToken ---

func (msg *MsgTransferToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgTransferToken) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.TokenId == "" {
		return fmt.Errorf("token_id cannot be empty")
	}
	if msg.To == "" {
		return fmt.Errorf("to cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.To); err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	if msg.Sender == msg.To {
		return fmt.Errorf("sender and to cannot be the same")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}
	return nil
}

// --- MsgApproveToken ---

func (msg *MsgApproveToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{addr}
}

func (msg *MsgApproveToken) ValidateBasic() error {
	if msg.Owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.TokenId == "" {
		return fmt.Errorf("token_id cannot be empty")
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

// --- MsgTransferFrom ---

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

// --- MsgPauseToken ---

func (msg *MsgPauseToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgPauseToken) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.TokenId == "" {
		return fmt.Errorf("token_id cannot be empty")
	}
	return nil
}

// --- MsgUnpauseToken ---

func (msg *MsgUnpauseToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUnpauseToken) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.TokenId == "" {
		return fmt.Errorf("token_id cannot be empty")
	}
	return nil
}

// --- MsgAddLiquidity ---

func (msg *MsgAddLiquidity) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Provider)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAddLiquidity) ValidateBasic() error {
	if msg.Provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}
	if msg.TokenAId == "" || msg.TokenBId == "" {
		return fmt.Errorf("both token IDs are required")
	}
	if msg.TokenAId == msg.TokenBId {
		return fmt.Errorf("token A and token B cannot be the same")
	}
	amtA := new(big.Int)
	if _, ok := amtA.SetString(msg.AmountA, 10); !ok || amtA.Sign() <= 0 {
		return fmt.Errorf("amount_a must be a positive integer")
	}
	amtB := new(big.Int)
	if _, ok := amtB.SetString(msg.AmountB, 10); !ok || amtB.Sign() <= 0 {
		return fmt.Errorf("amount_b must be a positive integer")
	}
	return nil
}

// --- MsgRemoveLiquidity ---

func (msg *MsgRemoveLiquidity) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Provider)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRemoveLiquidity) ValidateBasic() error {
	if msg.Provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}
	if msg.PoolId == "" {
		return fmt.Errorf("pool_id cannot be empty")
	}
	lp := new(big.Int)
	if _, ok := lp.SetString(msg.LpTokens, 10); !ok || lp.Sign() <= 0 {
		return fmt.Errorf("lp_tokens must be a positive integer")
	}
	return nil
}

// --- MsgUpdateParams ---

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
	if msg.Params != nil {
		if err := msg.Params.Validate(); err != nil {
			return fmt.Errorf("invalid params: %w", err)
		}
	}
	return nil
}

// --- MsgDelegatePower ---

func (msg *MsgDelegatePower) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Delegator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgDelegatePower) ValidateBasic() error {
	if msg.Delegator == "" {
		return fmt.Errorf("delegator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Delegator); err != nil {
		return fmt.Errorf("invalid delegator address: %w", err)
	}
	if msg.TokenId == "" {
		return fmt.Errorf("token_id cannot be empty")
	}
	if msg.Delegate == "" {
		return fmt.Errorf("delegate cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Delegate); err != nil {
		return fmt.Errorf("invalid delegate address: %w", err)
	}
	if msg.Delegator == msg.Delegate {
		return fmt.Errorf("delegator and delegate cannot be the same")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() < 0 {
		return fmt.Errorf("amount must be a non-negative integer")
	}
	return nil
}

// --- MsgWrapToken ---

func (msg *MsgWrapToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgWrapToken) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.TokenId == "" {
		return fmt.Errorf("token_id cannot be empty")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}
	return nil
}

// --- MsgUnwrapToken ---

func (msg *MsgUnwrapToken) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUnwrapToken) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.WrappedDenom == "" {
		return fmt.Errorf("wrapped_denom cannot be empty")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}
	return nil
}

// --- MsgSwap ---

func (msg *MsgSwap) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSwap) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.PoolId == "" {
		return fmt.Errorf("pool_id cannot be empty")
	}
	if msg.TokenInId == "" {
		return fmt.Errorf("token_in_id cannot be empty")
	}
	amtIn := new(big.Int)
	if _, ok := amtIn.SetString(msg.AmountIn, 10); !ok || amtIn.Sign() <= 0 {
		return fmt.Errorf("amount_in must be a positive integer")
	}
	if msg.MinAmountOut != "" {
		minOut := new(big.Int)
		if _, ok := minOut.SetString(msg.MinAmountOut, 10); !ok || minOut.Sign() < 0 {
			return fmt.Errorf("min_amount_out must be a non-negative integer")
		}
	}
	return nil
}
