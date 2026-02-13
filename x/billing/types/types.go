package types

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// --- Params Validation ---
// Methods below attach to protobuf-generated types from state.pb.go and genesis.pb.go.

// Validate performs parameter validation.
func (p Params) Validate() error {
	base := new(big.Int)
	if _, ok := base.SetString(p.BaseCostPerFact, 10); !ok || base.Sign() <= 0 {
		return fmt.Errorf("invalid base_cost_per_fact: %s", p.BaseCostPerFact)
	}
	if p.ConfidenceThreshold > 1000000 {
		return fmt.Errorf("confidence_threshold must be <= 1000000 bps")
	}
	if p.ConfidencePremiumBps > 1000000 {
		return fmt.Errorf("confidence_premium_bps must be <= 1000000 bps")
	}
	if p.FreshnessWindowBlocks == 0 {
		return fmt.Errorf("freshness_window_blocks must be positive")
	}
	if p.FreshnessDiscountBps > 1000000 {
		return fmt.Errorf("freshness_discount_bps must be <= 1000000 bps")
	}
	if p.QuoteValidityBlocks == 0 {
		return fmt.Errorf("quote_validity_blocks must be positive")
	}
	stake := new(big.Int)
	if _, ok := stake.SetString(p.MinProviderStake, 10); !ok || stake.Sign() <= 0 {
		return fmt.Errorf("invalid min_provider_stake: %s", p.MinProviderStake)
	}
	if p.ResearchShareBps == 0 || p.ResearchShareBps > 1000000 {
		return fmt.Errorf("research_share_bps must be between 1 and 1000000")
	}
	if p.ResearchMinBps > p.ResearchShareBps {
		return fmt.Errorf("research_min_bps must be <= research_share_bps")
	}
	return nil
}

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		BaseCostPerFact:       "1000000",
		ConfidenceThreshold:   500000,
		ConfidencePremiumBps:  200000,
		FreshnessWindowBlocks: 1000,
		FreshnessDiscountBps:  100000,
		QuoteValidityBlocks:   100,
		MinProviderStake:      "100000000",
		ResearchShareBps:      220000,
		ResearchMinBps:        50000,          // 5% starting rate
		ResearchRampBlocks:    34272 * 90,     // ~90 days to reach full 22%
	}
}

// --- Genesis state validation ---

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:    &p,
		Providers: []*Provider{},
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
// Message types (not in proto-generated files)
// -----------------------------------------------------------------------

// MsgRegisterProvider registers a new knowledge API provider.
type MsgRegisterProvider struct {
	Sender      string   `json:"sender"`
	Domains     []string `json:"domains"`
	StakeAmount string   `json:"stake_amount"`
}

func (msg *MsgRegisterProvider) Reset()        {}
func (msg *MsgRegisterProvider) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRegisterProvider) ProtoMessage()  {}

func (msg *MsgRegisterProvider) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterProvider) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if len(msg.Domains) == 0 {
		return fmt.Errorf("at least one domain is required")
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.StakeAmount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("stake amount must be a positive integer")
	}
	return nil
}

// MsgRegisterProviderResponse is the response for MsgRegisterProvider.
type MsgRegisterProviderResponse struct{}

func (msg *MsgRegisterProviderResponse) Reset()        {}
func (msg *MsgRegisterProviderResponse) String() string { return "" }
func (msg *MsgRegisterProviderResponse) ProtoMessage()  {}

// MsgDeregisterProvider removes a provider and refunds stake.
type MsgDeregisterProvider struct {
	Sender string `json:"sender"`
}

func (msg *MsgDeregisterProvider) Reset()        {}
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

func (msg *MsgDeregisterProviderResponse) Reset()        {}
func (msg *MsgDeregisterProviderResponse) String() string { return "" }
func (msg *MsgDeregisterProviderResponse) ProtoMessage()  {}

// MsgRequestQuote requests a price quote for querying facts.
type MsgRequestQuote struct {
	Sender   string   `json:"sender"`
	Provider string   `json:"provider"`
	FactIds  []string `json:"fact_ids"`
}

func (msg *MsgRequestQuote) Reset()        {}
func (msg *MsgRequestQuote) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRequestQuote) ProtoMessage()  {}

func (msg *MsgRequestQuote) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRequestQuote) ValidateBasic() error {
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
	if len(msg.FactIds) == 0 {
		return fmt.Errorf("at least one fact_id is required")
	}
	return nil
}

// MsgRequestQuoteResponse is the response for MsgRequestQuote.
type MsgRequestQuoteResponse struct {
	Quote APIQuote `json:"quote"`
}

func (msg *MsgRequestQuoteResponse) Reset()        {}
func (msg *MsgRequestQuoteResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgRequestQuoteResponse) ProtoMessage()  {}

// MsgExecutePayment executes a quoted payment for knowledge access.
type MsgExecutePayment struct {
	Sender  string `json:"sender"`
	QuoteId uint64 `json:"quote_id"`
}

func (msg *MsgExecutePayment) Reset()        {}
func (msg *MsgExecutePayment) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgExecutePayment) ProtoMessage()  {}

func (msg *MsgExecutePayment) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgExecutePayment) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	return nil
}

// MsgExecutePaymentResponse is the response for MsgExecutePayment.
type MsgExecutePaymentResponse struct {
	RecordId uint64 `json:"record_id"`
}

func (msg *MsgExecutePaymentResponse) Reset()        {}
func (msg *MsgExecutePaymentResponse) String() string { return fmt.Sprintf("%+v", *msg) }
func (msg *MsgExecutePaymentResponse) ProtoMessage()  {}

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

// MsgServer defines the billing module message service.
type MsgServer interface {
	RegisterProvider(context.Context, *MsgRegisterProvider) (*MsgRegisterProviderResponse, error)
	DeregisterProvider(context.Context, *MsgDeregisterProvider) (*MsgDeregisterProviderResponse, error)
	RequestQuote(context.Context, *MsgRequestQuote) (*MsgRequestQuoteResponse, error)
	ExecutePayment(context.Context, *MsgExecutePayment) (*MsgExecutePaymentResponse, error)
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
}

// -----------------------------------------------------------------------
// gRPC service descriptor (hand-written, replaces proto generation)
// -----------------------------------------------------------------------

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "legible.billing.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "RegisterProvider", Handler: _Msg_RegisterProvider_Handler},
		{MethodName: "DeregisterProvider", Handler: _Msg_DeregisterProvider_Handler},
		{MethodName: "RequestQuote", Handler: _Msg_RequestQuote_Handler},
		{MethodName: "ExecutePayment", Handler: _Msg_ExecutePayment_Handler},
		{MethodName: "UpdateParams", Handler: _Msg_UpdateParams_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "legible/billing/v1/tx.proto",
}

func _Msg_RegisterProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterProvider)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RegisterProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.billing.v1.Msg/RegisterProvider"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RegisterProvider(ctx, req.(*MsgRegisterProvider))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.billing.v1.Msg/DeregisterProvider"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).DeregisterProvider(ctx, req.(*MsgDeregisterProvider))
	})
}

func _Msg_RequestQuote_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRequestQuote)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).RequestQuote(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.billing.v1.Msg/RequestQuote"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).RequestQuote(ctx, req.(*MsgRequestQuote))
	})
}

func _Msg_ExecutePayment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgExecutePayment)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ExecutePayment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.billing.v1.Msg/ExecutePayment"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ExecutePayment(ctx, req.(*MsgExecutePayment))
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
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/legible.billing.v1.Msg/UpdateParams"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	})
}

// RegisterMsgServer registers the MsgServer implementation with the gRPC server.
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}
