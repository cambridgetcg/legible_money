package types

import "fmt"

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	params := DefaultParams()
	return &GenesisState{
		Params:  &params,
		Strata:  DefaultStrata(),
		Domains: DefaultDomains(),
	}
}

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	return Params{
		MinProposalStake:     "1000000", // 1 LGM in ulgm
		ProposalVotingPeriod: 34272,     // ~1 day (BLOCKS_PER_DAY at 2521ms block time)
		MinEndorsements:      3,         // minimum 3 votes for approval
		CrossStratumDiscount: 50000,     // 5% discount for cross-stratum references
		MaxDomainsPerStratum: 100,       // maximum 100 domains per stratum
		AllowNewStrata:       false,     // new strata require governance
	}
}

// DefaultStrata returns the 7-level ontological strata with Goedelian logic properties.
//
// The strata encode the fundamental epistemological hierarchy:
//   - Lower strata (0-2) are formally verifiable with algorithmic completeness
//   - Middle strata (3-4) involve computation and empirical observation
//   - Upper strata (5-6) rely on human testimony and evidence weighting
//
// MaxConfidence and DecayRate are in basis points where 1000000 = 1.0.
func DefaultStrata() []*StratumProperties {
	return []*StratumProperties{
		{
			Stratum:          uint32(StratumAxiomatic),
			Name:             "axiomatic",
			Description:      "Mathematical axioms, tautologies, and logical foundations. Complete and decidable within their formal system.",
			Complete:         true,
			Decidable:        true,
			GoedelApplies:    false,
			ConsistencyProof: "internal",
			MaxConfidence:    1000000, // 1.0 — axioms are definitionally true
			DecayRate:        0,       // axioms do not decay
		},
		{
			Stratum:          uint32(StratumFormal),
			Name:             "formal",
			Description:      "Formal proofs and theorems derived from axioms. Complete in restricted domains (propositional, Presburger).",
			Complete:         true,
			Decidable:        true,
			GoedelApplies:    false,
			ConsistencyProof: "internal",
			MaxConfidence:    1000000, // 1.0 — formal proofs are provably correct
			DecayRate:        1,       // negligible decay for proof validity
		},
		{
			Stratum:          uint32(StratumProtocol),
			Name:             "protocol",
			Description:      "Blockchain-verifiable, deterministic facts. Verifiable by any full node via state replay.",
			Complete:         false,
			Decidable:        true,
			GoedelApplies:    false,
			ConsistencyProof: "external",
			MaxConfidence:    990000, // 0.99 — deterministic but depends on protocol correctness
			DecayRate:        5,      // minimal decay; protocol facts remain valid
		},
		{
			Stratum:          uint32(StratumComputational),
			Name:             "computational",
			Description:      "Computation results that are reproducible but may involve undecidable problems (halting problem).",
			Complete:         false,
			Decidable:        false,
			GoedelApplies:    true,
			ConsistencyProof: "assumed",
			MaxConfidence:    980000, // 0.98 — reproducible but Goedel limits apply
			DecayRate:        10,     // slow decay; computations remain reproducible
		},
		{
			Stratum:          uint32(StratumEmpirical),
			Name:             "empirical",
			Description:      "Scientific observations, experimental results. Falsifiable and subject to revision by new evidence.",
			Complete:         false,
			Decidable:        false,
			GoedelApplies:    false,
			ConsistencyProof: "external",
			MaxConfidence:    950000, // 0.95 — strong but revisable by new evidence
			DecayRate:        50,     // moderate decay; empirical knowledge needs reverification
		},
		{
			Stratum:          uint32(StratumHistorical),
			Name:             "historical",
			Description:      "Historical records and evidence-based claims about past events. Dependent on source reliability.",
			Complete:         false,
			Decidable:        false,
			GoedelApplies:    false,
			ConsistencyProof: "external",
			MaxConfidence:    900000, // 0.90 — reliable but not independently reproducible
			DecayRate:        100,    // faster decay; historical sources degrade
		},
		{
			Stratum:          uint32(StratumTestimonial),
			Name:             "testimonial",
			Description:      "Human attestations and trust-weighted claims. Lowest formal verifiability, highest social dependency.",
			Complete:         false,
			Decidable:        false,
			GoedelApplies:    false,
			ConsistencyProof: "external",
			MaxConfidence:    800000, // 0.80 — trust-dependent ceiling
			DecayRate:        200,    // rapid decay; testimony needs frequent reaffirmation
		},
	}
}

// DefaultDomains returns bootstrap domains for the ontology module.
// These are included in genesis so claims have domains to reference from block 1.
func DefaultDomains() []*Domain {
	return []*Domain{
		{
			Name:        "logic",
			DisplayName: "Logic & Foundations",
			Description: "Propositional logic, predicate logic, set theory foundations, and metamathematics",
			Stratum:     uint32(StratumAxiomatic),
			Status:      "active",
		},
		{
			Name:        "mathematics",
			DisplayName: "Mathematics",
			Description: "Formal mathematical truths, proofs, and theorems",
			Stratum:     uint32(StratumFormal),
			Status:      "active",
		},
		{
			Name:        "protocol",
			DisplayName: "Blockchain Protocol",
			Description: "On-chain state, transaction results, and protocol-verifiable facts",
			Stratum:     uint32(StratumProtocol),
			Status:      "active",
		},
		{
			Name:        "computer_science",
			DisplayName: "Computer Science",
			Description: "Algorithms, data structures, protocols, and computational theory",
			Stratum:     uint32(StratumComputational),
			Status:      "active",
		},
		{
			Name:        "physics",
			DisplayName: "Physics",
			Description: "Physical laws, constants, and empirical observations about the natural world",
			Stratum:     uint32(StratumEmpirical),
			Status:      "active",
		},
		{
			Name:        "general",
			DisplayName: "General Knowledge",
			Description: "General knowledge claims not fitting a specific domain",
			Stratum:     uint32(StratumEmpirical),
			Status:      "active",
		},
		{
			Name:        "history",
			DisplayName: "History",
			Description: "Historical records, events, and evidence-based accounts of the past",
			Stratum:     uint32(StratumHistorical),
			Status:      "active",
		},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if gs.Params != nil {
		if err := gs.Params.Validate(); err != nil {
			return fmt.Errorf("invalid params: %w", err)
		}
	}

	// Validate strata
	seen := make(map[uint32]bool)
	for _, s := range gs.Strata {
		if !Stratum(s.Stratum).IsValid() {
			return fmt.Errorf("invalid stratum level: %d", s.Stratum)
		}
		if seen[s.Stratum] {
			return fmt.Errorf("duplicate stratum level: %d", s.Stratum)
		}
		seen[s.Stratum] = true
		if s.MaxConfidence > 1000000 {
			return fmt.Errorf("stratum %s max confidence exceeds 1000000", s.Name)
		}
	}

	// Validate domains
	domainNames := make(map[string]bool)
	for _, d := range gs.Domains {
		if d.Name == "" {
			return fmt.Errorf("domain name cannot be empty")
		}
		if domainNames[d.Name] {
			return fmt.Errorf("duplicate domain name: %s", d.Name)
		}
		domainNames[d.Name] = true
		if !Stratum(d.Stratum).IsValid() {
			return fmt.Errorf("domain %s has invalid stratum: %d", d.Name, d.Stratum)
		}
	}

	return nil
}
