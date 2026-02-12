package types

import (
	"fmt"
	"math/big"
)

// DefaultParams returns default module parameters
func DefaultParams() *Params {
	return &Params{
		ResearchShareBps:           220000,        // 22%
		ResearchTreasuryAddress:    "",            // set in genesis
		BlockRewardAmount:          "11000000",    // 11 LGM base (scaled by validator participation)
		EmptyBlockRewardRate:       0,             // 0% for empty blocks (PoT alignment)
		MinValidatorsForFullReward: 22,            // target validator count
		VestingEnabled:             true,
		ReleasedClawbackRate:       3300,          // 33% of released clawed back on falsification
		InitialFundBalance:        "133333333200000", // 133,333,333.2 LGM in ulgm (60% of 222,222,222)
		BlocksPerEpoch:            240000,            // ~1 week at 2521ms blocks
		DecayFactorBps:            900000,            // 0.9 per epoch (bps on 1,000,000 scale)
		FloorReward:               "100000",          // 0.1 LGM in ulgm
		FundHealthThresholdBps:    100000,            // 10% (bps on 1,000,000 scale)
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

// validateFundParams validates the block reward fund parameters.
func validateFundParams(p *Params) error {
	if p.InitialFundBalance == "" || p.InitialFundBalance == "0" {
		return fmt.Errorf("initial_fund_balance is required (no infinite mint)")
	}
	fund := new(big.Int)
	if _, ok := fund.SetString(p.InitialFundBalance, 10); !ok {
		return fmt.Errorf("initial_fund_balance is not a valid integer: %s", p.InitialFundBalance)
	}
	if fund.Sign() < 0 {
		return fmt.Errorf("initial_fund_balance must be non-negative")
	}
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
	return nil
}
