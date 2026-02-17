package orchestrator

import (
	"fmt"
	"time"
)

// BootstrapScore returns a score for bootstrapping a devnet with agents.
// Phases: register all → stake → start verification loops.
func BootstrapScore(validatorAddr string, stakePerAgent uint64) *Score {
	return &Score{
		Name: "devnet-bootstrap",
		Ensemble: EnsembleSpec{
			Sections: []SectionSpec{
				{Name: SectionStrings, MinCount: 2, MaxCount: 5, Domains: []string{"general"}},
				{Name: SectionBrass, MinCount: 1, MaxCount: 3, Domains: []string{"general"}},
				{Name: SectionPercussion, MinCount: 1, MaxCount: 2, Domains: []string{"general"}},
			},
		},
		Movements: []MovementSpec{
			{
				Name: "registration",
				Parts: []PartSpec{
					{
						ID:             "register-verifiers",
						Section:        SectionStrings,
						Action:         ActionRegister,
						Params:         map[string]string{"display_name": "verifier", "domains": "general", "stake": "100000000"},
						RepeatPerAgent: true,
					},
					{
						ID:             "register-claimers",
						Section:        SectionBrass,
						Action:         ActionRegister,
						Params:         map[string]string{"display_name": "claimer", "domains": "general", "stake": "100000000"},
						RepeatPerAgent: true,
					},
					{
						ID:             "register-challengers",
						Section:        SectionPercussion,
						Action:         ActionRegister,
						Params:         map[string]string{"display_name": "challenger", "domains": "general", "stake": "100000000"},
						RepeatPerAgent: true,
					},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200, // ~8 min
			},
			{
				Name: "staking",
				Parts: []PartSpec{
					{
						ID:      "stake-verifiers",
						Section: SectionStrings,
						Action:  ActionStake,
						Params: map[string]string{
							"validator_addr": validatorAddr,
							"amount":         uintToStr(stakePerAgent),
						},
						RepeatPerAgent: true,
					},
					{
						ID:      "stake-claimers",
						Section: SectionBrass,
						Action:  ActionStake,
						Params: map[string]string{
							"validator_addr": validatorAddr,
							"amount":         uintToStr(stakePerAgent),
						},
						RepeatPerAgent: true,
					},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
			{
				Name: "verification-start",
				Parts: []PartSpec{
					{
						ID:             "start-auto-verify",
						Section:        SectionStrings,
						Action:         ActionAutoVerify,
						RepeatPerAgent: true,
					},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 100,
			},
		},
		Tempo: TempoSpec{
			PollInterval:       2 * time.Second,
			StaggerDelay:       500 * time.Millisecond,
			CheckpointInterval: 30 * time.Second,
		},
		Economics: EconomicsSpec{
			MinOperatingBalance: 10_000_000,    // 10 LGM
			MaxTotalGasSpend:    100_000_000,   // 100 LGM
			StakePerAgent:       stakePerAgent,
			ValidatorAddr:       validatorAddr,
		},
	}
}

// KnowledgeBlitzScore returns a score for rapid knowledge claim submission
// followed by verification. Used for seeding a domain with initial facts.
func KnowledgeBlitzScore(domain string, claimCount int) *Score {
	parts := make([]PartSpec, 0, claimCount)
	for i := 0; i < claimCount; i++ {
		parts = append(parts, PartSpec{
			ID:      fmt.Sprintf("claim-%d", i+1),
			Section: SectionBrass,
			Action:  ActionSubmitClaim,
			Params: map[string]string{
				"domain":   domain,
				"category": "seed",
				"stake":    "1000000", // 1 LGM per claim
			},
		})
	}

	return &Score{
		Name: "knowledge-blitz-" + domain,
		Ensemble: EnsembleSpec{
			Sections: []SectionSpec{
				{Name: SectionStrings, MinCount: 3, MaxCount: 10, Domains: []string{domain}},
				{Name: SectionBrass, MinCount: 2, MaxCount: 5, Domains: []string{domain}},
			},
		},
		Movements: []MovementSpec{
			{
				Name:    "claim-submission",
				Parts:   parts,
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: int64(claimCount) * 50, // ~2 min per claim
			},
			{
				Name: "verification-sweep",
				Parts: []PartSpec{
					{
						ID:             "auto-verify-sweep",
						Section:        SectionStrings,
						Action:         ActionAutoVerify,
						RepeatPerAgent: true,
					},
				},
				Gate:    GateSpec{Type: GateBlockHeight, BlockHeight: 500},
				Timeout: 1000,
			},
		},
		Tempo: TempoSpec{
			PollInterval:       2 * time.Second,
			StaggerDelay:       time.Second,
			CheckpointInterval: time.Minute,
		},
		Economics: EconomicsSpec{
			MinOperatingBalance: 5_000_000,
			MaxTotalGasSpend:    50_000_000,
			StakePerAgent:       100_000_000,
		},
	}
}

// VerificationSwarmScore returns a score for continuous verification work.
// The swarm verifies claims across multiple domains indefinitely (until block gate).
func VerificationSwarmScore(domains []string, untilBlock int64) *Score {
	return &Score{
		Name: "verification-swarm",
		Ensemble: EnsembleSpec{
			Sections: []SectionSpec{
				{Name: SectionStrings, MinCount: 5, MaxCount: 20, Domains: domains},
			},
		},
		Movements: []MovementSpec{
			{
				Name: "continuous-verification",
				Parts: []PartSpec{
					{
						ID:             "auto-verify-all",
						Section:        SectionStrings,
						Action:         ActionAutoVerify,
						RepeatPerAgent: true,
					},
				},
				Gate:    GateSpec{Type: GateBlockHeight, BlockHeight: untilBlock},
				Timeout: untilBlock + 100,
			},
		},
		Tempo: TempoSpec{
			PollInterval:       3 * time.Second,
			StaggerDelay:       200 * time.Millisecond,
			CheckpointInterval: 2 * time.Minute,
		},
		Economics: EconomicsSpec{
			MinOperatingBalance: 10_000_000,
			MaxTotalGasSpend:    500_000_000,
		},
	}
}

// KnowledgeExplorationScore returns a 6-movement full-lifecycle performance score
// that exercises knowledge tree navigation, gap detection, claim generation,
// tree-aware verification, and reference chain challenge detection.
func KnowledgeExplorationScore(domains []string, validatorAddr string, stakePerAgent uint64) *Score {
	domainCSV := ""
	for i, d := range domains {
		if i > 0 {
			domainCSV += ","
		}
		domainCSV += d
	}

	return &Score{
		Name: "knowledge-exploration",
		Ensemble: EnsembleSpec{
			Sections: []SectionSpec{
				{Name: SectionStrings, MinCount: 3, MaxCount: 10, Domains: domains},
				{Name: SectionBrass, MinCount: 2, MaxCount: 5, Domains: domains},
				{Name: SectionPercussion, MinCount: 1, MaxCount: 3, Domains: domains},
				{Name: SectionWinds, MinCount: 1, MaxCount: 2, Domains: domains},
			},
		},
		Movements: []MovementSpec{
			// Movement 1: Prelude — all sections register + stake
			{
				Name: "prelude",
				Parts: []PartSpec{
					{ID: "register-strings", Section: SectionStrings, Action: ActionRegister,
						Params: map[string]string{"display_name": "explorer-verifier", "domains": domainCSV, "stake": uintToStr(stakePerAgent)}, RepeatPerAgent: true},
					{ID: "register-brass", Section: SectionBrass, Action: ActionRegister,
						Params: map[string]string{"display_name": "explorer-claimer", "domains": domainCSV, "stake": uintToStr(stakePerAgent)}, RepeatPerAgent: true},
					{ID: "register-percussion", Section: SectionPercussion, Action: ActionRegister,
						Params: map[string]string{"display_name": "explorer-challenger", "domains": domainCSV, "stake": uintToStr(stakePerAgent)}, RepeatPerAgent: true},
					{ID: "register-winds", Section: SectionWinds, Action: ActionRegister,
						Params: map[string]string{"display_name": "explorer-guardian", "domains": domainCSV, "stake": uintToStr(stakePerAgent)}, RepeatPerAgent: true},
					{ID: "stake-all", Section: SectionStrings, Action: ActionStake,
						Params: map[string]string{"validator_addr": validatorAddr, "amount": uintToStr(stakePerAgent)}, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 300,
			},
			// Movement 2: Reconnaissance — brass detect gaps via GapDetector
			{
				Name: "reconnaissance",
				Parts: []PartSpec{
					{ID: "scan-gaps", Section: SectionBrass, Action: ActionCustom,
						Params: map[string]string{"command": "detect_gaps", "domain": domainCSV}, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
			// Movement 3: Composition — brass generate and submit claims targeting gaps
			{
				Name: "composition",
				Parts: []PartSpec{
					{ID: "submit-claims", Section: SectionBrass, Action: ActionSubmitClaim,
						Params: map[string]string{"domain": domains[0], "stake": "1000000"}, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 300,
			},
			// Movement 4: Verification — strings verify with tree-aware evaluation
			{
				Name: "verification",
				Parts: []PartSpec{
					{ID: "tree-verify", Section: SectionStrings, Action: ActionAutoVerify, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateBlockHeight, BlockHeight: 500},
				Timeout: 600,
			},
			// Movement 5: Challenge — percussion challenge facts with broken reference chains
			{
				Name: "challenge",
				Parts: []PartSpec{
					{ID: "challenge-weak", Section: SectionPercussion, Action: ActionChallenge,
						Params: map[string]string{"domain": domains[0]}, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 300,
			},
			// Movement 6: Coda — all sections heartbeat, winds sync homes
			{
				Name: "coda",
				Parts: []PartSpec{
					{ID: "heartbeat-all", Section: SectionStrings, Action: ActionHeartbeat, RepeatPerAgent: true},
					{ID: "sync-homes", Section: SectionWinds, Action: ActionSyncHome,
						Params: map[string]string{"home_id": "default", "memory_cid": ""}, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
		},
		Tempo: TempoSpec{
			PollInterval:       2 * time.Second,
			StaggerDelay:       500 * time.Millisecond,
			CheckpointInterval: 30 * time.Second,
		},
		Economics: EconomicsSpec{
			MinOperatingBalance: 10_000_000,
			MaxTotalGasSpend:    200_000_000,
			StakePerAgent:       stakePerAgent,
			ValidatorAddr:       validatorAddr,
		},
	}
}

func uintToStr(v uint64) string {
	return fmt.Sprintf("%d", v)
}
