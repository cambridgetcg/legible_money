package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// -----------------------------------------------------------------------
// Compute service types
// -----------------------------------------------------------------------

// ComputeServiceType identifies what kind of compute a provider offers.
type ComputeServiceType string

const (
	ServiceAIInference ComputeServiceType = "ai_inference"
	ServiceGPUCompute  ComputeServiceType = "gpu_compute"
	ServiceStorage     ComputeServiceType = "storage"
)

// ValidServiceTypes lists all valid service types.
var ValidServiceTypes = []ComputeServiceType{ServiceAIInference, ServiceGPUCompute, ServiceStorage}

// IsValidServiceType checks whether a service type string is valid.
func IsValidServiceType(s string) bool {
	for _, t := range ValidServiceTypes {
		if string(t) == s {
			return true
		}
	}
	return false
}

// ProviderStatus tracks the lifecycle of a compute provider.
type ProviderStatus string

const (
	ProviderStatusActive       ProviderStatus = "active"
	ProviderStatusPaused       ProviderStatus = "paused"
	ProviderStatusSuspended    ProviderStatus = "suspended"
	ProviderStatusUnbonding    ProviderStatus = "unbonding"
	ProviderStatusDeregistered ProviderStatus = "deregistered"
)

// TicketStatus tracks the lifecycle of a redemption ticket.
type TicketStatus string

const (
	TicketStatusPending   TicketStatus = "pending"
	TicketStatusConfirmed TicketStatus = "confirmed"
	TicketStatusDisputed  TicketStatus = "disputed"
	TicketStatusExpired   TicketStatus = "expired"
)

// TaskStatus tracks the lifecycle of a compute task.
type TaskStatus string

const (
	TaskStatusPending       TaskStatus = "pending"        // submitted, awaiting assignment
	TaskStatusAssigned      TaskStatus = "assigned"       // assigned to a provider
	TaskStatusResultPending TaskStatus = "result_pending" // result submitted, awaiting verification
	TaskStatusAccepted      TaskStatus = "accepted"       // result verified and accepted
	TaskStatusRejected      TaskStatus = "rejected"       // result rejected
	TaskStatusExpired       TaskStatus = "expired"        // timed out
	TaskStatusCancelled     TaskStatus = "cancelled"      // cancelled by requester
)

// -----------------------------------------------------------------------
// Hand-written task types (json.Marshal, not proto — per project convention)
// -----------------------------------------------------------------------

// ComputeSpec describes the compute requirements for a task.
type ComputeSpec struct {
	ServiceType string `json:"service_type"`           // ai_inference, gpu_compute, storage
	Model       string `json:"model,omitempty"`        // model name for AI inference
	InputHash   string `json:"input_hash"`             // SHA-256 hash of task input
	MaxCU       uint64 `json:"max_cu"`                 // maximum compute units allowed
	MaxLatencyMs uint64 `json:"max_latency_ms,omitempty"` // optional latency requirement
}

// ComputeTask represents a submitted compute job.
type ComputeTask struct {
	Id             uint64      `json:"id"`
	RequesterDID   string      `json:"requester_did"`
	RequesterAddr  string      `json:"requester_addr"`
	Spec           ComputeSpec `json:"spec"`
	Reward         string      `json:"reward"`           // ulgm amount escrowed
	Status         string      `json:"status"`
	Provider       string      `json:"provider,omitempty"`        // assigned provider address
	OutputHash     string      `json:"output_hash,omitempty"`     // result hash from provider
	ResultMetadata string      `json:"result_metadata,omitempty"` // optional result info
	SubmittedAt    uint64      `json:"submitted_at"`
	AssignedAt     uint64      `json:"assigned_at,omitempty"`
	ResultAt       uint64      `json:"result_at,omitempty"`
	CompletedAt    uint64      `json:"completed_at,omitempty"`
	ExpiresAt      uint64      `json:"expires_at"`
	ActualCU       uint64      `json:"actual_cu,omitempty"`
}

// AgentUsageRecord tracks compute units consumed per agent DID.
type AgentUsageRecord struct {
	AgentDID       string `json:"agent_did"`
	TotalCU        uint64 `json:"total_cu"`
	TotalTasks     uint64 `json:"total_tasks"`
	TotalReward    string `json:"total_reward"`     // total ulgm spent
	AcceptedTasks  uint64 `json:"accepted_tasks"`
	RejectedTasks  uint64 `json:"rejected_tasks"`
	LastActivityAt uint64 `json:"last_activity_at"`
}

// -----------------------------------------------------------------------
// Hand-written Msg types for task lifecycle
// -----------------------------------------------------------------------

// MsgSubmitTask submits a compute task for execution.
type MsgSubmitTask struct {
	Sender       string      `json:"sender"`
	RequesterDID string      `json:"requester_did"`
	Spec         ComputeSpec `json:"spec"`
	Reward       string      `json:"reward"` // ulgm to escrow
}

// MsgSubmitTaskResponse is the response for MsgSubmitTask.
type MsgSubmitTaskResponse struct {
	TaskId uint64 `json:"task_id"`
}

// MsgAssignTask assigns a pending task to a provider.
type MsgAssignTask struct {
	Sender   string `json:"sender"`   // provider address
	TaskId   uint64 `json:"task_id"`
}

// MsgAssignTaskResponse is the response for MsgAssignTask.
type MsgAssignTaskResponse struct{}

// MsgSubmitResult submits a result for an assigned task.
type MsgSubmitResult struct {
	Sender         string `json:"sender"`          // provider address
	TaskId         uint64 `json:"task_id"`
	OutputHash     string `json:"output_hash"`
	ActualCU       uint64 `json:"actual_cu"`
	ResultMetadata string `json:"result_metadata,omitempty"`
}

// MsgSubmitResultResponse is the response for MsgSubmitResult.
type MsgSubmitResultResponse struct{}

// MsgVerifyResult accepts or rejects a task result.
type MsgVerifyResult struct {
	Sender string `json:"sender"` // requester address
	TaskId uint64 `json:"task_id"`
	Accept bool   `json:"accept"`
	Reason string `json:"reason,omitempty"` // required if rejecting
}

// MsgVerifyResultResponse is the response for MsgVerifyResult.
type MsgVerifyResultResponse struct{}

// MsgCancelTask cancels a pending (unassigned) task.
type MsgCancelTask struct {
	Sender string `json:"sender"` // requester address
	TaskId uint64 `json:"task_id"`
}

// MsgCancelTaskResponse is the response for MsgCancelTask.
type MsgCancelTaskResponse struct{}

// -----------------------------------------------------------------------
// Proto type methods (structs defined in state.pb.go, genesis.pb.go, query.pb.go, tx.pb.go)
// -----------------------------------------------------------------------

// Validate performs parameter validation.
func (p *Params) Validate() error {
	if p.ComputePoolShareBps > 1000000 {
		return fmt.Errorf("compute_pool_share_bps must be <= 1000000")
	}
	if p.BaseCuPerVerification == 0 {
		return fmt.Errorf("base_cu_per_verification must be positive")
	}
	stake := new(big.Int)
	if _, ok := stake.SetString(p.MinProviderStake, 10); !ok || stake.Sign() <= 0 {
		return fmt.Errorf("invalid min_provider_stake: %s", p.MinProviderStake)
	}
	ceiling := new(big.Int)
	if _, ok := ceiling.SetString(p.MaxPricePerCu, 10); !ok || ceiling.Sign() <= 0 {
		return fmt.Errorf("invalid max_price_per_cu: %s", p.MaxPricePerCu)
	}
	if p.MinUptimeBps > 1000000 {
		return fmt.Errorf("min_uptime_bps must be <= 1000000")
	}
	if p.HeartbeatIntervalBlocks == 0 {
		return fmt.Errorf("heartbeat_interval_blocks must be positive")
	}
	if p.CreditExpiryBlocks == 0 {
		return fmt.Errorf("credit_expiry_blocks must be positive")
	}
	if p.RedemptionExpiryBlocks == 0 {
		return fmt.Errorf("redemption_expiry_blocks must be positive")
	}
	if p.MaxProviderShareBps > 1000000 {
		return fmt.Errorf("max_provider_share_bps must be <= 1000000")
	}
	if p.TargetUtilizationLowBps >= p.TargetUtilizationHighBps {
		return fmt.Errorf("target_utilization_low_bps (%d) must be < target_utilization_high_bps (%d)", p.TargetUtilizationLowBps, p.TargetUtilizationHighBps)
	}
	if p.TargetUtilizationHighBps > 1000000 {
		return fmt.Errorf("target_utilization_high_bps must be <= 1000000")
	}
	if p.MaxAdjustmentPerEpochBps == 0 {
		return fmt.Errorf("max_adjustment_per_epoch_bps must be positive")
	}
	if p.MinAdjustmentFactorBps == 0 {
		return fmt.Errorf("min_adjustment_factor_bps must be positive")
	}
	if p.MinAdjustmentFactorBps >= p.MaxAdjustmentFactorBps {
		return fmt.Errorf("min_adjustment_factor_bps (%d) must be < max_adjustment_factor_bps (%d)", p.MinAdjustmentFactorBps, p.MaxAdjustmentFactorBps)
	}
	return nil
}

// DefaultParams returns the default module parameters.
func DefaultParams() *Params {
	return &Params{
		ComputePoolShareBps:        DefaultComputePoolShareBps,
		BaseCuPerVerification:      DefaultBaseCUPerVerification,
		ComplexityBonusBps:         DefaultComplexityBonusBps,
		CrossDomainBonusBps:        DefaultCrossDomainBonusBps,
		MaxCreditBalanceMultiplier: DefaultMaxCreditBalanceMultiplier,
		MinProviderStake:           DefaultMinProviderStake,
		MinUptimeBps:               DefaultMinUptimeBps,
		HeartbeatIntervalBlocks:    DefaultHeartbeatIntervalBlocks,
		MaxPricePerCu:              DefaultMaxPricePerCU,
		ProviderUnbondingBlocks:    DefaultProviderUnbondingBlocks,
		PriceChangeDelayBlocks:     DefaultPriceChangeDelayBlocks,
		CreditExpiryBlocks:         DefaultCreditExpiryBlocks,
		MaxProviderShareBps:        DefaultMaxProviderShareBps,
		RedemptionExpiryBlocks:     DefaultRedemptionExpiryBlocks,
		DisputeWindowBlocks:        DefaultDisputeWindowBlocks,
		MinTasksForFullShare:       DefaultMinTasksForFullShare,
		MaxLatencyMs:               DefaultMaxLatencyMs,
		SlaWindowBlocks:            DefaultSLAWindowBlocks,
		TargetUtilizationLowBps:    DefaultTargetUtilizationLowBps,
		TargetUtilizationHighBps:   DefaultTargetUtilizationHighBps,
		MaxAdjustmentPerEpochBps:   DefaultMaxAdjustmentPerEpochBps,
		MinAdjustmentFactorBps:     DefaultMinAdjustmentFactorBps,
		MaxAdjustmentFactorBps:     DefaultMaxAdjustmentFactorBps,
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Providers: []*ComputeProvider{},
		Credits:   []*ComputeCredit{},
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params != nil {
		if err := gs.Params.Validate(); err != nil {
			return fmt.Errorf("invalid params: %w", err)
		}
	}
	seen := make(map[string]bool)
	for _, p := range gs.Providers {
		if seen[p.Address] {
			return fmt.Errorf("duplicate provider address: %s", p.Address)
		}
		seen[p.Address] = true
	}
	return nil
}

// -----------------------------------------------------------------------
// GetSigners / ValidateBasic for proto-generated Msg types (tx.pb.go)
// -----------------------------------------------------------------------

func (msg *MsgRegisterComputeProvider) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterComputeProvider) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if !IsValidServiceType(msg.ServiceType) {
		return fmt.Errorf("invalid service type: %s", msg.ServiceType)
	}
	if msg.Endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Stake, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("stake amount must be a positive integer")
	}
	price := new(big.Int)
	if _, ok := price.SetString(msg.PricePerCu, 10); !ok || price.Sign() <= 0 {
		return fmt.Errorf("price per CU must be a positive integer")
	}
	return nil
}

func (msg *MsgUpdateProvider) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateProvider) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.Pause && msg.Unpause {
		return fmt.Errorf("cannot pause and unpause simultaneously")
	}
	if msg.PricePerCu != "" {
		price := new(big.Int)
		if _, ok := price.SetString(msg.PricePerCu, 10); !ok || price.Sign() <= 0 {
			return fmt.Errorf("price must be a positive integer")
		}
	}
	return nil
}

func (msg *MsgDeregisterProvider) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgDeregisterProvider) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	return nil
}

func (msg *MsgProviderHeartbeat) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgProviderHeartbeat) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	return nil
}

func (msg *MsgRedeemCredits) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRedeemCredits) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.Provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}
	if msg.Cu == 0 {
		return fmt.Errorf("cu_amount must be positive")
	}
	return nil
}

func (msg *MsgConfirmUsage) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgConfirmUsage) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	return nil
}

func (msg *MsgDisputeUsage) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgDisputeUsage) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.Reason == "" {
		return fmt.Errorf("reason cannot be empty")
	}
	return nil
}

func (msg *MsgUpdateComputeParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateComputeParams) ValidateBasic() error {
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

// -----------------------------------------------------------------------
// GetSigners / ValidateBasic for hand-written task Msg types
// -----------------------------------------------------------------------

func (msg *MsgSubmitTask) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitTask) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.RequesterDID == "" {
		return fmt.Errorf("requester_did cannot be empty")
	}
	if !IsValidServiceType(msg.Spec.ServiceType) {
		return fmt.Errorf("invalid service type: %s", msg.Spec.ServiceType)
	}
	if msg.Spec.InputHash == "" {
		return fmt.Errorf("input_hash cannot be empty")
	}
	if msg.Spec.MaxCU == 0 {
		return fmt.Errorf("max_cu must be positive")
	}
	reward := new(big.Int)
	if _, ok := reward.SetString(msg.Reward, 10); !ok || reward.Sign() <= 0 {
		return fmt.Errorf("reward must be a positive integer")
	}
	return nil
}

func (msg *MsgAssignTask) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAssignTask) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.TaskId == 0 {
		return fmt.Errorf("task_id must be positive")
	}
	return nil
}

func (msg *MsgSubmitResult) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSubmitResult) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.TaskId == 0 {
		return fmt.Errorf("task_id must be positive")
	}
	if msg.OutputHash == "" {
		return fmt.Errorf("output_hash cannot be empty")
	}
	if msg.ActualCU == 0 {
		return fmt.Errorf("actual_cu must be positive")
	}
	return nil
}

func (msg *MsgVerifyResult) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgVerifyResult) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.TaskId == 0 {
		return fmt.Errorf("task_id must be positive")
	}
	if !msg.Accept && msg.Reason == "" {
		return fmt.Errorf("reason is required when rejecting")
	}
	return nil
}

func (msg *MsgCancelTask) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCancelTask) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.TaskId == 0 {
		return fmt.Errorf("task_id must be positive")
	}
	return nil
}
