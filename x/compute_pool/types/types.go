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
// Proto type methods (structs defined in state.pb.go, genesis.pb.go, query.pb.go)
// -----------------------------------------------------------------------

// Validate performs parameter validation.
func (p Params) Validate() error {
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
func DefaultParams() Params {
	return Params{
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
	PricePerCu  string `json:"price_per_cu"`
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
	if _, ok := price.SetString(msg.PricePerCu, 10); !ok || price.Sign() <= 0 {
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
	CuAmount uint64 `json:"cu_amount"`
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
	if msg.CuAmount == 0 {
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
	ActualCu uint64 `json:"actual_cu"`
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

// MsgUpdateParams is the governance message to update module params.
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

// MsgUpdateParamsResponse is the response type for MsgUpdateParams.
type MsgUpdateParamsResponse struct{}

func (*MsgUpdateParams) ProtoMessage()         {}
func (*MsgUpdateParams) Reset()                {}
func (*MsgUpdateParams) String() string        { return "MsgUpdateParams" }
func (*MsgUpdateParamsResponse) ProtoMessage()  {}
func (*MsgUpdateParamsResponse) Reset()         {}
func (*MsgUpdateParamsResponse) String() string { return "MsgUpdateParamsResponse" }

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
