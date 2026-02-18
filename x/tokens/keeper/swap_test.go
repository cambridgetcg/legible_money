package keeper_test

import (
	"math/big"
	"testing"

	"github.com/legible-money/legible/x/tokens/keeper"
	"github.com/legible-money/legible/x/tokens/types"
)

// -----------------------------------------------------------------------
// Pool ID determinism
// -----------------------------------------------------------------------

func TestGeneratePoolID_Deterministic(t *testing.T) {
	id1 := keeper.GeneratePoolID("tokenA", "tokenB")
	id2 := keeper.GeneratePoolID("tokenA", "tokenB")
	id3 := keeper.GeneratePoolID("tokenB", "tokenA") // reversed order

	if id1 != id2 {
		t.Fatal("same inputs should produce same pool ID")
	}
	if id1 != id3 {
		t.Fatal("reversed token order should produce same pool ID")
	}
	if len(id1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(id1))
	}
}

func TestGeneratePoolID_DifferentPairs(t *testing.T) {
	id1 := keeper.GeneratePoolID("tokenA", "tokenB")
	id2 := keeper.GeneratePoolID("tokenA", "tokenC")

	if id1 == id2 {
		t.Fatal("different token pairs should produce different pool IDs")
	}
}

// -----------------------------------------------------------------------
// Pool and LP Position CRUD
// -----------------------------------------------------------------------

func TestPoolSetGetRoundTrip(t *testing.T) {
	k, ctx := setupKeeper(t)

	pool := &types.LiquidityPool{
		Id:            "pool123",
		TokenAId:      "tA",
		TokenBId:      "tB",
		ReserveA:      "1000",
		ReserveB:      "2000",
		TotalLpSupply: "1414",
		FeeBps:        30,
	}
	k.SetPool(ctx, pool)

	got := k.GetPool(ctx, "pool123")
	if got == nil {
		t.Fatal("expected pool, got nil")
	}
	if got.ReserveA != "1000" || got.ReserveB != "2000" {
		t.Fatalf("reserves mismatch: %s/%s", got.ReserveA, got.ReserveB)
	}
	if got.FeeBps != 30 {
		t.Fatalf("expected fee 30 bps, got %d", got.FeeBps)
	}
}

func TestPoolNotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	if k.GetPool(ctx, "nonexistent") != nil {
		t.Fatal("expected nil for nonexistent pool")
	}
}

func TestPoolByTokenPair(t *testing.T) {
	k, ctx := setupKeeper(t)

	pool := &types.LiquidityPool{
		Id:       "pool456",
		TokenAId: "alpha",
		TokenBId: "beta",
	}
	k.SetPool(ctx, pool)

	// Lookup by both orders
	got1 := k.GetPoolByTokenPair(ctx, "alpha", "beta")
	got2 := k.GetPoolByTokenPair(ctx, "beta", "alpha")

	if got1 == nil || got2 == nil {
		t.Fatal("expected pool from both orderings")
	}
	if got1.Id != "pool456" || got2.Id != "pool456" {
		t.Fatal("both orderings should return same pool")
	}
}

func TestLPPositionSetGetDelete(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Default is zero
	pos := k.GetLPPosition(ctx, "pool1", testUser1)
	if pos.Sign() != 0 {
		t.Fatalf("expected zero, got %s", pos.String())
	}

	// Set and get
	k.SetLPPosition(ctx, "pool1", testUser1, big.NewInt(500))
	pos = k.GetLPPosition(ctx, "pool1", testUser1)
	if pos.Cmp(big.NewInt(500)) != 0 {
		t.Fatalf("expected 500, got %s", pos.String())
	}

	// Set to zero deletes
	k.SetLPPosition(ctx, "pool1", testUser1, big.NewInt(0))
	pos = k.GetLPPosition(ctx, "pool1", testUser1)
	if pos.Sign() != 0 {
		t.Fatalf("expected zero after delete, got %s", pos.String())
	}
}

// -----------------------------------------------------------------------
// AddLiquidity
// -----------------------------------------------------------------------

func TestAddLiquidity_CreatePool(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "ALPHA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "BETA", allFeatures())

	// Creator has 1000000 of each from initial supply.
	resp, err := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "10000",
		AmountB:  "40000",
	})
	if err != nil {
		t.Fatalf("AddLiquidity failed: %v", err)
	}
	if resp.PoolId == "" {
		t.Fatal("expected non-empty pool ID")
	}

	// LP minted = sqrt(10000 * 40000) = sqrt(400000000) = 20000
	expected := big.NewInt(20000)
	minted := new(big.Int)
	minted.SetString(resp.LpTokensMinted, 10)
	if minted.Cmp(expected) != 0 {
		t.Fatalf("expected LP minted %s, got %s", expected, minted)
	}

	// Verify pool stored
	pool := k.GetPool(ctx, resp.PoolId)
	if pool == nil {
		t.Fatal("pool not found")
	}

	// Verify provider balances deducted
	balA := k.GetBalance(ctx, tokenAId, testCreator)
	if balA.Cmp(big.NewInt(990000)) != 0 {
		t.Fatalf("expected 990000, got %s", balA.String())
	}
	balB := k.GetBalance(ctx, tokenBId, testCreator)
	if balB.Cmp(big.NewInt(960000)) != 0 {
		t.Fatalf("expected 960000, got %s", balB.String())
	}

	// Verify LP position
	lp := k.GetLPPosition(ctx, resp.PoolId, testCreator)
	if lp.Cmp(expected) != 0 {
		t.Fatalf("expected LP position %s, got %s", expected, lp)
	}
}

func TestAddLiquidity_ExistingPool(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "EXA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "EXB", allFeatures())

	// First add creates pool
	resp1, err := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "10000",
		AmountB:  "10000",
	})
	if err != nil {
		t.Fatalf("first AddLiquidity failed: %v", err)
	}

	// Transfer some tokens to user1 for second add
	_, _ = srv.TransferToken(ctx, &types.MsgTransferToken{
		Sender: testCreator, TokenId: tokenAId, To: testUser1, Amount: "5000",
	})
	_, _ = srv.TransferToken(ctx, &types.MsgTransferToken{
		Sender: testCreator, TokenId: tokenBId, To: testUser1, Amount: "5000",
	})

	// Second add by different provider
	resp2, err := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testUser1,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "5000",
		AmountB:  "5000",
	})
	if err != nil {
		t.Fatalf("second AddLiquidity failed: %v", err)
	}
	if resp2.PoolId != resp1.PoolId {
		t.Fatal("second add should use same pool ID")
	}

	// Verify total LP increased
	pool := k.GetPool(ctx, resp1.PoolId)
	totalLP := new(big.Int)
	totalLP.SetString(pool.TotalLpSupply, 10)
	firstLP := new(big.Int)
	firstLP.SetString(resp1.LpTokensMinted, 10)
	secondLP := new(big.Int)
	secondLP.SetString(resp2.LpTokensMinted, 10)
	expectedTotal := new(big.Int).Add(firstLP, secondLP)
	if totalLP.Cmp(expectedTotal) != 0 {
		t.Fatalf("expected total LP %s, got %s", expectedTotal, totalLP)
	}
}

func TestAddLiquidity_SameToken(t *testing.T) {
	_, ctx, srv := setupMsgServer(t)

	tokenId := createTestToken(t, srv, ctx, testCreator, "SAME", allFeatures())

	_, err := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenId,
		TokenBId: tokenId,
		AmountA:  "1000",
		AmountB:  "1000",
	})
	if err == nil {
		t.Fatal("expected error for same token pair")
	}
}

func TestAddLiquidity_InsufficientBalance(t *testing.T) {
	_, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "INSA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "INSB", allFeatures())

	_, err := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "9999999", // more than initial supply of 1000000
		AmountB:  "1000",
	})
	if err == nil {
		t.Fatal("expected insufficient balance error")
	}
}

func TestAddLiquidity_PausedToken(t *testing.T) {
	_, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "PAUA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "PAUB", allFeatures())

	_, _ = srv.PauseToken(ctx, &types.MsgPauseToken{Authority: testCreator, TokenId: tokenAId})

	_, err := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "1000",
		AmountB:  "1000",
	})
	if err == nil {
		t.Fatal("expected paused token error")
	}
}

func TestAddLiquidity_SlippageCheck(t *testing.T) {
	_, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "SLPA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "SLPB", allFeatures())

	_, err := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider:    testCreator,
		TokenAId:    tokenAId,
		TokenBId:    tokenBId,
		AmountA:     "100",
		AmountB:     "100",
		MinLpTokens: "999999", // impossible to get this many LP tokens
	})
	if err == nil {
		t.Fatal("expected slippage exceeded error")
	}
}

func TestAddLiquidity_ReversedTokenOrder(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "REVA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "REVB", allFeatures())

	// Add with B, A order
	resp, err := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenBId, // reversed
		TokenBId: tokenAId, // reversed
		AmountA:  "5000",
		AmountB:  "10000",
	})
	if err != nil {
		t.Fatalf("reversed order AddLiquidity failed: %v", err)
	}

	// Pool should exist with canonical ordering
	pool := k.GetPool(ctx, resp.PoolId)
	if pool == nil {
		t.Fatal("pool not found")
	}
	// Token ordering in pool should be canonical (sorted)
	if pool.TokenAId > pool.TokenBId {
		t.Fatal("pool tokens should be in sorted order")
	}
}

// -----------------------------------------------------------------------
// RemoveLiquidity
// -----------------------------------------------------------------------

func TestRemoveLiquidity_Full(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "RMFA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "RMFB", allFeatures())

	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "10000",
		AmountB:  "10000",
	})

	// Remove all LP tokens
	removeResp, err := srv.RemoveLiquidity(ctx, &types.MsgRemoveLiquidity{
		Provider: testCreator,
		PoolId:   addResp.PoolId,
		LpTokens: addResp.LpTokensMinted,
	})
	if err != nil {
		t.Fatalf("RemoveLiquidity failed: %v", err)
	}

	// Should get back original amounts
	gotA := new(big.Int)
	gotA.SetString(removeResp.AmountA, 10)
	gotB := new(big.Int)
	gotB.SetString(removeResp.AmountB, 10)

	if gotA.Cmp(big.NewInt(10000)) != 0 {
		t.Fatalf("expected 10000 token A back, got %s", gotA)
	}
	if gotB.Cmp(big.NewInt(10000)) != 0 {
		t.Fatalf("expected 10000 token B back, got %s", gotB)
	}

	// LP position should be zero
	lp := k.GetLPPosition(ctx, addResp.PoolId, testCreator)
	if lp.Sign() != 0 {
		t.Fatalf("expected zero LP after full removal, got %s", lp)
	}

	// Balances restored
	balA := k.GetBalance(ctx, tokenAId, testCreator)
	if balA.Cmp(big.NewInt(1000000)) != 0 {
		t.Fatalf("expected original balance 1000000, got %s", balA)
	}
}

func TestRemoveLiquidity_Partial(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "RMPA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "RMPB", allFeatures())

	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "10000",
		AmountB:  "20000",
	})

	// Determine which token is pool.TokenA (canonical sorted order)
	pool := k.GetPool(ctx, addResp.PoolId)
	var expectHalfA, expectHalfB int64
	if pool.TokenAId == tokenAId {
		// Pool canonical order matches msg order
		expectHalfA, expectHalfB = 5000, 10000
	} else {
		// Pool swapped the order: pool.TokenA = tokenBId, pool.TokenB = tokenAId
		expectHalfA, expectHalfB = 10000, 5000
	}

	// Remove half
	totalLP := new(big.Int)
	totalLP.SetString(addResp.LpTokensMinted, 10)
	halfLP := new(big.Int).Div(totalLP, big.NewInt(2))

	removeResp, err := srv.RemoveLiquidity(ctx, &types.MsgRemoveLiquidity{
		Provider: testCreator,
		PoolId:   addResp.PoolId,
		LpTokens: halfLP.String(),
	})
	if err != nil {
		t.Fatalf("partial RemoveLiquidity failed: %v", err)
	}

	gotA := new(big.Int)
	gotA.SetString(removeResp.AmountA, 10)
	gotB := new(big.Int)
	gotB.SetString(removeResp.AmountB, 10)

	// Half of reserves (in pool canonical token order)
	if gotA.Cmp(big.NewInt(expectHalfA)) != 0 {
		t.Fatalf("expected %d pool token A, got %s", expectHalfA, gotA)
	}
	if gotB.Cmp(big.NewInt(expectHalfB)) != 0 {
		t.Fatalf("expected %d pool token B, got %s", expectHalfB, gotB)
	}

	// Remaining LP
	remaining := k.GetLPPosition(ctx, addResp.PoolId, testCreator)
	expectedRemaining := new(big.Int).Sub(totalLP, halfLP)
	if remaining.Cmp(expectedRemaining) != 0 {
		t.Fatalf("expected remaining LP %s, got %s", expectedRemaining, remaining)
	}
}

func TestRemoveLiquidity_InsufficientLP(t *testing.T) {
	_, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "RILA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "RILB", allFeatures())

	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "10000",
		AmountB:  "10000",
	})

	_, err := srv.RemoveLiquidity(ctx, &types.MsgRemoveLiquidity{
		Provider: testCreator,
		PoolId:   addResp.PoolId,
		LpTokens: "999999999", // way more than minted
	})
	if err == nil {
		t.Fatal("expected insufficient LP error")
	}
}

func TestRemoveLiquidity_PoolNotFound(t *testing.T) {
	_, ctx, srv := setupMsgServer(t)

	_, err := srv.RemoveLiquidity(ctx, &types.MsgRemoveLiquidity{
		Provider: testCreator,
		PoolId:   "nonexistent",
		LpTokens: "100",
	})
	if err == nil {
		t.Fatal("expected pool not found error")
	}
}

func TestRemoveLiquidity_SlippageCheck(t *testing.T) {
	_, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "RSLA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "RSLB", allFeatures())

	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "10000",
		AmountB:  "10000",
	})

	_, err := srv.RemoveLiquidity(ctx, &types.MsgRemoveLiquidity{
		Provider:  testCreator,
		PoolId:    addResp.PoolId,
		LpTokens:  "100",
		MinAmountA: "999999", // impossible
	})
	if err == nil {
		t.Fatal("expected slippage exceeded error")
	}
}

// -----------------------------------------------------------------------
// Swap
// -----------------------------------------------------------------------

func TestSwap_Basic(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "SWPA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "SWPB", allFeatures())

	// Create pool: 100000 A / 100000 B
	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "100000",
		AmountB:  "100000",
	})

	pool := k.GetPool(ctx, addResp.PoolId)
	if pool == nil {
		t.Fatal("pool not found")
	}

	// Give user1 some token A to swap
	_, _ = srv.TransferToken(ctx, &types.MsgTransferToken{
		Sender: testCreator, TokenId: tokenAId, To: testUser1, Amount: "10000",
	})

	// Swap 1000 A for B
	swapResp, err := srv.Swap(ctx, &types.MsgSwap{
		Sender:    testUser1,
		PoolId:    addResp.PoolId,
		TokenInId: tokenAId,
		AmountIn:  "1000",
	})
	if err != nil {
		t.Fatalf("Swap failed: %v", err)
	}

	amountOut := new(big.Int)
	amountOut.SetString(swapResp.AmountOut, 10)

	// With 0.3% fee on 1000 input into 100k/100k pool:
	// amountInAfterFee = 1000 * 9970 = 9970000
	// amountOut = 9970000 * 100000 / (100000 * 10000 + 9970000)
	// = 997000000000 / 1009970000 ≈ 987
	if amountOut.Sign() <= 0 {
		t.Fatal("expected positive output amount")
	}
	if amountOut.Cmp(big.NewInt(1000)) >= 0 {
		t.Fatal("output should be less than input due to fee + price impact")
	}
	if amountOut.Cmp(big.NewInt(900)) < 0 {
		t.Fatalf("output too low: %s (expected ~987)", amountOut)
	}

	// Verify user1 balances
	user1BalA := k.GetBalance(ctx, tokenAId, testUser1)
	if user1BalA.Cmp(big.NewInt(9000)) != 0 {
		t.Fatalf("expected user1 token A balance 9000, got %s", user1BalA)
	}
	user1BalB := k.GetBalance(ctx, tokenBId, testUser1)
	if user1BalB.Cmp(amountOut) != 0 {
		t.Fatalf("expected user1 token B balance %s, got %s", amountOut, user1BalB)
	}
}

func TestSwap_ConstantProductInvariant(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "CPKA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "CPKB", allFeatures())

	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "100000",
		AmountB:  "200000",
	})

	pool := k.GetPool(ctx, addResp.PoolId)
	rA := new(big.Int)
	rA.SetString(pool.ReserveA, 10)
	rB := new(big.Int)
	rB.SetString(pool.ReserveB, 10)
	kBefore := new(big.Int).Mul(rA, rB) // k = reserveA * reserveB

	// Give user1 tokens and swap
	_, _ = srv.TransferToken(ctx, &types.MsgTransferToken{
		Sender: testCreator, TokenId: tokenAId, To: testUser1, Amount: "5000",
	})
	_, err := srv.Swap(ctx, &types.MsgSwap{
		Sender:    testUser1,
		PoolId:    addResp.PoolId,
		TokenInId: tokenAId,
		AmountIn:  "5000",
	})
	if err != nil {
		t.Fatalf("Swap failed: %v", err)
	}

	// After swap, k should be >= kBefore (fees increase k)
	poolAfter := k.GetPool(ctx, addResp.PoolId)
	rAAfter := new(big.Int)
	rAAfter.SetString(poolAfter.ReserveA, 10)
	rBAfter := new(big.Int)
	rBAfter.SetString(poolAfter.ReserveB, 10)
	kAfter := new(big.Int).Mul(rAAfter, rBAfter)

	if kAfter.Cmp(kBefore) < 0 {
		t.Fatalf("constant product violated: k before=%s, k after=%s", kBefore, kAfter)
	}
}

func TestSwap_BothDirections(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "BDIA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "BDIB", allFeatures())

	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "100000",
		AmountB:  "100000",
	})

	pool := k.GetPool(ctx, addResp.PoolId)

	// Give user1 some of both tokens
	_, _ = srv.TransferToken(ctx, &types.MsgTransferToken{
		Sender: testCreator, TokenId: tokenAId, To: testUser1, Amount: "10000",
	})
	_, _ = srv.TransferToken(ctx, &types.MsgTransferToken{
		Sender: testCreator, TokenId: tokenBId, To: testUser1, Amount: "10000",
	})

	// Swap A -> B
	_, err := srv.Swap(ctx, &types.MsgSwap{
		Sender:    testUser1,
		PoolId:    addResp.PoolId,
		TokenInId: pool.TokenAId,
		AmountIn:  "1000",
	})
	if err != nil {
		t.Fatalf("A->B swap failed: %v", err)
	}

	// Swap B -> A
	_, err = srv.Swap(ctx, &types.MsgSwap{
		Sender:    testUser1,
		PoolId:    addResp.PoolId,
		TokenInId: pool.TokenBId,
		AmountIn:  "1000",
	})
	if err != nil {
		t.Fatalf("B->A swap failed: %v", err)
	}
}

func TestSwap_PoolNotFound(t *testing.T) {
	_, ctx, srv := setupMsgServer(t)

	_, err := srv.Swap(ctx, &types.MsgSwap{
		Sender:    testCreator,
		PoolId:    "nonexistent",
		TokenInId: "tok1",
		AmountIn:  "100",
	})
	if err == nil {
		t.Fatal("expected pool not found error")
	}
}

func TestSwap_InvalidToken(t *testing.T) {
	_, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "IVTA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "IVTB", allFeatures())
	tokenCId := createTestToken(t, srv, ctx, testCreator, "IVTC", allFeatures())

	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "10000",
		AmountB:  "10000",
	})

	// Swap with a token not in the pool
	_, err := srv.Swap(ctx, &types.MsgSwap{
		Sender:    testCreator,
		PoolId:    addResp.PoolId,
		TokenInId: tokenCId,
		AmountIn:  "100",
	})
	if err == nil {
		t.Fatal("expected invalid pool tokens error")
	}
}

func TestSwap_InsufficientBalance(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "ISBA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "ISBB", allFeatures())

	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "100000",
		AmountB:  "100000",
	})

	pool := k.GetPool(ctx, addResp.PoolId)

	// user1 has no tokens
	_, err := srv.Swap(ctx, &types.MsgSwap{
		Sender:    testUser1,
		PoolId:    addResp.PoolId,
		TokenInId: pool.TokenAId,
		AmountIn:  "100",
	})
	if err == nil {
		t.Fatal("expected insufficient balance error")
	}
}

func TestSwap_SlippageProtection(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "SLSA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "SLSB", allFeatures())

	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "100000",
		AmountB:  "100000",
	})

	pool := k.GetPool(ctx, addResp.PoolId)

	// Swap with unreasonably high min_amount_out
	_, err := srv.Swap(ctx, &types.MsgSwap{
		Sender:       testCreator,
		PoolId:       addResp.PoolId,
		TokenInId:    pool.TokenAId,
		AmountIn:     "1000",
		MinAmountOut: "999999", // impossible
	})
	if err == nil {
		t.Fatal("expected slippage exceeded error")
	}
}

func TestSwap_PausedToken(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "PSWA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "PSWB", allFeatures())

	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "100000",
		AmountB:  "100000",
	})

	pool := k.GetPool(ctx, addResp.PoolId)

	// Pause token A
	_, _ = srv.PauseToken(ctx, &types.MsgPauseToken{Authority: testCreator, TokenId: pool.TokenAId})

	_, err := srv.Swap(ctx, &types.MsgSwap{
		Sender:    testCreator,
		PoolId:    addResp.PoolId,
		TokenInId: pool.TokenAId,
		AmountIn:  "100",
	})
	if err == nil {
		t.Fatal("expected paused token error")
	}
}

// -----------------------------------------------------------------------
// ValidateBasic
// -----------------------------------------------------------------------

func TestValidateBasic_AddLiquidity(t *testing.T) {
	msg := &types.MsgAddLiquidity{
		Provider: testCreator, TokenAId: "tA", TokenBId: "tB",
		AmountA: "100", AmountB: "200",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Fatalf("expected valid: %v", err)
	}

	// Same token
	msg2 := &types.MsgAddLiquidity{
		Provider: testCreator, TokenAId: "tA", TokenBId: "tA",
		AmountA: "100", AmountB: "200",
	}
	if err := msg2.ValidateBasic(); err == nil {
		t.Fatal("expected error for same token")
	}

	// Negative amount
	msg3 := &types.MsgAddLiquidity{
		Provider: testCreator, TokenAId: "tA", TokenBId: "tB",
		AmountA: "-100", AmountB: "200",
	}
	if err := msg3.ValidateBasic(); err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestValidateBasic_RemoveLiquidity(t *testing.T) {
	msg := &types.MsgRemoveLiquidity{
		Provider: testCreator, PoolId: "pool1", LpTokens: "100",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Fatalf("expected valid: %v", err)
	}

	msg2 := &types.MsgRemoveLiquidity{
		Provider: testCreator, PoolId: "pool1", LpTokens: "0",
	}
	if err := msg2.ValidateBasic(); err == nil {
		t.Fatal("expected error for zero LP tokens")
	}
}

func TestValidateBasic_Swap(t *testing.T) {
	msg := &types.MsgSwap{
		Sender: testCreator, PoolId: "pool1", TokenInId: "tA", AmountIn: "100",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Fatalf("expected valid: %v", err)
	}

	msg2 := &types.MsgSwap{
		Sender: testCreator, PoolId: "pool1", TokenInId: "tA", AmountIn: "-100",
	}
	if err := msg2.ValidateBasic(); err == nil {
		t.Fatal("expected error for negative amount")
	}

	msg3 := &types.MsgSwap{
		Sender: testCreator, PoolId: "pool1", TokenInId: "tA",
		AmountIn: "100", MinAmountOut: "-5",
	}
	if err := msg3.ValidateBasic(); err == nil {
		t.Fatal("expected error for negative min_amount_out")
	}
}

// -----------------------------------------------------------------------
// Integration: swap + remove liquidity with fees
// -----------------------------------------------------------------------

func TestSwap_FeesAccrueToPools(t *testing.T) {
	k, ctx, srv := setupMsgServer(t)

	tokenAId := createTestToken(t, srv, ctx, testCreator, "FEEA", allFeatures())
	tokenBId := createTestToken(t, srv, ctx, testCreator, "FEEB", allFeatures())

	// Create pool with 100k/100k
	addResp, _ := srv.AddLiquidity(ctx, &types.MsgAddLiquidity{
		Provider: testCreator,
		TokenAId: tokenAId,
		TokenBId: tokenBId,
		AmountA:  "100000",
		AmountB:  "100000",
	})

	pool := k.GetPool(ctx, addResp.PoolId)

	// Give user1 tokens and do many swaps
	_, _ = srv.TransferToken(ctx, &types.MsgTransferToken{
		Sender: testCreator, TokenId: tokenAId, To: testUser1, Amount: "50000",
	})

	for i := 0; i < 10; i++ {
		_, _ = srv.Swap(ctx, &types.MsgSwap{
			Sender:    testUser1,
			PoolId:    addResp.PoolId,
			TokenInId: pool.TokenAId,
			AmountIn:  "1000",
		})
	}

	// Now LP provider removes all liquidity — should get more than initial deposit
	// because fees accrued
	totalLP := k.GetLPPosition(ctx, addResp.PoolId, testCreator)
	removeResp, err := srv.RemoveLiquidity(ctx, &types.MsgRemoveLiquidity{
		Provider: testCreator,
		PoolId:   addResp.PoolId,
		LpTokens: totalLP.String(),
	})
	if err != nil {
		t.Fatalf("RemoveLiquidity failed: %v", err)
	}

	// Total withdrawn should be > initial 200k (100k + 100k) due to fees
	gotA := new(big.Int)
	gotA.SetString(removeResp.AmountA, 10)
	gotB := new(big.Int)
	gotB.SetString(removeResp.AmountB, 10)
	total := new(big.Int).Add(gotA, gotB)

	if total.Cmp(big.NewInt(200000)) <= 0 {
		t.Fatalf("expected total withdrawn > 200000 due to fees, got %s", total)
	}
}
