package types

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:  DefaultParams(),
		Domains: DefaultDomains(),
	}
}

// DefaultDomains returns the bootstrap domains for the knowledge module.
// These are included in genesis so claims have domains to reference from block 1.
func DefaultDomains() []OntologicalDomain {
	return []OntologicalDomain{
		{
			Name:        "mathematics",
			Description: "Formal mathematical truths, proofs, and theorems",
			Status:      "active",
		},
		{
			Name:        "physics",
			Description: "Physical laws, constants, and empirical observations about the natural world",
			Status:      "active",
		},
		{
			Name:        "computer_science",
			Description: "Algorithms, data structures, protocols, and computational theory",
			Status:      "active",
		},
		{
			Name:        "general",
			Description: "General knowledge claims not fitting a specific domain",
			Status:      "active",
		},
	}
}

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		OntologyVersion:                1,
		CrossStratumDiscount:           50000, // 5%
		MinDomainProposalStake:         "1000000", // 1 LGM
		DomainVotingPeriod:             34272, // ~1 day (BLOCKS_PER_DAY)
		MinEndorsements:                3,
		AllowPartialCoverage:           true,
		PartialCoveragePenalty:         100000, // 10%
		LowerStratumVetoWeight:         200000, // 20%
		CrossStratumAgreementThreshold: 670000, // 67%
		MinClaimTextLength:             20,
		MaxClaimRewardMultiplier:       2,
		CommitPhaseDuration:            4,       // 4 blocks (~10s)
		RevealPhaseDuration:            4,       // 4 blocks (~10s)
		AggregationPhaseDuration:       3,       // 3 blocks (~7.5s)
		MinVerifiers:                   3,
		AcceptanceThreshold:            770000,  // 77% agreement required
		QuorumThreshold:                660000,  // 66% participation required
		MaxValidatorsPerRound:          22,
		InvalidClaimSlashBps:           220000,  // 22% slash on rejection (matches TS)
		VerificationRewardAmount:       100000,  // 0.1 LGM per correct verification
		ResearchTaxRate:                220000,  // 22% of rewards taxed to research fund (matches TS)
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

// Validate performs parameter validation.
func (p Params) Validate() error {
	if p.CrossStratumDiscount > 1000000 {
		return ErrInvalidConfidence
	}
	return nil
}
