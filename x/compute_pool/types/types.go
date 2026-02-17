package types

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
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

// -----------------------------------------------------------------------
// Proto-compatible types (hand-written, matching proto patterns)
// -----------------------------------------------------------------------

// Params holds governance-tunable parameters for the compute pool module.
type Params struct {
	ComputePoolShareBps        uint64 `json:"compute_pool_share_bps"`
	BaseCUPerVerification      uint64 `json:"base_cu_per_verification"`
	ComplexityBonusBps         uint64 `json:"complexity_bonus_bps"`
	CrossDomainBonusBps        uint64 `json:"cross_domain_bonus_bps"`
	MaxCreditBalanceMultiplier uint64 `json:"max_credit_balance_multiplier"`
	MinProviderStake           string `json:"min_provider_stake"`
	MinUptimeBps               uint64 `json:"min_uptime_bps"`
	HeartbeatIntervalBlocks    uint64 `json:"heartbeat_interval_blocks"`
	MaxPricePerCU              string `json:"max_price_per_cu"`
	ProviderUnbondingBlocks    uint64 `json:"provider_unbonding_blocks"`
	PriceChangeDelayBlocks     uint64 `json:"price_change_delay_blocks"`
	CreditExpiryBlocks         uint64 `json:"credit_expiry_blocks"`
	MaxProviderShareBps        uint64 `json:"max_provider_share_bps"`
	RedemptionExpiryBlocks     uint64 `json:"redemption_expiry_blocks"`
	DisputeWindowBlocks        uint64 `json:"dispute_window_blocks"`
	MinTasksForFullShare       uint64 `json:"min_tasks_for_full_share"`
	MaxLatencyMs               uint64 `json:"max_latency_ms"`
	SLAWindowBlocks            uint64 `json:"sla_window_blocks"`

	// Utilization-based auto-tuning parameters
	TargetUtilizationLowBps  uint64 `json:"target_utilization_low_bps"`
	TargetUtilizationHighBps uint64 `json:"target_utilization_high_bps"`
	MaxAdjustmentPerEpochBps uint64 `json:"max_adjustment_per_epoch_bps"`
	MinAdjustmentFactorBps   uint64 `json:"min_adjustment_factor_bps"`
	MaxAdjustmentFactorBps   uint64 `json:"max_adjustment_factor_bps"`
}

func (p *Params) Reset()         {}
func (p *Params) String() string { return fmt.Sprintf("%+v", *p) }
func (p *Params) ProtoMessage()  {}

// Validate performs parameter validation.
func (p Params) Validate() error {
	if p.ComputePoolShareBps > 1000000 {
		return fmt.Errorf("compute_pool_share_bps must be <= 1000000")
	}
	if p.BaseCUPerVerification == 0 {
		return fmt.Errorf("base_cu_per_verification must be positive")
	}
	stake := new(big.Int)
	if _, ok := stake.SetString(p.MinProviderStake, 10); !ok || stake.Sign() <= 0 {
		return fmt.Errorf("invalid min_provider_stake: %s", p.MinProviderStake)
	}
	ceiling := new(big.Int)
	if _, ok := ceiling.SetString(p.MaxPricePerCU, 10); !ok || ceiling.Sign() <= 0 {
		return fmt.Errorf("invalid max_price_per_cu: %s", p.MaxPricePerCU)
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
func DefaultParams() Params {
	return Params{
		ComputePoolShareBps:        DefaultComputePoolShareBps,
		BaseCUPerVerification:      DefaultBaseCUPerVerification,
		ComplexityBonusBps:         DefaultComplexityBonusBps,
		CrossDomainBonusBps:        DefaultCrossDomainBonusBps,
		MaxCreditBalanceMultiplier: DefaultMaxCreditBalanceMultiplier,
		MinProviderStake:           DefaultMinProviderStake,
		MinUptimeBps:               DefaultMinUptimeBps,
		HeartbeatIntervalBlocks:    DefaultHeartbeatIntervalBlocks,
		MaxPricePerCU:              DefaultMaxPricePerCU,
		ProviderUnbondingBlocks:    DefaultProviderUnbondingBlocks,
		PriceChangeDelayBlocks:     DefaultPriceChangeDelayBlocks,
		CreditExpiryBlocks:         DefaultCreditExpiryBlocks,
		MaxProviderShareBps:        DefaultMaxProviderShareBps,
		RedemptionExpiryBlocks:     DefaultRedemptionExpiryBlocks,
		DisputeWindowBlocks:        DefaultDisputeWindowBlocks,
		MinTasksForFullShare:       DefaultMinTasksForFullShare,
		MaxLatencyMs:               DefaultMaxLatencyMs,
		SLAWindowBlocks:            DefaultSLAWindowBlocks,
		TargetUtilizationLowBps:    DefaultTargetUtilizationLowBps,
		TargetUtilizationHighBps:   DefaultTargetUtilizationHighBps,
		MaxAdjustmentPerEpochBps:   DefaultMaxAdjustmentPerEpochBps,
		MinAdjustmentFactorBps:     DefaultMinAdjustmentFactorBps,
		MaxAdjustmentFactorBps:     DefaultMaxAdjustmentFactorBps,
	}
}

// -----------------------------------------------------------------------
// State types
// -----------------------------------------------------------------------

// ComputeProvider represents a registered compute infrastructure provider.
type ComputeProvider struct {
	Address       string `json:"address"`
	Did           string `json:"did"`
	ServiceType   string `json:"service_type"`
	Endpoint      string `json:"endpoint"`
	PricePerCU    string `json:"price_per_cu"`
	Stake         string `json:"stake"`
	Status        string `json:"status"`
	RegisteredAt  uint64 `json:"registered_at"`
	TasksServed   uint64 `json:"tasks_served"`
	TasksFailed   uint64 `json:"tasks_failed"`
	AvgLatencyMs  uint64 `json:"avg_latency_ms"`
	UptimeBps     uint64 `json:"uptime_bps"`
	LastHeartbeat uint64 `json:"last_heartbeat"`
	UnbondingAt   uint64 `json:"unbonding_at"` // block when unbonding started (0 = not unbonding)

	// Pending price change (takes effect after PriceChangeDelayBlocks)
	PendingPrice  string `json:"pending_price"`
	PriceChangeAt uint64 `json:"price_change_at"` // block when pending price takes effect
}

func (p *ComputeProvider) Reset()         {}
func (p *ComputeProvider) String() string { return fmt.Sprintf("%+v", *p) }
func (p *ComputeProvider) ProtoMessage()  {}

// ComputeCredit tracks a validator's compute credit balance.
type ComputeCredit struct {
	ValidatorAddr string `json:"validator_addr"`
	Balance       uint64 `json:"balance"`
	EarnedTotal   uint64 `json:"earned_total"`
	RedeemedTotal uint64 `json:"redeemed_total"`
	ExpiredTotal  uint64 `json:"expired_total"`
	LastEarnedAt  uint64 `json:"last_earned_at"`
	ExpiresAt     uint64 `json:"expires_at"`
}

func (c *ComputeCredit) Reset()         {}
func (c *ComputeCredit) String() string { return fmt.Sprintf("%+v", *c) }
func (c *ComputeCredit) ProtoMessage()  {}

// RedemptionTicket tracks a credit-to-compute exchange.
type RedemptionTicket struct {
	Id        uint64 `json:"id"`
	Validator string `json:"validator"`
	Provider  string `json:"provider"`
	CUAmount  uint64 `json:"cu_amount"`
	LGMAmount string `json:"lgm_amount"` // ulgm to transfer
	Status    string `json:"status"`
	CreatedAt uint64 `json:"created_at"`
	ExpiresAt uint64 `json:"expires_at"`
	ActualCU  uint64 `json:"actual_cu"` // filled on confirmation
}

func (t *RedemptionTicket) Reset()         {}
func (t *RedemptionTicket) String() string { return fmt.Sprintf("%+v", *t) }
func (t *RedemptionTicket) ProtoMessage()  {}

// UsageReport records confirmed compute usage.
type UsageReport struct {
	TicketId    uint64 `json:"ticket_id"`
	Validator   string `json:"validator"`
	Provider    string `json:"provider"`
	RequestedCU uint64 `json:"requested_cu"`
	ActualCU    uint64 `json:"actual_cu"`
	LGMPaid     string `json:"lgm_paid"`
	ConfirmedAt uint64 `json:"confirmed_at"`
}

func (u *UsageReport) Reset()         {}
func (u *UsageReport) String() string { return fmt.Sprintf("%+v", *u) }
func (u *UsageReport) ProtoMessage()  {}

// EpochStats tracks per-epoch compute pool statistics.
type EpochStats struct {
	Epoch               uint64 `json:"epoch"`
	PoolInflow          string `json:"pool_inflow"`
	CreditsIssued       uint64 `json:"credits_issued"`
	CreditsRedeemed     uint64 `json:"credits_redeemed"`
	CreditsExpired      uint64 `json:"credits_expired"`
	LGMDistributed      string `json:"lgm_distributed"`
	ActiveProviders     uint32 `json:"active_providers"`
	ActiveValidators    uint32 `json:"active_validators"`
	UtilizationBps      uint64 `json:"utilization_bps"`       // outflow/inflow ratio (bps)
	AdjustmentFactorBps uint64 `json:"adjustment_factor_bps"` // factor applied to credits this epoch
}

func (e *EpochStats) Reset()         {}
func (e *EpochStats) String() string { return fmt.Sprintf("%+v", *e) }
func (e *EpochStats) ProtoMessage()  {}

// ProviderEpochFlow tracks how much LGM flowed to a provider in an epoch.
type ProviderEpochFlow struct {
	Epoch       uint64 `json:"epoch"`
	Provider    string `json:"provider"`
	LGMReceived string `json:"lgm_received"`
	TasksServed uint64 `json:"tasks_served"`
}

func (f *ProviderEpochFlow) Reset()         {}
func (f *ProviderEpochFlow) String() string { return fmt.Sprintf("%+v", *f) }
func (f *ProviderEpochFlow) ProtoMessage()  {}

// -----------------------------------------------------------------------
// Genesis state
// -----------------------------------------------------------------------

// GenesisState defines the compute_pool module genesis state.
type GenesisState struct {
	Params    *Params            `json:"params"`
	Providers []*ComputeProvider `json:"providers"`
	Credits   []*ComputeCredit   `json:"credits"`
}

func (gs *GenesisState) Reset()         {}
func (gs *GenesisState) String() string { return fmt.Sprintf("%+v", *gs) }
func (gs *GenesisState) ProtoMessage()  {}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:    &p,
		Providers: []*ComputeProvider{},
		Credits:   []*ComputeCredit{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
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
// Message types
// -----------------------------------------------------------------------

// MsgRegisterComputeProvider registers a new compute provider.
type MsgRegisterComputeProvider struct {
	Sender      string `json:"sender"`
	ServiceType string `json:"service_type"`
	Endpoint    string `json:"endpoint"`
	PricePerCU  string `json:"price_per_cu"`
	StakeAmount string `json:"stake_amount"`
	Did         string `json:"did"`
}

func (msg *MsgRegisterComputeProvider) Reset()         {}
func (msg *MsgRegisterComputeProvider) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRegisterComputeProvider) ProtoMessage()  {}

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
	if _, ok := amt.SetString(msg.StakeAmount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("stake amount must be a positive integer")
	}
	price := new(big.Int)
	if _, ok := price.SetString(msg.PricePerCU, 10); !ok || price.Sign() <= 0 {
		return fmt.Errorf("price per CU must be a positive integer")
	}
	return nil
}

// MsgRegisterComputeProviderResponse is the response for MsgRegisterComputeProvider.
type MsgRegisterComputeProviderResponse struct{}

func (msg *MsgRegisterComputeProviderResponse) Reset()         {}
func (msg *MsgRegisterComputeProviderResponse) String() string { return "" }
func (msg *MsgRegisterComputeProviderResponse) ProtoMessage()  {}

// MsgUpdateProvider updates provider endpoint, price, or status.
type MsgUpdateProvider struct {
	Sender   string `json:"sender"`
	Endpoint string `json:"endpoint,omitempty"`
	Price    string `json:"price,omitempty"`
	Pause    bool   `json:"pause,omitempty"`
	Unpause  bool   `json:"unpause,omitempty"`
}

func (msg *MsgUpdateProvider) Reset()         {}
func (msg *MsgUpdateProvider) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgUpdateProvider) ProtoMessage()  {}

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
	if msg.Price != "" {
		price := new(big.Int)
		if _, ok := price.SetString(msg.Price, 10); !ok || price.Sign() <= 0 {
			return fmt.Errorf("price must be a positive integer")
		}
	}
	return nil
}

// MsgUpdateProviderResponse is the response for MsgUpdateProvider.
type MsgUpdateProviderResponse struct{}

func (msg *MsgUpdateProviderResponse) Reset()         {}
func (msg *MsgUpdateProviderResponse) String() string { return "" }
func (msg *MsgUpdateProviderResponse) ProtoMessage()  {}

// MsgDeregisterProvider removes a provider and begins unbonding.
type MsgDeregisterProvider struct {
	Sender string `json:"sender"`
}

func (msg *MsgDeregisterProvider) Reset()         {}
func (msg *MsgDeregisterProvider) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgDeregisterProvider) ProtoMessage()  {}

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

// MsgDeregisterProviderResponse is the response for MsgDeregisterProvider.
type MsgDeregisterProviderResponse struct{}

func (msg *MsgDeregisterProviderResponse) Reset()         {}
func (msg *MsgDeregisterProviderResponse) String() string { return "" }
func (msg *MsgDeregisterProviderResponse) ProtoMessage()  {}

// MsgProviderHeartbeat proves provider liveness.
type MsgProviderHeartbeat struct {
	Sender string `json:"sender"`
}

func (msg *MsgProviderHeartbeat) Reset()         {}
func (msg *MsgProviderHeartbeat) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgProviderHeartbeat) ProtoMessage()  {}

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

// MsgProviderHeartbeatResponse is the response for MsgProviderHeartbeat.
type MsgProviderHeartbeatResponse struct{}

func (msg *MsgProviderHeartbeatResponse) Reset()         {}
func (msg *MsgProviderHeartbeatResponse) String() string { return "" }
func (msg *MsgProviderHeartbeatResponse) ProtoMessage()  {}

// MsgRedeemCredits spends compute credits for provider services.
type MsgRedeemCredits struct {
	Sender   string `json:"sender"`
	Provider string `json:"provider"`
	CUAmount uint64 `json:"cu_amount"`
}

func (msg *MsgRedeemCredits) Reset()         {}
func (msg *MsgRedeemCredits) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRedeemCredits) ProtoMessage()  {}

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
	if msg.CUAmount == 0 {
		return fmt.Errorf("cu_amount must be positive")
	}
	return nil
}

// MsgRedeemCreditsResponse is the response for MsgRedeemCredits.
type MsgRedeemCreditsResponse struct {
	TicketId uint64 `json:"ticket_id"`
}

func (msg *MsgRedeemCreditsResponse) Reset()         {}
func (msg *MsgRedeemCreditsResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRedeemCreditsResponse) ProtoMessage()  {}

// MsgConfirmUsage confirms that compute was delivered.
type MsgConfirmUsage struct {
	Sender   string `json:"sender"`
	TicketId uint64 `json:"ticket_id"`
	ActualCU uint64 `json:"actual_cu"`
}

func (msg *MsgConfirmUsage) Reset()         {}
func (msg *MsgConfirmUsage) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgConfirmUsage) ProtoMessage()  {}

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

// MsgConfirmUsageResponse is the response for MsgConfirmUsage.
type MsgConfirmUsageResponse struct{}

func (msg *MsgConfirmUsageResponse) Reset()         {}
func (msg *MsgConfirmUsageResponse) String() string { return "" }
func (msg *MsgConfirmUsageResponse) ProtoMessage()  {}

// MsgDisputeUsage disputes a redemption (compute not delivered).
type MsgDisputeUsage struct {
	Sender   string `json:"sender"`
	TicketId uint64 `json:"ticket_id"`
	Reason   string `json:"reason"`
}

func (msg *MsgDisputeUsage) Reset()         {}
func (msg *MsgDisputeUsage) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgDisputeUsage) ProtoMessage()  {}

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

// MsgDisputeUsageResponse is the response for MsgDisputeUsage.
type MsgDisputeUsageResponse struct{}

func (msg *MsgDisputeUsageResponse) Reset()         {}
func (msg *MsgDisputeUsageResponse) String() string { return "" }
func (msg *MsgDisputeUsageResponse) ProtoMessage()  {}

// MsgUpdateComputeParams updates module parameters (governance-gated).
type MsgUpdateComputeParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (msg *MsgUpdateComputeParams) Reset()         {}
func (msg *MsgUpdateComputeParams) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgUpdateComputeParams) ProtoMessage()  {}

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
	return msg.Params.Validate()
}

// MsgUpdateComputeParamsResponse is the response for MsgUpdateComputeParams.
type MsgUpdateComputeParamsResponse struct{}

func (msg *MsgUpdateComputeParamsResponse) Reset()         {}
func (msg *MsgUpdateComputeParamsResponse) String() string { return "" }
func (msg *MsgUpdateComputeParamsResponse) ProtoMessage()  {}

// -----------------------------------------------------------------------
// MsgServer interface
// -----------------------------------------------------------------------

// MsgServer defines the compute_pool module message service.
type MsgServer interface {
	RegisterComputeProvider(context.Context, *MsgRegisterComputeProvider) (*MsgRegisterComputeProviderResponse, error)
	UpdateProvider(context.Context, *MsgUpdateProvider) (*MsgUpdateProviderResponse, error)
	DeregisterProvider(context.Context, *MsgDeregisterProvider) (*MsgDeregisterProviderResponse, error)
	ProviderHeartbeat(context.Context, *MsgProviderHeartbeat) (*MsgProviderHeartbeatResponse, error)
	RedeemCredits(context.Context, *MsgRedeemCredits) (*MsgRedeemCreditsResponse, error)
	ConfirmUsage(context.Context, *MsgConfirmUsage) (*MsgConfirmUsageResponse, error)
	DisputeUsage(context.Context, *MsgDisputeUsage) (*MsgDisputeUsageResponse, error)
	UpdateComputeParams(context.Context, *MsgUpdateComputeParams) (*MsgUpdateComputeParamsResponse, error)
}

// -----------------------------------------------------------------------
// gRPC service descriptor (hand-written, replaces proto generation)
// -----------------------------------------------------------------------

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.compute_pool.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "RegisterComputeProvider", Handler: _Msg_RegisterComputeProvider_Handler},
		{MethodName: "UpdateProvider", Handler: _Msg_UpdateProvider_Handler},
		{MethodName: "DeregisterProvider", Handler: _Msg_DeregisterProvider_Handler},
		{MethodName: "ProviderHeartbeat", Handler: _Msg_ProviderHeartbeat_Handler},
		{MethodName: "RedeemCredits", Handler: _Msg_RedeemCredits_Handler},
		{MethodName: "ConfirmUsage", Handler: _Msg_ConfirmUsage_Handler},
		{MethodName: "DisputeUsage", Handler: _Msg_DisputeUsage_Handler},
		{MethodName: "UpdateComputeParams", Handler: _Msg_UpdateComputeParams_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/compute_pool/v1/tx.proto",
}

func _Msg_RegisterComputeProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterComputeProvider)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RegisterComputeProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Msg/RegisterComputeProvider"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RegisterComputeProvider(ctx, req.(*MsgRegisterComputeProvider))
	})
}

func _Msg_UpdateProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateProvider)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Msg/UpdateProvider"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateProvider(ctx, req.(*MsgUpdateProvider))
	})
}

func _Msg_DeregisterProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgDeregisterProvider)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).DeregisterProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Msg/DeregisterProvider"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).DeregisterProvider(ctx, req.(*MsgDeregisterProvider))
	})
}

func _Msg_ProviderHeartbeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgProviderHeartbeat)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ProviderHeartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Msg/ProviderHeartbeat"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ProviderHeartbeat(ctx, req.(*MsgProviderHeartbeat))
	})
}

func _Msg_RedeemCredits_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRedeemCredits)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RedeemCredits(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Msg/RedeemCredits"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RedeemCredits(ctx, req.(*MsgRedeemCredits))
	})
}

func _Msg_ConfirmUsage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgConfirmUsage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ConfirmUsage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Msg/ConfirmUsage"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ConfirmUsage(ctx, req.(*MsgConfirmUsage))
	})
}

func _Msg_DisputeUsage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgDisputeUsage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).DisputeUsage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Msg/DisputeUsage"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).DisputeUsage(ctx, req.(*MsgDisputeUsage))
	})
}

func _Msg_UpdateComputeParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateComputeParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateComputeParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Msg/UpdateComputeParams"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateComputeParams(ctx, req.(*MsgUpdateComputeParams))
	})
}

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

// -----------------------------------------------------------------------
// Query types
// -----------------------------------------------------------------------

// QueryParamsRequest is the request for the Params query.
type QueryParamsRequest struct{}

func (q *QueryParamsRequest) Reset()         {}
func (q *QueryParamsRequest) String() string { return "" }
func (q *QueryParamsRequest) ProtoMessage()  {}

// QueryParamsResponse is the response for the Params query.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (q *QueryParamsResponse) Reset()         {}
func (q *QueryParamsResponse) String() string { return fmt.Sprintf("%+v", *q) }
func (q *QueryParamsResponse) ProtoMessage()  {}

// QueryProviderRequest is the request for the Provider query.
type QueryProviderRequest struct {
	Address string `json:"address"`
}

func (q *QueryProviderRequest) Reset()         {}
func (q *QueryProviderRequest) String() string { return fmt.Sprintf("%+v", *q) }
func (q *QueryProviderRequest) ProtoMessage()  {}

// QueryProviderResponse is the response for the Provider query.
type QueryProviderResponse struct {
	Provider ComputeProvider `json:"provider"`
}

func (q *QueryProviderResponse) Reset()         {}
func (q *QueryProviderResponse) String() string { return fmt.Sprintf("%+v", *q) }
func (q *QueryProviderResponse) ProtoMessage()  {}

// QueryActiveProvidersRequest lists active providers.
type QueryActiveProvidersRequest struct {
	ServiceType string `json:"service_type"`
}

func (q *QueryActiveProvidersRequest) Reset()         {}
func (q *QueryActiveProvidersRequest) String() string { return fmt.Sprintf("%+v", *q) }
func (q *QueryActiveProvidersRequest) ProtoMessage()  {}

// QueryActiveProvidersResponse lists active providers.
type QueryActiveProvidersResponse struct {
	Providers []ComputeProvider `json:"providers"`
}

func (q *QueryActiveProvidersResponse) Reset()         {}
func (q *QueryActiveProvidersResponse) String() string { return fmt.Sprintf("%+v", *q) }
func (q *QueryActiveProvidersResponse) ProtoMessage()  {}

// QueryCreditBalanceRequest returns a validator's credit balance.
type QueryCreditBalanceRequest struct {
	ValidatorAddr string `json:"validator_addr"`
}

func (q *QueryCreditBalanceRequest) Reset()         {}
func (q *QueryCreditBalanceRequest) String() string { return fmt.Sprintf("%+v", *q) }
func (q *QueryCreditBalanceRequest) ProtoMessage()  {}

// QueryCreditBalanceResponse returns a validator's credit balance.
type QueryCreditBalanceResponse struct {
	Credit ComputeCredit `json:"credit"`
}

func (q *QueryCreditBalanceResponse) Reset()         {}
func (q *QueryCreditBalanceResponse) String() string { return fmt.Sprintf("%+v", *q) }
func (q *QueryCreditBalanceResponse) ProtoMessage()  {}

// QueryRedemptionTicketRequest returns a ticket's status.
type QueryRedemptionTicketRequest struct {
	TicketId uint64 `json:"ticket_id"`
}

func (q *QueryRedemptionTicketRequest) Reset()         {}
func (q *QueryRedemptionTicketRequest) String() string { return fmt.Sprintf("%+v", *q) }
func (q *QueryRedemptionTicketRequest) ProtoMessage()  {}

// QueryRedemptionTicketResponse returns a ticket's status.
type QueryRedemptionTicketResponse struct {
	Ticket RedemptionTicket `json:"ticket"`
}

func (q *QueryRedemptionTicketResponse) Reset()         {}
func (q *QueryRedemptionTicketResponse) String() string { return fmt.Sprintf("%+v", *q) }
func (q *QueryRedemptionTicketResponse) ProtoMessage()  {}

// QueryPoolStatsRequest returns current pool statistics.
type QueryPoolStatsRequest struct{}

func (q *QueryPoolStatsRequest) Reset()         {}
func (q *QueryPoolStatsRequest) String() string { return "" }
func (q *QueryPoolStatsRequest) ProtoMessage()  {}

// QueryPoolStatsResponse returns current pool statistics.
type QueryPoolStatsResponse struct {
	PoolBalance         string     `json:"pool_balance"`
	CurrentEpoch        EpochStats `json:"current_epoch"`
	AdjustmentFactorBps uint64     `json:"adjustment_factor_bps"` // current credit multiplier
}

func (q *QueryPoolStatsResponse) Reset()         {}
func (q *QueryPoolStatsResponse) String() string { return fmt.Sprintf("%+v", *q) }
func (q *QueryPoolStatsResponse) ProtoMessage()  {}

// -----------------------------------------------------------------------
// QueryServer interface
// -----------------------------------------------------------------------

// QueryServer defines the compute_pool module query service.
type QueryServer interface {
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	Provider(context.Context, *QueryProviderRequest) (*QueryProviderResponse, error)
	ActiveProviders(context.Context, *QueryActiveProvidersRequest) (*QueryActiveProvidersResponse, error)
	CreditBalance(context.Context, *QueryCreditBalanceRequest) (*QueryCreditBalanceResponse, error)
	RedemptionTicket(context.Context, *QueryRedemptionTicketRequest) (*QueryRedemptionTicketResponse, error)
	PoolStats(context.Context, *QueryPoolStatsRequest) (*QueryPoolStatsResponse, error)
}

// -----------------------------------------------------------------------
// gRPC query service descriptor
// -----------------------------------------------------------------------

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.compute_pool.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Params", Handler: _Query_Params_Handler},
		{MethodName: "Provider", Handler: _Query_Provider_Handler},
		{MethodName: "ActiveProviders", Handler: _Query_ActiveProviders_Handler},
		{MethodName: "CreditBalance", Handler: _Query_CreditBalance_Handler},
		{MethodName: "RedemptionTicket", Handler: _Query_RedemptionTicket_Handler},
		{MethodName: "PoolStats", Handler: _Query_PoolStats_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/compute_pool/v1/query.proto",
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Query/Params"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	})
}

func _Query_Provider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Provider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Query/Provider"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Provider(ctx, req.(*QueryProviderRequest))
	})
}

func _Query_ActiveProviders_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryActiveProvidersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ActiveProviders(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Query/ActiveProviders"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ActiveProviders(ctx, req.(*QueryActiveProvidersRequest))
	})
}

func _Query_CreditBalance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryCreditBalanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).CreditBalance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Query/CreditBalance"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).CreditBalance(ctx, req.(*QueryCreditBalanceRequest))
	})
}

func _Query_RedemptionTicket_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryRedemptionTicketRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).RedemptionTicket(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Query/RedemptionTicket"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).RedemptionTicket(ctx, req.(*QueryRedemptionTicketRequest))
	})
}

func _Query_PoolStats_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryPoolStatsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).PoolStats(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.compute_pool.v1.Query/PoolStats"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).PoolStats(ctx, req.(*QueryPoolStatsRequest))
	})
}

// RegisterQueryServer registers the QueryServer implementation with the gRPC server.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}
