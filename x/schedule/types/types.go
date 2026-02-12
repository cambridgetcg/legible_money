package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// -----------------------------------------------------------------------
// Core domain types
// -----------------------------------------------------------------------

// ProcessStatus represents the lifecycle of a schedule process.
type ProcessStatus string

const (
	StatusActive    ProcessStatus = "active"
	StatusPaused    ProcessStatus = "paused"
	StatusCompleted ProcessStatus = "completed"
	StatusCancelled ProcessStatus = "cancelled"
	StatusExhausted ProcessStatus = "exhausted" // ran out of prepaid fee
)

// TimeConditionType defines how time-based conditions work.
type TimeConditionType string

const (
	TimeAbsolute TimeConditionType = "absolute" // fire at specific block
	TimeRelative TimeConditionType = "relative" // fire N blocks from creation
	TimePeriodic TimeConditionType = "periodic" // fire every N blocks
)

// LogicConditionType defines state-based conditions.
type LogicConditionType string

const (
	LogicStateCompare    LogicConditionType = "state_compare"
	LogicThresholdCrossed LogicConditionType = "threshold_crossed"
)

// CompoundOperator for combining conditions.
type CompoundOperator string

const (
	CompoundAnd CompoundOperator = "and"
	CompoundOr  CompoundOperator = "or"
)

// TimeCondition triggers based on block height.
type TimeCondition struct {
	Type           TimeConditionType `json:"type"`
	ExecuteAtBlock uint64            `json:"execute_at_block,omitempty"` // absolute/relative target
	IntervalBlocks uint64            `json:"interval_blocks,omitempty"` // periodic interval
	StartBlock     uint64            `json:"start_block,omitempty"`     // periodic start
	EndBlock       uint64            `json:"end_block,omitempty"`       // periodic end (0 = no end)
}

func (tc *TimeCondition) Reset()        {}
func (tc *TimeCondition) String() string { return fmt.Sprintf("%+v", *tc) }
func (tc *TimeCondition) ProtoMessage()  {}

// LogicCondition triggers based on state comparison.
type LogicCondition struct {
	Type         LogicConditionType `json:"type"`
	StateKey     string             `json:"state_key"`               // key to watch
	Comparator   string             `json:"comparator"`              // "eq", "gt", "lt", "gte", "lte", "neq"
	CompareValue string             `json:"compare_value"`           // value to compare against
	Direction    string             `json:"direction,omitempty"`     // "up" or "down" for threshold_crossed
}

func (lc *LogicCondition) Reset()        {}
func (lc *LogicCondition) String() string { return fmt.Sprintf("%+v", *lc) }
func (lc *LogicCondition) ProtoMessage()  {}

// CompoundCondition combines multiple conditions.
type CompoundCondition struct {
	Operator   CompoundOperator   `json:"operator"`
	Conditions []ScheduleCondition `json:"conditions"`
}

func (cc *CompoundCondition) Reset()        {}
func (cc *CompoundCondition) String() string { return fmt.Sprintf("%+v", *cc) }
func (cc *CompoundCondition) ProtoMessage()  {}

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

// ScheduleCondition wraps the three condition types (exactly one must be set).
type ScheduleCondition struct {
	TimeCondition     *TimeCondition     `json:"time_condition,omitempty"`
	LogicCondition    *LogicCondition    `json:"logic_condition,omitempty"`
	CompoundCondition *CompoundCondition `json:"compound_condition,omitempty"`
}

func (sc *ScheduleCondition) Reset()        {}
func (sc *ScheduleCondition) String() string { return fmt.Sprintf("%+v", *sc) }
func (sc *ScheduleCondition) ProtoMessage()  {}

// ScheduleProcess represents a scheduled automation.
type ScheduleProcess struct {
	Id               string            `json:"id"`
	Creator          string            `json:"creator"`           // bech32 address
	Condition        ScheduleCondition `json:"condition"`
	Status           ProcessStatus     `json:"status"`
	ExecutionCount   uint64            `json:"execution_count"`
	MaxExecutions    uint64            `json:"max_executions"`    // 0 = unlimited
	RemainingFee     string            `json:"remaining_fee"`     // ulgm prepaid
	FeePerExecution  string            `json:"fee_per_execution"` // ulgm per run
	TargetAddress    string            `json:"target_address"`    // bech32 target
	CallData         string            `json:"call_data"`         // encoded message
	TransferValue    string            `json:"transfer_value"`    // ulgm to transfer on execution
	LinkedEntityType string            `json:"linked_entity_type,omitempty"` // "project", "partnership", etc.
	LinkedEntityId   string            `json:"linked_entity_id,omitempty"`
	CreatedAtBlock   uint64            `json:"created_at_block"`
	ExpiresAtBlock   uint64            `json:"expires_at_block"`  // 0 = no expiry
	NextExecuteAt    uint64            `json:"next_execute_at"`   // next scheduled block
}

func (sp *ScheduleProcess) Reset()        {}
func (sp *ScheduleProcess) String() string { return fmt.Sprintf("%+v", *sp) }
func (sp *ScheduleProcess) ProtoMessage()  {}

// -----------------------------------------------------------------------
// Params
// -----------------------------------------------------------------------

// Params holds module parameters.
type Params struct {
	MaxActivePerAccount  uint32 `json:"max_active_per_account"`  // max active schedules per creator
	MaxGasPerBlock       uint64 `json:"max_gas_per_block"`       // gas budget for schedule execution per block
	MinIntervalBlocks    uint64 `json:"min_interval_blocks"`     // minimum periodic interval
	MaxFutureBlocks      uint64 `json:"max_future_blocks"`       // max blocks into the future
	MinPrepaidExecutions uint64 `json:"min_prepaid_executions"`  // min executions to prepay
	ExpiredRetentionBlocks uint64 `json:"expired_retention_blocks"` // blocks to retain completed/cancelled
}

func (p *Params) Reset()        {}
func (p *Params) String() string { return fmt.Sprintf("%+v", *p) }
func (p *Params) ProtoMessage()  {}

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
// Genesis state
// -----------------------------------------------------------------------

// GenesisState defines the schedule module genesis state.
type GenesisState struct {
	Params    Params            `json:"params"`
	Processes []ScheduleProcess `json:"processes"`
	Sequence  uint64            `json:"sequence"`
}

func (gs *GenesisState) Reset()        {}
func (gs *GenesisState) String() string { return fmt.Sprintf("%+v", *gs) }
func (gs *GenesisState) ProtoMessage()  {}

// -----------------------------------------------------------------------
// Message types
// -----------------------------------------------------------------------

// MsgCreateSchedule creates a new scheduled process.
type MsgCreateSchedule struct {
	Creator          string            `json:"creator"`
	Condition        ScheduleCondition `json:"condition"`
	MaxExecutions    uint64            `json:"max_executions"`
	FeePerExecution  string            `json:"fee_per_execution"`
	PrepaidFee       string            `json:"prepaid_fee"`
	TargetAddress    string            `json:"target_address"`
	CallData         string            `json:"call_data"`
	TransferValue    string            `json:"transfer_value"`
	LinkedEntityType string            `json:"linked_entity_type,omitempty"`
	LinkedEntityId   string            `json:"linked_entity_id,omitempty"`
}

func (msg *MsgCreateSchedule) Reset()        {}
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

func (msg *MsgCreateScheduleResponse) Reset()        {}
func (msg *MsgCreateScheduleResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgCreateScheduleResponse) ProtoMessage()  {}

// MsgPauseSchedule pauses an active schedule.
type MsgPauseSchedule struct {
	Creator   string `json:"creator"`
	ProcessId string `json:"process_id"`
}

func (msg *MsgPauseSchedule) Reset()        {}
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

func (msg *MsgPauseScheduleResponse) Reset()        {}
func (msg *MsgPauseScheduleResponse) String() string { return "" }
func (msg *MsgPauseScheduleResponse) ProtoMessage()  {}

// MsgResumeSchedule resumes a paused schedule.
type MsgResumeSchedule struct {
	Creator   string `json:"creator"`
	ProcessId string `json:"process_id"`
}

func (msg *MsgResumeSchedule) Reset()        {}
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

func (msg *MsgResumeScheduleResponse) Reset()        {}
func (msg *MsgResumeScheduleResponse) String() string { return "" }
func (msg *MsgResumeScheduleResponse) ProtoMessage()  {}

// MsgCancelSchedule cancels a schedule and refunds remaining fee.
type MsgCancelSchedule struct {
	Creator   string `json:"creator"`
	ProcessId string `json:"process_id"`
}

func (msg *MsgCancelSchedule) Reset()        {}
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

func (msg *MsgCancelScheduleResponse) Reset()        {}
func (msg *MsgCancelScheduleResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgCancelScheduleResponse) ProtoMessage()  {}

// MsgAddFee adds prepaid fee to an existing schedule.
type MsgAddFee struct {
	Creator   string `json:"creator"`
	ProcessId string `json:"process_id"`
	Amount    string `json:"amount"` // ulgm to add
}

func (msg *MsgAddFee) Reset()        {}
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

func (msg *MsgAddFeeResponse) Reset()        {}
func (msg *MsgAddFeeResponse) String() string { return "" }
func (msg *MsgAddFeeResponse) ProtoMessage()  {}

// MsgUpdateParams updates module parameters (governance-gated).
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (msg *MsgUpdateParams) Reset()        {}
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

func (msg *MsgUpdateParamsResponse) Reset()        {}
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
// QueryServer interface
// -----------------------------------------------------------------------

// QueryServer defines the schedule module query service.
type QueryServer interface {
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	Process(context.Context, *QueryProcessRequest) (*QueryProcessResponse, error)
	ProcessesByCreator(context.Context, *QueryProcessesByCreatorRequest) (*QueryProcessesByCreatorResponse, error)
}

// Query request/response types.

type QueryParamsRequest struct{}

func (r *QueryParamsRequest) Reset()        {}
func (r *QueryParamsRequest) String() string { return "QueryParamsRequest" }
func (r *QueryParamsRequest) ProtoMessage()  {}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (r *QueryParamsResponse) Reset()        {}
func (r *QueryParamsResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryParamsResponse) ProtoMessage()  {}

type QueryProcessRequest struct {
	ProcessId string `json:"process_id"`
}

func (r *QueryProcessRequest) Reset()        {}
func (r *QueryProcessRequest) String() string { return fmt.Sprintf("QueryProcess{%s}", r.ProcessId) }
func (r *QueryProcessRequest) ProtoMessage()  {}

type QueryProcessResponse struct {
	Process ScheduleProcess `json:"process"`
}

func (r *QueryProcessResponse) Reset()        {}
func (r *QueryProcessResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryProcessResponse) ProtoMessage()  {}

type QueryProcessesByCreatorRequest struct {
	Creator string `json:"creator"`
}

func (r *QueryProcessesByCreatorRequest) Reset()        {}
func (r *QueryProcessesByCreatorRequest) String() string { return fmt.Sprintf("QueryProcessesByCreator{%s}", r.Creator) }
func (r *QueryProcessesByCreatorRequest) ProtoMessage()  {}

type QueryProcessesByCreatorResponse struct {
	Processes []ScheduleProcess `json:"processes"`
}

func (r *QueryProcessesByCreatorResponse) Reset()        {}
func (r *QueryProcessesByCreatorResponse) String() string { return fmt.Sprintf("%+v", *r) }
func (r *QueryProcessesByCreatorResponse) ProtoMessage()  {}

// Query service descriptor.

// gRPC handler functions for Query service.
func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.schedule.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

func _Query_Process_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryProcessRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Process(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.schedule.v1.Query/Process"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Process(ctx, req.(*QueryProcessRequest))
	})
}

func _Query_ProcessesByCreator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryProcessesByCreatorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ProcessesByCreator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.schedule.v1.Query/ProcessesByCreator"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ProcessesByCreator(ctx, req.(*QueryProcessesByCreatorRequest))
	})
}
var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.schedule.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Params", Handler: _Query_Params_Handler},
		{MethodName: "Process", Handler: _Query_Process_Handler},
		{MethodName: "ProcessesByCreator", Handler: _Query_ProcessesByCreator_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/schedule/v1/query.proto",
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
