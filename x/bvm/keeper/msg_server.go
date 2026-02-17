package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"

	corestore "cosmossdk.io/core/store"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/bvm/types"
	"github.com/legible-money/legible/x/bvm/vm"
)

// kvStoreAdapter wraps a cosmossdk.io/core/store.KVStore to implement
// cosmossdk.io/store/types.KVStore, which the BVM interpreter requires.
type kvStoreAdapter struct {
	store corestore.KVStore
}

func (a *kvStoreAdapter) Get(key []byte) []byte {
	bz, err := a.store.Get(key)
	if err != nil {
		return nil
	}
	return bz
}

func (a *kvStoreAdapter) Has(key []byte) bool {
	bz, err := a.store.Get(key)
	return err == nil && bz != nil
}

func (a *kvStoreAdapter) Set(key, value []byte) {
	_ = a.store.Set(key, value)
}

func (a *kvStoreAdapter) Delete(key []byte) {
	_ = a.store.Delete(key)
}

func (a *kvStoreAdapter) Iterator(start, end []byte) storetypes.Iterator {
	iter, err := a.store.Iterator(start, end)
	if err != nil {
		return &noopIterator{}
	}
	return iter
}

func (a *kvStoreAdapter) ReverseIterator(start, end []byte) storetypes.Iterator {
	iter, err := a.store.ReverseIterator(start, end)
	if err != nil {
		return &noopIterator{}
	}
	return iter
}

func (a *kvStoreAdapter) GetStoreType() storetypes.StoreType {
	return storetypes.StoreTypeIAVL
}

func (a *kvStoreAdapter) CacheWrap() storetypes.CacheWrap {
	panic("CacheWrap not supported on kvStoreAdapter")
}

func (a *kvStoreAdapter) CacheWrapWithTrace(_ io.Writer, _ storetypes.TraceContext) storetypes.CacheWrap {
	panic("CacheWrapWithTrace not supported on kvStoreAdapter")
}

// noopIterator is returned when iterator creation fails.
type noopIterator struct{}

func (n *noopIterator) Domain() (start, end []byte) { return nil, nil }
func (n *noopIterator) Valid() bool                 { return false }
func (n *noopIterator) Next()                       {}
func (n *noopIterator) Key() []byte                 { return nil }
func (n *noopIterator) Value() []byte               { return nil }
func (n *noopIterator) Close() error                { return nil }
func (n *noopIterator) Error() error                { return nil }

// balanceQuerierAdapter wraps a BankKeeper + sdk.Context to implement vm.BalanceQuerier.
// Resolves real on-chain balances for BALANCE/SELFBALANCE opcodes.
type balanceQuerierAdapter struct {
	bk  types.BankKeeper
	ctx sdk.Context
}

func (a *balanceQuerierAdapter) GetBalanceByAddress(addr []byte) *big.Int {
	// addr is 20 bytes (EVM-style) — convert to bech32 sdk.AccAddress
	accAddr := sdk.AccAddress(addr)
	coin := a.bk.GetBalance(a.ctx, accAddr, "ulgm")
	return coin.Amount.BigInt()
}

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns an ExtendedMsgServer implementation that includes
// both proto-generated and hand-written message handlers.
func NewMsgServerImpl(keeper Keeper) types.ExtendedMsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.ExtendedMsgServer = &msgServer{}

// scheduleKeeperAdapter wraps a Keeper + sdk.Context to implement vm.ScheduleKeeper.
// Created per-call so the context is always correct for deterministic state access.
type scheduleKeeperAdapter struct {
	keeper Keeper
	ctx    sdk.Context
}

func (a *scheduleKeeperAdapter) CreateSchedule(contractAddr, caller, method, payload string, executeAtBlock, maxGas uint64) (string, error) {
	currentBlock := uint64(a.ctx.BlockHeight())
	if executeAtBlock <= currentBlock {
		return "", fmt.Errorf("execute_at_block must be in the future")
	}
	counter := a.keeper.GetNextScheduleId(a.ctx)
	scheduleId := fmt.Sprintf("sched-%d-%d", currentBlock, counter)
	schedule := types.ContractSchedule{
		ScheduleId:      scheduleId,
		ContractAddress: contractAddr,
		Caller:          caller,
		Method:          method,
		Payload:         payload,
		ExecuteAtBlock:  executeAtBlock,
		MaxGas:          maxGas,
		Executed:        false,
		Cancelled:       false,
	}
	a.keeper.SetSchedule(a.ctx, schedule)
	return scheduleId, nil
}

func (a *scheduleKeeperAdapter) CancelScheduleByID(scheduleId, caller string) error {
	schedule, found := a.keeper.GetSchedule(a.ctx, scheduleId)
	if !found {
		return fmt.Errorf("schedule not found")
	}
	if schedule.Executed || schedule.Cancelled {
		return fmt.Errorf("schedule already executed or cancelled")
	}
	store := a.keeper.storeService.OpenKVStore(a.ctx)
	_ = store.Delete(types.ScheduleBlockIndexKey(schedule.ExecuteAtBlock, schedule.ScheduleId))
	schedule.Cancelled = true
	a.keeper.SetSchedule(a.ctx, schedule)
	return nil
}

func (a *scheduleKeeperAdapter) ModifyScheduleBlock(scheduleId, caller string, newBlock uint64) error {
	schedule, found := a.keeper.GetSchedule(a.ctx, scheduleId)
	if !found {
		return fmt.Errorf("schedule not found")
	}
	if schedule.Executed || schedule.Cancelled {
		return fmt.Errorf("schedule already executed or cancelled")
	}
	currentBlock := uint64(a.ctx.BlockHeight())
	if newBlock <= currentBlock {
		return fmt.Errorf("new block must be in the future")
	}
	store := a.keeper.storeService.OpenKVStore(a.ctx)
	_ = store.Delete(types.ScheduleBlockIndexKey(schedule.ExecuteAtBlock, schedule.ScheduleId))
	schedule.ExecuteAtBlock = newBlock
	a.keeper.SetSchedule(a.ctx, schedule)
	return nil
}

func (a *scheduleKeeperAdapter) CountActiveSchedules(contractAddr string) uint64 {
	return a.keeper.CountContractSchedules(a.ctx, contractAddr)
}

func (a *scheduleKeeperAdapter) MaxSchedulesAllowed() uint64 {
	return a.keeper.GetParams(a.ctx).MaxSchedulesPerContract
}

// vmAuthKeeperAdapter bridges types.AuthKeeper → vm.VMAuthKeeper.
type vmAuthKeeperAdapter struct {
	ak  types.AuthKeeper
	ctx sdk.Context
}

func (a *vmAuthKeeperAdapter) RegisterSessionKey(_ context.Context, ownerDID, pubKeyHex string, canTransfer, canStake, canClaims, canVote bool, expiresAtBlock uint64) error {
	caps := types.SessionCapabilities{
		CanTransfer:     canTransfer,
		CanStake:        canStake,
		CanSubmitClaims: canClaims,
		CanVote:         canVote,
	}
	// Use ownerDID as both owner and keyHash (the adapter resolves internally)
	return a.ak.RegisterSessionKey(a.ctx, ownerDID, "", pubKeyHex, caps, expiresAtBlock)
}

func (a *vmAuthKeeperAdapter) RevokeSessionKey(_ context.Context, ownerDID, keyHash string) error {
	return a.ak.RevokeSessionKey(a.ctx, ownerDID, keyHash)
}

func (a *vmAuthKeeperAdapter) GetSessionForDID(_ context.Context, ownerDID string, blockHeight uint64) (expiresAt uint64, canTransfer, canStake, canClaims, canVote, found bool) {
	height := blockHeight
	if height == 0 {
		height = uint64(a.ctx.BlockHeight())
	}
	caps, ok := a.ak.GetSessionCapabilities(a.ctx, ownerDID, height)
	if !ok {
		return 0, false, false, false, false, false
	}
	// We don't have direct access to expiresAt via the interface, return 0
	return 0, caps.CanTransfer, caps.CanStake, caps.CanSubmitClaims, caps.CanVote, true
}

// buildHost creates a HostFunctions implementation wired to the keeper for the given context.
func (k Keeper) buildHost(ctx sdk.Context, stateDB vm.StateDB) vm.HostFunctions {
	adapter := &scheduleKeeperAdapter{keeper: k, ctx: ctx}
	agent := vm.NewAgentHostWithSchedules(ctx, stateDB, adapter)

	// Wire auth into treasury host for dual key unification
	var treasury *vm.TreasuryHost
	if k.authKeeper != nil {
		vmAuth := &vmAuthKeeperAdapter{ak: k.authKeeper, ctx: ctx}
		treasury = vm.NewTreasuryHostWithAuth(stateDB, vmAuth, ctx)
	} else {
		treasury = vm.NewTreasuryHost(stateDB)
	}

	knowledge := vm.NewKnowledgeHost(k.knowledgeKeeper, ctx)
	return &prodCompositeHost{agent: agent, treasury: treasury, kh: knowledge}
}

// prodCompositeHost combines AgentHost, TreasuryHost, and KnowledgeHost into a full vm.HostFunctions.
type prodCompositeHost struct {
	agent    *vm.AgentHost
	treasury *vm.TreasuryHost
	kh       *vm.KnowledgeHost
}

// Knowledge
func (c *prodCompositeHost) QueryFact(factId []byte) (bool, uint64, []byte) {
	return c.kh.QueryFact(factId)
}
func (c *prodCompositeHost) QueryDomain(domain []byte, limit uint64) (uint64, [][]byte) {
	return c.kh.QueryDomain(domain, limit)
}
func (c *prodCompositeHost) SubmitClaim(callerDID string, domain, dataHash, evidenceHash []byte, blockNumber uint64) []byte {
	return c.kh.SubmitClaim(callerDID, domain, dataHash, evidenceHash, blockNumber)
}
func (c *prodCompositeHost) CommitVote(callerDID string, claimId, voteHash []byte, blockNumber uint64) bool {
	return c.kh.CommitVote(callerDID, claimId, voteHash, blockNumber)
}
func (c *prodCompositeHost) RevealVote(callerDID string, claimId []byte, vote, salt *big.Int, blockNumber uint64) bool {
	return c.kh.RevealVote(callerDID, claimId, vote, salt, blockNumber)
}
func (c *prodCompositeHost) InitiateChallenge(callerDID string, factId, evidenceHash []byte, stake *big.Int, blockNumber uint64) []byte {
	return c.kh.InitiateChallenge(callerDID, factId, evidenceHash, stake, blockNumber)
}

// Agent Identity
func (c *prodCompositeHost) AgentDID(caller []byte) []byte     { return c.agent.AgentDID(caller) }
func (c *prodCompositeHost) AgentHome(callerDID string) []byte { return c.agent.AgentHome(callerDID) }
func (c *prodCompositeHost) VerifyIdentity(targetDID []byte) bool {
	return c.agent.VerifyIdentity(targetDID)
}
func (c *prodCompositeHost) AttestIdentity(callerDID string, targetDID, attType, data []byte) []byte {
	return c.agent.AttestIdentity(callerDID, targetDID, attType, data)
}

// Agent Belief
func (c *prodCompositeHost) GetBelief(callerDID string, factId []byte) uint64 {
	return c.agent.GetBelief(callerDID, factId)
}
func (c *prodCompositeHost) SetBelief(callerDID string, factId []byte, conf uint64, src []byte) {
	c.agent.SetBelief(callerDID, factId, conf, src)
}
func (c *prodCompositeHost) CompareBelief(callerDID string, factId []byte) int64 {
	return c.agent.CompareBelief(callerDID, factId)
}
func (c *prodCompositeHost) BeliefCount(callerDID string, domain []byte) uint64 {
	return c.agent.BeliefCount(callerDID, domain)
}

// Agent Relationship
func (c *prodCompositeHost) GetTrust(callerDID string, targetDID []byte) uint64 {
	return c.agent.GetTrust(callerDID, targetDID)
}
func (c *prodCompositeHost) SetTrust(callerDID string, targetDID []byte, level uint64, reason []byte) {
	c.agent.SetTrust(callerDID, targetDID, level, reason)
}
func (c *prodCompositeHost) IsTrusted(callerDID string, targetDID []byte, threshold uint64) bool {
	return c.agent.IsTrusted(callerDID, targetDID, threshold)
}
func (c *prodCompositeHost) Delegate(callerDID string, targetDID, cap, constraints []byte) []byte {
	return c.agent.Delegate(callerDID, targetDID, cap, constraints)
}

// Agent Autonomy
func (c *prodCompositeHost) ScheduleSelf(contract, callData []byte, execBlock, gasLimit uint64) []byte {
	return c.agent.ScheduleSelf(contract, callData, execBlock, gasLimit)
}
func (c *prodCompositeHost) CancelScheduleHost(scheduleId []byte) bool {
	return c.agent.CancelScheduleHost(scheduleId)
}
func (c *prodCompositeHost) ModifySchedule(scheduleId []byte, newBlock uint64) bool {
	return c.agent.ModifySchedule(scheduleId, newBlock)
}
func (c *prodCompositeHost) GetScheduleCount(contract []byte) uint64 {
	return c.agent.GetScheduleCount(contract)
}

// Extended Relationships
func (c *prodCompositeHost) ListTrusted(callerDID string, offset, limit uint64) (uint64, [][]byte) {
	return c.agent.ListTrusted(callerDID, offset, limit)
}
func (c *prodCompositeHost) BlockAgent(callerDID string, targetDID, reason []byte) bool {
	return c.agent.BlockAgent(callerDID, targetDID, reason)
}
func (c *prodCompositeHost) UnblockAgent(callerDID string, targetDID []byte) bool {
	return c.agent.UnblockAgent(callerDID, targetDID)
}
func (c *prodCompositeHost) RevokeDelegation(callerDID string, delegationId []byte) bool {
	return c.agent.RevokeDelegation(callerDID, delegationId)
}

// Treasury
func (c *prodCompositeHost) ClaimVesting(callerDID string, vestingId []byte) *big.Int {
	return c.treasury.ClaimVesting(callerDID, vestingId)
}
func (c *prodCompositeHost) QueryVesting(callerDID string, off, lim uint64) (uint64, [][]byte) {
	return c.treasury.QueryVesting(callerDID, off, lim)
}
func (c *prodCompositeHost) StakeTokens(callerDID string, amount *big.Int, valDID []byte) []byte {
	return c.treasury.StakeTokens(callerDID, amount, valDID)
}
func (c *prodCompositeHost) UnstakeTokens(callerDID string, stakeId []byte) *big.Int {
	return c.treasury.UnstakeTokens(callerDID, stakeId)
}
func (c *prodCompositeHost) QueryExpiring(callerDID string, periodId []byte) (*big.Int, uint64) {
	return c.treasury.QueryExpiring(callerDID, periodId)
}
func (c *prodCompositeHost) ExtendExpiring(callerDID string, periodId []byte, newExpiry uint64) bool {
	return c.treasury.ExtendExpiring(callerDID, periodId, newExpiry)
}
func (c *prodCompositeHost) SetSpendLimit(callerDID string, limitType, amount []byte, period uint64) bool {
	return c.treasury.SetSpendLimit(callerDID, limitType, amount, period)
}
func (c *prodCompositeHost) QueryTreasury(callerDID string) (*big.Int, *big.Int, *big.Int, *big.Int) {
	return c.treasury.QueryTreasury(callerDID)
}

// Guardian
func (c *prodCompositeHost) QueryAlerts(callerDID string, off, lim uint64) (uint64, [][]byte) {
	return c.treasury.QueryAlerts(callerDID, off, lim)
}
func (c *prodCompositeHost) DismissAlert(callerDID string, alertId []byte) bool {
	return c.treasury.DismissAlert(callerDID, alertId)
}
func (c *prodCompositeHost) SetDefense(callerDID string, defenseType, config, threshold []byte) bool {
	return c.treasury.SetDefense(callerDID, defenseType, config, threshold)
}
func (c *prodCompositeHost) QueryDefenses(callerDID string) (uint64, [][]byte) {
	return c.treasury.QueryDefenses(callerDID)
}

// Access Control
func (c *prodCompositeHost) RegisterKey(callerDID string, keyType, pubKey, perms, expiry []byte) bool {
	return c.treasury.RegisterKey(callerDID, keyType, pubKey, perms, expiry)
}
func (c *prodCompositeHost) RevokeKey(callerDID string, keyId []byte) bool {
	return c.treasury.RevokeKey(callerDID, keyId)
}
func (c *prodCompositeHost) QueryPermissions(callerDID string, role []byte) *big.Int {
	return c.treasury.QueryPermissions(callerDID, role)
}
func (c *prodCompositeHost) QuerySession(callerDID string) ([]byte, []byte, uint64, uint64) {
	return c.treasury.QuerySession(callerDID)
}

// Introspection
func (c *prodCompositeHost) SelfBalance(callerDID string) *big.Int {
	return c.treasury.SelfBalance(callerDID)
}
func (c *prodCompositeHost) SelfStake(callerDID string) *big.Int {
	return c.treasury.SelfStake(callerDID)
}
func (c *prodCompositeHost) SelfReputation(callerDID string) uint64 {
	return c.treasury.SelfReputation(callerDID)
}
func (c *prodCompositeHost) SelfAccuracy(callerDID string) uint64 {
	return c.treasury.SelfAccuracy(callerDID)
}

// DeployContract deploys a new contract to the BVM.
func (m msgServer) DeployContract(goCtx context.Context, msg *types.MsgDeployContract) (*types.MsgDeployContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.GetParams(ctx)

	// Validate bytecode size
	if uint64(len(msg.Bytecode)) > params.MaxBytecodeSize {
		return nil, types.ErrBytecodeTooBig
	}

	// Compute code hash
	hash := sha256.Sum256(msg.Bytecode)
	codeHash := fmt.Sprintf("%x", hash)

	// Check for code dedup: increment refCount if code already exists
	existingCode, codeExists := m.GetCode(ctx, codeHash)
	if codeExists {
		existingCode.RefCount++
		m.SetCode(ctx, existingCode)
	} else {
		m.SetCode(ctx, types.ContractCode{
			CodeHash: codeHash,
			Bytecode: msg.Bytecode,
			RefCount: 1,
		})
	}

	// Generate deterministic contract address
	nonce := m.GetNextContractNonce(ctx)
	blockHeight := uint64(ctx.BlockHeight())
	addrInput := fmt.Sprintf("%s/%d/%d", msg.Deployer, blockHeight, nonce)
	addrHash := sha256.Sum256([]byte(addrInput))
	contractAddress := fmt.Sprintf("lgm1contract%x", addrHash[:20])

	// Store contract with current BVM version stamp
	contract := types.DeployedContract{
		Address:         contractAddress,
		CodeHash:        codeHash,
		Creator:         msg.Deployer,
		DeployedAtBlock: blockHeight,
		BytecodeSize:    uint64(len(msg.Bytecode)),
		BvmVersion:      params.CurrentBvmVersion,
	}
	m.SetContract(ctx, contract)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"bvm_deploy_contract",
		sdk.NewAttribute("address", contractAddress),
		sdk.NewAttribute("deployer", msg.Deployer),
		sdk.NewAttribute("code_hash", codeHash),
	))

	return &types.MsgDeployContractResponse{
		ContractAddress: contractAddress,
	}, nil
}

// CallContract executes contract bytecode in the BVM interpreter.
func (m msgServer) CallContract(goCtx context.Context, msg *types.MsgCallContract) (*types.MsgCallContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify contract exists
	contract, found := m.GetContract(ctx, msg.ContractAddress)
	if !found {
		return nil, types.ErrContractNotFound
	}

	// Get bytecode
	code, codeFound := m.GetCode(ctx, contract.CodeHash)
	if !codeFound {
		return nil, types.ErrCodeNotFound
	}

	// Use InputData directly as calldata
	callData := msg.InputData

	// Build caller address (20 bytes from bech32)
	callerAddr, _ := sdk.AccAddressFromBech32(msg.Caller)
	callerBytes := make([]byte, 20)
	copy(callerBytes, callerAddr)

	// Parse and transfer Value (payable contract support)
	callValue := new(big.Int)
	if msg.Value != "" && msg.Value != "0" {
		if v, ok := new(big.Int).SetString(msg.Value, 10); ok && v.Sign() > 0 {
			callValue = v
			// Transfer from caller to BVM module account
			if m.bankKeeper != nil {
				valueCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(callValue)))
				if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, callerAddr, types.ModuleName, valueCoins); err != nil {
					return nil, fmt.Errorf("insufficient funds for contract value: %w", err)
				}
			}
		}
	}

	// Resolve caller's DID from x/auth (not the contract's AgentDID)
	var callerDID string
	if m.authKeeper != nil {
		if did, found := m.authKeeper.GetAccountDID(ctx, msg.Caller); found {
			callerDID = did
		}
	}

	// Load session capabilities if available.
	// C-1 FIX: checkCapability() now defaults to DENY when Capabilities == nil.
	// Authenticated callers (identity/operational key) who have no session key
	// must receive an all-permissions capability set so they aren't blocked.
	var caps *vm.SessionCapabilities
	if m.authKeeper != nil && callerDID != "" {
		if sc, found := m.authKeeper.GetSessionCapabilities(ctx, msg.Caller, uint64(ctx.BlockHeight())); found {
			// Session key: use its restricted capabilities
			caps = &vm.SessionCapabilities{
				CanTransfer:     sc.CanTransfer,
				CanStake:        sc.CanStake,
				CanSubmitClaims: sc.CanSubmitClaims,
				CanVote:         sc.CanVote,
			}
		} else {
			// No session key — caller authenticated with identity/operational key.
			// Grant full capabilities (they own the account).
			caps = &vm.SessionCapabilities{
				CanTransfer:     true,
				CanStake:        true,
				CanSubmitClaims: true,
				CanVote:         true,
			}
		}
	} else if callerDID != "" {
		// authKeeper not wired but caller has DID — grant full capabilities
		caps = &vm.SessionCapabilities{
			CanTransfer:     true,
			CanStake:        true,
			CanSubmitClaims: true,
			CanVote:         true,
		}
	}
	// If callerDID == "" (anonymous), caps stays nil → all agent opcodes denied

	// Contract address as bytes
	contractBytes := []byte(msg.ContractAddress)

	// Build governance gas schedule overrides
	var gasSchedule *vm.GasSchedule
	if govSchedule := m.GetGasSchedule(ctx); govSchedule != nil {
		overrides := make(map[byte]uint64)
		for mnemonic, cost := range govSchedule {
			if opcode, ok := vm.OpcodeByMnemonic(mnemonic); ok {
				overrides[opcode] = cost
			}
		}
		if len(overrides) > 0 {
			gasSchedule = &vm.GasSchedule{Overrides: overrides}
		}
	}

	// Build execution context
	execCtx := &vm.ExecutionContext{
		Caller:             callerBytes,
		CallerDID:          callerDID,
		Origin:             callerBytes,
		OriginDID:          callerDID,
		CurrentContract:    contractBytes,
		BlockNumber:        uint64(ctx.BlockHeight()),
		Timestamp:          uint64(ctx.BlockTime().Unix()),
		GasPrice:           new(big.Int),
		GasLimit:           msg.GasLimit,
		Value:              callValue,
		CallData:           callData,
		Mode:               vm.ModeCall,
		Bytecode:           code.Bytecode,
		ChainID:            big.NewInt(2521), // Legible chain ID
		Capabilities:       caps,
		ContractBvmVersion: contract.BvmVersion,
		GasSchedule:        gasSchedule,
	}

	// Create state adapter: wrap new KVStoreService API for VM compatibility
	kvStore := m.storeService.OpenKVStore(ctx)
	var stateDB *vm.CosmosStateDB
	if m.bankKeeper != nil {
		bq := &balanceQuerierAdapter{bk: m.bankKeeper, ctx: ctx}
		stateDB = vm.NewCosmosStateDBWithBalance(&kvStoreAdapter{store: kvStore}, bq)
	} else {
		stateDB = vm.NewCosmosStateDB(&kvStoreAdapter{store: kvStore})
	}

	// Build host functions wired to the keeper for agent opcodes
	host := m.Keeper.buildHost(ctx, stateDB)

	// Execute
	interp := vm.NewInterpreter()
	result := interp.Execute(code.Bytecode, execCtx, stateDB, host)

	// Apply state changes on success
	if result.Success {
		for _, change := range result.StateChanges {
			if change.Type == vm.StateChangeStorage {
				keyHex := change.Key
				valBytes := ""
				if change.NewValue != nil {
					b32 := vm.WordToBytes32(change.NewValue)
					valBytes = hex.EncodeToString(b32[:])
				}
				m.SetContractState(ctx, msg.ContractAddress, keyHex, valBytes)
			}
		}
	}

	// If execution failed and value was transferred, refund the caller
	if !result.Success && callValue.Sign() > 0 && m.bankKeeper != nil {
		refundCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(callValue)))
		_ = m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, callerAddr, refundCoins)
	}

	// Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"bvm_call_contract",
		sdk.NewAttribute("contract", msg.ContractAddress),
		sdk.NewAttribute("caller", msg.Caller),
		sdk.NewAttribute("value", callValue.String()),
		sdk.NewAttribute("gas_used", fmt.Sprintf("%d", result.GasUsed)),
		sdk.NewAttribute("success", fmt.Sprintf("%v", result.Success)),
	))

	if !result.Success {
		return nil, fmt.Errorf("contract execution failed: gas_used=%d", result.GasUsed)
	}

	return &types.MsgCallContractResponse{
		ReturnData: result.ReturnData,
		GasUsed:    result.GasUsed,
	}, nil
}

// ScheduleExecution implements the proto MsgServer ScheduleExecution RPC.
// It schedules a future contract execution based on a schedule condition.
func (m msgServer) ScheduleExecution(goCtx context.Context, msg *types.MsgScheduleExecution) (*types.MsgScheduleExecutionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	// Verify contract exists
	contract, found := m.GetContract(ctx, msg.ContractAddress)
	if !found {
		return nil, types.ErrContractNotFound
	}

	// Verify scheduler is the contract creator
	if msg.Scheduler != contract.Creator {
		return nil, types.ErrNotContractCreator
	}

	// Determine target block from condition
	var executeAtBlock uint64
	if cond := msg.Condition; cond != nil {
		if bi := cond.GetBlockInterval(); bi != nil {
			executeAtBlock = bi.StartHeight
			if executeAtBlock == 0 {
				executeAtBlock = currentBlock + bi.EveryNBlocks
			}
		} else if tc := cond.GetTimeCondition(); tc != nil {
			// Approximate: schedule at next block
			executeAtBlock = currentBlock + 1
		} else if sc := cond.GetStateCondition(); sc != nil {
			// State conditions are checked each block; schedule at next block
			executeAtBlock = currentBlock + 1
		}
	}
	if executeAtBlock == 0 {
		executeAtBlock = currentBlock + 1
	}

	if executeAtBlock <= currentBlock {
		return nil, types.ErrScheduleInPast
	}

	// Check schedule limit
	scheduleCount := m.CountContractSchedules(ctx, msg.ContractAddress)
	if scheduleCount >= params.MaxSchedulesPerContract {
		return nil, types.ErrScheduleLimitReached
	}

	// Generate schedule ID
	counter := m.GetNextScheduleId(ctx)
	scheduleId := fmt.Sprintf("sched-%d-%d", currentBlock, counter)

	schedule := types.ContractSchedule{
		ScheduleId:      scheduleId,
		ContractAddress: msg.ContractAddress,
		Caller:          msg.Scheduler,
		Method:          "", // InputData used directly
		Payload:         string(msg.InputData),
		ExecuteAtBlock:  executeAtBlock,
		MaxGas:          params.CallBaseGasCost,
		Executed:        false,
		Cancelled:       false,
	}
	m.SetSchedule(ctx, schedule)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"bvm_schedule_execution",
		sdk.NewAttribute("schedule_id", scheduleId),
		sdk.NewAttribute("contract", msg.ContractAddress),
		sdk.NewAttribute("scheduler", msg.Scheduler),
		sdk.NewAttribute("execute_at_block", fmt.Sprintf("%d", executeAtBlock)),
	))

	return &types.MsgScheduleExecutionResponse{
		ScheduleId: scheduleId,
	}, nil
}

// ScheduleContract schedules a future contract execution (hand-written message type).
func (m msgServer) ScheduleContract(goCtx context.Context, msg *types.MsgScheduleContract) (*types.MsgScheduleContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	// Verify contract exists
	contract, found := m.GetContract(ctx, msg.ContractAddress)
	if !found {
		return nil, types.ErrContractNotFound
	}

	// Verify caller is creator or agent
	if msg.Caller != contract.Creator {
		return nil, types.ErrNotContractCreator
	}

	// Verify execute block is in the future
	if msg.ExecuteAtBlock <= currentBlock {
		return nil, types.ErrScheduleInPast
	}

	// Check schedule limit
	scheduleCount := m.CountContractSchedules(ctx, msg.ContractAddress)
	if scheduleCount >= params.MaxSchedulesPerContract {
		return nil, types.ErrScheduleLimitReached
	}

	// Generate schedule ID
	counter := m.GetNextScheduleId(ctx)
	scheduleId := fmt.Sprintf("sched-%d-%d", currentBlock, counter)

	schedule := types.ContractSchedule{
		ScheduleId:      scheduleId,
		ContractAddress: msg.ContractAddress,
		Caller:          msg.Caller,
		Method:          msg.Method,
		Payload:         msg.Payload,
		ExecuteAtBlock:  msg.ExecuteAtBlock,
		MaxGas:          msg.MaxGas,
		Executed:        false,
		Cancelled:       false,
	}
	m.SetSchedule(ctx, schedule)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"bvm_schedule_contract",
		sdk.NewAttribute("schedule_id", scheduleId),
		sdk.NewAttribute("contract", msg.ContractAddress),
		sdk.NewAttribute("method", msg.Method),
		sdk.NewAttribute("execute_at_block", fmt.Sprintf("%d", msg.ExecuteAtBlock)),
	))

	return &types.MsgScheduleContractResponse{
		ScheduleId: scheduleId,
	}, nil
}

// CancelSchedule cancels a pending scheduled execution.
func (m msgServer) CancelSchedule(goCtx context.Context, msg *types.MsgCancelSchedule) (*types.MsgCancelScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	schedule, found := m.GetSchedule(ctx, msg.ScheduleId)
	if !found {
		return nil, types.ErrScheduleNotFound
	}

	// Verify not already executed
	if schedule.Executed {
		return nil, types.ErrExecutionFailed
	}

	// Verify not already cancelled
	if schedule.Cancelled {
		return nil, types.ErrScheduleNotFound
	}

	// Verify caller is the schedule owner
	if msg.Caller != schedule.Caller {
		return nil, types.ErrScheduleNotOwner
	}

	// Remove old block index entry before updating
	store := m.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.ScheduleBlockIndexKey(schedule.ExecuteAtBlock, schedule.ScheduleId)); err != nil {
		return nil, fmt.Errorf("failed to delete schedule block index: %w", err)
	}

	schedule.Cancelled = true
	m.SetSchedule(ctx, schedule)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"bvm_cancel_schedule",
		sdk.NewAttribute("schedule_id", msg.ScheduleId),
		sdk.NewAttribute("caller", msg.Caller),
	))

	return &types.MsgCancelScheduleResponse{}, nil
}

// UpdateContractState updates contract state (governance only).
func (m msgServer) UpdateContractState(goCtx context.Context, msg *types.MsgUpdateContractState) (*types.MsgUpdateContractStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify authority
	if msg.Authority != m.GetAuthority() {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", m.GetAuthority(), msg.Authority)
	}

	// Verify contract exists
	_, found := m.GetContract(ctx, msg.ContractAddress)
	if !found {
		return nil, types.ErrContractNotFound
	}

	m.SetContractState(ctx, msg.ContractAddress, msg.Key, msg.Value)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"bvm_update_contract_state",
		sdk.NewAttribute("contract", msg.ContractAddress),
		sdk.NewAttribute("key", msg.Key),
		sdk.NewAttribute("authority", msg.Authority),
	))

	return &types.MsgUpdateContractStateResponse{}, nil
}

// ExecutePendingSchedules executes all schedules due at the current block.
// Called from BeginBlock.
func (k Keeper) ExecutePendingSchedules(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())
	schedules := k.GetPendingSchedules(ctx, currentBlock)

	for _, schedule := range schedules {
		// Remove old block index entry
		kvStore := k.storeService.OpenKVStore(ctx)
		_ = kvStore.Delete(types.ScheduleBlockIndexKey(schedule.ExecuteAtBlock, schedule.ScheduleId))

		// Execute bytecode via the VM
		success := false
		var gasUsed uint64

		contract, found := k.GetContract(ctx, schedule.ContractAddress)
		if found {
			code, codeFound := k.GetCode(ctx, contract.CodeHash)
			if codeFound {
				callData := []byte(schedule.Method)
				if schedule.Payload != "" {
					callData = append(callData, []byte(schedule.Payload)...)
				}
				callerAddr, _ := sdk.AccAddressFromBech32(schedule.Caller)
				callerBytes := make([]byte, 20)
				copy(callerBytes, callerAddr)

				// Resolve caller's DID from x/auth (not the contract's AgentDID)
				var schedCallerDID string
				if k.authKeeper != nil {
					if did, found := k.authKeeper.GetAccountDID(ctx, schedule.Caller); found {
						schedCallerDID = did
					}
				}

				// ZD-B01 FIX: Scheduled executions must have explicit Capabilities,
				// otherwise the C-1 secure default (nil = deny all) silently blocks
				// all agent opcodes (SUBMIT_CLAIM, STAKE, UNSTAKE, etc.).
				// Scheduled executions run with full authority of the original caller.
				fullCaps := &vm.SessionCapabilities{
					CanStake:        true,
					CanTransfer:     true,
					CanSubmitClaims: true,
					CanVote:         true,
				}

				// Build governance gas schedule for scheduled execution
				var schedGasSchedule *vm.GasSchedule
				if govSched := k.GetGasSchedule(ctx); govSched != nil {
					overrides := make(map[byte]uint64)
					for mnemonic, cost := range govSched {
						if opcode, ok := vm.OpcodeByMnemonic(mnemonic); ok {
							overrides[opcode] = cost
						}
					}
					if len(overrides) > 0 {
						schedGasSchedule = &vm.GasSchedule{Overrides: overrides}
					}
				}

				execCtx := &vm.ExecutionContext{
					Caller:             callerBytes,
					CallerDID:          schedCallerDID,
					Origin:             callerBytes,
					OriginDID:          schedCallerDID,
					CurrentContract:    []byte(schedule.ContractAddress),
					BlockNumber:        currentBlock,
					Timestamp:          uint64(ctx.BlockTime().Unix()),
					GasPrice:           new(big.Int),
					GasLimit:           schedule.MaxGas,
					Value:              new(big.Int),
					CallData:           callData,
					Mode:               vm.ModeCall,
					Bytecode:           code.Bytecode,
					ChainID:            big.NewInt(2521),
					Capabilities:       fullCaps,
					ContractBvmVersion: contract.BvmVersion,
					GasSchedule:        schedGasSchedule,
				}

				stateDB := vm.NewCosmosStateDB(&kvStoreAdapter{store: kvStore})
				host := k.buildHost(ctx, stateDB)
				interp := vm.NewInterpreter()
				result := interp.Execute(code.Bytecode, execCtx, stateDB, host)

				success = result.Success
				gasUsed = result.GasUsed

				// Apply state changes on success
				if result.Success {
					for _, change := range result.StateChanges {
						if change.Type == vm.StateChangeStorage {
							valBytes := ""
							if change.NewValue != nil {
								b32 := vm.WordToBytes32(change.NewValue)
								valBytes = hex.EncodeToString(b32[:])
							}
							k.SetContractState(ctx, schedule.ContractAddress, change.Key, valBytes)
						}
					}
				}
			}
		}

		schedule.Executed = true
		k.SetSchedule(ctx, schedule)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"bvm_execute_schedule",
			sdk.NewAttribute("schedule_id", schedule.ScheduleId),
			sdk.NewAttribute("contract", schedule.ContractAddress),
			sdk.NewAttribute("method", schedule.Method),
			sdk.NewAttribute("block", fmt.Sprintf("%d", currentBlock)),
			sdk.NewAttribute("success", fmt.Sprintf("%v", success)),
			sdk.NewAttribute("gas_used", fmt.Sprintf("%d", gasUsed)),
		))
	}
}
