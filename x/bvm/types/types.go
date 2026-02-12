package types

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Validate validates the bvm module parameters.
func (p *Params) Validate() error {
	if p.MaxBytecodeSize == 0 {
		return fmt.Errorf("max bytecode size must be positive")
	}
	if p.MaxStateEntries == 0 {
		return fmt.Errorf("max state entries must be positive")
	}
	return nil
}

// --- Extension methods for proto-generated message types ---

// ValidateBasic performs stateless validation on MsgDeployContract.
func (m *MsgDeployContract) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Deployer); err != nil {
		return fmt.Errorf("invalid deployer address: %w", err)
	}
	if len(m.Bytecode) == 0 {
		return fmt.Errorf("bytecode cannot be empty")
	}
	return nil
}

// GetSigners returns the expected signers for MsgDeployContract.
func (m *MsgDeployContract) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Deployer)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation on MsgCallContract.
func (m *MsgCallContract) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Caller); err != nil {
		return fmt.Errorf("invalid caller address: %w", err)
	}
	if m.ContractAddress == "" {
		return fmt.Errorf("contract address cannot be empty")
	}
	if m.GasLimit == 0 {
		return fmt.Errorf("gas limit must be positive")
	}
	return nil
}

// GetSigners returns the expected signers for MsgCallContract.
func (m *MsgCallContract) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Caller)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation on MsgScheduleExecution.
func (m *MsgScheduleExecution) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Scheduler); err != nil {
		return fmt.Errorf("invalid scheduler address: %w", err)
	}
	if m.ContractAddress == "" {
		return fmt.Errorf("contract address cannot be empty")
	}
	return nil
}

// GetSigners returns the expected signers for MsgScheduleExecution.
func (m *MsgScheduleExecution) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Scheduler)
	return []sdk.AccAddress{addr}
}

// --- Hand-written message types (not in proto definitions) ---

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

// ExtendedMsgServer extends the proto-generated MsgServer with hand-written
// message handlers that are not defined in the proto service.
type ExtendedMsgServer interface {
	MsgServer
	ScheduleContract(context.Context, *MsgScheduleContract) (*MsgScheduleContractResponse, error)
	CancelSchedule(context.Context, *MsgCancelSchedule) (*MsgCancelScheduleResponse, error)
	UpdateContractState(context.Context, *MsgUpdateContractState) (*MsgUpdateContractStateResponse, error)
}
