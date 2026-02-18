package types

import "fmt"

// DefaultParams returns default staking module parameters.
func DefaultParams() *Params {
	return &Params{
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
		// Tier configs + parameterized constants (P1-A)
		TierConfigs:                DefaultTierConfigs(),
		ReputationCorrectDelta:     100,   // +0.01% reputation per correct verification
		ReputationIncorrectDelta:   200,   // -0.02% reputation per incorrect verification
		ReputationSlashDelta:       10000, // -1% reputation per slash
		RedelegationCooldownBlocks: 1111,  // ~46 min at 2.521s/block
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:           p,
		Validators:       []*LegibleValidator{},
		Delegations:      []*Delegation{},
		UnbondingEntries: []*UnbondingEntry{},
		TierConfigs:      p.TierConfigs, // sourced from params (canonical location)
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
	// Tier configs must exist in at least one location (params or legacy genesis field)
	hasTierConfigsInParams := len(gs.Params.TierConfigs) == 4
	hasTierConfigsLegacy := len(gs.TierConfigs) == 4
	if !hasTierConfigsInParams && !hasTierConfigsLegacy {
		return fmt.Errorf("expected 4 tier configs (in params or genesis), got params=%d genesis=%d",
			len(gs.Params.TierConfigs), len(gs.TierConfigs))
	}
	return nil
}
