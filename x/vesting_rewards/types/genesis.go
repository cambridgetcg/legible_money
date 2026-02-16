package types

import (
	"fmt"
)

// DefaultParams returns default module parameters
func DefaultParams() *Params {
	return &Params{
		ResearchShareBps:           70000,         // 7%
		ResearchTreasuryAddress:    "",            // set in genesis
		BlockRewardAmount:          "3000000",     // 3 LGM base (scaled by validator participation)
		EmptyBlockRewardRate:       0,             // 0% for empty blocks (PoT alignment)
		MinValidatorsForFullReward: 22,            // target validator count
		VestingEnabled:             true,
		ReleasedClawbackRate:       3300,          // 33% of released clawed back on falsification
		InitialFundBalance:        "0",               // Pure PoT: no pre-minted fund
		BlocksPerEpoch:            240000,            // ~1 week at 2521ms blocks
		DecayFactorBps:            850000,            // 0.85 per epoch (bps on 1,000,000 scale)
		FloorReward:               "100000",          // 0.1 LGM in ulgm
		FundHealthThresholdBps:    100000,            // 10% (bps on 1,000,000 scale) [unused in pure PoT]
		ApiSuppressionExponent:    2,                  // quadratic (deferred)
	}
}

// DefaultCategoryConfigs returns release curve configs based on scientometric research
// Half-lives based on citation half-life studies and replication crisis data
func DefaultCategoryConfigs() []*CategoryConfig {
	return []*CategoryConfig{
		{Category: string(CategoryAxiomatic), HalfLifeBlocks: 1_111_111, CliffBlocks: 11111, MaxRelease: 950000},
		{Category: string(CategoryFormalProof), HalfLifeBlocks: 555_555, CliffBlocks: 5555, MaxRelease: 920000},
		{Category: string(CategoryOnChain), HalfLifeBlocks: 222_222, CliffBlocks: 1111, MaxRelease: 900000},
		{Category: string(CategoryCryptographic), HalfLifeBlocks: 222_222, CliffBlocks: 3333, MaxRelease: 900000},
		{Category: string(CategoryComputational), HalfLifeBlocks: 333_333, CliffBlocks: 2222, MaxRelease: 880000},
		{Category: string(CategoryPeerReviewed), HalfLifeBlocks: 111_111, CliffBlocks: 5555, MaxRelease: 850000},
		{Category: string(CategoryReplicated), HalfLifeBlocks: 111_111, CliffBlocks: 3333, MaxRelease: 880000},
		{Category: string(CategoryOracleFeed), HalfLifeBlocks: 55_555, CliffBlocks: 555, MaxRelease: 800000},
		{Category: string(CategoryAttestation), HalfLifeBlocks: 77_777, CliffBlocks: 2222, MaxRelease: 800000},
		{Category: string(CategoryContested), HalfLifeBlocks: 22_222, CliffBlocks: 1111, MaxRelease: 600000},
	}
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		CategoryConfigs:  DefaultCategoryConfigs(),
		VestingSchedules: []*VestingSchedule{},
	}
}

// Validate performs basic genesis state validation
func (gs *GenesisState) Validate() error {
	if gs.Params != nil {
		if gs.Params.ResearchShareBps > 1000000 {
			return fmt.Errorf("research share rate cannot exceed 100%% (1000000 bps)")
		}
		if err := validateFundParams(gs.Params); err != nil {
			return err
		}
	}
	if len(gs.CategoryConfigs) == 0 {
		return fmt.Errorf("at least one category config required")
	}
	for _, cfg := range gs.CategoryConfigs {
		if cfg.MaxRelease > 1000000 {
			return fmt.Errorf("max release for %s cannot exceed 100%% (1000000 bps)", cfg.Category)
		}
		if cfg.HalfLifeBlocks == 0 {
			return fmt.Errorf("half-life for %s must be positive", cfg.Category)
		}
	}
	return nil
}

// validateFundParams validates the block reward parameters.
func validateFundParams(p *Params) error {
	if p.BlocksPerEpoch == 0 {
		return fmt.Errorf("blocks_per_epoch must be positive")
	}
	if p.DecayFactorBps == 0 {
		return fmt.Errorf("decay_factor_bps must be positive")
	}
	if p.DecayFactorBps > 1000000 {
		return fmt.Errorf("decay_factor_bps cannot exceed 1000000 (1.0)")
	}
	if p.FundHealthThresholdBps > 1000000 {
		return fmt.Errorf("fund_health_threshold_bps cannot exceed 1000000 (100%%)")
	}
	if p.BlockRewardAmount == "" || p.BlockRewardAmount == "0" {
		return fmt.Errorf("block_reward_amount must be positive")
	}
	if p.ResearchShareBps > 0 && p.ResearchTreasuryAddress == "" {
		// Research tax is enabled but no treasury address is configured.
		// Block rewards still route to the research_fund module account, but
		// governance disbursements need a treasury address to function.
		// Allow for devnet (module account suffices) but warn for production.
	}
	return nil
}
