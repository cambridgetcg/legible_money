# x/partnerships — Multi-Party Deliberation

## Purpose

Multi-party partnerships with deliberative consensus, lock-based commitment tiers (6 levels), anti-coercion safety freezes, common pot management, and reward splits. Supports seed partnerships for genesis cold-start.

## Key Concepts

- **Partnership Tiers**: Trial(0) → Standard(1) → Trusted(2) → Elite(3) → Sovereign(4) → Eternal(5)
- **Deliberation**: Counter-proposals with configurable depth
- **Anti-Coercion**: Safety freeze mechanism; coercion signals trigger review
- **Common Pot**: Shared treasury with consensus-based operations
- **Exit**: Cooling period before dissolution

## CLI Commands

### Transactions
```bash
legbled tx partnerships initiate [partner-addr] --tier standard --from alice
legbled tx partnerships accept [partnership-id] --from bob
legbled tx partnerships deposit [partnership-id] --amount 1000000ulgm --from alice
legbled tx partnerships distribute-reward [partnership-id] --from alice
legbled tx partnerships propose-operation [partnership-id] --type transfer --from alice
legbled tx partnerships approve-operation [operation-id] --from bob
legbled tx partnerships reject-operation [operation-id] --from bob
legbled tx partnerships safety-freeze [partnership-id] --reason "..." --from alice
legbled tx partnerships raise-coercion-signal [partnership-id] --from alice
legbled tx partnerships initiate-exit [partnership-id] --from alice
```

### Queries
```bash
legbled query partnerships params
legbled query partnerships partnership [id]
legbled query partnerships by-participant [address]
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| FormationWindowBlocks | varies | Time to accept partnership |
| CoolingPeriodBlocks | varies | Cooling before dissolution |
| CommonPotShareBps | varies | Common pot contribution share |
| SafetyFreezeDurationBlocks | varies | Freeze duration |

## Events

`partnership_initiated`, `partnership_accepted`, `partnership_active`, `safety_freeze_triggered`, `coercion_signal_raised`, `partnership_exit_initiated`, `partnership_dissolved`, `reward_distributed`

## Dependencies

BankKeeper, StakingKeeper
