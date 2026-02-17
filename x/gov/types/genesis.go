package types

// DefaultParams returns default governance parameters.
func DefaultParams() Params {
	p := &Params{}
	p.VotingPeriodBlocks = 100000
	p.ReviewPeriodBlocks = 50000
	p.LastCallPeriodBlocks = 50000
	p.BaseCostPerVote = "1000000" // 1 LGM in ulgm
	p.MaxVotesPerVoter = 100
	p.SupportThresholdBps = 7500       // 75%
	p.QuorumThresholdBps = 1500        // 15%
	p.DisbursementStake = "2000000000" // 2,000 LGM in ulgm
	p.DisbursementTimelockBlocks = 50000
	p.DisbursementSupportBps = 8000    // 80%
	p.DisbursementQuorumBps = 2000     // 20%
	p.LipStakeConsensus = "1000000000" // 1,000 LGM in ulgm
	p.LipStakeEconomics = "800000000"  // 800 LGM in ulgm
	p.LipStakeInterface = "400000000"  // 400 LGM in ulgm
	p.LipStakeProcess = "200000000"    // 200 LGM in ulgm
	return *p
}

// DefaultGenesis returns default genesis state.
func DefaultGenesis() *GenesisState {
	params := DefaultParams()
	return &GenesisState{
		Params:         &params,
		Lips:           []*LIP{},
		Votes:          []*QuadraticVote{},
		Disbursements:  []*DisbursementProposal{},
		NextLipNumber:  1,
		NextDisbNumber: 1,
	}
}

// Validate validates genesis state.
func (gs GenesisState) Validate() error {
	if gs.Params != nil {
		return gs.Params.Validate()
	}
	return nil
}
