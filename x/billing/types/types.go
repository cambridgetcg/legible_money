package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Params Validation ---
// Methods below attach to protobuf-generated types from state.pb.go and genesis.pb.go.

// Validate performs parameter validation.
func (p *Params) Validate() error {
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
	if p.FoundationAddress != "" {
		if _, err := sdk.AccAddressFromBech32(p.FoundationAddress); err != nil {
			return fmt.Errorf("invalid foundation_address: %w", err)
		}
	}
	return nil
}

// DefaultParams returns default module parameters.
func DefaultParams() *Params {
	return &Params{
		BaseCostPerFact:       "1000000",
		ConfidenceThreshold:   500000,
		ConfidencePremiumBps:  200000,
		FreshnessWindowBlocks: 1000,
		FreshnessDiscountBps:  100000,
		QuoteValidityBlocks:   100,
		MinProviderStake:      "100000000",
		ResearchShareBps:      70000,
		ResearchMinBps:        50000,      // 5% starting rate
		ResearchRampBlocks:    34272 * 90, // ~90 days to reach full 7%
		FoundationAddress:     "",         // empty = module account fallback
	}
}

// --- Genesis state validation ---

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Providers: []*Provider{},
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
// GetSigners / ValidateBasic for proto-generated Msg types
// -----------------------------------------------------------------------

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
	if _, ok := amt.SetString(msg.Stake, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("stake amount must be a positive integer")
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
