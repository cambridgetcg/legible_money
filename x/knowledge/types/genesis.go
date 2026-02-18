package types

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	params := DefaultParams()
	return &GenesisState{
		Params:  &params,
		Domains: DefaultDomains(),
	}
}

// DefaultDomains returns the bootstrap domains for the knowledge module.
// These are included in genesis so claims have domains to reference from block 1.
func DefaultDomains() []*OntologicalDomain {
	return []*OntologicalDomain{
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
// Note: proto-generated Params has a subset of fields (18 of 29).
// Extra fields (slash rates, adversarial params) are tracked in keeper state.
func DefaultParams() Params {
	return Params{
		OntologyVersion:                1,
		CrossStratumDiscount:           50000,     // 5%
		MinDomainProposalStake:         "1000000", // 1 LGM
		DomainVotingPeriod:             34272,     // ~1 day (BLOCKS_PER_DAY)
		MinEndorsements:                3,
		AllowPartialCoverage:           true,
		PartialCoveragePenalty:         100000, // 10%
		LowerStratumVetoWeight:         200000, // 20%
		CrossStratumAgreementThreshold: 670000, // 67%
		MinClaimTextLength:             20,
		MaxClaimRewardMultiplier:       2,
		CommitPhaseDuration:            4, // 4 blocks (~10s)
		RevealPhaseDuration:            4, // 4 blocks (~10s)
		AggregationPhaseDuration:       3, // 3 blocks (~7.5s)
		MinVerifiers:                   3,
		AcceptanceThreshold:            770000, // 77% agreement required
		QuorumThreshold:                660000, // 66% participation required
		MaxValidatorsPerRound:          22,
		InvalidClaimSlashBps:           220000,  // 22% slash on rejected claims
		VerificationRewardAmount:       3000000, // 3 LGM base reward per correct verification (ulgm)
		// Adversarial verification defaults (must match hardcoded constants)
		ProvisionalThreshold:           ProvisionalThresholdDefault,
		BaseChallengeDuration:          BaseChallengeDurationDefault,
		MinChallengeStake:              "11000000", // 11 LGM in ulgm
		ConfidenceGrowthPerEpoch:       ConfidenceGrowthPerEpochDefault,
		MaxSurvivalConfidence:          MaxSurvivalConfidenceDefault,
		FailedChallengeSlashBps:        FailedChallengeSlashBpsDefault,
		SuccessfulChallengeRewardBps:   SuccessfulChallengeRewardBpsDefault,
		AdversarialVerificationEnabled: true,
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if gs.Params != nil {
		return gs.Params.Validate()
	}
	return nil
}

// Validate performs parameter validation.
func (p Params) Validate() error {
	if p.CrossStratumDiscount > 1000000 {
		return ErrInvalidConfidence
	}
	return nil
}
