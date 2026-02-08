package types

import "fmt"

// DefaultParams returns default module parameters
func DefaultParams() Params {
	return Params{
		ResearchShareBps:           2200,          // 22%
		ResearchTreasuryAddress:    "",            // set in genesis
		BlockRewardAmount:          "11000000",    // 11 LGM base (scaled by validator participation)
		EmptyBlockRewardRate:       0,             // 0% for empty blocks (PoT alignment)
		MinValidatorsForFullReward: 22,            // target validator count
		VestingEnabled:             true,
		ReleasedClawbackRate:       3300,          // 33% of released clawed back on falsification
	}
}

// DefaultCategoryConfigs returns release curve configs based on scientometric research
// Half-lives based on citation half-life studies and replication crisis data
func DefaultCategoryConfigs() []CategoryConfig {
	return []CategoryConfig{
		{Category: CategoryAxiomatic, HalfLifeBlocks: 1_111_111, CliffBlocks: 11111, MaxRelease: 950000},
		{Category: CategoryFormalProof, HalfLifeBlocks: 555_555, CliffBlocks: 5555, MaxRelease: 920000},
		{Category: CategoryOnChain, HalfLifeBlocks: 222_222, CliffBlocks: 1111, MaxRelease: 900000},
		{Category: CategoryCryptographic, HalfLifeBlocks: 222_222, CliffBlocks: 3333, MaxRelease: 900000},
		{Category: CategoryComputational, HalfLifeBlocks: 333_333, CliffBlocks: 2222, MaxRelease: 880000},
		{Category: CategoryPeerReviewed, HalfLifeBlocks: 111_111, CliffBlocks: 5555, MaxRelease: 850000},
		{Category: CategoryReplicated, HalfLifeBlocks: 111_111, CliffBlocks: 3333, MaxRelease: 880000},
		{Category: CategoryOracleFeed, HalfLifeBlocks: 55_555, CliffBlocks: 555, MaxRelease: 800000},
		{Category: CategoryAttestation, HalfLifeBlocks: 77_777, CliffBlocks: 2222, MaxRelease: 800000},
		{Category: CategoryContested, HalfLifeBlocks: 22_222, CliffBlocks: 1111, MaxRelease: 600000},
	}
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		CategoryConfigs:  DefaultCategoryConfigs(),
		VestingSchedules: []VestingSchedule{},
	}
}

// Validate performs basic genesis state validation
func (gs GenesisState) Validate() error {
	if gs.Params.ResearchShareBps > 10000 {
		return fmt.Errorf("research share rate cannot exceed 100%% (10000 bps)")
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
