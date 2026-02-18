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
