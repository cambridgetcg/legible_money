package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// -----------------------------------------------------------------------
// Core domain types -- type aliases and constants
// Struct definitions for TimeCondition, LogicCondition, CompoundCondition,
// ScheduleCondition, ScheduleProcess, Params, and GenesisState are provided
// by the proto-generated .pb.go files (state.pb.go and genesis.pb.go).
// Msg structs, Response structs, MsgServer interface, gRPC service descriptor,
// handler functions, and RegisterMsgServer are provided by tx.pb.go and
// tx_grpc.pb.go.
// Only methods, type aliases, constants, and non-proto types are kept here.
// -----------------------------------------------------------------------

// ProcessStatus represents the lifecycle of a schedule process.
type ProcessStatus = string

const (
	StatusActive    ProcessStatus = "active"
	StatusPaused    ProcessStatus = "paused"
	StatusCompleted ProcessStatus = "completed"
	StatusCancelled ProcessStatus = "cancelled"
	StatusExhausted ProcessStatus = "exhausted" // ran out of prepaid fee
)

// TimeConditionType defines how time-based conditions work.
type TimeConditionType = string

const (
	TimeAbsolute TimeConditionType = "absolute" // fire at specific block
	TimeRelative TimeConditionType = "relative" // fire N blocks from creation
	TimePeriodic TimeConditionType = "periodic" // fire every N blocks
)

// LogicConditionType defines state-based conditions.
type LogicConditionType = string

const (
	LogicStateCompare     LogicConditionType = "state_compare"
	LogicThresholdCrossed LogicConditionType = "threshold_crossed"
)

// CompoundOperator for combining conditions.
type CompoundOperator = string

const (
	CompoundAnd CompoundOperator = "and"
	CompoundOr  CompoundOperator = "or"
)

// Depth returns the nesting depth of the compound condition.
func (cc *CompoundCondition) Depth() int {
	maxDepth := 0
	for _, c := range cc.Conditions {
		if c.CompoundCondition != nil {
			d := c.CompoundCondition.Depth()
			if d > maxDepth {
				maxDepth = d
			}
		}
	}
	return maxDepth + 1
}

// Validate performs parameter validation.
func (p *Params) Validate() error {
	if p.MaxActivePerAccount == 0 {
		return fmt.Errorf("max_active_per_account must be positive")
	}
	if p.MaxGasPerBlock == 0 {
		return fmt.Errorf("max_gas_per_block must be positive")
	}
	if p.MinIntervalBlocks == 0 {
		return fmt.Errorf("min_interval_blocks must be positive")
	}
	if p.MinPrepaidExecutions == 0 {
		return fmt.Errorf("min_prepaid_executions must be positive")
	}
	if p.CleanupIntervalBlocks == 0 {
		return fmt.Errorf("cleanup_interval_blocks must be positive")
	}
	return nil
}

// -----------------------------------------------------------------------
// GetSigners and ValidateBasic methods for proto-generated Msg types
// -----------------------------------------------------------------------

func (msg *MsgCreateSchedule) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCreateSchedule) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.TargetAddress != "" {
		if _, err := sdk.AccAddressFromBech32(msg.TargetAddress); err != nil {
			return fmt.Errorf("invalid target address: %w", err)
		}
	}
	return nil
}

func (msg *MsgPauseSchedule) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgPauseSchedule) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.ProcessId == "" {
		return fmt.Errorf("process_id cannot be empty")
	}
	return nil
}

func (msg *MsgResumeSchedule) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgResumeSchedule) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.ProcessId == "" {
		return fmt.Errorf("process_id cannot be empty")
	}
	return nil
}

func (msg *MsgCancelSchedule) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCancelSchedule) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.ProcessId == "" {
		return fmt.Errorf("process_id cannot be empty")
	}
	return nil
}

func (msg *MsgAddFee) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAddFee) ValidateBasic() error {
	if msg.Creator == "" {
		return fmt.Errorf("creator cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.ProcessId == "" {
		return fmt.Errorf("process_id cannot be empty")
	}
	if msg.Amount == "" {
		return fmt.Errorf("amount cannot be empty")
	}
	return nil
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return msg.Params.Validate()
}
