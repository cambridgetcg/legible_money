package keeper_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/billing/keeper"
	"github.com/legible-money/legible/x/billing/types"
)

// -----------------------------------------------------------------------
// Mock BankKeeper
// -----------------------------------------------------------------------

type mockBankKeeper struct {
	balances map[string]sdkmath.Int
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]sdkmath.Int),
	}
}

func (m *mockBankKeeper) setBalance(addr sdk.AccAddress, denom string, amount sdkmath.Int) {
	m.balances[addr.String()+"/"+denom] = amount
}

func (m *mockBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		fromKey := fromAddr.String() + "/" + coin.Denom
		toKey := toAddr.String() + "/" + coin.Denom

		fromBal, ok := m.balances[fromKey]
		if !ok {
			fromBal = sdkmath.ZeroInt()
		}
		if fromBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance")
		}
		m.balances[fromKey] = fromBal.Sub(coin.Amount)

		toBal, ok := m.balances[toKey]
		if !ok {
			toBal = sdkmath.ZeroInt()
		}
		m.balances[toKey] = toBal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		key := senderAddr.String() + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		if bal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance for send to module")
		}
		m.balances[key] = bal.Sub(coin.Amount)
		// Credit module account
		modKey := recipientModule + "/" + coin.Denom
		modBal, ok := m.balances[modKey]
		if !ok {
			modBal = sdkmath.ZeroInt()
		}
		m.balances[modKey] = modBal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		modKey := senderModule + "/" + coin.Denom
		modBal, ok := m.balances[modKey]
		if !ok {
			modBal = sdkmath.ZeroInt()
		}
		if modBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient module balance")
		}
		m.balances[modKey] = modBal.Sub(coin.Amount)

		key := recipientAddr.String() + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		m.balances[key] = bal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		fromKey := senderModule + "/" + coin.Denom
		toKey := recipientModule + "/" + coin.Denom

		fromBal, ok := m.balances[fromKey]
		if !ok {
			fromBal = sdkmath.ZeroInt()
		}
		if fromBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient module balance")
		}
		m.balances[fromKey] = fromBal.Sub(coin.Amount)

		toBal, ok := m.balances[toKey]
		if !ok {
			toBal = sdkmath.ZeroInt()
		}
		m.balances[toKey] = toBal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	for _, coin := range amt {
		key := moduleName + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		if bal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance to burn")
		}
		m.balances[key] = bal.Sub(coin.Amount)
	}
	return nil
}

// -----------------------------------------------------------------------
// Mock KnowledgeKeeper
// -----------------------------------------------------------------------

type mockKnowledgeKeeper struct {
	facts map[string]mockFact
}

type mockFact struct {
	confidence    uint64
	citationCount uint64
	submitter     string
	createdBlock  uint64
}

func newMockKnowledgeKeeper() *mockKnowledgeKeeper {
	return &mockKnowledgeKeeper{
		facts: make(map[string]mockFact),
	}
}

func (m *mockKnowledgeKeeper) addFact(factId string, confidence, citations, createdBlock uint64, submitter string) {
	m.facts[factId] = mockFact{
		confidence:    confidence,
		citationCount: citations,
		submitter:     submitter,
		createdBlock:  createdBlock,
	}
}

func (m *mockKnowledgeKeeper) GetFactConfidence(_ sdk.Context, factId string) (uint64, bool) {
	f, ok := m.facts[factId]
	if !ok {
		return 0, false
	}
	return f.confidence, true
}

func (m *mockKnowledgeKeeper) GetFactCitationCount(_ sdk.Context, factId string) (uint64, bool) {
	f, ok := m.facts[factId]
	if !ok {
		return 0, false
	}
	return f.citationCount, true
}

func (m *mockKnowledgeKeeper) GetFactSubmitter(_ sdk.Context, factId string) (string, bool) {
	f, ok := m.facts[factId]
	if !ok {
		return "", false
	}
	return f.submitter, true
}

func (m *mockKnowledgeKeeper) GetFactCreatedBlock(_ sdk.Context, factId string) (uint64, bool) {
	f, ok := m.facts[factId]
	if !ok {
		return 0, false
	}
	return f.createdBlock, true
}

func (m *mockKnowledgeKeeper) IncrementCitationCount(_ sdk.Context, factId string) error {
	f, ok := m.facts[factId]
	if !ok {
		return fmt.Errorf("fact not found: %s", factId)
	}
	f.citationCount++
	m.facts[factId] = f
	return nil
}

// -----------------------------------------------------------------------
// Test Setup
// -----------------------------------------------------------------------

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper, *mockKnowledgeKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	bk := newMockBankKeeper()
	kk := newMockKnowledgeKeeper()

	authority := sdk.AccAddress([]byte("authority-addr------")).String()
	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), bk, kk, authority)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx, bk, kk
}

func testAddr(i int) string {
	addr := sdk.AccAddress([]byte(fmt.Sprintf("test-addr-%010d", i)))
	return addr.String()
}

// -----------------------------------------------------------------------
// Tests: Params
// -----------------------------------------------------------------------

func TestParamsDefault(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	params := k.GetParams(ctx)
	if params.BaseCostPerFact != "1000000" {
		t.Errorf("expected BaseCostPerFact 1000000, got %s", params.BaseCostPerFact)
	}
	if params.ResearchShareBps != 220000 {
		t.Errorf("expected ResearchShareBps 220000, got %d", params.ResearchShareBps)
	}
	if params.QuoteValidityBlocks != 100 {
		t.Errorf("expected QuoteValidityBlocks 100, got %d", params.QuoteValidityBlocks)
	}
}

func TestParamsSetGet(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	custom := types.Params{
		BaseCostPerFact:       "2000000",
		ConfidenceThreshold:   600000,
		ConfidencePremiumBps:  300000,
		FreshnessWindowBlocks: 500,
		FreshnessDiscountBps:  150000,
		QuoteValidityBlocks:   200,
		MinProviderStake:      "50000000",
		ResearchShareBps:      250000,
	}
	k.SetParams(ctx, custom)
	got := k.GetParams(ctx)
	if got.BaseCostPerFact != "2000000" {
		t.Errorf("expected BaseCostPerFact 2000000, got %s", got.BaseCostPerFact)
	}
	if got.ConfidenceThreshold != 600000 {
		t.Errorf("expected ConfidenceThreshold 600000, got %d", got.ConfidenceThreshold)
	}
	if got.ResearchShareBps != 250000 {
		t.Errorf("expected ResearchShareBps 250000, got %d", got.ResearchShareBps)
	}
}

// -----------------------------------------------------------------------
// Tests: Provider CRUD
// -----------------------------------------------------------------------

func TestProviderCRUD(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	addr := testAddr(1)

	// Not found initially
	_, found := k.GetProvider(ctx, addr)
	if found {
		t.Error("expected provider not found")
	}

	// Set
	provider := types.Provider{
		Address:      addr,
		Domains:      []string{"mathematics", "physics"},
		StakeAmount:  "100000000",
		Active:       true,
		TotalCalls:   0,
		TotalRevenue: "0",
		RegisteredAt: 100,
	}
	k.SetProvider(ctx, provider)

	// Get
	got, found := k.GetProvider(ctx, addr)
	if !found {
		t.Fatal("expected provider to be found")
	}
	if got.StakeAmount != "100000000" {
		t.Errorf("expected stake 100000000, got %s", got.StakeAmount)
	}
	if len(got.Domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(got.Domains))
	}

	// Delete
	k.DeleteProvider(ctx, addr)
	_, found = k.GetProvider(ctx, addr)
	if found {
		t.Error("expected provider to be deleted")
	}
}

func TestGetAllProviders(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	for i := 0; i < 3; i++ {
		k.SetProvider(ctx, types.Provider{
			Address:      testAddr(i),
			Domains:      []string{"test"},
			StakeAmount:  "100000000",
			Active:       true,
			TotalRevenue: "0",
		})
	}
	all := k.GetAllProviders(ctx)
	if len(all) != 3 {
		t.Errorf("expected 3 providers, got %d", len(all))
	}
}

// -----------------------------------------------------------------------
// Tests: Distribution
// -----------------------------------------------------------------------

func TestCalculateDistribution(t *testing.T) {
	k, ctx, _, kk := setupKeeper(t)

	// Use block height past the research share ramp period (default ~90 days)
	// so that the full 22% research share rate applies.
	ctx = ctx.WithBlockHeight(4_000_000)

	// Set up facts
	submitter1 := testAddr(10)
	submitter2 := testAddr(11)
	kk.addFact("fact-1", 800000, 100, 50, submitter1) // high confidence, many citations
	kk.addFact("fact-2", 300000, 10, 80, submitter2)  // low confidence, few citations

	totalPayment := big.NewInt(10000000) // 10 LGM
	distribution := k.CalculateDistribution(ctx, totalPayment, []string{"fact-1", "fact-2"})

	// Verify research share = 22%
	researchShare := new(big.Int)
	researchShare.SetString(distribution.ResearchShare, 10)
	expectedResearch := big.NewInt(2200000) // 10M * 220000 / 1000000 = 2.2M
	if researchShare.Cmp(expectedResearch) != 0 {
		t.Errorf("expected research share %s, got %s", expectedResearch.String(), researchShare.String())
	}

	// Verify remaining = 78%
	remaining := new(big.Int).Sub(totalPayment, researchShare)
	expectedRemaining := big.NewInt(7800000) // 7.8M
	if remaining.Cmp(expectedRemaining) != 0 {
		t.Errorf("expected remaining %s, got %s", expectedRemaining.String(), remaining.String())
	}

	// Verify provider share = 60% of remaining
	providerShare := new(big.Int)
	providerShare.SetString(distribution.ProviderShare, 10)
	expectedProvider := big.NewInt(4680000) // 7.8M * 60% = 4.68M
	if providerShare.Cmp(expectedProvider) != 0 {
		t.Errorf("expected provider share %s, got %s", expectedProvider.String(), providerShare.String())
	}

	// Verify all shares sum to total
	sum := new(big.Int)
	sum.Add(sum, researchShare)
	sum.Add(sum, providerShare)

	knowledgePoolSum := new(big.Int)
	for _, entry := range distribution.KnowledgePool {
		amt := new(big.Int)
		amt.SetString(entry.Amount, 10)
		knowledgePoolSum.Add(knowledgePoolSum, amt)
	}
	sum.Add(sum, knowledgePoolSum)

	burnAmt := new(big.Int)
	burnAmt.SetString(distribution.ProtocolBurn, 10)
	sum.Add(sum, burnAmt)

	treasuryAmt := new(big.Int)
	treasuryAmt.SetString(distribution.ProtocolTreasury, 10)
	sum.Add(sum, treasuryAmt)

	if sum.Cmp(totalPayment) != 0 {
		t.Errorf("distribution sum %s does not match total payment %s", sum.String(), totalPayment.String())
	}

	// Verify research 5-way split sums to research share
	researchSplitSum := new(big.Int)
	r := new(big.Int)
	r.SetString(distribution.ResearchSplit.Research, 10)
	researchSplitSum.Add(researchSplitSum, r)
	f := new(big.Int)
	f.SetString(distribution.ResearchSplit.Foundation, 10)
	researchSplitSum.Add(researchSplitSum, f)
	c := new(big.Int)
	c.SetString(distribution.ResearchSplit.Community, 10)
	researchSplitSum.Add(researchSplitSum, c)
	d := new(big.Int)
	d.SetString(distribution.ResearchSplit.Developers, 10)
	researchSplitSum.Add(researchSplitSum, d)
	rv := new(big.Int)
	rv.SetString(distribution.ResearchSplit.Reserve, 10)
	researchSplitSum.Add(researchSplitSum, rv)

	if researchSplitSum.Cmp(researchShare) != 0 {
		t.Errorf("research split sum %s does not match research share %s", researchSplitSum.String(), researchShare.String())
	}
}

// -----------------------------------------------------------------------
// Tests: RegisterProvider Handler
// -----------------------------------------------------------------------

func TestRegisterProvider(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)

	// Fund the account
	bk.setBalance(accAddr, "ulgm", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		Domains:     []string{"mathematics", "physics"},
		StakeAmount: "100000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify provider exists
	provider, found := k.GetProvider(ctx, addr)
	if !found {
		t.Fatal("expected provider to be found")
	}
	if !provider.Active {
		t.Error("expected provider to be active")
	}
	if provider.StakeAmount != "100000000" {
		t.Errorf("expected stake 100000000, got %s", provider.StakeAmount)
	}

	// Verify domain index
	mathProviders := k.GetProvidersByDomain(ctx, "mathematics")
	if len(mathProviders) != 1 {
		t.Errorf("expected 1 math provider, got %d", len(mathProviders))
	}
}

func TestRegisterProviderInsufficientStake(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "ulgm", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		Domains:     []string{"mathematics"},
		StakeAmount: "1000", // Way below minimum
	})
	if err == nil {
		t.Error("expected insufficient stake error")
	}
}

func TestRegisterProviderDuplicate(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "ulgm", sdkmath.NewInt(500000000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		Domains:     []string{"mathematics"},
		StakeAmount: "100000000",
	})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	_, err = srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		Domains:     []string{"physics"},
		StakeAmount: "100000000",
	})
	if err == nil {
		t.Error("expected duplicate provider error")
	}
}

func TestDeregisterProvider(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	addr := testAddr(1)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "ulgm", sdkmath.NewInt(200000000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Sender:      addr,
		Domains:     []string{"mathematics"},
		StakeAmount: "100000000",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	_, err = srv.DeregisterProvider(ctx, &types.MsgDeregisterProvider{
		Sender: addr,
	})
	if err != nil {
		t.Fatalf("deregistration failed: %v", err)
	}

	// Provider should be gone
	_, found := k.GetProvider(ctx, addr)
	if found {
		t.Error("expected provider to be deleted")
	}

	// Domain index should be clean
	mathProviders := k.GetProvidersByDomain(ctx, "mathematics")
	if len(mathProviders) != 0 {
		t.Errorf("expected 0 math providers after deregister, got %d", len(mathProviders))
	}
}

// -----------------------------------------------------------------------
// Tests: Pricing
// -----------------------------------------------------------------------

func TestPricingBasic(t *testing.T) {
	k, ctx, _, kk := setupKeeper(t)

	// Simple fact: 50% confidence, 0 citations, outside freshness window
	kk.addFact("fact-basic", 500000, 0, 1, testAddr(10))

	// Query at block 2000: age = 1999, window = 1000 → outside window, no discount
	totalPrice, breakdown := k.CalculateQueryPrice(ctx, []string{"fact-basic"}, 2000)

	if totalPrice.Sign() <= 0 {
		t.Error("expected positive total price")
	}
	if len(breakdown) != 1 {
		t.Fatalf("expected 1 breakdown entry, got %d", len(breakdown))
	}

	// Should be exactly base cost (50% = threshold, no premium; 0 citations, no bonus; old, no discount)
	baseCost := new(big.Int)
	baseCost.SetString("1000000", 10)
	if totalPrice.Cmp(baseCost) != 0 {
		t.Errorf("expected base cost %s, got %s", baseCost.String(), totalPrice.String())
	}
}

func TestPricingHighConfidence(t *testing.T) {
	k, ctx, _, kk := setupKeeper(t)

	// Very high confidence (>850k): already in LLM training data → discount
	kk.addFact("fact-hi", 900000, 0, 1, testAddr(10))

	totalPrice, _ := k.CalculateQueryPrice(ctx, []string{"fact-hi"}, 2000)

	// Bell curve: conf 900k > 850k → discount (base - adj)
	baseCost := new(big.Int)
	baseCost.SetString("1000000", 10)
	adj := new(big.Int).Mul(baseCost, big.NewInt(200000))
	adj.Div(adj, big.NewInt(1000000))
	expected := new(big.Int).Sub(baseCost, adj)

	if totalPrice.Cmp(expected) != 0 {
		t.Errorf("expected %s with high-confidence discount, got %s", expected.String(), totalPrice.String())
	}
}

func TestPricingPeakValueZone(t *testing.T) {
	k, ctx, _, kk := setupKeeper(t)

	// Medium-high confidence (600k): peak value zone → base rate, no adjustment
	kk.addFact("fact-mid", 600000, 0, 1, testAddr(10))

	totalPrice, _ := k.CalculateQueryPrice(ctx, []string{"fact-mid"}, 2000)

	baseCost := new(big.Int)
	baseCost.SetString("1000000", 10)

	if totalPrice.Cmp(baseCost) != 0 {
		t.Errorf("expected base cost %s in peak value zone, got %s", baseCost.String(), totalPrice.String())
	}
}

func TestPricingLowConfidence(t *testing.T) {
	k, ctx, _, kk := setupKeeper(t)

	// Low confidence (200k < 500k threshold): surcharge
	kk.addFact("fact-lo", 200000, 0, 1, testAddr(10))

	totalPrice, _ := k.CalculateQueryPrice(ctx, []string{"fact-lo"}, 2000)

	baseCost := new(big.Int)
	baseCost.SetString("1000000", 10)
	adj := new(big.Int).Mul(baseCost, big.NewInt(200000))
	adj.Div(adj, big.NewInt(1000000))
	expected := new(big.Int).Add(baseCost, adj)

	if totalPrice.Cmp(expected) != 0 {
		t.Errorf("expected %s with low-confidence surcharge, got %s", expected.String(), totalPrice.String())
	}
}

func TestPricingFreshness(t *testing.T) {
	k, ctx, _, kk := setupKeeper(t)

	// Fresh fact: created at block 50, current block 100, window 1000
	kk.addFact("fact-fresh", 600000, 0, 50, testAddr(10))

	totalPrice, _ := k.CalculateQueryPrice(ctx, []string{"fact-fresh"}, 100)

	// Freshness is now a PREMIUM (recent = novel = more valuable)
	// premium = base * 100000 / 1000000 = base * 0.1
	baseCost := new(big.Int)
	baseCost.SetString("1000000", 10)
	premium := new(big.Int).Mul(baseCost, big.NewInt(100000))
	premium.Div(premium, big.NewInt(1000000))
	expected := new(big.Int).Add(baseCost, premium)

	if totalPrice.Cmp(expected) != 0 {
		t.Errorf("expected %s with freshness premium, got %s", expected.String(), totalPrice.String())
	}
}

func TestPricingNovelty(t *testing.T) {
	k, ctx, _, kk := setupKeeper(t)

	// Highly cited fact: 100 citations, log2(101) ~= 6, outside freshness window
	// At threshold confidence (500k) → no confidence adjustment
	kk.addFact("fact-cited", 600000, 100, 1, testAddr(10))

	totalPrice, _ := k.CalculateQueryPrice(ctx, []string{"fact-cited"}, 2000)

	baseCost := new(big.Int)
	baseCost.SetString("1000000", 10)

	// Novelty DISCOUNT: highly cited = well-known = cheaper
	// discount = baseCost * 6 / 10, capped at 50% of base
	discount := new(big.Int).Mul(baseCost, big.NewInt(6))
	discount.Div(discount, big.NewInt(10))
	// Cap at 50%
	halfBase := new(big.Int).Div(baseCost, big.NewInt(2))
	if discount.Cmp(halfBase) > 0 {
		discount.Set(halfBase)
	}
	expected := new(big.Int).Sub(baseCost, discount)

	if totalPrice.Cmp(expected) != 0 {
		t.Errorf("expected %s with novelty discount, got %s", expected.String(), totalPrice.String())
	}
}

// -----------------------------------------------------------------------
// Tests: Queries
// -----------------------------------------------------------------------

func TestQueryParams(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.BaseCostPerFact != "1000000" {
		t.Errorf("expected 1000000, got %s", resp.Params.BaseCostPerFact)
	}
}

func TestQueryProvider(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	addr := testAddr(1)
	k.SetProvider(ctx, types.Provider{
		Address:      addr,
		Domains:      []string{"test"},
		StakeAmount:  "100000000",
		Active:       true,
		TotalRevenue: "0",
	})

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Provider(ctx, &types.QueryProviderRequest{Address: addr})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider.StakeAmount != "100000000" {
		t.Errorf("expected stake 100000000, got %s", resp.Provider.StakeAmount)
	}
}

func TestQueryProviderNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.Provider(ctx, &types.QueryProviderRequest{Address: testAddr(99)})
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestQueryQuote(t *testing.T) {
	k, ctx, _, kk := setupKeeper(t)
	kk.addFact("fact-1", 500000, 10, 50, testAddr(10))

	quote := k.GenerateQuote(ctx, testAddr(1), testAddr(2), []string{"fact-1"}, 100)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Quote(ctx, &types.QueryQuoteRequest{QuoteId: quote.QuoteId})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Quote.Provider != testAddr(1) {
		t.Errorf("expected provider %s, got %s", testAddr(1), resp.Quote.Provider)
	}
}

func TestQueryQuoteNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.Quote(ctx, &types.QueryQuoteRequest{QuoteId: 999})
	if err == nil {
		t.Error("expected not found error")
	}
}

// -----------------------------------------------------------------------
// Tests: UpdateParams
// -----------------------------------------------------------------------

func TestUpdateParams(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	authority := k.GetAuthority()

	srv := keeper.NewMsgServerImpl(k)
	newParams := types.Params{
		BaseCostPerFact:       "5000000",
		ConfidenceThreshold:   700000,
		ConfidencePremiumBps:  150000,
		FreshnessWindowBlocks: 2000,
		FreshnessDiscountBps:  50000,
		QuoteValidityBlocks:   50,
		MinProviderStake:      "200000000",
		ResearchShareBps:      300000,
		ResearchMinBps:        60000,
		ResearchRampBlocks:    100000,
	}
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    newParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := k.GetParams(ctx)
	if got.BaseCostPerFact != "5000000" {
		t.Errorf("expected 5000000, got %s", got.BaseCostPerFact)
	}
	if got.ResearchShareBps != 300000 {
		t.Errorf("expected 300000, got %d", got.ResearchShareBps)
	}
}

func TestUpdateParamsUnauthorized(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr(99),
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Error("expected unauthorized error")
	}
}

// -----------------------------------------------------------------------
// Tests: Genesis
// -----------------------------------------------------------------------

func TestGenesis(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	addr1 := testAddr(1)
	addr2 := testAddr(2)
	k.SetProvider(ctx, types.Provider{
		Address: addr1, Domains: []string{"math"}, StakeAmount: "100000000",
		Active: true, TotalRevenue: "0",
	})
	k.SetProvider(ctx, types.Provider{
		Address: addr2, Domains: []string{"physics"}, StakeAmount: "200000000",
		Active: true, TotalRevenue: "0",
	})

	// Export
	genState := k.ExportGenesis(ctx)
	if len(genState.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(genState.Providers))
	}

	// New keeper, import
	k2, ctx2, _, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, *genState)

	got := k2.ExportGenesis(ctx2)
	if len(got.Providers) != 2 {
		t.Errorf("expected 2 providers after import, got %d", len(got.Providers))
	}
}

func TestGenesisValidation(t *testing.T) {
	valid := types.DefaultGenesis()
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Duplicate provider
	dup := types.DefaultGenesis()
	dup.Providers = []*types.Provider{
		{Address: "lgm1abc"}, {Address: "lgm1abc"},
	}
	if err := dup.Validate(); err == nil {
		t.Error("expected duplicate provider error")
	}
}

// -----------------------------------------------------------------------
// Tests: Progressive Research Share Ramp
// -----------------------------------------------------------------------

func TestGetEffectiveResearchShareBps(t *testing.T) {
	tests := []struct {
		name       string
		height     int64
		maxBps     uint64
		minBps     uint64
		rampBlocks uint64
		expected   uint64
	}{
		{
			name: "no ramp (rampBlocks=0) returns max immediately",
			height: 0, maxBps: 220000, minBps: 50000, rampBlocks: 0,
			expected: 220000,
		},
		{
			name: "block 0 with ramp returns min",
			height: 0, maxBps: 220000, minBps: 50000, rampBlocks: 1000,
			expected: 50000,
		},
		{
			name: "midpoint returns halfway between min and max",
			height: 500, maxBps: 220000, minBps: 20000, rampBlocks: 1000,
			expected: 120000, // 20000 + (200000 * 500 / 1000) = 120000
		},
		{
			name: "at ramp boundary returns max",
			height: 1000, maxBps: 220000, minBps: 50000, rampBlocks: 1000,
			expected: 220000,
		},
		{
			name: "past ramp returns max",
			height: 5000, maxBps: 220000, minBps: 50000, rampBlocks: 1000,
			expected: 220000,
		},
		{
			name: "quarter ramp",
			height: 250, maxBps: 220000, minBps: 20000, rampBlocks: 1000,
			expected: 70000, // 20000 + (200000 * 250 / 1000) = 70000
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := types.Params{
				ResearchShareBps:   tc.maxBps,
				ResearchMinBps:     tc.minBps,
				ResearchRampBlocks: tc.rampBlocks,
			}
			got := keeper.GetEffectiveResearchShareBps(tc.height, params)
			if got != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, got)
			}
		})
	}
}
