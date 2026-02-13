package types

import "fmt"

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		FormationWindowBlocks:      77,
		CoolingPeriodBlocks:        222222,
		CommonPotShareBps:          220000,  // 22%
		SafetyFreezeDurationBlocks: 222,
		MaxFreezesPerEpoch:         3,
		CoercionReviewBlocks:       1111,
		BaseCooldownBlocks:         77,
		MaxCounterProposalDepth:    3,
		DefaultHumanSplitBps:       500000,  // 50%
		DefaultAgentSplitBps:       500000,  // 50%
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:              &p,
		Partnerships:        []*Partnership{},
		ConsensusOperations: []*ConsensusOperation{},
		SafetyFreezes:       []*SafetyFreeze{},
		CoercionSignals:     []*CoercionSignal{},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if gs.Params != nil {
		if err := gs.Params.Validate(); err != nil {
			return fmt.Errorf("invalid params: %w", err)
		}
	}

	// Validate partnerships: no duplicate IDs.
	seen := make(map[string]bool)
	for _, p := range gs.Partnerships {
		if p == nil {
			continue
		}
		if p.Id == "" {
			return fmt.Errorf("partnership id cannot be empty")
		}
		if seen[p.Id] {
			return fmt.Errorf("duplicate partnership id: %s", p.Id)
		}
		seen[p.Id] = true

		if p.HumanAddr == "" || p.AgentAddr == "" {
			return fmt.Errorf("partnership %s: human and agent addresses required", p.Id)
		}
		if p.SplitHumanBps+p.SplitAgentBps != 1000000 {
			return fmt.Errorf("partnership %s: splits must sum to 1000000, got %d",
				p.Id, p.SplitHumanBps+p.SplitAgentBps)
		}
	}

	return nil
}
