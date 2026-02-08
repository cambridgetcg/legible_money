package types

import "fmt"

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		PeriodBlocks:              34272,  // ~1 day at 2521ms block time
		PeriodsUntilExpiry:        365,    // ~1 year
		DustThreshold:             "1000", // 1000 ulgm
		SweepBatchSize:            100,
		RAIDistributionRate:       34272,  // daily
		FreshExpirationOnTransfer: false,  // preserve sender's period-of-origin
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		Accounts: []ExpirableAccount{},
		RAI: RAIHoldingAccount{
			Balance:          "0",
			TotalReceived:    "0",
			TotalDistributed: "0",
			LastDistBlock:    0,
		},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Validate accounts
	seen := make(map[string]bool)
	for _, acct := range gs.Accounts {
		if acct.Address == "" {
			return fmt.Errorf("account address cannot be empty")
		}
		if seen[acct.Address] {
			return fmt.Errorf("duplicate account address: %s", acct.Address)
		}
		seen[acct.Address] = true

		switch acct.AccountType {
		case "standard", "validator", "foundation", "research":
			// valid
		default:
			return fmt.Errorf("invalid account type %q for address %s", acct.AccountType, acct.Address)
		}
	}

	return nil
}
