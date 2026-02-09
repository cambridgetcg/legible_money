package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Enums ---

// PotStatus represents the lifecycle state of a claiming pot.
type PotStatus string

const (
	PotStatusActive    PotStatus = "active"
	PotStatusExhausted PotStatus = "exhausted"
	PotStatusExpired   PotStatus = "expired"
	PotStatusClosed    PotStatus = "closed"
)

// --- Eligibility ---

// PotEligibility defines who can claim from a pot.
type PotEligibility struct {
	MinAccountAge             uint64   `json:"min_account_age"`              // Blocks (0 = no restriction)
	MinActivityScore          string   `json:"min_activity_score"`           // "0.0"-"1.0" (0 = no restriction)
	MaxBalance                string   `json:"max_balance,omitempty"`        // Max balance to qualify (ulgm, "" = no cap)
	RequiredVerificationLevel uint32   `json:"required_verification_level"`  // 0 = no restriction
	ExcludedTypes             []string `json:"excluded_types"`               // e.g. ["system","contract"]
	Allowlist                 []string `json:"allowlist"`                    // If non-empty, ONLY these addresses can claim
}

// --- Config ---

// PotConfig is immutable after creation.
type PotConfig struct {
	AmountPerClaim      string         `json:"amount_per_claim"`       // ulgm per claim
	MaxClaims           uint64         `json:"max_claims"`             // 0 = unlimited
	MaxClaimsPerAccount uint64         `json:"max_claims_per_account"` // Typically 1
	Eligibility         PotEligibility `json:"eligibility"`
	ExpiresAtBlock      uint64         `json:"expires_at_block,omitempty"` // 0 = no expiration
	CreatorCanClose     bool           `json:"creator_can_close"`
}

// --- Pot State ---

// ClaimingPot represents the full pot state.
type ClaimingPot struct {
	PotId          string    `json:"pot_id"`
	Creator        string    `json:"creator"`          // bech32 address
	Config         PotConfig `json:"config"`
	TotalDeposited string    `json:"total_deposited"`  // ulgm cumulative
	TotalClaimed   string    `json:"total_claimed"`    // ulgm cumulative
	Balance        string    `json:"balance"`          // ulgm current available
	ClaimCount     uint64    `json:"claim_count"`
	CreatedAtBlock uint64    `json:"created_at_block"`
	ClosedAtBlock  uint64    `json:"closed_at_block,omitempty"`
	Status         PotStatus `json:"status"`
}

func (p *ClaimingPot) Reset()         { *p = ClaimingPot{} }
func (p *ClaimingPot) String() string { return fmt.Sprintf("ClaimingPot{%s}", p.PotId) }
func (p *ClaimingPot) ProtoMessage()  {}

// --- Claim Record ---

// ClaimRecord tracks per-account claims for a pot.
type ClaimRecord struct {
	PotId          string `json:"pot_id"`
	Claimer        string `json:"claimer"` // bech32
	ClaimCount     uint64 `json:"claim_count"`
	TotalAmount    string `json:"total_amount"` // ulgm
	LastClaimBlock uint64 `json:"last_claim_block"`
}

// --- Params ---

// Params defines the claiming_pot module parameters.
type Params struct {
	MaxPotsPerCreator uint64 `json:"max_pots_per_creator"` // Anti-spam
	MinInitialDeposit string `json:"min_initial_deposit"`  // ulgm
	MaxAllowlistSize  uint32 `json:"max_allowlist_size"`
	MinAmountPerClaim string `json:"min_amount_per_claim"` // ulgm
}

func (p *Params) Reset()         { *p = Params{} }
func (p *Params) String() string { return fmt.Sprintf("Params{max_pots:%d}", p.MaxPotsPerCreator) }
func (p *Params) ProtoMessage()  {}

func DefaultParams() Params {
	return Params{
		MaxPotsPerCreator: 100,
		MinInitialDeposit: "1000000",  // 1 LGM in ulgm
		MaxAllowlistSize:  10000,
		MinAmountPerClaim: "1000",     // 0.001 LGM in ulgm
	}
}

func (p Params) Validate() error {
	if p.MaxPotsPerCreator == 0 {
		return fmt.Errorf("max pots per creator must be positive")
	}
	if p.MaxAllowlistSize == 0 {
		return fmt.Errorf("max allowlist size must be positive")
	}
	return nil
}

// --- Genesis ---

// GenesisState defines the claiming_pot module genesis state.
type GenesisState struct {
	Params Params        `json:"params"`
	Pots   []ClaimingPot `json:"pots"`
	Claims []ClaimRecord `json:"claims"`
}

func (gs *GenesisState) Reset()         { *gs = GenesisState{} }
func (gs *GenesisState) String() string { return fmt.Sprintf("GenesisState{pots:%d}", len(gs.Pots)) }
func (gs *GenesisState) ProtoMessage()  {}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Pots:   []ClaimingPot{},
		Claims: []ClaimRecord{},
	}
}

func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

// --- Message Types ---

// MsgCreatePot creates a new claiming pot.
type MsgCreatePot struct {
	Creator        string    `json:"creator"`
	Config         PotConfig `json:"config"`
	InitialDeposit string    `json:"initial_deposit"` // ulgm
}

func (m *MsgCreatePot) Reset()         { *m = MsgCreatePot{} }
func (m *MsgCreatePot) String() string { return fmt.Sprintf("MsgCreatePot{%s}", m.Creator) }
func (m *MsgCreatePot) ProtoMessage()  {}

func (m *MsgCreatePot) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.Config.AmountPerClaim == "" || m.Config.AmountPerClaim == "0" {
		return fmt.Errorf("amount per claim must be positive")
	}
	if m.Config.MaxClaimsPerAccount == 0 {
		return fmt.Errorf("max claims per account must be positive")
	}
	if m.InitialDeposit == "" || m.InitialDeposit == "0" {
		return fmt.Errorf("initial deposit must be positive")
	}
	return nil
}

type MsgCreatePotResponse struct {
	PotId string `json:"pot_id"`
}

func (m *MsgCreatePotResponse) Reset()         { *m = MsgCreatePotResponse{} }
func (m *MsgCreatePotResponse) String() string { return fmt.Sprintf("MsgCreatePotResponse{%s}", m.PotId) }
func (m *MsgCreatePotResponse) ProtoMessage()  {}

// MsgFundPot adds funds to an existing pot.
type MsgFundPot struct {
	Funder string `json:"funder"`
	PotId  string `json:"pot_id"`
	Amount string `json:"amount"` // ulgm
}

func (m *MsgFundPot) Reset()         { *m = MsgFundPot{} }
func (m *MsgFundPot) String() string { return fmt.Sprintf("MsgFundPot{%s, %s}", m.PotId, m.Amount) }
func (m *MsgFundPot) ProtoMessage()  {}

func (m *MsgFundPot) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Funder); err != nil {
		return fmt.Errorf("invalid funder address: %w", err)
	}
	if m.PotId == "" {
		return fmt.Errorf("pot id cannot be empty")
	}
	if m.Amount == "" || m.Amount == "0" {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

type MsgFundPotResponse struct{}

func (m *MsgFundPotResponse) Reset()         { *m = MsgFundPotResponse{} }
func (m *MsgFundPotResponse) String() string { return "MsgFundPotResponse{}" }
func (m *MsgFundPotResponse) ProtoMessage()  {}

// MsgClaimFromPot claims from an eligible pot.
type MsgClaimFromPot struct {
	Claimer string `json:"claimer"`
	PotId   string `json:"pot_id"`
}

func (m *MsgClaimFromPot) Reset()         { *m = MsgClaimFromPot{} }
func (m *MsgClaimFromPot) String() string { return fmt.Sprintf("MsgClaimFromPot{%s, %s}", m.PotId, m.Claimer) }
func (m *MsgClaimFromPot) ProtoMessage()  {}

func (m *MsgClaimFromPot) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Claimer); err != nil {
		return fmt.Errorf("invalid claimer address: %w", err)
	}
	if m.PotId == "" {
		return fmt.Errorf("pot id cannot be empty")
	}
	return nil
}

type MsgClaimFromPotResponse struct {
	Amount string `json:"amount"` // ulgm claimed
}

func (m *MsgClaimFromPotResponse) Reset()         { *m = MsgClaimFromPotResponse{} }
func (m *MsgClaimFromPotResponse) String() string { return fmt.Sprintf("MsgClaimFromPotResponse{%s}", m.Amount) }
func (m *MsgClaimFromPotResponse) ProtoMessage()  {}

// MsgClosePot closes a pot (creator only, if allowed by config).
type MsgClosePot struct {
	Creator string `json:"creator"`
	PotId   string `json:"pot_id"`
}

func (m *MsgClosePot) Reset()         { *m = MsgClosePot{} }
func (m *MsgClosePot) String() string { return fmt.Sprintf("MsgClosePot{%s}", m.PotId) }
func (m *MsgClosePot) ProtoMessage()  {}

func (m *MsgClosePot) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.PotId == "" {
		return fmt.Errorf("pot id cannot be empty")
	}
	return nil
}

type MsgClosePotResponse struct {
	RefundedAmount string `json:"refunded_amount"` // ulgm returned to creator
}

func (m *MsgClosePotResponse) Reset()         { *m = MsgClosePotResponse{} }
func (m *MsgClosePotResponse) String() string { return fmt.Sprintf("MsgClosePotResponse{%s}", m.RefundedAmount) }
func (m *MsgClosePotResponse) ProtoMessage()  {}

// --- Service Interfaces ---

// MsgServer defines the claiming_pot message handler interface.
type MsgServer interface {
	CreatePot(context.Context, *MsgCreatePot) (*MsgCreatePotResponse, error)
	FundPot(context.Context, *MsgFundPot) (*MsgFundPotResponse, error)
	ClaimFromPot(context.Context, *MsgClaimFromPot) (*MsgClaimFromPotResponse, error)
	ClosePot(context.Context, *MsgClosePot) (*MsgClosePotResponse, error)
}
