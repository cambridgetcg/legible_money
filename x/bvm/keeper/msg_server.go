package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/bvm/types"
	"github.com/legible-money/legible/x/bvm/vm"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns a MsgServer implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

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
	addrInput := fmt.Sprintf("%s/%d/%d", msg.Creator, blockHeight, nonce)
	addrHash := sha256.Sum256([]byte(addrInput))
	contractAddress := fmt.Sprintf("lgm1contract%x", addrHash[:20])

	// Store contract
	contract := types.DeployedContract{
		Address:         contractAddress,
		CodeHash:        codeHash,
		Creator:         msg.Creator,
		DeployedAtBlock: blockHeight,
		IsAgent:         msg.IsAgent,
		AgentDID:        msg.AgentDID,
		BytecodeSize:    uint64(len(msg.Bytecode)),
		Label:           msg.Label,
	}
	m.SetContract(ctx, contract)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"bvm_deploy_contract",
		sdk.NewAttribute("address", contractAddress),
		sdk.NewAttribute("creator", msg.Creator),
		sdk.NewAttribute("code_hash", codeHash),
		sdk.NewAttribute("label", msg.Label),
	))

	return &types.MsgDeployContractResponse{
		Address:  contractAddress,
		CodeHash: codeHash,
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

	// Build calldata from method + payload
	callData := []byte(msg.Method)
	if msg.Payload != "" {
		callData = append(callData, []byte(msg.Payload)...)
	}

	// Build caller address (20 bytes from bech32)
	callerAddr, _ := sdk.AccAddressFromBech32(msg.Caller)
	callerBytes := make([]byte, 20)
	copy(callerBytes, callerAddr)

	// Contract address as bytes
	contractBytes := []byte(msg.ContractAddress)

	// Build execution context
	execCtx := &vm.ExecutionContext{
		Caller:          callerBytes,
		CallerDID:       contract.AgentDID,
		Origin:          callerBytes,
		CurrentContract: contractBytes,
		BlockNumber:     uint64(ctx.BlockHeight()),
		Timestamp:       uint64(ctx.BlockTime().Unix()),
		GasPrice:        new(big.Int),
		GasLimit:        msg.GasLimit,
		Value:           new(big.Int),
		CallData:        callData,
		Mode:            vm.ModeCall,
		Bytecode:        code.Bytecode,
		ChainID:         big.NewInt(2521), // Legible chain ID
	}

	// Create state adapter
	store := ctx.KVStore(m.storeKey)
	stateDB := vm.NewCosmosStateDB(store)

	// Execute
	interp := vm.NewInterpreter()
	result := interp.Execute(code.Bytecode, execCtx, stateDB, nil)

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

	// Encode return data
	returnDataHex := ""
	if result.ReturnData != nil {
		returnDataHex = hex.EncodeToString(result.ReturnData)
	}

	// Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"bvm_call_contract",
		sdk.NewAttribute("contract", msg.ContractAddress),
		sdk.NewAttribute("caller", msg.Caller),
		sdk.NewAttribute("method", msg.Method),
		sdk.NewAttribute("gas_used", fmt.Sprintf("%d", result.GasUsed)),
		sdk.NewAttribute("success", fmt.Sprintf("%v", result.Success)),
	))

	return &types.MsgCallContractResponse{
		ReturnData: returnDataHex,
		GasUsed:    result.GasUsed,
		Success:    result.Success,
	}, nil
}

// ScheduleContract schedules a future contract execution.
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
	store := ctx.KVStore(m.storeKey)
	store.Delete(types.ScheduleBlockIndexKey(schedule.ExecuteAtBlock, schedule.ScheduleId))

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
		kvStore := ctx.KVStore(k.storeKey)
		kvStore.Delete(types.ScheduleBlockIndexKey(schedule.ExecuteAtBlock, schedule.ScheduleId))

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

				execCtx := &vm.ExecutionContext{
					Caller:          callerBytes,
					CallerDID:       contract.AgentDID,
					Origin:          callerBytes,
					CurrentContract: []byte(schedule.ContractAddress),
					BlockNumber:     currentBlock,
					Timestamp:       uint64(ctx.BlockTime().Unix()),
					GasPrice:        new(big.Int),
					GasLimit:        schedule.MaxGas,
					Value:           new(big.Int),
					CallData:        callData,
					Mode:            vm.ModeCall,
					Bytecode:        code.Bytecode,
					ChainID:         big.NewInt(2521),
				}

				stateDB := vm.NewCosmosStateDB(kvStore)
				interp := vm.NewInterpreter()
				result := interp.Execute(code.Bytecode, execCtx, stateDB, nil)

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
