# x/disputes — Multi-Tier Arbitration

## Purpose

Multi-tier dispute resolution for challenged facts. Features evidence commit-reveal (privacy-preserving), stake-weighted arbiter voting, and escalation to higher tiers with more arbiters and higher bonds.

## Key Concepts

- **Tiers**: Tier 1 (3 arbiters) → escalation adds more arbiters and requires higher bonds
- **Commit-Reveal Evidence**: Hash committed first, then revealed (prevents copying)
- **Arbiter Voting**: challenger/defender/abstain with stake weighting
- **Escalation**: Loser can escalate to higher tier
- **Settlement**: Automatic after voting deadline

## CLI Commands

### Transactions
```bash
legbled tx disputes initiate [fact-id] --stake 1000000ulgm --reason "..." --from alice
legbled tx disputes commit-evidence [dispute-id] [evidence-hash] --from alice
legbled tx disputes reveal-evidence [dispute-id] [evidence] [salt] --from alice
legbled tx disputes vote [dispute-id] --verdict challenger --from arbiter1
legbled tx disputes escalate [dispute-id] --from loser
legbled tx disputes settle [dispute-id] --from anyone
legbled tx disputes timeout [dispute-id] --from anyone
```

### Queries
```bash
legbled query disputes params
legbled query disputes dispute [id]
legbled query disputes disputes-by-fact [fact-id]
legbled query disputes active-disputes
legbled query disputes evidence [dispute-id]
legbled query disputes votes [dispute-id]
```

## Parameters

Arbiter selection rules, bond amounts per tier, voting periods, escalation thresholds.

## Events

`DisputeInitiated`, `EvidenceCommitted`, `EvidenceRevealed`, `ArbiterVoted`, `DisputeEscalated`, `DisputeSettled`

## Dependencies

StakingKeeper, KnowledgeKeeper, EvidenceMgmtKeeper

## EndBlock

Processes expired disputes (transitions evidence→voting, tallies expired votes).
