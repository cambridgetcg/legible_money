package keeper_test

import (
	"context"
	"fmt"
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

	"github.com/legible-money/legible/x/evidence_mgmt/keeper"
	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

// ---------- Mock Bank Keeper ----------

type mockBankKeeper struct {
	balances map[string]sdkmath.Int // address -> balance
	modules  map[string]sdkmath.Int // module -> balance
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]sdkmath.Int),
		modules:  make(map[string]sdkmath.Int),
	}
}

func (m *mockBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	from := fromAddr.String()
	to := toAddr.String()
	amount := amt.AmountOf("ulgm")
	balance, ok := m.balances[from]
	if !ok {
		return fmt.Errorf("insufficient funds")
	}
	if balance.LT(amount) {
		return fmt.Errorf("insufficient funds: %s < %s", balance, amount)
	}
	m.balances[from] = balance.Sub(amount)
	toBal, ok := m.balances[to]
	if !ok {
		toBal = sdkmath.ZeroInt()
	}
	m.balances[to] = toBal.Add(amount)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	addr := senderAddr.String()
	balance, ok := m.balances[addr]
	if !ok {
		return fmt.Errorf("insufficient funds")
	}
	amount := amt.AmountOf("ulgm")
	if balance.LT(amount) {
		return fmt.Errorf("insufficient funds: %s < %s", balance, amount)
	}
	m.balances[addr] = balance.Sub(amount)
	modBal, ok := m.modules[recipientModule]
	if !ok {
		modBal = sdkmath.ZeroInt()
	}
	m.modules[recipientModule] = modBal.Add(amount)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	addr := recipientAddr.String()
	amount := amt.AmountOf("ulgm")
	modBal, ok := m.modules[senderModule]
	if !ok {
		modBal = sdkmath.ZeroInt()
	}
	if modBal.LT(amount) {
		return fmt.Errorf("insufficient module funds")
	}
	m.modules[senderModule] = modBal.Sub(amount)
	balance, ok := m.balances[addr]
	if !ok {
		balance = sdkmath.ZeroInt()
	}
	m.balances[addr] = balance.Add(amount)
	return nil
}

func (m *mockBankKeeper) fundAccount(addr string, amount sdkmath.Int) {
	m.balances[addr] = amount
}

func (m *mockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) fundModule(moduleName string, amount sdkmath.Int) {
	m.modules[moduleName] = amount
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require(t, stateStore.LoadLatestVersion() == nil, "LoadLatestVersion failed")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	ir := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(ir)

	bk := newMockBankKeeper()

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		bk,
		"lgm1authority",
	)
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx, bk
}

func require(t *testing.T, cond bool, msg string) {
	t.Helper()
	if !cond {
		t.Fatal(msg)
	}
}

func testAddr(i int) string {
	addr := sdk.AccAddress(fmt.Sprintf("addr%016d", i))
	return addr.String()
}

// ---------- Tests: Params ----------

func TestSetGetParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	params := k.GetParams(ctx)
	if params.MinAuditsForConsensus != 3 {
		t.Fatalf("expected MinAuditsForConsensus 3, got %d", params.MinAuditsForConsensus)
	}

	params.MinAuditsForConsensus = 5
	params.AuditWindow = 50000
	k.SetParams(ctx, params)

	updated := k.GetParams(ctx)
	if updated.MinAuditsForConsensus != 5 {
		t.Fatalf("expected updated MinAuditsForConsensus 5, got %d", updated.MinAuditsForConsensus)
	}
	if updated.AuditWindow != 50000 {
		t.Fatalf("expected AuditWindow 50000, got %d", updated.AuditWindow)
	}
}

// ---------- Tests: Evidence CRUD ----------

func TestSetGetEvidence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	e := types.Evidence{
		Id:           "ev-1",
		ClaimId:      "claim-1",
		Submitter:    testAddr(1),
		EvidenceType: string(types.EvidenceTypeDocument),
		ContentHash:  "hash1",
		Description:  "test evidence",
		Status:       string(types.EvidenceStatusSubmitted),
		SubmittedAt:  100,
	}

	k.SetEvidence(ctx, e)
	got, found := k.GetEvidence(ctx, "ev-1")
	if !found {
		t.Fatal("evidence not found")
	}
	if got.ClaimId != "claim-1" {
		t.Fatalf("expected claim-1, got %s", got.ClaimId)
	}
	if got.Description != "test evidence" {
		t.Fatalf("expected 'test evidence', got %s", got.Description)
	}
}

func TestDeleteEvidence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	e := types.Evidence{Id: "ev-1", ClaimId: "claim-1", Submitter: testAddr(1)}
	k.SetEvidence(ctx, e)

	k.DeleteEvidence(ctx, "ev-1")
	_, found := k.GetEvidence(ctx, "ev-1")
	if found {
		t.Fatal("evidence should be deleted")
	}
}

func TestGetEvidenceByClaimId(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for i := 0; i < 3; i++ {
		k.SetEvidence(ctx, types.Evidence{
			Id:      fmt.Sprintf("ev-%d", i),
			ClaimId: "claim-1",
		})
	}
	k.SetEvidence(ctx, types.Evidence{Id: "ev-other", ClaimId: "claim-2"})

	results := k.GetEvidenceByClaimId(ctx, "claim-1")
	if len(results) != 3 {
		t.Fatalf("expected 3 evidence for claim-1, got %d", len(results))
	}
}

func TestGetEvidenceCountForClaim(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for i := 0; i < 5; i++ {
		k.SetEvidence(ctx, types.Evidence{
			Id:      fmt.Sprintf("ev-%d", i),
			ClaimId: "claim-1",
		})
	}
	if count := k.GetEvidenceCountForClaim(ctx, "claim-1"); count != 5 {
		t.Fatalf("expected count 5, got %d", count)
	}
}

func TestIterateEvidence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for i := 0; i < 4; i++ {
		k.SetEvidence(ctx, types.Evidence{
			Id:      fmt.Sprintf("ev-%d", i),
			ClaimId: "claim-1",
		})
	}

	var count int
	k.IterateEvidence(ctx, func(e types.Evidence) bool {
		count++
		return false
	})
	if count != 4 {
		t.Fatalf("expected 4 iterations, got %d", count)
	}
}

// ---------- Tests: Audit CRUD ----------

func TestSetGetAudit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	a := types.EvidenceAudit{
		Id:         "audit-1",
		EvidenceId: "ev-1",
		Auditor:    testAddr(2),
		Stake:      "50000000",
		Score:      700000,
		Rationale:  "looks good",
		Status:     string(types.AuditStatusPending),
	}

	k.SetAudit(ctx, a)
	got, found := k.GetAudit(ctx, "audit-1")
	if !found {
		t.Fatal("audit not found")
	}
	if got.Score != 700000 {
		t.Fatalf("expected score 700000, got %d", got.Score)
	}
}

func TestDeleteAudit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	a := types.EvidenceAudit{Id: "audit-1", EvidenceId: "ev-1", Auditor: testAddr(2)}
	k.SetAudit(ctx, a)
	k.DeleteAudit(ctx, "audit-1")

	_, found := k.GetAudit(ctx, "audit-1")
	if found {
		t.Fatal("audit should be deleted")
	}
}

func TestGetAuditsForEvidence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for i := 0; i < 3; i++ {
		k.SetAudit(ctx, types.EvidenceAudit{
			Id:         fmt.Sprintf("audit-%d", i),
			EvidenceId: "ev-1",
			Auditor:    testAddr(i + 10),
		})
	}

	audits := k.GetAuditsForEvidence(ctx, "ev-1")
	if len(audits) != 3 {
		t.Fatalf("expected 3 audits, got %d", len(audits))
	}
}

// ---------- Tests: Oracle CRUD ----------

func TestSetGetOracle(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	o := types.Oracle{
		Id:       "oracle-1",
		Operator: testAddr(3),
		Name:     "TestOracle",
		Stake:    "200000000",
		IsActive: true,
	}

	k.SetOracle(ctx, o)
	got, found := k.GetOracle(ctx, "oracle-1")
	if !found {
		t.Fatal("oracle not found")
	}
	if got.Name != "TestOracle" {
		t.Fatalf("expected TestOracle, got %s", got.Name)
	}
}

func TestDeleteOracle(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetOracle(ctx, types.Oracle{Id: "oracle-1", Operator: testAddr(3), IsActive: true})
	k.SetActiveOracle(ctx, "oracle-1")
	k.DeleteOracle(ctx, "oracle-1")

	_, found := k.GetOracle(ctx, "oracle-1")
	if found {
		t.Fatal("oracle should be deleted")
	}
}

func TestGetOracleByOperator(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	addr := testAddr(4)
	k.SetOracle(ctx, types.Oracle{
		Id:       "oracle-1",
		Operator: addr,
		Name:     "OperatorOracle",
	})

	got, found := k.GetOracleByOperator(ctx, addr)
	if !found {
		t.Fatal("oracle not found by operator")
	}
	if got.Id != "oracle-1" {
		t.Fatalf("expected oracle-1, got %s", got.Id)
	}
}

func TestActiveOracles(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Add 3 oracles, 2 active, 1 inactive
	k.SetOracle(ctx, types.Oracle{Id: "o1", Operator: testAddr(1), IsActive: true})
	k.SetActiveOracle(ctx, "o1")
	k.SetOracle(ctx, types.Oracle{Id: "o2", Operator: testAddr(2), IsActive: true})
	k.SetActiveOracle(ctx, "o2")
	k.SetOracle(ctx, types.Oracle{Id: "o3", Operator: testAddr(3), IsActive: false})

	active := k.GetActiveOracles(ctx)
	if len(active) != 2 {
		t.Fatalf("expected 2 active oracles, got %d", len(active))
	}
}

// ---------- Tests: Consensus CRUD ----------

func TestSetGetConsensus(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	c := types.AuditConsensus{
		EvidenceId:    "ev-1",
		WeightedScore: 750000,
		TotalStake:    "100000000",
		AuditCount:    3,
	}
	k.SetConsensus(ctx, c)

	got, found := k.GetConsensus(ctx, "ev-1")
	if !found {
		t.Fatal("consensus not found")
	}
	if got.WeightedScore != 750000 {
		t.Fatalf("expected 750000, got %d", got.WeightedScore)
	}
}

// ---------- Tests: ID Generation ----------

func TestGenerateEvidenceID(t *testing.T) {
	id1 := keeper.GenerateEvidenceID(testAddr(1), "claim-1", "hash1", 100)
	id2 := keeper.GenerateEvidenceID(testAddr(1), "claim-1", "hash2", 100)
	id3 := keeper.GenerateEvidenceID(testAddr(1), "claim-1", "hash1", 100)

	if id1 == id2 {
		t.Fatal("different content hashes should produce different IDs")
	}
	if id1 != id3 {
		t.Fatal("same inputs should produce same ID")
	}
	if len(id1) != 32 {
		t.Fatalf("expected 32 char ID, got %d", len(id1))
	}
}

func TestGenerateAuditID(t *testing.T) {
	id1 := keeper.GenerateAuditID(testAddr(1), "ev-1", 100)
	id2 := keeper.GenerateAuditID(testAddr(2), "ev-1", 100)
	if id1 == id2 {
		t.Fatal("different auditors should produce different IDs")
	}
}

func TestGenerateOracleID(t *testing.T) {
	id1 := keeper.GenerateOracleID(testAddr(1), "Oracle1", 100)
	id2 := keeper.GenerateOracleID(testAddr(1), "Oracle2", 100)
	if id1 == id2 {
		t.Fatal("different names should produce different IDs")
	}
}

// ---------- Tests: SubmitEvidence ----------

func TestSubmitEvidence_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	e := types.Evidence{
		Id:           "ev-1",
		ClaimId:      "claim-1",
		Submitter:    testAddr(1),
		EvidenceType: string(types.EvidenceTypeDocument),
		ContentHash:  "hash1",
		Description:  "test doc",
		SubmittedAt:  100,
	}

	if err := k.SubmitEvidence(ctx, e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, found := k.GetEvidence(ctx, "ev-1")
	if !found {
		t.Fatal("evidence not found after submission")
	}
	if got.Status != string(types.EvidenceStatusSubmitted) {
		t.Fatalf("expected submitted status, got %s", got.Status)
	}
	if got.AuditCount != 0 {
		t.Fatalf("expected audit count 0, got %d", got.AuditCount)
	}
}

func TestSubmitEvidence_MaxReached(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Set max to 2 for testing
	params := k.GetParams(ctx)
	params.MaxEvidencePerClaim = 2
	k.SetParams(ctx, params)

	for i := 0; i < 2; i++ {
		e := types.Evidence{
			Id:           fmt.Sprintf("ev-%d", i),
			ClaimId:      "claim-1",
			Submitter:    testAddr(1),
			EvidenceType: string(types.EvidenceTypeDocument),
			ContentHash:  fmt.Sprintf("hash%d", i),
			Description:  "test",
			SubmittedAt:  100,
		}
		if err := k.SubmitEvidence(ctx, e); err != nil {
			t.Fatalf("unexpected error on evidence %d: %v", i, err)
		}
	}

	// Third should fail
	e := types.Evidence{
		Id:           "ev-2",
		ClaimId:      "claim-1",
		EvidenceType: string(types.EvidenceTypeDocument),
		ContentHash:  "hash2",
		Description:  "test",
	}
	if err := k.SubmitEvidence(ctx, e); err == nil {
		t.Fatal("expected max evidence error")
	}
}

func TestSubmitEvidence_DuplicateHash(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	e := types.Evidence{
		Id:           "ev-1",
		ClaimId:      "claim-1",
		EvidenceType: string(types.EvidenceTypeDocument),
		ContentHash:  "samehash",
		Description:  "first",
		SubmittedAt:  100,
	}
	if err := k.SubmitEvidence(ctx, e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	e2 := types.Evidence{
		Id:           "ev-2",
		ClaimId:      "claim-1",
		EvidenceType: string(types.EvidenceTypeAttestation),
		ContentHash:  "samehash",
		Description:  "second",
	}
	if err := k.SubmitEvidence(ctx, e2); err == nil {
		t.Fatal("expected duplicate hash error")
	}
}

func TestSubmitEvidence_InvalidType(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	e := types.Evidence{
		Id:           "ev-1",
		ClaimId:      "claim-1",
		EvidenceType: "invalid_type",
		ContentHash:  "hash1",
		Description:  "test",
	}
	if err := k.SubmitEvidence(ctx, e); err == nil {
		t.Fatal("expected invalid type error")
	}
}

// ---------- Tests: SubmitAudit ----------

func TestSubmitAudit_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Set up evidence in auditable state
	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		ClaimId:     "claim-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 50, // Within audit window (50 + 34272 > 100)
	})

	auditor := testAddr(1)
	bk.fundAccount(auditor, sdkmath.NewInt(100000000))

	audit := types.EvidenceAudit{
		Id:          "audit-1",
		EvidenceId:  "ev-1",
		Auditor:     auditor,
		Stake:       "22000000",
		Score:       750000,
		Rationale:   "solid evidence",
		SubmittedAt: 100,
	}

	if err := k.SubmitAudit(ctx, audit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, found := k.GetAudit(ctx, "audit-1")
	if !found {
		t.Fatal("audit not found")
	}
	if got.Status != string(types.AuditStatusPending) {
		t.Fatalf("expected pending, got %s", got.Status)
	}
}

func TestSubmitAudit_EvidenceNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	audit := types.EvidenceAudit{
		Id:         "audit-1",
		EvidenceId: "nonexistent",
		Auditor:    testAddr(1),
		Stake:      "22000000",
		Score:      500000,
		Rationale:  "test",
	}
	if err := k.SubmitAudit(ctx, audit); err == nil {
		t.Fatal("expected evidence not found error")
	}
}

func TestSubmitAudit_WindowClosed(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Evidence submitted long ago — audit window has closed
	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		ClaimId:     "claim-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 1, // 1 + 34272 < 100... wait, that's > 100. Need to make sure ctx height > submitted + window.
	})

	// Override params to have small window
	params := k.GetParams(ctx)
	params.AuditWindow = 10
	k.SetParams(ctx, params)

	// Now at height 100, evidence submitted at 1, window=10 → window closed at 11
	auditor := testAddr(1)
	bk.fundAccount(auditor, sdkmath.NewInt(100000000))

	audit := types.EvidenceAudit{
		Id:         "audit-1",
		EvidenceId: "ev-1",
		Auditor:    auditor,
		Stake:      "22000000",
		Score:      500000,
		Rationale:  "late",
	}
	if err := k.SubmitAudit(ctx, audit); err == nil {
		t.Fatal("expected audit window closed error")
	}
}

func TestSubmitAudit_InsufficientStake(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 50,
	})

	auditor := testAddr(1)
	bk.fundAccount(auditor, sdkmath.NewInt(100000000))

	audit := types.EvidenceAudit{
		Id:         "audit-1",
		EvidenceId: "ev-1",
		Auditor:    auditor,
		Stake:      "1000000", // Below minimum 22000000
		Score:      500000,
		Rationale:  "test",
	}
	if err := k.SubmitAudit(ctx, audit); err == nil {
		t.Fatal("expected insufficient stake error")
	}
}

func TestSubmitAudit_DuplicateAuditor(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 50,
	})

	auditor := testAddr(1)
	bk.fundAccount(auditor, sdkmath.NewInt(500000000))

	audit1 := types.EvidenceAudit{
		Id:         "audit-1",
		EvidenceId: "ev-1",
		Auditor:    auditor,
		Stake:      "22000000",
		Score:      700000,
		Rationale:  "first",
	}
	if err := k.SubmitAudit(ctx, audit1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	audit2 := types.EvidenceAudit{
		Id:         "audit-2",
		EvidenceId: "ev-1",
		Auditor:    auditor,
		Stake:      "22000000",
		Score:      800000,
		Rationale:  "duplicate",
	}
	if err := k.SubmitAudit(ctx, audit2); err == nil {
		t.Fatal("expected duplicate audit error")
	}
}

func TestSubmitAudit_InvalidScore(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 50,
	})

	auditor := testAddr(1)
	bk.fundAccount(auditor, sdkmath.NewInt(100000000))

	audit := types.EvidenceAudit{
		Id:         "audit-1",
		EvidenceId: "ev-1",
		Auditor:    auditor,
		Stake:      "22000000",
		Score:      1000001, // Exceeds max
		Rationale:  "test",
	}
	if err := k.SubmitAudit(ctx, audit); err == nil {
		t.Fatal("expected invalid score error")
	}
}

// ---------- Tests: ComputeConsensus ----------

func TestComputeConsensus(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Set up 3 audits with different stakes and scores
	audits := []types.EvidenceAudit{
		{Id: "a1", EvidenceId: "ev-1", Stake: "100000000", Score: 800000, Status: string(types.AuditStatusPending)},
		{Id: "a2", EvidenceId: "ev-1", Stake: "200000000", Score: 600000, Status: string(types.AuditStatusPending)},
		{Id: "a3", EvidenceId: "ev-1", Stake: "100000000", Score: 900000, Status: string(types.AuditStatusPending)},
	}
	for _, a := range audits {
		k.SetAudit(ctx, a)
	}

	consensus, err := k.ComputeConsensus(ctx, "ev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: (100M*800K + 200M*600K + 100M*900K) / 400M
	// = (80T + 120T + 90T) / 400M = 290T / 400M = 725000
	if consensus.WeightedScore != 725000 {
		t.Fatalf("expected weighted score 725000, got %d", consensus.WeightedScore)
	}
	if consensus.AuditCount != 3 {
		t.Fatalf("expected audit count 3, got %d", consensus.AuditCount)
	}
}

func TestComputeConsensus_SkipsExpired(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetAudit(ctx, types.EvidenceAudit{Id: "a1", EvidenceId: "ev-1", Stake: "100000000", Score: 800000, Status: string(types.AuditStatusPending)})
	k.SetAudit(ctx, types.EvidenceAudit{Id: "a2", EvidenceId: "ev-1", Stake: "100000000", Score: 200000, Status: string(types.AuditStatusExpired)})

	consensus, err := k.ComputeConsensus(ctx, "ev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consensus.WeightedScore != 800000 {
		t.Fatalf("expected 800000 (expired audit skipped), got %d", consensus.WeightedScore)
	}
	if consensus.AuditCount != 1 {
		t.Fatalf("expected audit count 1, got %d", consensus.AuditCount)
	}
}

// ---------- Tests: IsAccurate ----------

func TestIsAccurate(t *testing.T) {
	// Within threshold
	if !keeper.IsAccurate(700000, 750000, 100000) {
		t.Fatal("score 700000 should be accurate (threshold 100000 of 750000)")
	}

	// Exactly at threshold
	if !keeper.IsAccurate(700000, 800000, 100000) {
		t.Fatal("score 700000 should be accurate (exactly at threshold of 800000)")
	}

	// Outside threshold
	if keeper.IsAccurate(500000, 800000, 100000) {
		t.Fatal("score 500000 should NOT be accurate (300000 > threshold 100000)")
	}
}

// ---------- Tests: ResolveAudits ----------

func TestResolveAudits(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Fund the module to pay rewards
	bk.fundModule(types.ModuleName, sdkmath.NewInt(1000000000))

	auditor1 := testAddr(1)
	auditor2 := testAddr(2)

	// Accurate audit (close to ground truth)
	k.SetAudit(ctx, types.EvidenceAudit{
		Id:         "a1",
		EvidenceId: "ev-1",
		Auditor:    auditor1,
		Stake:      "100000000",
		Score:      720000,
		Status:     string(types.AuditStatusPending),
	})

	// Inaccurate audit (far from ground truth)
	k.SetAudit(ctx, types.EvidenceAudit{
		Id:         "a2",
		EvidenceId: "ev-1",
		Auditor:    auditor2,
		Stake:      "100000000",
		Score:      200000,
		Status:     string(types.AuditStatusPending),
	})

	groundTruth := uint64(750000)
	err := k.ResolveAudits(ctx, "ev-1", groundTruth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check auditor 1 was rewarded (delta=30000, threshold=100000 → accurate)
	a1, _ := k.GetAudit(ctx, "a1")
	if a1.Status != string(types.AuditStatusRewarded) {
		t.Fatalf("expected rewarded, got %s", a1.Status)
	}

	// Check auditor 2 was slashed (delta=550000, threshold=100000 → inaccurate)
	a2, _ := k.GetAudit(ctx, "a2")
	if a2.Status != string(types.AuditStatusSlashed) {
		t.Fatalf("expected slashed, got %s", a2.Status)
	}
}

// ---------- Tests: ResolveEvidence ----------

func TestResolveEvidence_Verified(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	bk.fundModule(types.ModuleName, sdkmath.NewInt(10000000000))

	// Set audit window to 10 blocks
	params := k.GetParams(ctx)
	params.AuditWindow = 10
	k.SetParams(ctx, params)

	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		ClaimId:     "claim-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 80, // 80 + 10 = 90 < 100 (current height) → window closed
	})

	// Add 3 audits (min required) with high scores → verified
	for i := 0; i < 3; i++ {
		k.SetAudit(ctx, types.EvidenceAudit{
			Id:         fmt.Sprintf("a%d", i),
			EvidenceId: "ev-1",
			Auditor:    testAddr(i + 10),
			Stake:      "50000000",
			Score:      800000,
			Status:     string(types.AuditStatusPending),
		})
	}

	if err := k.ResolveEvidence(ctx, "ev-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := k.GetEvidence(ctx, "ev-1")
	if got.Status != string(types.EvidenceStatusVerified) {
		t.Fatalf("expected verified, got %s", got.Status)
	}
	if got.VerifiedAt != 100 {
		t.Fatalf("expected verifiedAt=100, got %d", got.VerifiedAt)
	}
}

func TestResolveEvidence_Rejected(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	bk.fundModule(types.ModuleName, sdkmath.NewInt(10000000000))

	params := k.GetParams(ctx)
	params.AuditWindow = 10
	k.SetParams(ctx, params)

	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 80,
	})

	// 3 audits with low scores → rejected
	for i := 0; i < 3; i++ {
		k.SetAudit(ctx, types.EvidenceAudit{
			Id:         fmt.Sprintf("a%d", i),
			EvidenceId: "ev-1",
			Auditor:    testAddr(i + 10),
			Stake:      "50000000",
			Score:      200000, // Below 50% threshold
			Status:     string(types.AuditStatusPending),
		})
	}

	if err := k.ResolveEvidence(ctx, "ev-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := k.GetEvidence(ctx, "ev-1")
	if got.Status != string(types.EvidenceStatusRejected) {
		t.Fatalf("expected rejected, got %s", got.Status)
	}
}

func TestResolveEvidence_Expired_InsufficientAudits(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	params := k.GetParams(ctx)
	params.AuditWindow = 10
	k.SetParams(ctx, params)

	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 80,
	})

	// Only 1 audit, but min is 3
	k.SetAudit(ctx, types.EvidenceAudit{
		Id:         "a1",
		EvidenceId: "ev-1",
		Stake:      "50000000",
		Score:      800000,
		Status:     string(types.AuditStatusPending),
	})

	if err := k.ResolveEvidence(ctx, "ev-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := k.GetEvidence(ctx, "ev-1")
	if got.Status != string(types.EvidenceStatusExpired) {
		t.Fatalf("expected expired, got %s", got.Status)
	}
}

func TestResolveEvidence_WindowStillOpen(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Evidence submitted recently — window still open (default 34272 blocks)
	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 95, // 95 + 34272 >> 100
	})

	if err := k.ResolveEvidence(ctx, "ev-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should remain submitted (window not closed yet)
	got, _ := k.GetEvidence(ctx, "ev-1")
	if got.Status != string(types.EvidenceStatusSubmitted) {
		t.Fatalf("expected still submitted, got %s", got.Status)
	}
}

// ---------- Tests: ComputeReliability ----------

func TestComputeReliability(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetEvidence(ctx, types.Evidence{
		Id:     "ev-1",
		Status: string(types.EvidenceStatusSubmitted),
	})

	// Add 2 audits
	k.SetAudit(ctx, types.EvidenceAudit{
		Id:         "a1",
		EvidenceId: "ev-1",
		Stake:      "100000000",
		Score:      800000,
		Status:     string(types.AuditStatusPending),
	})
	k.SetAudit(ctx, types.EvidenceAudit{
		Id:         "a2",
		EvidenceId: "ev-1",
		Stake:      "100000000",
		Score:      600000,
		Status:     string(types.AuditStatusPending),
	})

	reliability, err := k.ComputeReliability(ctx, "ev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Equal stakes → simple average: (800000 + 600000) / 2 = 700000
	if reliability != 700000 {
		t.Fatalf("expected 700000, got %d", reliability)
	}
}

// ---------- Tests: Oracle Registration ----------

func TestRegisterOracle_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	operator := testAddr(1)
	bk.fundAccount(operator, sdkmath.NewInt(500000000))

	oracle := types.Oracle{
		Id:        "oracle-1",
		Operator:  operator,
		Name:      "TestOracle",
		DataTypes: []string{string(types.EvidenceTypeOracleFeed)},
		Stake:     "100000000",
	}

	if err := k.RegisterOracle(ctx, oracle); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, found := k.GetOracle(ctx, "oracle-1")
	if !found {
		t.Fatal("oracle not found")
	}
	if !got.IsActive {
		t.Fatal("oracle should be active")
	}
	if got.ReputationScore != 500000 {
		t.Fatalf("expected initial reputation 500000, got %d", got.ReputationScore)
	}
}

func TestRegisterOracle_DuplicateOperator(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	operator := testAddr(1)
	bk.fundAccount(operator, sdkmath.NewInt(500000000))

	oracle1 := types.Oracle{
		Id:        "oracle-1",
		Operator:  operator,
		Name:      "First",
		DataTypes: []string{string(types.EvidenceTypeOracleFeed)},
		Stake:     "100000000",
	}
	if err := k.RegisterOracle(ctx, oracle1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oracle2 := types.Oracle{
		Id:        "oracle-2",
		Operator:  operator,
		Name:      "Second",
		DataTypes: []string{string(types.EvidenceTypeComputation)},
		Stake:     "100000000",
	}
	if err := k.RegisterOracle(ctx, oracle2); err == nil {
		t.Fatal("expected duplicate operator error")
	}
}

func TestRegisterOracle_InsufficientStake(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	operator := testAddr(1)
	bk.fundAccount(operator, sdkmath.NewInt(500000000))

	oracle := types.Oracle{
		Id:        "oracle-1",
		Operator:  operator,
		Name:      "Cheap",
		DataTypes: []string{string(types.EvidenceTypeOracleFeed)},
		Stake:     "50000000", // Below 100000000 minimum
	}
	if err := k.RegisterOracle(ctx, oracle); err == nil {
		t.Fatal("expected insufficient oracle stake error")
	}
}

// ---------- Tests: Oracle Reputation ----------

func TestRecordOracleSubmission(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetOracle(ctx, types.Oracle{
		Id:              "oracle-1",
		Operator:        testAddr(1),
		IsActive:        true,
		ReputationScore: 500000,
		DataPoints:      0,
	})

	// Record accurate submission
	if err := k.RecordOracleSubmission(ctx, "oracle-1", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := k.GetOracle(ctx, "oracle-1")
	if got.DataPoints != 1 {
		t.Fatalf("expected 1 data point, got %d", got.DataPoints)
	}
	if got.AccuracyRate != 1000000 {
		t.Fatalf("expected accuracy 1000000, got %d", got.AccuracyRate)
	}

	// Record inaccurate submission
	if err := k.RecordOracleSubmission(ctx, "oracle-1", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ = k.GetOracle(ctx, "oracle-1")
	if got.DataPoints != 2 {
		t.Fatalf("expected 2 data points, got %d", got.DataPoints)
	}
	// Moving avg: (1000000 * 1 + 0) / 2 = 500000
	if got.AccuracyRate != 500000 {
		t.Fatalf("expected accuracy 500000, got %d", got.AccuracyRate)
	}
}

// ---------- Tests: SlashOracle ----------

func TestSlashOracle(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetOracle(ctx, types.Oracle{
		Id:       "oracle-1",
		Operator: testAddr(1),
		Stake:    "200000000",
		IsActive: true,
	})
	k.SetActiveOracle(ctx, "oracle-1")

	if err := k.SlashOracle(ctx, "oracle-1", "50000000"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := k.GetOracle(ctx, "oracle-1")
	if got.Stake != "150000000" {
		t.Fatalf("expected stake 150000000, got %s", got.Stake)
	}
	if !got.IsActive {
		t.Fatal("oracle should still be active (above min stake)")
	}
}

func TestSlashOracle_Deactivated(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetOracle(ctx, types.Oracle{
		Id:       "oracle-1",
		Operator: testAddr(1),
		Stake:    "100000000", // Exactly at minimum
		IsActive: true,
	})
	k.SetActiveOracle(ctx, "oracle-1")

	// Slash below minimum → should deactivate
	if err := k.SlashOracle(ctx, "oracle-1", "50000000"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := k.GetOracle(ctx, "oracle-1")
	if got.IsActive {
		t.Fatal("oracle should be deactivated after slash below minimum")
	}
}

// ---------- Tests: ExpireStaleEvidence ----------

func TestExpireStaleEvidence(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	bk.fundModule(types.ModuleName, sdkmath.NewInt(10000000000))

	params := k.GetParams(ctx)
	params.AuditWindow = 10
	k.SetParams(ctx, params)

	// Evidence 1: stale (past window), no audits → expired
	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 80,
	})

	// Evidence 2: fresh (within window) → stays submitted
	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-2",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 95,
	})

	k.ExpireStaleEvidence(ctx)

	e1, _ := k.GetEvidence(ctx, "ev-1")
	if e1.Status != string(types.EvidenceStatusExpired) {
		t.Fatalf("expected ev-1 expired, got %s", e1.Status)
	}

	e2, _ := k.GetEvidence(ctx, "ev-2")
	if e2.Status != string(types.EvidenceStatusSubmitted) {
		t.Fatalf("expected ev-2 still submitted, got %s", e2.Status)
	}
}

// ---------- Tests: PruneInactiveOracles ----------

func TestPruneInactiveOracles(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	params := k.GetParams(ctx)
	params.AuditWindow = 10
	k.SetParams(ctx, params)

	// Old inactive oracle with no data → should be pruned
	k.SetOracle(ctx, types.Oracle{
		Id:           "old",
		Operator:     testAddr(1),
		IsActive:     false,
		DataPoints:   0,
		RegisteredAt: 1, // 1 + 20 < 100 → eligible for pruning
	})

	// Active oracle → should not be pruned
	k.SetOracle(ctx, types.Oracle{
		Id:       "active",
		Operator: testAddr(2),
		IsActive: true,
	})

	k.PruneInactiveOracles(ctx)

	_, found := k.GetOracle(ctx, "old")
	if found {
		t.Fatal("old inactive oracle should be pruned")
	}

	_, found = k.GetOracle(ctx, "active")
	if !found {
		t.Fatal("active oracle should not be pruned")
	}
}

// ---------- Tests: UpdateEvidenceStatus ----------

func TestUpdateEvidenceStatus(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetEvidence(ctx, types.Evidence{
		Id:     "ev-1",
		Status: string(types.EvidenceStatusSubmitted),
	})

	if err := k.UpdateEvidenceStatus(ctx, "ev-1", types.EvidenceStatusVerified); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := k.GetEvidence(ctx, "ev-1")
	if got.Status != string(types.EvidenceStatusVerified) {
		t.Fatalf("expected verified, got %s", got.Status)
	}
	if got.VerifiedAt != 100 {
		t.Fatalf("expected verifiedAt=100, got %d", got.VerifiedAt)
	}
}

func TestUpdateEvidenceStatus_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	if err := k.UpdateEvidenceStatus(ctx, "nonexistent", types.EvidenceStatusVerified); err == nil {
		t.Fatal("expected not found error")
	}
}

// ---------- Tests: MsgServer ----------

func TestMsgServer_SubmitEvidence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter:    testAddr(1),
		ClaimId:      "claim-1",
		EvidenceType: string(types.EvidenceTypeDocument),
		ContentHash:  "hash1",
		Description:  "test evidence",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.EvidenceId == "" {
		t.Fatal("expected evidence ID in response")
	}
}

func TestMsgServer_AuditEvidence(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	k.SetEvidence(ctx, types.Evidence{
		Id:          "ev-1",
		Status:      string(types.EvidenceStatusSubmitted),
		SubmittedAt: 50,
	})

	auditor := testAddr(1)
	bk.fundAccount(auditor, sdkmath.NewInt(100000000))

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.AuditEvidence(ctx, &types.MsgAuditEvidence{
		Auditor:    auditor,
		EvidenceId: "ev-1",
		Score:      750000,
		Rationale:  "solid",
		Stake:      "22000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AuditId == "" {
		t.Fatal("expected audit ID in response")
	}
}

func TestMsgServer_RegisterOracle(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	operator := testAddr(1)
	bk.fundAccount(operator, sdkmath.NewInt(500000000))

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.RegisterOracle(ctx, &types.MsgRegisterOracle{
		Operator:    operator,
		Name:        "TestOracle",
		Description: "A test oracle",
		DataTypes:   []string{string(types.EvidenceTypeOracleFeed)},
		Stake:       "100000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OracleId == "" {
		t.Fatal("expected oracle ID in response")
	}
}

func TestMsgServer_SubmitOracleData(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetOracle(ctx, types.Oracle{
		Id:        "oracle-1",
		Operator:  testAddr(1),
		Name:      "TestOracle",
		DataTypes: []string{string(types.EvidenceTypeOracleFeed)},
		IsActive:  true,
	})

	srv := keeper.NewMsgServerImpl(k)
	resp, err := srv.SubmitOracleData(ctx, &types.MsgSubmitOracleData{
		Operator:    testAddr(1),
		OracleId:    "oracle-1",
		ClaimId:     "claim-1",
		ContentHash: "oraclehash1",
		DataType:    string(types.EvidenceTypeOracleFeed),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.EvidenceId == "" {
		t.Fatal("expected evidence ID in response")
	}
}

func TestMsgServer_UpdateParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	srv := keeper.NewMsgServerImpl(k)

	// Authorized
	newParams := types.DefaultParams()
	newParams.MinAuditsForConsensus = 5
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "lgm1authority",
		Params:    &newParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := k.GetParams(sdk.UnwrapSDKContext(ctx))
	if got.MinAuditsForConsensus != 5 {
		t.Fatalf("expected 5, got %d", got.MinAuditsForConsensus)
	}

	// Unauthorized
	badParams := types.DefaultParams()
	_, err = srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "lgm1badactor",
		Params:    &badParams,
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

// ---------- Tests: Query Handlers ----------

func TestQuery_Evidence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	k.SetEvidence(ctx, types.Evidence{Id: "ev-1", ClaimId: "claim-1", Description: "test"})

	resp, err := qs.Evidence(ctx, &types.QueryEvidenceRequest{EvidenceId: "ev-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Evidence.Description != "test" {
		t.Fatalf("expected 'test', got %s", resp.Evidence.Description)
	}
}

func TestQuery_EvidenceByClaimId(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	k.SetEvidence(ctx, types.Evidence{Id: "ev-1", ClaimId: "claim-1"})
	k.SetEvidence(ctx, types.Evidence{Id: "ev-2", ClaimId: "claim-1"})
	k.SetEvidence(ctx, types.Evidence{Id: "ev-3", ClaimId: "claim-2"})

	resp, err := qs.EvidenceByClaimId(ctx, &types.QueryEvidenceByClaimIdRequest{ClaimId: "claim-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Evidences) != 2 {
		t.Fatalf("expected 2 evidences, got %d", len(resp.Evidences))
	}
}

func TestQuery_Audit(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	k.SetAudit(ctx, types.EvidenceAudit{Id: "audit-1", EvidenceId: "ev-1", Score: 800000})

	resp, err := qs.Audit(ctx, &types.QueryAuditRequest{AuditId: "audit-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Audit.Score != 800000 {
		t.Fatalf("expected score 800000, got %d", resp.Audit.Score)
	}
}

func TestQuery_AuditsForEvidence(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for i := 0; i < 3; i++ {
		k.SetAudit(ctx, types.EvidenceAudit{
			Id:         fmt.Sprintf("audit-%d", i),
			EvidenceId: "ev-1",
		})
	}

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.AuditsForEvidence(ctx, &types.QueryAuditsForEvidenceRequest{EvidenceId: "ev-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Audits) != 3 {
		t.Fatalf("expected 3 audits, got %d", len(resp.Audits))
	}
}

func TestQuery_Oracle(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	k.SetOracle(ctx, types.Oracle{Id: "oracle-1", Name: "TestOracle"})

	resp, err := qs.Oracle(ctx, &types.QueryOracleRequest{OracleId: "oracle-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Oracle.Name != "TestOracle" {
		t.Fatalf("expected TestOracle, got %s", resp.Oracle.Name)
	}
}

func TestQuery_ActiveOracles(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetOracle(ctx, types.Oracle{Id: "o1", IsActive: true})
	k.SetActiveOracle(ctx, "o1")
	k.SetOracle(ctx, types.Oracle{Id: "o2", IsActive: true})
	k.SetActiveOracle(ctx, "o2")
	k.SetOracle(ctx, types.Oracle{Id: "o3", IsActive: false})

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.ActiveOracles(ctx, &types.QueryActiveOraclesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Oracles) != 2 {
		t.Fatalf("expected 2 active oracles, got %d", len(resp.Oracles))
	}
}

func TestQuery_Consensus(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	k.SetConsensus(ctx, types.AuditConsensus{
		EvidenceId:    "ev-1",
		WeightedScore: 750000,
		AuditCount:    3,
	})

	resp, err := qs.Consensus(ctx, &types.QueryConsensusRequest{EvidenceId: "ev-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Consensus.WeightedScore != 750000 {
		t.Fatalf("expected 750000, got %d", resp.Consensus.WeightedScore)
	}
}

func TestQuery_Params(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.MinAuditsForConsensus != 3 {
		t.Fatalf("expected 3, got %d", resp.Params.MinAuditsForConsensus)
	}
}

// ---------- Tests: Genesis ----------

func TestInitExportGenesis(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	genState := types.GenesisState{
		Params: &types.Params{
			MinAuditStake:         "50000000",
			AuditWindow:           50000,
			MinAuditsForConsensus: 5,
			AccuracyThreshold:     150000,
			SlashRate:             150000,
			RewardRate:            75000,
			MinOracleStake:        "200000000",
			OracleSlashRate:       250000,
			MaxEvidencePerClaim:   20,
		},
		EvidenceItems: []*types.Evidence{
			{Id: "ev-1", ClaimId: "claim-1", Status: string(types.EvidenceStatusSubmitted)},
		},
		Oracles: []*types.Oracle{
			{Id: "oracle-1", Operator: testAddr(1), IsActive: true},
		},
	}

	k.InitGenesis(ctx, genState)

	// Verify params
	params := k.GetParams(ctx)
	if params.MinAuditsForConsensus != 5 {
		t.Fatalf("expected 5, got %d", params.MinAuditsForConsensus)
	}

	// Verify evidence
	ev, found := k.GetEvidence(ctx, "ev-1")
	if !found {
		t.Fatal("evidence not found after init")
	}
	if ev.ClaimId != "claim-1" {
		t.Fatalf("expected claim-1, got %s", ev.ClaimId)
	}

	// Verify oracle
	o, found := k.GetOracle(ctx, "oracle-1")
	if !found {
		t.Fatal("oracle not found after init")
	}
	if !o.IsActive {
		t.Fatal("oracle should be active")
	}

	// Export and verify
	exported := k.ExportGenesis(ctx)
	if exported.Params.MinAuditsForConsensus != 5 {
		t.Fatalf("exported params mismatch")
	}
	if len(exported.EvidenceItems) != 1 {
		t.Fatalf("expected 1 evidence in export, got %d", len(exported.EvidenceItems))
	}
	if len(exported.Oracles) != 1 {
		t.Fatalf("expected 1 oracle in export, got %d", len(exported.Oracles))
	}
}

// ---------- Tests: Full Lifecycle ----------

func TestFullEvidenceLifecycle(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Set short audit window for testing
	params := k.GetParams(ctx)
	params.AuditWindow = 10
	k.SetParams(ctx, params)

	// Fund module for rewards
	bk.fundModule(types.ModuleName, sdkmath.NewInt(10000000000))

	srv := keeper.NewMsgServerImpl(k)

	// 1. Submit evidence
	submitter := testAddr(1)
	resp, err := srv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter:    submitter,
		ClaimId:      "claim-1",
		EvidenceType: string(types.EvidenceTypeExperiment),
		ContentHash:  "experiment_hash_1",
		Description:  "Experiment results for claim",
	})
	if err != nil {
		t.Fatalf("submit evidence failed: %v", err)
	}
	evidenceId := resp.EvidenceId

	// 2. Three auditors audit the evidence
	for i := 0; i < 3; i++ {
		auditor := testAddr(10 + i)
		bk.fundAccount(auditor, sdkmath.NewInt(100000000))
		_, err := srv.AuditEvidence(ctx, &types.MsgAuditEvidence{
			Auditor:    auditor,
			EvidenceId: evidenceId,
			Score:      uint64(700000 + i*50000), // 700K, 750K, 800K
			Rationale:  fmt.Sprintf("Auditor %d assessment", i),
			Stake:      "22000000",
		})
		if err != nil {
			t.Fatalf("audit %d failed: %v", i, err)
		}
	}

	// 3. Advance to after audit window
	ctx = ctx.WithBlockHeight(111) // 100 + 11 > 100 + 10

	// 4. Resolve evidence
	if err := k.ResolveEvidence(ctx, evidenceId); err != nil {
		t.Fatalf("resolve evidence failed: %v", err)
	}

	// 5. Verify outcome
	evidence, found := k.GetEvidence(ctx, evidenceId)
	if !found {
		t.Fatal("evidence not found")
	}
	if evidence.Status != string(types.EvidenceStatusVerified) {
		t.Fatalf("expected verified, got %s", evidence.Status)
	}
	if evidence.AuditCount < 3 {
		t.Fatalf("expected at least 3 audits, got %d", evidence.AuditCount)
	}

	// 6. Verify consensus was stored
	consensus, found := k.GetConsensus(ctx, evidenceId)
	if !found {
		t.Fatal("consensus not found")
	}
	if consensus.AuditCount != 3 {
		t.Fatalf("expected 3 in consensus, got %d", consensus.AuditCount)
	}

	// Weighted score should be: (22M*700K + 22M*750K + 22M*800K) / 66M = 750000
	if consensus.WeightedScore != 750000 {
		t.Fatalf("expected weighted score 750000, got %d", consensus.WeightedScore)
	}
}

// ---------- Tests: Oracle Data Submission ----------

func TestOracleDataSubmission_InactiveOracle(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetOracle(ctx, types.Oracle{
		Id:        "oracle-1",
		Operator:  testAddr(1),
		DataTypes: []string{string(types.EvidenceTypeOracleFeed)},
		IsActive:  false,
	})

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.SubmitOracleData(ctx, &types.MsgSubmitOracleData{
		OracleId:    "oracle-1",
		ClaimId:     "claim-1",
		ContentHash: "hash",
		DataType:    string(types.EvidenceTypeOracleFeed),
	})
	if err == nil {
		t.Fatal("expected inactive oracle error")
	}
}

func TestOracleDataSubmission_UnsupportedType(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetOracle(ctx, types.Oracle{
		Id:        "oracle-1",
		Operator:  testAddr(1),
		DataTypes: []string{string(types.EvidenceTypeOracleFeed)},
		IsActive:  true,
	})

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.SubmitOracleData(ctx, &types.MsgSubmitOracleData{
		OracleId:    "oracle-1",
		ClaimId:     "claim-1",
		ContentHash: "hash",
		DataType:    string(types.EvidenceTypeDocument), // Oracle doesn't support this
	})
	if err == nil {
		t.Fatal("expected unsupported data type error")
	}
}

// ---------- Tests: ValidateBasic ----------

func TestMsgSubmitEvidence_ValidateBasic(t *testing.T) {
	addr := testAddr(1)

	// Valid
	msg := &types.MsgSubmitEvidence{
		Submitter:    addr,
		ClaimId:      "claim-1",
		EvidenceType: string(types.EvidenceTypeDocument),
		ContentHash:  "hash1",
		Description:  "test",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty submitter
	bad := &types.MsgSubmitEvidence{ClaimId: "claim-1", EvidenceType: string(types.EvidenceTypeDocument), ContentHash: "hash1", Description: "test"}
	if err := bad.ValidateBasic(); err == nil {
		t.Fatal("expected error for empty submitter")
	}

	// Invalid evidence type
	bad2 := &types.MsgSubmitEvidence{Submitter: addr, ClaimId: "claim-1", EvidenceType: "bad", ContentHash: "hash1", Description: "test"}
	if err := bad2.ValidateBasic(); err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestMsgAuditEvidence_ValidateBasic(t *testing.T) {
	addr := testAddr(1)

	msg := &types.MsgAuditEvidence{
		Auditor:    addr,
		EvidenceId: "ev-1",
		Score:      500000,
		Rationale:  "test",
		Stake:      "22000000",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Score too high
	bad := &types.MsgAuditEvidence{Auditor: addr, EvidenceId: "ev-1", Score: 1000001, Rationale: "test", Stake: "22000000"}
	if err := bad.ValidateBasic(); err == nil {
		t.Fatal("expected error for score > 1000000")
	}
}

func TestMsgRegisterOracle_ValidateBasic(t *testing.T) {
	addr := testAddr(1)

	msg := &types.MsgRegisterOracle{
		Operator:  addr,
		Name:      "Oracle",
		DataTypes: []string{string(types.EvidenceTypeOracleFeed)},
		Stake:     "100000000",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty data types
	bad := &types.MsgRegisterOracle{Operator: addr, Name: "Oracle", Stake: "100000000"}
	if err := bad.ValidateBasic(); err == nil {
		t.Fatal("expected error for empty data_types")
	}
}

func TestParamsValidate(t *testing.T) {
	// Valid
	p := types.DefaultParams()
	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid: accuracy > 1M
	p.AccuracyThreshold = 1000001
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for accuracy > 1000000")
	}

	// Invalid: min_audits = 0
	p2 := types.DefaultParams()
	p2.MinAuditsForConsensus = 0
	if err := p2.Validate(); err == nil {
		t.Fatal("expected error for min_audits = 0")
	}
}
