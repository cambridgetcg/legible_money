package orchestrator

import "time"

// RepoSymphonyScore creates a full 5-movement orchestra performance:
// 1. Overture: All sections register + stake
// 2. First Movement: Brass submit claims across domains
// 3. Second Movement: Strings verify all pending rounds
// 4. Third Movement: Percussion challenge weak facts
// 5. Finale: Winds secure homes, all sections heartbeat
func RepoSymphonyScore(verifiers, claimers, challengers, guardians int) *Score {
	return &Score{
		Name: "repo-symphony",
		Ensemble: EnsembleSpec{
			Sections: []SectionSpec{
				{Name: SectionStrings, MinCount: verifiers, MaxCount: verifiers * 2, Domains: []string{"general", "mathematics", "physics"}},
				{Name: SectionBrass, MinCount: claimers, MaxCount: claimers * 2, Domains: []string{"general", "mathematics"}},
				{Name: SectionPercussion, MinCount: challengers, MaxCount: challengers * 2, Domains: []string{"general"}},
				{Name: SectionWinds, MinCount: guardians, MaxCount: guardians * 2, Domains: []string{"general"}},
			},
		},
		Movements: []MovementSpec{
			{
				Name: "overture",
				Parts: []PartSpec{
					{ID: "reg-strings", Section: SectionStrings, Action: ActionRegister, RepeatPerAgent: true, Params: map[string]string{
						"domains": "general,mathematics,physics", "stake": "100000000", "display_name": "verifier",
					}},
					{ID: "reg-brass", Section: SectionBrass, Action: ActionRegister, RepeatPerAgent: true, Params: map[string]string{
						"domains": "general,mathematics", "stake": "100000000", "display_name": "claimer",
					}},
					{ID: "reg-percussion", Section: SectionPercussion, Action: ActionRegister, RepeatPerAgent: true, Params: map[string]string{
						"domains": "general", "stake": "100000000", "display_name": "challenger",
					}},
					{ID: "reg-winds", Section: SectionWinds, Action: ActionRegister, RepeatPerAgent: true, Params: map[string]string{
						"domains": "general", "stake": "100000000", "display_name": "guardian",
					}},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
			{
				Name: "first-movement",
				Parts: []PartSpec{
					{ID: "submit-claims", Section: SectionBrass, Action: ActionSubmitClaim, Params: map[string]string{
						"domain": "mathematics", "category": "formal",
					}},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
			{
				Name: "second-movement",
				Parts: []PartSpec{
					{ID: "verify-rounds", Section: SectionStrings, Action: ActionVerify, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 300,
			},
			{
				Name: "third-movement",
				Parts: []PartSpec{
					{ID: "challenge-facts", Section: SectionPercussion, Action: ActionChallenge, Params: map[string]string{
						"domain": "general",
					}},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
			{
				Name: "finale",
				Parts: []PartSpec{
					{ID: "create-homes", Section: SectionWinds, Action: ActionCreateHome, RepeatPerAgent: true},
					{ID: "hb-strings", Section: SectionStrings, Action: ActionHeartbeat, RepeatPerAgent: true},
					{ID: "hb-brass", Section: SectionBrass, Action: ActionHeartbeat, RepeatPerAgent: true},
					{ID: "hb-percussion", Section: SectionPercussion, Action: ActionHeartbeat, RepeatPerAgent: true},
					{ID: "hb-winds", Section: SectionWinds, Action: ActionHeartbeat, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
		},
		Tempo: TempoSpec{
			PollInterval:       2521 * time.Millisecond,
			StaggerDelay:       300 * time.Millisecond,
			CheckpointInterval: 30 * time.Second,
		},
		Economics: EconomicsSpec{
			MinOperatingBalance: 10_000_000,
			MaxTotalGasSpend:    500_000_000,
			StakePerAgent:       100_000_000,
		},
	}
}

// VerificationConcertoScore creates a strings-only focused verification score.
func VerificationConcertoScore(verifiers int) *Score {
	return &Score{
		Name: "verification-concerto",
		Ensemble: EnsembleSpec{
			Sections: []SectionSpec{
				{Name: SectionStrings, MinCount: verifiers, MaxCount: verifiers * 2, Domains: []string{"general", "mathematics", "physics", "computer_science"}},
			},
		},
		Movements: []MovementSpec{
			{
				Name: "tuning",
				Parts: []PartSpec{
					{ID: "reg-verifiers", Section: SectionStrings, Action: ActionRegister, RepeatPerAgent: true, Params: map[string]string{
						"domains":      "general,mathematics,physics,computer_science",
						"stake":        "100000000",
						"display_name": "verification-specialist",
					}},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
			{
				Name: "first-theme",
				Parts: []PartSpec{
					{ID: "verify-math", Section: SectionStrings, Action: ActionVerify, RepeatPerAgent: true, Params: map[string]string{
						"domain": "mathematics",
					}},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 300,
			},
			{
				Name: "second-theme",
				Parts: []PartSpec{
					{ID: "verify-physics", Section: SectionStrings, Action: ActionVerify, RepeatPerAgent: true, Params: map[string]string{
						"domain": "physics",
					}},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 300,
			},
			{
				Name: "cadenza",
				Parts: []PartSpec{
					{ID: "heartbeat-all", Section: SectionStrings, Action: ActionHeartbeat, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 100,
			},
		},
		Tempo: TempoSpec{
			PollInterval:       2521 * time.Millisecond,
			StaggerDelay:       100 * time.Millisecond,
			CheckpointInterval: 30 * time.Second,
		},
		Economics: EconomicsSpec{
			MinOperatingBalance: 10_000_000,
			MaxTotalGasSpend:    100_000_000,
			StakePerAgent:       100_000_000,
		},
	}
}

// ChallengeStudyScore creates a percussion-focused challenge exercise.
func ChallengeStudyScore(challengers int) *Score {
	return &Score{
		Name: "challenge-study",
		Ensemble: EnsembleSpec{
			Sections: []SectionSpec{
				{Name: SectionPercussion, MinCount: challengers, MaxCount: challengers * 2, Domains: []string{"general", "predictive"}},
			},
		},
		Movements: []MovementSpec{
			{
				Name: "preparation",
				Parts: []PartSpec{
					{ID: "reg-challengers", Section: SectionPercussion, Action: ActionRegister, RepeatPerAgent: true, Params: map[string]string{
						"domains":      "general",
						"stake":        "100000000",
						"display_name": "challenge-specialist",
					}},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
			{
				Name: "reconnaissance",
				Parts: []PartSpec{
					{ID: "challenge-general", Section: SectionPercussion, Action: ActionChallenge, Params: map[string]string{
						"domain": "general",
					}},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
			{
				Name: "assault",
				Parts: []PartSpec{
					{ID: "challenge-predictive", Section: SectionPercussion, Action: ActionChallenge, Params: map[string]string{
						"domain": "predictive",
					}},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 200,
			},
			{
				Name: "debrief",
				Parts: []PartSpec{
					{ID: "debrief-hb", Section: SectionPercussion, Action: ActionHeartbeat, RepeatPerAgent: true},
				},
				Gate:    GateSpec{Type: GateAllParts},
				Timeout: 100,
			},
		},
		Tempo: TempoSpec{
			PollInterval:       2521 * time.Millisecond,
			StaggerDelay:       500 * time.Millisecond,
			CheckpointInterval: 30 * time.Second,
		},
		Economics: EconomicsSpec{
			MinOperatingBalance: 10_000_000,
			MaxTotalGasSpend:    100_000_000,
			StakePerAgent:       100_000_000,
		},
	}
}
