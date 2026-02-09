package types

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- DeployedContract ---

// DeployedContract represents a contract deployed to the BVM.
type DeployedContract struct {
	Address         string `json:"address"`
	CodeHash        string `json:"code_hash"`
	Creator         string `json:"creator"`
	DeployedAtBlock uint64 `json:"deployed_at_block"`
	IsAgent         bool   `json:"is_agent"`
	AgentDID        string `json:"agent_did,omitempty"`
	BytecodeSize    uint64 `json:"bytecode_size"`
	Label           string `json:"label,omitempty"`
}

func (m *DeployedContract) Reset()         { *m = DeployedContract{} }
func (m *DeployedContract) String() string { bz, _ := json.Marshal(m); return string(bz) }
func (m *DeployedContract) ProtoMessage()  {}

// --- ContractCode ---

// ContractCode stores the bytecode for a deployed contract, keyed by code hash.
type ContractCode struct {
	CodeHash string `json:"code_hash"`
	Bytecode []byte `json:"bytecode"`
	RefCount uint64 `json:"ref_count"`
}

func (m *ContractCode) Reset()         { *m = ContractCode{} }
func (m *ContractCode) String() string { bz, _ := json.Marshal(m); return string(bz) }
func (m *ContractCode) ProtoMessage()  {}

// --- ContractStateEntry ---

// ContractStateEntry represents a key-value state entry for a contract.
type ContractStateEntry struct {
	ContractAddress string `json:"contract_address"`
	Key             string `json:"key"`
	Value           string `json:"value"`
}

// --- ContractSchedule ---

// ContractSchedule represents a scheduled contract execution.
type ContractSchedule struct {
	ScheduleId      string `json:"schedule_id"`
	ContractAddress string `json:"contract_address"`
	Caller          string `json:"caller"`
	Method          string `json:"method"`
	Payload         string `json:"payload,omitempty"`
	ExecuteAtBlock  uint64 `json:"execute_at_block"`
	MaxGas          uint64 `json:"max_gas"`
	Executed        bool   `json:"executed"`
	Cancelled       bool   `json:"cancelled"`
}

func (m *ContractSchedule) Reset()         { *m = ContractSchedule{} }
func (m *ContractSchedule) String() string { bz, _ := json.Marshal(m); return string(bz) }
func (m *ContractSchedule) ProtoMessage()  {}

// --- ContractCallRecord ---

// ContractCallRecord stores the result of a contract call.
type ContractCallRecord struct {
	TxHash          string `json:"tx_hash"`
	ContractAddress string `json:"contract_address"`
	Caller          string `json:"caller"`
	Method          string `json:"method"`
	GasUsed         uint64 `json:"gas_used"`
	Success         bool   `json:"success"`
	ReturnData      string `json:"return_data,omitempty"`
	Error           string `json:"error,omitempty"`
	BlockNumber     uint64 `json:"block_number"`
}

// --- Params ---

// Params defines the bvm module parameters.
type Params struct {
	MaxBytecodeSize         uint64 `json:"max_bytecode_size"`
	MaxStateEntries         uint64 `json:"max_state_entries"`
	MaxSchedulesPerContract uint64 `json:"max_schedules_per_contract"`
	DeployGasCost           uint64 `json:"deploy_gas_cost"`
	CallBaseGasCost         uint64 `json:"call_base_gas_cost"`
	StorageWriteGasCost     uint64 `json:"storage_write_gas_cost"`
	StorageReadGasCost      uint64 `json:"storage_read_gas_cost"`
}

// DefaultParams returns the default bvm module parameters.
func DefaultParams() Params {
	return Params{
		MaxBytecodeSize:         1_048_576, // 1 MB
		MaxStateEntries:         100_000,
		MaxSchedulesPerContract: 100,
		DeployGasCost:           200_000,
		CallBaseGasCost:         21_000,
		StorageWriteGasCost:     5_000,
		StorageReadGasCost:      2_000,
	}
}

// Validate validates the bvm module parameters.
func (p Params) Validate() error {
	if p.MaxBytecodeSize == 0 {
		return fmt.Errorf("max bytecode size must be positive")
	}
	if p.MaxStateEntries == 0 {
		return fmt.Errorf("max state entries must be positive")
	}
	return nil
}

// --- GenesisState ---

// GenesisState defines the bvm module's genesis state.
type GenesisState struct {
	Params    Params               `json:"params"`
	Contracts []DeployedContract   `json:"contracts"`
	Code      []ContractCode       `json:"code"`
	Schedules []ContractSchedule   `json:"schedules"`
	State     []ContractStateEntry `json:"state"`
}

// DefaultGenesisState returns the default genesis state for the bvm module.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Contracts: []DeployedContract{},
		Code:      []ContractCode{},
		Schedules: []ContractSchedule{},
		State:     []ContractStateEntry{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	for i, c := range gs.Contracts {
		if c.Address == "" {
			return fmt.Errorf("contract %d: empty address", i)
		}
		if c.CodeHash == "" {
			return fmt.Errorf("contract %d: empty code hash", i)
		}
		if c.Creator == "" {
			return fmt.Errorf("contract %d: empty creator", i)
		}
	}
	for i, code := range gs.Code {
		if code.CodeHash == "" {
			return fmt.Errorf("code %d: empty code hash", i)
		}
	}
	return nil
}

// --- Message Types ---

// MsgDeployContract deploys a new contract to the BVM.
type MsgDeployContract struct {
	Creator  string `json:"creator"`
	Bytecode []byte `json:"bytecode"`
	Label    string `json:"label,omitempty"`
	IsAgent  bool   `json:"is_agent"`
	AgentDID string `json:"agent_did,omitempty"`
}

func (m *MsgDeployContract) Reset()         { *m = MsgDeployContract{} }
func (m *MsgDeployContract) String() string { bz, _ := json.Marshal(m); return string(bz) }
func (m *MsgDeployContract) ProtoMessage()  {}

func (m *MsgDeployContract) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if len(m.Bytecode) == 0 {
		return fmt.Errorf("bytecode cannot be empty")
	}
	if m.IsAgent && m.AgentDID == "" {
		return fmt.Errorf("agent contracts require agent_did")
	}
	return nil
}

func (m *MsgDeployContract) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Creator)
	return []sdk.AccAddress{addr}
}

// MsgDeployContractResponse is the response for MsgDeployContract.
type MsgDeployContractResponse struct {
	Address  string `json:"address"`
	CodeHash string `json:"code_hash"`
}

// MsgCallContract calls an existing contract.
type MsgCallContract struct {
	Caller          string `json:"caller"`
	ContractAddress string `json:"contract_address"`
	Method          string `json:"method"`
	Payload         string `json:"payload,omitempty"`
	GasLimit        uint64 `json:"gas_limit"`
}

func (m *MsgCallContract) Reset()         { *m = MsgCallContract{} }
func (m *MsgCallContract) String() string { bz, _ := json.Marshal(m); return string(bz) }
func (m *MsgCallContract) ProtoMessage()  {}

func (m *MsgCallContract) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Caller); err != nil {
		return fmt.Errorf("invalid caller address: %w", err)
	}
	if m.ContractAddress == "" {
		return fmt.Errorf("contract address cannot be empty")
	}
	if m.Method == "" {
		return fmt.Errorf("method cannot be empty")
	}
	if m.GasLimit == 0 {
		return fmt.Errorf("gas limit must be positive")
	}
	return nil
}

func (m *MsgCallContract) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Caller)
	return []sdk.AccAddress{addr}
}

// MsgCallContractResponse is the response for MsgCallContract.
type MsgCallContractResponse struct {
	ReturnData string `json:"return_data,omitempty"`
	GasUsed    uint64 `json:"gas_used"`
	Success    bool   `json:"success"`
}

// MsgScheduleContract schedules a future contract execution.
type MsgScheduleContract struct {
	Caller          string `json:"caller"`
	ContractAddress string `json:"contract_address"`
	Method          string `json:"method"`
	Payload         string `json:"payload,omitempty"`
	ExecuteAtBlock  uint64 `json:"execute_at_block"`
	MaxGas          uint64 `json:"max_gas"`
}

func (m *MsgScheduleContract) Reset()         { *m = MsgScheduleContract{} }
func (m *MsgScheduleContract) String() string { bz, _ := json.Marshal(m); return string(bz) }
func (m *MsgScheduleContract) ProtoMessage()  {}

func (m *MsgScheduleContract) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Caller); err != nil {
		return fmt.Errorf("invalid caller address: %w", err)
	}
	if m.ContractAddress == "" {
		return fmt.Errorf("contract address cannot be empty")
	}
	if m.Method == "" {
		return fmt.Errorf("method cannot be empty")
	}
	if m.ExecuteAtBlock == 0 {
		return fmt.Errorf("execute_at_block must be positive")
	}
	if m.MaxGas == 0 {
		return fmt.Errorf("max_gas must be positive")
	}
	return nil
}

func (m *MsgScheduleContract) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Caller)
	return []sdk.AccAddress{addr}
}

// MsgScheduleContractResponse is the response for MsgScheduleContract.
type MsgScheduleContractResponse struct {
	ScheduleId string `json:"schedule_id"`
}

// MsgCancelSchedule cancels a pending scheduled execution.
type MsgCancelSchedule struct {
	Caller     string `json:"caller"`
	ScheduleId string `json:"schedule_id"`
}

func (m *MsgCancelSchedule) Reset()         { *m = MsgCancelSchedule{} }
func (m *MsgCancelSchedule) String() string { bz, _ := json.Marshal(m); return string(bz) }
func (m *MsgCancelSchedule) ProtoMessage()  {}

func (m *MsgCancelSchedule) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Caller); err != nil {
		return fmt.Errorf("invalid caller address: %w", err)
	}
	if m.ScheduleId == "" {
		return fmt.Errorf("schedule_id cannot be empty")
	}
	return nil
}

func (m *MsgCancelSchedule) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Caller)
	return []sdk.AccAddress{addr}
}

// MsgCancelScheduleResponse is the response for MsgCancelSchedule.
type MsgCancelScheduleResponse struct{}

// MsgUpdateContractState is a governance-only message to update contract state.
type MsgUpdateContractState struct {
	Authority       string `json:"authority"`
	ContractAddress string `json:"contract_address"`
	Key             string `json:"key"`
	Value           string `json:"value"`
}

func (m *MsgUpdateContractState) Reset()         { *m = MsgUpdateContractState{} }
func (m *MsgUpdateContractState) String() string { bz, _ := json.Marshal(m); return string(bz) }
func (m *MsgUpdateContractState) ProtoMessage()  {}

func (m *MsgUpdateContractState) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if m.ContractAddress == "" {
		return fmt.Errorf("contract address cannot be empty")
	}
	if m.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	return nil
}

func (m *MsgUpdateContractState) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// MsgUpdateContractStateResponse is the response for MsgUpdateContractState.
type MsgUpdateContractStateResponse struct{}

// --- MsgServer Interface ---

// MsgServer defines the bvm module's msg server interface.
type MsgServer interface {
	DeployContract(context.Context, *MsgDeployContract) (*MsgDeployContractResponse, error)
	CallContract(context.Context, *MsgCallContract) (*MsgCallContractResponse, error)
	ScheduleContract(context.Context, *MsgScheduleContract) (*MsgScheduleContractResponse, error)
	CancelSchedule(context.Context, *MsgCancelSchedule) (*MsgCancelScheduleResponse, error)
	UpdateContractState(context.Context, *MsgUpdateContractState) (*MsgUpdateContractStateResponse, error)
}
