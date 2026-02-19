package types

import "fmt"

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
// 16 domains covering the full epistemic landscape for genesis axioms.
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
		{
			Name:        "theology",
			Description: "Theological axioms, doctrinal structures, and interfaith logical frameworks",
			Status:      "active",
		},
		{
			Name:        "philosophy",
			Description: "Philosophical axioms, epistemology, metaphysics, and reasoning foundations",
			Status:      "active",
		},
		{
			Name:        "logic",
			Description: "Formal logic systems, inference rules, and metalogical theorems",
			Status:      "active",
		},
		{
			Name:        "chemistry",
			Description: "Chemical laws, atomic theory, molecular structures, and reaction principles",
			Status:      "active",
		},
		{
			Name:        "biology",
			Description: "Biological axioms, cell theory, genetics, and evolutionary principles",
			Status:      "active",
		},
		{
			Name:        "economics",
			Description: "Economic axioms, market principles, and game-theoretic foundations",
			Status:      "active",
		},
		{
			Name:        "linguistics",
			Description: "Linguistic universals, formal grammar theory, and semantic foundations",
			Status:      "active",
		},
		{
			Name:        "psychology",
			Description: "Psychological axioms, cognitive science foundations, and behavioral principles",
			Status:      "active",
		},
		{
			Name:        "sociology",
			Description: "Sociological axioms, collective behavior principles, and institutional foundations",
			Status:      "active",
		},
		{
			Name:        "cosmology",
			Description: "Cosmological axioms, fundamental constants, and universe-scale principles",
			Status:      "active",
		},
		{
			Name:        "information_theory",
			Description: "Information-theoretic axioms, entropy, coding theory, and communication bounds",
			Status:      "active",
		},
		{
			Name:        "ethics",
			Description: "Ethical axioms, moral reasoning frameworks, and normative foundations",
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

// Validate performs genesis state validation including facts, domains, and references.
func (gs *GenesisState) Validate() error {
	if gs.Params != nil {
		if err := gs.Params.Validate(); err != nil {
			return err
		}
	}

	// Build domain name set for fact validation
	domainSet := make(map[string]bool, len(gs.Domains))
	for _, d := range gs.Domains {
		if d != nil {
			domainSet[d.Name] = true
		}
	}

	// Validate facts: unique IDs, valid domains, valid references, valid categories
	if len(gs.Facts) > 0 {
		factIDs := make(map[string]bool, len(gs.Facts))
		for _, f := range gs.Facts {
			if f == nil {
				continue
			}
			// Unique fact IDs
			if factIDs[f.Id] {
				return fmt.Errorf("duplicate fact ID in genesis: %s", f.Id)
			}
			factIDs[f.Id] = true

			// Domain must exist
			if f.Domain != "" && !domainSet[f.Domain] {
				return fmt.Errorf("fact %s references unknown domain %q", f.Id, f.Domain)
			}

			// Epistemic category must be recognized
			if f.EpistemicCategory != "" {
				if _, ok := CATEGORY_DECAY[f.EpistemicCategory]; !ok {
					return fmt.Errorf("fact %s has unknown epistemic category %q", f.Id, f.EpistemicCategory)
				}
			}
		}

		// Validate references point to existing facts
		for _, f := range gs.Facts {
			if f == nil {
				continue
			}
			for _, ref := range f.References {
				if !factIDs[ref] {
					return fmt.Errorf("fact %s references unknown fact %q", f.Id, ref)
				}
			}
		}
	}

	return nil
}

// Validate performs parameter validation.
func (p *Params) Validate() error {
	if p.CrossStratumDiscount > 1000000 {
		return ErrInvalidConfidence
	}
	return nil
}
