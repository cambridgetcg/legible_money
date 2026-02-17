package types

import "fmt"

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params: &p,
	}
}

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		MinAuditStake:         "22000000", // 22 LGM (in ulgm)
		AuditWindow:           34272,      // ~1 day (BLOCKS_PER_DAY)
		MinAuditsForConsensus: 3,
		AccuracyThreshold:     100000,      // 10% (basis points of 1000000)
		SlashRate:             100000,      // 10% of stake
		RewardRate:            50000,       // 5% of stake
		MinOracleStake:        "100000000", // 100 LGM (in ulgm)
		OracleSlashRate:       200000,      // 20% of stake
		MaxEvidencePerClaim:   10,
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}
