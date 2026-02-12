package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tokens/types"
)

type msgServer struct {
	Keeper
	types.UnimplementedMsgServer
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
// The returned *msgServer also provides native LGM token methods (Transfer,
// ApproveAllowance, Burn, UpdateParams) that are not part of the proto-generated
// MsgServer interface.
func NewMsgServerImpl(keeper Keeper) *msgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = (*msgServer)(nil)

// Transfer handles MsgTransfer — transfer tokens with period preservation.
//
// Workflow:
// 1. Validate sender and receiver addresses
// 2. Check sender has sufficient active balance
// 3. Transfer using period-preserving FIFO debit from sender
// 4. Credit receiver at sender's original periods (prevents expiration reset)
// 5. Emit events
func (k msgServer) Transfer(goCtx context.Context, msg *types.MsgTransfer) (*types.MsgTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	// Validate addresses
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Receiver); err != nil {
		return nil, fmt.Errorf("invalid receiver address: %w", err)
	}

	amtInt := new(big.Int)
	if _, ok := amtInt.SetString(msg.Amount, 10); !ok || amtInt.Sign() <= 0 {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, msg.Amount)
	}

	if params.FreshExpirationOnTransfer {
		// Fresh expiration mode: debit sender, credit receiver at current period
		if err := k.DebitBalance(ctx, msg.Sender, msg.Amount); err != nil {
			return nil, fmt.Errorf("%w: %v", types.ErrInsufficientBalance, err)
		}
		height := uint64(ctx.BlockHeight())
		if err := k.CreditBalance(ctx, msg.Receiver, msg.Amount, height); err != nil {
			return nil, fmt.Errorf("failed to credit receiver: %w", err)
		}
	} else {
		// Period-preserving mode: maintain sender's period-of-origin on receiver
		if err := k.TransferPreservingPeriods(ctx, msg.Sender, msg.Receiver, msg.Amount); err != nil {
			return nil, err
		}
	}

	// Also execute the underlying bank send for the native coin representation
	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	receiverAddr, _ := sdk.AccAddressFromBech32(msg.Receiver)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amtInt)))
	if err := k.bankKeeper.SendCoins(ctx, senderAddr, receiverAddr, coins); err != nil {
		return nil, fmt.Errorf("bank send failed: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"expirable_transfer",
			sdk.NewAttribute("sender", msg.Sender),
			sdk.NewAttribute("receiver", msg.Receiver),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("preserve_period", fmt.Sprintf("%t", !params.FreshExpirationOnTransfer)),
		),
	)

	k.Logger(ctx).Info("expirable transfer executed",
		"sender", msg.Sender,
		"receiver", msg.Receiver,
		"amount", msg.Amount,
	)

	return &types.MsgTransferResponse{}, nil
}

// TransferFrom handles MsgTransferFrom — transfer tokens on behalf of an owner
// using a previously approved allowance.
//
// Workflow:
// 1. Validate all addresses
// 2. Check spender has sufficient allowance from owner
// 3. Decrement the allowance by the transfer amount
// 4. Transfer from owner to receiver (period-preserving or fresh, per params)
// 5. Execute underlying bank send from owner to receiver
// 6. Emit events
func (k msgServer) TransferFrom(goCtx context.Context, msg *types.MsgTransferFrom) (*types.MsgTransferFromResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	// Validate addresses (proto fields: Spender, From, To)
	if _, err := sdk.AccAddressFromBech32(msg.Spender); err != nil {
		return nil, fmt.Errorf("invalid spender address: %w", err)
	}
	fromAddr, err := sdk.AccAddressFromBech32(msg.From)
	if err != nil {
		return nil, fmt.Errorf("invalid from address: %w", err)
	}
	toAddr, err := sdk.AccAddressFromBech32(msg.To)
	if err != nil {
		return nil, fmt.Errorf("invalid to address: %w", err)
	}

	amtInt := new(big.Int)
	if _, ok := amtInt.SetString(msg.Amount, 10); !ok || amtInt.Sign() <= 0 {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, msg.Amount)
	}

	// Check and decrement allowance
	allowanceStr := k.GetAllowance(ctx, msg.From, msg.Spender)
	allowance := new(big.Int)
	allowance.SetString(allowanceStr, 10)

	if allowance.Cmp(amtInt) < 0 {
		return nil, fmt.Errorf("%w: allowance %s < requested %s",
			types.ErrAllowanceExceeded, allowanceStr, msg.Amount)
	}

	// Decrement allowance
	allowance.Sub(allowance, amtInt)
	if allowance.Sign() == 0 {
		k.DeleteAllowance(ctx, msg.From, msg.Spender)
	} else {
		k.SetAllowance(ctx, msg.From, msg.Spender, allowance.String())
	}

	// Transfer from owner to receiver
	if params.FreshExpirationOnTransfer {
		if err := k.DebitBalance(ctx, msg.From, msg.Amount); err != nil {
			return nil, fmt.Errorf("%w: %v", types.ErrInsufficientBalance, err)
		}
		height := uint64(ctx.BlockHeight())
		if err := k.CreditBalance(ctx, msg.To, msg.Amount, height); err != nil {
			return nil, fmt.Errorf("failed to credit receiver: %w", err)
		}
	} else {
		if err := k.TransferPreservingPeriods(ctx, msg.From, msg.To, msg.Amount); err != nil {
			return nil, err
		}
	}

	// Execute underlying bank send
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amtInt)))
	if err := k.bankKeeper.SendCoins(ctx, fromAddr, toAddr, coins); err != nil {
		return nil, fmt.Errorf("bank send failed: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"expirable_transfer_from",
			sdk.NewAttribute("spender", msg.Spender),
			sdk.NewAttribute("from", msg.From),
			sdk.NewAttribute("to", msg.To),
			sdk.NewAttribute("amount", msg.Amount),
		),
	)

	k.Logger(ctx).Info("expirable transfer_from executed",
		"spender", msg.Spender,
		"from", msg.From,
		"to", msg.To,
		"amount", msg.Amount,
	)

	return &types.MsgTransferFromResponse{}, nil
}

// ApproveAllowance handles MsgApproveAllowance — set spending allowance.
func (k msgServer) ApproveAllowance(goCtx context.Context, msg *types.MsgApproveAllowance) (*types.MsgApproveAllowanceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Spender); err != nil {
		return nil, fmt.Errorf("invalid spender address: %w", err)
	}

	amtInt := new(big.Int)
	if _, ok := amtInt.SetString(msg.Amount, 10); !ok || amtInt.Sign() < 0 {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, msg.Amount)
	}

	if amtInt.Sign() == 0 {
		// Revoke allowance
		k.DeleteAllowance(ctx, msg.Owner, msg.Spender)
	} else {
		k.SetAllowance(ctx, msg.Owner, msg.Spender, msg.Amount)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"allowance_approved",
			sdk.NewAttribute("owner", msg.Owner),
			sdk.NewAttribute("spender", msg.Spender),
			sdk.NewAttribute("amount", msg.Amount),
		),
	)

	k.Logger(ctx).Info("allowance approved",
		"owner", msg.Owner,
		"spender", msg.Spender,
		"amount", msg.Amount,
	)

	return &types.MsgApproveAllowanceResponse{}, nil
}

// Burn handles MsgBurn — destroy tokens from sender's account.
func (k msgServer) Burn(goCtx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	amtInt := new(big.Int)
	if _, ok := amtInt.SetString(msg.Amount, 10); !ok || amtInt.Sign() <= 0 {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, msg.Amount)
	}

	// Debit from expirable account (FIFO)
	if err := k.DebitBalance(ctx, msg.Sender, msg.Amount); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInsufficientBalance, err)
	}

	// Burn from the bank module (sender -> module -> burn)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amtInt)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to send to module: %w", err)
	}
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to burn coins: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"tokens_burned",
			sdk.NewAttribute("sender", msg.Sender),
			sdk.NewAttribute("amount", msg.Amount),
		),
	)

	k.Logger(ctx).Info("tokens burned",
		"sender", msg.Sender,
		"amount", msg.Amount,
	)

	return &types.MsgBurnResponse{}, nil
}

// UpdateParams handles MsgUpdateParams — governance-gated parameter update.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"tokens_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
