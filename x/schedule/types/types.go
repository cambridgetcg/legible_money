package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// -----------------------------------------------------------------------
// Core domain types — type aliases and constants
// Struct definitions for TimeCondition, LogicCondition, CompoundCondition,
// ScheduleCondition, ScheduleProcess, Params, and GenesisState are provided
// by the proto-generated .pb.go files (state.pb.go and genesis.pb.go).
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
func (p Params) Validate() error {
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
	return nil
}

// -----------------------------------------------------------------------
// Message types
// -----------------------------------------------------------------------

// MsgCreateSchedule creates a new scheduled process.
type MsgCreateSchedule struct {
	Creator          string             `json:"creator"`
	Condition        *ScheduleCondition `json:"condition"`
	MaxExecutions    uint64             `json:"max_executions"`
	FeePerExecution  string             `json:"fee_per_execution"`
	PrepaidFee       string             `json:"prepaid_fee"`
	TargetAddress    string             `json:"target_address"`
	CallData         string             `json:"call_data"`
	TransferValue    string             `json:"transfer_value"`
	LinkedEntityType string             `json:"linked_entity_type,omitempty"`
	LinkedEntityId   string             `json:"linked_entity_id,omitempty"`
}

func (msg *MsgCreateSchedule) Reset()         {}
func (msg *MsgCreateSchedule) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgCreateSchedule) ProtoMessage()  {}

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

// MsgCreateScheduleResponse is the response for MsgCreateSchedule.
type MsgCreateScheduleResponse struct {
	ProcessId string `json:"process_id"`
}

func (msg *MsgCreateScheduleResponse) Reset()         {}
func (msg *MsgCreateScheduleResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgCreateScheduleResponse) ProtoMessage()  {}

// MsgPauseSchedule pauses an active schedule.
type MsgPauseSchedule struct {
	Creator   string `json:"creator"`
	ProcessId string `json:"process_id"`
}

func (msg *MsgPauseSchedule) Reset()         {}
func (msg *MsgPauseSchedule) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgPauseSchedule) ProtoMessage()  {}

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

// MsgPauseScheduleResponse is the response for MsgPauseSchedule.
type MsgPauseScheduleResponse struct{}

func (msg *MsgPauseScheduleResponse) Reset()         {}
func (msg *MsgPauseScheduleResponse) String() string { return "" }
func (msg *MsgPauseScheduleResponse) ProtoMessage()  {}

// MsgResumeSchedule resumes a paused schedule.
type MsgResumeSchedule struct {
	Creator   string `json:"creator"`
	ProcessId string `json:"process_id"`
}

func (msg *MsgResumeSchedule) Reset()         {}
func (msg *MsgResumeSchedule) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgResumeSchedule) ProtoMessage()  {}

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

// MsgResumeScheduleResponse is the response for MsgResumeSchedule.
type MsgResumeScheduleResponse struct{}

func (msg *MsgResumeScheduleResponse) Reset()         {}
func (msg *MsgResumeScheduleResponse) String() string { return "" }
func (msg *MsgResumeScheduleResponse) ProtoMessage()  {}

// MsgCancelSchedule cancels a schedule and refunds remaining fee.
type MsgCancelSchedule struct {
	Creator   string `json:"creator"`
	ProcessId string `json:"process_id"`
}

func (msg *MsgCancelSchedule) Reset()         {}
func (msg *MsgCancelSchedule) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgCancelSchedule) ProtoMessage()  {}

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

// MsgCancelScheduleResponse is the response for MsgCancelSchedule.
type MsgCancelScheduleResponse struct {
	RefundedAmount string `json:"refunded_amount"`
}

func (msg *MsgCancelScheduleResponse) Reset()         {}
func (msg *MsgCancelScheduleResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgCancelScheduleResponse) ProtoMessage()  {}

// MsgAddFee adds prepaid fee to an existing schedule.
type MsgAddFee struct {
	Creator   string `json:"creator"`
	ProcessId string `json:"process_id"`
	Amount    string `json:"amount"` // ulgm to add
}

func (msg *MsgAddFee) Reset()         {}
func (msg *MsgAddFee) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgAddFee) ProtoMessage()  {}

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

// MsgAddFeeResponse is the response for MsgAddFee.
type MsgAddFeeResponse struct{}

func (msg *MsgAddFeeResponse) Reset()         {}
func (msg *MsgAddFeeResponse) String() string { return "" }
func (msg *MsgAddFeeResponse) ProtoMessage()  {}

// MsgUpdateParams updates module parameters (governance-gated).
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (msg *MsgUpdateParams) Reset()         {}
func (msg *MsgUpdateParams) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgUpdateParams) ProtoMessage()  {}

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
	return msg.Params.Validate()
}

// MsgUpdateParamsResponse is the response for MsgUpdateParams.
type MsgUpdateParamsResponse struct{}

func (msg *MsgUpdateParamsResponse) Reset()         {}
func (msg *MsgUpdateParamsResponse) String() string { return "" }
func (msg *MsgUpdateParamsResponse) ProtoMessage()  {}

// -----------------------------------------------------------------------
// MsgServer interface
// -----------------------------------------------------------------------

// MsgServer defines the schedule module message service.
type MsgServer interface {
	CreateSchedule(context.Context, *MsgCreateSchedule) (*MsgCreateScheduleResponse, error)
	PauseSchedule(context.Context, *MsgPauseSchedule) (*MsgPauseScheduleResponse, error)
	ResumeSchedule(context.Context, *MsgResumeSchedule) (*MsgResumeScheduleResponse, error)
	CancelSchedule(context.Context, *MsgCancelSchedule) (*MsgCancelScheduleResponse, error)
	AddFee(context.Context, *MsgAddFee) (*MsgAddFeeResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}

// -----------------------------------------------------------------------
// gRPC service descriptor (hand-written, replaces proto generation)
// -----------------------------------------------------------------------

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.schedule.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "CreateSchedule", Handler: _Msg_CreateSchedule_Handler},
		{MethodName: "PauseSchedule", Handler: _Msg_PauseSchedule_Handler},
		{MethodName: "ResumeSchedule", Handler: _Msg_ResumeSchedule_Handler},
		{MethodName: "CancelSchedule", Handler: _Msg_CancelSchedule_Handler},
		{MethodName: "AddFee", Handler: _Msg_AddFee_Handler},
		{MethodName: "UpdateParams", Handler: _Msg_UpdateParams_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/schedule/v1/tx.proto",
}

func _Msg_CreateSchedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCreateSchedule)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CreateSchedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.schedule.v1.Msg/CreateSchedule"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CreateSchedule(ctx, req.(*MsgCreateSchedule))
	})
}

func _Msg_PauseSchedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgPauseSchedule)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).PauseSchedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.schedule.v1.Msg/PauseSchedule"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).PauseSchedule(ctx, req.(*MsgPauseSchedule))
	})
}

func _Msg_ResumeSchedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgResumeSchedule)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ResumeSchedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.schedule.v1.Msg/ResumeSchedule"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ResumeSchedule(ctx, req.(*MsgResumeSchedule))
	})
}

func _Msg_CancelSchedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCancelSchedule)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CancelSchedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.schedule.v1.Msg/CancelSchedule"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CancelSchedule(ctx, req.(*MsgCancelSchedule))
	})
}

func _Msg_AddFee_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAddFee)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).AddFee(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.schedule.v1.Msg/AddFee"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).AddFee(ctx, req.(*MsgAddFee))
	})
}

func _Msg_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.schedule.v1.Msg/UpdateParams"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	})
}

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// -----------------------------------------------------------------------

// Query types are now proto-generated in query.pb.go
// QueryServer and RegisterQueryServer are in query_grpc.pb.go
