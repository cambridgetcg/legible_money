# x/knowledge — Proof of Truth Consensus

## Purpose

Core module implementing Proof of Truth (PoT) consensus. Manages knowledge claims, commit-reveal verification rounds, and facts with a 10-state lifecycle. Features adversarial verification with provisional facts, challenges, counter-facts, citations, and patronage.

## Key Concepts

- **Claim**: Submitted assertion with stake, domain, and epistemic category
- **VerificationRound**: Commit-reveal round with VRF-selected verifiers
- **Fact**: Accepted claim with confidence score, maturity level, and decay profile
- **CounterFact**: Alternative or negation of an existing fact
- **FactReference**: Citation between facts (boosts confidence)
- **Fact Status**: pending → provisional → verified → active → contested/challenged/superseded/expired/disproven/revoked/pruned

## CLI Commands

### Transactions
```bash
legbled tx knowledge submit-claim [domain] [text] --stake 1000000ulgm --from alice
legbled tx knowledge submit-commitment [round-id] [hash] --from validator1
legbled tx knowledge submit-reveal [round-id] [verdict] [confidence] [salt] --from validator1
legbled tx knowledge challenge-fact [fact-id] --stake 500000ulgm --from bob
legbled tx knowledge add-fact [domain] [text] --from authority
legbled tx knowledge submit-contradiction [fact-id] [counter-text] --from carol
legbled tx knowledge patronize-fact [fact-id] --amount 100000ulgm --from alice
legbled tx knowledge propose-domain [name] [stratum] --stake 5000000ulgm --from alice
legbled tx knowledge endorse-domain-proposal [proposal-id] --from bob
legbled tx knowledge challenge-domain-proposal [proposal-id] --from carol
legbled tx knowledge register-stratum [name] [level] --from authority
legbled tx knowledge challenge-provisional-fact [fact-id] --from bob
```

### Queries
```bash
legbled query knowledge params
legbled query knowledge fact [fact-id]
legbled query knowledge facts
legbled query knowledge facts-by-domain [domain]
legbled query knowledge claim [claim-id]
legbled query knowledge round [round-id]
legbled query knowledge active-rounds
legbled query knowledge domain [name]
legbled query knowledge domains
legbled query knowledge strata
legbled query knowledge calculate-confidence [fact-id]
legbled query knowledge counter-fact [id]
legbled query knowledge domain-proposal [id]
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| commit_phase_duration | 50 blocks | Commit phase length |
| reveal_phase_duration | 50 blocks | Reveal phase length |
| min_verifiers | 3 | Minimum verifiers per round |
| acceptance_threshold | 500,000 BPS | Vote threshold for acceptance |
| invalid_claim_slash_bps | 220,000 BPS | Slash on rejected claim (22%) |
| wrong_verification_slash_bps | 60,000 BPS | Slash for wrong verification (6%) |
| missed_reveal_slash_bps | 100 BPS | Slash for missed reveal (0.01%) |
| research_tax_rate | 220,000 BPS | Research fund tax (22%) |
| provisional_threshold | 500,000 BPS | Score for provisional fact (50%) |
| adversarial_verification_enabled | true | Enable adversarial mode |

34 total governance-adjustable parameters + 72 ExtendedParams.

## Events

`round_completed`, `fact_created`, `contested_fact_created`, `claim_stake_returned`, `claim_stake_slashed`, `pot_equivocation_detected`, `contradiction_provisional`, `fact_verified_survival`, `fact_superseded`, `reverification_triggered`, `fact_cited`, `citation_reward`, `fact_expired`, `fact_archived`, `patronage_settled`

## Dependencies

AccountKeeper, BankKeeper, StakingKeeper, OntologyKeeper, CaptureDefenseKeeper, VestingRewardsKeeper, EvidenceMgmtKeeper, HomeKeeper (optional), AutopoiesisKeeper (optional)
