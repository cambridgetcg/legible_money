package types

import "fmt"

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// DefaultTierConfigs returns the default tier configurations.
// Tier 1: 3 arbiters, 111 LGM bond, 1-day evidence, 2-day voting
// Tier 2: 7 arbiters, 555 LGM bond, 2-day evidence, 4-day voting
// Tier 3: 15 arbiters, 1111 LGM bond, 3-day evidence, 7-day voting
func DefaultTierConfigs() []*TierConfig {
	return []*TierConfig{
		{
			Tier:              uint32(TierBasic),
			ArbiterCount:      3,
			MinBond:           "111000000", // 111 LGM in ulgm
			EvidencePeriod:    34272,       // ~1 day (BLOCKS_PER_DAY)
			VotingPeriod:      68544,       // ~2 days
			QuorumThreshold:   660000,      // 66%
			MajorityThreshold: 770000,      // 77%
		},
		{
			Tier:              uint32(TierEscalated),
			ArbiterCount:      7,
			MinBond:           "555000000", // 555 LGM in ulgm
			EvidencePeriod:    68544,       // ~2 days
			VotingPeriod:      137088,      // ~4 days
			QuorumThreshold:   660000,      // 66%
			MajorityThreshold: 770000,      // 77%
		},
		{
			Tier:              uint32(TierFinal),
			ArbiterCount:      15,
			MinBond:           "1111000000", // 1111 LGM in ulgm
			EvidencePeriod:    102816,       // ~3 days
			VotingPeriod:      239904,       // ~7 days
			QuorumThreshold:   660000,       // 66%
			MajorityThreshold: 770000,       // 77%
		},
	}
}

// DefaultParams returns the default module parameters.
func DefaultParams() *Params {
	return &Params{
		TierConfigs:       DefaultTierConfigs(),
		MaxActiveDisputes: 3,      // max 3 active disputes per fact
		EscalationDelay:   17136,  // ~0.5 day before escalation allowed
		SlashRateLoser:    500000, // 50% of bond slashed from loser
		RewardRateWinner:  300000, // 30% of slashed bond to winner
		ArbiterRewardRate: 200000, // 20% of slashed bond to arbiters
	}
}

// Validate performs basic genesis state validation.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}
