package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/bvm/keeper"
	"github.com/legible-money/legible/x/bvm/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

// ---------- Mock BankKeeper ----------

type mockBankKeeper struct {
	balances map[string]map[string]int64
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]map[string]int64),
	}
}

func (m *mockBankKeeper) SendCoins(_ context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		from := fromAddr.String()
		to := toAddr.String()
		if m.balances[from] == nil {
			m.balances[from] = make(map[string]int64)
		}
		if m.balances[to] == nil {
			m.balances[to] = make(map[string]int64)
		}
		m.balances[from][coin.Denom] -= coin.Amount.Int64()
		m.balances[to][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

func (m *mockBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	if bal, ok := m.balances[addr.String()]; ok {
		if amt, exists := bal[denom]; exists {
			return sdk.NewCoin(denom, sdkmath.NewInt(amt))
		}
	}
	return sdk.NewCoin(denom, sdkmath.ZeroInt())
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
	t.Helper()
	initAddresses()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockBK := newMockBankKeeper()

	k := keeper.NewKeeper(cdc, storeKey, mockBK, "lgm1authority")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	// Init default params
	k.InitGenesis(ctx, types.DefaultGenesisState())

	return k, ctx, mockBK
}

func testAddr(name string) string {
	return sdk.AccAddress([]byte(name + "_address_padding")).String()
}

var (
	creator1  string
	creator2  string
	caller1   string
	authority string
)

func initAddresses() {
	creator1 = testAddr("creator01")
	creator2 = testAddr("creator02")
	caller1 = testAddr("caller001")
	authority = "lgm1authority"
}

// ---------- State CRUD Tests ----------

func TestDefaultParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	params := k.GetParams(ctx)
	if params.MaxBytecodeSize != 1_048_576 {
		t.Fatalf("expected MaxBytecodeSize 1048576, got %d", params.MaxBytecodeSize)
	}
	if params.CallBaseGasCost != 21_000 {
		t.Fatalf("expected CallBaseGasCost 21000, got %d", params.CallBaseGasCost)
	}
}

func TestSetGetParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	custom := types.Params{
		MaxBytecodeSize:         512_000,
		MaxStateEntries:         50_000,
		MaxSchedulesPerContract: 50,
		DeployGasCost:           100_000,
		CallBaseGasCost:         10_000,
		StorageWriteGasCost:     3_000,
		StorageReadGasCost:      1_000,
	}
	k.SetParams(ctx, custom)
	got := k.GetParams(ctx)
	if got.MaxBytecodeSize != 512_000 {
		t.Fatalf("expected MaxBytecodeSize 512000, got %d", got.MaxBytecodeSize)
	}
	if got.MaxSchedulesPerContract != 50 {
		t.Fatalf("expected MaxSchedulesPerContract 50, got %d", got.MaxSchedulesPerContract)
	}
}

func TestSetGetContract(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	contract := types.DeployedContract{
		Address:         "lgm1contractabc123",
		CodeHash:        "hash123",
		Creator:         creator1,
		DeployedAtBlock: 100,
		IsAgent:         false,
		BytecodeSize:    256,
		Label:           "test-contract",
	}
	k.SetContract(ctx, contract)
	got, found := k.GetContract(ctx, "lgm1contractabc123")
	if !found {
		t.Fatal("contract not found")
	}
	if got.Creator != creator1 {
		t.Fatalf("expected creator %s, got %s", creator1, got.Creator)
	}
	if got.Label != "test-contract" {
		t.Fatalf("expected label test-contract, got %s", got.Label)
	}
}

func TestGetContractNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	_, found := k.GetContract(ctx, "nonexistent")
	if found {
		t.Fatal("expected not found")
	}
}

func TestGetContractsByCreator(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	for i := 0; i < 3; i++ {
		k.SetContract(ctx, types.DeployedContract{
			Address:  testAddr("contract" + string(rune('A'+i))),
			CodeHash: "hash",
			Creator:  creator1,
		})
	}
	k.SetContract(ctx, types.DeployedContract{
		Address:  testAddr("other"),
		CodeHash: "hash",
		Creator:  creator2,
	})
	contracts := k.GetContractsByCreator(ctx, creator1)
	if len(contracts) != 3 {
		t.Fatalf("expected 3 contracts, got %d", len(contracts))
	}
}

func TestSetGetCode(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	code := types.ContractCode{
		CodeHash: "abc123",
		Bytecode: []byte{0x01, 0x02, 0x03},
		RefCount: 1,
	}
	k.SetCode(ctx, code)
	got, found := k.GetCode(ctx, "abc123")
	if !found {
		t.Fatal("code not found")
	}
	if got.RefCount != 1 {
		t.Fatalf("expected refcount 1, got %d", got.RefCount)
	}
}

func TestContractState_SetGet(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetContractState(ctx, "contract1", "key1", "value1")
	k.SetContractState(ctx, "contract1", "key2", "value2")
	k.SetContractState(ctx, "contract2", "key1", "other")

	val, found := k.GetContractState(ctx, "contract1", "key1")
	if !found || val != "value1" {
		t.Fatalf("expected value1, got %s (found=%v)", val, found)
	}

	val, found = k.GetContractState(ctx, "contract1", "key2")
	if !found || val != "value2" {
		t.Fatalf("expected value2, got %s (found=%v)", val, found)
	}

	// Different contract
	val, found = k.GetContractState(ctx, "contract2", "key1")
	if !found || val != "other" {
		t.Fatalf("expected other, got %s (found=%v)", val, found)
	}

	// Not found
	_, found = k.GetContractState(ctx, "contract1", "nonexistent")
	if found {
		t.Fatal("expected not found")
	}

	// Count
	count := k.CountContractState(ctx, "contract1")
	if count != 2 {
		t.Fatalf("expected 2 state entries, got %d", count)
	}
}

func TestSetGetSchedule(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	schedule := types.ContractSchedule{
		ScheduleId:      "sched-100-0",
		ContractAddress: "contract1",
		Caller:          creator1,
		Method:          "execute",
		ExecuteAtBlock:  200,
		MaxGas:          50000,
	}
	k.SetSchedule(ctx, schedule)
	got, found := k.GetSchedule(ctx, "sched-100-0")
	if !found {
		t.Fatal("schedule not found")
	}
	if got.Method != "execute" {
		t.Fatalf("expected method execute, got %s", got.Method)
	}
}

func TestGetPendingSchedules(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Schedule at block 200
	k.SetSchedule(ctx, types.ContractSchedule{
		ScheduleId:      "sched-1",
		ContractAddress: "contract1",
		Caller:          creator1,
		Method:          "a",
		ExecuteAtBlock:  200,
	})
	// Schedule at block 300
	k.SetSchedule(ctx, types.ContractSchedule{
		ScheduleId:      "sched-2",
		ContractAddress: "contract1",
		Caller:          creator1,
		Method:          "b",
		ExecuteAtBlock:  300,
	})
	// Cancelled schedule at block 200
	k.SetSchedule(ctx, types.ContractSchedule{
		ScheduleId:      "sched-3",
		ContractAddress: "contract1",
		Caller:          creator1,
		Method:          "c",
		ExecuteAtBlock:  200,
		Cancelled:       true,
	})

	pending := k.GetPendingSchedules(ctx, 200)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending schedule at block 200, got %d", len(pending))
	}
	if pending[0].ScheduleId != "sched-1" {
		t.Fatalf("expected sched-1, got %s", pending[0].ScheduleId)
	}
}

// ---------- Msg Handler Tests ----------

func TestDeployContract_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x00, 0x61, 0x73, 0x6d}, // wasm magic bytes
		Label:    "my-contract",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Address == "" {
		t.Fatal("expected non-empty contract address")
	}
	if resp.CodeHash == "" {
		t.Fatal("expected non-empty code hash")
	}

	// Verify contract stored
	contract, found := k.GetContract(ctx, resp.Address)
	if !found {
		t.Fatal("contract not found in state")
	}
	if contract.Creator != creator1 {
		t.Fatalf("expected creator %s, got %s", creator1, contract.Creator)
	}
	if contract.Label != "my-contract" {
		t.Fatalf("expected label my-contract, got %s", contract.Label)
	}

	// Verify code stored
	code, found := k.GetCode(ctx, resp.CodeHash)
	if !found {
		t.Fatal("code not found in state")
	}
	if code.RefCount != 1 {
		t.Fatalf("expected refcount 1, got %d", code.RefCount)
	}
}

func TestDeployContract_BytecodeTooBig(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set small max
	params := k.GetParams(ctx)
	params.MaxBytecodeSize = 10
	k.SetParams(ctx, params)

	_, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: make([]byte, 20),
	})
	if err == nil {
		t.Fatal("expected error for bytecode too big")
	}
}

func TestDeployContract_CodeDedup(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	bytecode := []byte{0x00, 0x61, 0x73, 0x6d, 0x01}

	resp1, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: bytecode,
		Label:    "contract-a",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp2, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator2,
		Bytecode: bytecode,
		Label:    "contract-b",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same code hash
	if resp1.CodeHash != resp2.CodeHash {
		t.Fatalf("expected same code hash, got %s vs %s", resp1.CodeHash, resp2.CodeHash)
	}
	// Different addresses
	if resp1.Address == resp2.Address {
		t.Fatal("expected different addresses")
	}

	// RefCount should be 2
	code, _ := k.GetCode(ctx, resp1.CodeHash)
	if code.RefCount != 2 {
		t.Fatalf("expected refcount 2, got %d", code.RefCount)
	}
}

func TestDeployContract_AgentContract(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01, 0x02},
		IsAgent:  true,
		AgentDID: "did:lgm:agent123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	contract, _ := k.GetContract(ctx, resp.Address)
	if !contract.IsAgent {
		t.Fatal("expected is_agent=true")
	}
	if contract.AgentDID != "did:lgm:agent123" {
		t.Fatalf("expected agent DID did:lgm:agent123, got %s", contract.AgentDID)
	}
}

func TestCallContract_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy a simple contract: PUSH1 42, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
	// Returns 42 as a 32-byte big-endian value
	bytecode := []byte{
		0x60, 42,   // PUSH1 42
		0x60, 0,    // PUSH1 0
		0x52,       // MSTORE
		0x60, 32,   // PUSH1 32
		0x60, 0,    // PUSH1 0
		0xF3,       // RETURN
	}
	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: bytecode,
	})

	// Call the contract
	resp, err := msgServer.CallContract(ctx, &types.MsgCallContract{
		Caller:          caller1,
		ContractAddress: deployResp.Address,
		Method:          "transfer",
		GasLimit:        100_000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.GasUsed == 0 {
		t.Fatal("expected non-zero gas used")
	}
	// ReturnData should be the hex encoding of 32 bytes containing 42
	if resp.ReturnData == "" {
		t.Fatal("expected non-empty return data")
	}
}

func TestCallContract_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.CallContract(ctx, &types.MsgCallContract{
		Caller:          caller1,
		ContractAddress: "nonexistent",
		Method:          "transfer",
		GasLimit:        100_000,
	})
	if err == nil {
		t.Fatal("expected error for contract not found")
	}
}

func TestScheduleContract_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy first
	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01},
	})

	resp, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.Address,
		Method:          "tick",
		ExecuteAtBlock:  200,
		MaxGas:          50_000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScheduleId == "" {
		t.Fatal("expected non-empty schedule ID")
	}

	// Verify stored
	schedule, found := k.GetSchedule(ctx, resp.ScheduleId)
	if !found {
		t.Fatal("schedule not found")
	}
	if schedule.Method != "tick" {
		t.Fatalf("expected method tick, got %s", schedule.Method)
	}
	if schedule.ExecuteAtBlock != 200 {
		t.Fatalf("expected execute at block 200, got %d", schedule.ExecuteAtBlock)
	}
}

func TestScheduleContract_InPast(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01},
	})

	// Block height is 100, try to schedule at 50
	_, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.Address,
		Method:          "tick",
		ExecuteAtBlock:  50,
		MaxGas:          50_000,
	})
	if err == nil {
		t.Fatal("expected error for schedule in past")
	}
}

func TestScheduleContract_LimitReached(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set max schedules to 2
	params := k.GetParams(ctx)
	params.MaxSchedulesPerContract = 2
	k.SetParams(ctx, params)

	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01},
	})

	// Schedule 2 (fills limit)
	for i := 0; i < 2; i++ {
		_, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
			Caller:          creator1,
			ContractAddress: deployResp.Address,
			Method:          "tick",
			ExecuteAtBlock:  uint64(200 + i),
			MaxGas:          50_000,
		})
		if err != nil {
			t.Fatalf("unexpected error on schedule %d: %v", i, err)
		}
	}

	// Third should fail
	_, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.Address,
		Method:          "tick",
		ExecuteAtBlock:  300,
		MaxGas:          50_000,
	})
	if err == nil {
		t.Fatal("expected error for schedule limit reached")
	}
}

func TestScheduleContract_ContractNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: "nonexistent",
		Method:          "tick",
		ExecuteAtBlock:  200,
		MaxGas:          50_000,
	})
	if err == nil {
		t.Fatal("expected error for contract not found")
	}
}

func TestCancelSchedule_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01},
	})

	schedResp, _ := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.Address,
		Method:          "tick",
		ExecuteAtBlock:  200,
		MaxGas:          50_000,
	})

	_, err := msgServer.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Caller:     creator1,
		ScheduleId: schedResp.ScheduleId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cancelled
	schedule, _ := k.GetSchedule(ctx, schedResp.ScheduleId)
	if !schedule.Cancelled {
		t.Fatal("expected schedule to be cancelled")
	}
}

func TestCancelSchedule_NotOwner(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01},
	})

	schedResp, _ := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.Address,
		Method:          "tick",
		ExecuteAtBlock:  200,
		MaxGas:          50_000,
	})

	// Try to cancel as different caller
	_, err := msgServer.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Caller:     creator2,
		ScheduleId: schedResp.ScheduleId,
	})
	if err == nil {
		t.Fatal("expected error for not owner")
	}
}

func TestCancelSchedule_AlreadyExecuted(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Manually create an executed schedule
	schedule := types.ContractSchedule{
		ScheduleId:      "sched-exec",
		ContractAddress: "contract1",
		Caller:          creator1,
		Method:          "tick",
		ExecuteAtBlock:  50,
		Executed:        true,
	}
	k.SetSchedule(ctx, schedule)

	msgServer := keeper.NewMsgServerImpl(k)
	_, err := msgServer.CancelSchedule(ctx, &types.MsgCancelSchedule{
		Caller:     creator1,
		ScheduleId: "sched-exec",
	})
	if err == nil {
		t.Fatal("expected error for already executed")
	}
}

func TestUpdateContractState_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy contract first
	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01},
	})

	_, err := msgServer.UpdateContractState(ctx, &types.MsgUpdateContractState{
		Authority:       authority,
		ContractAddress: deployResp.Address,
		Key:             "admin",
		Value:           "new_admin_address",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify state set
	val, found := k.GetContractState(ctx, deployResp.Address, "admin")
	if !found || val != "new_admin_address" {
		t.Fatalf("expected new_admin_address, got %s (found=%v)", val, found)
	}
}

func TestUpdateContractState_NotAuthority(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01},
	})

	_, err := msgServer.UpdateContractState(ctx, &types.MsgUpdateContractState{
		Authority:       creator1, // not the authority
		ContractAddress: deployResp.Address,
		Key:             "admin",
		Value:           "hacked",
	})
	if err == nil {
		t.Fatal("expected error for unauthorized")
	}
}

// ---------- BeginBlock Tests ----------

func TestBeginBlock_ExecutesSchedules(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01},
	})

	// Schedule for block 200
	schedResp, _ := msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.Address,
		Method:          "tick",
		ExecuteAtBlock:  200,
		MaxGas:          50_000,
	})

	// Move to block 200 and execute
	ctx = ctx.WithBlockHeight(200)
	k.ExecutePendingSchedules(ctx)

	// Verify executed
	schedule, _ := k.GetSchedule(ctx, schedResp.ScheduleId)
	if !schedule.Executed {
		t.Fatal("expected schedule to be executed")
	}
}

// ---------- Query Tests ----------

func TestQueryContract(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)
	queryServer := keeper.NewQueryServerImpl(k)

	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01, 0x02, 0x03},
		Label:    "query-test",
	})

	resp, err := queryServer.Contract(ctx, &types.QueryContractRequest{
		Address: deployResp.Address,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Contract.Label != "query-test" {
		t.Fatalf("expected label query-test, got %s", resp.Contract.Label)
	}
}

func TestQueryContractNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	_, err := queryServer.Contract(ctx, &types.QueryContractRequest{
		Address: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for contract not found")
	}
}

func TestQueryContractsByCreator(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)
	queryServer := keeper.NewQueryServerImpl(k)

	for i := 0; i < 3; i++ {
		_, err := msgServer.DeployContract(ctx, &types.MsgDeployContract{
			Creator:  creator1,
			Bytecode: []byte{byte(i)},
		})
		if err != nil {
			t.Fatalf("unexpected error deploying contract %d: %v", i, err)
		}
	}

	resp, err := queryServer.ContractsByCreator(ctx, &types.QueryContractsByCreatorRequest{
		Creator: creator1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 3 {
		t.Fatalf("expected total 3, got %d", resp.Total)
	}
}

func TestQueryCode(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)
	queryServer := keeper.NewQueryServerImpl(k)

	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0xCA, 0xFE},
	})

	resp, err := queryServer.Code(ctx, &types.QueryCodeRequest{
		CodeHash: deployResp.CodeHash,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Code.Bytecode) != 2 {
		t.Fatalf("expected 2 bytecode bytes, got %d", len(resp.Code.Bytecode))
	}
}

func TestQueryContractState(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	k.SetContractState(ctx, "contract1", "key1", "value1")

	resp, err := queryServer.ContractState(ctx, &types.QueryContractStateRequest{
		ContractAddress: "contract1",
		Key:             "key1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Found || resp.Value != "value1" {
		t.Fatalf("expected value1, got %s (found=%v)", resp.Value, resp.Found)
	}
}

func TestQuerySchedule(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	k.SetSchedule(ctx, types.ContractSchedule{
		ScheduleId:      "sched-test",
		ContractAddress: "contract1",
		Caller:          creator1,
		Method:          "run",
		ExecuteAtBlock:  500,
	})

	resp, err := queryServer.Schedule(ctx, &types.QueryScheduleRequest{
		ScheduleId: "sched-test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Schedule.Method != "run" {
		t.Fatalf("expected method run, got %s", resp.Schedule.Method)
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	queryServer := keeper.NewQueryServerImpl(k)

	resp, err := queryServer.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.MaxBytecodeSize != 1_048_576 {
		t.Fatalf("expected MaxBytecodeSize 1048576, got %d", resp.Params.MaxBytecodeSize)
	}
}

// ---------- Genesis Tests ----------

func TestGenesisRoundtrip(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy a contract
	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01, 0x02, 0x03},
		Label:    "genesis-test",
	})

	// Schedule something
	msgServer.ScheduleContract(ctx, &types.MsgScheduleContract{
		Caller:          creator1,
		ContractAddress: deployResp.Address,
		Method:          "tick",
		ExecuteAtBlock:  500,
		MaxGas:          50_000,
	})

	// Export
	gs := k.ExportGenesis(ctx)

	if len(gs.Contracts) != 1 {
		t.Fatalf("expected 1 contract in genesis, got %d", len(gs.Contracts))
	}
	if len(gs.Code) != 1 {
		t.Fatalf("expected 1 code in genesis, got %d", len(gs.Code))
	}
	if len(gs.Schedules) != 1 {
		t.Fatalf("expected 1 schedule in genesis, got %d", len(gs.Schedules))
	}

	// Re-init into fresh keeper
	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, gs)

	// Verify contract survives roundtrip
	contract, found := k2.GetContract(ctx2, deployResp.Address)
	if !found {
		t.Fatal("contract not found after genesis roundtrip")
	}
	if contract.Label != "genesis-test" {
		t.Fatalf("expected label genesis-test, got %s", contract.Label)
	}
}

func TestGenesisRoundtrip_IncludesState(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Deploy a contract
	deployResp, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: []byte{0x01, 0x02, 0x03},
		Label:    "state-test",
	})

	// Set contract state
	k.SetContractState(ctx, deployResp.Address, "counter", "42")
	k.SetContractState(ctx, deployResp.Address, "owner", creator1)

	// Export
	gs := k.ExportGenesis(ctx)
	if len(gs.State) != 2 {
		t.Fatalf("expected 2 state entries in genesis, got %d", len(gs.State))
	}

	// Re-init into fresh keeper
	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, gs)

	// Verify state survives roundtrip
	val, found := k2.GetContractState(ctx2, deployResp.Address, "counter")
	if !found || val != "42" {
		t.Fatalf("expected counter=42, got %s (found=%v)", val, found)
	}
	val, found = k2.GetContractState(ctx2, deployResp.Address, "owner")
	if !found || val != creator1 {
		t.Fatalf("expected owner=%s, got %s (found=%v)", creator1, val, found)
	}
}

// ---------- RefCount Lifecycle Tests ----------

func TestDeleteContract_DecrRefCount(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	bytecode := []byte{0x01, 0x02, 0x03}

	// Deploy two contracts with same bytecode (RefCount = 2)
	resp1, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator1,
		Bytecode: bytecode,
	})
	resp2, _ := msgServer.DeployContract(ctx, &types.MsgDeployContract{
		Creator:  creator2,
		Bytecode: bytecode,
	})

	// Verify RefCount = 2
	code, _ := k.GetCode(ctx, resp1.CodeHash)
	if code.RefCount != 2 {
		t.Fatalf("expected refcount 2, got %d", code.RefCount)
	}

	// Delete first contract — RefCount should drop to 1
	contract1, _ := k.GetContract(ctx, resp1.Address)
	k.DeleteContract(ctx, contract1)

	code, found := k.GetCode(ctx, resp1.CodeHash)
	if !found {
		t.Fatal("code should still exist with refcount 1")
	}
	if code.RefCount != 1 {
		t.Fatalf("expected refcount 1 after delete, got %d", code.RefCount)
	}

	// Delete second contract — code should be garbage collected
	contract2, _ := k.GetContract(ctx, resp2.Address)
	k.DeleteContract(ctx, contract2)

	_, found = k.GetCode(ctx, resp1.CodeHash)
	if found {
		t.Fatal("code should be garbage collected when refcount reaches 0")
	}
}

// ---------- Overdue Schedule Tests ----------

func TestGetPendingSchedules_IncludesOverdue(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Schedule at block 150 (overdue if chain is at 200)
	k.SetSchedule(ctx, types.ContractSchedule{
		ScheduleId:      "sched-overdue",
		ContractAddress: "contract1",
		Caller:          creator1,
		Method:          "a",
		ExecuteAtBlock:  150,
	})
	// Schedule at block 200 (on-time)
	k.SetSchedule(ctx, types.ContractSchedule{
		ScheduleId:      "sched-ontime",
		ContractAddress: "contract1",
		Caller:          creator1,
		Method:          "b",
		ExecuteAtBlock:  200,
	})
	// Schedule at block 300 (future, should not be included)
	k.SetSchedule(ctx, types.ContractSchedule{
		ScheduleId:      "sched-future",
		ContractAddress: "contract1",
		Caller:          creator1,
		Method:          "c",
		ExecuteAtBlock:  300,
	})

	// GetPendingSchedules up to block 200 should include overdue (150) + on-time (200)
	pending := k.GetPendingSchedules(ctx, 200)
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending schedules (overdue + on-time), got %d", len(pending))
	}
}
