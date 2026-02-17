package types

import "fmt"

// DefaultParams returns default staking module parameters.
func DefaultParams() Params {
	return Params{
		MaxApprenticeValidators: 0,          // deprecated: Sybil defense via MinSelfDelegation
		UnbondingPeriod:         268560,     // ~7 days
		VirtualStake:            "11000000", // 11 LGM in ulgm (H-3 fix)
		MaxValidators:           100,        // max block producers (tier 2+ only); verifiers are uncapped
		MinSelfDelegation:       "111000",   // 0.111 LGM — required to register as validator (Sybil defense)
		// Anti-grief parameters (R1-R4 remediation)
		MaxSlashesPerEpoch:      2,        // R2: max 2 slashes per validator per epoch
		SlashDecayPeriodBlocks:  34272,    // R1: decay 1 slash per epoch (~1 day)
		MaxSlashCountDeactivate: 3,        // slashes to deactivate
		MinStakeForVerification: "111000", // R3: 0.111 LGM min real stake for VRF eligibility (tier 0 threshold)
		SlashEscalationBps:      100000,   // 10% escalation per prior slash (in 1M BPS)
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:           &p,
		Validators:       []*LegibleValidator{},
		Delegations:      []*Delegation{},
		UnbondingEntries: []*UnbondingEntry{},
		TierConfigs:      DefaultTierConfigs(),
	}
}

// Validate performs basic genesis state validation.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	if gs.Params.UnbondingPeriod == 0 {
		return fmt.Errorf("unbonding period must be positive")
	}
	if len(gs.TierConfigs) != 4 {
		return fmt.Errorf("expected 4 tier configs, got %d", len(gs.TierConfigs))
	}
	return nil
}
