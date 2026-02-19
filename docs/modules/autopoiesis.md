# x/autopoiesis — Adaptive Self-Tuning

## Purpose

Adaptive self-tuning engine operating on 1,111-block epochs. Cycles through OBSERVE, COMPUTE, ADAPT using 5 hormones. All arithmetic uses integer basis points (1,000,000 BPS = 100%) for determinism. Manages multipliers for slash rates, block rewards, verification counts, and more.

## Key Concepts

- **5 Hormones**: SSI (ABA), CRI (AUXIN), DVI (CYTOKININ), VCF (ETHYLENE), GRM (GIBBERELLIN)
- **Epoch**: 1,111 blocks per cycle
- **Multipliers**: Adjustable paths (slash.system, reward.block, verify.min, etc.)
- **Learning**: After 20 epochs, historical data informs adaptation
- **Governance Override**: Authority can override/freeze multipliers
- **Godelian Feedback**: Acknowledges system cannot fully optimize itself

## CLI Commands

### Transactions
```bash
legbled tx autopoiesis activate true --from authority
legbled tx autopoiesis override-multiplier [path] [value-bps] --from authority
legbled tx autopoiesis freeze-multiplier [path] --from authority
```

No query CLI (governance read via external query).

## Parameters

All tuning via multiplier overrides (no standard params structure).

## Events

`autopoiesis_epoch`, `autopoiesis_proposal_suggested`, `multiplier_overridden`

## Dependencies

StakingKeeper, KnowledgeKeeper, EmergencyKeeper, GovKeeper (optional, set after construction)

## EndBlock

Runs epoch cycle (observe → compute → adapt) at epoch boundaries.
