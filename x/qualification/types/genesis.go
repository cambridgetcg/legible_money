package types

// DefaultParams returns default qualification module parameters.
func DefaultParams() Params {
	return Params{
		StakeLeaf:              "22000000",   // 22 LGM in ulgm
		StakeMid:               "222000000",  // 222 LGM in ulgm
		StakeTop:               "2222000000", // 2222 LGM in ulgm
		StakeLockPeriod:        7777,
		MinVerifications:       19,
		MinAccuracyBps:         7700,
		MinEndorsers:           3,
		EndorserStake:          "22000000", // 22 LGM in ulgm
		EndorsementDecayBlocks: 77777,
		StratumDiscountBps:     7000, // 70%
		InactivityExpiration:   777777,
		ProbationaryPeriod:     777,
		ProbationaryWeightBps:  5000, // 50%
		SuspensionAccuracyBps:  5000, // 50%
		SuspensionConsecutive:  5,
		RevocationChallenges:   3,
		MaxQualificationDepth:  7,
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	params := DefaultParams()
	return &GenesisState{
		Params:         &params,
		Qualifications: []*DomainQualification{},
		Endorsements:   []*QualificationEndorsement{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Params != nil {
		return gs.Params.Validate()
	}
	return nil
}
