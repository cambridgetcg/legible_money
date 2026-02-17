package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/tokens/types"
)

type msgServer struct {
	Keeper
	types.UnimplementedMsgServer
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
// Phase 1 implements 8 core LGM-20 handlers; the remaining 6 return Unimplemented.
func NewMsgServerImpl(keeper Keeper) *msgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = (*msgServer)(nil)

// CreateToken creates a new LGM-20 secondary token.
func (s *msgServer) CreateToken(goCtx context.Context, msg *types.MsgCreateToken) (*types.MsgCreateTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate symbol format
	if !isValidSymbol(msg.Symbol) {
		return nil, types.ErrInvalidSymbol
	}

	// Validate name length
	if len(msg.Name) == 0 || len(msg.Name) > 64 {
		return nil, types.ErrInvalidName
	}

	// Check symbol uniqueness
	if existing := s.GetTokenBySymbol(ctx, msg.Symbol); existing != nil {
		return nil, types.ErrTokenAlreadyExists
	}

	// Parse initial supply (default 0)
	initialSupply := new(big.Int)
	if msg.InitialSupply != "" {
		if _, ok := initialSupply.SetString(msg.InitialSupply, 10); !ok {
			return nil, types.ErrInvalidAmount
		}
	}

	// Parse max supply (default 0 = unlimited)
	maxSupply := new(big.Int)
	if msg.MaxSupply != "" {
		if _, ok := maxSupply.SetString(msg.MaxSupply, 10); !ok {
			return nil, types.ErrInvalidAmount
		}
	}

	// Enforce initial <= max (when max > 0)
	if maxSupply.Sign() > 0 && initialSupply.Cmp(maxSupply) > 0 {
		return nil, types.ErrSupplyExceeded
	}

	// Generate deterministic token ID
	tokenId := GenerateTokenID(msg.Creator, msg.Symbol, ctx.BlockHeight())

	// Default features if not provided
	features := msg.Features
	if features == nil {
		features = &types.TokenFeatures{}
	}

	// Create token definition
	token := &types.TokenDefinition{
		Id:          tokenId,
		Creator:     msg.Creator,
		Name:        msg.Name,
		Symbol:      msg.Symbol,
		Decimals:    msg.Decimals,
		TotalSupply: initialSupply.String(),
		MaxSupply:   maxSupply.String(),
		Features:    features,
		Paused:      false,
		CreatedAt:   uint64(ctx.BlockHeight()),
	}

	s.SetToken(ctx, token)

	// Credit initial supply to creator
	if initialSupply.Sign() > 0 {
		s.SetBalance(ctx, tokenId, msg.Creator, initialSupply)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"create_token",
		sdk.NewAttribute("token_id", tokenId),
		sdk.NewAttribute("creator", msg.Creator),
		sdk.NewAttribute("symbol", msg.Symbol),
		sdk.NewAttribute("initial_supply", initialSupply.String()),
	))

	return &types.MsgCreateTokenResponse{TokenId: tokenId}, nil
}

// MintToken mints new tokens to a recipient. Only creator or governance can mint.
func (s *msgServer) MintToken(goCtx context.Context, msg *types.MsgMintToken) (*types.MsgMintTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	token := s.GetToken(ctx, msg.TokenId)
	if token == nil {
		return nil, types.ErrTokenNotFound
	}

	if token.Paused {
		return nil, types.ErrTokenPaused
	}

	if token.Features == nil || !token.Features.Mintable {
		return nil, types.ErrNotMintable
	}

	// Auth: creator or governance authority
	if msg.Authority != token.Creator && msg.Authority != s.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(msg.Amount, 10); !ok || amount.Sign() <= 0 {
		return nil, types.ErrInvalidAmount
	}

	// Check supply cap
	totalSupply := new(big.Int)
	totalSupply.SetString(token.TotalSupply, 10)
	newSupply := new(big.Int).Add(totalSupply, amount)

	maxSupply := new(big.Int)
	maxSupply.SetString(token.MaxSupply, 10)
	if maxSupply.Sign() > 0 && newSupply.Cmp(maxSupply) > 0 {
		return nil, types.ErrSupplyExceeded
	}

	// Update total supply
	token.TotalSupply = newSupply.String()
	s.SetToken(ctx, token)

	// Credit recipient
	bal := s.GetBalance(ctx, msg.TokenId, msg.To)
	bal.Add(bal, amount)
	s.SetBalance(ctx, msg.TokenId, msg.To, bal)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"mint_token",
		sdk.NewAttribute("token_id", msg.TokenId),
		sdk.NewAttribute("to", msg.To),
		sdk.NewAttribute("amount", amount.String()),
	))

	return &types.MsgMintTokenResponse{}, nil
}

// BurnToken burns tokens from the burner's own balance.
func (s *msgServer) BurnToken(goCtx context.Context, msg *types.MsgBurnToken) (*types.MsgBurnTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	token := s.GetToken(ctx, msg.TokenId)
	if token == nil {
		return nil, types.ErrTokenNotFound
	}

	if token.Paused {
		return nil, types.ErrTokenPaused
	}

	if token.Features == nil || !token.Features.Burnable {
		return nil, types.ErrNotBurnable
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(msg.Amount, 10); !ok || amount.Sign() <= 0 {
		return nil, types.ErrInvalidAmount
	}

	// Check burner balance
	bal := s.GetBalance(ctx, msg.TokenId, msg.Burner)
	if bal.Cmp(amount) < 0 {
		return nil, types.ErrInsufficientBalance
	}

	// Debit burner
	bal.Sub(bal, amount)
	s.SetBalance(ctx, msg.TokenId, msg.Burner, bal)

	// Reduce total supply
	totalSupply := new(big.Int)
	totalSupply.SetString(token.TotalSupply, 10)
	totalSupply.Sub(totalSupply, amount)
	token.TotalSupply = totalSupply.String()
	s.SetToken(ctx, token)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"burn_token",
		sdk.NewAttribute("token_id", msg.TokenId),
		sdk.NewAttribute("burner", msg.Burner),
		sdk.NewAttribute("amount", amount.String()),
	))

	return &types.MsgBurnTokenResponse{}, nil
}

// TransferToken transfers tokens from sender to recipient.
func (s *msgServer) TransferToken(goCtx context.Context, msg *types.MsgTransferToken) (*types.MsgTransferTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	token := s.GetToken(ctx, msg.TokenId)
	if token == nil {
		return nil, types.ErrTokenNotFound
	}

	if token.Paused {
		return nil, types.ErrTokenPaused
	}

	if msg.Sender == msg.To {
		return nil, types.ErrSelfTransfer
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(msg.Amount, 10); !ok || amount.Sign() <= 0 {
		return nil, types.ErrInvalidAmount
	}

	// Check sender balance
	senderBal := s.GetBalance(ctx, msg.TokenId, msg.Sender)
	if senderBal.Cmp(amount) < 0 {
		return nil, types.ErrInsufficientBalance
	}

	// Debit sender, credit recipient
	senderBal.Sub(senderBal, amount)
	s.SetBalance(ctx, msg.TokenId, msg.Sender, senderBal)

	recipientBal := s.GetBalance(ctx, msg.TokenId, msg.To)
	recipientBal.Add(recipientBal, amount)
	s.SetBalance(ctx, msg.TokenId, msg.To, recipientBal)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"transfer_token",
		sdk.NewAttribute("token_id", msg.TokenId),
		sdk.NewAttribute("from", msg.Sender),
		sdk.NewAttribute("to", msg.To),
		sdk.NewAttribute("amount", amount.String()),
	))

	return &types.MsgTransferTokenResponse{}, nil
}

// ApproveToken sets an allowance for a spender. Amount=0 revokes. Replaces previous allowance (ERC-20 semantics).
func (s *msgServer) ApproveToken(goCtx context.Context, msg *types.MsgApproveToken) (*types.MsgApproveTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	token := s.GetToken(ctx, msg.TokenId)
	if token == nil {
		return nil, types.ErrTokenNotFound
	}

	if msg.Owner == msg.Spender {
		return nil, types.ErrSelfTransfer
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(msg.Amount, 10); !ok || amount.Sign() < 0 {
		return nil, types.ErrInvalidAmount
	}

	s.SetAllowance(ctx, msg.TokenId, msg.Owner, msg.Spender, amount)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"approve_token",
		sdk.NewAttribute("token_id", msg.TokenId),
		sdk.NewAttribute("owner", msg.Owner),
		sdk.NewAttribute("spender", msg.Spender),
		sdk.NewAttribute("amount", amount.String()),
	))

	return &types.MsgApproveTokenResponse{}, nil
}

// TransferFrom transfers tokens on behalf of the owner using an allowance.
func (s *msgServer) TransferFrom(goCtx context.Context, msg *types.MsgTransferFrom) (*types.MsgTransferFromResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	token := s.GetToken(ctx, msg.TokenId)
	if token == nil {
		return nil, types.ErrTokenNotFound
	}

	if token.Paused {
		return nil, types.ErrTokenPaused
	}

	if msg.From == msg.To {
		return nil, types.ErrSelfTransfer
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(msg.Amount, 10); !ok || amount.Sign() <= 0 {
		return nil, types.ErrInvalidAmount
	}

	// Check allowance
	allowance := s.GetAllowance(ctx, msg.TokenId, msg.From, msg.Spender)
	if allowance.Cmp(amount) < 0 {
		return nil, types.ErrAllowanceExceeded
	}

	// Check from balance
	fromBal := s.GetBalance(ctx, msg.TokenId, msg.From)
	if fromBal.Cmp(amount) < 0 {
		return nil, types.ErrInsufficientBalance
	}

	// Deduct allowance
	allowance.Sub(allowance, amount)
	s.SetAllowance(ctx, msg.TokenId, msg.From, msg.Spender, allowance)

	// Debit from, credit to
	fromBal.Sub(fromBal, amount)
	s.SetBalance(ctx, msg.TokenId, msg.From, fromBal)

	toBal := s.GetBalance(ctx, msg.TokenId, msg.To)
	toBal.Add(toBal, amount)
	s.SetBalance(ctx, msg.TokenId, msg.To, toBal)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"transfer_from",
		sdk.NewAttribute("token_id", msg.TokenId),
		sdk.NewAttribute("spender", msg.Spender),
		sdk.NewAttribute("from", msg.From),
		sdk.NewAttribute("to", msg.To),
		sdk.NewAttribute("amount", amount.String()),
	))

	return &types.MsgTransferFromResponse{}, nil
}

// PauseToken pauses a token. Only creator or governance can pause.
func (s *msgServer) PauseToken(goCtx context.Context, msg *types.MsgPauseToken) (*types.MsgPauseTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	token := s.GetToken(ctx, msg.TokenId)
	if token == nil {
		return nil, types.ErrTokenNotFound
	}

	if token.Features == nil || !token.Features.Pausable {
		return nil, types.ErrNotPausable
	}

	// Auth: creator or governance authority
	if msg.Authority != token.Creator && msg.Authority != s.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	if token.Paused {
		return nil, types.ErrTokenPaused
	}

	token.Paused = true
	s.SetToken(ctx, token)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"pause_token",
		sdk.NewAttribute("token_id", msg.TokenId),
		sdk.NewAttribute("authority", msg.Authority),
	))

	return &types.MsgPauseTokenResponse{}, nil
}

// UnpauseToken unpauses a paused token. Only creator or governance can unpause.
func (s *msgServer) UnpauseToken(goCtx context.Context, msg *types.MsgUnpauseToken) (*types.MsgUnpauseTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	token := s.GetToken(ctx, msg.TokenId)
	if token == nil {
		return nil, types.ErrTokenNotFound
	}

	if token.Features == nil || !token.Features.Pausable {
		return nil, types.ErrNotPausable
	}

	// Auth: creator or governance authority
	if msg.Authority != token.Creator && msg.Authority != s.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	if !token.Paused {
		return nil, types.ErrTokenNotPaused
	}

	token.Paused = false
	s.SetToken(ctx, token)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"unpause_token",
		sdk.NewAttribute("token_id", msg.TokenId),
		sdk.NewAttribute("authority", msg.Authority),
	))

	return &types.MsgUnpauseTokenResponse{}, nil
}

// ---------- Liquidity Pool Handlers ----------

// DefaultFeeBps is the default swap fee in basis points (0.3%).
const DefaultFeeBps = 30

// AddLiquidity adds liquidity to a pool. Creates the pool if it doesn't exist.
// First provider sets the initial price ratio. Subsequent providers must match it.
// LP tokens minted: first = sqrt(amountA * amountB), subsequent = min(A*LP/rA, B*LP/rB).
func (s *msgServer) AddLiquidity(goCtx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.TokenAId == msg.TokenBId {
		return nil, types.ErrSameToken
	}

	// Validate both tokens exist and are not paused
	tokenA := s.GetToken(ctx, msg.TokenAId)
	if tokenA == nil {
		return nil, types.ErrTokenNotFound
	}
	if tokenA.Paused {
		return nil, types.ErrTokenPaused
	}
	tokenB := s.GetToken(ctx, msg.TokenBId)
	if tokenB == nil {
		return nil, types.ErrTokenNotFound
	}
	if tokenB.Paused {
		return nil, types.ErrTokenPaused
	}

	amountA := new(big.Int)
	if _, ok := amountA.SetString(msg.AmountA, 10); !ok || amountA.Sign() <= 0 {
		return nil, types.ErrInvalidAmount
	}
	amountB := new(big.Int)
	if _, ok := amountB.SetString(msg.AmountB, 10); !ok || amountB.Sign() <= 0 {
		return nil, types.ErrInvalidAmount
	}

	// Check provider has sufficient balances
	balA := s.GetBalance(ctx, msg.TokenAId, msg.Provider)
	if balA.Cmp(amountA) < 0 {
		return nil, types.ErrInsufficientBalance
	}
	balB := s.GetBalance(ctx, msg.TokenBId, msg.Provider)
	if balB.Cmp(amountB) < 0 {
		return nil, types.ErrInsufficientBalance
	}

	// Normalize token order for pool storage (A < B lexicographically)
	poolTokenA, poolTokenB := msg.TokenAId, msg.TokenBId
	poolAmountA, poolAmountB := amountA, amountB
	if poolTokenA > poolTokenB {
		poolTokenA, poolTokenB = poolTokenB, poolTokenA
		poolAmountA, poolAmountB = poolAmountB, poolAmountA
	}

	pool := s.GetPoolByTokenPair(ctx, poolTokenA, poolTokenB)
	var lpMinted *big.Int

	if pool == nil {
		// Create new pool
		poolId := GeneratePoolID(poolTokenA, poolTokenB)

		// First LP: mint sqrt(amountA * amountB)
		product := new(big.Int).Mul(poolAmountA, poolAmountB)
		lpMinted = new(big.Int).Sqrt(product)
		if lpMinted.Sign() == 0 {
			return nil, types.ErrZeroLiquidity
		}

		pool = &types.LiquidityPool{
			Id:            poolId,
			TokenAId:      poolTokenA,
			TokenBId:      poolTokenB,
			ReserveA:      poolAmountA.String(),
			ReserveB:      poolAmountB.String(),
			TotalLpSupply: lpMinted.String(),
			FeeBps:        DefaultFeeBps,
		}
	} else {
		// Existing pool: calculate proportional LP tokens
		reserveA := new(big.Int)
		reserveA.SetString(pool.ReserveA, 10)
		reserveB := new(big.Int)
		reserveB.SetString(pool.ReserveB, 10)
		totalLP := new(big.Int)
		totalLP.SetString(pool.TotalLpSupply, 10)

		if reserveA.Sign() == 0 || reserveB.Sign() == 0 || totalLP.Sign() == 0 {
			return nil, types.ErrInsufficientLiquidity
		}

		// LP tokens = min(amountA * totalLP / reserveA, amountB * totalLP / reserveB)
		lpFromA := new(big.Int).Mul(poolAmountA, totalLP)
		lpFromA.Div(lpFromA, reserveA)
		lpFromB := new(big.Int).Mul(poolAmountB, totalLP)
		lpFromB.Div(lpFromB, reserveB)

		lpMinted = lpFromA
		if lpFromB.Cmp(lpFromA) < 0 {
			lpMinted = lpFromB
		}
		if lpMinted.Sign() == 0 {
			return nil, types.ErrZeroLiquidity
		}

		// Update reserves
		reserveA.Add(reserveA, poolAmountA)
		reserveB.Add(reserveB, poolAmountB)
		totalLP.Add(totalLP, lpMinted)

		pool.ReserveA = reserveA.String()
		pool.ReserveB = reserveB.String()
		pool.TotalLpSupply = totalLP.String()
	}

	// Slippage check on LP tokens
	if msg.MinLpTokens != "" {
		minLP := new(big.Int)
		if _, ok := minLP.SetString(msg.MinLpTokens, 10); ok && lpMinted.Cmp(minLP) < 0 {
			return nil, types.ErrSlippageExceeded
		}
	}

	// Debit provider token balances
	balA.Sub(balA, amountA)
	s.SetBalance(ctx, msg.TokenAId, msg.Provider, balA)
	balB.Sub(balB, amountB)
	s.SetBalance(ctx, msg.TokenBId, msg.Provider, balB)

	// Credit LP position
	existingLP := s.GetLPPosition(ctx, pool.Id, msg.Provider)
	existingLP.Add(existingLP, lpMinted)
	s.SetLPPosition(ctx, pool.Id, msg.Provider, existingLP)

	// Store pool
	s.SetPool(ctx, pool)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"add_liquidity",
		sdk.NewAttribute("pool_id", pool.Id),
		sdk.NewAttribute("provider", msg.Provider),
		sdk.NewAttribute("amount_a", amountA.String()),
		sdk.NewAttribute("amount_b", amountB.String()),
		sdk.NewAttribute("lp_minted", lpMinted.String()),
	))

	return &types.MsgAddLiquidityResponse{
		PoolId:         pool.Id,
		LpTokensMinted: lpMinted.String(),
	}, nil
}

// RemoveLiquidity burns LP tokens and returns the proportional share of pool reserves.
func (s *msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pool := s.GetPool(ctx, msg.PoolId)
	if pool == nil {
		return nil, types.ErrPoolNotFound
	}

	lpTokens := new(big.Int)
	if _, ok := lpTokens.SetString(msg.LpTokens, 10); !ok || lpTokens.Sign() <= 0 {
		return nil, types.ErrInvalidAmount
	}

	// Check provider has enough LP tokens
	providerLP := s.GetLPPosition(ctx, msg.PoolId, msg.Provider)
	if providerLP.Cmp(lpTokens) < 0 {
		return nil, types.ErrInsufficientLP
	}

	reserveA := new(big.Int)
	reserveA.SetString(pool.ReserveA, 10)
	reserveB := new(big.Int)
	reserveB.SetString(pool.ReserveB, 10)
	totalLP := new(big.Int)
	totalLP.SetString(pool.TotalLpSupply, 10)

	if totalLP.Sign() == 0 {
		return nil, types.ErrInsufficientLiquidity
	}

	// Calculate proportional shares: amount = lpTokens * reserve / totalLP
	amountA := new(big.Int).Mul(lpTokens, reserveA)
	amountA.Div(amountA, totalLP)
	amountB := new(big.Int).Mul(lpTokens, reserveB)
	amountB.Div(amountB, totalLP)

	// Slippage checks
	if msg.MinAmountA != "" {
		minA := new(big.Int)
		if _, ok := minA.SetString(msg.MinAmountA, 10); ok && amountA.Cmp(minA) < 0 {
			return nil, types.ErrSlippageExceeded
		}
	}
	if msg.MinAmountB != "" {
		minB := new(big.Int)
		if _, ok := minB.SetString(msg.MinAmountB, 10); ok && amountB.Cmp(minB) < 0 {
			return nil, types.ErrSlippageExceeded
		}
	}

	// Burn LP tokens
	providerLP.Sub(providerLP, lpTokens)
	s.SetLPPosition(ctx, msg.PoolId, msg.Provider, providerLP)

	// Update pool reserves and total supply
	reserveA.Sub(reserveA, amountA)
	reserveB.Sub(reserveB, amountB)
	totalLP.Sub(totalLP, lpTokens)

	pool.ReserveA = reserveA.String()
	pool.ReserveB = reserveB.String()
	pool.TotalLpSupply = totalLP.String()
	s.SetPool(ctx, pool)

	// Credit provider with withdrawn tokens
	balA := s.GetBalance(ctx, pool.TokenAId, msg.Provider)
	balA.Add(balA, amountA)
	s.SetBalance(ctx, pool.TokenAId, msg.Provider, balA)

	balB := s.GetBalance(ctx, pool.TokenBId, msg.Provider)
	balB.Add(balB, amountB)
	s.SetBalance(ctx, pool.TokenBId, msg.Provider, balB)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"remove_liquidity",
		sdk.NewAttribute("pool_id", msg.PoolId),
		sdk.NewAttribute("provider", msg.Provider),
		sdk.NewAttribute("amount_a", amountA.String()),
		sdk.NewAttribute("amount_b", amountB.String()),
		sdk.NewAttribute("lp_burned", lpTokens.String()),
	))

	return &types.MsgRemoveLiquidityResponse{
		AmountA: amountA.String(),
		AmountB: amountB.String(),
	}, nil
}

// Swap executes a constant-product (x*y=k) swap against a liquidity pool.
// Fee is deducted from the input amount before computing the output.
// Formula: amountOut = (amountInAfterFee * reserveOut) / (reserveIn + amountInAfterFee)
func (s *msgServer) Swap(goCtx context.Context, msg *types.MsgSwap) (*types.MsgSwapResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pool := s.GetPool(ctx, msg.PoolId)
	if pool == nil {
		return nil, types.ErrPoolNotFound
	}

	// Determine swap direction
	var reserveIn, reserveOut *big.Int
	var tokenOutId string

	reserveA := new(big.Int)
	reserveA.SetString(pool.ReserveA, 10)
	reserveB := new(big.Int)
	reserveB.SetString(pool.ReserveB, 10)

	switch msg.TokenInId {
	case pool.TokenAId:
		reserveIn = reserveA
		reserveOut = reserveB
		tokenOutId = pool.TokenBId
	case pool.TokenBId:
		reserveIn = reserveB
		reserveOut = reserveA
		tokenOutId = pool.TokenAId
	default:
		return nil, types.ErrInvalidPoolTokens
	}

	if reserveIn.Sign() == 0 || reserveOut.Sign() == 0 {
		return nil, types.ErrInsufficientLiquidity
	}

	amountIn := new(big.Int)
	if _, ok := amountIn.SetString(msg.AmountIn, 10); !ok || amountIn.Sign() <= 0 {
		return nil, types.ErrInvalidAmount
	}

	// Check sender has enough input tokens
	senderBal := s.GetBalance(ctx, msg.TokenInId, msg.Sender)
	if senderBal.Cmp(amountIn) < 0 {
		return nil, types.ErrInsufficientBalance
	}

	// Check input token is not paused
	tokenIn := s.GetToken(ctx, msg.TokenInId)
	if tokenIn != nil && tokenIn.Paused {
		return nil, types.ErrTokenPaused
	}

	// Constant product with fee:
	// amountInAfterFee = amountIn * (10000 - feeBps)
	// amountOut = (amountInAfterFee * reserveOut) / (reserveIn * 10000 + amountInAfterFee)
	feeBps := int64(pool.FeeBps)
	amountInAfterFee := new(big.Int).Mul(amountIn, big.NewInt(10000-feeBps))
	numerator := new(big.Int).Mul(amountInAfterFee, reserveOut)
	denominator := new(big.Int).Add(
		new(big.Int).Mul(reserveIn, big.NewInt(10000)),
		amountInAfterFee,
	)
	amountOut := new(big.Int).Div(numerator, denominator)

	if amountOut.Sign() == 0 {
		return nil, types.ErrInsufficientLiquidity
	}

	// Slippage check
	if msg.MinAmountOut != "" {
		minOut := new(big.Int)
		if _, ok := minOut.SetString(msg.MinAmountOut, 10); ok && amountOut.Cmp(minOut) < 0 {
			return nil, types.ErrSlippageExceeded
		}
	}

	// Execute swap: debit input, credit output
	senderBal.Sub(senderBal, amountIn)
	s.SetBalance(ctx, msg.TokenInId, msg.Sender, senderBal)

	outBal := s.GetBalance(ctx, tokenOutId, msg.Sender)
	outBal.Add(outBal, amountOut)
	s.SetBalance(ctx, tokenOutId, msg.Sender, outBal)

	// Update pool reserves
	reserveIn.Add(reserveIn, amountIn)
	reserveOut.Sub(reserveOut, amountOut)

	if msg.TokenInId == pool.TokenAId {
		pool.ReserveA = reserveIn.String()
		pool.ReserveB = reserveOut.String()
	} else {
		pool.ReserveA = reserveOut.String()
		pool.ReserveB = reserveIn.String()
	}
	s.SetPool(ctx, pool)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"swap",
		sdk.NewAttribute("pool_id", msg.PoolId),
		sdk.NewAttribute("sender", msg.Sender),
		sdk.NewAttribute("token_in", msg.TokenInId),
		sdk.NewAttribute("amount_in", amountIn.String()),
		sdk.NewAttribute("token_out", tokenOutId),
		sdk.NewAttribute("amount_out", amountOut.String()),
		sdk.NewAttribute("fee_bps", fmt.Sprintf("%d", pool.FeeBps)),
	))

	return &types.MsgSwapResponse{AmountOut: amountOut.String()}, nil
}
